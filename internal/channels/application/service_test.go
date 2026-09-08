package application

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"github.com/Mr9esx/Pixoma/internal/channels/domain"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
)

func offlineFetch(_ context.Context, _ string) (json.RawMessage, error) {
	return nil, errors.New("offline")
}

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
	svc := &Service{Store: store, Key: key, FetchTelegram: offlineFetch}

	ch, err := svc.Create(context.Background(), "tg-default", domain.PlatformTelegram, "主机器人", "12345:TOKEN", "")
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
		Store:         store,
		Key:           key,
		Notify:        n,
		FetchTelegram: offlineFetch,
		DeleteWithCleanup: func(_ context.Context, id string) ([]sharedkernel.ChatID, error) {
			if err := store.Delete(context.Background(), id); err != nil {
				return nil, err
			}
			return []sharedkernel.ChatID{"tg:9"}, nil
		},
	}

	if _, err := svc.Create(context.Background(), "tg-1", domain.PlatformTelegram, "a", "t", ""); err != nil {
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
	svc := &Service{Store: store, Key: make([]byte, 32), FetchTelegram: offlineFetch}
	if _, err := svc.Create(context.Background(), "tg-1", domain.PlatformTelegram, "a", "t", ""); err != nil {
		t.Fatal(err)
	}
	if err := svc.Delete(context.Background(), "tg-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Get(context.Background(), "tg-1"); err != domain.ErrNotFound {
		t.Fatalf("want not found, got %v", err)
	}
}

func TestService_CreateStoresExtraInfo(t *testing.T) {
	store := &memStore{rows: map[string]domain.Channel{}}
	svc := &Service{Store: store, Key: make([]byte, 32), FetchTelegram: offlineFetch}
	const extra = `{"id":1,"username":"demo_bot","first_name":"Demo"}`
	ch, err := svc.Create(context.Background(), "tg-x", domain.PlatformTelegram, "Demo", "t", extra)
	if err != nil {
		t.Fatal(err)
	}
	if ch.ExtraInfo != extra {
		t.Fatalf("extra_info=%q", ch.ExtraInfo)
	}
	got, err := svc.Get(context.Background(), "tg-x")
	if err != nil {
		t.Fatal(err)
	}
	if got.ExtraInfo != extra {
		t.Fatalf("persisted extra_info=%q", got.ExtraInfo)
	}
}

func TestService_CreateAutoFetchesTelegramInfo(t *testing.T) {
	store := &memStore{rows: map[string]domain.Channel{}}
	svc := &Service{Store: store, Key: make([]byte, 32), FetchTelegram: func(context.Context, string) (json.RawMessage, error) {
		return json.RawMessage(`{"id":1,"username":"auto"}`), nil
	}}
	ch, err := svc.Create(context.Background(), "tg-z", domain.PlatformTelegram, "", "t", "")
	if err != nil {
		t.Fatal(err)
	}
	if ch.Name != "auto" {
		t.Fatalf("backfilled name=%q", ch.Name)
	}
	if ch.ExtraInfo != `{"id":1,"username":"auto"}` {
		t.Fatalf("auto extra_info=%q", ch.ExtraInfo)
	}
}

func TestService_CreatePrefersFirstNameForUnnamedChannel(t *testing.T) {
	store := &memStore{rows: map[string]domain.Channel{}}
	svc := &Service{Store: store, Key: make([]byte, 32), FetchTelegram: func(context.Context, string) (json.RawMessage, error) {
		return json.RawMessage(`{"id":1,"first_name":"Demo","username":"auto"}`), nil
	}}
	ch, err := svc.Create(context.Background(), "tg-f", domain.PlatformTelegram, "", "t", "")
	if err != nil {
		t.Fatal(err)
	}
	if ch.Name != "Demo" {
		t.Fatalf("backfilled name=%q", ch.Name)
	}
}

func TestService_CheckReachabilityPersistsLastCheck(t *testing.T) {
	store := &memStore{rows: map[string]domain.Channel{}}
	svc := &Service{
		Store:         store,
		Key:           make([]byte, 32),
		FetchTelegram: offlineFetch,
		CheckTelegram: func(context.Context, string) (ReachabilityResult, error) {
			return ReachabilityResult{OK: false, Kind: ReachabilityNetwork, Message: "i/o timeout"}, nil
		},
	}
	if _, err := svc.Create(context.Background(), "tg-1", domain.PlatformTelegram, "a", "t", ""); err != nil {
		t.Fatal(err)
	}
	res, err := svc.CheckReachability(context.Background(), "tg-1")
	if err != nil {
		t.Fatal(err)
	}
	if res.Kind != ReachabilityNetwork {
		t.Fatalf("kind=%s", res.Kind)
	}
	got, err := svc.Get(context.Background(), "tg-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.LastCheckKind != string(ReachabilityNetwork) || got.LastCheckMessage != "i/o timeout" || got.LastCheckAt == nil {
		t.Fatalf("persisted last check: %+v", got)
	}
}

func TestService_UpdateTokenClearsLastCheck(t *testing.T) {
	store := &memStore{rows: map[string]domain.Channel{}}
	svc := &Service{Store: store, Key: make([]byte, 32), FetchTelegram: offlineFetch}
	if _, err := svc.Create(context.Background(), "tg-1", domain.PlatformTelegram, "a", "t", ""); err != nil {
		t.Fatal(err)
	}
	now := svc.nowFn()().UTC()
	ch, err := svc.Get(context.Background(), "tg-1")
	if err != nil {
		t.Fatal(err)
	}
	ch.LastCheckKind = string(ReachabilityOK)
	ch.LastCheckMessage = "ok"
	ch.LastCheckAt = &now
	if err := store.Update(context.Background(), ch); err != nil {
		t.Fatal(err)
	}
	token := "new-token"
	got, err := svc.Update(context.Background(), "tg-1", "", &token)
	if err != nil {
		t.Fatal(err)
	}
	if got.LastCheckKind != "" || got.LastCheckMessage != "" || got.LastCheckAt != nil {
		t.Fatalf("token change must clear last check: %+v", got)
	}
}

func TestService_CreateRejectsInvalidExtraInfo(t *testing.T) {
	store := &memStore{rows: map[string]domain.Channel{}}
	svc := &Service{Store: store, Key: make([]byte, 32), FetchTelegram: offlineFetch}
	if _, err := svc.Create(context.Background(), "tg-y", domain.PlatformTelegram, "a", "t", "{bad"); err == nil {
		t.Fatal("want error for invalid extra_info")
	}
}
