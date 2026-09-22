package application

import (
	"context"
	"fmt"
	"html"

	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

type MockEngine struct{}

func NewMockEngine() *MockEngine { return &MockEngine{} }

func (e *MockEngine) Execute(ctx context.Context, request AgentRequest, sink AgentSink) error {
	if err := sink.Emit(ctx, EventRunStarted, map[string]any{
		"run_id": request.Run.ID, "session_id": request.Session.ID,
	}); err != nil {
		return err
	}
	if request.Session.PermissionMode == "request_approval" && !hasApprovedAction(request.Approvals, "workflow.execute") {
		if _, err := sink.RequestApproval(ctx, "mock-storyboard", "workflow.execute"); err != nil {
			return err
		}
		return ErrApprovalRequired
	}
	if _, err := sink.AssistantMessage(ctx, "我先整理故事大纲，再调用分镜工作流生成预览图。你可以在右侧 Flow 中继续调整这条创作路线。"); err != nil {
		return err
	}
	stage, err := sink.CreateFlowNode(ctx, FlowNodeInput{Type: "stage", Title: "立住故事", Body: "先明确主角、冲突与结尾。", SortOrder: 10})
	if err != nil {
		return err
	}
	for index, asset := range request.Assets {
		inputVersionID := ""
		if len(asset.Versions) > 0 {
			inputVersionID = asset.Versions[len(asset.Versions)-1].ID
		}
		inputNode, err := sink.CreateFlowNode(ctx, FlowNodeInput{Type: "asset", Title: "输入：" + asset.Name, Body: "本轮对话选中的 Session 资产", AssetID: asset.ID, AssetVersionID: inputVersionID, SortOrder: 15 + index})
		if err != nil {
			return err
		}
		if _, err := sink.CreateFlowEdge(ctx, inputNode.ID, stage.ID, "作为上下文"); err != nil {
			return err
		}
	}
	outline := fmt.Sprintf("# 雨夜侦探\n\n## 创作目标\n\n%s\n\n## 故事大纲\n\n侦探在雨夜收到一封没有署名的委托信，循着线索进入旧车站，并在最后一班列车到站前找到真相。\n", request.UserText)
	outlineAsset, err := sink.CreateAsset(ctx, GeneratedAsset{
		Name: "故事大纲.md", Kind: "document", Origin: "agent", MIMEType: "text/markdown",
		Content: []byte(outline), Metadata: map[string]any{"mock": true, "role": "story_outline"},
	})
	if err != nil {
		return err
	}
	outlineNode, err := sink.CreateFlowNode(ctx, FlowNodeInput{Type: "asset", Title: "故事大纲", Body: "Agent 生成的 Markdown 故事大纲", AssetID: outlineAsset.ID, AssetVersionID: outlineAsset.Versions[0].ID, SortOrder: 20})
	if err != nil {
		return err
	}
	if _, err := sink.CreateFlowEdge(ctx, stage.ID, outlineNode.ID, "产出"); err != nil {
		return err
	}
	if err := sink.Emit(ctx, EventToolCallStart, map[string]any{
		"tool_call_id": "mock-storyboard", "tool_name": "分镜生成", "input_asset_ids": []string{outlineAsset.ID},
	}); err != nil {
		return err
	}
	if err := sink.Emit(ctx, EventToolCallArgs, map[string]any{
		"tool_call_id": "mock-storyboard", "delta": fmt.Sprintf(`{"input_asset_ids":[%q]}`, outlineAsset.ID),
	}); err != nil {
		return err
	}
	operation, err := sink.CreateFlowNode(ctx, FlowNodeInput{Type: "operation", Title: "分镜工作流", Body: "使用故事大纲生成分镜预览", SortOrder: 30})
	if err != nil {
		return err
	}
	if _, err := sink.CreateFlowEdge(ctx, outlineNode.ID, operation.ID, "作为输入"); err != nil {
		return err
	}
	svg := mockStoryboardSVG(request.UserText)
	imageAsset, err := sink.CreateAsset(ctx, GeneratedAsset{
		Name: "雨夜侦探-分镜预览.svg", Kind: "image", Origin: "workflow", MIMEType: "image/svg+xml",
		Content: []byte(svg), Metadata: map[string]any{"mock": true, "workflow": "storyboard"},
	})
	if err != nil {
		return err
	}
	imageNode, err := sink.CreateFlowNode(ctx, FlowNodeInput{Type: "asset", Title: "分镜预览", Body: "Mock 分镜工作流输出", AssetID: imageAsset.ID, AssetVersionID: imageAsset.Versions[0].ID, SortOrder: 40})
	if err != nil {
		return err
	}
	if _, err := sink.CreateFlowEdge(ctx, operation.ID, imageNode.ID, "输出"); err != nil {
		return err
	}
	if err := sink.Emit(ctx, EventToolCallResult, map[string]any{
		"tool_call_id": "mock-storyboard", "content": fmt.Sprintf("已生成分镜预览资产：%s", imageAsset.ID), "is_error": false,
	}); err != nil {
		return err
	}
	if err := sink.Emit(ctx, EventToolCallEnd, map[string]any{
		"tool_call_id": "mock-storyboard", "tool_name": "分镜生成", "output_asset_ids": []string{imageAsset.ID},
	}); err != nil {
		return err
	}
	return sink.Emit(ctx, EventRunFinished, map[string]any{"run_id": request.Run.ID, "status": "succeeded"})
}

func hasApprovedAction(approvals []*domain.Approval, action string) bool {
	for _, approval := range approvals {
		if approval.Action == action && approval.Status == domain.ApprovalApproved {
			return true
		}
	}
	return false
}

func mockStoryboardSVG(prompt string) string {
	safePrompt := html.EscapeString(prompt)
	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="1200" height="720" viewBox="0 0 1200 720" role="img" aria-label="雨夜侦探漫画分镜预览">
  <rect width="1200" height="720" fill="#111827"/>
  <g fill="none" stroke="#e5e7eb" stroke-width="6">
    <rect x="36" y="36" width="548" height="300" rx="12"/>
    <rect x="616" y="36" width="548" height="300" rx="12"/>
    <rect x="36" y="384" width="548" height="300" rx="12"/>
    <rect x="616" y="384" width="548" height="300" rx="12"/>
  </g>
  <g fill="#f8fafc" font-family="sans-serif" font-size="28">
    <text x="68" y="92">01 雨夜来信</text><text x="648" y="92">02 旧站追踪</text>
    <text x="68" y="440">03 最后一班车</text><text x="648" y="440">04 真相揭晓</text>
  </g>
  <g fill="#38bdf8" opacity=".75"><circle cx="270" cy="210" r="64"/><circle cx="850" cy="210" r="64"/><circle cx="270" cy="555" r="64"/><circle cx="850" cy="555" r="64"/></g>
  <text x="600" y="708" text-anchor="middle" fill="#94a3b8" font-family="sans-serif" font-size="16">%s</text>
</svg>`, safePrompt)
}

var _ AgentEngine = (*MockEngine)(nil)
