package capability

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/mr9esx/comfyui_tgbot/internal/channel/protocol"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

type stubCap struct{}

func (stubCap) ID() string                  { return "stub" }
func (stubCap) DisplayName() string         { return "Stub" }
func (stubCap) ParamsSchema() json.RawMessage { return json.RawMessage(`{"type":"object"}`) }
func (stubCap) Render(_ string, override map[string]any) (protocol.RenderDecl, error) {
	return MergeRender(protocol.RenderDecl{Entry: "root", Config: map[string]any{"columns": 2}}, override), nil
}
func (stubCap) Invoke(context.Context, protocol.AccountCtx, protocol.Nav, sharedkernel.ChatID, map[string]any) (protocol.Result, error) {
	return protocol.Result{Text: "stub ok"}, nil
}

func TestRegistryRegisterDuplicate(t *testing.T) {
	r := NewRegistry()
	if err := r.Register(stubCap{}); err != nil {
		t.Fatal(err)
	}
	if err := r.Register(stubCap{}); err == nil {
		t.Fatal("duplicate registration must error")
	}
}

func TestRegistryGetAndList(t *testing.T) {
	r := NewRegistry()
	if err := r.Register(stubCap{}); err != nil {
		t.Fatal(err)
	}
	c, ok := r.Get("stub")
	if !ok || c.ID() != "stub" {
		t.Fatalf("get: %+v %v", c, ok)
	}
	if _, ok := r.Get("nope"); ok {
		t.Fatal("unknown capability must not be found")
	}
	list := r.List()
	if len(list) != 1 || list[0].ID() != "stub" {
		t.Fatalf("list=%d", len(list))
	}
}

func TestMergeRender(t *testing.T) {
	base := protocol.RenderDecl{Entry: "root", Config: map[string]any{"columns": 2}}
	merged := MergeRender(base, map[string]any{"columns": 3})
	if merged.Config["columns"] != 3 {
		t.Fatalf("override columns: %+v", merged.Config)
	}
	if merged.Entry != "root" {
		t.Fatalf("entry changed: %+v", merged)
	}
	// 未覆盖的 key 保留
	merged2 := MergeRender(base, map[string]any{"rows": 2})
	if merged2.Config["columns"] != 2 {
		t.Fatalf("uncovered key lost: %+v", merged2.Config)
	}
}

func TestRegistryInvokeRejectsUnknown(t *testing.T) {
	r := NewRegistry()
	if _, err := r.Invoke(context.Background(), protocol.CapabilityInvoke{CapabilityID: "nope"}); err == nil {
		t.Fatal("unknown capability invoke must error")
	}
	if err := r.Register(stubCap{}); err != nil {
		t.Fatal(err)
	}
	res, err := r.Invoke(context.Background(), protocol.CapabilityInvoke{CapabilityID: "stub"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Text != "stub ok" {
		t.Fatalf("res=%+v", res)
	}
}

var _ = errors.New
