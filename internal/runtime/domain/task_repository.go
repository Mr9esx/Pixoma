package domain

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
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

// AdminListQuery filters tasks for admin list (parameterized; no client keys in SQL).
type AdminListQuery struct {
	Q           string
	Status      sharedkernel.TaskStatus
	InstanceID  sharedkernel.InstanceID
	ChatID      sharedkernel.ChatID // 0 = no filter
	SessionID   sharedkernel.SessionID
	CaseID      sharedkernel.CaseID
	CreatedFrom *time.Time
	CreatedTo   *time.Time
	Limit       int
	Offset      int
}

type TaskRepository interface {
	Create(ctx context.Context, t *Task) error
	Get(ctx context.Context, id sharedkernel.TaskID) (*Task, error)
	Update(ctx context.Context, t *Task) error
	// ClaimQueued atomically moves a pending task to queued with instanceID.
	// Returns (true, nil) on success; (false, nil) if not pending; ErrTaskNotFound if missing.
	ClaimQueued(ctx context.Context, id sharedkernel.TaskID, instanceID sharedkernel.InstanceID, now time.Time) (bool, error)
	// PrepareForClaim atomically pending→queued with instance + job_ref (claimable).
	PrepareForClaim(ctx context.Context, id sharedkernel.TaskID, instanceID sharedkernel.InstanceID, jobRef sharedkernel.BlobRef, now time.Time) (bool, error)
	// ClaimNextWithLease claims the oldest claimable queued task for instance (queued→running+lease).
	ClaimNextWithLease(ctx context.Context, instanceID sharedkernel.InstanceID, lease time.Duration, now time.Time) (*Task, error)
	// HeartbeatLease extends lease_until for a running task owned by instance.
	HeartbeatLease(ctx context.Context, id sharedkernel.TaskID, instanceID sharedkernel.InstanceID, lease time.Duration, now time.Time) (bool, error)
	// RequeueExpiredLeases moves running tasks with expired leases back to queued.
	RequeueExpiredLeases(ctx context.Context, now time.Time) (int, error)
	ListByChat(ctx context.Context, chatID sharedkernel.ChatID, limit int) ([]*Task, error)
	ListByStatus(ctx context.Context, st sharedkernel.TaskStatus, limit int) ([]*Task, error)
	ListByInstance(ctx context.Context, instanceID sharedkernel.InstanceID, q ListByInstanceQuery) ([]*Task, error)
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

func (r *MemoryTaskRepository) ClaimQueued(_ context.Context, id sharedkernel.TaskID, instanceID sharedkernel.InstanceID, now time.Time) (bool, error) {
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
	t.InstanceID = instanceID
	t.UpdatedAt = now
	return true, nil
}

func (r *MemoryTaskRepository) PrepareForClaim(_ context.Context, id sharedkernel.TaskID, instanceID sharedkernel.InstanceID, jobRef sharedkernel.BlobRef, now time.Time) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.byID[id]
	if !ok {
		return false, ErrTaskNotFound
	}
	if t.Status != sharedkernel.TaskPending {
		return false, nil
	}
	if err := t.PrepareForClaim(instanceID, jobRef, now); err != nil {
		return false, err
	}
	return true, nil
}

func (r *MemoryTaskRepository) ClaimNextWithLease(_ context.Context, instanceID sharedkernel.InstanceID, lease time.Duration, now time.Time) (*Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if instanceID == "" || lease <= 0 {
		return nil, nil
	}
	var best *Task
	for _, t := range r.byID {
		if t.Status != sharedkernel.TaskQueued || t.InstanceID != instanceID || t.JobRef.Key == "" {
			continue
		}
		if best == nil || t.CreatedAt.Before(best.CreatedAt) {
			best = t
		}
	}
	if best == nil {
		return nil, nil
	}
	if err := best.ClaimWithLease(instanceID, lease, now); err != nil {
		return nil, err
	}
	cp := *best
	cp.Outputs = append([]OutputRef(nil), best.Outputs...)
	return &cp, nil
}

func (r *MemoryTaskRepository) HeartbeatLease(_ context.Context, id sharedkernel.TaskID, instanceID sharedkernel.InstanceID, lease time.Duration, now time.Time) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.byID[id]
	if !ok {
		return false, ErrTaskNotFound
	}
	if t.Status != sharedkernel.TaskRunning || t.InstanceID != instanceID || lease <= 0 {
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

func (r *MemoryTaskRepository) ListByInstance(_ context.Context, instanceID sharedkernel.InstanceID, q ListByInstanceQuery) ([]*Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if instanceID == "" {
		return nil, nil
	}
	var out []*Task
	skipped := 0
	for _, t := range r.byID {
		if t.InstanceID != instanceID || t.InstanceID == "" {
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

func (r *MemoryTaskRepository) List(_ context.Context, q AdminListQuery) ([]*Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var matched []*Task
	for _, t := range r.byID {
		if q.Status != "" && t.Status != q.Status {
			continue
		}
		if q.InstanceID != "" && t.InstanceID != q.InstanceID {
			continue
		}
		if q.ChatID != 0 && t.ChatID != q.ChatID {
			continue
		}
		if q.SessionID != "" && t.SessionID != q.SessionID {
			continue
		}
		if q.CaseID != "" && t.CaseID != q.CaseID {
			continue
		}
		if q.Q != "" {
			needle := strings.ToLower(q.Q)
			hay := strings.ToLower(string(t.ID) + " " + string(t.CaseID) + " " + string(t.SessionID))
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
