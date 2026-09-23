package studiotool

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	einojsonschema "github.com/eino-contrib/jsonschema"
	"github.com/google/uuid"

	"github.com/Mr9esx/Pixoma/internal/platform/blob"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
	studioapp "github.com/Mr9esx/Pixoma/internal/studio/application"
	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

const maxReadAssetBytes = 64 << 10

// ToolAccess binds built-in Studio capabilities to one audited Agent run.
type ToolAccess struct {
	PermissionMode  domain.PermissionMode
	IsApproved      func(action string) bool
	RequestApproval func(context.Context, string, string, string) error
	Sink            studioapp.AgentSink
	Blob            blob.Store
	Assets          []*domain.Asset
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
	tools := []einotool.BaseTool{&createTextAssetTool{
		info: &schema.ToolInfo{
			Name: "create_text_asset", Desc: "在当前 Session 创建一个可编辑、可版本化的 Markdown 资产，并加入创作 Flow。",
			ParamsOneOf: schema.NewParamsOneOfByJSONSchema(&rawSchema),
		},
		access: access,
	}}
	if access.Blob != nil && len(access.Assets) > 0 {
		tools = append(tools, &readAssetTool{info: &schema.ToolInfo{Name: "read_asset", Desc: "读取本次 Run 已选择且固定版本的文本资产内容。", ParamsOneOf: assetIDParams()}, access: access})
	}
	return tools, nil
}

func assetIDParams() *schema.ParamsOneOf {
	var raw einojsonschema.Schema
	_ = json.Unmarshal([]byte(`{"type":"object","additionalProperties":false,"required":["asset_id"],"properties":{"asset_id":{"type":"string","description":"当前 Run 已选择资产的 ID"}}}`), &raw)
	return schema.NewParamsOneOfByJSONSchema(&raw)
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
		if err := t.access.RequestApproval(ctx, action+"."+uuid.NewString(), action, fmt.Sprintf("创建资产「%s」", input.Name)); err != nil {
			return "", err
		}
		return "", compose.Interrupt(ctx, action)
	}
	if err := t.access.Sink.Emit(ctx, studioapp.EventToolCallStart, map[string]any{"tool_call_id": action, "tool_name": t.info.Name, "argument_bytes": len(arguments)}); err != nil {
		return "", err
	}
	if err := t.access.Sink.Emit(ctx, studioapp.EventToolCallArgs, map[string]any{"tool_call_id": action, "delta": arguments}); err != nil {
		return "", err
	}
	asset, err := t.access.Sink.CreateAsset(ctx, studioapp.GeneratedAsset{ActionID: action, Name: input.Name, Kind: domain.AssetDocument, Origin: domain.AssetOriginAgent, MIMEType: "text/markdown", Content: []byte(input.Content)})
	if err != nil {
		return "", t.finish(ctx, action, err)
	}
	if asset == nil || len(asset.Versions) == 0 {
		return "", t.finish(ctx, action, fmt.Errorf("studio: create_text_asset returned no asset version"))
	}
	version := asset.Versions[len(asset.Versions)-1]
	if _, err := t.access.Sink.CreateFlowNode(ctx, studioapp.FlowNodeInput{ActionID: action + ".flow", Type: domain.FlowNodeAsset, Title: asset.Name, Body: "Agent 创建的 Markdown 文档", AssetID: asset.ID, AssetVersionID: version.ID, SortOrder: 500}); err != nil {
		return "", t.finish(ctx, action, err)
	}
	output := fmt.Sprintf("已创建 Markdown 资产「%s」（v%d）。", asset.Name, version.Version)
	if err := t.access.Sink.Emit(ctx, studioapp.EventToolCallResult, map[string]any{"tool_call_id": action, "content": output, "is_error": false}); err != nil {
		return "", err
	}
	if err := t.access.Sink.Emit(ctx, studioapp.EventToolCallEnd, map[string]any{"tool_call_id": action, "tool_name": t.info.Name, "asset_id": asset.ID, "asset_version_id": version.ID, "is_error": false}); err != nil {
		return "", err
	}
	return output, nil
}

func (t *createTextAssetTool) finish(ctx context.Context, action string, cause error) error {
	if err := t.access.Sink.Emit(ctx, studioapp.EventToolCallResult, map[string]any{"tool_call_id": action, "content": cause.Error(), "is_error": true}); err != nil {
		return err
	}
	if err := t.access.Sink.Emit(ctx, studioapp.EventToolCallEnd, map[string]any{"tool_call_id": action, "tool_name": t.info.Name, "is_error": true}); err != nil {
		return err
	}
	return cause
}

func createTextAssetAction(name, content string) string {
	sum := sha256.Sum256([]byte(name + "\x00" + content))
	return fmt.Sprintf("asset.create_text.%x", sum)
}

type readAssetTool struct {
	info   *schema.ToolInfo
	access ToolAccess
}

func (t *readAssetTool) Info(context.Context) (*schema.ToolInfo, error) { return t.info, nil }

func (t *readAssetTool) InvokableRun(ctx context.Context, arguments string, _ ...einotool.Option) (string, error) {
	var input struct {
		AssetID string `json:"asset_id"`
	}
	if err := json.Unmarshal([]byte(arguments), &input); err != nil {
		return "", fmt.Errorf("studio: invalid read_asset arguments: %w", err)
	}
	input.AssetID = strings.TrimSpace(input.AssetID)
	for _, asset := range t.access.Assets {
		if asset == nil || asset.ID != input.AssetID || len(asset.Versions) != 1 {
			continue
		}
		version := asset.Versions[0]
		if !strings.HasPrefix(version.MIMEType, "text/") && version.MIMEType != "application/json" {
			return "", fmt.Errorf("studio: read_asset only supports text assets")
		}
		toolCallID := "asset.read." + asset.ID + "." + version.ID
		if err := t.access.Sink.Emit(ctx, studioapp.EventToolCallStart, map[string]any{"tool_call_id": toolCallID, "tool_name": t.info.Name}); err != nil {
			return "", err
		}
		if err := t.access.Sink.Emit(ctx, studioapp.EventToolCallArgs, map[string]any{"tool_call_id": toolCallID, "delta": arguments}); err != nil {
			return "", err
		}
		reader, err := t.access.Blob.Get(ctx, sharedkernel.BlobRef{Key: version.BlobKey, MIME: version.MIMEType, Size: version.SizeBytes})
		if err != nil {
			return "", t.finish(ctx, asset, version, err)
		}
		raw, readErr := io.ReadAll(io.LimitReader(reader, maxReadAssetBytes+1))
		_ = reader.Close()
		if readErr != nil {
			return "", t.finish(ctx, asset, version, readErr)
		}
		truncated := len(raw) > maxReadAssetBytes
		if truncated {
			raw = raw[:maxReadAssetBytes]
		}
		output := string(raw)
		if truncated {
			output += "\n\n[内容已截断]"
		}
		if err := t.access.Sink.Emit(ctx, studioapp.EventToolCallResult, map[string]any{"tool_call_id": toolCallID, "content": output, "is_error": false}); err != nil {
			return "", err
		}
		if err := t.access.Sink.Emit(ctx, studioapp.EventToolCallEnd, map[string]any{"tool_call_id": toolCallID, "tool_name": t.info.Name, "result_bytes": len(output), "is_error": false}); err != nil {
			return "", err
		}
		return output, nil
	}
	return "", fmt.Errorf("studio: asset is not available in this run")
}

func (t *readAssetTool) finish(ctx context.Context, asset *domain.Asset, version domain.AssetVersion, cause error) error {
	toolCallID := "asset.read." + asset.ID + "." + version.ID
	if err := t.access.Sink.Emit(ctx, studioapp.EventToolCallResult, map[string]any{"tool_call_id": toolCallID, "content": cause.Error(), "is_error": true}); err != nil {
		return err
	}
	if err := t.access.Sink.Emit(ctx, studioapp.EventToolCallEnd, map[string]any{"tool_call_id": toolCallID, "tool_name": t.info.Name, "is_error": true}); err != nil {
		return err
	}
	return cause
}

var _ einotool.InvokableTool = (*createTextAssetTool)(nil)
var _ einotool.InvokableTool = (*readAssetTool)(nil)
