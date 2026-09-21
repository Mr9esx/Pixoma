package modelprovider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

// EinoChatModel adapts a configured OpenAI-compatible model to Eino's
// BaseChatModel contract. The agent layer remains provider-neutral while model
// credentials stay encrypted and resolved only in the server process.
type EinoChatModel struct {
	client *OpenAICompatibleClient
	config domain.ResolvedModelConfig
	tools  []ToolDefinition
}

func NewEinoChatModel(client *OpenAICompatibleClient, config domain.ResolvedModelConfig) *EinoChatModel {
	if client == nil {
		client = NewOpenAICompatibleClient(nil)
	}
	return &EinoChatModel{client: client, config: config}
}

func (m *EinoChatModel) Generate(ctx context.Context, input []*schema.Message, _ ...model.Option) (*schema.Message, error) {
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
		messages = append(messages, ChatMessage{Role: role, Content: message.Content, ToolCallID: message.ToolCallID, ToolCalls: toolCalls})
	}
	result, err := m.client.Chat(ctx, ChatRequest{Config: m.config, Messages: messages, Tools: m.tools})
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
	return &schema.Message{Role: schema.Assistant, Content: result.Text, ToolCalls: toolCalls, ResponseMeta: &schema.ResponseMeta{Usage: &schema.TokenUsage{PromptTokens: result.InputTokens, CompletionTokens: result.OutputTokens, TotalTokens: result.InputTokens + result.OutputTokens}}}, nil
}

func (m *EinoChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	message, err := m.Generate(ctx, input, opts...)
	if err != nil {
		return nil, err
	}
	return schema.StreamReaderFromArray([]*schema.Message{message}), nil
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
