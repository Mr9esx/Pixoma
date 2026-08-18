package runtime

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/mr9esx/comfyui_tgbot/internal/channel/domain"
)

type fakeAdapter struct {
	id    string
	started int
	stopped int
}

func (f *fakeAdapter) Start(context.Context) error { f.started++; return nil }
func (f *fakeAdapter) Stop(context.Context) error  { f.stopped++; return nil }

type fakeFactory struct {
	mu       sync.Mutex
	created  map[string]int
	adapters map[string]*fakeAdapter
	failNext map[string]bool
}

func newFakeFactory() *fakeFactory {
	return &fakeFactory{created: map[string]int{}, adapters: map[string]*fakeAdapter{}, failNext: map[string]bool{}}
}

func (f *fakeFactory) Create(snap ChannelSnapshot) (Adapter, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	credential := snap.Credential
	f.created[credential]++
	if f.failNext[credential] {
		f.failNext[credential] = false
		return nil, errFakeStart
	}
	ad := &fakeAdapter{id: credential}
	f.adapters[credential] = ad
	return ad, nil
}

var errFakeStart = &fakeError{}

type fakeError struct{}

func (*fakeError) Error() string { return "fake start error" }

type memSnapshotStore struct {
	mu     sync.Mutex
	rows   map[string]ChannelSnapshot
	order  []string
	seq    int
}

func newMemStore() *memSnapshotStore {
	return &memSnapshotStore{rows: map[string]ChannelSnapshot{}}
}

func (s *memSnapshotStore) upsert(id, cred string, enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	if _, ok := s.rows[id]; !ok {
		s.order = append(s.order, id)
	}
	s.rows[id] = ChannelSnapshot{
		ID: id, Platform: string(domain.PlatformTelegram), Credential: cred,
		CredentialHash: hashCredential(cred), Enabled: enabled, UpdatedAt: time.Now().Add(time.Duration(s.seq) * time.Millisecond),
	}
}

func (s *memSnapshotStore) remove(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.rows, id)
}

func (s *memSnapshotStore) ListChannels(context.Context) ([]ChannelSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]ChannelSnapshot, 0, len(s.order))
	for _, id := range s.order {
		if ch, ok := s.rows[id]; ok {
			out = append(out, ch)
		}
	}
	return out, nil
}

func runOnce(a *Assembler, ctx context.Context) {
	a.reconcile(ctx)
}

func TestAssembler_StartStopRestartDelete(t *testing.T) {
	store := newMemStore()
	store.upsert("tg-1", "token-a", true)
	factory := newFakeFactory()
	as := &Assembler{Store: store, Factory: factory}

	ctx := context.Background()
	runOnce(as, ctx)
	if factory.adapters["token-a"] == nil || factory.adapters["token-a"].started != 1 {
		t.Fatalf("expect start once, got %+v", factory.adapters)
	}

	// 凭证变化 → 停旧建新
	store.upsert("tg-1", "token-b", true)
	runOnce(as, ctx)
	if factory.adapters["token-a"].stopped != 1 {
		t.Fatalf("old adapter must stop: %+v", factory.adapters["token-a"])
	}
	if factory.adapters["token-b"] == nil || factory.adapters["token-b"].started != 1 {
		t.Fatalf("new adapter must start: %+v", factory.adapters)
	}

	// 停用 → stop
	store.upsert("tg-1", "token-b", false)
	runOnce(as, ctx)
	if factory.adapters["token-b"].stopped != 1 {
		t.Fatalf("disabled must stop: %+v", factory.adapters["token-b"])
	}

	// 删除 → 不再重建
	store.remove("tg-1")
	runOnce(as, ctx)
	if factory.adapters["token-b"].started != 1 {
		t.Fatalf("deleted channel must not restart: %+v", factory.adapters["token-b"])
	}
}

func TestAssembler_StartFailureBackoffThenRetry(t *testing.T) {
	store := newMemStore()
	store.upsert("tg-1", "token-x", true)
	factory := newFakeFactory()
	factory.failNext["token-x"] = true
	as := &Assembler{Store: store, Factory: factory}

	ctx := context.Background()
	runOnce(as, ctx) // start fails -> error state
	if _, ok := as.adapters["tg-1"]; !ok {
		t.Fatal("expected error-state entry")
	}
	if as.adapters["tg-1"].state != stateError {
		t.Fatalf("state=%s", as.adapters["tg-1"].state)
	}

	// 下一轮重试成功
	runOnce(as, ctx)
	if factory.adapters["token-x"] == nil || factory.adapters["token-x"].started != 1 {
		t.Fatalf("retry must start adapter: %+v", factory.adapters)
	}
	if as.adapters["tg-1"].state != stateRunning {
		t.Fatalf("state=%s", as.adapters["tg-1"].state)
	}
}

func TestAssembler_CancelStopsAll(t *testing.T) {
	store := newMemStore()
	store.upsert("tg-1", "token-a", true)
	store.upsert("tg-2", "token-b", true)
	factory := newFakeFactory()
	as := &Assembler{Store: store, Factory: factory}

	ctx, cancel := context.WithCancel(context.Background())
	runOnce(as, ctx)
	cancel()
	if err := as.StopAll(context.Background()); err != nil {
		t.Fatal(err)
	}
	if factory.adapters["token-a"].stopped != 1 || factory.adapters["token-b"].stopped != 1 {
		t.Fatalf("all adapters must stop: %+v", factory.adapters)
	}
}
