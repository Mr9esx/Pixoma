package application

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/mr9esx/comfyui_tgbot/internal/channel/domain"
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
	svc := &Service{Store: store, Key: key, HasActiveRefs: func(context.Context, string) (bool, error) { return false, nil }}

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

func TestService_DeleteRestricted(t *testing.T) {
	store := &memStore{rows: map[string]domain.Channel{}}
	key := make([]byte, 32)
	refs := func(context.Context, string) (bool, error) { return false, nil }
	svc := &Service{Store: store, Key: key, HasActiveRefs: refs}

	if _, err := svc.Create(context.Background(), "tg-1", domain.PlatformTelegram, "a", "t"); err != nil {
		t.Fatal(err)
	}

	// 启用中删除被拒
	if err := svc.Delete(context.Background(), "tg-1"); !errors.Is(err, domain.ErrDeleteRestricted) {
		t.Fatalf("enabled delete: %v", err)
	}
	// 禁用但有活跃引用被拒
	if err := svc.Disable(context.Background(), "tg-1"); err != nil {
		t.Fatal(err)
	}
	svc.HasActiveRefs = func(context.Context, string) (bool, error) { return true, nil }
	if err := svc.Delete(context.Background(), "tg-1"); !errors.Is(err, domain.ErrDeleteRestricted) {
		t.Fatalf("referenced delete: %v", err)
	}
	// 禁用且无引用可删
	svc.HasActiveRefs = refs
	if err := svc.Delete(context.Background(), "tg-1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := svc.Get(context.Background(), "tg-1"); err != domain.ErrNotFound {
		t.Fatalf("want not found, got %v", err)
	}
}
