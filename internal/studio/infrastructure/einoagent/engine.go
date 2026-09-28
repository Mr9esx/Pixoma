package einoagent

import (
	"context"
	"encoding/base64"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cloudwego/eino/adk"
	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"github.com/Mr9esx/Pixoma/internal/platform/blob"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
	studioapp "github.com/Mr9esx/Pixoma/internal/studio/application"
	"github.com/Mr9esx/Pixoma/internal/studio/application/contextcompaction"
	"github.com/Mr9esx/Pixoma/internal/studio/domain"
	"github.com/Mr9esx/Pixoma/internal/studio/infrastructure/mcpconnector"
	"github.com/Mr9esx/Pixoma/internal/studio/infrastructure/modelprovider"
	"github.com/Mr9esx/Pixoma/internal/studio/infrastructure/studiotool"
	"github.com/Mr9esx/Pixoma/internal/studio/infrastructure/workflowtool"
)

type ModelResolver interface {
	Resolve(ctx context.Context, accountID, configID string) (*domain.ResolvedModelConfig, error)
}

type CapabilityResolver interface {
	ResolveMCPConnectors(ctx context.Context, accountID string) ([]studioapp.ResolvedMCPConnector, error)
}

type WorkflowResolver interface {
	ResolveWorkflows(ctx context.Context, accountID string) ([]studioapp.ResolvedWorkflow, error)
}

// Engine is the single-agent Eino ReAct entrypoint for configured online
// models. MockEngine remains independently injectable for deterministic local
// workflow verification; production dispatch chooses this engine only when a
// session selected an Agent-enabled model.
type Engine struct {
	Models          ModelResolver
	Capabilities    CapabilityResolver
	SkillCreator    studioapp.SkillCreator
	Workflows       WorkflowResolver
	WorkflowStarter workflowtool.Starter
	Client          *modelprovider.OpenAICompatibleClient
	Blob            blob.Store
	Checkpoints     adk.CheckPointStore
}

const (
	maxModelImageBytes      = 8 << 20
	maxModelImageTotalBytes = 20 << 20
)

func init() {
	gob.Register([][]byte{})
	gob.Register([]json.RawMessage{})
}

func supportsModelImageMIME(mimeType string) bool {
	switch mimeType {
	case "image/jpeg", "image/png", "image/gif", "image/webp":
		return true
	default:
		return false
	}
}

func selectedAssetVersion(asset *domain.Asset) (domain.AssetVersion, error) {
	if asset == nil {
		return domain.AssetVersion{}, fmt.Errorf("studio: selected asset is missing")
	}
	for _, version := range asset.Versions {
		if version.Version == asset.CurrentVersion {
			return version, nil
		}
	}
	return domain.AssetVersion{}, fmt.Errorf("studio: selected asset %s has no current version", asset.ID)
}

func (e *Engine) selectedImageInputs(ctx context.Context, assets []*domain.Asset) ([]schema.MessageInputPart, error) {
	parts := make([]schema.MessageInputPart, 0)
	total := int64(0)
	for _, asset := range assets {
		if asset == nil || asset.Kind != domain.AssetImage {
			continue
		}
		version, err := selectedAssetVersion(asset)
		if err != nil {
			return nil, err
		}
		if !supportsModelImageMIME(version.MIMEType) {
			continue
		}
		if e.Blob == nil {
			return nil, fmt.Errorf("studio: asset storage is required for image input")
		}
		if version.SizeBytes > maxModelImageBytes || total+version.SizeBytes > maxModelImageTotalBytes {
			return nil, fmt.Errorf("studio: selected images exceed the model request size limit")
		}
		reader, err := e.Blob.Get(ctx, sharedkernel.BlobRef{Key: version.BlobKey, MIME: version.MIMEType, Size: version.SizeBytes})
		if err != nil {
			return nil, err
		}
		raw, readErr := io.ReadAll(io.LimitReader(reader, maxModelImageBytes+1))
		closeErr := reader.Close()
		if readErr != nil {
			return nil, readErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
		if len(raw) == 0 || len(raw) > maxModelImageBytes || total+int64(len(raw)) > maxModelImageTotalBytes {
			return nil, fmt.Errorf("studio: selected image %s has invalid size", asset.ID)
		}
		total += int64(len(raw))
		encoded := base64.StdEncoding.EncodeToString(raw)
		parts = append(parts, schema.MessageInputPart{Type: schema.ChatMessagePartTypeImageURL, Image: &schema.MessageInputImage{MessagePartCommon: schema.MessagePartCommon{Base64Data: &encoded, MIMEType: version.MIMEType}}})
	}
	return parts, nil
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
	if err := config.Limits.Validate(); err != nil {
		return fmt.Errorf("studio: Agent model limits are not configured: %w", err)
	}
	if err := sink.Emit(ctx, studioapp.EventRunStarted, map[string]any{"run_id": request.Run.ID, "session_id": request.Session.ID, "engine": "eino"}); err != nil {
		return err
	}
	var workflows []studioapp.ResolvedWorkflow
	if e.Workflows != nil {
		if e.WorkflowStarter == nil {
			return fmt.Errorf("studio: workflow starter is not configured")
		}
		workflows, err = e.Workflows.ResolveWorkflows(ctx, request.Run.AccountID)
		if err != nil {
			return fmt.Errorf("studio: resolve Agent workflows: %w", err)
		}
	}
	workflowWaiting := &atomic.Bool{}
	traceEmitter := newModelTraceEmitter(request, sink, *config)
	tools, err := e.resolveTools(ctx, request, sink, workflows, workflowWaiting, traceEmitter.anthropicOutput)
	if err != nil {
		return err
	}
	clarificationTool, err := newAskClarificationTool(sink, request.Clarifications, request.Run.Locale)
	if err != nil {
		return err
	}
	tools = append(tools, clarificationTool)
	var skillTool *loadSkillTool
	availableSkills := append([]domain.RunSkill(nil), request.AvailableSkills...)
	availableSkillIDs := make(map[string]struct{}, len(availableSkills)+len(request.Skills))
	for _, skill := range availableSkills {
		availableSkillIDs[skill.ID] = struct{}{}
	}
	for _, skill := range request.Skills {
		if _, exists := availableSkillIDs[skill.ID]; exists {
			continue
		}
		availableSkills = append(availableSkills, domain.RunSkill{
			ID: skill.ID, Name: skill.Name, Description: skill.Description, Prompt: skill.Prompt,
			Files: append([]domain.SkillFile(nil), skill.Files...),
		})
		availableSkillIDs[skill.ID] = struct{}{}
	}
	if len(availableSkills) > 0 {
		skillTool, err = newLoadSkillTool(availableSkills, sink, request.Run.Locale)
		if err != nil {
			return fmt.Errorf("studio: create Skill loader: %w", err)
		}
		tools = append(tools, skillTool)
	}
	if err := sortRegisteredTools(ctx, tools); err != nil {
		return err
	}
	instruction, err := buildStudioPrompt(ctx, request.Run.Locale, tools, len(availableSkills) > 0)
	if err != nil {
		return err
	}
	userText := request.UserText
	if len(request.UserParts) > 0 {
		userText, err = studioapp.ModelMessagePartsText(request.UserParts, append([]domain.RunSkill{}, availableSkills...), append([]*domain.Asset{}, request.Assets...), append([]studioapp.ResolvedWorkflow{}, workflows...), config.Capabilities.Vision, request.Run.Locale)
		if err != nil {
			return err
		}
	}
	runContext := buildStudioRunContext(request.Run.Locale, availableSkills, workflows)
	traceEmitter.runContext = runContext
	userText = runContext + "\n\n" + userText
	budget := budgetForConfig(*config, instruction, tools)
	if budget.ConversationTokens() <= 0 {
		return fmt.Errorf("studio: fixed instructions exceed the model context")
	}
	if skillTool != nil {
		skillTool.maxTokens = budget.ConversationTokens() - contextcompaction.EstimateTokens([]*schema.Message{schema.UserMessage(userText)}) - 256
	}
	compactor, err := newContextCompactor(ctx, request, config, instruction, tools, modelprovider.NewEinoChatModelWithTrace(e.Client, *config, traceEmitter.factory("context_summary")), sink)
	if err != nil {
		return err
	}
	var retryEventErr atomic.Value
	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name: "pixoma_studio", Description: "Pixoma Studio creative assistant",
		Instruction: instruction,
		Model:       modelprovider.NewEinoChatModelWithTrace(e.Client, *config, traceEmitter.factory("agent")).WithRestoredAnthropicOutput(request.Clarifications),
		ToolsConfig: adk.ToolsConfig{ToolsNodeConfig: compose.ToolsNodeConfig{
			Tools: tools, ExecuteSequentially: true,
		}},
		Middlewares: []adk.AgentMiddleware{compactor},
		ModelRetryConfig: &adk.ModelRetryConfig{
			MaxRetries: 1,
			ShouldRetry: func(retryCtx context.Context, retry *adk.RetryContext) *adk.RetryDecision {
				if retry == nil || retry.RetryAttempt > 1 || !isPromptTooLongError(retry.Err) {
					return nil
				}
				compacted, ok := compactForRetry(retryCtx, retry.InputMessages, budgetForConfig(*config, instruction, tools))
				if !ok {
					return nil
				}
				before := contextcompaction.EstimateTokens(retry.InputMessages)
				after := contextcompaction.EstimateTokens(compacted)
				if err := sink.Emit(retryCtx, studioapp.EventContextPruned, map[string]any{
					"source": "provider_retry", "detail": "模型请求超出上下文容量", "reason": "provider_rejected_context",
					"before_tokens_estimated": before, "after_tokens_estimated": after, "delta_tokens_estimated": after - before,
				}); err != nil {
					retryEventErr.Store(err)
					return nil
				}
				return &adk.RetryDecision{
					Retry: true, ModifiedInputMessages: compacted,
					PersistModifiedInputMessages: true,
					RejectReason:                 "provider rejected an oversized context",
				}
			},
		},
	})
	if err != nil {
		return fmt.Errorf("studio: create Eino agent: %w", err)
	}
	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent, EnableStreaming: true, CheckPointStore: e.Checkpoints})
	var iterator *adk.AsyncIterator[*adk.AgentEvent]
	if hasApprovedApproval(request.Approvals) || hasResolvedClarification(request.Clarifications) {
		if e.Checkpoints == nil {
			return fmt.Errorf("studio: interrupt checkpoint store is not configured")
		}
		iterator, err = runner.Resume(ctx, request.Run.ID)
		if err != nil {
			return fmt.Errorf("studio: resume interrupted run: %w", err)
		}
	} else {
		if err := sink.Emit(ctx, studioapp.EventContextInjected, map[string]any{
			"source": "run_context", "detail": "会话运行上下文",
			"delta_tokens_estimated": contextcompaction.EstimateTokens([]*schema.Message{schema.UserMessage(runContext)}),
		}); err != nil {
			return err
		}
		initialMessages := make([]*schema.Message, 0, len(request.History)+2)
		for i, historical := range request.History {
			if i >= len(request.HistoryMessageIDs) {
				return fmt.Errorf("studio: model history boundary is missing")
			}
			initialMessages = append(initialMessages, withBoundary(historical, request.HistoryMessageIDs[i]))
		}
		if strings.TrimSpace(request.ContextSummary) != "" {
			checkpoint := contextCheckpointMessage(request.ContextSummary, request.Run.Locale)
			initialMessages = append([]*schema.Message{checkpoint}, initialMessages...)
			if err := sink.Emit(ctx, studioapp.EventContextInjected, map[string]any{
				"source": "context_summary", "detail": "已有上下文摘要",
				"delta_tokens_estimated": contextcompaction.EstimateTokens([]*schema.Message{checkpoint}),
			}); err != nil {
				return err
			}
		}
		userMessage := withBoundary(schema.UserMessage(userText), request.Run.TriggerMessageID)
		if config.Capabilities.Vision {
			imageParts, err := e.selectedImageInputs(ctx, request.Assets)
			if err != nil {
				return err
			}
			userMessage.UserInputMultiContent = imageParts
		}
		initialMessages = append(initialMessages, userMessage)
		iterator = runner.Run(ctx, initialMessages, adk.WithCheckPointID(request.Run.ID))
	}
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
			if stored := retryEventErr.Load(); stored != nil {
				return fmt.Errorf("studio: persist context prune event: %w", stored.(error))
			}
			if errors.Is(event.Err, adk.ErrExceedMaxIterations) {
				return fmt.Errorf("studio: 本轮模型调用次数已达到上限，可在当前会话发送“继续”以使用已完成的工具结果: %w", event.Err)
			}
			return event.Err
		}
		if event.Action != nil && event.Action.Interrupted != nil {
			if e.Checkpoints == nil {
				return fmt.Errorf("studio: interrupt checkpoint store is not configured")
			}
			if _, exists, checkpointErr := e.Checkpoints.Get(ctx, request.Run.ID); checkpointErr != nil {
				return checkpointErr
			} else if !exists {
				return fmt.Errorf("studio: interrupt checkpoint was not persisted")
			}
			if clarificationTool.waiting.Load() || workflowWaiting.Load() {
				return studioapp.ErrClarificationRequired
			}
			return studioapp.ErrApprovalRequired
		}
		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}
		if event.Output.MessageOutput.IsStreaming && event.Output.MessageOutput.MessageStream != nil {
			if streamSink, ok := sink.(studioapp.AssistantStreamSink); ok {
				streamResponded, err := consumeAssistantStream(ctx, sink, streamSink, event.Output.MessageOutput.MessageStream)
				if err != nil {
					return err
				}
				responded = responded || streamResponded
				continue
			}
		}
		message, err := event.Output.MessageOutput.GetMessage()
		if err != nil {
			return err
		}
		if message == nil || message.Role != schema.Assistant || strings.TrimSpace(message.Content) == "" {
			continue
		}
		if reasoning, _ := message.Extra["pixoma.reasoning"].(string); strings.TrimSpace(reasoning) != "" {
			if err := emitReasoning(ctx, sink, reasoning); err != nil {
				return err
			}
		}
		if _, err := sink.AssistantMessage(ctx, message.Content); err != nil {
			return err
		}
		responded = true
	}
	if stored := retryEventErr.Load(); stored != nil {
		return fmt.Errorf("studio: persist context prune event: %w", stored.(error))
	}
	if !responded {
		return fmt.Errorf("studio: model returned no assistant message")
	}
	return sink.Emit(ctx, studioapp.EventRunFinished, map[string]any{"run_id": request.Run.ID, "status": "succeeded"})
}

func hasApprovedApproval(approvals []*domain.Approval) bool {
	for _, approval := range approvals {
		if approval != nil && approval.Status == domain.ApprovalApproved {
			return true
		}
	}
	return false
}

func hasResolvedClarification(clarifications []*domain.Clarification) bool {
	for _, clarification := range clarifications {
		if clarification != nil && (clarification.Status == domain.ClarificationAnswered || clarification.Status == domain.ClarificationSkipped) {
			return true
		}
	}
	return false
}

type modelTraceEmitter struct {
	runID               string
	sessionID           string
	turnID              string
	config              domain.ResolvedModelConfig
	runContext          string
	sink                studioapp.AgentSink
	next                atomic.Uint64
	mu                  sync.Mutex
	lastAnthropicOutput []json.RawMessage
}

func (e *modelTraceEmitter) anthropicOutput() []json.RawMessage {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]json.RawMessage(nil), e.lastAnthropicOutput...)
}

func newModelTraceEmitter(request studioapp.AgentRequest, sink studioapp.AgentSink, config domain.ResolvedModelConfig) *modelTraceEmitter {
	return &modelTraceEmitter{runID: request.Run.ID, sessionID: request.Session.ID, turnID: request.Run.TriggerMessageID, config: config, sink: sink}
}

func (e *modelTraceEmitter) factory(purpose string) func() modelprovider.TraceSink {
	return func() modelprovider.TraceSink {
		attemptID := fmt.Sprintf("%s:model:%d", e.runID, e.next.Add(1))
		return func(ctx context.Context, trace modelprovider.TraceEvent) error {
			if e == nil || e.sink == nil {
				return nil
			}
			eventType := modelTraceEventType(trace.Phase)
			if eventType == "" {
				return nil
			}
			if purpose == "agent" && e.config.Protocol == domain.ModelProtocolAnthropic && trace.Phase == modelprovider.TraceRequestFinished {
				var response struct {
					Content []json.RawMessage `json:"content"`
				}
				if err := json.Unmarshal(trace.ResponseBody, &response); err != nil {
					return err
				}
				e.mu.Lock()
				e.lastAnthropicOutput = response.Content
				e.mu.Unlock()
			}
			payload := map[string]any{
				"run_id": e.runID, "session_id": e.sessionID, "turn_id": e.turnID, "attempt_id": attemptID,
				"purpose": purpose, "protocol": e.config.Protocol, "model": e.config.Model,
				"context_window_tokens": e.config.Limits.ContextWindowTokens,
				"max_input_tokens":      e.config.Limits.MaxInputTokens,
				"max_output_tokens":     e.config.Limits.MaxOutputTokens,
				"at":                    trace.At.UTC().Format(time.RFC3339Nano), "elapsed_ms": trace.Elapsed.Milliseconds(),
				"status_code": trace.StatusCode, "provider_request_id": trace.ProviderRequestID,
				"input_tokens": trace.InputTokens, "output_tokens": trace.OutputTokens,
			}
			if len(trace.RequestBody) > 0 {
				payload["request_body"] = trace.RequestBody
				parts, err := modelprovider.AnalyzeRequest(trace.RequestBody, e.runContext)
				if err != nil {
					return err
				}
				for index := len(parts) - 1; index >= 0; index-- {
					if parts[index].Category != "user_message" {
						continue
					}
					preview := []rune(strings.TrimSpace(parts[index].Content))
					if len(preview) > 100 {
						preview = preview[:100]
					}
					payload["context_preview"] = string(preview)
					break
				}
				for index := range parts {
					parts[index].Content = ""
				}
				payload["context_parts"] = parts
				payload["run_context"] = e.runContext
			}
			if len(trace.ResponseBody) > 0 {
				payload["response_body"] = trace.ResponseBody
			}
			if trace.Phase == modelprovider.TraceRequestFinished && !trace.UsageReported {
				delete(payload, "input_tokens")
				delete(payload, "output_tokens")
			}
			if trace.CacheReadTokens != nil {
				payload["cache_read_tokens"] = *trace.CacheReadTokens
			}
			if trace.CacheWriteTokens != nil {
				payload["cache_write_tokens"] = *trace.CacheWriteTokens
			}
			if trace.ReasoningTokens != nil {
				payload["reasoning_tokens"] = *trace.ReasoningTokens
			}
			if strings.TrimSpace(trace.Error) != "" {
				payload["error"] = trace.Error
			}
			return e.sink.Emit(ctx, eventType, payload)
		}
	}
}

func modelTraceEventType(phase modelprovider.TracePhase) string {
	switch phase {
	case modelprovider.TraceRequestStarted:
		return studioapp.EventModelRequestStarted
	case modelprovider.TraceFirstToken:
		return studioapp.EventModelFirstToken
	case modelprovider.TraceRequestFinished:
		return studioapp.EventModelRequestFinished
	case modelprovider.TraceRequestFailed:
		return studioapp.EventModelRequestFailed
	default:
		return ""
	}
}

func budgetForConfig(config domain.ResolvedModelConfig, instruction string, tools []einotool.BaseTool) contextcompaction.Budget {
	reserved := contextcompaction.EstimateTokens([]*schema.Message{schema.SystemMessage(instruction)})
	for _, tool := range tools {
		if tool == nil {
			continue
		}
		info, err := tool.Info(context.Background())
		if err != nil {
			continue
		}
		encoded, err := json.Marshal(info)
		if err != nil {
			continue
		}
		reserved += contextcompaction.EstimateTokens([]*schema.Message{{Role: schema.System, Content: string(encoded)}})
	}
	return contextcompaction.Budget{ContextWindowTokens: config.Limits.ContextWindowTokens, MaxInputTokens: config.Limits.MaxInputTokens, MaxOutputTokens: config.Limits.MaxOutputTokens, ReservedTokens: reserved}
}

func compactForRetry(ctx context.Context, input []*schema.Message, budget contextcompaction.Budget) ([]*schema.Message, bool) {
	systems := make([]*schema.Message, 0, 1)
	checkpoints := make([]*schema.Message, 0, 1)
	conversation := make([]*schema.Message, 0, len(input))
	for _, message := range input {
		if message == nil {
			continue
		}
		if message.Role == schema.System {
			systems = append(systems, message)
		} else if isContextCheckpoint(message) {
			checkpoints = append(checkpoints, message)
		} else {
			conversation = append(conversation, message)
		}
	}
	budget.ReservedTokens += contextcompaction.EstimateTokens(checkpoints)
	result, err := contextcompaction.Manage(ctx, conversation, contextcompaction.Options{
		Budget: budget, RecentRounds: 1, Force: true, ClearToolResult: compactableToolResult,
	})
	if err != nil || !result.Compressed {
		return nil, false
	}
	compacted := append([]*schema.Message(nil), systems...)
	compacted = append(compacted, checkpoints...)
	compacted = append(compacted, result.Messages...)
	if len(compacted) >= len(input) && contextcompaction.EstimateTokens(compacted) >= contextcompaction.EstimateTokens(input) {
		return nil, false
	}
	return compacted, true
}

func isPromptTooLongError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	for _, marker := range []string{"context_length_exceeded", "context length", "prompt_too_long", "prompt too long", "too many tokens", "maximum context"} {
		if strings.Contains(message, marker) {
			return true
		}
	}
	return false
}

func newContextCompactor(ctx context.Context, request studioapp.AgentRequest, config *domain.ResolvedModelConfig, instruction string, tools []einotool.BaseTool, summaryModel *modelprovider.EinoChatModel, sink studioapp.AgentSink) (adk.AgentMiddleware, error) {
	reserved := contextcompaction.EstimateTokens([]*schema.Message{schema.SystemMessage(instruction)})
	for _, tool := range tools {
		if tool == nil {
			continue
		}
		info, err := tool.Info(ctx)
		if err != nil {
			return adk.AgentMiddleware{}, fmt.Errorf("studio: inspect tool schema for context budget: %w", err)
		}
		encoded, err := json.Marshal(info)
		if err != nil {
			return adk.AgentMiddleware{}, fmt.Errorf("studio: encode tool schema for context budget: %w", err)
		}
		reserved += contextcompaction.EstimateTokens([]*schema.Message{{Role: schema.System, Content: string(encoded)}})
	}
	budget := contextcompaction.Budget{
		ContextWindowTokens: config.Limits.ContextWindowTokens,
		MaxInputTokens:      config.Limits.MaxInputTokens,
		MaxOutputTokens:     config.Limits.MaxOutputTokens,
		ReservedTokens:      reserved,
	}
	liveSummary := strings.TrimSpace(request.ContextSummary)
	return adk.AgentMiddleware{BeforeChatModel: func(compactCtx context.Context, state *adk.ChatModelAgentState) error {
		if state == nil || len(state.Messages) == 0 {
			return nil
		}
		systems := make([]*schema.Message, 0, 1)
		conversation := make([]*schema.Message, 0, len(state.Messages))
		for _, message := range state.Messages {
			if message == nil {
				continue
			}
			if message.Role == schema.System {
				systems = append(systems, message)
				continue
			}
			if isContextCheckpoint(message) {
				continue
			}
			conversation = append(conversation, message)
		}
		options := contextcompaction.Options{
			Budget:          budget,
			RecentRounds:    3,
			TriggerRatio:    0.8,
			ClearToolResult: compactableToolResult,
		}
		if liveSummary != "" {
			options.Budget.ReservedTokens += contextcompaction.EstimateTokens([]*schema.Message{contextCheckpointMessage(liveSummary, request.Run.Locale)})
		}
		options.Summarize = func(summaryCtx context.Context, messages []*schema.Message) (string, error) {
			return summarizeMessages(summaryCtx, summaryModel, messages, liveSummary, request.Run.Locale)
		}
		result, err := contextcompaction.Manage(compactCtx, conversation, options)
		if err != nil {
			return err
		}
		if !result.Compressed {
			return nil
		}
		if result.HardTruncated {
			return fmt.Errorf("studio: current conversation exceeds the model context")
		}
		if result.AutoCompactApplied {
			if result.RetainedFrom <= 0 || result.RetainedFrom > len(conversation) {
				return fmt.Errorf("studio: context summary boundary is outside model history")
			}
			boundaryMessageID := messageBoundary(conversation[result.RetainedFrom-1], request.Run.TriggerMessageID)
			if request.SaveContextSummary != nil {
				if err := request.SaveContextSummary(compactCtx, boundaryMessageID, result.Summary); err != nil {
					return fmt.Errorf("studio: persist context summary: %w", err)
				}
			}
			liveSummary = result.Summary
		}
		beforeTokens := contextcompaction.EstimateTokens(state.Messages)
		conversationBefore := contextcompaction.EstimateTokens(conversation)
		rebuilt := append([]*schema.Message(nil), systems...)
		if liveSummary != "" {
			rebuilt = append(rebuilt, contextCheckpointMessage(liveSummary, request.Run.Locale))
		}
		rebuilt = append(rebuilt, result.Messages...)
		state.Messages = rebuilt
		if sink != nil {
			if result.MicroCompactApplied {
				if err := sink.Emit(compactCtx, studioapp.EventContextPruned, map[string]any{
					"source": "tool_results", "detail": "清理工具结果", "reason": result.ActiveLayer,
					"before_tokens_estimated": conversationBefore,
					"after_tokens_estimated":  result.MicroCompactTokens,
					"delta_tokens_estimated":  result.MicroCompactTokens - conversationBefore,
				}); err != nil {
					return err
				}
			}
			if result.AutoCompactApplied || result.ActiveLayer == "sliding_window" {
				eventType := studioapp.EventContextCompacted
				if result.ActiveLayer == "sliding_window" {
					eventType = studioapp.EventContextPruned
				}
				afterTokens := contextcompaction.EstimateTokens(rebuilt)
				adjustedBefore := beforeTokens
				if result.MicroCompactApplied {
					adjustedBefore += result.MicroCompactTokens - conversationBefore
				}
				if err := sink.Emit(compactCtx, eventType, map[string]any{
					"source": result.ActiveLayer, "detail": "上下文压缩", "reason": result.ActiveLayer,
					"before_tokens_estimated": adjustedBefore, "after_tokens_estimated": afterTokens,
					"delta_tokens_estimated": afterTokens - adjustedBefore,
					"retained_from":          result.RetainedFrom, "auto_compact": result.AutoCompactApplied,
					"summary": result.Summary,
				}); err != nil {
					return err
				}
			}
		}
		return nil
	}}, nil
}

func compactableToolResult(message *schema.Message) bool {
	if message == nil || message.Role != schema.Tool {
		return false
	}
	name := strings.ToLower(strings.TrimSpace(message.Name))
	if name == "read_asset" || name == "load_skill" {
		return true
	}
	for _, marker := range []string{"read", "search", "fetch", "list", "get", "query", "inspect"} {
		if strings.Contains(name, marker) {
			return true
		}
	}
	return false
}

func consumeAssistantStream(ctx context.Context, sink studioapp.AgentSink, streamSink studioapp.AssistantStreamSink, stream *schema.StreamReader[*schema.Message]) (bool, error) {
	defer stream.Close()
	first, err := stream.Recv()
	if err == io.EOF {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if first == nil {
		return false, nil
	}
	var reasoningMessageID string
	beginReasoning := func() error {
		if reasoningMessageID != "" {
			return nil
		}
		reasoningMessageID = fmt.Sprintf("reasoning-%d", time.Now().UnixNano())
		if err := sink.Emit(ctx, studioapp.EventReasoningStart, map[string]any{}); err != nil {
			return err
		}
		return sink.Emit(ctx, studioapp.EventReasoningMessageStart, map[string]any{"message_id": reasoningMessageID})
	}
	appendReasoning := func(value string) error {
		if strings.TrimSpace(value) == "" {
			return nil
		}
		if err := beginReasoning(); err != nil {
			return err
		}
		return sink.Emit(ctx, studioapp.EventReasoningMessageContent, map[string]any{"message_id": reasoningMessageID, "delta": value})
	}
	finishReasoning := func() error {
		if reasoningMessageID == "" {
			return nil
		}
		messageID := reasoningMessageID
		reasoningMessageID = ""
		if err := sink.Emit(ctx, studioapp.EventReasoningMessageEnd, map[string]any{"message_id": messageID}); err != nil {
			return err
		}
		return sink.Emit(ctx, studioapp.EventReasoningEnd, map[string]any{})
	}
	if value, _ := first.Extra["pixoma.reasoning"].(string); strings.TrimSpace(value) != "" {
		if err := appendReasoning(value); err != nil {
			return false, err
		}
	}
	var messageID string
	var full strings.Builder
	appendContent := func(content string) error {
		if content == "" {
			return nil
		}
		if err := finishReasoning(); err != nil {
			return err
		}
		if messageID == "" {
			var err error
			messageID, err = streamSink.BeginAssistantMessage(ctx)
			if err != nil {
				return err
			}
		}
		full.WriteString(content)
		return streamSink.AppendAssistantMessage(ctx, messageID, content)
	}
	if err := appendContent(first.Content); err != nil {
		return false, err
	}
	for {
		chunk, recvErr := stream.Recv()
		if recvErr == io.EOF {
			break
		}
		if recvErr != nil {
			return false, recvErr
		}
		if chunk == nil {
			continue
		}
		if value, _ := chunk.Extra["pixoma.reasoning"].(string); strings.TrimSpace(value) != "" {
			if err := appendReasoning(value); err != nil {
				return false, err
			}
		}
		if err := appendContent(chunk.Content); err != nil {
			return false, err
		}
	}
	if err := finishReasoning(); err != nil {
		return false, err
	}
	if messageID == "" {
		return false, nil
	}
	_, err = streamSink.EndAssistantMessage(ctx, messageID, full.String())
	return true, err
}

func emitReasoning(ctx context.Context, sink studioapp.AgentSink, reasoning string) error {
	messageID := fmt.Sprintf("reasoning-%d", time.Now().UnixNano())
	if err := sink.Emit(ctx, studioapp.EventReasoningStart, map[string]any{}); err != nil {
		return err
	}
	if err := sink.Emit(ctx, studioapp.EventReasoningMessageStart, map[string]any{"message_id": messageID}); err != nil {
		return err
	}
	if err := sink.Emit(ctx, studioapp.EventReasoningMessageContent, map[string]any{"message_id": messageID, "delta": reasoning}); err != nil {
		return err
	}
	if err := sink.Emit(ctx, studioapp.EventReasoningMessageEnd, map[string]any{"message_id": messageID}); err != nil {
		return err
	}
	return sink.Emit(ctx, studioapp.EventReasoningEnd, map[string]any{})
}

func (e *Engine) resolveTools(ctx context.Context, request studioapp.AgentRequest, sink studioapp.AgentSink, workflows []studioapp.ResolvedWorkflow, workflowWaiting *atomic.Bool, anthropicOutput func() []json.RawMessage) ([]einotool.BaseTool, error) {
	authorizer := newApprovalAuthorizer(request.Approvals)
	requestApproval := func(ctx context.Context, toolCallID, action, description string) error {
		_, err := sink.RequestApproval(ctx, toolCallID, action, description)
		return err
	}
	tools := make([]einotool.BaseTool, 0)
	builtInTools, err := studiotool.NewRuntimeTools(studiotool.ToolAccess{
		Locale:         request.Run.Locale,
		PermissionMode: request.Session.PermissionMode, IsApproved: authorizer.Consume,
		RequestApproval: requestApproval, Sink: sink, Blob: e.Blob, Assets: request.Assets,
	})
	if err != nil {
		return nil, fmt.Errorf("studio: create built-in Agent tools: %w", err)
	}
	tools = append(tools, builtInTools...)
	if e.SkillCreator != nil {
		installSkillTool, err := newInstallSkillTool(request.Run.AccountID, e.SkillCreator, sink, request.Run.Locale)
		if err != nil {
			return nil, fmt.Errorf("studio: create Skill installer: %w", err)
		}
		tools = append(tools, installSkillTool)
	}
	if e.Capabilities != nil {
		connectors, err := e.Capabilities.ResolveMCPConnectors(ctx, request.Run.AccountID)
		if err != nil {
			return nil, fmt.Errorf("studio: resolve Agent MCP connectors: %w", err)
		}
		mcpTools, err := mcpconnector.NewRuntimeTools(ctx, connectors, mcpconnector.ToolAccess{
			PermissionMode: request.Session.PermissionMode, IsApproved: authorizer.Consume,
			RequestApproval: requestApproval, Emit: sink.Emit,
		})
		if err != nil {
			return nil, fmt.Errorf("studio: create Agent MCP tools: %w", err)
		}
		tools = append(tools, mcpTools...)
	}
	if e.Workflows != nil {
		workflowTools, err := workflowtool.NewRuntimeTools(workflows, workflowtool.ToolAccess{
			AccountID: request.Run.AccountID, SessionID: request.Session.ID, RunID: request.Run.ID,
			Sink: sink, Starter: e.WorkflowStarter,
			Clarifications: request.Clarifications, Waiting: workflowWaiting,
			AnthropicOutput: anthropicOutput,
		})
		if err != nil {
			return nil, fmt.Errorf("studio: create Agent workflow tools: %w", err)
		}
		tools = append(tools, workflowTools...)
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
