package application

import (
	"context"
	"sync"
	"testing"

	"github.com/mr9esx/comfyui_tgbot/internal/channel/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

type memStore struct {
	mu   sync.Mutex
	rows map[string]domain.Channel
}

func (s *memStore) Create(_ context.Context, ch domain.Channel) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rows[ch.ID] = ch
	return nil
}
func (s *memStore) Get(_ context.Context, id string) (domain.Channel, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ch, ok := s.rows[id]
	if !ok {
		return domain.Channel{}, domain.ErrNotFound
	}
	return ch, nil
}
func (s *memStore) List(_ context.Context) ([]domain.Channel, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]domain.Channel, 0, len(s.rows))
	for _, ch := range s.rows {
		out = append(out, ch)
	}
	return out, nil
}
func (s *memStore) Update(_ context.Context, ch domain.Channel) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rows[ch.ID] = ch
	return nil
}
func (s *memStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.rows, id)
	return nil
}

func TestService_CreateEncryptsAndGetDecrypts(t *testing.T) {
	store := &memStore{rows: map[string]domain.Channel{}}
	key := make([]byte, 32)
	svc := &Service{Store: store, Key: key}

	ch, err := svc.Create(context.Background(), "tg-default", domain.PlatformTelegram, "主机器人", "12345:TOKEN")
	if err != nil {
		t.Fatal(err)
	}
	if ch.CredentialCiphertext == "12345:TOKEN" {
		t.Fatal("credential must be encrypted at rest")
	}
	got, err := svc.Get(context.Background(), "tg-default")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "主机器人" || !got.Enabled {
		t.Fatalf("get: %+v", got)
	}
}

type memNotify struct {
	items []sharedkernel.UserNotify
}

func (m *memNotify) Publish(_ context.Context, n sharedkernel.UserNotify) error {
	m.items = append(m.items, n)
	return nil
}

func TestService_DeleteDirectWithCleanup(t *testing.T) {
	store := &memStore{rows: map[string]domain.Channel{}}
	key := make([]byte, 32)
	n := &memNotify{}
	svc := &Service{
		Store:  store,
		Key:    key,
		Notify: n,
		DeleteWithCleanup: func(_ context.Context, id string) ([]sharedkernel.ChatID, error) {
			if err := store.Delete(context.Background(), id); err != nil {
				return nil, err
			}
			return []sharedkernel.ChatID{"tg:9"}, nil
		},
	}

	if _, err := svc.Create(context.Background(), "tg-1", domain.PlatformTelegram, "a", "t"); err != nil {
		t.Fatal(err)
	}

	// 启用中直接删除成功，不再要求停用。
	if err := svc.Delete(context.Background(), "tg-1"); err != nil {
		t.Fatalf("enabled delete: %v", err)
	}
	if _, err := svc.Get(context.Background(), "tg-1"); err != domain.ErrNotFound {
		t.Fatalf("want not found, got %v", err)
	}
	if len(n.items) != 1 || n.items[0].Kind != "session_terminated" {
		t.Fatalf("notifies=%+v", n.items)
	}
}

func TestService_DeleteFallsBackToStore(t *testing.T) {
	store := &memStore{rows: map[string]domain.Channel{}}
	svc := &Service{Store: store, Key: make([]byte, 32)}
	if _, err := svc.Create(context.Background(), "tg-1", domain.PlatformTelegram, "a", "t"); err != nil {
		t.Fatal(err)
	}
	if err := svc.Delete(context.Background(), "tg-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Get(context.Background(), "tg-1"); err != domain.ErrNotFound {
		t.Fatalf("want not found, got %v", err)
	}
}
