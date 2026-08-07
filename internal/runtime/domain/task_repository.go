package domain

import (
	"context"
	"errors"
	"sync"

	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

var ErrTaskNotFound = errors.New("task not found")

type ListTaskQuery struct {
	Status sharedkernel.TaskStatus
	Limit  int
}

type TaskRepository interface {
	Create(ctx context.Context, t *Task) error
	Get(ctx context.Context, id sharedkernel.TaskID) (*Task, error)
	Update(ctx context.Context, t *Task) error
	ListByChat(ctx context.Context, chatID sharedkernel.ChatID, limit int) ([]*Task, error)
	ListByStatus(ctx context.Context, st sharedkernel.TaskStatus, limit int) ([]*Task, error)
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

func (r *MemoryTaskRepository) ListByChat(_ context.Context, chatID sharedkernel.ChatID, limit int) ([]*Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*Task
	for _, t := range r.byID {
		if t.ChatID != chatID {
			continue
		}
		cp := *t
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
		out = append(out, &cp)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}
