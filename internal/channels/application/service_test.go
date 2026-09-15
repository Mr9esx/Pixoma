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

func TestService_CreateFeishuEncryptsAndMasks(t *testing.T) {
	store := &memStore{rows: map[string]domain.Channel{}}
	svc := &Service{Store: store, Key: make([]byte, 32)}
	ch, err := svc.CreateWithCredential(context.Background(), "fs-default", domain.PlatformFeishu, "飞书助手", domain.Credential{AppID: "cli_abcdef012345", AppSecret: "supersecretvalue1234567"}, "")
	if err != nil {
		t.Fatalf("create feishu: %v", err)
	}
	if ch.Platform != string(domain.PlatformFeishu) {
		t.Fatalf("platform = %q", ch.Platform)
	}
	cred, err := domain.DecryptCredential(svc.Key, ch.CredentialCiphertext)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if cred.AppID != "cli_abcdef012345" || cred.AppSecret != "supersecretvalue1234567" {
		t.Fatalf("credential not round-tripped: %+v", cred)
	}
	masked, err := svc.Masked(context.Background(), ch.ID)
	if err != nil {
		t.Fatalf("masked: %v", err)
	}
	if masked == "supersecretvalue1234567" || masked == "" {
		t.Fatalf("masked leaks secret: %q", masked)
	}
}

func TestService_CreateFeishuRequiresAppIDAndSecret(t *testing.T) {
	store := &memStore{rows: map[string]domain.Channel{}}
	svc := &Service{Store: store, Key: make([]byte, 32)}
	if _, err := svc.CreateWithCredential(context.Background(), "fs-x", domain.PlatformFeishu, "x", domain.Credential{AppID: "cli_1", AppSecret: ""}, ""); err == nil {
		t.Fatal("expected error for missing app_secret")
	}
	if _, err := svc.CreateWithCredential(context.Background(), "fs-x", domain.PlatformFeishu, "x", domain.Credential{AppID: "", AppSecret: "s"}, ""); err == nil {
		t.Fatal("expected error for missing app_id")
	}
}

func TestService_CreateWeComEncryptsAndMasks(t *testing.T) {
	store := &memStore{rows: map[string]domain.Channel{}}
	svc := &Service{Store: store, Key: make([]byte, 32)}
	ch, err := svc.CreateWithCredential(context.Background(), "wc-default", domain.PlatformWeCom, "企微助手", domain.Credential{
		WeComBotID: "aibot_1", WeComSecret: "secret-12345678", WeComWSURL: "wss://private.example/ws",
	}, "")
	if err != nil {
		t.Fatalf("create wecom: %v", err)
	}
	cred, err := domain.DecryptCredential(svc.Key, ch.CredentialCiphertext)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if cred.WeComBotID != "aibot_1" || cred.WeComSecret != "secret-12345678" || cred.WeComWSURL != "wss://private.example/ws" {
		t.Fatalf("credential = %#v", cred)
	}
	masked, err := svc.Masked(context.Background(), ch.ID)
	if err != nil {
		t.Fatal(err)
	}
	if masked == cred.WeComSecret || masked == "" {
		t.Fatalf("masked = %q", masked)
	}
}

func TestService_CreateWeComRequiresBotIDAndSecret(t *testing.T) {
	svc := &Service{Store: &memStore{rows: map[string]domain.Channel{}}, Key: make([]byte, 32)}
	if _, err := svc.CreateWithCredential(context.Background(), "wc-1", domain.PlatformWeCom, "企微助手", domain.Credential{WeComBotID: "aibot_1"}, ""); err == nil {
		t.Fatal("want error for missing WeCom secret")
	}
	if _, err := svc.CreateWithCredential(context.Background(), "wc-1", domain.PlatformWeCom, "企微助手", domain.Credential{WeComSecret: "secret-12345678"}, ""); err == nil {
		t.Fatal("want error for missing WeCom bot ID")
	}
}

func TestService_CheckReachabilityWeComUsesAdapterStatus(t *testing.T) {
	store := &memStore{rows: map[string]domain.Channel{}}
	svc := &Service{
		Store: store,
		Key:   make([]byte, 32),
		AdapterStatus: func(_ context.Context, id string) (string, string, bool) {
			if id != "wc-1" {
				t.Fatalf("adapter status channel = %q", id)
			}
			return "running", "", true
		},
	}
	if _, err := svc.CreateWithCredential(context.Background(), "wc-1", domain.PlatformWeCom, "企微助手", domain.Credential{WeComBotID: "aibot_1", WeComSecret: "secret-12345678"}, ""); err != nil {
		t.Fatal(err)
	}
	got, err := svc.CheckReachability(context.Background(), "wc-1")
	if err != nil {
		t.Fatal(err)
	}
	if !got.OK || got.Kind != ReachabilityOK {
		t.Fatalf("reachability = %#v", got)
	}
}

func TestService_CheckReachabilityFeishuNeverUsesTelegram(t *testing.T) {
	store := &memStore{rows: map[string]domain.Channel{}}
	svc := &Service{Store: store, Key: make([]byte, 32)}
	if _, err := svc.CreateWithCredential(context.Background(), "fs-1", domain.PlatformFeishu, "f", domain.Credential{AppID: "cli_1", AppSecret: "secretsecretsecret"}, ""); err != nil {
		t.Fatalf("create: %v", err)
	}
	var tgCalled bool
	svc.CheckTelegram = func(_ context.Context, _ string) (ReachabilityResult, error) {
		tgCalled = true
		return ReachabilityResult{OK: true, Kind: ReachabilityOK}, nil
	}
	svc.CheckFeishu = func(_ context.Context, appID, _ string) (ReachabilityResult, error) {
		if appID != "cli_1" {
			t.Fatalf("appID = %q", appID)
		}
		return ReachabilityResult{OK: true, Kind: ReachabilityOK}, nil
	}
	res, err := svc.CheckReachability(context.Background(), "fs-1")
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if !res.OK {
		t.Fatalf("expected ok, got %+v", res)
	}
	if tgCalled {
		t.Fatal("feishu reachability must not call telegram getMe")
	}
	// last_check persisted on the channel
	got, _ := svc.Get(context.Background(), "fs-1")
	if got.LastCheckKind != string(ReachabilityOK) {
		t.Fatalf("last_check_kind = %q", got.LastCheckKind)
	}
}
