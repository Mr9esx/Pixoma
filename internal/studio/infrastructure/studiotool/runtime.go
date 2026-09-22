package studiotool

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"

	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	einojsonschema "github.com/eino-contrib/jsonschema"
	"github.com/google/uuid"

	studioapp "github.com/Mr9esx/Pixoma/internal/studio/application"
	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

// ToolAccess binds built-in Studio capabilities to one audited Agent run.
type ToolAccess struct {
	PermissionMode  domain.PermissionMode
	IsApproved      func(action string) bool
	RequestApproval func(context.Context, string, string) error
	Sink            studioapp.AgentSink
}

// NewRuntimeTools returns local authoring capabilities. They intentionally use
// the AgentSink rather than persistence directly so asset creation, Flow
// updates, events, and trace visibility share exactly one execution path.
func NewRuntimeTools(access ToolAccess) ([]einotool.BaseTool, error) {
	if access.Sink == nil {
		return nil, fmt.Errorf("studio: built-in tool access is not configured")
	}
	var rawSchema einojsonschema.Schema
	if err := json.Unmarshal([]byte(`{"type":"object","additionalProperties":false,"required":["name","content"],"properties":{"name":{"type":"string","description":"Markdown 文件名，例如 story.md"},"content":{"type":"string","description":"完整 Markdown 内容"}}}`), &rawSchema); err != nil {
		return nil, err
	}
	return []einotool.BaseTool{&createTextAssetTool{
		info: &schema.ToolInfo{
			Name: "create_text_asset", Desc: "在当前 Session 创建一个可编辑、可版本化的 Markdown 资产，并加入创作 Flow。",
			ParamsOneOf: schema.NewParamsOneOfByJSONSchema(&rawSchema),
		},
		access: access,
	}}, nil
}

type createTextAssetTool struct {
	info   *schema.ToolInfo
	access ToolAccess
}

func (t *createTextAssetTool) Info(context.Context) (*schema.ToolInfo, error) { return t.info, nil }

func (t *createTextAssetTool) InvokableRun(ctx context.Context, arguments string, _ ...einotool.Option) (string, error) {
	var input struct {
		Name    string `json:"name"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal([]byte(arguments), &input); err != nil {
		return "", fmt.Errorf("studio: invalid create_text_asset arguments: %w", err)
	}
	input.Name = strings.TrimSpace(input.Name)
	input.Content = strings.TrimSpace(input.Content)
	if input.Name == "" || input.Content == "" {
		return "", fmt.Errorf("studio: create_text_asset requires name and content")
	}
	if !strings.HasSuffix(strings.ToLower(input.Name), ".md") {
		input.Name += ".md"
	}
	action := createTextAssetAction(input.Name, input.Content)
	if t.access.PermissionMode == domain.PermissionRequestApproval && (t.access.IsApproved == nil || !t.access.IsApproved(action)) {
		if t.access.RequestApproval == nil {
			return "", fmt.Errorf("studio: built-in tool approval handler is not configured")
		}
		if err := t.access.RequestApproval(ctx, action+"."+uuid.NewString(), action); err != nil {
			return "", err
		}
		return "", studioapp.ErrApprovalRequired
	}
	if err := t.access.Sink.Emit(ctx, studioapp.EventToolCallStart, map[string]any{"tool_call_id": action, "tool_name": t.info.Name, "argument_bytes": len(arguments)}); err != nil {
		return "", err
	}
	asset, err := t.access.Sink.CreateAsset(ctx, studioapp.GeneratedAsset{Name: input.Name, Kind: domain.AssetDocument, Origin: domain.AssetOriginAgent, MIMEType: "text/markdown", Content: []byte(input.Content)})
	if err != nil {
		return "", t.finish(ctx, action, err)
	}
	if asset == nil || len(asset.Versions) == 0 {
		return "", t.finish(ctx, action, fmt.Errorf("studio: create_text_asset returned no asset version"))
	}
	version := asset.Versions[len(asset.Versions)-1]
	if _, err := t.access.Sink.CreateFlowNode(ctx, studioapp.FlowNodeInput{Type: domain.FlowNodeAsset, Title: asset.Name, Body: "Agent 创建的 Markdown 文档", AssetID: asset.ID, AssetVersionID: version.ID, SortOrder: 500}); err != nil {
		return "", t.finish(ctx, action, err)
	}
	if err := t.access.Sink.Emit(ctx, studioapp.EventToolCallEnd, map[string]any{"tool_call_id": action, "tool_name": t.info.Name, "asset_id": asset.ID, "asset_version_id": version.ID, "is_error": false}); err != nil {
		return "", err
	}
	return fmt.Sprintf("已创建 Markdown 资产「%s」（v%d）。", asset.Name, version.Version), nil
}

func (t *createTextAssetTool) finish(ctx context.Context, action string, cause error) error {
	if err := t.access.Sink.Emit(ctx, studioapp.EventToolCallEnd, map[string]any{"tool_call_id": action, "tool_name": t.info.Name, "is_error": true}); err != nil {
		return err
	}
	return cause
}

func createTextAssetAction(name, content string) string {
	sum := sha256.Sum256([]byte(name + "\x00" + content))
	return fmt.Sprintf("asset.create_text.%x", sum)
}

var _ einotool.InvokableTool = (*createTextAssetTool)(nil)
