package modelprovider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

// EinoChatModel adapts a configured OpenAI-compatible model to Eino's
// BaseChatModel contract. The agent layer remains provider-neutral while model
// credentials stay encrypted and resolved only in the server process.
type EinoChatModel struct {
	client       *OpenAICompatibleClient
	config       domain.ResolvedModelConfig
	tools        []ToolDefinition
	traceFactory func() TraceSink
}

const responsesOutputExtraKey = "pixoma.responses_output"
const reasoningExtraKey = "pixoma.reasoning"

func NewEinoChatModel(client *OpenAICompatibleClient, config domain.ResolvedModelConfig) *EinoChatModel {
	return NewEinoChatModelWithTrace(client, config, nil)
}

// NewEinoChatModelWithTrace associates each provider attempt made by this
// request-scoped model with a fresh trace sink. WithTools preserves the
// factory, so ReAct tool loops remain separately observable attempts.
func NewEinoChatModelWithTrace(client *OpenAICompatibleClient, config domain.ResolvedModelConfig, traceFactory func() TraceSink) *EinoChatModel {
	if client == nil {
		client = NewOpenAICompatibleClient(nil)
	}
	return &EinoChatModel{client: client, config: config, traceFactory: traceFactory}
}

func (m *EinoChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	options := model.GetCommonOptions(nil, opts...)
	if options.Tools != nil {
		bound, err := m.WithTools(options.Tools)
		if err != nil {
			return nil, err
		}
		return bound.Generate(ctx, input)
	}
	messages := make([]ChatMessage, 0, len(input))
	for _, message := range input {
		if message == nil {
			continue
		}
		role := string(message.Role)
		if role == "" {
			role = "user"
		}
		toolCalls := make([]ToolCall, 0, len(message.ToolCalls))
		for _, call := range message.ToolCalls {
			toolCalls = append(toolCalls, ToolCall{
				ID: call.ID, Type: call.Type,
				Function: FunctionCall{Name: call.Function.Name, Arguments: call.Function.Arguments},
			})
		}
		messages = append(messages, ChatMessage{Role: role, Content: message.Content, ToolCallID: message.ToolCallID, ToolCalls: toolCalls, ResponsesOutput: responsesOutputFromExtra(message.Extra)})
	}
	result, err := m.client.Chat(ctx, ChatRequest{Config: m.config, Messages: messages, Tools: m.tools, Trace: m.nextTraceSink()})
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, fmt.Errorf("model provider: empty result")
	}
	toolCalls := make([]schema.ToolCall, 0, len(result.ToolCalls))
	for _, call := range result.ToolCalls {
		toolCalls = append(toolCalls, schema.ToolCall{
			ID: call.ID, Type: call.Type,
			Function: schema.FunctionCall{Name: call.Function.Name, Arguments: call.Function.Arguments},
		})
	}
	message := &schema.Message{Role: schema.Assistant, Content: result.Text, ToolCalls: toolCalls, ResponseMeta: &schema.ResponseMeta{Usage: &schema.TokenUsage{PromptTokens: result.InputTokens, CompletionTokens: result.OutputTokens, TotalTokens: result.InputTokens + result.OutputTokens}}}
	if len(result.ResponsesOutput) > 0 || strings.TrimSpace(result.Reasoning) != "" {
		message.Extra = map[string]any{}
		if len(result.ResponsesOutput) > 0 {
			message.Extra[responsesOutputExtraKey] = result.ResponsesOutput
		}
		if strings.TrimSpace(result.Reasoning) != "" {
			message.Extra[reasoningExtraKey] = result.Reasoning
		}
	}
	return message, nil
}

func responsesOutputFromExtra(extra map[string]any) []json.RawMessage {
	if extra == nil {
		return nil
	}
	output, _ := extra[responsesOutputExtraKey].([]json.RawMessage)
	return cloneResponsesOutput(output)
}

func (m *EinoChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	options := model.GetCommonOptions(nil, opts...)
	if len(m.tools) == 0 && options.Tools == nil && (m.config.Protocol == "" || m.config.Protocol == domain.ModelProtocolOpenAIChat) {
		messages := make([]ChatMessage, 0, len(input))
		for _, message := range input {
			if message == nil {
				continue
			}
			messages = append(messages, ChatMessage{Role: string(message.Role), Content: message.Content, ToolCallID: message.ToolCallID, ResponsesOutput: responsesOutputFromExtra(message.Extra)})
		}
		if stream, err := m.client.StreamChat(ctx, ChatRequest{Config: m.config, Messages: messages, Trace: m.nextTraceSink()}); err == nil {
			return stream, nil
		}
	}
	message, err := m.Generate(ctx, input, opts...)
	if err != nil {
		return nil, err
	}
	if message == nil || len(message.ToolCalls) > 0 || len([]rune(message.Content)) <= 24 {
		return schema.StreamReaderFromArray([]*schema.Message{message}), nil
	}
	runes := []rune(message.Content)
	chunks := make([]*schema.Message, 0, (len(runes)+23)/24)
	for start := 0; start < len(runes); start += 24 {
		end := start + 24
		if end > len(runes) {
			end = len(runes)
		}
		chunk := &schema.Message{Role: schema.Assistant, Content: string(runes[start:end])}
		if start == 0 {
			chunk.Extra = message.Extra
		}
		if end == len(runes) {
			chunk.ResponseMeta = message.ResponseMeta
		}
		chunks = append(chunks, chunk)
	}
	return schema.StreamReaderFromArray(chunks), nil
}

func (m *EinoChatModel) nextTraceSink() TraceSink {
	if m == nil || m.traceFactory == nil {
		return nil
	}
	return m.traceFactory()
}

var _ model.BaseChatModel = (*EinoChatModel)(nil)

// WithTools returns an immutable request-scoped model, as required by Eino's
// ReAct agent. Tool schemas are sent through the native OpenAI function-call
// contract instead of being interpolated into prompts.
func (m *EinoChatModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	definitions := make([]ToolDefinition, 0, len(tools))
	for _, tool := range tools {
		if tool == nil || tool.Name == "" {
			continue
		}
		encoded, err := json.Marshal(tool)
		if err != nil {
			return nil, fmt.Errorf("model provider: encode tool schema: %w", err)
		}
		var raw struct {
			JSONSchema map[string]any `json:"json_schema"`
		}
		if err := json.Unmarshal(encoded, &raw); err != nil {
			return nil, fmt.Errorf("model provider: decode tool schema: %w", err)
		}
		parameters := raw.JSONSchema
		if parameters == nil {
			parameters = map[string]any{"type": "object", "properties": map[string]any{}}
		}
		definitions = append(definitions, ToolDefinition{Name: tool.Name, Description: tool.Desc, Parameters: parameters})
	}
	clone := *m
	clone.tools = definitions
	return &clone, nil
}

var _ model.ToolCallingChatModel = (*EinoChatModel)(nil)
