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
	if lister, ok := access.Sink.(studioapp.SessionAssetLister); ok {
		var listSchema einojsonschema.Schema
		if err := json.Unmarshal([]byte(`{"type":"object","additionalProperties":false,"properties":{}}`), &listSchema); err != nil {
			return nil, err
		}
		tools = append(tools, &listSessionAssetsTool{info: &schema.ToolInfo{Name: "list_session_assets", Desc: "列出当前 Session 的资产名称、类型和版本 ID；读取内容前仍需在本轮对话中选择资产。", ParamsOneOf: schema.NewParamsOneOfByJSONSchema(&listSchema)}, access: access, lister: lister})
	}
	if access.Blob != nil && len(access.Assets) > 0 {
		tools = append(tools, &readAssetTool{info: &schema.ToolInfo{Name: "read_asset", Desc: "读取本次 Run 已选择且固定版本的文本资产内容。", ParamsOneOf: assetIDParams()}, access: access})
	}
	if _, ok := access.Sink.(studioapp.TextAssetVersionAppender); ok && hasPinnedMarkdownAsset(access.Assets) {
		tools = append(tools, &updateTextAssetTool{info: &schema.ToolInfo{Name: "update_text_asset", Desc: "将本次 Run 选择的 Markdown 资产更新为新版本，并加入创作 Flow。", ParamsOneOf: updateTextAssetParams()}, access: access})
	}
	return tools, nil
}

type listSessionAssetsTool struct {
	info   *schema.ToolInfo
	access ToolAccess
	lister studioapp.SessionAssetLister
}

func (t *listSessionAssetsTool) Info(context.Context) (*schema.ToolInfo, error) { return t.info, nil }

func (t *listSessionAssetsTool) InvokableRun(ctx context.Context, arguments string, _ ...einotool.Option) (string, error) {
	var input map[string]json.RawMessage
	if err := json.Unmarshal([]byte(arguments), &input); err != nil {
		return "", fmt.Errorf("studio: invalid list_session_assets arguments: %w", err)
	}
	if len(input) != 0 {
		return "", fmt.Errorf("studio: list_session_assets takes no arguments")
	}
	const limit = 100
	assets, err := t.lister.ListSessionAssets(ctx, limit+1)
	if err != nil {
		return "", err
	}
	type versionView struct {
		ID       string `json:"id"`
		Number   int    `json:"number"`
		MIMEType string `json:"mime_type"`
	}
	type assetView struct {
		ID             string             `json:"id"`
		Name           string             `json:"name"`
		Kind           domain.AssetKind   `json:"kind"`
		Origin         domain.AssetOrigin `json:"origin"`
		CurrentVersion int                `json:"current_version"`
		Versions       []versionView      `json:"versions"`
	}
	view := make([]assetView, 0, min(len(assets), limit))
	for _, asset := range assets {
		if asset == nil || len(view) == limit {
			break
		}
		versions := make([]versionView, 0, len(asset.Versions))
		for _, version := range asset.Versions {
			versions = append(versions, versionView{ID: version.ID, Number: version.Version, MIMEType: version.MIMEType})
		}
		view = append(view, assetView{ID: asset.ID, Name: asset.Name, Kind: asset.Kind, Origin: asset.Origin, CurrentVersion: asset.CurrentVersion, Versions: versions})
	}
	output, err := json.Marshal(struct {
		Assets    []assetView `json:"assets"`
		Truncated bool        `json:"truncated"`
	}{Assets: view, Truncated: len(assets) > limit})
	if err != nil {
		return "", err
	}
	toolCallID := "asset.list"
	if err := t.access.Sink.Emit(ctx, studioapp.EventToolCallStart, map[string]any{"tool_call_id": toolCallID, "tool_name": t.info.Name}); err != nil {
		return "", err
	}
	if err := t.access.Sink.Emit(ctx, studioapp.EventToolCallArgs, map[string]any{"tool_call_id": toolCallID, "delta": arguments}); err != nil {
		return "", err
	}
	if err := t.access.Sink.Emit(ctx, studioapp.EventToolCallResult, map[string]any{"tool_call_id": toolCallID, "content": string(output), "is_error": false}); err != nil {
		return "", err
	}
	if err := t.access.Sink.Emit(ctx, studioapp.EventToolCallEnd, map[string]any{"tool_call_id": toolCallID, "tool_name": t.info.Name, "asset_count": len(view), "is_error": false}); err != nil {
		return "", err
	}
	return string(output), nil
}

func assetIDParams() *schema.ParamsOneOf {
	var raw einojsonschema.Schema
	_ = json.Unmarshal([]byte(`{"type":"object","additionalProperties":false,"required":["asset_id"],"properties":{"asset_id":{"type":"string","description":"当前 Run 已选择资产的 ID"}}}`), &raw)
	return schema.NewParamsOneOfByJSONSchema(&raw)
}

func updateTextAssetParams() *schema.ParamsOneOf {
	var raw einojsonschema.Schema
	_ = json.Unmarshal([]byte(`{"type":"object","additionalProperties":false,"required":["asset_id","content"],"properties":{"asset_id":{"type":"string","description":"当前 Run 已选择的 Markdown 资产 ID"},"content":{"type":"string","description":"更新后的完整 Markdown 内容"}}}`), &raw)
	return schema.NewParamsOneOfByJSONSchema(&raw)
}

func hasPinnedMarkdownAsset(assets []*domain.Asset) bool {
	for _, asset := range assets {
		if asset != nil && asset.Kind == domain.AssetDocument && len(asset.Versions) == 1 && asset.Versions[0].MIMEType == "text/markdown" {
			return true
		}
	}
	return false
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

type updateTextAssetTool struct {
	info   *schema.ToolInfo
	access ToolAccess
}

func (t *updateTextAssetTool) Info(context.Context) (*schema.ToolInfo, error) { return t.info, nil }

func (t *updateTextAssetTool) InvokableRun(ctx context.Context, arguments string, _ ...einotool.Option) (string, error) {
	var input struct {
		AssetID string `json:"asset_id"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal([]byte(arguments), &input); err != nil {
		return "", fmt.Errorf("studio: invalid update_text_asset arguments: %w", err)
	}
	input.AssetID = strings.TrimSpace(input.AssetID)
	input.Content = strings.TrimSpace(input.Content)
	if input.AssetID == "" || input.Content == "" {
		return "", fmt.Errorf("studio: update_text_asset requires asset_id and content")
	}
	asset, version, ok := pinnedMarkdownAsset(t.access.Assets, input.AssetID)
	if !ok {
		return "", fmt.Errorf("studio: Markdown asset is not available in this run")
	}
	action := updateTextAssetAction(asset.ID, version.ID, input.Content)
	if t.access.PermissionMode == domain.PermissionRequestApproval && (t.access.IsApproved == nil || !t.access.IsApproved(action)) {
		if t.access.RequestApproval == nil {
			return "", fmt.Errorf("studio: built-in tool approval handler is not configured")
		}
		if err := t.access.RequestApproval(ctx, action+"."+uuid.NewString(), action, fmt.Sprintf("更新资产「%s」", asset.Name)); err != nil {
			return "", err
		}
		return "", compose.Interrupt(ctx, action)
	}
	updater, ok := t.access.Sink.(studioapp.TextAssetVersionAppender)
	if !ok {
		return "", fmt.Errorf("studio: text asset update is not configured")
	}
	if err := t.access.Sink.Emit(ctx, studioapp.EventToolCallStart, map[string]any{"tool_call_id": action, "tool_name": t.info.Name, "asset_id": asset.ID}); err != nil {
		return "", err
	}
	if err := t.access.Sink.Emit(ctx, studioapp.EventToolCallArgs, map[string]any{"tool_call_id": action, "delta": arguments}); err != nil {
		return "", err
	}
	updated, nextVersion, err := updater.AppendTextAssetVersion(ctx, asset.ID, version.ID, action, []byte(input.Content))
	if err != nil {
		return "", t.finish(ctx, action, err)
	}
	if updated == nil || nextVersion.ID == "" {
		return "", t.finish(ctx, action, fmt.Errorf("studio: update_text_asset returned no asset version"))
	}
	if _, err := t.access.Sink.CreateFlowNode(ctx, studioapp.FlowNodeInput{ActionID: action + ".flow", Type: domain.FlowNodeAsset, Title: updated.Name, Body: fmt.Sprintf("Agent 更新的 Markdown 文档（v%d）", nextVersion.Version), AssetID: updated.ID, AssetVersionID: nextVersion.ID, SortOrder: 500}); err != nil {
		return "", t.finish(ctx, action, err)
	}
	output := fmt.Sprintf("已更新 Markdown 资产「%s」（v%d）。", updated.Name, nextVersion.Version)
	if err := t.access.Sink.Emit(ctx, studioapp.EventToolCallResult, map[string]any{"tool_call_id": action, "content": output, "is_error": false}); err != nil {
		return "", err
	}
	if err := t.access.Sink.Emit(ctx, studioapp.EventToolCallEnd, map[string]any{"tool_call_id": action, "tool_name": t.info.Name, "asset_id": updated.ID, "asset_version_id": nextVersion.ID, "is_error": false}); err != nil {
		return "", err
	}
	return output, nil
}

func pinnedMarkdownAsset(assets []*domain.Asset, assetID string) (*domain.Asset, domain.AssetVersion, bool) {
	for _, asset := range assets {
		if asset == nil || asset.ID != assetID || asset.Kind != domain.AssetDocument || len(asset.Versions) != 1 {
			continue
		}
		version := asset.Versions[0]
		if version.MIMEType == "text/markdown" {
			return asset, version, true
		}
	}
	return nil, domain.AssetVersion{}, false
}

func (t *updateTextAssetTool) finish(ctx context.Context, action string, cause error) error {
	if err := t.access.Sink.Emit(ctx, studioapp.EventToolCallResult, map[string]any{"tool_call_id": action, "content": cause.Error(), "is_error": true}); err != nil {
		return err
	}
	if err := t.access.Sink.Emit(ctx, studioapp.EventToolCallEnd, map[string]any{"tool_call_id": action, "tool_name": t.info.Name, "is_error": true}); err != nil {
		return err
	}
	return cause
}

func updateTextAssetAction(assetID, versionID, content string) string {
	sum := sha256.Sum256([]byte(assetID + "\x00" + versionID + "\x00" + content))
	return fmt.Sprintf("asset.update_text.%x", sum)
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
