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
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Config   domain.ResolvedModelConfig
	Messages []ChatMessage
}

type ChatResult struct {
	Text         string
	InputTokens  int
	OutputTokens int
}

type OpenAICompatibleClient struct {
	client *http.Client
}

func NewOpenAICompatibleClient(client *http.Client) *OpenAICompatibleClient {
	if client == nil {
		client = &http.Client{Timeout: 90 * time.Second}
	}
	return &OpenAICompatibleClient{client: client}
}

func (c *OpenAICompatibleClient) Chat(ctx context.Context, input ChatRequest) (*ChatResult, error) {
	if strings.TrimSpace(input.Config.BaseURL) == "" || strings.TrimSpace(input.Config.Model) == "" || strings.TrimSpace(input.Config.APIKey) == "" {
		return nil, fmt.Errorf("model provider: incomplete OpenAI compatible config")
	}
	body, err := json.Marshal(map[string]any{
		"model": input.Config.Model, "messages": input.Messages, "stream": false,
	})
	if err != nil {
		return nil, err
	}
	endpoint := strings.TrimRight(input.Config.BaseURL, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+input.Config.APIKey)
	req.Header.Set("Content-Type", "application/json")
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
	var decoded struct {
		Choices []struct {
			Message ChatMessage `json:"message"`
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
	return &ChatResult{Text: decoded.Choices[0].Message.Content, InputTokens: decoded.Usage.PromptTokens, OutputTokens: decoded.Usage.CompletionTokens}, nil
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
