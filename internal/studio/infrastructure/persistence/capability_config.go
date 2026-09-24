package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

type SkillRow struct {
	ID          string    `gorm:"primaryKey;size:64"`
	AccountID   string    `gorm:"size:64;not null;index:idx_studio_skills_account_created"`
	Name        string    `gorm:"size:256;not null"`
	Description string    `gorm:"type:text"`
	Prompt      string    `gorm:"type:text;not null"`
	Enabled     bool      `gorm:"not null;index"`
	CreatedAt   time.Time `gorm:"index:idx_studio_skills_account_created"`
	UpdatedAt   time.Time
}

func (SkillRow) TableName() string { return "studio_skills" }

type MCPConnectorRow struct {
	ID               string    `gorm:"primaryKey;size:64"`
	AccountID        string    `gorm:"size:64;not null;index:idx_studio_mcp_connectors_account_created"`
	Name             string    `gorm:"size:256;not null"`
	URL              string    `gorm:"type:text;not null"`
	CredentialCipher string    `gorm:"type:text;not null"`
	Enabled          bool      `gorm:"not null;default:true;index"`
	Policy           string    `gorm:"size:32;not null"`
	DiscoveredTools  []byte    `gorm:"type:blob;not null"`
	CreatedAt        time.Time `gorm:"index:idx_studio_mcp_connectors_account_created"`
	UpdatedAt        time.Time
}

func (MCPConnectorRow) TableName() string { return "studio_mcp_connectors" }

type AgentWorkflowSettingRow struct {
	AccountID    string    `gorm:"primaryKey;size:64"`
	WorkflowID   string    `gorm:"primaryKey;size:64"`
	AgentEnabled bool      `gorm:"not null;default:false"`
	UpdatedAt    time.Time `gorm:"not null"`
}

func (AgentWorkflowSettingRow) TableName() string { return "studio_agent_workflow_settings" }

func (r *GormRepository) CreateSkill(ctx context.Context, skill *domain.Skill) error {
	if skill == nil {
		return domain.ErrInvalid
	}
	return translateCreateError(r.db.WithContext(ctx).Create(skillToRow(skill)).Error)
}

func (r *GormRepository) UpdateSkill(ctx context.Context, skill *domain.Skill) error {
	if skill == nil {
		return domain.ErrInvalid
	}
	return resultError(r.db.WithContext(ctx).Model(&SkillRow{}).Where("account_id = ? AND id = ?", skill.AccountID, skill.ID).
		Select("name", "description", "prompt", "enabled", "updated_at").Updates(skillToRow(skill)))
}

func (r *GormRepository) GetSkill(ctx context.Context, accountID, skillID string) (*domain.Skill, error) {
	var row SkillRow
	err := r.db.WithContext(ctx).Where("account_id = ? AND id = ?", accountID, skillID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return skillFromRow(row), nil
}

func (r *GormRepository) ListSkills(ctx context.Context, accountID string) ([]*domain.Skill, error) {
	var rows []SkillRow
	if err := r.db.WithContext(ctx).Where("account_id = ?", accountID).Order("created_at ASC, id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*domain.Skill, 0, len(rows))
	for _, row := range rows {
		out = append(out, skillFromRow(row))
	}
	return out, nil
}

func (r *GormRepository) CreateMCPConnector(ctx context.Context, connector *domain.MCPConnector) error {
	if connector == nil {
		return domain.ErrInvalid
	}
	row, err := connectorToRow(connector)
	if err != nil {
		return err
	}
	return translateCreateError(r.db.WithContext(ctx).Create(row).Error)
}

func (r *GormRepository) UpdateMCPConnector(ctx context.Context, connector *domain.MCPConnector) error {
	if connector == nil {
		return domain.ErrInvalid
	}
	row, err := connectorToRow(connector)
	if err != nil {
		return err
	}
	return resultError(r.db.WithContext(ctx).Model(&MCPConnectorRow{}).Where("account_id = ? AND id = ?", connector.AccountID, connector.ID).Updates(row))
}

func (r *GormRepository) GetMCPConnector(ctx context.Context, accountID, connectorID string) (*domain.MCPConnector, error) {
	var row MCPConnectorRow
	err := r.db.WithContext(ctx).Where("account_id = ? AND id = ?", accountID, connectorID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return connectorFromRow(row)
}

func (r *GormRepository) ListMCPConnectors(ctx context.Context, accountID string) ([]*domain.MCPConnector, error) {
	var rows []MCPConnectorRow
	if err := r.db.WithContext(ctx).Where("account_id = ?", accountID).Order("created_at ASC, id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*domain.MCPConnector, 0, len(rows))
	for _, row := range rows {
		connector, err := connectorFromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, connector)
	}
	return out, nil
}

func (r *GormRepository) UpsertAgentWorkflowSetting(ctx context.Context, setting *domain.AgentWorkflowSetting) error {
	if setting == nil || setting.AccountID == "" || setting.WorkflowID == "" {
		return domain.ErrInvalid
	}
	row := AgentWorkflowSettingRow{AccountID: setting.AccountID, WorkflowID: setting.WorkflowID, AgentEnabled: setting.AgentEnabled, UpdatedAt: setting.UpdatedAt}
	return r.db.WithContext(ctx).Save(&row).Error
}

func (r *GormRepository) ListAgentWorkflowSettings(ctx context.Context, accountID string) ([]*domain.AgentWorkflowSetting, error) {
	var rows []AgentWorkflowSettingRow
	if err := r.db.WithContext(ctx).Where("account_id = ?", accountID).Order("workflow_id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*domain.AgentWorkflowSetting, 0, len(rows))
	for _, row := range rows {
		out = append(out, &domain.AgentWorkflowSetting{AccountID: row.AccountID, WorkflowID: row.WorkflowID, AgentEnabled: row.AgentEnabled, UpdatedAt: row.UpdatedAt})
	}
	return out, nil
}

func skillToRow(skill *domain.Skill) *SkillRow {
	return &SkillRow{ID: skill.ID, AccountID: skill.AccountID, Name: skill.Name, Description: skill.Description, Prompt: skill.Prompt, Enabled: skill.Enabled, CreatedAt: skill.CreatedAt, UpdatedAt: skill.UpdatedAt}
}
func skillFromRow(row SkillRow) *domain.Skill {
	return &domain.Skill{ID: row.ID, AccountID: row.AccountID, Name: row.Name, Description: row.Description, Prompt: row.Prompt, Enabled: row.Enabled, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}

func connectorToRow(connector *domain.MCPConnector) (*MCPConnectorRow, error) {
	tools, err := json.Marshal(connector.DiscoveredTools)
	if err != nil {
		return nil, err
	}
	return &MCPConnectorRow{ID: connector.ID, AccountID: connector.AccountID, Name: connector.Name, URL: connector.URL, CredentialCipher: connector.CredentialCipher, Enabled: connector.Enabled, Policy: string(connector.Policy), DiscoveredTools: tools, CreatedAt: connector.CreatedAt, UpdatedAt: connector.UpdatedAt}, nil
}

func connectorFromRow(row MCPConnectorRow) (*domain.MCPConnector, error) {
	connector := &domain.MCPConnector{ID: row.ID, AccountID: row.AccountID, Name: row.Name, URL: row.URL, CredentialCipher: row.CredentialCipher, Enabled: row.Enabled, Policy: domain.ConnectorPolicy(row.Policy), CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
	if err := json.Unmarshal(row.DiscoveredTools, &connector.DiscoveredTools); err != nil {
		return nil, err
	}
	return connector, nil
}
