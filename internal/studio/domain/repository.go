package domain

import (
	"context"
	"time"
)

type SessionListQuery struct {
	Limit     int
	Offset    int
	ProjectID *string
}

type ProjectAssetListQuery struct {
	ProjectID       string
	Search          string
	Kind            AssetKind
	Format          string
	CategoryID      string
	CategoryDirect  bool
	SessionID       string
	Rating          *int
	TagIDs          []string
	Untagged        bool
	NoSession       bool
	Uncategorized   bool
	IncludeArchived bool
	ArchivedOnly    bool
	MinWidthPx      *int
	MaxWidthPx      *int
	MinHeightPx     *int
	MaxHeightPx     *int
	MinSizeBytes    *int64
	MaxSizeBytes    *int64
	AddedFrom       *time.Time
	AddedTo         *time.Time
	DuplicatesOnly  bool
	Limit           int
	Cursor          string
	Sort            string
}

type ProjectAssetPage struct {
	Items      []*ProjectAsset
	Total      int64
	NextCursor string
}

type ProjectAssetPatch struct {
	ID          string
	DisplayName *string
	CategoryID  *string
	Rating      *int
	Archived    *bool
}

type ProjectAssetTreeQuery struct {
	ProjectID string
	Mode      string
	ParentID  string
	Limit     int
	Cursor    string
}

type ProjectAssetTreeNode struct {
	ID             string
	Name           string
	Type           string
	Count          int64
	HasChildren    bool
	ProjectAssetID string
	AssetKind      AssetKind
}

type ProjectAssetTreePage struct {
	Nodes      []ProjectAssetTreeNode
	NextCursor string
}

// SessionTranscriptData is the complete persisted read set for replaying a
// Studio session. It intentionally has no caller-provided limit: truncating
// any one of these collections would make historical replay lossy.
type SessionTranscriptData struct {
	Messages []*Message
	Runs     []*Run
	Events   []*Event
}

// Repository persists Studio aggregates. Every read API requires accountID so
// ownership is enforced at the data-access boundary, not only by HTTP handlers.
type Repository interface {
	CreateProject(ctx context.Context, project *Project) error
	GetProject(ctx context.Context, accountID, projectID string) (*Project, error)
	ListProjects(ctx context.Context, accountID string) ([]*Project, error)
	RenameProject(ctx context.Context, accountID, projectID, name string, now time.Time) error
	DeleteProject(ctx context.Context, accountID, projectID string) error
	MoveSessionToProject(ctx context.Context, accountID, sessionID, projectID string) error
	CreateSession(ctx context.Context, session *Session) error
	UpdateSession(ctx context.Context, session *Session) error
	GetSession(ctx context.Context, accountID, sessionID string) (*Session, error)
	ListSessions(ctx context.Context, accountID string, query SessionListQuery) ([]*Session, error)
	ClearSessions(ctx context.Context, accountID string) error

	AppendMessage(ctx context.Context, message *Message) error
	GetMessage(ctx context.Context, accountID, messageID string) (*Message, error)
	ListMessages(ctx context.Context, accountID, sessionID string, limit int) ([]*Message, error)
	ListSessionTranscript(ctx context.Context, accountID, sessionID string) (*SessionTranscriptData, error)

	CreateRun(ctx context.Context, run *Run) error
	CreateRunTurn(ctx context.Context, message *Message, run *Run) (*Run, bool, error)
	GetRunByRequestID(ctx context.Context, accountID, sessionID, requestID string) (*Run, error)
	UpdateRun(ctx context.Context, run *Run) error
	GetRun(ctx context.Context, accountID, runID string) (*Run, error)
	ListSessionRuns(ctx context.Context, accountID, sessionID string, limit int) ([]*Run, error)
	ListSessionTraceRuns(ctx context.Context, accountID, sessionID string, before time.Time, beforeID string, limit int) ([]*Run, error)
	CountSessionTraceRuns(ctx context.Context, accountID, sessionID string) (int64, error)
	ListLatestSessionRuns(ctx context.Context, accountID string, sessionIDs []string) (map[string]*Run, error)
	GetRunProgress(ctx context.Context, accountID, runID string) (*RunProgress, error)
	UpsertRunProgress(ctx context.Context, progress *RunProgress) error
	DeleteRunProgress(ctx context.Context, accountID, runID string) error
	ListRecoverableRuns(ctx context.Context, limit int) ([]*Run, error)
	ListRecoverableRunsAfter(ctx context.Context, afterID string, limit int) ([]*Run, error)
	AppendEvent(ctx context.Context, event *Event) error
	AppendRunEvent(ctx context.Context, event *Event) (*Event, error)
	LastRunEventSequence(ctx context.Context, accountID, runID string) (uint64, error)
	ListEventsAfter(ctx context.Context, accountID, runID string, after uint64, limit int) ([]*Event, error)
	ListRunTraceEvents(ctx context.Context, accountID, runID string) ([]*Event, error)
	ListRunTraceSummaryEvents(ctx context.Context, accountID, runID string) ([]*Event, error)

	CreateApproval(ctx context.Context, approval *Approval) error
	UpdateApproval(ctx context.Context, approval *Approval) error
	GetApproval(ctx context.Context, accountID, approvalID string) (*Approval, error)
	ListApprovals(ctx context.Context, accountID, runID string) ([]*Approval, error)
	CreateClarification(ctx context.Context, clarification *Clarification) error
	UpdateClarification(ctx context.Context, clarification *Clarification) error
	GetClarification(ctx context.Context, accountID, clarificationID string) (*Clarification, error)
	ListClarifications(ctx context.Context, accountID, runID string) ([]*Clarification, error)

	CreateWorkflowExecution(ctx context.Context, execution *WorkflowExecution) error
	UpdateWorkflowExecution(ctx context.Context, execution *WorkflowExecution) error
	GetWorkflowExecutionByTask(ctx context.Context, accountID, taskID string) (*WorkflowExecution, error)
	GetWorkflowExecutionByRunTool(ctx context.Context, accountID, runID, toolCallID string) (*WorkflowExecution, error)
	ListSessionWorkflowExecutions(ctx context.Context, accountID, sessionID string) ([]*WorkflowExecution, error)
	ListActiveWorkflowCounts(ctx context.Context, accountID string, sessionIDs []string) (map[string]int, error)
	ListPendingWorkflowExecutions(ctx context.Context, limit int) ([]*WorkflowExecution, error)

	CreateAsset(ctx context.Context, asset *Asset) error
	CreateAssetWithPlacement(ctx context.Context, asset *Asset, placement *ProjectAsset, usage *SessionAssetUsage) error
	AppendAssetVersion(ctx context.Context, assetID, accountID string, version AssetVersion) error
	GetAsset(ctx context.Context, accountID, assetID string) (*Asset, error)
	GetAssetVersion(ctx context.Context, accountID, versionID string) (*AssetVersion, error)
	GetAssetByCreationKey(ctx context.Context, accountID, creationKey string) (*Asset, error)
	ReferenceAssetInSession(ctx context.Context, placement *ProjectAsset, usage *SessionAssetUsage) error
	CopyProjectAsset(ctx context.Context, accountID, sourceProjectAssetID string, target *ProjectAsset) error
	DeleteProjectAsset(ctx context.Context, accountID, projectAssetID string) error
	GetProjectAsset(ctx context.Context, accountID, projectAssetID string) (*ProjectAsset, error)
	GetProjectAssetByAsset(ctx context.Context, accountID, projectID, assetID string) (*ProjectAsset, error)
	ListDuplicateProjectAssets(ctx context.Context, accountID, projectAssetID string) ([]*ProjectAsset, error)
	ListProjectAssets(ctx context.Context, accountID string, query ProjectAssetListQuery) (*ProjectAssetPage, error)
	ListProjectAssetFormats(ctx context.Context, accountID, projectID string) ([]string, error)
	ListProjectAssetCounts(ctx context.Context, accountID string) (map[string]int64, error)
	ListProjectAssetTree(ctx context.Context, accountID string, query ProjectAssetTreeQuery) (*ProjectAssetTreePage, error)
	ListSessionAssetUsages(ctx context.Context, accountID, sessionID string) ([]*SessionAssetUsage, error)
	ListAssetUsages(ctx context.Context, accountID, assetID string) ([]*SessionAssetUsage, error)
	UpdateProjectAsset(ctx context.Context, accountID string, patch ProjectAssetPatch, now time.Time) error
	SetProjectAssetVersion(ctx context.Context, accountID, projectAssetID, versionID string, now time.Time) error
	CreateAssetCategory(ctx context.Context, category *AssetCategory) error
	GetAssetCategory(ctx context.Context, accountID, categoryID string) (*AssetCategory, error)
	ListAssetCategories(ctx context.Context, accountID, projectID string) ([]*AssetCategory, error)
	UpdateAssetCategory(ctx context.Context, accountID, categoryID, parentID, name string, now time.Time) error
	DeleteAssetCategory(ctx context.Context, accountID, categoryID string) error
	CreateAssetTag(ctx context.Context, tag *AssetTag) error
	ListAssetTags(ctx context.Context, accountID string) ([]*AssetTag, error)
	RenameAssetTag(ctx context.Context, accountID, tagID, name string, now time.Time) error
	SetProjectAssetTags(ctx context.Context, accountID, projectAssetID string, tagIDs []string) error
	GetAssetVersionPalette(ctx context.Context, accountID, versionID string) (*AssetVersionPalette, error)
	ClaimPendingPaletteJobs(ctx context.Context, limit int, now time.Time) ([]*AssetVersionPalette, error)
	CompleteAssetVersionPalette(ctx context.Context, accountID, versionID string, colors []PaletteColor, samplePoints []float64, widthPx, heightPx int, now time.Time) error
	FailAssetVersionPalette(ctx context.Context, accountID, versionID, errorCode string, nextRetryAt time.Time) error
	RetryAssetVersionPalette(ctx context.Context, accountID, versionID string, now time.Time) error
	GetAssetLibraryPreferences(ctx context.Context, accountID string) (*AssetLibraryPreferences, error)
	SaveAssetLibraryPreferences(ctx context.Context, preferences *AssetLibraryPreferences) error
	RegisterBlobWriteIntent(ctx context.Context, intent *BlobWriteIntent) error
	ClaimExpiredBlobWriteIntents(ctx context.Context, limit int, now time.Time) ([]*BlobWriteIntent, error)
	IsBlobReferenced(ctx context.Context, accountID, blobKey string) (bool, error)
	DeleteBlobWriteIntent(ctx context.Context, accountID, blobKey string) error
	ListSessionAssets(ctx context.Context, accountID, sessionID string, limit int) ([]*Asset, error)

	SaveFlowNode(ctx context.Context, node *FlowNode) error
	SaveFlowEdge(ctx context.Context, edge *FlowEdge) error
	DeleteFlowNode(ctx context.Context, accountID, sessionID, nodeID string) error
	DeleteFlowEdge(ctx context.Context, accountID, sessionID, edgeID string) error
	GetFlow(ctx context.Context, accountID, sessionID string) ([]*FlowNode, []*FlowEdge, error)

	CreateModelConfig(ctx context.Context, config *ModelConfig) error
	UpdateModelConfig(ctx context.Context, config *ModelConfig) error
	GetModelConfig(ctx context.Context, accountID, configID string) (*ModelConfig, error)
	ListModelConfigs(ctx context.Context, accountID string) ([]*ModelConfig, error)

	CreateSkill(ctx context.Context, skill *Skill) error
	GetSkill(ctx context.Context, accountID, skillID string) (*Skill, error)
	ListSkills(ctx context.Context, accountID string) ([]*Skill, error)
	ListSkillVersions(ctx context.Context, accountID, skillID string) ([]*SkillVersion, error)
	GetSkillVersion(ctx context.Context, accountID, skillID, version string) (*SkillVersion, error)

	CreateMCPConnector(ctx context.Context, connector *MCPConnector) error
	UpdateMCPConnector(ctx context.Context, connector *MCPConnector) error
	GetMCPConnector(ctx context.Context, accountID, connectorID string) (*MCPConnector, error)
	ListMCPConnectors(ctx context.Context, accountID string) ([]*MCPConnector, error)

	UpsertAgentWorkflowSetting(ctx context.Context, setting *AgentWorkflowSetting) error
	ListAgentWorkflowSettings(ctx context.Context, accountID string) ([]*AgentWorkflowSetting, error)
}
