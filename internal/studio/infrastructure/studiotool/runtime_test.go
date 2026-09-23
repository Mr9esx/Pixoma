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
	events []toolEvent
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
func (*toolSink) CreateFlowNode(context.Context, studioapp.FlowNodeInput) (*domain.FlowNode, error) {
	return nil, nil
}
func (*toolSink) CreateFlowEdge(context.Context, string, string, string) (*domain.FlowEdge, error) {
	return nil, nil
}
func (*toolSink) RequestApproval(context.Context, string, string, string) (*domain.Approval, error) {
	return nil, nil
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
	if len(tools) != 2 {
		t.Fatalf("tools = %d, want create and read", len(tools))
	}
	reader, ok := tools[1].(einotool.InvokableTool)
	if !ok {
		t.Fatalf("read tool is not invokable: %T", tools[1])
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
