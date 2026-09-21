package modelprovider

import (
	"context"
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
		messages = append(messages, ChatMessage{Role: role, Content: message.Content})
	}
	result, err := m.client.Chat(ctx, ChatRequest{Config: m.config, Messages: messages})
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, fmt.Errorf("model provider: empty result")
	}
	return &schema.Message{Role: schema.Assistant, Content: result.Text, ResponseMeta: &schema.ResponseMeta{Usage: &schema.TokenUsage{PromptTokens: result.InputTokens, CompletionTokens: result.OutputTokens, TotalTokens: result.InputTokens + result.OutputTokens}}}, nil
}

func (m *EinoChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	message, err := m.Generate(ctx, input, opts...)
	if err != nil {
		return nil, err
	}
	return schema.StreamReaderFromArray([]*schema.Message{message}), nil
}

var _ model.BaseChatModel = (*EinoChatModel)(nil)
