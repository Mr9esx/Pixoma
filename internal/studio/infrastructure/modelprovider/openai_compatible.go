package modelprovider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

const maxProviderResponseBytes = 4 << 20

type ChatMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
}

type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type ToolDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters"`
}

type ChatRequest struct {
	Config   domain.ResolvedModelConfig
	Messages []ChatMessage
	Tools    []ToolDefinition
}

type ChatResult struct {
	Text         string
	InputTokens  int
	OutputTokens int
	ToolCalls    []ToolCall
}

type OpenAICompatibleClient struct {
	client *http.Client
}

// ConnectionTester adapts the shared protocol client for model configuration
// health checks. A normal, tiny chat request exercises endpoint, auth and
// model compatibility without persisting its output.
type ConnectionTester struct {
	Client *OpenAICompatibleClient
}

func (t ConnectionTester) Test(ctx context.Context, config domain.ResolvedModelConfig) error {
	client := t.Client
	if client == nil {
		client = NewOpenAICompatibleClient(nil)
	}
	result, err := client.Chat(ctx, ChatRequest{
		Config: config,
		Messages: []ChatMessage{{
			Role:    "user",
			Content: "Reply with OK.",
		}},
	})
	if err != nil {
		return err
	}
	if strings.TrimSpace(result.Text) == "" {
		return fmt.Errorf("model provider: connection test returned no text")
	}
	return nil
}

func NewOpenAICompatibleClient(client *http.Client) *OpenAICompatibleClient {
	if client == nil {
		client = &http.Client{Timeout: 90 * time.Second}
	}
	return &OpenAICompatibleClient{client: client}
}

func (c *OpenAICompatibleClient) Chat(ctx context.Context, input ChatRequest) (*ChatResult, error) {
	if strings.TrimSpace(input.Config.BaseURL) == "" || strings.TrimSpace(input.Config.Model) == "" || strings.TrimSpace(input.Config.APIKey) == "" {
		return nil, fmt.Errorf("model provider: incomplete model config")
	}
	switch input.Config.Protocol {
	case domain.ModelProtocolOpenAIResponses:
		if len(input.Tools) > 0 {
			return nil, fmt.Errorf("model provider: tool calling is not implemented for OpenAI Responses")
		}
		return c.openAIResponses(ctx, input)
	case domain.ModelProtocolAnthropic:
		if len(input.Tools) > 0 {
			return nil, fmt.Errorf("model provider: tool calling is not implemented for Anthropic Messages")
		}
		return c.anthropicMessages(ctx, input)
	case "", domain.ModelProtocolOpenAIChat:
		return c.openAIChat(ctx, input)
	default:
		return nil, fmt.Errorf("model provider: unsupported protocol %q", input.Config.Protocol)
	}
}

func (c *OpenAICompatibleClient) openAIChat(ctx context.Context, input ChatRequest) (*ChatResult, error) {
	body := map[string]any{"model": input.Config.Model, "messages": input.Messages, "stream": false}
	if len(input.Tools) > 0 {
		tools := make([]map[string]any, 0, len(input.Tools))
		for _, definition := range input.Tools {
			tools = append(tools, map[string]any{"type": "function", "function": definition})
		}
		body["tools"] = tools
		body["tool_choice"] = "auto"
	}
	if input.Config.Thinking.Enabled && input.Config.Thinking.Effort != "" {
		body["reasoning_effort"] = input.Config.Thinking.Effort
	}
	raw, err := c.postJSON(ctx, input.Config, "/chat/completions", body, map[string]string{"Authorization": "Bearer " + input.Config.APIKey})
	if err != nil {
		return nil, err
	}
	var decoded struct {
		Choices []struct {
			Message struct {
				Content   string     `json:"content"`
				ToolCalls []ToolCall `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, fmt.Errorf("model provider: decode response: %w", err)
	}
	if len(decoded.Choices) == 0 {
		return nil, fmt.Errorf("model provider: response has no choices")
	}
	return &ChatResult{Text: decoded.Choices[0].Message.Content, ToolCalls: decoded.Choices[0].Message.ToolCalls, InputTokens: decoded.Usage.PromptTokens, OutputTokens: decoded.Usage.CompletionTokens}, nil
}

func (c *OpenAICompatibleClient) openAIResponses(ctx context.Context, input ChatRequest) (*ChatResult, error) {
	content := make([]map[string]string, 0, len(input.Messages))
	for _, message := range input.Messages {
		content = append(content, map[string]string{"type": "input_text", "text": message.Content})
	}
	body := map[string]any{"model": input.Config.Model, "input": content, "stream": false}
	if input.Config.Thinking.Enabled && input.Config.Thinking.Effort != "" {
		body["reasoning"] = map[string]string{"effort": input.Config.Thinking.Effort}
	}
	raw, err := c.postJSON(ctx, input.Config, "/responses", body, map[string]string{"Authorization": "Bearer " + input.Config.APIKey})
	if err != nil {
		return nil, err
	}
	var decoded struct {
		Output []struct {
			Type    string `json:"type"`
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
		OutputText string `json:"output_text"`
		Usage      struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, fmt.Errorf("model provider: decode Responses response: %w", err)
	}
	text := strings.TrimSpace(decoded.OutputText)
	if text == "" {
		for _, output := range decoded.Output {
			for _, item := range output.Content {
				if item.Type == "output_text" || item.Type == "text" {
					text += item.Text
				}
			}
		}
	}
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("model provider: Responses response has no text output")
	}
	return &ChatResult{Text: text, InputTokens: decoded.Usage.InputTokens, OutputTokens: decoded.Usage.OutputTokens}, nil
}

func (c *OpenAICompatibleClient) anthropicMessages(ctx context.Context, input ChatRequest) (*ChatResult, error) {
	messages := make([]ChatMessage, 0, len(input.Messages))
	system := make([]string, 0, 1)
	for _, message := range input.Messages {
		if message.Role == "system" {
			system = append(system, message.Content)
			continue
		}
		messages = append(messages, message)
	}
	body := map[string]any{"model": input.Config.Model, "messages": messages, "max_tokens": 4096}
	if len(system) > 0 {
		body["system"] = strings.Join(system, "\n\n")
	}
	if input.Config.Thinking.Enabled {
		budget := input.Config.Thinking.BudgetTokens
		if budget <= 0 {
			budget = 2048
		}
		body["thinking"] = map[string]any{"type": "enabled", "budget_tokens": budget}
		body["max_tokens"] = budget + 4096
	}
	raw, err := c.postJSON(ctx, input.Config, "/messages", body, map[string]string{
		"X-Api-Key": input.Config.APIKey, "Anthropic-Version": "2023-06-01",
	})
	if err != nil {
		return nil, err
	}
	var decoded struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, fmt.Errorf("model provider: decode Anthropic response: %w", err)
	}
	var texts []string
	for _, content := range decoded.Content {
		if content.Type == "text" && strings.TrimSpace(content.Text) != "" {
			texts = append(texts, content.Text)
		}
	}
	if len(texts) == 0 {
		return nil, fmt.Errorf("model provider: Anthropic response has no text output")
	}
	return &ChatResult{Text: strings.Join(texts, ""), InputTokens: decoded.Usage.InputTokens, OutputTokens: decoded.Usage.OutputTokens}, nil
}

func (c *OpenAICompatibleClient) postJSON(ctx context.Context, config domain.ResolvedModelConfig, path string, body any, headers map[string]string) ([]byte, error) {
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(config.BaseURL, "/")+path, bytes.NewReader(encoded))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("model provider: request failed: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxProviderResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("model provider: read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("model provider: HTTP %d: %s", resp.StatusCode, providerErrorMessage(raw))
	}
	return raw, nil
}

func providerErrorMessage(raw []byte) string {
	var decoded struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(raw, &decoded) == nil && decoded.Error.Message != "" {
		return decoded.Error.Message
	}
	message := strings.TrimSpace(string(raw))
	if len(message) > 512 {
		message = message[:512]
	}
	if message == "" {
		return "empty error response"
	}
	return message
}
