package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

type SessionRow struct {
	ID             string `gorm:"primaryKey;size:64"`
	AccountID      string `gorm:"size:64;not null;index:idx_studio_sessions_account_updated"`
	Title          string `gorm:"size:256;not null"`
	PermissionMode string `gorm:"size:32;not null"`
	ModelConfigID  string `gorm:"size:64;index"`
	Status         string `gorm:"size:32;not null"`
	CreatedAt      time.Time
	UpdatedAt      time.Time `gorm:"index:idx_studio_sessions_account_updated"`
}

func (SessionRow) TableName() string { return "studio_sessions" }

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
	ID               string `gorm:"primaryKey;size:64"`
	SessionID        string `gorm:"size:64;not null;index"`
	AccountID        string `gorm:"size:64;not null;index"`
	TriggerMessageID string `gorm:"size:64;not null;index"`
	Status           string `gorm:"size:32;not null;index"`
	ModelConfigID    string `gorm:"size:64;index"`
	SkillIDsJSON     []byte `gorm:"type:blob"`
	AssetIDsJSON     []byte `gorm:"type:blob"`
	ErrorCode        string `gorm:"size:128"`
	ErrorMessage     string `gorm:"type:text"`
	CreatedAt        time.Time
	StartedAt        time.Time
	CompletedAt      time.Time
	UpdatedAt        time.Time
}

func (RunRow) TableName() string { return "studio_runs" }

type EventRow struct {
	ID        string `gorm:"primaryKey;size:64"`
	RunID     string `gorm:"size:64;not null;uniqueIndex:idx_studio_events_run_sequence;index:idx_studio_events_run_sequence_order"`
	SessionID string `gorm:"size:64;not null;index"`
	AccountID string `gorm:"size:64;not null;index"`
	Sequence  uint64 `gorm:"not null;uniqueIndex:idx_studio_events_run_sequence;index:idx_studio_events_run_sequence_order"`
	Type      string `gorm:"size:96;not null"`
	Payload   []byte `gorm:"type:blob;not null"`
	CreatedAt time.Time
}

func (EventRow) TableName() string { return "studio_events" }

type ApprovalRow struct {
	ID         string `gorm:"primaryKey;size:64"`
	RunID      string `gorm:"size:64;not null;index;uniqueIndex:idx_studio_approvals_run_tool"`
	SessionID  string `gorm:"size:64;not null;index"`
	AccountID  string `gorm:"size:64;not null;index"`
	ToolCallID string `gorm:"size:128;not null;uniqueIndex:idx_studio_approvals_run_tool"`
	Action     string `gorm:"size:128;not null"`
	Status     string `gorm:"size:32;not null;index"`
	ResolvedBy string `gorm:"size:64"`
	CreatedAt  time.Time
	ResolvedAt time.Time
	UpdatedAt  time.Time
}

func (ApprovalRow) TableName() string { return "studio_approvals" }

type WorkflowExecutionRow struct {
	ID              string    `gorm:"primaryKey;size:64"`
	AccountID       string    `gorm:"size:64;not null;index"`
	SessionID       string    `gorm:"size:64;not null;index"`
	RunID           string    `gorm:"size:64;not null;uniqueIndex:idx_studio_workflow_executions_run_tool;index"`
	ToolCallID      string    `gorm:"size:128;not null;uniqueIndex:idx_studio_workflow_executions_run_tool"`
	TaskID          string    `gorm:"size:128;not null;uniqueIndex;index"`
	WorkflowID      string    `gorm:"size:64;not null;index"`
	OperationNodeID string    `gorm:"size:64;not null;index"`
	Status          string    `gorm:"size:32;not null;index"`
	ErrorMessage    string    `gorm:"type:text"`
	CreatedAt       time.Time `gorm:"index"`
	UpdatedAt       time.Time `gorm:"index"`
	CompletedAt     time.Time
}

func (WorkflowExecutionRow) TableName() string { return "studio_workflow_executions" }

type AssetRow struct {
	ID             string `gorm:"primaryKey;size:64"`
	SessionID      string `gorm:"size:64;not null;index"`
	AccountID      string `gorm:"size:64;not null;index"`
	Name           string `gorm:"size:512;not null"`
	Kind           string `gorm:"size:32;not null;index"`
	Origin         string `gorm:"size:32;not null;index"`
	SourceRunID    string `gorm:"size:64;index"`
	CurrentVersion int    `gorm:"not null"`
	LibrarySavedAt time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (AssetRow) TableName() string { return "studio_assets" }

type AssetVersionRow struct {
	ID        string `gorm:"primaryKey;size:64"`
	AssetID   string `gorm:"size:64;not null;uniqueIndex:idx_studio_asset_versions_asset_version;index"`
	AccountID string `gorm:"size:64;not null;index"`
	Version   int    `gorm:"not null;uniqueIndex:idx_studio_asset_versions_asset_version"`
	MIMEType  string `gorm:"size:256;not null"`
	BlobKey   string `gorm:"size:1024;not null"`
	SizeBytes int64  `gorm:"not null"`
	Metadata  []byte `gorm:"type:blob"`
	CreatedAt time.Time
}

func (AssetVersionRow) TableName() string { return "studio_asset_versions" }

type LibraryFolderRow struct {
	ID        string `gorm:"primaryKey;size:64"`
	AccountID string `gorm:"size:64;not null;index"`
	ParentID  string `gorm:"size:64;index"`
	Name      string `gorm:"size:256;not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (LibraryFolderRow) TableName() string { return "studio_library_folders" }

type LibraryAssetRow struct {
	ID             uint64 `gorm:"primaryKey;autoIncrement"`
	AccountID      string `gorm:"size:64;not null;uniqueIndex:idx_studio_library_account_asset;index"`
	AssetID        string `gorm:"size:64;not null;uniqueIndex:idx_studio_library_account_asset;index"`
	AssetVersionID string `gorm:"size:64;not null"`
	FolderID       string `gorm:"size:64;index"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (LibraryAssetRow) TableName() string { return "studio_library_assets" }

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
	RunID          string `gorm:"size:64;index"`
	PositionX      float64
	PositionY      float64
	SortOrder      int `gorm:"not null;index:idx_studio_flow_nodes_session_order"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (FlowNodeRow) TableName() string { return "studio_flow_nodes" }

type FlowEdgeRow struct {
	ID           string `gorm:"primaryKey;size:64"`
	SessionID    string `gorm:"size:64;not null;index"`
	AccountID    string `gorm:"size:64;not null;index"`
	SourceNodeID string `gorm:"size:64;not null;index"`
	TargetNodeID string `gorm:"size:64;not null;index"`
	Label        string `gorm:"size:256"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (FlowEdgeRow) TableName() string { return "studio_flow_edges" }

func Models() []any {
	return []any{
		&SessionRow{}, &MessageRow{}, &RunRow{}, &EventRow{}, &ApprovalRow{},
		&WorkflowExecutionRow{},
		&AssetRow{}, &AssetVersionRow{}, &LibraryFolderRow{}, &LibraryAssetRow{},
		&FlowNodeRow{}, &FlowEdgeRow{}, &ModelConfigRow{},
		&SkillRow{}, &MCPConnectorRow{}, &AgentWorkflowSettingRow{},
	}
}

type GormRepository struct {
	db *gorm.DB
}

func NewGormRepository(db *gorm.DB) *GormRepository { return &GormRepository{db: db} }

func (r *GormRepository) CreateSession(ctx context.Context, session *domain.Session) error {
	if session == nil {
		return fmt.Errorf("%w: nil session", domain.ErrInvalid)
	}
	err := r.db.WithContext(ctx).Create(sessionToRow(session)).Error
	return translateCreateError(err)
}

func (r *GormRepository) UpdateSession(ctx context.Context, session *domain.Session) error {
	if session == nil {
		return fmt.Errorf("%w: nil session", domain.ErrInvalid)
	}
	result := r.db.WithContext(ctx).Model(&SessionRow{}).
		Where("id = ? AND account_id = ?", session.ID, session.AccountID).
		Updates(sessionToRow(session))
	return resultError(result)
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
	err := r.db.WithContext(ctx).Where("account_id = ?", accountID).
		Order("updated_at DESC, id DESC").Limit(limit).Offset(query.Offset).Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]*domain.Session, 0, len(rows))
	for _, row := range rows {
		out = append(out, sessionFromRow(row))
	}
	return out, nil
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

func (r *GormRepository) CreateRun(ctx context.Context, run *domain.Run) error {
	if run == nil {
		return fmt.Errorf("%w: nil run", domain.ErrInvalid)
	}
	return translateCreateError(r.db.WithContext(ctx).Create(runToRow(run)).Error)
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

func (r *GormRepository) AppendEvent(ctx context.Context, event *domain.Event) error {
	if event == nil {
		return fmt.Errorf("%w: nil event", domain.ErrInvalid)
	}
	row := eventToRow(event)
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "run_id"}, {Name: "sequence"}},
		DoNothing: true,
	}).Create(row).Error
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

func (r *GormRepository) ListSessionAssets(ctx context.Context, accountID, sessionID string, limit int) ([]*domain.Asset, error) {
	var rows []AssetRow
	err := r.db.WithContext(ctx).Where("account_id = ? AND session_id = ?", accountID, sessionID).
		Order("created_at ASC, id ASC").Limit(normalizeLimit(limit)).Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return r.assetsFromRows(ctx, rows)
}

func (r *GormRepository) SaveAssetToLibrary(ctx context.Context, accountID, assetID, folderID string, savedAt time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var asset AssetRow
		if err := tx.Where("id = ? AND account_id = ?", assetID, accountID).First(&asset).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrNotFound
			}
			return err
		}
		var version AssetVersionRow
		if err := tx.Where("asset_id = ? AND account_id = ? AND version = ?", assetID, accountID, asset.CurrentVersion).First(&version).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("%w: asset has no current version", domain.ErrInvalid)
			}
			return err
		}
		if err := resultError(tx.Model(&AssetRow{}).Where("id = ? AND account_id = ?", assetID, accountID).
			Update("library_saved_at", savedAt.UTC())); err != nil {
			return err
		}
		row := &LibraryAssetRow{AccountID: accountID, AssetID: assetID, AssetVersionID: version.ID, FolderID: folderID, CreatedAt: savedAt.UTC(), UpdatedAt: savedAt.UTC()}
		return tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "account_id"}, {Name: "asset_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"asset_version_id", "folder_id", "updated_at"}),
		}).Create(row).Error
	})
}

func (r *GormRepository) ListLibraryAssets(ctx context.Context, accountID, folderID string, limit int) ([]*domain.Asset, error) {
	referenceQuery := r.db.WithContext(ctx).Where("account_id = ?", accountID)
	if folderID != "" {
		referenceQuery = referenceQuery.Where("folder_id = ?", folderID)
	}
	var references []LibraryAssetRow
	if err := referenceQuery.Order("updated_at DESC, asset_id DESC").Limit(normalizeLimit(limit)).Find(&references).Error; err != nil {
		return nil, err
	}
	if len(references) == 0 {
		return []*domain.Asset{}, nil
	}
	assetIDs := make([]string, 0, len(references))
	for _, reference := range references {
		assetIDs = append(assetIDs, reference.AssetID)
	}
	query := r.db.WithContext(ctx).Table("studio_assets AS a").
		Select("a.*").
		Where("a.account_id = ? AND a.id IN ?", accountID, assetIDs)
	var rows []AssetRow
	if err := query.Scan(&rows).Error; err != nil {
		return nil, err
	}
	assets, err := r.assetsFromRows(ctx, rows)
	if err != nil {
		return nil, err
	}
	assetsByID := make(map[string]*domain.Asset, len(assets))
	for _, asset := range assets {
		assetsByID[asset.ID] = asset
	}
	ordered := make([]*domain.Asset, 0, len(references))
	for _, reference := range references {
		asset := assetsByID[reference.AssetID]
		if asset == nil {
			continue
		}
		if err := keepLibraryVersion(asset, reference.AssetVersionID); err != nil {
			return nil, err
		}
		ordered = append(ordered, asset)
	}
	return ordered, nil
}

func keepLibraryVersion(asset *domain.Asset, versionID string) error {
	if asset == nil || len(asset.Versions) == 0 {
		return fmt.Errorf("%w: library asset has no versions", domain.ErrInvalid)
	}
	if versionID == "" {
		// References created before version pinning retain the version that was
		// current when this migration first reads them. Newly saved references
		// always persist an explicit version ID.
		versionID = asset.Versions[len(asset.Versions)-1].ID
	}
	for _, version := range asset.Versions {
		if version.ID == versionID {
			asset.Versions = []domain.AssetVersion{version}
			asset.CurrentVersion = version.Version
			return nil
		}
	}
	return fmt.Errorf("%w: library asset references a missing version", domain.ErrInvalid)
}

func (r *GormRepository) CreateLibraryFolder(ctx context.Context, folder *domain.LibraryFolder) error {
	if folder == nil {
		return fmt.Errorf("%w: nil library folder", domain.ErrInvalid)
	}
	return translateCreateError(r.db.WithContext(ctx).Create(&LibraryFolderRow{
		ID: folder.ID, AccountID: folder.AccountID, ParentID: folder.ParentID, Name: folder.Name,
		CreatedAt: folder.CreatedAt, UpdatedAt: folder.UpdatedAt,
	}).Error)
}

func (r *GormRepository) ListLibraryFolders(ctx context.Context, accountID string) ([]*domain.LibraryFolder, error) {
	var rows []LibraryFolderRow
	if err := r.db.WithContext(ctx).Where("account_id = ?", accountID).Order("name ASC, id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*domain.LibraryFolder, 0, len(rows))
	for _, row := range rows {
		out = append(out, &domain.LibraryFolder{ID: row.ID, AccountID: row.AccountID, ParentID: row.ParentID, Name: row.Name, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt})
	}
	return out, nil
}

func (r *GormRepository) SaveFlowNode(ctx context.Context, node *domain.FlowNode) error {
	if node == nil {
		return fmt.Errorf("%w: nil flow node", domain.ErrInvalid)
	}
	row := flowNodeToRow(node)
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"type", "title", "body", "asset_id", "run_id", "position_x", "position_y", "sort_order", "updated_at",
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
		DoUpdates: clause.AssignmentColumns([]string{"source_node_id", "target_node_id", "label", "updated_at"}),
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
	for _, row := range rows {
		asset := assetFromRow(row)
		if err := r.loadVersions(ctx, asset); err != nil {
			return nil, err
		}
		out = append(out, asset)
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
	return &SessionRow{ID: value.ID, AccountID: value.AccountID, Title: value.Title, PermissionMode: string(value.PermissionMode), ModelConfigID: value.ModelConfigID, Status: string(value.Status), CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt}
}

func sessionFromRow(row SessionRow) *domain.Session {
	return &domain.Session{ID: row.ID, AccountID: row.AccountID, Title: row.Title, PermissionMode: domain.PermissionMode(row.PermissionMode), ModelConfigID: row.ModelConfigID, Status: domain.SessionStatus(row.Status), CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}

func messageToRow(value *domain.Message) *MessageRow {
	return &MessageRow{ID: value.ID, SessionID: value.SessionID, AccountID: value.AccountID, RunID: value.RunID, Role: string(value.Role), ContentJSON: append([]byte(nil), value.ContentJSON...), CreatedAt: value.CreatedAt}
}

func messageFromRow(row MessageRow) *domain.Message {
	return &domain.Message{ID: row.ID, SessionID: row.SessionID, AccountID: row.AccountID, RunID: row.RunID, Role: domain.MessageRole(row.Role), ContentJSON: append([]byte(nil), row.ContentJSON...), CreatedAt: row.CreatedAt}
}

func runToRow(value *domain.Run) *RunRow {
	skillIDs, _ := json.Marshal(value.SkillIDs)
	assetIDs, _ := json.Marshal(value.AssetIDs)
	if len(value.AssetReferences) > 0 {
		assetIDs, _ = json.Marshal(value.AssetReferences)
	}
	return &RunRow{ID: value.ID, SessionID: value.SessionID, AccountID: value.AccountID, TriggerMessageID: value.TriggerMessageID, Status: string(value.Status), ModelConfigID: value.ModelConfigID, SkillIDsJSON: skillIDs, AssetIDsJSON: assetIDs, ErrorCode: value.ErrorCode, ErrorMessage: value.ErrorMessage, CreatedAt: value.CreatedAt, StartedAt: value.StartedAt, CompletedAt: value.CompletedAt, UpdatedAt: value.UpdatedAt}
}

func runFromRow(row RunRow) *domain.Run {
	var skillIDs []string
	_ = json.Unmarshal(row.SkillIDsJSON, &skillIDs)
	var assetIDs []string
	var assetReferences []domain.AssetReference
	if err := json.Unmarshal(row.AssetIDsJSON, &assetReferences); err == nil && len(assetReferences) > 0 {
		for _, reference := range assetReferences {
			assetIDs = append(assetIDs, reference.AssetID)
		}
	} else {
		_ = json.Unmarshal(row.AssetIDsJSON, &assetIDs)
	}
	return &domain.Run{ID: row.ID, SessionID: row.SessionID, AccountID: row.AccountID, TriggerMessageID: row.TriggerMessageID, Status: domain.RunStatus(row.Status), ModelConfigID: row.ModelConfigID, SkillIDs: skillIDs, AssetIDs: assetIDs, AssetReferences: assetReferences, ErrorCode: row.ErrorCode, ErrorMessage: row.ErrorMessage, CreatedAt: row.CreatedAt, StartedAt: row.StartedAt, CompletedAt: row.CompletedAt, UpdatedAt: row.UpdatedAt}
}

func eventToRow(value *domain.Event) *EventRow {
	return &EventRow{ID: value.ID, RunID: value.RunID, SessionID: value.SessionID, AccountID: value.AccountID, Sequence: value.Sequence, Type: value.Type, Payload: append([]byte(nil), value.Payload...), CreatedAt: value.CreatedAt}
}

func eventFromRow(row EventRow) *domain.Event {
	return &domain.Event{ID: row.ID, RunID: row.RunID, SessionID: row.SessionID, AccountID: row.AccountID, Sequence: row.Sequence, Type: row.Type, Payload: append([]byte(nil), row.Payload...), CreatedAt: row.CreatedAt}
}

func approvalToRow(value *domain.Approval) *ApprovalRow {
	return &ApprovalRow{ID: value.ID, RunID: value.RunID, SessionID: value.SessionID, AccountID: value.AccountID, ToolCallID: value.ToolCallID, Action: value.Action, Status: string(value.Status), ResolvedBy: value.ResolvedBy, CreatedAt: value.CreatedAt, ResolvedAt: value.ResolvedAt, UpdatedAt: value.UpdatedAt}
}

func approvalFromRow(row ApprovalRow) *domain.Approval {
	return &domain.Approval{ID: row.ID, RunID: row.RunID, SessionID: row.SessionID, AccountID: row.AccountID, ToolCallID: row.ToolCallID, Action: row.Action, Status: domain.ApprovalStatus(row.Status), ResolvedBy: row.ResolvedBy, CreatedAt: row.CreatedAt, ResolvedAt: row.ResolvedAt, UpdatedAt: row.UpdatedAt}
}

func workflowExecutionToRow(value *domain.WorkflowExecution) *WorkflowExecutionRow {
	return &WorkflowExecutionRow{ID: value.ID, AccountID: value.AccountID, SessionID: value.SessionID, RunID: value.RunID, ToolCallID: value.ToolCallID, TaskID: value.TaskID, WorkflowID: value.WorkflowID, OperationNodeID: value.OperationNodeID, Status: string(value.Status), ErrorMessage: value.ErrorMessage, CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt, CompletedAt: value.CompletedAt}
}

func workflowExecutionFromRow(row WorkflowExecutionRow) *domain.WorkflowExecution {
	return &domain.WorkflowExecution{ID: row.ID, AccountID: row.AccountID, SessionID: row.SessionID, RunID: row.RunID, ToolCallID: row.ToolCallID, TaskID: row.TaskID, WorkflowID: row.WorkflowID, OperationNodeID: row.OperationNodeID, Status: domain.WorkflowExecutionStatus(row.Status), ErrorMessage: row.ErrorMessage, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, CompletedAt: row.CompletedAt}
}

func assetToRow(value *domain.Asset) *AssetRow {
	return &AssetRow{ID: value.ID, SessionID: value.SessionID, AccountID: value.AccountID, Name: value.Name, Kind: string(value.Kind), Origin: string(value.Origin), SourceRunID: value.SourceRunID, CurrentVersion: value.CurrentVersion, LibrarySavedAt: value.LibrarySavedAt, CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt}
}

func assetFromRow(row AssetRow) *domain.Asset {
	return &domain.Asset{ID: row.ID, SessionID: row.SessionID, AccountID: row.AccountID, Name: row.Name, Kind: domain.AssetKind(row.Kind), Origin: domain.AssetOrigin(row.Origin), SourceRunID: row.SourceRunID, CurrentVersion: row.CurrentVersion, LibrarySavedAt: row.LibrarySavedAt, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}

func assetVersionToRow(value domain.AssetVersion) *AssetVersionRow {
	return &AssetVersionRow{ID: value.ID, AssetID: value.AssetID, AccountID: value.AccountID, Version: value.Version, MIMEType: value.MIMEType, BlobKey: value.BlobKey, SizeBytes: value.SizeBytes, Metadata: append([]byte(nil), value.Metadata...), CreatedAt: value.CreatedAt}
}

func assetVersionFromRow(row AssetVersionRow) domain.AssetVersion {
	return domain.AssetVersion{ID: row.ID, AssetID: row.AssetID, AccountID: row.AccountID, Version: row.Version, MIMEType: row.MIMEType, BlobKey: row.BlobKey, SizeBytes: row.SizeBytes, Metadata: append([]byte(nil), row.Metadata...), CreatedAt: row.CreatedAt}
}

func flowNodeToRow(value *domain.FlowNode) *FlowNodeRow {
	return &FlowNodeRow{ID: value.ID, SessionID: value.SessionID, AccountID: value.AccountID, Type: string(value.Type), Title: value.Title, Body: value.Body, AssetID: value.AssetID, AssetVersionID: value.AssetVersionID, AssetVersion: value.AssetVersion, RunID: value.RunID, PositionX: value.PositionX, PositionY: value.PositionY, SortOrder: value.SortOrder, CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt}
}

func flowNodeFromRow(row FlowNodeRow) *domain.FlowNode {
	return &domain.FlowNode{ID: row.ID, SessionID: row.SessionID, AccountID: row.AccountID, Type: domain.FlowNodeType(row.Type), Title: row.Title, Body: row.Body, AssetID: row.AssetID, AssetVersionID: row.AssetVersionID, AssetVersion: row.AssetVersion, RunID: row.RunID, PositionX: row.PositionX, PositionY: row.PositionY, SortOrder: row.SortOrder, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}

func flowEdgeToRow(value *domain.FlowEdge) *FlowEdgeRow {
	return &FlowEdgeRow{ID: value.ID, SessionID: value.SessionID, AccountID: value.AccountID, SourceNodeID: value.SourceNodeID, TargetNodeID: value.TargetNodeID, Label: value.Label, CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt}
}

func flowEdgeFromRow(row FlowEdgeRow) *domain.FlowEdge {
	return &domain.FlowEdge{ID: row.ID, SessionID: row.SessionID, AccountID: row.AccountID, SourceNodeID: row.SourceNodeID, TargetNodeID: row.TargetNodeID, Label: row.Label, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}

var _ domain.Repository = (*GormRepository)(nil)
