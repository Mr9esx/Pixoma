package domain

import (
	"context"
	"time"
)

type SessionListQuery struct {
	Limit  int
	Offset int
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
	CreateSession(ctx context.Context, session *Session) error
	UpdateSession(ctx context.Context, session *Session) error
	GetSession(ctx context.Context, accountID, sessionID string) (*Session, error)
	ListSessions(ctx context.Context, accountID string, query SessionListQuery) ([]*Session, error)

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

	CreateWorkflowExecution(ctx context.Context, execution *WorkflowExecution) error
	UpdateWorkflowExecution(ctx context.Context, execution *WorkflowExecution) error
	GetWorkflowExecutionByTask(ctx context.Context, accountID, taskID string) (*WorkflowExecution, error)
	GetWorkflowExecutionByRunTool(ctx context.Context, accountID, runID, toolCallID string) (*WorkflowExecution, error)
	ListPendingWorkflowExecutions(ctx context.Context, limit int) ([]*WorkflowExecution, error)

	CreateAsset(ctx context.Context, asset *Asset) error
	AppendAssetVersion(ctx context.Context, assetID, accountID string, version AssetVersion) error
	GetAsset(ctx context.Context, accountID, assetID string) (*Asset, error)
	ListSessionAssets(ctx context.Context, accountID, sessionID string, limit int) ([]*Asset, error)
	SaveAssetToLibrary(ctx context.Context, accountID, assetID, folderID string, savedAt time.Time) error
	ListLibraryAssets(ctx context.Context, accountID, folderID string, limit int) ([]*Asset, error)
	CreateLibraryFolder(ctx context.Context, folder *LibraryFolder) error
	ListLibraryFolders(ctx context.Context, accountID string) ([]*LibraryFolder, error)

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
	UpdateSkill(ctx context.Context, skill *Skill) error
	GetSkill(ctx context.Context, accountID, skillID string) (*Skill, error)
	ListSkills(ctx context.Context, accountID string) ([]*Skill, error)

	CreateMCPConnector(ctx context.Context, connector *MCPConnector) error
	UpdateMCPConnector(ctx context.Context, connector *MCPConnector) error
	GetMCPConnector(ctx context.Context, accountID, connectorID string) (*MCPConnector, error)
	ListMCPConnectors(ctx context.Context, accountID string) ([]*MCPConnector, error)

	UpsertAgentWorkflowSetting(ctx context.Context, setting *AgentWorkflowSetting) error
	ListAgentWorkflowSettings(ctx context.Context, accountID string) ([]*AgentWorkflowSetting, error)
}
