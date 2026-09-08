package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	catalogdomain "github.com/Mr9esx/Pixoma/internal/cases/domain"
	convdomain "github.com/Mr9esx/Pixoma/internal/sessions/domain"
	edge "github.com/Mr9esx/Pixoma/internal/edge/domain"
	"github.com/Mr9esx/Pixoma/internal/platform/notify"
	"github.com/Mr9esx/Pixoma/internal/platform/queue"
	statsdomain "github.com/Mr9esx/Pixoma/internal/stats/domain"
	"github.com/Mr9esx/Pixoma/internal/tasks/application/routing"
	runtimedomain "github.com/Mr9esx/Pixoma/internal/tasks/domain"
	"github.com/Mr9esx/Pixoma/internal/tasks/domain/condition"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
)

// ErrStaleHolder is returned when a status report is not from the current claim holder.
var ErrStaleHolder = errors.New("orchestrator: status from non-holder")

type ExecutionQuery interface {
	GetRun(ctx context.Context, taskID sharedkernel.TaskID) (*ExecutionView, error)
}

// JobPreparer builds scheme-A job packages into Blob before dispatch.
type JobPreparer interface {
	PrepareJob(ctx context.Context, taskID sharedkernel.TaskID) (sharedkernel.BlobRef, error)
}

// CaseReader loads the protocol document of a Case for routing evaluation.
type CaseReader interface {
	GetCase(ctx context.Context, caseID sharedkernel.CaseID) (*catalogdomain.CaseDocument, error)
}

type ExecutionView struct {
	TaskID   sharedkernel.TaskID
	Phase    string // accepted|running|succeeded|failed|unknown
	PromptID string
	Outputs  []sharedkernel.BlobRef
	ErrorMsg string
}

type Service struct {
	Tasks     runtimedomain.TaskRepository
	Sessions  convdomain.Repository // optional; notify joins chat via session_id
	Instances edge.Registry
	Dispatch  queue.Publisher
	Notify    notify.Publisher
	Query     ExecutionQuery
	Prep      JobPreparer
	Cases     CaseReader
	Condition *condition.Registry
	// Stats optionally records terminal task rollups; nil disables stats writes.
	Stats statsdomain.Repository
	// Online optionally filters candidates in split mode (nil = no extra filter).
	Online func(ctx context.Context, id sharedkernel.EdgeID) bool
	Storm  *StormGuard
	Now    func() time.Time

	// notifyDedupe tracks terminal notifies already sent.
	notifiedMu sync.Mutex
	notified   map[string]struct{}
}

func New(tasks runtimedomain.TaskRepository, instances edge.Registry, dispatch queue.Publisher, n notify.Publisher) *Service {
	return &Service{
		Tasks:     tasks,
		Instances: instances,
		Dispatch:  dispatch,
		Notify:    n,
		Storm:     NewStormGuard(StormConfig{}),
		Now:       func() time.Time { return time.Now().UTC() },
		notified:  map[string]struct{}{},
	}
}

func (s *Service) OnTaskCreated(ctx context.Context, ev sharedkernel.TaskCreated) error {
	return s.dispatchTask(ctx, ev.TaskID)
}

func (s *Service) SchedulePending(ctx context.Context, limit int) error {
	if !s.Storm.AllowSchedule(1) {
		return nil
	}
	list, err := s.Tasks.ListByStatus(ctx, sharedkernel.TaskPending, limit)
	if err != nil {
		return err
	}
	for _, t := range list {
		if !s.Storm.AllowSchedule(1) {
			break
		}
		if err := s.dispatchTask(ctx, t.ID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) dispatchTask(ctx context.Context, taskID sharedkernel.TaskID) error {
	t, err := s.Tasks.Get(ctx, taskID)
	if err != nil {
		return err
	}
	if t.Status != sharedkernel.TaskPending {
		return nil
	}
	topicKey, err := s.resolveTopic(ctx, t)
	if err != nil {
		if errors.Is(err, catalogdomain.ErrNotFound) {
			now := s.Now()
			if merr := t.MarkFailed(sharedkernel.TaskErrorCaseDeleted, sharedkernel.CaseDeletedMessage, now); merr == nil {
				_ = s.Tasks.Update(ctx, t)
				_ = s.publishNotify(ctx, t)
			}
			return nil
		}
		if errors.Is(err, routing.ErrNoMatch) {
			now := s.Now()
			if merr := t.MarkFailed(sharedkernel.TaskErrorRoutingNoMatch, sharedkernel.TaskErrorMessageRoutingNoMatch, now); merr == nil {
				_ = s.Tasks.Update(ctx, t)
				_ = s.publishNotify(ctx, t)
			}
			return nil
		}
		// Evaluation failure: keep pending with a recorded reason; the next
		// SchedulePending cycle retries (transient provider errors self-heal).
		t.ErrorMessage = "routing: " + err.Error()
		t.UpdatedAt = s.Now()
		_ = s.Tasks.Update(ctx, t)
		return nil
	}
	if !s.topicHasOnlineConsumer(ctx, topicKey) {
		t.ErrorMessage = fmt.Sprintf("routing: no online consumer for topic %s", topicKey)
		t.UpdatedAt = s.Now()
		_ = s.Tasks.Update(ctx, t)
		return nil
	}
	now := s.Now()
	if s.Prep == nil {
		return fmt.Errorf("orchestrator: job preparer required for claimable dispatch")
	}
	ref, err := s.Prep.PrepareJob(ctx, t.ID)
	if err != nil {
		return fmt.Errorf("orchestrator: prepare job: %w", err)
	}
	if ref.Key == "" {
		return fmt.Errorf("orchestrator: empty job_ref")
	}
	prepared, err := s.Tasks.PrepareForClaim(ctx, taskID, topicKey, ref, now)
	if err != nil {
		return err
	}
	if !prepared {
		return nil
	}
	return nil
}

func (s *Service) resolveTopic(ctx context.Context, t *runtimedomain.Task) (string, error) {
	if s.Cases == nil || s.Condition == nil {
		return "", routing.ErrNoMatch
	}
	caseDoc, err := s.Cases.GetCase(ctx, t.CaseID)
	if err != nil {
		return "", fmt.Errorf("load case %d: %w", t.CaseID, err)
	}
	return routing.Resolve(caseDoc.Routing, ctx, s.Condition)
}

func (s *Service) topicHasOnlineConsumer(ctx context.Context, topicKey string) bool {
	if s.Instances == nil {
		return true
	}
	insts, err := s.Instances.ListEnabled(ctx, edge.CapabilityFilter{})
	if err != nil {
		return true
	}
	for _, inst := range insts {
		for _, t := range inst.EffectiveTopics() {
			if t == topicKey {
				if s.Online == nil || s.Online(ctx, inst.ID) {
					return true
				}
			}
		}
	}
	return false
}

func (s *Service) OnStatus(ctx context.Context, ev sharedkernel.TaskStatusEvent) error {
	return s.applyStatus(ctx, ev)
}

func (s *Service) applyStatus(ctx context.Context, ev sharedkernel.TaskStatusEvent) error {
	t, err := s.Tasks.Get(ctx, ev.TaskID)
	if err != nil {
		return err
	}
	if ev.EdgeID != "" && t.EdgeID != "" && ev.EdgeID != t.EdgeID {
		return ErrStaleHolder
	}
	now := ev.At
	if now.IsZero() {
		now = s.Now()
	}
	prev := t.Status

	switch ev.Status {
	case sharedkernel.TaskRunning:
		if err := t.MarkRunning(ev.PromptID, now); err != nil {
			return err
		}
	case sharedkernel.TaskSucceeded:
		outs := make([]runtimedomain.OutputRef, 0, len(ev.Outputs))
		for i, b := range ev.Outputs {
			outs = append(outs, runtimedomain.OutputRef{Key: fmt.Sprintf("out-%d", i), Blob: b})
		}
		if err := t.MarkSucceeded(outs, now); err != nil {
			return err
		}
	case sharedkernel.TaskFailed:
		if t.Status == sharedkernel.TaskFailed {
			return nil // idempotent
		}
		requeued, err := t.RequeueAfterFailure(runtimedomain.BackoffFor(t.Attempts+1), now, ev.ErrorMsg)
		if err != nil {
			return err
		}
		if requeued {
			if err := s.Tasks.Update(ctx, t); err != nil {
				return err
			}
			return nil
		}
	case sharedkernel.TaskCancelled:
		if err := t.MarkCancelled(now); err != nil {
			return err
		}
	default:
		return fmt.Errorf("orchestrator: unsupported status %s", ev.Status)
	}

	if err := s.Tasks.Update(ctx, t); err != nil {
		return err
	}

	if isTerminal(t.Status) && t.Status != prev {
		if err := s.recordTerminalStats(ctx, t, now); err != nil {
			return err
		}
		return s.publishNotify(ctx, t)
	}
	return nil
}

func isTerminal(st sharedkernel.TaskStatus) bool {
	return st == sharedkernel.TaskSucceeded || st == sharedkernel.TaskFailed || st == sharedkernel.TaskCancelled
}

func (s *Service) publishNotify(ctx context.Context, t *runtimedomain.Task) error {
	key := string(t.ID) + ":" + string(t.Status)
	s.notifiedMu.Lock()
	defer s.notifiedMu.Unlock()
	if _, ok := s.notified[key]; ok {
		return nil
	}
	chatID := t.ChatID
	if chatID == "" && s.Sessions != nil {
		sess, err := s.Sessions.GetByID(ctx, t.SessionID)
		if err != nil {
			return fmt.Errorf("notify chat via session: %w", err)
		}
		chatID = sess.ChatID
	}
	kind := "task_" + string(t.Status)
	n := sharedkernel.UserNotify{
		ChatID:   chatID,
		TaskID:   t.ID,
		Kind:     kind,
		ErrorMsg: t.ErrorMessage,
	}
	for _, o := range t.Outputs {
		n.Outputs = append(n.Outputs, o.Blob)
	}
	if err := s.Notify.Publish(ctx, n); err != nil {
		return err
	}
	s.notified[key] = struct{}{}
	return nil
}

func (s *Service) RequestCancel(ctx context.Context, taskID sharedkernel.TaskID) error {
	t, err := s.Tasks.Get(ctx, taskID)
	if err != nil {
		return err
	}
	now := s.Now()
	prev := t.Status
	if err := t.MarkCancelled(now); err != nil {
		return err
	}
	if err := s.Tasks.Update(ctx, t); err != nil {
		return err
	}
	if prev != sharedkernel.TaskCancelled {
		if err := s.recordTerminalStats(ctx, t, now); err != nil {
			return err
		}
	}
	return s.publishNotify(ctx, t)
}

// recordTerminalStats writes one terminal transition into the daily rollup.
// The old!=new guard is applied by callers; a zero CompletedAt (e.g. retries
// exhausted) falls back to the event time so the row is bucketed correctly.
func (s *Service) recordTerminalStats(ctx context.Context, t *runtimedomain.Task, now time.Time) error {
	if s.Stats == nil {
		return nil
	}
	status := statsdomain.Status(t.Status)
	switch status {
	case statsdomain.StatusSucceeded, statsdomain.StatusFailed, statsdomain.StatusCancelled:
	default:
		return nil
	}
	completedAt := t.CompletedAt
	if completedAt.IsZero() {
		completedAt = now
	}
	queueMS := t.StartedAt.Sub(t.CreatedAt).Milliseconds()
	if queueMS < 0 {
		queueMS = 0
	}
	execMS := completedAt.Sub(t.StartedAt).Milliseconds()
	if execMS < 0 {
		execMS = 0
	}
	return s.Stats.AddTerminal(ctx, statsdomain.AddTerminalInput{
		EdgeID:          string(t.EdgeID),
		ErrorCode:       t.ErrorCode,
		Status:          status,
		CompletedAt:     completedAt,
		CreatedAt:       t.CreatedAt,
		CaseID:          uint64(t.CaseID),
		QueueDurationMS: queueMS,
		ExecDurationMS:  execMS,
	})
}

func (s *Service) ReconcileStale(ctx context.Context, staleAfter time.Duration, limit int) error {
	if !s.Storm.AllowReconcile(1) {
		return nil
	}
	now := s.Now()
	for _, st := range []sharedkernel.TaskStatus{sharedkernel.TaskQueued, sharedkernel.TaskRunning} {
		list, err := s.Tasks.ListByStatus(ctx, st, limit)
		if err != nil {
			return err
		}
		for _, t := range list {
			if !s.Storm.AllowReconcile(1) {
				return nil
			}
			if now.Sub(t.UpdatedAt) < staleAfter {
				continue
			}
			fresh, err := s.Tasks.Get(ctx, t.ID)
			if err != nil {
				s.Storm.Breaker.RecordFailure(t.EdgeID)
				continue
			}
			s.Storm.Breaker.RecordSuccess(t.EdgeID)
			view := executionViewFromTask(fresh)
			switch view.Phase {
			case "succeeded":
				_ = s.applyStatus(ctx, sharedkernel.TaskStatusEvent{
					TaskID: t.ID, EdgeID: t.EdgeID, Status: sharedkernel.TaskSucceeded,
					Outputs: view.Outputs, At: now,
				})
			case "failed":
				_ = s.applyStatus(ctx, sharedkernel.TaskStatusEvent{
					TaskID: t.ID, EdgeID: t.EdgeID, Status: sharedkernel.TaskFailed,
					ErrorMsg: view.ErrorMsg, At: now,
				})
			case "running":
				// Task-only view has no external progress; do not refresh UpdatedAt.
			case "accepted":
				// Stale queued without job_ref never became claimable — re-pend for redispatch.
				// Queued with job_ref is waiting for Edge pull; leave it.
				if fresh.Status == sharedkernel.TaskQueued && fresh.PromptID == "" && fresh.JobRef.Key == "" {
					fresh.Status = sharedkernel.TaskPending
					fresh.EdgeID = ""
					fresh.UpdatedAt = now
					_ = s.Tasks.Update(ctx, fresh)
				} else if fresh.Status == sharedkernel.TaskRunning {
					_, _ = s.Tasks.RequeueExpiredLeases(ctx, now)
				}
			}
		}
	}
	return nil
}

func executionViewFromTask(t *runtimedomain.Task) *ExecutionView {
	view := &ExecutionView{
		TaskID:   t.ID,
		Phase:    phaseFromTaskStatus(t.Status),
		PromptID: t.PromptID,
		ErrorMsg: t.ErrorMessage,
	}
	for _, o := range t.Outputs {
		view.Outputs = append(view.Outputs, o.Blob)
	}
	return view
}

func phaseFromTaskStatus(st sharedkernel.TaskStatus) string {
	switch st {
	case sharedkernel.TaskQueued:
		return "accepted"
	case sharedkernel.TaskRunning:
		return "running"
	case sharedkernel.TaskSucceeded:
		return "succeeded"
	case sharedkernel.TaskFailed:
		return "failed"
	default:
		return "unknown"
	}
}
