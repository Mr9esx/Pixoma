package domain

import (
	"context"

	"errors"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
)

var ErrEmptyUserID = errors.New("empty user id")

// ListQuery filters sessions for admin list.
type ListQuery struct {
	Q           string
	UserID      string
	ChatID      *sharedkernel.ChatID
	CaseID      sharedkernel.CaseID
	ChannelID   string
	Status      Status
	CreatedFrom *time.Time
	CreatedTo   *time.Time
	Limit       int
	Offset      int
}

type Repository interface {
	GetActiveByChat(ctx context.Context, chatID sharedkernel.ChatID) (*Session, error)
	GetByID(ctx context.Context, id sharedkernel.SessionID) (*Session, error)
	List(ctx context.Context, q ListQuery) ([]*Session, error)
	// ListActiveByCase returns collecting/confirming sessions for a case.
	ListActiveByCase(ctx context.Context, caseID sharedkernel.CaseID) ([]*Session, error)
	Save(ctx context.Context, s *Session) error
	ClearActive(ctx context.Context, chatID sharedkernel.ChatID) error
}

// MemoryRepository is an in-memory session store for tests.
// Inactive sessions are retained by ID (no physical delete) for Task join.
type MemoryRepository struct {
	mu     sync.Mutex
	byChat map[sharedkernel.ChatID]*Session // active index only
	byID   map[sharedkernel.SessionID]*Session
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		byChat: map[sharedkernel.ChatID]*Session{},
		byID:   map[sharedkernel.SessionID]*Session{},
	}
}

func (r *MemoryRepository) GetActiveByChat(_ context.Context, chatID sharedkernel.ChatID) (*Session, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.byChat[chatID]
	if !ok || !s.Status.IsActive() {
		return nil, ErrNoActiveSession
	}
	return cloneSession(s), nil
}

func (r *MemoryRepository) GetByID(_ context.Context, id sharedkernel.SessionID) (*Session, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.byID[id]
	if !ok {
		return nil, ErrNotFound
	}
	return cloneSession(s), nil
}

func (r *MemoryRepository) List(_ context.Context, q ListQuery) ([]*Session, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*Session, 0)
	for _, s := range r.byID {
		if q.UserID != "" && s.UserID != q.UserID {
			continue
		}
		if q.CaseID != 0 && s.CaseID != q.CaseID {
			continue
		}
		if q.ChannelID != "" {
			addr, err := sharedkernel.ParseChatID(string(s.ChatID))
			if err != nil || addr.ChannelID != q.ChannelID {
				continue
			}
		}
		if q.ChatID != nil && s.ChatID != *q.ChatID {
			continue
		}
		if q.Status != "" && s.Status != q.Status {
			continue
		}
		if q.CreatedFrom != nil && s.CreatedAt.Before(*q.CreatedFrom) {
			continue
		}
		if q.CreatedTo != nil && s.CreatedAt.After(*q.CreatedTo) {
			continue
		}
		if q.Q != "" {
			if !strings.Contains(string(s.ID), q.Q) &&
				!strings.Contains(strconv.FormatUint(uint64(s.CaseID), 10), q.Q) &&
				!strings.Contains(s.UserID, q.Q) {
				continue
			}
		}
		out = append(out, cloneSession(s))
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].UpdatedAt.After(out[j].UpdatedAt)
	})
	if q.Offset > 0 {
		if q.Offset >= len(out) {
			return []*Session{}, nil
		}
		out = out[q.Offset:]
	}
	if q.Limit > 0 && len(out) > q.Limit {
		out = out[:q.Limit]
	}
	return out, nil
}

func (r *MemoryRepository) ListActiveByCase(_ context.Context, caseID sharedkernel.CaseID) ([]*Session, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*Session
	for _, s := range r.byID {
		if s.CaseID != caseID || !s.Status.IsActive() {
			continue
		}
		out = append(out, cloneSession(s))
	}
	return out, nil
}

func (r *MemoryRepository) Save(_ context.Context, s *Session) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := cloneSession(s)
	r.byID[s.ID] = cp
	if s.Status.IsActive() {
		r.byChat[s.ChatID] = cp
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

func cloneSession(s *Session) *Session {
	cp := *s
	if s.Draft != nil {
		cp.Draft = copyDraft(s.Draft)
	}
	if s.InputKeys != nil {
		cp.InputKeys = append([]string(nil), s.InputKeys...)
	}
	return &cp
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

func (svc *Service) StartCase(ctx context.Context, chatID sharedkernel.ChatID, userID string, caseID sharedkernel.CaseID, inputKeys []string) (*Session, error) {
	if userID == "" {
		return nil, ErrEmptyUserID
	}
	if cur, err := svc.repo.GetActiveByChat(ctx, chatID); err == nil && cur.Status.IsActive() {
		// 用户重新开启新会话时，旧会话自动退出，避免被锁住报错。
		if err := cur.Exit(svc.now()); err != nil {
			return nil, err
		}
		if err := svc.repo.Save(ctx, cur); err != nil {
			return nil, err
		}
	} else if err != nil && err != ErrNoActiveSession {
		return nil, err
	}
	s := NewCollecting(svc.idGen(), chatID, caseID, inputKeys, svc.now())
	s.UserID = userID
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
