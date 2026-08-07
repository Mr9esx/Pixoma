package domain

import (
	"context"
	"sync"
	"time"

	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

type Repository interface {
	GetActiveByChat(ctx context.Context, chatID sharedkernel.ChatID) (*Session, error)
	Save(ctx context.Context, s *Session) error
	ClearActive(ctx context.Context, chatID sharedkernel.ChatID) error
}

// MemoryRepository is an in-memory active-session store for tests and Phase1.
type MemoryRepository struct {
	mu   sync.Mutex
	byChat map[sharedkernel.ChatID]*Session
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{byChat: map[sharedkernel.ChatID]*Session{}}
}

func (r *MemoryRepository) GetActiveByChat(_ context.Context, chatID sharedkernel.ChatID) (*Session, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.byChat[chatID]
	if !ok || !s.Status.IsActive() {
		return nil, ErrNoActiveSession
	}
	cp := *s
	return &cp, nil
}

func (r *MemoryRepository) Save(_ context.Context, s *Session) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *s
	if cp.Draft != nil {
		cp.Draft = copyDraft(s.Draft)
	}
	if s.Status.IsActive() {
		r.byChat[s.ChatID] = &cp
	} else {
		delete(r.byChat, s.ChatID)
	}
	return nil
}

func (r *MemoryRepository) ClearActive(_ context.Context, chatID sharedkernel.ChatID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.byChat, chatID)
	return nil
}

func copyDraft(in map[string]DraftValue) map[string]DraftValue {
	out := make(map[string]DraftValue, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

type IDGen func() sharedkernel.SessionID

type Clock func() time.Time

type Service struct {
	repo  Repository
	idGen IDGen
	now   Clock
}

func NewService(repo Repository, idGen IDGen, now Clock) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{repo: repo, idGen: idGen, now: now}
}

func (svc *Service) StartCase(ctx context.Context, chatID sharedkernel.ChatID, caseID sharedkernel.CaseID, inputKeys []string) (*Session, error) {
	if cur, err := svc.repo.GetActiveByChat(ctx, chatID); err == nil && cur.Status.IsActive() {
		return nil, ErrSessionLocked
	} else if err != nil && err != ErrNoActiveSession {
		return nil, err
	}
	s := NewCollecting(svc.idGen(), chatID, caseID, inputKeys, svc.now())
	if err := svc.repo.Save(ctx, s); err != nil {
		return nil, err
	}
	return s, nil
}

func (svc *Service) SubmitInput(ctx context.Context, chatID sharedkernel.ChatID, v DraftValue) (*Session, error) {
	s, err := svc.repo.GetActiveByChat(ctx, chatID)
	if err != nil {
		return nil, err
	}
	if err := s.SubmitInput(v, svc.now()); err != nil {
		return nil, err
	}
	if err := svc.repo.Save(ctx, s); err != nil {
		return nil, err
	}
	return s, nil
}

func (svc *Service) SkipInput(ctx context.Context, chatID sharedkernel.ChatID) (*Session, error) {
	s, err := svc.repo.GetActiveByChat(ctx, chatID)
	if err != nil {
		return nil, err
	}
	if err := s.SkipInput(svc.now()); err != nil {
		return nil, err
	}
	if err := svc.repo.Save(ctx, s); err != nil {
		return nil, err
	}
	return s, nil
}

func (svc *Service) Exit(ctx context.Context, chatID sharedkernel.ChatID) error {
	s, err := svc.repo.GetActiveByChat(ctx, chatID)
	if err != nil {
		return err
	}
	if err := s.Exit(svc.now()); err != nil {
		return err
	}
	return svc.repo.Save(ctx, s)
}

func (svc *Service) Get(ctx context.Context, chatID sharedkernel.ChatID) (*Session, error) {
	return svc.repo.GetActiveByChat(ctx, chatID)
}

func (svc *Service) BeginConfirm(ctx context.Context, chatID sharedkernel.ChatID) (*Session, error) {
	s, err := svc.repo.GetActiveByChat(ctx, chatID)
	if err != nil {
		return nil, err
	}
	if s.Status == StatusConfirming {
		return s, nil
	}
	if s.Status != StatusCollecting || s.CurrentInputIndex < len(s.InputKeys) {
		return nil, ErrInvalidState
	}
	s.Status = StatusConfirming
	s.UpdatedAt = svc.now()
	if err := svc.repo.Save(ctx, s); err != nil {
		return nil, err
	}
	return s, nil
}
