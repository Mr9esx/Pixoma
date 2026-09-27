package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

type SkillRow struct {
	ID          string    `gorm:"primaryKey;size:64"`
	AccountID   string    `gorm:"size:64;not null;index:idx_studio_skills_account_created"`
	Name        string    `gorm:"size:256;not null"`
	Description string    `gorm:"type:text"`
	Prompt      string    `gorm:"type:text;not null"`
	FilesJSON   []byte    `gorm:"type:blob"`
	Version     string    `gorm:"size:255;not null;default:1.0.0"`
	Enabled     bool      `gorm:"not null;index"`
	CreatedAt   time.Time `gorm:"index:idx_studio_skills_account_created"`
	UpdatedAt   time.Time
}

func (SkillRow) TableName() string { return "studio_skills" }

type SkillVersionRow struct {
	SkillID     string    `gorm:"primaryKey;size:64"`
	Version     string    `gorm:"primaryKey;size:255"`
	AccountID   string    `gorm:"size:64;not null"`
	Name        string    `gorm:"size:256;not null"`
	Description string    `gorm:"type:text"`
	Prompt      string    `gorm:"type:text;not null"`
	FilesJSON   []byte    `gorm:"type:blob"`
	CreatedAt   time.Time `gorm:"not null"`
}

func (SkillVersionRow) TableName() string { return "studio_skill_versions" }

func MigrateSkillVersions(ctx context.Context, db *gorm.DB) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var rows []SkillRow
		if err := tx.Where("NOT EXISTS (SELECT 1 FROM studio_skill_versions WHERE skill_id = studio_skills.id AND version = studio_skills.version)").Find(&rows).Error; err != nil {
			return err
		}
		for _, row := range rows {
			if row.Version == "" {
				row.Version = "1.0.0"
				if err := tx.Model(&SkillRow{}).Where("id = ?", row.ID).UpdateColumn("version", row.Version).Error; err != nil {
					return err
				}
			}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(skillVersionToRow(row, row.UpdatedAt)).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

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
	if skill == nil || !domain.ValidSkillVersion(skill.Version) {
		return domain.ErrInvalid
	}
	row, err := skillToRow(skill)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := translateCreateError(tx.Create(row).Error); err != nil {
			return err
		}
		return tx.Create(skillVersionToRow(*row, skill.CreatedAt)).Error
	})
}

func (r *GormRepository) UpdateSkillIfUnchanged(ctx context.Context, skill *domain.Skill, expected time.Time, expectedVersion string) error {
	if skill == nil || expected.IsZero() || !domain.ValidSkillVersion(skill.Version) || !domain.ValidSkillVersion(expectedVersion) {
		return domain.ErrInvalid
	}
	if domain.CompareSkillVersions(skill.Version, expectedVersion) <= 0 {
		return domain.ErrConflict
	}
	row, err := skillToRow(skill)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&SkillRow{}).
			Where("account_id = ? AND id = ? AND updated_at = ? AND version = ?", skill.AccountID, skill.ID, expected, expectedVersion).
			UpdateColumns(map[string]any{
				"name": row.Name, "description": row.Description, "prompt": row.Prompt,
				"files_json": row.FilesJSON, "version": row.Version, "updated_at": row.UpdatedAt,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return domain.ErrConflict
		}
		if err := tx.Create(skillVersionToRow(*row, skill.UpdatedAt)).Error; err != nil {
			if errors.Is(translateCreateError(err), domain.ErrAlreadyExists) {
				return domain.ErrConflict
			}
			return err
		}
		return nil
	})
}

func (r *GormRepository) SetSkillEnabled(ctx context.Context, accountID, skillID string, enabled bool) error {
	return resultError(r.db.WithContext(ctx).Model(&SkillRow{}).
		Where("account_id = ? AND id = ?", accountID, skillID).UpdateColumn("enabled", enabled))
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
	return skillFromRow(row)
}

func (r *GormRepository) ListSkills(ctx context.Context, accountID string) ([]*domain.Skill, error) {
	var rows []SkillRow
	if err := r.db.WithContext(ctx).Where("account_id = ?", accountID).Order("created_at ASC, id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*domain.Skill, 0, len(rows))
	for _, row := range rows {
		skill, err := skillFromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, skill)
	}
	return out, nil
}

func (r *GormRepository) ListSkillSummaries(ctx context.Context, accountID string) ([]*domain.Skill, error) {
	var rows []SkillRow
	if err := r.db.WithContext(ctx).Select("id", "name", "description", "version", "enabled", "created_at", "updated_at").
		Where("account_id = ?", accountID).Order("created_at ASC, id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*domain.Skill, 0, len(rows))
	for _, row := range rows {
		out = append(out, &domain.Skill{
			ID: row.ID, Name: row.Name, Description: row.Description,
			Version: row.Version, Enabled: row.Enabled, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
		})
	}
	return out, nil
}

func (r *GormRepository) ListSkillVersions(ctx context.Context, accountID, skillID string) ([]*domain.SkillVersion, error) {
	var skill SkillRow
	if err := r.db.WithContext(ctx).Select("id").Where("account_id = ? AND id = ?", accountID, skillID).First(&skill).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	var rows []SkillVersionRow
	if err := r.db.WithContext(ctx).Select("skill_id", "account_id", "version", "created_at").
		Where("account_id = ? AND skill_id = ?", accountID, skillID).Order("created_at DESC, version DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*domain.SkillVersion, 0, len(rows))
	for _, row := range rows {
		out = append(out, &domain.SkillVersion{SkillID: row.SkillID, AccountID: row.AccountID, Version: row.Version, CreatedAt: row.CreatedAt})
	}
	return out, nil
}

func (r *GormRepository) GetSkillVersion(ctx context.Context, accountID, skillID, version string) (*domain.SkillVersion, error) {
	var row SkillVersionRow
	if err := r.db.WithContext(ctx).Where("account_id = ? AND skill_id = ? AND version = ?", accountID, skillID, version).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	result := &domain.SkillVersion{
		SkillID: row.SkillID, AccountID: row.AccountID, Version: row.Version,
		Name: row.Name, Description: row.Description, Prompt: row.Prompt, CreatedAt: row.CreatedAt,
	}
	if len(row.FilesJSON) > 0 {
		if err := json.Unmarshal(row.FilesJSON, &result.Files); err != nil {
			return nil, err
		}
	}
	return result, nil
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
	return resultError(r.db.WithContext(ctx).Model(&MCPConnectorRow{}).
		Where("account_id = ? AND id = ?", connector.AccountID, connector.ID).
		Select("name", "url", "credential_cipher", "enabled", "policy", "discovered_tools", "updated_at").
		Updates(row))
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

func skillToRow(skill *domain.Skill) (*SkillRow, error) {
	files, err := json.Marshal(skill.Files)
	if err != nil {
		return nil, err
	}
	return &SkillRow{ID: skill.ID, AccountID: skill.AccountID, Name: skill.Name, Description: skill.Description, Prompt: skill.Prompt, FilesJSON: files, Version: skill.Version, Enabled: skill.Enabled, CreatedAt: skill.CreatedAt, UpdatedAt: skill.UpdatedAt}, nil
}
func skillFromRow(row SkillRow) (*domain.Skill, error) {
	skill := &domain.Skill{ID: row.ID, AccountID: row.AccountID, Name: row.Name, Description: row.Description, Prompt: row.Prompt, Version: row.Version, Enabled: row.Enabled, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
	if len(row.FilesJSON) > 0 {
		if err := json.Unmarshal(row.FilesJSON, &skill.Files); err != nil {
			return nil, err
		}
	}
	return skill, nil
}

func skillVersionToRow(skill SkillRow, createdAt time.Time) *SkillVersionRow {
	return &SkillVersionRow{
		SkillID: skill.ID, AccountID: skill.AccountID, Version: skill.Version,
		Name: skill.Name, Description: skill.Description, Prompt: skill.Prompt,
		FilesJSON: skill.FilesJSON, CreatedAt: createdAt,
	}
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
