package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

type SessionRow struct {
	ID                             string `gorm:"primaryKey;size:64"`
	AccountID                      string `gorm:"size:64;not null;index:idx_studio_sessions_account_updated"`
	ProjectID                      string `gorm:"size:64;not null;default:'';index:idx_studio_sessions_project_updated"`
	Title                          string `gorm:"size:256;not null"`
	PermissionMode                 string `gorm:"size:32;not null"`
	ModelConfigID                  string `gorm:"size:64;index"`
	ContextSummary                 string `gorm:"type:text"`
	ContextSummaryThroughMessageID string `gorm:"size:64;index"`
	Status                         string `gorm:"size:32;not null"`
	CreatedAt                      time.Time
	UpdatedAt                      time.Time `gorm:"index:idx_studio_sessions_account_updated;index:idx_studio_sessions_project_updated"`
}

func (SessionRow) TableName() string { return "studio_sessions" }

type ProjectRow struct {
	ID        string `gorm:"primaryKey;size:64"`
	AccountID string `gorm:"size:64;not null;uniqueIndex:idx_studio_projects_account_name"`
	Name      string `gorm:"size:256;not null;uniqueIndex:idx_studio_projects_account_name"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (ProjectRow) TableName() string { return "studio_projects" }

type MessageRow struct {
	ID          string    `gorm:"primaryKey;size:64"`
	SessionID   string    `gorm:"size:64;not null;index:idx_studio_messages_session_created"`
	AccountID   string    `gorm:"size:64;not null;index"`
	RunID       string    `gorm:"size:64;index"`
	Role        string    `gorm:"size:24;not null"`
	ContentJSON []byte    `gorm:"type:blob;not null"`
	CreatedAt   time.Time `gorm:"index:idx_studio_messages_session_created"`
}

func (MessageRow) TableName() string { return "studio_messages" }

type RunRow struct {
	ID                string    `gorm:"primaryKey;size:64;index:idx_studio_runs_trace_page,priority:4"`
	SessionID         string    `gorm:"size:64;not null;index;uniqueIndex:idx_studio_runs_request;index:idx_studio_runs_trace_page,priority:2"`
	AccountID         string    `gorm:"size:64;not null;index;uniqueIndex:idx_studio_runs_request;index:idx_studio_runs_trace_page,priority:1"`
	RequestID         *string   `gorm:"size:128;uniqueIndex:idx_studio_runs_request"`
	TriggerMessageID  string    `gorm:"size:64;not null;index"`
	LastEventSequence uint64    `gorm:"not null;default:0"`
	Status            string    `gorm:"size:32;not null;index"`
	ModelConfigID     string    `gorm:"size:64;index"`
	Locale            string    `gorm:"size:8;not null;default:'zh'"`
	SkillIDsJSON      []byte    `gorm:"type:blob"`
	SkillSnapshotJSON []byte    `gorm:"type:blob"`
	AssetIDsJSON      []byte    `gorm:"type:blob"`
	ErrorCode         string    `gorm:"size:128"`
	ErrorMessage      string    `gorm:"type:text"`
	CreatedAt         time.Time `gorm:"index:idx_studio_runs_trace_page,priority:3"`
	StartedAt         time.Time
	CompletedAt       time.Time
	UpdatedAt         time.Time
}

func (RunRow) TableName() string { return "studio_runs" }

type RunProgressRow struct {
	RunID              string `gorm:"primaryKey;size:64"`
	SessionID          string `gorm:"size:64;not null;index"`
	AccountID          string `gorm:"size:64;not null;index"`
	AssistantMessageID string `gorm:"size:64"`
	AssistantText      string `gorm:"type:text"`
	ReasoningText      string `gorm:"type:text"`
	ToolCallsJSON      []byte `gorm:"type:blob"`
	LastSequence       uint64 `gorm:"not null"`
	UpdatedAt          time.Time
}

func (RunProgressRow) TableName() string { return "studio_run_progress" }

type CheckpointRow struct {
	RunID     string `gorm:"primaryKey;size:64"`
	Data      []byte `gorm:"type:blob;not null"`
	UpdatedAt time.Time
}

func (CheckpointRow) TableName() string { return "studio_run_checkpoints" }

type EventRow struct {
	ID             string `gorm:"primaryKey;size:64"`
	RunID          string `gorm:"size:64;not null;uniqueIndex:idx_studio_events_run_sequence;index:idx_studio_events_run_sequence_order"`
	SessionID      string `gorm:"size:64;not null;index;index:idx_studio_events_context,priority:2"`
	AccountID      string `gorm:"size:64;not null;index;index:idx_studio_events_context,priority:1"`
	Sequence       uint64 `gorm:"not null;uniqueIndex:idx_studio_events_run_sequence;index:idx_studio_events_run_sequence_order"`
	Type           string `gorm:"size:96;not null;index:idx_studio_events_context,priority:3"`
	Payload        []byte `gorm:"type:blob;not null"`
	SummaryPayload []byte `gorm:"type:blob"`
	CreatedAt      time.Time
}

func (EventRow) TableName() string { return "studio_events" }

type ApprovalRow struct {
	ID          string `gorm:"primaryKey;size:64"`
	RunID       string `gorm:"size:64;not null;index;uniqueIndex:idx_studio_approvals_run_tool"`
	SessionID   string `gorm:"size:64;not null;index"`
	AccountID   string `gorm:"size:64;not null;index"`
	ToolCallID  string `gorm:"size:128;not null;uniqueIndex:idx_studio_approvals_run_tool"`
	Action      string `gorm:"size:128;not null"`
	Description string `gorm:"size:256"`
	Status      string `gorm:"size:32;not null;index"`
	ResolvedBy  string `gorm:"size:64"`
	CreatedAt   time.Time
	ResolvedAt  time.Time
	UpdatedAt   time.Time
}

func (ApprovalRow) TableName() string { return "studio_approvals" }

type ClarificationRow struct {
	ID           string `gorm:"primaryKey;size:64"`
	RunID        string `gorm:"size:64;not null;index"`
	SessionID    string `gorm:"size:64;not null;index"`
	AccountID    string `gorm:"size:64;not null;index"`
	Question     string `gorm:"type:text;not null"`
	OptionsJSON  []byte `gorm:"type:blob;not null"`
	WorkflowJSON []byte `gorm:"type:blob"`
	Status       string `gorm:"size:32;not null;index"`
	Selected     string `gorm:"size:32"`
	Answer       string `gorm:"type:text"`
	ResolvedBy   string `gorm:"size:64"`
	CreatedAt    time.Time
	ResolvedAt   time.Time
	UpdatedAt    time.Time
}

func (ClarificationRow) TableName() string { return "studio_clarifications" }

type WorkflowExecutionRow struct {
	ID              string             `gorm:"primaryKey;size:64"`
	AccountID       string             `gorm:"size:64;not null;index"`
	SessionID       string             `gorm:"size:64;not null;index"`
	RunID           string             `gorm:"size:64;not null;uniqueIndex:idx_studio_workflow_executions_run_tool;index"`
	ToolCallID      string             `gorm:"size:128;not null;uniqueIndex:idx_studio_workflow_executions_run_tool"`
	TaskID          string             `gorm:"size:128;not null;uniqueIndex;index"`
	WorkflowID      string             `gorm:"size:64;not null;index"`
	OperationNodeID string             `gorm:"size:64;not null;index"`
	InputFields     []domain.FlowPort  `gorm:"serializer:json;type:text"`
	OutputFields    []domain.FlowPort  `gorm:"serializer:json;type:text"`
	Inputs          []domain.FlowInput `gorm:"serializer:json;type:text"`
	Status          string             `gorm:"size:32;not null;index"`
	ErrorMessage    string             `gorm:"type:text"`
	CreatedAt       time.Time          `gorm:"index"`
	UpdatedAt       time.Time          `gorm:"index"`
	CompletedAt     time.Time
}

func (WorkflowExecutionRow) TableName() string { return "studio_workflow_executions" }

type AssetRow struct {
	ID             string  `gorm:"primaryKey;size:64"`
	CreationKey    *string `gorm:"size:128;uniqueIndex:idx_studio_asset_creation_key"`
	AccountID      string  `gorm:"size:64;not null;index;uniqueIndex:idx_studio_asset_creation_key"`
	Name           string  `gorm:"size:512;not null"`
	Kind           string  `gorm:"size:32;not null;index"`
	Origin         string  `gorm:"size:32;not null;index"`
	CurrentVersion int     `gorm:"not null"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (AssetRow) TableName() string { return "studio_assets" }

type AssetVersionRow struct {
	ID               string `gorm:"primaryKey;size:64"`
	AssetID          string `gorm:"size:64;not null;uniqueIndex:idx_studio_asset_versions_asset_version;uniqueIndex:idx_studio_asset_versions_operation;index"`
	AccountID        string `gorm:"size:64;not null;index"`
	Version          int    `gorm:"not null;uniqueIndex:idx_studio_asset_versions_asset_version"`
	MIMEType         string `gorm:"size:256;not null"`
	Format           string `gorm:"size:32;not null;index"`
	WidthPx          *int
	HeightPx         *int
	SHA256           string  `gorm:"size:64;index"`
	ContentOrigin    string  `gorm:"size:32"`
	OperationKey     *string `gorm:"size:128;uniqueIndex:idx_studio_asset_versions_operation"`
	SourceCreatedAt  *time.Time
	SourceModifiedAt *time.Time
	BlobKey          string `gorm:"size:1024;not null"`
	SizeBytes        int64  `gorm:"not null"`
	Metadata         []byte `gorm:"type:blob"`
	CreatedAt        time.Time
}

func (AssetVersionRow) TableName() string { return "studio_asset_versions" }

type FlowNodeRow struct {
	ID             string `gorm:"primaryKey;size:64"`
	SessionID      string `gorm:"size:64;not null;index:idx_studio_flow_nodes_session_order"`
	AccountID      string `gorm:"size:64;not null;index"`
	Type           string `gorm:"size:32;not null"`
	Title          string `gorm:"size:512;not null"`
	Body           string `gorm:"type:text"`
	AssetID        string `gorm:"size:64;index"`
	AssetVersionID string `gorm:"size:64;index"`
	AssetVersion   int
	RunID          string              `gorm:"size:64;index"`
	Outputs        []domain.FlowOutput `gorm:"serializer:json;type:text"`
	PositionX      float64
	PositionY      float64
	SortOrder      int `gorm:"not null;index:idx_studio_flow_nodes_session_order"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (FlowNodeRow) TableName() string { return "studio_flow_nodes" }

type FlowEdgeRow struct {
	ID              string `gorm:"primaryKey;size:64"`
	SessionID       string `gorm:"size:64;not null;index"`
	AccountID       string `gorm:"size:64;not null;index"`
	SourceNodeID    string `gorm:"size:64;not null;index"`
	TargetNodeID    string `gorm:"size:64;not null;index"`
	SourceOutputKey string `gorm:"size:128"`
	TargetInputKey  string `gorm:"size:128"`
	Label           string `gorm:"size:256"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (FlowEdgeRow) TableName() string { return "studio_flow_edges" }

func Models() []any {
	return []any{
		&ProjectRow{}, &SessionRow{}, &MessageRow{}, &RunRow{}, &RunProgressRow{}, &CheckpointRow{}, &EventRow{}, &ContextRequestRow{}, &ContextEventRow{}, &ApprovalRow{}, &ClarificationRow{},
		&WorkflowExecutionRow{},
		&AssetRow{}, &AssetVersionRow{},
		&ProjectAssetRow{}, &SessionAssetUsageRow{}, &AssetCategoryRow{}, &AssetTagRow{}, &ProjectAssetTagRow{}, &AssetVersionPaletteRow{}, &AssetLibraryPreferencesRow{}, &BlobWriteIntentRow{},
		&FlowNodeRow{}, &FlowEdgeRow{}, &ModelConfigRow{},
		&SkillRow{}, &SkillVersionRow{}, &MCPConnectorRow{}, &AgentWorkflowSettingRow{},
	}
}

type GormRepository struct {
	db *gorm.DB
}

func NewGormRepository(db *gorm.DB) *GormRepository { return &GormRepository{db: db} }

func (r *GormRepository) CreateProject(ctx context.Context, project *domain.Project) error {
	if project == nil {
		return fmt.Errorf("%w: nil project", domain.ErrInvalid)
	}
	return translateCreateError(r.db.WithContext(ctx).Create(&ProjectRow{ID: project.ID, AccountID: project.AccountID, Name: project.Name, CreatedAt: project.CreatedAt, UpdatedAt: project.UpdatedAt}).Error)
}

func (r *GormRepository) GetProject(ctx context.Context, accountID, projectID string) (*domain.Project, error) {
	var row ProjectRow
	err := r.db.WithContext(ctx).Where("account_id = ? AND id = ?", accountID, projectID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &domain.Project{ID: row.ID, AccountID: row.AccountID, Name: row.Name, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}, nil
}

func (r *GormRepository) ListProjects(ctx context.Context, accountID string) ([]*domain.Project, error) {
	var rows []ProjectRow
	if err := r.db.WithContext(ctx).Where("account_id = ?", accountID).Order("updated_at DESC, id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	projects := make([]*domain.Project, 0, len(rows))
	for _, row := range rows {
		projects = append(projects, &domain.Project{ID: row.ID, AccountID: row.AccountID, Name: row.Name, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt})
	}
	return projects, nil
}

func (r *GormRepository) RenameProject(ctx context.Context, accountID, projectID, name string, now time.Time) error {
	name = strings.TrimSpace(name)
	if name == "" || len([]rune(name)) > 80 {
		return fmt.Errorf("%w: invalid project name", domain.ErrInvalid)
	}
	result := r.db.WithContext(ctx).Model(&ProjectRow{}).Where("account_id = ? AND id = ?", accountID, projectID).Updates(map[string]any{"name": name, "updated_at": now.UTC()})
	if result.Error != nil {
		return translateCreateError(result.Error)
	}
	return resultError(result)
}

func (r *GormRepository) DeleteProject(ctx context.Context, accountID, projectID string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var project ProjectRow
		if err := tx.Where("account_id = ? AND id = ?", accountID, projectID).First(&project).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrNotFound
			}
			return err
		}
		var entries []ProjectAssetRow
		if err := tx.Where("account_id = ? AND project_id = ?", accountID, projectID).Find(&entries).Error; err != nil {
			return err
		}
		var categories []AssetCategoryRow
		if err := tx.Where("account_id = ? AND project_id = ?", accountID, projectID).Find(&categories).Error; err != nil {
			return err
		}
		if len(categories) > 0 {
			rootName := project.Name
			for suffix := 2; ; suffix++ {
				var existing AssetCategoryRow
				err := tx.Where("account_id = ? AND project_id = '' AND parent_id = '' AND name = ?", accountID, rootName).First(&existing).Error
				if errors.Is(err, gorm.ErrRecordNotFound) {
					break
				}
				if err != nil {
					return err
				}
				rootName = fmt.Sprintf("%s (%d)", project.Name, suffix)
			}
			now := time.Now().UTC()
			root := &AssetCategoryRow{ID: uuid.NewString(), AccountID: accountID, ProjectID: "", ParentID: "", Name: rootName, CreatedAt: now, UpdatedAt: now}
			if err := tx.Create(root).Error; err != nil {
				return translateCreateError(err)
			}
			if err := tx.Model(&AssetCategoryRow{}).Where("account_id = ? AND project_id = ? AND parent_id = ''", accountID, projectID).
				Update("parent_id", root.ID).Error; err != nil {
				return err
			}
			if err := tx.Model(&AssetCategoryRow{}).Where("account_id = ? AND project_id = ?", accountID, projectID).
				Update("project_id", "").Error; err != nil {
				return err
			}
		}
		for _, entry := range entries {
			var duplicate ProjectAssetRow
			err := tx.Where("account_id = ? AND project_id = '' AND asset_id = ?", accountID, entry.AssetID).First(&duplicate).Error
			if err == nil {
				if err := tx.Where("account_id = ? AND project_asset_id = ?", accountID, entry.ID).Delete(&ProjectAssetTagRow{}).Error; err != nil {
					return err
				}
				if err := tx.Where("account_id = ? AND id = ?", accountID, entry.ID).Delete(&ProjectAssetRow{}).Error; err != nil {
					return err
				}
			} else if errors.Is(err, gorm.ErrRecordNotFound) {
				if err := tx.Model(&ProjectAssetRow{}).Where("account_id = ? AND id = ?", accountID, entry.ID).
					Update("project_id", "").Error; err != nil {
					return err
				}
			} else {
				return err
			}
		}
		if err := tx.Model(&SessionRow{}).Where("account_id = ? AND project_id = ?", accountID, projectID).Update("project_id", "").Error; err != nil {
			return err
		}
		if err := tx.Model(&AssetLibraryPreferencesRow{}).Where("account_id = ? AND last_project_id = ?", accountID, projectID).Update("last_project_id", "").Error; err != nil {
			return err
		}
		return resultError(tx.Where("account_id = ? AND id = ?", accountID, projectID).Delete(&ProjectRow{}))
	})
}

func (r *GormRepository) MoveSessionToProject(ctx context.Context, accountID, sessionID, projectID string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if projectID != "" {
			var project ProjectRow
			if err := tx.Where("account_id = ? AND id = ?", accountID, projectID).First(&project).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return domain.ErrNotFound
				}
				return err
			}
		}
		var session SessionRow
		if err := tx.Where("account_id = ? AND id = ?", accountID, sessionID).First(&session).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrNotFound
			}
			return err
		}
		if session.ProjectID == projectID {
			return nil
		}
		sourceProjectName := "未归属项目"
		if session.ProjectID != "" {
			var sourceProject ProjectRow
			if err := tx.Where("account_id = ? AND id = ?", accountID, session.ProjectID).First(&sourceProject).Error; err != nil {
				return err
			}
			sourceProjectName = sourceProject.Name
		}
		var usages []SessionAssetUsageRow
		if err := tx.Where("account_id = ? AND session_id = ?", accountID, sessionID).
			Order("created_at DESC, id DESC").Find(&usages).Error; err != nil {
			return err
		}
		seen := make(map[string]bool, len(usages))
		for _, usage := range usages {
			if seen[usage.AssetID] {
				continue
			}
			seen[usage.AssetID] = true
			var existing ProjectAssetRow
			err := tx.Where("account_id = ? AND project_id = ? AND asset_id = ?", accountID, projectID, usage.AssetID).First(&existing).Error
			if err == nil {
				continue
			}
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			var asset AssetRow
			if err := tx.Where("account_id = ? AND id = ?", accountID, usage.AssetID).First(&asset).Error; err != nil {
				return err
			}
			now := time.Now().UTC()
			entry := ProjectAssetRow{ID: uuid.NewString(), AccountID: accountID, ProjectID: projectID, AssetID: usage.AssetID, AssetVersionID: usage.AssetVersionID, DisplayName: asset.Name, AddedAt: now, UpdatedAt: now}
			var source ProjectAssetRow
			if err := tx.Where("account_id = ? AND project_id = ? AND asset_id = ?", accountID, session.ProjectID, usage.AssetID).First(&source).Error; err == nil {
				entry.AssetVersionID = source.AssetVersionID
				entry.DisplayName = source.DisplayName
				entry.SourceProjectAssetID = source.ID
				entry.SourceProjectID = source.ProjectID
				entry.SourceProjectNameSnapshot = sourceProjectName
				entry.SourceDisplayNameSnapshot = source.DisplayName
				entry.SourceAssetVersionID = source.AssetVersionID
				entry.CopiedAt = &now
			} else if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			if err := tx.Create(&entry).Error; err != nil {
				return translateCreateError(err)
			}
		}
		return tx.Model(&SessionRow{}).Where("account_id = ? AND id = ?", accountID, sessionID).Update("project_id", projectID).Error
	})
}

func (r *GormRepository) CreateSession(ctx context.Context, session *domain.Session) error {
	if session == nil {
		return fmt.Errorf("%w: nil session", domain.ErrInvalid)
	}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(sessionToRow(session)).Error; err != nil {
			return err
		}
		if session.ModelConfigID != "" {
			return appendSessionContextChange(tx, session, "MODEL_SWITCHED", "", session.ModelConfigID)
		}
		return nil
	})
	return translateCreateError(err)
}

func (r *GormRepository) UpdateSession(ctx context.Context, session *domain.Session) error {
	if session == nil {
		return fmt.Errorf("%w: nil session", domain.ErrInvalid)
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var previous SessionRow
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND account_id = ?", session.ID, session.AccountID).First(&previous).Error; err != nil {
			return err
		}
		result := tx.Model(&SessionRow{}).Where("id = ? AND account_id = ?", session.ID, session.AccountID).Updates(sessionToRow(session))
		if err := resultError(result); err != nil {
			return err
		}
		if previous.ModelConfigID != session.ModelConfigID {
			if err := appendSessionContextChange(tx, session, "MODEL_SWITCHED", previous.ModelConfigID, session.ModelConfigID); err != nil {
				return err
			}
		}
		if previous.PermissionMode != string(session.PermissionMode) {
			return appendSessionContextChange(tx, session, "MODE_SWITCHED", previous.PermissionMode, string(session.PermissionMode))
		}
		return nil
	})
}

func (r *GormRepository) GetSession(ctx context.Context, accountID, sessionID string) (*domain.Session, error) {
	var row SessionRow
	err := r.db.WithContext(ctx).Where("account_id = ? AND id = ?", accountID, sessionID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return sessionFromRow(row), nil
}

func (r *GormRepository) ListSessions(ctx context.Context, accountID string, query domain.SessionListQuery) ([]*domain.Session, error) {
	limit := normalizeLimit(query.Limit)
	var rows []SessionRow
	statement := r.db.WithContext(ctx).Where("account_id = ?", accountID)
	if query.ProjectID != nil {
		statement = statement.Where("project_id = ?", *query.ProjectID)
	}
	err := statement.Order("updated_at DESC, id DESC").Limit(limit).Offset(query.Offset).Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]*domain.Session, 0, len(rows))
	for _, row := range rows {
		out = append(out, sessionFromRow(row))
	}
	return out, nil
}

func (r *GormRepository) ClearSessions(ctx context.Context, accountID string) error {
	if strings.TrimSpace(accountID) == "" {
		return fmt.Errorf("%w: account is required", domain.ErrInvalid)
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&SessionRow{}).Where("account_id = ?", accountID).
			UpdateColumn("updated_at", gorm.Expr("updated_at")).Error; err != nil {
			return err
		}
		runIDs := tx.Model(&RunRow{}).Select("id").Where("account_id = ?", accountID)
		if err := tx.Where("run_id IN (?)", runIDs).Delete(&CheckpointRow{}).Error; err != nil {
			return err
		}
		if err := tx.Where("account_id = ?", accountID).Delete(&RunProgressRow{}).Error; err != nil {
			return err
		}
		if err := tx.Where("account_id = ?", accountID).Delete(&EventRow{}).Error; err != nil {
			return err
		}
		if err := tx.Where("account_id = ?", accountID).Delete(&ContextRequestRow{}).Error; err != nil {
			return err
		}
		if err := tx.Where("account_id = ?", accountID).Delete(&ContextEventRow{}).Error; err != nil {
			return err
		}
		if err := tx.Where("account_id = ?", accountID).Delete(&ApprovalRow{}).Error; err != nil {
			return err
		}
		if err := tx.Where("account_id = ?", accountID).Delete(&ClarificationRow{}).Error; err != nil {
			return err
		}
		if err := tx.Where("account_id = ?", accountID).Delete(&WorkflowExecutionRow{}).Error; err != nil {
			return err
		}
		if err := tx.Where("account_id = ?", accountID).Delete(&FlowEdgeRow{}).Error; err != nil {
			return err
		}
		if err := tx.Where("account_id = ?", accountID).Delete(&FlowNodeRow{}).Error; err != nil {
			return err
		}
		if err := tx.Where("account_id = ?", accountID).Delete(&MessageRow{}).Error; err != nil {
			return err
		}
		if err := tx.Where("account_id = ?", accountID).Delete(&RunRow{}).Error; err != nil {
			return err
		}
		return tx.Where("account_id = ?", accountID).Delete(&SessionRow{}).Error
	})
}

func (r *GormRepository) AppendMessage(ctx context.Context, message *domain.Message) error {
	if message == nil {
		return fmt.Errorf("%w: nil message", domain.ErrInvalid)
	}
	return translateCreateError(r.db.WithContext(ctx).Create(messageToRow(message)).Error)
}

func (r *GormRepository) GetMessage(ctx context.Context, accountID, messageID string) (*domain.Message, error) {
	var row MessageRow
	err := r.db.WithContext(ctx).Where("account_id = ? AND id = ?", accountID, messageID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return messageFromRow(row), nil
}

func (r *GormRepository) ListMessages(ctx context.Context, accountID, sessionID string, limit int) ([]*domain.Message, error) {
	var rows []MessageRow
	err := r.db.WithContext(ctx).Where("account_id = ? AND session_id = ?", accountID, sessionID).
		Order("created_at ASC, id ASC").Limit(normalizeLimit(limit)).Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]*domain.Message, 0, len(rows))
	for _, row := range rows {
		out = append(out, messageFromRow(row))
	}
	return out, nil
}

func (r *GormRepository) ListSessionTranscript(ctx context.Context, accountID, sessionID string) (*domain.SessionTranscriptData, error) {
	var messageRows []MessageRow
	if err := r.db.WithContext(ctx).
		Where("account_id = ? AND session_id = ?", accountID, sessionID).
		Order("created_at ASC, id ASC").Find(&messageRows).Error; err != nil {
		return nil, err
	}
	var runRows []RunRow
	if err := r.db.WithContext(ctx).
		Where("account_id = ? AND session_id = ?", accountID, sessionID).
		Order("created_at ASC, id ASC").Find(&runRows).Error; err != nil {
		return nil, err
	}
	var eventRows []EventRow
	if err := r.db.WithContext(ctx).
		Where("account_id = ? AND session_id = ?", accountID, sessionID).
		Not("type IN ?", []string{"MODEL_REQUEST_STARTED", "MODEL_FIRST_TOKEN", "MODEL_REQUEST_FINISHED", "MODEL_REQUEST_FAILED", "CONTEXT_COMPACTED"}).
		Order("created_at ASC, id ASC").Find(&eventRows).Error; err != nil {
		return nil, err
	}
	data := &domain.SessionTranscriptData{
		Messages: make([]*domain.Message, 0, len(messageRows)),
		Runs:     make([]*domain.Run, 0, len(runRows)),
		Events:   make([]*domain.Event, 0, len(eventRows)),
	}
	for _, row := range messageRows {
		data.Messages = append(data.Messages, messageFromRow(row))
	}
	for _, row := range runRows {
		data.Runs = append(data.Runs, runFromRow(row))
	}
	for _, row := range eventRows {
		data.Events = append(data.Events, eventFromRow(row))
	}
	return data, nil
}

func (r *GormRepository) CreateRun(ctx context.Context, run *domain.Run) error {
	if run == nil {
		return fmt.Errorf("%w: nil run", domain.ErrInvalid)
	}
	return translateCreateError(r.db.WithContext(ctx).Create(runToRow(run)).Error)
}

func (r *GormRepository) GetRunByRequestID(ctx context.Context, accountID, sessionID, requestID string) (*domain.Run, error) {
	if requestID == "" {
		return nil, domain.ErrNotFound
	}
	var row RunRow
	err := r.db.WithContext(ctx).Where("account_id = ? AND session_id = ? AND request_id = ?", accountID, sessionID, requestID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return runFromRow(row), nil
}

// CreateRunTurn is the only path that creates a user message and its Run. The
// session row serializes competing sends across processes before active-run
// and request-id checks, so retries cannot create duplicate user turns.
func (r *GormRepository) CreateRunTurn(ctx context.Context, message *domain.Message, run *domain.Run) (*domain.Run, bool, error) {
	if message == nil || run == nil || run.RequestID == "" ||
		message.RunID != run.ID || message.ID != run.TriggerMessageID ||
		message.AccountID != run.AccountID || message.SessionID != run.SessionID {
		return nil, false, fmt.Errorf("%w: invalid run turn", domain.ErrInvalid)
	}
	var stored *domain.Run
	created := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		locked := tx.Model(&SessionRow{}).
			Where("id = ? AND account_id = ?", run.SessionID, run.AccountID).
			UpdateColumn("updated_at", gorm.Expr("updated_at"))
		if err := resultError(locked); err != nil {
			return err
		}
		var existing RunRow
		err := tx.Where("account_id = ? AND session_id = ? AND request_id = ?", run.AccountID, run.SessionID, run.RequestID).First(&existing).Error
		if err == nil {
			stored = runFromRow(existing)
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		err = tx.Where("account_id = ? AND session_id = ? AND status IN ?", run.AccountID, run.SessionID,
			[]string{string(domain.RunQueued), string(domain.RunRunning), string(domain.RunWaitingApproval), string(domain.RunWaitingClarification)}).
			First(&existing).Error
		if err == nil {
			return fmt.Errorf("%w: session has active run %s", domain.ErrInvalidTransition, existing.ID)
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err := tx.Create(messageToRow(message)).Error; err != nil {
			return err
		}
		if err := tx.Create(runToRow(run)).Error; err != nil {
			return err
		}
		stored, created = run, true
		return nil
	})
	if err != nil {
		return nil, false, translateCreateError(err)
	}
	return stored, created, nil
}

func (r *GormRepository) UpdateRun(ctx context.Context, run *domain.Run) error {
	if run == nil {
		return fmt.Errorf("%w: nil run", domain.ErrInvalid)
	}
	result := r.db.WithContext(ctx).Model(&RunRow{}).
		Where("id = ? AND account_id = ?", run.ID, run.AccountID).Updates(runToRow(run))
	return resultError(result)
}

func (r *GormRepository) GetRun(ctx context.Context, accountID, runID string) (*domain.Run, error) {
	var row RunRow
	err := r.db.WithContext(ctx).Where("account_id = ? AND id = ?", accountID, runID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return runFromRow(row), nil
}

func (r *GormRepository) ListSessionRuns(ctx context.Context, accountID, sessionID string, limit int) ([]*domain.Run, error) {
	var rows []RunRow
	if err := r.db.WithContext(ctx).Where("account_id = ? AND session_id = ?", accountID, sessionID).
		Order("created_at DESC, id DESC").Limit(normalizeLimit(limit)).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*domain.Run, 0, len(rows))
	for _, row := range rows {
		out = append(out, runFromRow(row))
	}
	return out, nil
}

func (r *GormRepository) ListSessionTraceRuns(ctx context.Context, accountID, sessionID string, before time.Time, beforeID string, limit int) ([]*domain.Run, error) {
	if limit <= 0 || limit > 21 {
		limit = 21
	}
	query := r.db.WithContext(ctx).Where("account_id = ? AND session_id = ?", accountID, sessionID)
	if !before.IsZero() {
		query = query.Where("created_at < ? OR (created_at = ? AND id < ?)", before, before, beforeID)
	}
	var rows []RunRow
	if err := query.Order("created_at DESC, id DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	runs := make([]*domain.Run, 0, len(rows))
	for _, row := range rows {
		runs = append(runs, runFromRow(row))
	}
	return runs, nil
}

func (r *GormRepository) CountSessionTraceRuns(ctx context.Context, accountID, sessionID string) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&RunRow{}).Where("account_id = ? AND session_id = ?", accountID, sessionID).Count(&total).Error
	return total, err
}

func (r *GormRepository) ListRunTraceEvents(ctx context.Context, accountID, runID string) ([]*domain.Event, error) {
	var rows []EventRow
	if err := r.db.WithContext(ctx).Where("account_id = ? AND run_id = ?", accountID, runID).Order("sequence ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	events := make([]*domain.Event, 0, len(rows))
	for _, row := range rows {
		events = append(events, eventFromRow(row))
	}
	return events, nil
}

// ListRunTraceSummaryEvents returns only the bounded per-event read projection.
// Historical rows without that projection fall back to their source payload.
func (r *GormRepository) ListRunTraceSummaryEvents(ctx context.Context, accountID, runID string) ([]*domain.Event, error) {
	var rows []EventRow
	if err := r.db.WithContext(ctx).Model(&EventRow{}).
		Select("id, run_id, session_id, account_id, sequence, type, COALESCE(summary_payload, payload) AS payload, created_at").
		Where("account_id = ? AND run_id = ?", accountID, runID).
		Order("sequence ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	events := make([]*domain.Event, 0, len(rows))
	for _, row := range rows {
		events = append(events, eventFromRow(row))
	}
	return events, nil
}

func (r *GormRepository) ListLatestSessionRuns(ctx context.Context, accountID string, sessionIDs []string) (map[string]*domain.Run, error) {
	latest := make(map[string]*domain.Run)
	if len(sessionIDs) == 0 {
		return latest, nil
	}

	var rows []RunRow
	if err := r.db.WithContext(ctx).
		Where("account_id = ? AND session_id IN ?", accountID, sessionIDs).
		Order("session_id ASC, created_at DESC, id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		if _, exists := latest[row.SessionID]; exists {
			continue
		}
		latest[row.SessionID] = runFromRow(row)
	}
	return latest, nil
}

func (r *GormRepository) GetRunProgress(ctx context.Context, accountID, runID string) (*domain.RunProgress, error) {
	var row RunProgressRow
	err := r.db.WithContext(ctx).Where("account_id = ? AND run_id = ?", accountID, runID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return runProgressFromRow(row), nil
}

func (r *GormRepository) UpsertRunProgress(ctx context.Context, progress *domain.RunProgress) error {
	if progress == nil || progress.RunID == "" || progress.SessionID == "" || progress.AccountID == "" {
		return fmt.Errorf("%w: run progress ownership is required", domain.ErrInvalid)
	}
	row := runProgressToRow(progress)
	var existing RunProgressRow
	err := r.db.WithContext(ctx).Where("run_id = ? AND account_id = ?", progress.RunID, progress.AccountID).First(&existing).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return translateCreateError(r.db.WithContext(ctx).Create(row).Error)
	case err != nil:
		return err
	default:
		result := r.db.WithContext(ctx).Model(&RunProgressRow{}).
			Where("run_id = ? AND account_id = ?", progress.RunID, progress.AccountID).
			Updates(map[string]any{
				"session_id":           row.SessionID,
				"assistant_message_id": row.AssistantMessageID,
				"assistant_text":       row.AssistantText,
				"reasoning_text":       row.ReasoningText,
				"tool_calls_json":      row.ToolCallsJSON,
				"last_sequence":        row.LastSequence,
				"updated_at":           row.UpdatedAt,
			})
		return resultError(result)
	}
}

func (r *GormRepository) DeleteRunProgress(ctx context.Context, accountID, runID string) error {
	return r.db.WithContext(ctx).Where("account_id = ? AND run_id = ?", accountID, runID).Delete(&RunProgressRow{}).Error
}

func (r *GormRepository) ListRecoverableRuns(ctx context.Context, limit int) ([]*domain.Run, error) {
	var rows []RunRow
	err := r.db.WithContext(ctx).
		Where("status IN ?", []string{string(domain.RunQueued), string(domain.RunRunning)}).
		Order("created_at ASC, id ASC").Limit(normalizeLimit(limit)).Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]*domain.Run, 0, len(rows))
	for _, row := range rows {
		out = append(out, runFromRow(row))
	}
	return out, nil
}

func (r *GormRepository) ListRecoverableRunsAfter(ctx context.Context, afterID string, limit int) ([]*domain.Run, error) {
	var rows []RunRow
	err := r.db.WithContext(ctx).
		Where("status IN ? AND id > ?", []string{string(domain.RunQueued), string(domain.RunRunning)}, afterID).
		Order("id ASC").Limit(normalizeLimit(limit)).Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]*domain.Run, 0, len(rows))
	for _, row := range rows {
		out = append(out, runFromRow(row))
	}
	return out, nil
}

func (r *GormRepository) AppendEvent(ctx context.Context, event *domain.Event) error {
	if event == nil {
		return fmt.Errorf("%w: nil event", domain.ErrInvalid)
	}
	row := eventToRow(event)
	return r.db.WithContext(ctx).Create(row).Error
}

// AppendRunEvent allocates a sequence under the Run row lock before inserting
// the event. MAX(sequence) also accounts for events written before the cursor
// column existed, including runs that already contain more than 200 events.
func (r *GormRepository) AppendRunEvent(ctx context.Context, event *domain.Event) (*domain.Event, error) {
	if event == nil || event.RunID == "" || event.AccountID == "" || event.Sequence != 0 {
		return nil, fmt.Errorf("%w: run event requires an unnumbered owned run", domain.ErrInvalid)
	}
	stored := *event
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&RunRow{}).
			Where("id = ? AND account_id = ?", event.RunID, event.AccountID).
			UpdateColumn("last_event_sequence", gorm.Expr(
				"CASE WHEN last_event_sequence < (SELECT COALESCE(MAX(sequence), 0) FROM studio_events WHERE run_id = ?) THEN (SELECT COALESCE(MAX(sequence), 0) FROM studio_events WHERE run_id = ?) + 1 ELSE last_event_sequence + 1 END",
				event.RunID, event.RunID,
			))
		if err := resultError(result); err != nil {
			return err
		}
		var row RunRow
		if err := tx.Select("last_event_sequence").Where("id = ? AND account_id = ?", event.RunID, event.AccountID).First(&row).Error; err != nil {
			return err
		}
		stored.Sequence = row.LastEventSequence
		if err := tx.Create(eventToRow(&stored)).Error; err != nil {
			return err
		}
		return projectContextRequestEvent(tx, &stored)
	})
	if err != nil {
		return nil, err
	}
	return &stored, nil
}

func (r *GormRepository) LastRunEventSequence(ctx context.Context, accountID, runID string) (uint64, error) {
	var row RunRow
	if err := r.db.WithContext(ctx).Select("id").Where("id = ? AND account_id = ?", runID, accountID).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, domain.ErrNotFound
		}
		return 0, err
	}
	var sequence uint64
	if err := r.db.WithContext(ctx).Model(&EventRow{}).
		Where("run_id = ? AND account_id = ?", runID, accountID).
		Select("COALESCE(MAX(sequence), 0)").Scan(&sequence).Error; err != nil {
		return 0, err
	}
	return sequence, nil
}

func (r *GormRepository) ListEventsAfter(ctx context.Context, accountID, runID string, after uint64, limit int) ([]*domain.Event, error) {
	var rows []EventRow
	err := r.db.WithContext(ctx).
		Where("account_id = ? AND run_id = ? AND sequence > ?", accountID, runID, after).
		Order("sequence ASC").Limit(normalizeLimit(limit)).Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]*domain.Event, 0, len(rows))
	for _, row := range rows {
		out = append(out, eventFromRow(row))
	}
	return out, nil
}

func (r *GormRepository) CreateApproval(ctx context.Context, approval *domain.Approval) error {
	if approval == nil {
		return fmt.Errorf("%w: nil approval", domain.ErrInvalid)
	}
	return translateCreateError(r.db.WithContext(ctx).Create(approvalToRow(approval)).Error)
}

func (r *GormRepository) UpdateApproval(ctx context.Context, approval *domain.Approval) error {
	if approval == nil {
		return fmt.Errorf("%w: nil approval", domain.ErrInvalid)
	}
	result := r.db.WithContext(ctx).Model(&ApprovalRow{}).
		Where("id = ? AND account_id = ?", approval.ID, approval.AccountID).
		Updates(approvalToRow(approval))
	return resultError(result)
}

func (r *GormRepository) GetApproval(ctx context.Context, accountID, approvalID string) (*domain.Approval, error) {
	var row ApprovalRow
	err := r.db.WithContext(ctx).Where("account_id = ? AND id = ?", accountID, approvalID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return approvalFromRow(row), nil
}

func (r *GormRepository) ListApprovals(ctx context.Context, accountID, runID string) ([]*domain.Approval, error) {
	var rows []ApprovalRow
	if err := r.db.WithContext(ctx).Where("account_id = ? AND run_id = ?", accountID, runID).
		Order("created_at ASC, id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*domain.Approval, 0, len(rows))
	for _, row := range rows {
		out = append(out, approvalFromRow(row))
	}
	return out, nil
}

func (r *GormRepository) CreateClarification(ctx context.Context, clarification *domain.Clarification) error {
	if clarification == nil {
		return fmt.Errorf("%w: nil clarification", domain.ErrInvalid)
	}
	row, err := clarificationToRow(clarification)
	if err != nil {
		return err
	}
	return translateCreateError(r.db.WithContext(ctx).Create(row).Error)
}

func (r *GormRepository) UpdateClarification(ctx context.Context, clarification *domain.Clarification) error {
	if clarification == nil {
		return fmt.Errorf("%w: nil clarification", domain.ErrInvalid)
	}
	row, err := clarificationToRow(clarification)
	if err != nil {
		return err
	}
	result := r.db.WithContext(ctx).Model(&ClarificationRow{}).
		Where("id = ? AND account_id = ? AND status = ?", clarification.ID, clarification.AccountID, string(domain.ClarificationPending)).
		Updates(map[string]any{
			"status": row.Status, "selected": row.Selected, "answer": row.Answer,
			"workflow_json": row.WorkflowJSON,
			"resolved_by":   row.ResolvedBy, "resolved_at": row.ResolvedAt, "updated_at": row.UpdatedAt,
		})
	return resultError(result)
}

func (r *GormRepository) GetClarification(ctx context.Context, accountID, clarificationID string) (*domain.Clarification, error) {
	var row ClarificationRow
	err := r.db.WithContext(ctx).Where("account_id = ? AND id = ?", accountID, clarificationID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return clarificationFromRow(row)
}

func (r *GormRepository) ListClarifications(ctx context.Context, accountID, runID string) ([]*domain.Clarification, error) {
	var rows []ClarificationRow
	if err := r.db.WithContext(ctx).Where("account_id = ? AND run_id = ?", accountID, runID).
		Order("created_at ASC, id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*domain.Clarification, 0, len(rows))
	for _, row := range rows {
		clarification, err := clarificationFromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, clarification)
	}
	return out, nil
}

func (r *GormRepository) CreateWorkflowExecution(ctx context.Context, execution *domain.WorkflowExecution) error {
	if execution == nil {
		return fmt.Errorf("%w: nil workflow execution", domain.ErrInvalid)
	}
	return translateCreateError(r.db.WithContext(ctx).Create(workflowExecutionToRow(execution)).Error)
}

func (r *GormRepository) UpdateWorkflowExecution(ctx context.Context, execution *domain.WorkflowExecution) error {
	if execution == nil {
		return fmt.Errorf("%w: nil workflow execution", domain.ErrInvalid)
	}
	result := r.db.WithContext(ctx).Model(&WorkflowExecutionRow{}).
		Where("id = ? AND account_id = ?", execution.ID, execution.AccountID).
		Updates(workflowExecutionToRow(execution))
	return resultError(result)
}

func (r *GormRepository) GetWorkflowExecutionByTask(ctx context.Context, accountID, taskID string) (*domain.WorkflowExecution, error) {
	var row WorkflowExecutionRow
	err := r.db.WithContext(ctx).Where("account_id = ? AND task_id = ?", accountID, taskID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return workflowExecutionFromRow(row), nil
}

func (r *GormRepository) GetWorkflowExecutionByRunTool(ctx context.Context, accountID, runID, toolCallID string) (*domain.WorkflowExecution, error) {
	var row WorkflowExecutionRow
	err := r.db.WithContext(ctx).Where("account_id = ? AND run_id = ? AND tool_call_id = ?", accountID, runID, toolCallID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return workflowExecutionFromRow(row), nil
}

func (r *GormRepository) ListSessionWorkflowExecutions(ctx context.Context, accountID, sessionID string) ([]*domain.WorkflowExecution, error) {
	var rows []WorkflowExecutionRow
	if err := r.db.WithContext(ctx).Where("account_id = ? AND session_id = ?", accountID, sessionID).
		Order("created_at ASC, id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*domain.WorkflowExecution, 0, len(rows))
	for _, row := range rows {
		out = append(out, workflowExecutionFromRow(row))
	}
	return out, nil
}

func (r *GormRepository) ListActiveWorkflowCounts(ctx context.Context, accountID string, sessionIDs []string) (map[string]int, error) {
	counts := make(map[string]int)
	if len(sessionIDs) == 0 {
		return counts, nil
	}
	var rows []struct {
		SessionID string
		Count     int
	}
	if err := r.db.WithContext(ctx).Model(&WorkflowExecutionRow{}).Select("session_id, COUNT(*) AS count").Where("account_id = ? AND session_id IN ? AND status = ?", accountID, sessionIDs, domain.WorkflowExecutionSubmitted).Group("session_id").Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		counts[row.SessionID] = row.Count
	}
	return counts, nil
}

func (r *GormRepository) ListPendingWorkflowExecutions(ctx context.Context, limit int) ([]*domain.WorkflowExecution, error) {
	var rows []WorkflowExecutionRow
	if err := r.db.WithContext(ctx).Where("status = ?", string(domain.WorkflowExecutionSubmitted)).
		Order("created_at ASC, id ASC").Limit(normalizeLimit(limit)).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*domain.WorkflowExecution, 0, len(rows))
	for _, row := range rows {
		out = append(out, workflowExecutionFromRow(row))
	}
	return out, nil
}

func (r *GormRepository) CreateAsset(ctx context.Context, asset *domain.Asset) error {
	if asset == nil {
		return fmt.Errorf("%w: nil asset", domain.ErrInvalid)
	}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(assetToRow(asset)).Error; err != nil {
			return translateCreateError(err)
		}
		for _, version := range asset.Versions {
			if err := tx.Create(assetVersionToRow(version)).Error; err != nil {
				return translateCreateError(err)
			}
			if supportsPalette(version.MIMEType) {
				if err := tx.Create(newPendingPaletteRow(version)).Error; err != nil {
					return err
				}
			}
			if err := tx.Where("key_hash = ? AND account_id = ?", blobIntentKeyHash(asset.AccountID, version.BlobKey), asset.AccountID).Delete(&BlobWriteIntentRow{}).Error; err != nil {
				return err
			}
		}
		return nil
	})
	return err
}

func (r *GormRepository) AppendAssetVersion(ctx context.Context, assetID, accountID string, version domain.AssetVersion) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var asset AssetRow
		if err := tx.Where("id = ? AND account_id = ?", assetID, accountID).First(&asset).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrNotFound
			}
			return err
		}
		if version.AssetID != assetID || version.AccountID != accountID || version.Version != asset.CurrentVersion+1 {
			return fmt.Errorf("%w: asset version is not next", domain.ErrInvalid)
		}
		if err := tx.Create(assetVersionToRow(version)).Error; err != nil {
			return translateCreateError(err)
		}
		if supportsPalette(version.MIMEType) {
			if err := tx.Create(newPendingPaletteRow(version)).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("key_hash = ? AND account_id = ?", blobIntentKeyHash(accountID, version.BlobKey), accountID).Delete(&BlobWriteIntentRow{}).Error; err != nil {
			return err
		}
		return tx.Model(&AssetRow{}).Where("id = ? AND account_id = ?", assetID, accountID).
			Updates(map[string]any{"current_version": version.Version, "updated_at": version.CreatedAt}).Error
	})
}

func (r *GormRepository) GetAsset(ctx context.Context, accountID, assetID string) (*domain.Asset, error) {
	var row AssetRow
	err := r.db.WithContext(ctx).Where("account_id = ? AND id = ?", accountID, assetID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	asset := assetFromRow(row)
	if err := r.loadVersions(ctx, asset); err != nil {
		return nil, err
	}
	return asset, nil
}

func (r *GormRepository) GetAssetByCreationKey(ctx context.Context, accountID, creationKey string) (*domain.Asset, error) {
	if strings.TrimSpace(creationKey) == "" {
		return nil, fmt.Errorf("%w: empty creation key", domain.ErrInvalid)
	}
	var row AssetRow
	if err := r.db.WithContext(ctx).Where("account_id = ? AND creation_key = ?", accountID, creationKey).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return r.GetAsset(ctx, accountID, row.ID)
}

func (r *GormRepository) ListSessionAssets(ctx context.Context, accountID, sessionID string, limit int) ([]*domain.Asset, error) {
	var rows []AssetRow
	err := r.db.WithContext(ctx).Table("studio_assets AS a").Select("a.*").
		Joins("JOIN studio_session_asset_usages AS u ON u.asset_id = a.id AND u.account_id = a.account_id").
		Where("a.account_id = ? AND u.session_id = ?", accountID, sessionID).
		Group("a.id").Order("MIN(u.created_at) ASC, a.id ASC").
		Limit(normalizeLimit(limit)).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return r.assetsFromRows(ctx, rows)
}

func (r *GormRepository) SaveFlowNode(ctx context.Context, node *domain.FlowNode) error {
	if node == nil {
		return fmt.Errorf("%w: nil flow node", domain.ErrInvalid)
	}
	row := flowNodeToRow(node)
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"type", "title", "body", "asset_id", "asset_version_id", "asset_version", "run_id", "outputs", "position_x", "position_y", "sort_order", "updated_at",
		}),
	}).Create(row).Error
}

func (r *GormRepository) SaveFlowEdge(ctx context.Context, edge *domain.FlowEdge) error {
	if edge == nil {
		return fmt.Errorf("%w: nil flow edge", domain.ErrInvalid)
	}
	row := flowEdgeToRow(edge)
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"source_node_id", "target_node_id", "source_output_key", "target_input_key", "label", "updated_at"}),
	}).Create(row).Error
}

func (r *GormRepository) DeleteFlowNode(ctx context.Context, accountID, sessionID, nodeID string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Where("account_id = ? AND session_id = ? AND id = ?", accountID, sessionID, nodeID).Delete(&FlowNodeRow{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return domain.ErrNotFound
		}
		return tx.Where(
			"account_id = ? AND session_id = ? AND (source_node_id = ? OR target_node_id = ?)",
			accountID, sessionID, nodeID, nodeID,
		).Delete(&FlowEdgeRow{}).Error
	})
}

func (r *GormRepository) DeleteFlowEdge(ctx context.Context, accountID, sessionID, edgeID string) error {
	result := r.db.WithContext(ctx).Where(
		"account_id = ? AND session_id = ? AND id = ?", accountID, sessionID, edgeID,
	).Delete(&FlowEdgeRow{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *GormRepository) GetFlow(ctx context.Context, accountID, sessionID string) ([]*domain.FlowNode, []*domain.FlowEdge, error) {
	var nodeRows []FlowNodeRow
	if err := r.db.WithContext(ctx).Where("account_id = ? AND session_id = ?", accountID, sessionID).
		Order("sort_order ASC, created_at ASC, id ASC").Find(&nodeRows).Error; err != nil {
		return nil, nil, err
	}
	var edgeRows []FlowEdgeRow
	if err := r.db.WithContext(ctx).Where("account_id = ? AND session_id = ?", accountID, sessionID).
		Order("created_at ASC, id ASC").Find(&edgeRows).Error; err != nil {
		return nil, nil, err
	}
	nodes := make([]*domain.FlowNode, 0, len(nodeRows))
	for _, row := range nodeRows {
		nodes = append(nodes, flowNodeFromRow(row))
	}
	edges := make([]*domain.FlowEdge, 0, len(edgeRows))
	for _, row := range edgeRows {
		edges = append(edges, flowEdgeFromRow(row))
	}
	return nodes, edges, nil
}

func (r *GormRepository) loadVersions(ctx context.Context, asset *domain.Asset) error {
	var rows []AssetVersionRow
	if err := r.db.WithContext(ctx).Where("account_id = ? AND asset_id = ?", asset.AccountID, asset.ID).
		Order("version ASC").Find(&rows).Error; err != nil {
		return err
	}
	asset.Versions = make([]domain.AssetVersion, 0, len(rows))
	for _, row := range rows {
		asset.Versions = append(asset.Versions, assetVersionFromRow(row))
	}
	return nil
}

func (r *GormRepository) assetsFromRows(ctx context.Context, rows []AssetRow) ([]*domain.Asset, error) {
	out := make([]*domain.Asset, 0, len(rows))
	if len(rows) == 0 {
		return out, nil
	}
	assetIDs := make([]string, 0, len(rows))
	for _, row := range rows {
		assetIDs = append(assetIDs, row.ID)
		out = append(out, assetFromRow(row))
	}
	var versions []AssetVersionRow
	if err := r.db.WithContext(ctx).Where("asset_id IN ?", assetIDs).
		Order("asset_id ASC, version ASC").Find(&versions).Error; err != nil {
		return nil, err
	}
	assetsByID := make(map[string]*domain.Asset, len(out))
	for _, asset := range out {
		assetsByID[asset.ID] = asset
	}
	for _, version := range versions {
		assetsByID[version.AssetID].Versions = append(assetsByID[version.AssetID].Versions, assetVersionFromRow(version))
	}
	return out, nil
}

func normalizeLimit(limit int) int {
	if limit <= 0 {
		return 50
	}
	if limit > 200 {
		return 200
	}
	return limit
}

func resultError(result *gorm.DB) error {
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func translateCreateError(err error) error {
	if err == nil {
		return nil
	}
	message := strings.ToLower(err.Error())
	if errors.Is(err, gorm.ErrDuplicatedKey) ||
		strings.Contains(message, "unique constraint") ||
		strings.Contains(message, "duplicate entry") ||
		strings.Contains(message, "duplicate key") {
		return domain.ErrAlreadyExists
	}
	return err
}

func sessionToRow(value *domain.Session) *SessionRow {
	return &SessionRow{ID: value.ID, AccountID: value.AccountID, ProjectID: value.ProjectID, Title: value.Title, PermissionMode: string(value.PermissionMode), ModelConfigID: value.ModelConfigID, ContextSummary: value.ContextSummary, ContextSummaryThroughMessageID: value.ContextSummaryThroughMessageID, Status: string(value.Status), CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt}
}

func sessionFromRow(row SessionRow) *domain.Session {
	return &domain.Session{ID: row.ID, AccountID: row.AccountID, ProjectID: row.ProjectID, Title: row.Title, PermissionMode: domain.PermissionMode(row.PermissionMode), ModelConfigID: row.ModelConfigID, ContextSummary: row.ContextSummary, ContextSummaryThroughMessageID: row.ContextSummaryThroughMessageID, Status: domain.SessionStatus(row.Status), CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}

func messageToRow(value *domain.Message) *MessageRow {
	return &MessageRow{ID: value.ID, SessionID: value.SessionID, AccountID: value.AccountID, RunID: value.RunID, Role: string(value.Role), ContentJSON: append([]byte(nil), value.ContentJSON...), CreatedAt: value.CreatedAt}
}

func messageFromRow(row MessageRow) *domain.Message {
	return &domain.Message{ID: row.ID, SessionID: row.SessionID, AccountID: row.AccountID, RunID: row.RunID, Role: domain.MessageRole(row.Role), ContentJSON: append([]byte(nil), row.ContentJSON...), CreatedAt: row.CreatedAt}
}

func runToRow(value *domain.Run) *RunRow {
	skillIDs, _ := json.Marshal(value.SkillIDs)
	skillSnapshot, _ := json.Marshal(value.SkillSnapshot)
	assetIDs, _ := json.Marshal(value.AssetIDs)
	if len(value.AssetReferences) > 0 {
		assetIDs, _ = json.Marshal(value.AssetReferences)
	}
	var requestID *string
	if value.RequestID != "" {
		requestID = &value.RequestID
	}
	return &RunRow{ID: value.ID, SessionID: value.SessionID, AccountID: value.AccountID, RequestID: requestID, TriggerMessageID: value.TriggerMessageID, Status: string(value.Status), ModelConfigID: value.ModelConfigID, Locale: value.Locale, SkillIDsJSON: skillIDs, SkillSnapshotJSON: skillSnapshot, AssetIDsJSON: assetIDs, ErrorCode: value.ErrorCode, ErrorMessage: value.ErrorMessage, CreatedAt: value.CreatedAt, StartedAt: value.StartedAt, CompletedAt: value.CompletedAt, UpdatedAt: value.UpdatedAt}
}

func runFromRow(row RunRow) *domain.Run {
	var skillIDs []string
	_ = json.Unmarshal(row.SkillIDsJSON, &skillIDs)
	var skillSnapshot []domain.RunSkill
	_ = json.Unmarshal(row.SkillSnapshotJSON, &skillSnapshot)
	var assetIDs []string
	var assetReferences []domain.AssetReference
	if err := json.Unmarshal(row.AssetIDsJSON, &assetReferences); err == nil && len(assetReferences) > 0 {
		for _, reference := range assetReferences {
			assetIDs = append(assetIDs, reference.AssetID)
		}
	} else {
		_ = json.Unmarshal(row.AssetIDsJSON, &assetIDs)
	}
	requestID := ""
	if row.RequestID != nil {
		requestID = *row.RequestID
	}
	locale := row.Locale
	if locale == "" {
		locale = "zh"
	}
	return &domain.Run{ID: row.ID, SessionID: row.SessionID, AccountID: row.AccountID, RequestID: requestID, TriggerMessageID: row.TriggerMessageID, Status: domain.RunStatus(row.Status), ModelConfigID: row.ModelConfigID, Locale: locale, SkillIDs: skillIDs, SkillSnapshot: skillSnapshot, AssetIDs: assetIDs, AssetReferences: assetReferences, ErrorCode: row.ErrorCode, ErrorMessage: row.ErrorMessage, CreatedAt: row.CreatedAt, StartedAt: row.CompletedAt, UpdatedAt: row.UpdatedAt}
}

func runProgressToRow(value *domain.RunProgress) *RunProgressRow {
	return &RunProgressRow{
		RunID: value.RunID, SessionID: value.SessionID, AccountID: value.AccountID,
		AssistantMessageID: value.AssistantMessageID, AssistantText: value.AssistantText,
		ReasoningText: value.ReasoningText, ToolCallsJSON: append([]byte(nil), value.ToolCallsJSON...),
		LastSequence: value.LastSequence, UpdatedAt: value.UpdatedAt,
	}
}

func runProgressFromRow(row RunProgressRow) *domain.RunProgress {
	return &domain.RunProgress{
		RunID: row.RunID, SessionID: row.SessionID, AccountID: row.AccountID,
		AssistantMessageID: row.AssistantMessageID, AssistantText: row.AssistantText,
		ReasoningText: row.ReasoningText, ToolCallsJSON: append([]byte(nil), row.ToolCallsJSON...),
		LastSequence: row.LastSequence, UpdatedAt: row.UpdatedAt,
	}
}

func eventToRow(value *domain.Event) *EventRow {
	return &EventRow{ID: value.ID, RunID: value.RunID, SessionID: value.SessionID, AccountID: value.AccountID, Sequence: value.Sequence, Type: value.Type, Payload: append([]byte(nil), value.Payload...), SummaryPayload: trajectorySummaryPayload(value.Type, value.Payload), CreatedAt: value.CreatedAt}
}

func eventFromRow(row EventRow) *domain.Event {
	return &domain.Event{ID: row.ID, RunID: row.RunID, SessionID: row.SessionID, AccountID: row.AccountID, Sequence: row.Sequence, Type: row.Type, Payload: append([]byte(nil), row.Payload...), CreatedAt: row.CreatedAt}
}

func approvalToRow(value *domain.Approval) *ApprovalRow {
	return &ApprovalRow{ID: value.ID, RunID: value.RunID, SessionID: value.SessionID, AccountID: value.AccountID, ToolCallID: value.ToolCallID, Action: value.Action, Description: value.Description, Status: string(value.Status), ResolvedBy: value.ResolvedBy, CreatedAt: value.CreatedAt, ResolvedAt: value.ResolvedAt, UpdatedAt: value.UpdatedAt}
}

func approvalFromRow(row ApprovalRow) *domain.Approval {
	return &domain.Approval{ID: row.ID, RunID: row.RunID, SessionID: row.SessionID, AccountID: row.AccountID, ToolCallID: row.ToolCallID, Action: row.Action, Description: row.Description, Status: domain.ApprovalStatus(row.Status), ResolvedBy: row.ResolvedBy, CreatedAt: row.CreatedAt, ResolvedAt: row.ResolvedAt, UpdatedAt: row.UpdatedAt}
}

func clarificationToRow(value *domain.Clarification) (*ClarificationRow, error) {
	options, err := json.Marshal(value.Options)
	if err != nil {
		return nil, err
	}
	var workflow []byte
	if value.Workflow != nil {
		workflow, err = json.Marshal(value.Workflow)
		if err != nil {
			return nil, err
		}
	}
	return &ClarificationRow{
		ID: value.ID, RunID: value.RunID, SessionID: value.SessionID, AccountID: value.AccountID,
		Question: value.Question, OptionsJSON: options, WorkflowJSON: workflow, Status: string(value.Status),
		Selected: value.Selected, Answer: value.Answer, ResolvedBy: value.ResolvedBy,
		CreatedAt: value.CreatedAt, ResolvedAt: value.ResolvedAt, UpdatedAt: value.UpdatedAt,
	}, nil
}

func clarificationFromRow(row ClarificationRow) (*domain.Clarification, error) {
	var options []string
	if err := json.Unmarshal(row.OptionsJSON, &options); err != nil {
		return nil, fmt.Errorf("studio: decode clarification options: %w", err)
	}
	var workflow *domain.WorkflowRequest
	if len(row.WorkflowJSON) > 0 {
		if err := json.Unmarshal(row.WorkflowJSON, &workflow); err != nil {
			return nil, fmt.Errorf("studio: decode workflow request: %w", err)
		}
	}
	return &domain.Clarification{
		ID: row.ID, RunID: row.RunID, SessionID: row.SessionID, AccountID: row.AccountID,
		Question: row.Question, Options: options, Workflow: workflow, Status: domain.ClarificationStatus(row.Status),
		Selected: row.Selected, Answer: row.Answer, ResolvedBy: row.ResolvedBy,
		CreatedAt: row.CreatedAt, ResolvedAt: row.ResolvedAt, UpdatedAt: row.UpdatedAt,
	}, nil
}

func workflowExecutionToRow(value *domain.WorkflowExecution) *WorkflowExecutionRow {
	return &WorkflowExecutionRow{ID: value.ID, AccountID: value.AccountID, SessionID: value.SessionID, RunID: value.RunID, ToolCallID: value.ToolCallID, TaskID: value.TaskID, WorkflowID: value.WorkflowID, OperationNodeID: value.OperationNodeID, InputFields: value.InputFields, OutputFields: value.OutputFields, Inputs: value.Inputs, Status: string(value.Status), ErrorMessage: value.ErrorMessage, CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt, CompletedAt: value.CompletedAt}
}

func workflowExecutionFromRow(row WorkflowExecutionRow) *domain.WorkflowExecution {
	return &domain.WorkflowExecution{ID: row.ID, AccountID: row.AccountID, SessionID: row.SessionID, RunID: row.RunID, ToolCallID: row.ToolCallID, TaskID: row.TaskID, WorkflowID: row.WorkflowID, OperationNodeID: row.OperationNodeID, InputFields: row.InputFields, OutputFields: row.OutputFields, Inputs: row.Inputs, Status: domain.WorkflowExecutionStatus(row.Status), ErrorMessage: row.ErrorMessage, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, CompletedAt: row.CompletedAt}
}

func assetToRow(value *domain.Asset) *AssetRow {
	row := &AssetRow{ID: value.ID, AccountID: value.AccountID, Name: value.Name, Kind: string(value.Kind), Origin: string(value.Origin), CurrentVersion: value.CurrentVersion, CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt}
	if value.CreationKey != "" {
		row.CreationKey = &value.CreationKey
	}
	return row
}

func assetFromRow(row AssetRow) *domain.Asset {
	asset := &domain.Asset{ID: row.ID, AccountID: row.AccountID, Name: row.Name, Kind: domain.AssetKind(row.Kind), Origin: domain.AssetOrigin(row.Origin), CurrentVersion: row.CurrentVersion, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
	if row.CreationKey != nil {
		asset.CreationKey = *row.CreationKey
	}
	return asset
}

func assetVersionToRow(value domain.AssetVersion) *AssetVersionRow {
	format := value.Format
	if format == "" {
		format = "unknown"
	}
	row := &AssetVersionRow{ID: value.ID, AssetID: value.AssetID, AccountID: value.AccountID, Version: value.Version, MIMEType: value.MIMEType, Format: format, WidthPx: value.WidthPx, HeightPx: value.HeightPx, SHA256: value.SHA256, ContentOrigin: value.ContentOrigin, SourceCreatedAt: value.SourceCreatedAt, SourceModifiedAt: value.SourceModifiedAt, BlobKey: value.BlobKey, SizeBytes: value.SizeBytes, Metadata: append([]byte(nil), value.Metadata...), CreatedAt: value.CreatedAt}
	if value.OperationKey != "" {
		row.OperationKey = &value.OperationKey
	}
	return row
}

func assetVersionFromRow(row AssetVersionRow) domain.AssetVersion {
	version := domain.AssetVersion{ID: row.ID, AssetID: row.AssetID, AccountID: row.AccountID, Version: row.Version, MIMEType: row.MIMEType, Format: row.Format, WidthPx: row.WidthPx, HeightPx: row.HeightPx, SHA256: row.SHA256, ContentOrigin: row.ContentOrigin, SourceCreatedAt: row.SourceCreatedAt, SourceModifiedAt: row.SourceModifiedAt, BlobKey: row.BlobKey, SizeBytes: row.SizeBytes, Metadata: append([]byte(nil), row.Metadata...), CreatedAt: row.CreatedAt}
	if row.OperationKey != nil {
		version.OperationKey = *row.OperationKey
	}
	return version
}

func flowNodeToRow(value *domain.FlowNode) *FlowNodeRow {
	return &FlowNodeRow{ID: value.ID, SessionID: value.SessionID, AccountID: value.AccountID, Type: string(value.Type), Title: value.Title, Body: value.Body, AssetID: value.AssetID, AssetVersionID: value.AssetVersionID, AssetVersion: value.AssetVersion, RunID: value.RunID, Outputs: value.Outputs, PositionX: value.PositionX, PositionY: value.PositionY, SortOrder: value.SortOrder, CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt}
}

func flowNodeFromRow(row FlowNodeRow) *domain.FlowNode {
	return &domain.FlowNode{ID: row.ID, SessionID: row.SessionID, AccountID: row.AccountID, Type: domain.FlowNodeType(row.Type), Title: row.Title, Body: row.Body, AssetID: row.AssetID, AssetVersionID: row.AssetVersionID, AssetVersion: row.AssetVersion, RunID: row.RunID, Outputs: row.Outputs, PositionX: row.PositionX, PositionY: row.PositionY, SortOrder: row.SortOrder, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}

func flowEdgeToRow(value *domain.FlowEdge) *FlowEdgeRow {
	return &FlowEdgeRow{ID: value.ID, SessionID: value.SessionID, AccountID: value.AccountID, SourceNodeID: value.SourceNodeID, TargetNodeID: value.TargetNodeID, SourceOutputKey: value.SourceOutputKey, TargetInputKey: value.TargetInputKey, Label: value.Label, CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt}
}

func flowEdgeFromRow(row FlowEdgeRow) *domain.FlowEdge {
	return &domain.FlowEdge{ID: row.ID, SessionID: row.SessionID, AccountID: row.AccountID, SourceNodeID: row.SourceNodeID, TargetNodeID: row.TargetNodeID, SourceOutputKey: row.SourceOutputKey, TargetInputKey: row.TargetInputKey, Label: row.Label, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}

var _ domain.Repository = (*GormRepository)(nil)
