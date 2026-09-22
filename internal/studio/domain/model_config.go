package domain

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

type ModelProtocol string

const (
	ModelProtocolOpenAIResponses ModelProtocol = "openai_responses"
	ModelProtocolOpenAIChat      ModelProtocol = "openai_chat_compatible"
	ModelProtocolAnthropic       ModelProtocol = "anthropic_messages_compatible"
)

func (p ModelProtocol) Valid() bool {
	switch p {
	case ModelProtocolOpenAIResponses, ModelProtocolOpenAIChat, ModelProtocolAnthropic:
		return true
	default:
		return false
	}
}

type ThinkingConfig struct {
	Enabled      bool   `json:"enabled"`
	Effort       string `json:"effort,omitempty"`
	BudgetTokens int    `json:"budget_tokens,omitempty"`
}

// ModelLimits describes the provider limits needed to build a safe Agent
// context budget. They are configured per model because compatible endpoints
// do not expose a reliable common limit contract.
type ModelLimits struct {
	ContextWindowTokens int `json:"context_window_tokens"`
	MaxInputTokens      int `json:"max_input_tokens"`
	MaxOutputTokens     int `json:"max_output_tokens"`
}

func (l ModelLimits) Validate() error {
	if l.ContextWindowTokens <= 0 || l.MaxInputTokens <= 0 || l.MaxOutputTokens <= 0 {
		return fmt.Errorf("%w: model token limits must be positive", ErrInvalid)
	}
	if l.MaxInputTokens > l.ContextWindowTokens {
		return fmt.Errorf("%w: max input tokens exceed context window", ErrInvalid)
	}
	if l.MaxOutputTokens > l.ContextWindowTokens {
		return fmt.Errorf("%w: max output tokens exceed context window", ErrInvalid)
	}
	return nil
}

type ModelCapabilities struct {
	Tools       bool `json:"tools"`
	Vision      bool `json:"vision"`
	ImageOutput bool `json:"image_output"`
	Streaming   bool `json:"streaming"`
}

type ModelConfig struct {
	ID           string
	AccountID    string
	Name         string
	Protocol     ModelProtocol
	BaseURL      string
	Model        string
	APIKeyCipher string
	Enabled      bool
	AgentEnabled bool
	Default      bool
	Thinking     ThinkingConfig
	Limits       ModelLimits
	Capabilities ModelCapabilities
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type ResolvedModelConfig struct {
	ID           string
	Name         string
	Protocol     ModelProtocol
	BaseURL      string
	Model        string
	APIKey       string
	Thinking     ThinkingConfig
	Limits       ModelLimits
	Capabilities ModelCapabilities
}

func NewModelConfig(id, accountID, name string, protocol ModelProtocol, baseURL, model, apiKeyCipher string, now time.Time) (*ModelConfig, error) {
	baseURL = strings.TrimSpace(baseURL)
	if anyBlank(id, accountID, name, baseURL, model, apiKeyCipher) || !protocol.Valid() {
		return nil, fmt.Errorf("%w: invalid model config", ErrInvalid)
	}
	parsed, err := url.ParseRequestURI(strings.TrimSpace(baseURL))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return nil, fmt.Errorf("%w: model base URL must be HTTP(S)", ErrInvalid)
	}
	now = now.UTC()
	return &ModelConfig{
		ID: id, AccountID: accountID, Name: strings.TrimSpace(name), Protocol: protocol,
		BaseURL: baseURL, Model: strings.TrimSpace(model),
		APIKeyCipher: apiKeyCipher, CreatedAt: now, UpdatedAt: now,
	}, nil
}
