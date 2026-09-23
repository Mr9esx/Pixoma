package modelprovider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Mr9esx/Pixoma/internal/studio/domain"
	"github.com/cloudwego/eino/schema"
)

const maxProviderResponseBytes = 4 << 20

type ChatMessage struct {
	Role            string            `json:"role"`
	Content         string            `json:"content"`
	ToolCallID      string            `json:"tool_call_id,omitempty"`
	ToolCalls       []ToolCall        `json:"tool_calls,omitempty"`
	ResponsesOutput []json.RawMessage `json:"-"`
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
	Trace    TraceSink
}

type ChatResult struct {
	Text            string
	Reasoning       string
	InputTokens     int
	OutputTokens    int
	ToolCalls       []ToolCall
	ResponsesOutput []json.RawMessage
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

// StreamChat exposes OpenAI Chat Completions deltas as Eino message chunks.
// Protocols with different stream envelopes continue through the safe
// non-streaming path in EinoChatModel until their adapters are added.
func (c *OpenAICompatibleClient) StreamChat(ctx context.Context, input ChatRequest) (*schema.StreamReader[*schema.Message], error) {
	if input.Config.Protocol != "" && input.Config.Protocol != domain.ModelProtocolOpenAIChat {
		return nil, fmt.Errorf("model provider: streaming is not implemented for protocol %q", input.Config.Protocol)
	}
	if len(input.Tools) > 0 {
		return nil, fmt.Errorf("model provider: streaming tool calls are not implemented")
	}
	body := map[string]any{"model": input.Config.Model, "messages": input.Messages, "stream": true, "max_tokens": configuredMaxOutputTokens(input.Config)}
	if input.Config.Thinking.Enabled && input.Config.Thinking.Effort != "" {
		body["reasoning_effort"] = input.Config.Thinking.Effort
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	attempt, err := beginTrace(ctx, input.Trace, encoded, input.Config.APIKey)
	if err != nil {
		return nil, fmt.Errorf("model provider: persist request trace: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, input.Config.BaseURL, bytes.NewReader(encoded))
	if err != nil {
		_ = attempt.fail(ctx, err)
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+input.Config.APIKey)
	resp, err := c.client.Do(req)
	if err != nil {
		wrapped := fmt.Errorf("model provider: request failed: %w", err)
		_ = attempt.fail(ctx, wrapped)
		return nil, wrapped
	}
	attempt.receivedResponse(resp.StatusCode, providerRequestID(resp.Header))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, maxProviderResponseBytes))
		wrapped := fmt.Errorf("model provider: HTTP %d: %s", resp.StatusCode, providerErrorMessage(raw, input.Config.APIKey))
		_ = attempt.failWithBody(ctx, wrapped, raw)
		return nil, wrapped
	}
	if !strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "text/event-stream") {
		resp.Body.Close()
		wrapped := fmt.Errorf("model provider: streaming response is not an event stream")
		_ = attempt.fail(ctx, wrapped)
		return nil, wrapped
	}
	reader, writer := schema.Pipe[*schema.Message](16)
	go func() {
		defer resp.Body.Close()
		defer writer.Close()
		limited := &io.LimitedReader{R: resp.Body, N: maxProviderResponseBytes + 1}
		scanner := bufio.NewScanner(limited)
		scanner.Buffer(make([]byte, 64*1024), maxProviderResponseBytes)
		var reasoning strings.Builder
		firstTokenSeen := false
		inputTokens, outputTokens := 0, 0
		usageReported := false
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if !strings.HasPrefix(line, "data:") {
				continue
			}
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if data == "" {
				continue
			}
			if data == "[DONE]" {
				break
			}
			attempt.recordChunk([]byte(data))
			var chunk struct {
				Choices []struct {
					Delta struct {
						Content   string `json:"content"`
						Reasoning string `json:"reasoning_content"`
					} `json:"delta"`
				} `json:"choices"`
				Usage *struct {
					PromptTokens     int `json:"prompt_tokens"`
					CompletionTokens int `json:"completion_tokens"`
				} `json:"usage"`
			}
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				wrapped := fmt.Errorf("model provider: decode streaming response: %w", err)
				_ = attempt.fail(ctx, wrapped)
				writer.Send(nil, wrapped)
				return
			}
			if chunk.Usage != nil {
				usageReported = true
				inputTokens = chunk.Usage.PromptTokens
				outputTokens = chunk.Usage.CompletionTokens
			}
			if len(chunk.Choices) == 0 {
				continue
			}
			delta := chunk.Choices[0].Delta
			if delta.Reasoning != "" {
				reasoning.WriteString(delta.Reasoning)
			}
			message := &schema.Message{Role: schema.Assistant, Content: delta.Content}
			if reasoning.Len() > 0 {
				message.Extra = map[string]any{reasoningExtraKey: reasoning.String()}
				reasoning.Reset()
			}
			if delta.Content != "" || len(message.Extra) > 0 {
				if !firstTokenSeen {
					if err := attempt.firstToken(ctx); err != nil {
						writer.Send(nil, fmt.Errorf("model provider: persist first-token trace: %w", err))
						return
					}
					firstTokenSeen = true
				}
				if writer.Send(message, nil) {
					return
				}
			}
		}
		if err := scanner.Err(); err != nil {
			wrapped := fmt.Errorf("model provider: read streaming response: %w", err)
			_ = attempt.fail(ctx, wrapped)
			writer.Send(nil, wrapped)
			return
		}
		if limited.N == 0 {
			wrapped := fmt.Errorf("model provider: streaming response exceeds %d bytes", maxProviderResponseBytes)
			_ = attempt.fail(ctx, wrapped)
			writer.Send(nil, wrapped)
			return
		}
		if err := attempt.finish(ctx, inputTokens, outputTokens, usageReported, nil); err != nil {
			writer.Send(nil, fmt.Errorf("model provider: persist request trace: %w", err))
		}
	}()
	return reader, nil
}

func (c *OpenAICompatibleClient) openAIChat(ctx context.Context, input ChatRequest) (*ChatResult, error) {
	body := map[string]any{"model": input.Config.Model, "messages": input.Messages, "stream": false, "max_tokens": configuredMaxOutputTokens(input.Config)}
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
	response, err := c.postJSON(ctx, input.Config, body, map[string]string{"Authorization": "Bearer " + input.Config.APIKey}, input.Trace)
	if err != nil {
		return nil, err
	}
	raw := response.raw
	var decoded struct {
		Choices []struct {
			Message struct {
				Content   string     `json:"content"`
				Reasoning string     `json:"reasoning_content"`
				ToolCalls []ToolCall `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
		Usage *struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		wrapped := fmt.Errorf("model provider: decode response: %w", err)
		_ = response.trace.failWithBody(ctx, wrapped, raw)
		return nil, wrapped
	}
	if len(decoded.Choices) == 0 {
		_ = response.trace.failWithBody(ctx, fmt.Errorf("model provider: response has no choices"), raw)
		return nil, fmt.Errorf("model provider: response has no choices")
	}
	inputTokens, outputTokens := 0, 0
	if decoded.Usage != nil {
		inputTokens, outputTokens = decoded.Usage.PromptTokens, decoded.Usage.CompletionTokens
	}
	if err := response.trace.finish(ctx, inputTokens, outputTokens, decoded.Usage != nil, raw); err != nil {
		return nil, fmt.Errorf("model provider: persist request trace: %w", err)
	}
	return &ChatResult{Text: decoded.Choices[0].Message.Content, Reasoning: decoded.Choices[0].Message.Reasoning, ToolCalls: decoded.Choices[0].Message.ToolCalls, InputTokens: inputTokens, OutputTokens: outputTokens}, nil
}

func (c *OpenAICompatibleClient) openAIResponses(ctx context.Context, input ChatRequest) (*ChatResult, error) {
	responseInput, err := responsesInput(input.Messages)
	if err != nil {
		return nil, err
	}
	body := map[string]any{"model": input.Config.Model, "input": responseInput, "stream": false, "store": false, "max_output_tokens": configuredMaxOutputTokens(input.Config)}
	if len(input.Tools) > 0 {
		tools := make([]map[string]any, 0, len(input.Tools))
		for _, definition := range input.Tools {
			tools = append(tools, map[string]any{
				"type": "function", "name": definition.Name, "description": definition.Description, "parameters": definition.Parameters,
			})
		}
		body["tools"] = tools
		body["tool_choice"] = "auto"
	}
	if input.Config.Thinking.Enabled && input.Config.Thinking.Effort != "" {
		body["reasoning"] = map[string]string{"effort": input.Config.Thinking.Effort}
	}
	response, err := c.postJSON(ctx, input.Config, body, map[string]string{"Authorization": "Bearer " + input.Config.APIKey}, input.Trace)
	if err != nil {
		return nil, err
	}
	raw := response.raw
	var decoded struct {
		Output     []json.RawMessage `json:"output"`
		OutputText string            `json:"output_text"`
		Usage      *struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		wrapped := fmt.Errorf("model provider: decode Responses response: %w", err)
		_ = response.trace.failWithBody(ctx, wrapped, raw)
		return nil, wrapped
	}
	type outputItem struct {
		Type      string `json:"type"`
		CallID    string `json:"call_id"`
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
		Content   []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Summary []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"summary"`
	}
	output := make([]outputItem, 0, len(decoded.Output))
	for _, rawOutput := range decoded.Output {
		var item outputItem
		if err := json.Unmarshal(rawOutput, &item); err != nil {
			wrapped := fmt.Errorf("model provider: decode Responses output item: %w", err)
			_ = response.trace.failWithBody(ctx, wrapped, raw)
			return nil, wrapped
		}
		output = append(output, item)
	}
	text := strings.TrimSpace(decoded.OutputText)
	if text == "" {
		for _, item := range output {
			for _, content := range item.Content {
				if content.Type == "output_text" || content.Type == "text" {
					text += content.Text
				}
			}
		}
	}
	toolCalls := make([]ToolCall, 0)
	var reasoning strings.Builder
	for _, item := range output {
		if item.Type == "reasoning" {
			for _, summary := range item.Summary {
				if strings.TrimSpace(summary.Text) != "" {
					reasoning.WriteString(summary.Text)
				}
			}
		}
		if item.Type != "function_call" || strings.TrimSpace(item.CallID) == "" || strings.TrimSpace(item.Name) == "" {
			continue
		}
		toolCalls = append(toolCalls, ToolCall{
			ID: item.CallID, Type: "function",
			Function: FunctionCall{Name: item.Name, Arguments: item.Arguments},
		})
	}
	if strings.TrimSpace(text) == "" && len(toolCalls) == 0 {
		_ = response.trace.failWithBody(ctx, fmt.Errorf("model provider: Responses response has no text output"), raw)
		return nil, fmt.Errorf("model provider: Responses response has no text output")
	}
	inputTokens, outputTokens := 0, 0
	if decoded.Usage != nil {
		inputTokens, outputTokens = decoded.Usage.InputTokens, decoded.Usage.OutputTokens
	}
	if err := response.trace.finish(ctx, inputTokens, outputTokens, decoded.Usage != nil, raw); err != nil {
		return nil, fmt.Errorf("model provider: persist request trace: %w", err)
	}
	return &ChatResult{Text: text, Reasoning: reasoning.String(), ToolCalls: toolCalls, ResponsesOutput: cloneResponsesOutput(decoded.Output), InputTokens: inputTokens, OutputTokens: outputTokens}, nil
}

func responsesInput(messages []ChatMessage) ([]any, error) {
	input := make([]any, 0, len(messages))
	for _, message := range messages {
		if len(message.ResponsesOutput) > 0 {
			for _, rawOutput := range message.ResponsesOutput {
				var item map[string]any
				if err := json.Unmarshal(rawOutput, &item); err != nil {
					return nil, fmt.Errorf("model provider: invalid saved Responses output: %w", err)
				}
				input = append(input, append(json.RawMessage(nil), rawOutput...))
			}
			continue
		}
		if message.Role == "tool" {
			input = append(input, map[string]string{"type": "function_call_output", "call_id": message.ToolCallID, "output": message.Content})
			continue
		}
		if len(message.ToolCalls) > 0 {
			for _, call := range message.ToolCalls {
				input = append(input, map[string]string{"type": "function_call", "call_id": call.ID, "name": call.Function.Name, "arguments": call.Function.Arguments})
			}
		}
		if strings.TrimSpace(message.Content) == "" {
			continue
		}
		role := message.Role
		if role == "" {
			role = "user"
		}
		input = append(input, map[string]any{"role": role, "content": []map[string]string{{"type": "input_text", "text": message.Content}}})
	}
	return input, nil
}

func cloneResponsesOutput(output []json.RawMessage) []json.RawMessage {
	cloned := make([]json.RawMessage, 0, len(output))
	for _, item := range output {
		cloned = append(cloned, append(json.RawMessage(nil), item...))
	}
	return cloned
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
	body := map[string]any{"model": input.Config.Model, "messages": messages, "max_tokens": configuredMaxOutputTokens(input.Config)}
	if len(system) > 0 {
		body["system"] = strings.Join(system, "\n\n")
	}
	if input.Config.Thinking.Enabled {
		budget := input.Config.Thinking.BudgetTokens
		if budget <= 0 {
			budget = 2048
		}
		body["thinking"] = map[string]any{"type": "enabled", "budget_tokens": budget}
		body["max_tokens"] = budget + configuredMaxOutputTokens(input.Config)
	}
	response, err := c.postJSON(ctx, input.Config, body, map[string]string{
		"X-Api-Key": input.Config.APIKey, "Anthropic-Version": "2023-06-01",
	}, input.Trace)
	if err != nil {
		return nil, err
	}
	raw := response.raw
	var decoded struct {
		Content []struct {
			Type     string `json:"type"`
			Text     string `json:"text"`
			Thinking string `json:"thinking"`
		} `json:"content"`
		Usage *struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		wrapped := fmt.Errorf("model provider: decode Anthropic response: %w", err)
		_ = response.trace.failWithBody(ctx, wrapped, raw)
		return nil, wrapped
	}
	var texts []string
	var reasoning []string
	for _, content := range decoded.Content {
		if content.Type == "thinking" && strings.TrimSpace(content.Thinking) != "" {
			reasoning = append(reasoning, content.Thinking)
		}
		if content.Type == "text" && strings.TrimSpace(content.Text) != "" {
			texts = append(texts, content.Text)
		}
	}
	if len(texts) == 0 {
		_ = response.trace.failWithBody(ctx, fmt.Errorf("model provider: Anthropic response has no text output"), raw)
		return nil, fmt.Errorf("model provider: Anthropic response has no text output")
	}
	inputTokens, outputTokens := 0, 0
	if decoded.Usage != nil {
		inputTokens, outputTokens = decoded.Usage.InputTokens, decoded.Usage.OutputTokens
	}
	if err := response.trace.finish(ctx, inputTokens, outputTokens, decoded.Usage != nil, raw); err != nil {
		return nil, fmt.Errorf("model provider: persist request trace: %w", err)
	}
	return &ChatResult{Text: strings.Join(texts, ""), Reasoning: strings.Join(reasoning, ""), InputTokens: inputTokens, OutputTokens: outputTokens}, nil
}

func configuredMaxOutputTokens(config domain.ResolvedModelConfig) int {
	if config.Limits.MaxOutputTokens > 0 {
		return config.Limits.MaxOutputTokens
	}
	return 4096
}

type providerJSONResponse struct {
	raw   []byte
	trace *traceAttempt
}

func (c *OpenAICompatibleClient) postJSON(ctx context.Context, config domain.ResolvedModelConfig, body any, headers map[string]string, trace TraceSink) (*providerJSONResponse, error) {
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	attempt, err := beginTrace(ctx, trace, encoded, config.APIKey)
	if err != nil {
		return nil, fmt.Errorf("model provider: persist request trace: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, config.BaseURL, bytes.NewReader(encoded))
	if err != nil {
		_ = attempt.fail(ctx, err)
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		wrapped := fmt.Errorf("model provider: request failed: %w", err)
		_ = attempt.fail(ctx, wrapped)
		return nil, wrapped
	}
	defer resp.Body.Close()
	attempt.receivedResponse(resp.StatusCode, providerRequestID(resp.Header))
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxProviderResponseBytes+1))
	if err != nil {
		wrapped := fmt.Errorf("model provider: read response: %w", err)
		_ = attempt.fail(ctx, wrapped)
		return nil, wrapped
	}
	if len(raw) > maxProviderResponseBytes {
		wrapped := fmt.Errorf("model provider: response exceeds %d bytes", maxProviderResponseBytes)
		_ = attempt.fail(ctx, wrapped)
		return nil, wrapped
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		wrapped := fmt.Errorf("model provider: HTTP %d: %s", resp.StatusCode, providerErrorMessage(raw, config.APIKey))
		_ = attempt.failWithBody(ctx, wrapped, raw)
		return nil, wrapped
	}
	return &providerJSONResponse{raw: raw, trace: attempt}, nil
}

func providerRequestID(headers http.Header) string {
	for _, key := range []string{"X-Request-ID", "Request-ID", "X-Request-Id", "Anthropic-Request-Id"} {
		if value := strings.TrimSpace(headers.Get(key)); value != "" {
			return value
		}
	}
	return ""
}

func providerErrorMessage(raw []byte, credential string) string {
	var decoded struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(raw, &decoded) == nil && decoded.Error.Message != "" {
		message := decoded.Error.Message
		if credential != "" {
			message = strings.ReplaceAll(message, credential, "[REDACTED]")
		}
		return message
	}
	message := strings.TrimSpace(string(raw))
	if credential != "" {
		message = strings.ReplaceAll(message, credential, "[REDACTED]")
	}
	if len(message) > 512 {
		message = message[:512]
	}
	if message == "" {
		return "empty error response"
	}
	return message
}
