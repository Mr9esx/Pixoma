package capability_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mr9esx/comfyui_tgbot/internal/channel/capability"
	"github.com/mr9esx/comfyui_tgbot/internal/channel/protocol"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

func TestReplyTextInvokeReturnsText(t *testing.T) {
	res, err := (capability.ReplyText{}).Invoke(context.Background(), protocol.AccountCtx{}, protocol.Nav{}, "tg:1", map[string]any{"text": "即将上线"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Text != "即将上线" || len(res.Options) != 0 {
		t.Fatalf("res=%+v", res)
	}
}

func TestReplyMediaInvokeReturnsMediaURLs(t *testing.T) {
	res, err := (capability.ReplyMedia{}).Invoke(context.Background(), protocol.AccountCtx{}, protocol.Nav{}, "tg:1", map[string]any{
		"text":   "联系方式",
		"images": []any{"https://a/qr.png"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Text != "联系方式" || len(res.MediaURLs) != 1 || res.MediaURLs[0] != "https://a/qr.png" {
		t.Fatalf("res=%+v", res)
	}
}

func TestOpenCaseDisplayNameIsWorkflow(t *testing.T) {
	if got := (capability.OpenCase{}).DisplayName(); got != "打开工作流" {
		t.Fatalf("display_name=%q", got)
	}
}

func TestAdminSchemaReturnsOpenCaseCaseIDsOnly(t *testing.T) {
	schema := capability.AdminSchema(capability.OpenCase{})
	var doc map[string]any
	if err := json.Unmarshal(schema, &doc); err != nil {
		t.Fatal(err)
	}
	props, _ := doc["properties"].(map[string]any)
	if _, ok := props["case_ids"]; !ok {
		t.Fatalf("case_ids missing: %v", props)
	}
	if _, ok := props["step"]; ok {
		t.Fatalf("step must not be admin-configurable: %v", props)
	}
}

func TestAdminSchemaFallsBackToParamsSchema(t *testing.T) {
	if got := capability.AdminSchema(stubCap{}); string(got) != `{"type":"object"}` {
		t.Fatalf("fallback schema=%s", got)
	}
}

type stubCap struct{}

func (stubCap) ID() string                    { return "stub" }
func (stubCap) DisplayName() string           { return "stub" }
func (stubCap) ParamsSchema() json.RawMessage { return json.RawMessage(`{"type":"object"}`) }
func (stubCap) Render(string, map[string]any) (protocol.RenderDecl, error) {
	return protocol.RenderDecl{}, nil
}
func (stubCap) Invoke(context.Context, protocol.AccountCtx, protocol.Nav, sharedkernel.ChatID, map[string]any) (protocol.Result, error) {
	return protocol.Result{}, nil
}
