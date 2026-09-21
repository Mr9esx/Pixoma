package einoagent

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/adk"
	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	studioapp "github.com/Mr9esx/Pixoma/internal/studio/application"
	"github.com/Mr9esx/Pixoma/internal/studio/domain"
	"github.com/Mr9esx/Pixoma/internal/studio/infrastructure/mcpconnector"
	"github.com/Mr9esx/Pixoma/internal/studio/infrastructure/modelprovider"
)

type ModelResolver interface {
	Resolve(ctx context.Context, accountID, configID string) (*domain.ResolvedModelConfig, error)
}

type CapabilityResolver interface {
	ResolveMCPConnectors(ctx context.Context, accountID string) ([]studioapp.ResolvedMCPConnector, error)
}

// Engine is the single-agent Eino ReAct entrypoint for configured online
// models. MockEngine remains independently injectable for deterministic local
// workflow verification; production dispatch chooses this engine only when a
// session selected an Agent-enabled model.
type Engine struct {
	Models       ModelResolver
	Capabilities CapabilityResolver
	Client       *modelprovider.OpenAICompatibleClient
}

func (e *Engine) Execute(ctx context.Context, request studioapp.AgentRequest, sink studioapp.AgentSink) error {
	if e == nil || e.Models == nil {
		return fmt.Errorf("studio: Eino model resolver is required")
	}
	if strings.TrimSpace(request.Run.ModelConfigID) == "" {
		return fmt.Errorf("studio: model config is required")
	}
	config, err := e.Models.Resolve(ctx, request.Run.AccountID, request.Run.ModelConfigID)
	if err != nil {
		return err
	}
	if err := sink.Emit(ctx, studioapp.EventRunStarted, map[string]any{"run_id": request.Run.ID, "session_id": request.Session.ID, "engine": "eino"}); err != nil {
		return err
	}
	tools, err := e.resolveTools(ctx, request, sink)
	if err != nil {
		return err
	}
	instruction := "你是 Pixoma 创作 Studio 的单 Agent。以中文协助用户完成创作任务；清晰说明产出及下一步。"
	if len(request.Skills) > 0 {
		instruction += "\n\n本轮已选择以下 Skill。只在与其职责相关时遵循其中要求："
		for _, skill := range request.Skills {
			instruction += fmt.Sprintf("\n\n【%s】\n%s", skill.Name, skill.Prompt)
		}
	}
	if len(request.Assets) > 0 {
		instruction += "\n\n本轮已选中的资产上下文："
		for _, asset := range request.Assets {
			instruction += fmt.Sprintf("\n- %s（类型：%s，版本：%d）", asset.Name, asset.Kind, asset.CurrentVersion)
		}
	}
	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name: "pixoma_studio", Description: "Pixoma Studio 单 Agent",
		Instruction: instruction,
		Model:       modelprovider.NewEinoChatModel(e.Client, *config), MaxIterations: 8,
		ToolsConfig: adk.ToolsConfig{ToolsNodeConfig: compose.ToolsNodeConfig{
			Tools: tools, ExecuteSequentially: true,
		}},
	})
	if err != nil {
		return fmt.Errorf("studio: create Eino agent: %w", err)
	}
	iterator := adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent}).Query(ctx, request.UserText)
	responded := false
	for {
		event, ok := iterator.Next()
		if !ok {
			break
		}
		if event == nil {
			continue
		}
		if event.Err != nil {
			return event.Err
		}
		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}
		message, err := event.Output.MessageOutput.GetMessage()
		if err != nil {
			return err
		}
		if message == nil || message.Role != schema.Assistant || strings.TrimSpace(message.Content) == "" {
			continue
		}
		if _, err := sink.AssistantMessage(ctx, message.Content); err != nil {
			return err
		}
		responded = true
	}
	if !responded {
		return fmt.Errorf("studio: model returned no assistant message")
	}
	return sink.Emit(ctx, studioapp.EventRunFinished, map[string]any{"run_id": request.Run.ID, "status": "succeeded"})
}

func (e *Engine) resolveTools(ctx context.Context, request studioapp.AgentRequest, sink studioapp.AgentSink) ([]einotool.BaseTool, error) {
	if e.Capabilities == nil {
		return nil, nil
	}
	connectors, err := e.Capabilities.ResolveMCPConnectors(ctx, request.Run.AccountID)
	if err != nil {
		return nil, fmt.Errorf("studio: resolve Agent MCP connectors: %w", err)
	}
	authorizer := newApprovalAuthorizer(request.Approvals)
	tools, err := mcpconnector.NewRuntimeTools(ctx, connectors, mcpconnector.ToolAccess{
		PermissionMode: request.Session.PermissionMode,
		IsApproved:     authorizer.Consume,
		RequestApproval: func(ctx context.Context, toolCallID, action string) error {
			_, err := sink.RequestApproval(ctx, toolCallID, action)
			return err
		},
		Emit: sink.Emit,
	})
	if err != nil {
		return nil, fmt.Errorf("studio: create Agent MCP tools: %w", err)
	}
	return tools, nil
}

type approvalAuthorizer struct {
	approved map[string]struct{}
}

func newApprovalAuthorizer(approvals []*domain.Approval) *approvalAuthorizer {
	authorizer := &approvalAuthorizer{approved: make(map[string]struct{})}
	for _, approval := range approvals {
		if approval != nil && approval.Status == domain.ApprovalApproved {
			authorizer.approved[approval.Action] = struct{}{}
		}
	}
	return authorizer
}

// Consume makes an approved tool intent single-use for the current Agent run.
// A resumed run receives persisted approvals again, but a fresh tool call with
// different normalized arguments gets a different action and needs approval.
func (a *approvalAuthorizer) Consume(action string) bool {
	if a == nil {
		return false
	}
	if _, ok := a.approved[action]; !ok {
		return false
	}
	delete(a.approved, action)
	return true
}

var _ studioapp.AgentEngine = (*Engine)(nil)
