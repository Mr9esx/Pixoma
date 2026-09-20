package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

type ModelConfigRow struct {
	ID               string `gorm:"primaryKey;size:64"`
	AccountID        string `gorm:"size:64;not null;index"`
	Name             string `gorm:"size:256;not null"`
	Protocol         string `gorm:"size:64;not null;index"`
	BaseURL          string `gorm:"type:text;not null"`
	Model            string `gorm:"size:256;not null"`
	APIKeyCipher     string `gorm:"type:text;not null"`
	Enabled          bool   `gorm:"not null;default:true"`
	AgentEnabled     bool   `gorm:"not null;default:true;index"`
	IsDefault        bool   `gorm:"not null;default:false;index"`
	ThinkingJSON     []byte `gorm:"type:blob;not null"`
	CapabilitiesJSON []byte `gorm:"type:blob;not null"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (ModelConfigRow) TableName() string { return "studio_model_configs" }

func (r *GormRepository) CreateModelConfig(ctx context.Context, config *domain.ModelConfig) error {
	if config == nil {
		return domain.ErrInvalid
	}
	row, err := modelConfigToRow(config)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if config.Default {
			if err := tx.Model(&ModelConfigRow{}).Where("account_id = ?", config.AccountID).Update("is_default", false).Error; err != nil {
				return err
			}
		}
		return translateCreateError(tx.Create(row).Error)
	})
}

func (r *GormRepository) UpdateModelConfig(ctx context.Context, config *domain.ModelConfig) error {
	if config == nil {
		return domain.ErrInvalid
	}
	row, err := modelConfigToRow(config)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if config.Default {
			if err := tx.Model(&ModelConfigRow{}).
				Where("account_id = ? AND id <> ?", config.AccountID, config.ID).
				Update("is_default", false).Error; err != nil {
				return err
			}
		}
		result := tx.Model(&ModelConfigRow{}).Where("account_id = ? AND id = ?", config.AccountID, config.ID).Updates(row)
		return resultError(result)
	})
}

func (r *GormRepository) GetModelConfig(ctx context.Context, accountID, configID string) (*domain.ModelConfig, error) {
	var row ModelConfigRow
	err := r.db.WithContext(ctx).Where("account_id = ? AND id = ?", accountID, configID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return modelConfigFromRow(row)
}

func (r *GormRepository) ListModelConfigs(ctx context.Context, accountID string) ([]*domain.ModelConfig, error) {
	var rows []ModelConfigRow
	if err := r.db.WithContext(ctx).Where("account_id = ?", accountID).
		Order("is_default DESC, created_at ASC, id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*domain.ModelConfig, 0, len(rows))
	for _, row := range rows {
		config, err := modelConfigFromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, config)
	}
	return out, nil
}

func modelConfigToRow(config *domain.ModelConfig) (*ModelConfigRow, error) {
	thinking, err := json.Marshal(config.Thinking)
	if err != nil {
		return nil, err
	}
	capabilities, err := json.Marshal(config.Capabilities)
	if err != nil {
		return nil, err
	}
	return &ModelConfigRow{
		ID: config.ID, AccountID: config.AccountID, Name: config.Name, Protocol: string(config.Protocol),
		BaseURL: config.BaseURL, Model: config.Model, APIKeyCipher: config.APIKeyCipher,
		Enabled: config.Enabled, AgentEnabled: config.AgentEnabled, IsDefault: config.Default,
		ThinkingJSON: thinking, CapabilitiesJSON: capabilities,
		CreatedAt: config.CreatedAt, UpdatedAt: config.UpdatedAt,
	}, nil
}

func modelConfigFromRow(row ModelConfigRow) (*domain.ModelConfig, error) {
	config := &domain.ModelConfig{
		ID: row.ID, AccountID: row.AccountID, Name: row.Name, Protocol: domain.ModelProtocol(row.Protocol),
		BaseURL: row.BaseURL, Model: row.Model, APIKeyCipher: row.APIKeyCipher,
		Enabled: row.Enabled, AgentEnabled: row.AgentEnabled, Default: row.IsDefault,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
	if err := json.Unmarshal(row.ThinkingJSON, &config.Thinking); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(row.CapabilitiesJSON, &config.Capabilities); err != nil {
		return nil, err
	}
	return config, nil
}
