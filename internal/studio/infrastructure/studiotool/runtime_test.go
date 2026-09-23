package studiotool

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	einotool "github.com/cloudwego/eino/components/tool"

	"github.com/Mr9esx/Pixoma/internal/platform/blob"
	"github.com/Mr9esx/Pixoma/internal/platform/blob/localfs"
	studioapp "github.com/Mr9esx/Pixoma/internal/studio/application"
	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

type toolEvent struct {
	typ     string
	payload map[string]any
}

type toolSink struct {
	events             []toolEvent
	updatedAssetID     string
	updatedBaseVersion string
	updatedAction      string
	updatedContent     []byte
	flowNodes          []studioapp.FlowNodeInput
}

func (s *toolSink) Emit(_ context.Context, typ string, value any) error {
	raw, _ := json.Marshal(value)
	var payload map[string]any
	_ = json.Unmarshal(raw, &payload)
	s.events = append(s.events, toolEvent{typ: typ, payload: payload})
	return nil
}
func (*toolSink) AssistantMessage(context.Context, string) (*domain.Message, error) { return nil, nil }
func (*toolSink) CreateAsset(context.Context, studioapp.GeneratedAsset) (*domain.Asset, error) {
	return nil, nil
}
func (s *toolSink) CreateFlowNode(_ context.Context, input studioapp.FlowNodeInput) (*domain.FlowNode, error) {
	s.flowNodes = append(s.flowNodes, input)
	return &domain.FlowNode{ID: "flow-node"}, nil
}
func (*toolSink) CreateFlowEdge(context.Context, string, string, string) (*domain.FlowEdge, error) {
	return nil, nil
}
func (*toolSink) RequestApproval(context.Context, string, string, string) (*domain.Approval, error) {
	return nil, nil
}
func (s *toolSink) AppendTextAssetVersion(_ context.Context, assetID, baseVersionID, action string, content []byte) (*domain.Asset, domain.AssetVersion, error) {
	s.updatedAssetID = assetID
	s.updatedBaseVersion = baseVersionID
	s.updatedAction = action
	s.updatedContent = append([]byte(nil), content...)
	version := domain.AssetVersion{ID: "version-2", AssetID: assetID, AccountID: "account-a", Version: 2, MIMEType: "text/markdown"}
	return &domain.Asset{ID: assetID, SessionID: "session-1", AccountID: "account-a", Name: "story.md", Kind: domain.AssetDocument, CurrentVersion: 2, Versions: []domain.AssetVersion{version}}, version, nil
}

func TestReadAssetToolReadsOnlyThePinnedSelectedVersion(t *testing.T) {
	store, err := localfs.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ref, err := store.Put(context.Background(), "studio/account-a/story/v1.md", bytes.NewBufferString("# 雨夜侦探"), blob.PutOptions{MIME: "text/markdown"})
	if err != nil {
		t.Fatal(err)
	}
	asset, err := domain.NewAsset("asset-1", "session-1", "account-a", "story.md", domain.AssetDocument, domain.AssetOriginUser, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := asset.AppendVersion("version-1", "text/markdown", ref.Key, ref.Size, time.Now()); err != nil {
		t.Fatal(err)
	}
	sink := &toolSink{}
	tools, err := NewRuntimeTools(ToolAccess{Sink: sink, Blob: store, Assets: []*domain.Asset{asset}})
	if err != nil {
		t.Fatal(err)
	}
	if len(tools) != 3 {
		t.Fatalf("tools = %d, want create, read and update", len(tools))
	}
	var reader einotool.InvokableTool
	for _, candidate := range tools {
		info, infoErr := candidate.Info(context.Background())
		if infoErr == nil && info.Name == "read_asset" {
			reader, _ = candidate.(einotool.InvokableTool)
			break
		}
	}
	if reader == nil {
		t.Fatal("read_asset is not invokable")
	}
	result, err := reader.InvokableRun(context.Background(), `{"asset_id":"asset-1"}`)
	if err != nil {
		t.Fatal(err)
	}
	if result == "" || !bytes.Contains([]byte(result), []byte("雨夜侦探")) {
		t.Fatalf("result = %q", result)
	}
	if len(sink.events) == 0 {
		t.Fatal("tool events are empty")
	}
	if sink.events[0].typ != studioapp.EventToolCallStart || sink.events[0].payload["tool_call_id"] == nil {
		t.Fatalf("tool start event = %#v", sink.events[0])
	}
	if len(sink.events) < 3 || sink.events[1].typ != studioapp.EventToolCallArgs || sink.events[2].typ != studioapp.EventToolCallResult || sink.events[2].payload["content"] == nil {
		t.Fatalf("tool transcript events = %#v", sink.events)
	}
}

func TestUpdateTextAssetToolAppendsVersionToPinnedSelectedAsset(t *testing.T) {
	asset, err := domain.NewAsset("asset-1", "session-1", "account-a", "story.md", domain.AssetDocument, domain.AssetOriginUser, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := asset.AppendVersion("version-1", "text/markdown", "studio/account-a/session-1/asset-1/v1.md", 8, time.Now()); err != nil {
		t.Fatal(err)
	}
	sink := &toolSink{}
	tools, err := NewRuntimeTools(ToolAccess{Sink: sink, Assets: []*domain.Asset{asset}, PermissionMode: domain.PermissionAutoApprove})
	if err != nil {
		t.Fatal(err)
	}
	var update einotool.InvokableTool
	for _, candidate := range tools {
		info, infoErr := candidate.Info(context.Background())
		if infoErr == nil && info.Name == "update_text_asset" {
			update, _ = candidate.(einotool.InvokableTool)
			break
		}
	}
	if update == nil {
		t.Fatal("update_text_asset is not available for a pinned text asset")
	}

	result, err := update.InvokableRun(context.Background(), `{"asset_id":"asset-1","content":"# 更新后的故事"}`)
	if err != nil {
		t.Fatal(err)
	}
	if result != "已更新 Markdown 资产「story.md」（v2）。" {
		t.Fatalf("result = %q", result)
	}
	if sink.updatedAssetID != "asset-1" || sink.updatedBaseVersion != "version-1" || string(sink.updatedContent) != "# 更新后的故事" || sink.updatedAction == "" {
		t.Fatalf("update request = asset=%q base=%q action=%q content=%q", sink.updatedAssetID, sink.updatedBaseVersion, sink.updatedAction, sink.updatedContent)
	}
	if len(sink.flowNodes) != 1 || sink.flowNodes[0].AssetVersionID != "version-2" {
		t.Fatalf("flow nodes = %#v", sink.flowNodes)
	}
}
