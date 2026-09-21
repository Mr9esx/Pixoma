package domain

import (
	"context"
	"time"
)

type SessionListQuery struct {
	Limit  int
	Offset int
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

	CreateRun(ctx context.Context, run *Run) error
	UpdateRun(ctx context.Context, run *Run) error
	GetRun(ctx context.Context, accountID, runID string) (*Run, error)
	ListRecoverableRuns(ctx context.Context, limit int) ([]*Run, error)
	AppendEvent(ctx context.Context, event *Event) error
	ListEventsAfter(ctx context.Context, accountID, runID string, after uint64, limit int) ([]*Event, error)

	CreateApproval(ctx context.Context, approval *Approval) error
	UpdateApproval(ctx context.Context, approval *Approval) error
	GetApproval(ctx context.Context, accountID, approvalID string) (*Approval, error)
	ListApprovals(ctx context.Context, accountID, runID string) ([]*Approval, error)

	CreateAsset(ctx context.Context, asset *Asset) error
	AppendAssetVersion(ctx context.Context, assetID, accountID string, version AssetVersion) error
	GetAsset(ctx context.Context, accountID, assetID string) (*Asset, error)
	ListSessionAssets(ctx context.Context, accountID, sessionID string, limit int) ([]*Asset, error)
	SaveAssetToLibrary(ctx context.Context, accountID, assetID, folderID string, savedAt time.Time) error
	ListLibraryAssets(ctx context.Context, accountID, folderID string, limit int) ([]*Asset, error)

	SaveFlowNode(ctx context.Context, node *FlowNode) error
	SaveFlowEdge(ctx context.Context, edge *FlowEdge) error
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
}
