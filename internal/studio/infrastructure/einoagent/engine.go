package einoagent

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/adk"

	studioapp "github.com/Mr9esx/Pixoma/internal/studio/application"
	"github.com/Mr9esx/Pixoma/internal/studio/domain"
	"github.com/Mr9esx/Pixoma/internal/studio/infrastructure/modelprovider"
)

type ModelResolver interface {
	Resolve(ctx context.Context, accountID, configID string) (*domain.ResolvedModelConfig, error)
}

// Engine is the single-agent Eino ReAct entrypoint for configured online
// models. MockEngine remains independently injectable for deterministic local
// workflow verification; production dispatch chooses this engine only when a
// session selected an Agent-enabled model.
type Engine struct {
	Models ModelResolver
	Client *modelprovider.OpenAICompatibleClient
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
	instruction := "你是 Pixoma 创作 Studio 的单 Agent。以中文协助用户完成创作任务；清晰说明产出及下一步。"
	if len(request.Skills) > 0 {
		instruction += "\n\n本轮已选择以下 Skill。只在与其职责相关时遵循其中要求："
		for _, skill := range request.Skills {
			instruction += fmt.Sprintf("\n\n【%s】\n%s", skill.Name, skill.Prompt)
		}
	}
	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name: "pixoma_studio", Description: "Pixoma Studio 单 Agent",
		Instruction: instruction,
		Model:       modelprovider.NewEinoChatModel(e.Client, *config), MaxIterations: 8,
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
		if message == nil || strings.TrimSpace(message.Content) == "" {
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

var _ studioapp.AgentEngine = (*Engine)(nil)
