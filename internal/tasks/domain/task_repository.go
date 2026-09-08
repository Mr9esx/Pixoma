package domain

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
)

var ErrTaskNotFound = errors.New("task not found")

type ListTaskQuery struct {
	Status sharedkernel.TaskStatus
	Limit  int
}

type ListByInstanceQuery struct {
	Status sharedkernel.TaskStatus // empty = no filter
	Limit  int
	Offset int
}

// ListByTopicQuery filters tasks dispatched to a topic.
type ListByTopicQuery struct {
	Status      sharedkernel.TaskStatus // empty = no filter
	CreatedFrom *time.Time
	CreatedTo   *time.Time
	Limit       int
	Offset      int
}

// AdminListQuery filters tasks for admin list (parameterized; no client keys in SQL).
type AdminListQuery struct {
	Q      string
	Status sharedkernel.TaskStatus
	EdgeID sharedkernel.EdgeID
	ChatID sharedkernel.ChatID // "" = no filter
	// ChannelID filters tasks whose session belongs to a channel (join sessions).
	ChannelID string
	// DispatchTopic filters tasks dispatched to a topic.
	DispatchTopic string
	SessionID     sharedkernel.SessionID
	CaseID        sharedkernel.CaseID
	CreatedFrom   *time.Time
	CreatedTo     *time.Time
	Limit         int
	Offset        int
}

type TaskRepository interface {
	Create(ctx context.Context, t *Task) error
	Get(ctx context.Context, id sharedkernel.TaskID) (*Task, error)
	Update(ctx context.Context, t *Task) error
	// ClaimQueued atomically moves a pending task to queued with edgeID.
	// Returns (true, nil) on success; (false, nil) if not pending; ErrTaskNotFound if missing.
	ClaimQueued(ctx context.Context, id sharedkernel.TaskID, edgeID sharedkernel.EdgeID, now time.Time) (bool, error)
	// PrepareForClaim atomically pending→queued with topic + job_ref (claimable
	// by any node subscribed to the topic).
	PrepareForClaim(ctx context.Context, id sharedkernel.TaskID, topicKey string, jobRef sharedkernel.BlobRef, now time.Time) (bool, error)
	// ClaimNextWithLease atomically claims the oldest claimable queued task
	// subscribed by any of the given topics (queued→running+lease). At most one
	// caller ever wins a given task.
	ClaimNextWithLease(ctx context.Context, edgeID sharedkernel.EdgeID, topics []string, lease time.Duration, now time.Time) (*Task, error)
	// HeartbeatLease extends lease_until for a running task owned by edge.
	HeartbeatLease(ctx context.Context, id sharedkernel.TaskID, edgeID sharedkernel.EdgeID, lease time.Duration, now time.Time) (bool, error)
	// RequeueExpiredLeases moves running tasks with expired leases back to queued.
	RequeueExpiredLeases(ctx context.Context, now time.Time) (int, error)
	ListByChat(ctx context.Context, chatID sharedkernel.ChatID, limit int) ([]*Task, error)
	ListByStatus(ctx context.Context, st sharedkernel.TaskStatus, limit int) ([]*Task, error)
	ListByInstance(ctx context.Context, edgeID sharedkernel.EdgeID, q ListByInstanceQuery) ([]*Task, error)
	ListByTopic(ctx context.Context, topicKey string, q ListByTopicQuery) ([]*Task, error)
	List(ctx context.Context, q AdminListQuery) ([]*Task, error)
}

type MemoryTaskRepository struct {
	mu   sync.Mutex
	byID map[sharedkernel.TaskID]*Task
}

func NewMemoryTaskRepository() *MemoryTaskRepository {
	return &MemoryTaskRepository{byID: map[sharedkernel.TaskID]*Task{}}
}

func (r *MemoryTaskRepository) Create(_ context.Context, t *Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.byID[t.ID]; ok {
		return errors.New("task already exists")
	}
	cp := *t
	cp.Outputs = append([]OutputRef(nil), t.Outputs...)
	r.byID[t.ID] = &cp
	return nil
}

func (r *MemoryTaskRepository) Get(_ context.Context, id sharedkernel.TaskID) (*Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.byID[id]
	if !ok {
		return nil, ErrTaskNotFound
	}
	cp := *t
	cp.Outputs = append([]OutputRef(nil), t.Outputs...)
	return &cp, nil
}

func (r *MemoryTaskRepository) Update(_ context.Context, t *Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.byID[t.ID]; !ok {
		return ErrTaskNotFound
	}
	cp := *t
	cp.Outputs = append([]OutputRef(nil), t.Outputs...)
	r.byID[t.ID] = &cp
	return nil
}

func (r *MemoryTaskRepository) ClaimQueued(_ context.Context, id sharedkernel.TaskID, edgeID sharedkernel.EdgeID, now time.Time) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.byID[id]
	if !ok {
		return false, ErrTaskNotFound
	}
	if t.Status != sharedkernel.TaskPending {
		return false, nil
	}
	t.Status = sharedkernel.TaskQueued
	t.EdgeID = edgeID
	t.UpdatedAt = now
	return true, nil
}

func (r *MemoryTaskRepository) PrepareForClaim(_ context.Context, id sharedkernel.TaskID, topicKey string, jobRef sharedkernel.BlobRef, now time.Time) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.byID[id]
	if !ok {
		return false, ErrTaskNotFound
	}
	if t.Status != sharedkernel.TaskPending {
		return false, nil
	}
	if err := t.PrepareForTopic(topicKey, jobRef, now); err != nil {
		return false, err
	}
	return true, nil
}

func (r *MemoryTaskRepository) ClaimNextWithLease(_ context.Context, edgeID sharedkernel.EdgeID, topics []string, lease time.Duration, now time.Time) (*Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if edgeID == "" || len(topics) == 0 || lease <= 0 {
		return nil, nil
	}
	want := map[string]bool{}
	for _, t := range topics {
		want[t] = true
	}
	var best *Task
	for _, t := range r.byID {
		if t.Status != sharedkernel.TaskQueued || t.JobRef.Key == "" {
			continue
		}
		if !want[t.DispatchTopic] {
			continue
		}
		if !t.RequeueAt.IsZero() && now.Before(t.RequeueAt) {
			continue
		}
		if best == nil || t.CreatedAt.Before(best.CreatedAt) {
			best = t
		}
	}
	if best == nil {
		return nil, nil
	}
	if err := best.ClaimWithLease(edgeID, lease, now); err != nil {
		return nil, err
	}
	cp := *best
	cp.Outputs = append([]OutputRef(nil), best.Outputs...)
	return &cp, nil
}

func (r *MemoryTaskRepository) HeartbeatLease(_ context.Context, id sharedkernel.TaskID, edgeID sharedkernel.EdgeID, lease time.Duration, now time.Time) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.byID[id]
	if !ok {
		return false, ErrTaskNotFound
	}
	if t.Status != sharedkernel.TaskRunning || t.EdgeID != edgeID || lease <= 0 {
		return false, nil
	}
	t.LeaseUntil = now.Add(lease)
	t.UpdatedAt = now
	return true, nil
}

func (r *MemoryTaskRepository) RequeueExpiredLeases(_ context.Context, now time.Time) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for _, t := range r.byID {
		ok, err := t.RequeueIfLeaseExpired(now)
		if err != nil {
			return n, err
		}
		if ok {
			t.EdgeID = ""
			t.RequeueAt = now
			n++
		}
	}
	return n, nil
}

func (r *MemoryTaskRepository) ListByChat(_ context.Context, chatID sharedkernel.ChatID, limit int) ([]*Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*Task
	for _, t := range r.byID {
		if t.ChatID != chatID {
			continue
		}
		cp := *t
		cp.Outputs = append([]OutputRef(nil), t.Outputs...)
		out = append(out, &cp)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (r *MemoryTaskRepository) ListByStatus(_ context.Context, st sharedkernel.TaskStatus, limit int) ([]*Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*Task
	for _, t := range r.byID {
		if t.Status != st {
			continue
		}
		cp := *t
		cp.Outputs = append([]OutputRef(nil), t.Outputs...)
		out = append(out, &cp)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (r *MemoryTaskRepository) ListByInstance(_ context.Context, edgeID sharedkernel.EdgeID, q ListByInstanceQuery) ([]*Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if edgeID == "" {
		return nil, nil
	}
	var out []*Task
	skipped := 0
	for _, t := range r.byID {
		if t.EdgeID != edgeID || t.EdgeID == "" {
			continue
		}
		if q.Status != "" && t.Status != q.Status {
			continue
		}
		if q.Offset > 0 && skipped < q.Offset {
			skipped++
			continue
		}
		cp := *t
		cp.Outputs = append([]OutputRef(nil), t.Outputs...)
		out = append(out, &cp)
		if q.Limit > 0 && len(out) >= q.Limit {
			break
		}
	}
	return out, nil
}

func (r *MemoryTaskRepository) ListByTopic(_ context.Context, topicKey string, q ListByTopicQuery) ([]*Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if topicKey == "" {
		return nil, nil
	}
	var out []*Task
	skipped := 0
	for _, t := range r.byID {
		if t.DispatchTopic != topicKey {
			continue
		}
		if q.Status != "" && t.Status != q.Status {
			continue
		}
		if q.CreatedFrom != nil && t.CreatedAt.Before(*q.CreatedFrom) {
			continue
		}
		if q.CreatedTo != nil && t.CreatedAt.After(*q.CreatedTo) {
			continue
		}
		if q.Offset > 0 && skipped < q.Offset {
			skipped++
			continue
		}
		cp := *t
		cp.Outputs = append([]OutputRef(nil), t.Outputs...)
		out = append(out, &cp)
		if q.Limit > 0 && len(out) >= q.Limit {
			break
		}
	}
	return out, nil
}

func (r *MemoryTaskRepository) List(_ context.Context, q AdminListQuery) ([]*Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var matched []*Task
	for _, t := range r.byID {
		if q.Status != "" && t.Status != q.Status {
			continue
		}
		if q.EdgeID != "" && t.EdgeID != q.EdgeID {
			continue
		}
		if q.ChatID != "" && t.ChatID != q.ChatID {
			continue
		}
		if q.SessionID != "" && t.SessionID != q.SessionID {
			continue
		}
		if q.CaseID != 0 && t.CaseID != q.CaseID {
			continue
		}
		if q.DispatchTopic != "" && t.DispatchTopic != q.DispatchTopic {
			continue
		}
		if q.Q != "" {
			needle := strings.ToLower(q.Q)
			hay := strings.ToLower(string(t.ID) + " " + strconv.FormatUint(uint64(t.CaseID), 10) + " " + string(t.SessionID))
			if !strings.Contains(hay, needle) {
				continue
			}
		}
		if q.CreatedFrom != nil && t.CreatedAt.Before(*q.CreatedFrom) {
			continue
		}
		if q.CreatedTo != nil && t.CreatedAt.After(*q.CreatedTo) {
			continue
		}
		cp := *t
		cp.Outputs = append([]OutputRef(nil), t.Outputs...)
		matched = append(matched, &cp)
	}
	if q.Offset > 0 {
		if q.Offset >= len(matched) {
			return nil, nil
		}
		matched = matched[q.Offset:]
	}
	if q.Limit > 0 && len(matched) > q.Limit {
		matched = matched[:q.Limit]
	}
	return matched, nil
}
