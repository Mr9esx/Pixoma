package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	convdomain "github.com/mr9esx/comfyui_tgbot/internal/conversation/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/instance"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/notify"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/queue"
	runtimedomain "github.com/mr9esx/comfyui_tgbot/internal/runtime/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

type ExecutionQuery interface {
	GetRun(ctx context.Context, taskID sharedkernel.TaskID) (*ExecutionView, error)
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
	Instances instance.Registry
	Dispatch  queue.Publisher
	Notify    notify.Publisher
	Query     ExecutionQuery
	Storm     *StormGuard
	Now       func() time.Time

	// notifyDedupe tracks terminal notifies already sent.
	notified map[string]struct{}
}

func New(tasks runtimedomain.TaskRepository, instances instance.Registry, dispatch queue.Publisher, n notify.Publisher) *Service {
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
	insts, err := s.Instances.ListHealthy(ctx, instance.CapabilityFilter{})
	if err != nil {
		return err
	}
	var chosen *instance.Instance
	for i := range insts {
		id := insts[i].ID
		if s.Storm.Breaker.Allow(id) {
			chosen = &insts[i]
			break
		}
	}
	if chosen == nil {
		return fmt.Errorf("orchestrator: no healthy instance")
	}
	now := s.Now()
	if err := t.MarkQueued(chosen.ID, now); err != nil {
		return err
	}
	if err := s.Tasks.Update(ctx, t); err != nil {
		return err
	}
	cmd := sharedkernel.DispatchCommand{
		TaskID:      t.ID,
		InstanceID:  chosen.ID,
		InputPrefix: t.InputPrefix,
	}
	payload, err := json.Marshal(cmd)
	if err != nil {
		return err
	}
	topic := chosen.DispatchTopic
	if topic == "" {
		topic = sharedkernel.TopicDispatch(chosen.ID)
	}
	if err := s.Dispatch.Publish(ctx, queue.Message{Topic: topic, Key: string(t.ID), Payload: payload}); err != nil {
		s.Storm.Breaker.RecordFailure(chosen.ID)
		return err
	}
	s.Storm.Breaker.RecordSuccess(chosen.ID)
	return nil
}

func (s *Service) OnStatus(ctx context.Context, ev sharedkernel.TaskStatusEvent) error {
	return s.applyStatus(ctx, ev)
}

func (s *Service) applyStatus(ctx context.Context, ev sharedkernel.TaskStatusEvent) error {
	t, err := s.Tasks.Get(ctx, ev.TaskID)
	if err != nil {
		return err
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
		if err := t.MarkFailed(ev.ErrorCode, ev.ErrorMsg, now); err != nil {
			return err
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
		return s.publishNotify(ctx, t)
	}
	return nil
}

func isTerminal(st sharedkernel.TaskStatus) bool {
	return st == sharedkernel.TaskSucceeded || st == sharedkernel.TaskFailed || st == sharedkernel.TaskCancelled
}

func (s *Service) publishNotify(ctx context.Context, t *runtimedomain.Task) error {
	key := string(t.ID) + ":" + string(t.Status)
	if _, ok := s.notified[key]; ok {
		return nil
	}
	chatID := t.ChatID
	if chatID == 0 && s.Sessions != nil {
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
	if err := t.MarkCancelled(s.Now()); err != nil {
		return err
	}
	if err := s.Tasks.Update(ctx, t); err != nil {
		return err
	}
	return s.publishNotify(ctx, t)
}

func (s *Service) ReconcileStale(ctx context.Context, staleAfter time.Duration, limit int) error {
	if s.Query == nil {
		return nil
	}
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
			view, err := s.Query.GetRun(ctx, t.ID)
			if err != nil {
				s.Storm.Breaker.RecordFailure(t.InstanceID)
				continue
			}
			s.Storm.Breaker.RecordSuccess(t.InstanceID)
			switch view.Phase {
			case "succeeded":
				_ = s.applyStatus(ctx, sharedkernel.TaskStatusEvent{
					TaskID: t.ID, InstanceID: t.InstanceID, Status: sharedkernel.TaskSucceeded,
					Outputs: view.Outputs, At: now,
				})
			case "failed":
				_ = s.applyStatus(ctx, sharedkernel.TaskStatusEvent{
					TaskID: t.ID, InstanceID: t.InstanceID, Status: sharedkernel.TaskFailed,
					ErrorMsg: view.ErrorMsg, At: now,
				})
			case "running":
				_ = s.applyStatus(ctx, sharedkernel.TaskStatusEvent{
					TaskID: t.ID, InstanceID: t.InstanceID, Status: sharedkernel.TaskRunning,
					PromptID: view.PromptID, At: now,
				})
			}
		}
	}
	return nil
}
