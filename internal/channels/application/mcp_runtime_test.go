package application

import (
	"context"
	"testing"

	"github.com/Mr9esx/Pixoma/internal/channels/domain"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
)

func TestCreateMCPChannel_NoIMAdapter(t *testing.T) {
	store := newMemStore()
	store.upsertPlatform("mcp-1", string(domain.PlatformMCP), "", true)
	factory := newFakeFactory()
	as := &Assembler{Store: store, Factory: factory}

	runOnce(as, context.Background())
	if len(factory.created) != 0 {
		t.Fatalf("mcp channel must not start an IM adapter: %+v", factory.created)
	}
	st := as.Status()["mcp-1"]
	if st.State != stateAbsent {
		t.Fatalf("mcp adapter state=%q want absent", st.State)
	}
}

func TestNotify_SkipsMCPPlatform(t *testing.T) {
	handler := &fakeNotifyHandler{}
	router := &NotifyRouter{
		HandlerByChannel: func(string) (NotifyHandler, bool) {
			return handler, true
		},
		PlatformOf: func(_ context.Context, id string) (string, bool) {
			if id == "mcp-1" {
				return string(domain.PlatformMCP), true
			}
			return string(domain.PlatformTelegram), true
		},
	}
	n := sharedkernel.UserNotify{ChatID: "mcp-1:ext-1", TaskID: "t1", Kind: "task_succeeded"}
	if err := router.Publish(context.Background(), n); err != nil {
		t.Fatal(err)
	}
	if len(handler.calls) != 0 {
		t.Fatalf("mcp notify must skip IM handler: %+v", handler.calls)
	}

	n.ChatID = "tg:123"
	if err := router.Publish(context.Background(), n); err != nil {
		t.Fatal(err)
	}
	if len(handler.calls) != 1 {
		t.Fatalf("telegram notify still dispatches: %+v", handler.calls)
	}
}

func (s *memSnapshotStore) upsertPlatform(id, platform, cred string, enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	if _, ok := s.rows[id]; !ok {
		s.order = append(s.order, id)
	}
	s.rows[id] = ChannelSnapshot{
		ID:             id,
		Platform:       platform,
		Credential:     cred,
		CredentialHash: hashCredential(cred),
		Enabled:        enabled,
	}
}
