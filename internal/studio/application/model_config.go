package application

import (
	"context"
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

type ModelConfigService struct {
	Repo          ModelConfigRepository
	EncryptionKey []byte
	IDs           func() string
	Now           func() time.Time
}

type CreateModelConfigInput struct {
	AccountID    string
	Name         string
	Protocol     domain.ModelProtocol
	BaseURL      string
	Model        string
	APIKey       string
	Enabled      bool
	AgentEnabled bool
	Default      bool
	Thinking     domain.ThinkingConfig
	Capabilities domain.ModelCapabilities
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
	config.Default = input.Default
	config.Thinking = input.Thinking
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

func (s *ModelConfigService) Resolve(ctx context.Context, accountID, configID string) (*domain.ResolvedModelConfig, error) {
	config, err := s.Repo.GetModelConfig(ctx, accountID, configID)
	if err != nil {
		return nil, err
	}
	if !config.Enabled || !config.AgentEnabled {
		return nil, fmt.Errorf("%w: model is not available to Agent", domain.ErrInvalid)
	}
	apiKey, err := platformcrypto.Decrypt(s.EncryptionKey, config.APIKeyCipher)
	if err != nil {
		return nil, err
	}
	return &domain.ResolvedModelConfig{
		ID: config.ID, Name: config.Name, Protocol: config.Protocol, BaseURL: config.BaseURL,
		Model: config.Model, APIKey: apiKey, Thinking: config.Thinking, Capabilities: config.Capabilities,
	}, nil
}

func modelConfigView(config *domain.ModelConfig) *ModelConfigView {
	view := &ModelConfigView{
		ID: config.ID, Name: config.Name, Protocol: config.Protocol, BaseURL: config.BaseURL,
		Model: config.Model, HasAPIKey: config.APIKeyCipher != "", Enabled: config.Enabled,
		AgentEnabled: config.AgentEnabled, Default: config.Default, Thinking: config.Thinking,
		Capabilities: config.Capabilities, CreatedAt: config.CreatedAt, UpdatedAt: config.UpdatedAt,
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
