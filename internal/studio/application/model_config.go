package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	platformcrypto "github.com/Mr9esx/Pixoma/internal/platform/crypto"
	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

type ModelConfigRepository interface {
	CreateModelConfig(ctx context.Context, config *domain.ModelConfig) error
	UpdateModelConfig(ctx context.Context, config *domain.ModelConfig) error
	GetModelConfig(ctx context.Context, accountID, configID string) (*domain.ModelConfig, error)
	ListModelConfigs(ctx context.Context, accountID string) ([]*domain.ModelConfig, error)
}

// ModelConnectionTester is implemented by a protocol-aware infrastructure
// client. Keeping the contract here lets configuration stay independent of a
// specific provider SDK.
type ModelConnectionTester interface {
	Test(ctx context.Context, config domain.ResolvedModelConfig) error
}

var ErrModelConnectionTest = errors.New("model connection test failed")

type ModelConfigService struct {
	Repo          ModelConfigRepository
	EncryptionKey []byte
	IDs           func() string
	Now           func() time.Time
	Tester        ModelConnectionTester
}

type ModelConnectionTestResult struct {
	Success   bool  `json:"success"`
	LatencyMS int64 `json:"latency_ms"`
}

type CreateModelConfigInput struct {
	AccountID       string                   `json:"-"`
	ExistingModelID string                   `json:"existing_model_id,omitempty"`
	Name            string                   `json:"name"`
	Protocol        domain.ModelProtocol     `json:"protocol"`
	BaseURL         string                   `json:"base_url"`
	Model           string                   `json:"model"`
	APIKey          string                   `json:"api_key"`
	Enabled         bool                     `json:"enabled"`
	AgentEnabled    bool                     `json:"agent_enabled"`
	Default         bool                     `json:"default"`
	Thinking        domain.ThinkingConfig    `json:"thinking"`
	Limits          domain.ModelLimits       `json:"limits"`
	Capabilities    domain.ModelCapabilities `json:"capabilities"`
}

type UpdateModelConfigInput struct {
	Name         string                   `json:"name"`
	Protocol     domain.ModelProtocol     `json:"protocol"`
	BaseURL      string                   `json:"base_url"`
	Model        string                   `json:"model"`
	APIKey       string                   `json:"api_key"`
	Enabled      bool                     `json:"enabled"`
	AgentEnabled bool                     `json:"agent_enabled"`
	Default      bool                     `json:"default"`
	Thinking     domain.ThinkingConfig    `json:"thinking"`
	Limits       domain.ModelLimits       `json:"limits"`
	Capabilities domain.ModelCapabilities `json:"capabilities"`
}

type ModelConfigView struct {
	ID           string                   `json:"id"`
	Name         string                   `json:"name"`
	Protocol     domain.ModelProtocol     `json:"protocol"`
	BaseURL      string                   `json:"base_url"`
	Model        string                   `json:"model"`
	HasAPIKey    bool                     `json:"has_api_key"`
	APIKeyMasked string                   `json:"api_key_masked,omitempty"`
	Enabled      bool                     `json:"enabled"`
	AgentEnabled bool                     `json:"agent_enabled"`
	Default      bool                     `json:"default"`
	Thinking     domain.ThinkingConfig    `json:"thinking"`
	Limits       domain.ModelLimits       `json:"limits"`
	Capabilities domain.ModelCapabilities `json:"capabilities"`
	CreatedAt    time.Time                `json:"created_at"`
	UpdatedAt    time.Time                `json:"updated_at"`
}

func (s *ModelConfigService) Create(ctx context.Context, input CreateModelConfigInput) (*ModelConfigView, error) {
	if s == nil || s.Repo == nil {
		return nil, fmt.Errorf("studio: model config service is not configured")
	}
	if strings.TrimSpace(input.APIKey) == "" {
		return nil, fmt.Errorf("%w: API key is required", domain.ErrInvalid)
	}
	cipherText, err := platformcrypto.Encrypt(s.EncryptionKey, input.APIKey)
	if err != nil {
		return nil, err
	}
	config, err := domain.NewModelConfig(s.nextID(), input.AccountID, input.Name, input.Protocol, input.BaseURL, input.Model, cipherText, s.now())
	if err != nil {
		return nil, err
	}
	config.Enabled = input.Enabled
	config.AgentEnabled = input.AgentEnabled
	if err := validateAgentModelLimits(config.AgentEnabled, input.Limits); err != nil {
		return nil, err
	}
	config.Default = input.Default
	config.Thinking = input.Thinking
	config.Limits = input.Limits
	config.Capabilities = input.Capabilities
	if err := s.Repo.CreateModelConfig(ctx, config); err != nil {
		return nil, err
	}
	return modelConfigView(config), nil
}

func (s *ModelConfigService) List(ctx context.Context, accountID string) ([]*ModelConfigView, error) {
	configs, err := s.Repo.ListModelConfigs(ctx, accountID)
	if err != nil {
		return nil, err
	}
	out := make([]*ModelConfigView, 0, len(configs))
	for _, config := range configs {
		out = append(out, modelConfigView(config))
	}
	return out, nil
}

func (s *ModelConfigService) Update(ctx context.Context, accountID, configID string, input UpdateModelConfigInput) (*ModelConfigView, error) {
	if s == nil || s.Repo == nil {
		return nil, fmt.Errorf("studio: model config service is not configured")
	}
	existing, err := s.Repo.GetModelConfig(ctx, accountID, configID)
	if err != nil {
		return nil, err
	}
	apiKeyCipher := existing.APIKeyCipher
	if strings.TrimSpace(input.APIKey) != "" {
		apiKeyCipher, err = platformcrypto.Encrypt(s.EncryptionKey, input.APIKey)
		if err != nil {
			return nil, err
		}
	}
	updated, err := domain.NewModelConfig(
		existing.ID, accountID, input.Name, input.Protocol, input.BaseURL, input.Model, apiKeyCipher, s.now(),
	)
	if err != nil {
		return nil, err
	}
	updated.CreatedAt = existing.CreatedAt
	updated.Enabled = input.Enabled
	updated.AgentEnabled = input.AgentEnabled
	if err := validateAgentModelLimits(updated.AgentEnabled, input.Limits); err != nil {
		return nil, err
	}
	updated.Default = input.Default
	updated.Thinking = input.Thinking
	updated.Limits = input.Limits
	updated.Capabilities = input.Capabilities
	if err := s.Repo.UpdateModelConfig(ctx, updated); err != nil {
		return nil, err
	}
	return modelConfigView(updated), nil
}

func (s *ModelConfigService) Resolve(ctx context.Context, accountID, configID string) (*domain.ResolvedModelConfig, error) {
	config, err := s.Repo.GetModelConfig(ctx, accountID, configID)
	if err != nil {
		return nil, err
	}
	if !config.Enabled || !config.AgentEnabled {
		return nil, fmt.Errorf("%w: model is not available to Agent", domain.ErrInvalid)
	}
	if err := config.Limits.Validate(); err != nil {
		return nil, fmt.Errorf("%w: Agent model limits are not configured", err)
	}
	apiKey, err := platformcrypto.Decrypt(s.EncryptionKey, config.APIKeyCipher)
	if err != nil {
		return nil, err
	}
	return &domain.ResolvedModelConfig{
		ID: config.ID, Name: config.Name, Protocol: config.Protocol, BaseURL: config.BaseURL,
		Model: config.Model, APIKey: apiKey, Thinking: config.Thinking, Limits: config.Limits, Capabilities: config.Capabilities,
	}, nil
}

// TestConnection validates the stored protocol, endpoint, credentials and
// model with a minimal request. It intentionally works for disabled models so
// an administrator can repair a configuration before exposing it to Agent.
func (s *ModelConfigService) TestConnection(ctx context.Context, accountID, configID string) (*ModelConnectionTestResult, error) {
	if s == nil || s.Repo == nil || s.Tester == nil {
		return nil, fmt.Errorf("studio: model connection tester is not configured")
	}
	config, err := s.Repo.GetModelConfig(ctx, accountID, configID)
	if err != nil {
		return nil, err
	}
	apiKey, err := platformcrypto.Decrypt(s.EncryptionKey, config.APIKeyCipher)
	if err != nil {
		return nil, fmt.Errorf("%w: 已保存的 API Key 无法解密，请重新保存模型配置", ErrModelConnectionTest)
	}
	resolved := domain.ResolvedModelConfig{
		ID: config.ID, Name: config.Name, Protocol: config.Protocol, BaseURL: config.BaseURL,
		Model: config.Model, APIKey: apiKey, Thinking: config.Thinking, Limits: config.Limits, Capabilities: config.Capabilities,
	}
	return s.testResolvedConnection(ctx, resolved, apiKey)
}

// TestConnectionConfig tests a model configuration before it is persisted.
// The API key is kept in memory for the provider request and is never written
// to the model repository.
func (s *ModelConfigService) TestConnectionConfig(ctx context.Context, input CreateModelConfigInput) (*ModelConnectionTestResult, error) {
	if s == nil || s.Tester == nil {
		return nil, fmt.Errorf("studio: model connection tester is not configured")
	}
	apiKey := input.APIKey
	if strings.TrimSpace(apiKey) == "" && strings.TrimSpace(input.ExistingModelID) != "" {
		if s.Repo == nil {
			return nil, fmt.Errorf("studio: model config repository is not configured")
		}
		stored, err := s.Repo.GetModelConfig(ctx, input.AccountID, input.ExistingModelID)
		if err != nil {
			return nil, err
		}
		apiKey, err = platformcrypto.Decrypt(s.EncryptionKey, stored.APIKeyCipher)
		if err != nil {
			return nil, fmt.Errorf("%w: 已保存的 API Key 无法解密，请重新保存模型配置", ErrModelConnectionTest)
		}
	}
	if strings.TrimSpace(apiKey) == "" {
		return nil, fmt.Errorf("%w: API key is required", domain.ErrInvalid)
	}
	config, err := domain.NewModelConfig(
		s.nextID(), input.AccountID, input.Name, input.Protocol, input.BaseURL, input.Model, "transient-test-key", s.now(),
	)
	if err != nil {
		return nil, err
	}
	resolved := domain.ResolvedModelConfig{
		ID: config.ID, Name: config.Name, Protocol: config.Protocol, BaseURL: config.BaseURL,
		Model: config.Model, APIKey: apiKey, Thinking: input.Thinking, Limits: input.Limits, Capabilities: input.Capabilities,
	}
	return s.testResolvedConnection(ctx, resolved, apiKey)
}

func (s *ModelConfigService) testResolvedConnection(ctx context.Context, resolved domain.ResolvedModelConfig, apiKey string) (*ModelConnectionTestResult, error) {
	testContext, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	started := time.Now()
	if err := s.Tester.Test(testContext, resolved); err != nil {
		return nil, sanitizeModelTestError(err, apiKey)
	}
	return &ModelConnectionTestResult{Success: true, LatencyMS: time.Since(started).Milliseconds()}, nil
}

func sanitizeModelTestError(err error, apiKey string) error {
	message := strings.ReplaceAll(err.Error(), apiKey, "[REDACTED]")
	if len(message) > 512 {
		message = message[:512]
	}
	return fmt.Errorf("%w: %s", ErrModelConnectionTest, message)
}

func modelConfigView(config *domain.ModelConfig) *ModelConfigView {
	view := &ModelConfigView{
		ID: config.ID, Name: config.Name, Protocol: config.Protocol, BaseURL: config.BaseURL,
		Model: config.Model, HasAPIKey: config.APIKeyCipher != "", Enabled: config.Enabled,
		AgentEnabled: config.AgentEnabled, Default: config.Default, Thinking: config.Thinking,
		Limits: config.Limits, Capabilities: config.Capabilities, CreatedAt: config.CreatedAt, UpdatedAt: config.UpdatedAt,
	}
	if view.HasAPIKey {
		view.APIKeyMasked = "••••••••"
	}
	return view
}

func (s *ModelConfigService) nextID() string {
	if s.IDs != nil {
		return s.IDs()
	}
	return uuid.NewString()
}

func (s *ModelConfigService) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func validateAgentModelLimits(agentEnabled bool, limits domain.ModelLimits) error {
	if !agentEnabled {
		return nil
	}
	if err := limits.Validate(); err != nil {
		return fmt.Errorf("%w: Agent model limits are required", err)
	}
	return nil
}
