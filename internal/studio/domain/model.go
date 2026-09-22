package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrInvalid           = errors.New("studio: invalid value")
	ErrInvalidTransition = errors.New("studio: invalid state transition")
	ErrNotFound          = errors.New("studio: not found")
	ErrAlreadyExists     = errors.New("studio: already exists")
)

const DefaultSessionTitle = "新对话"

type PermissionMode string

const (
	PermissionRequestApproval PermissionMode = "request_approval"
	PermissionAutoApprove     PermissionMode = "auto_approve"
	PermissionFullAccess      PermissionMode = "full_access"
)

func (m PermissionMode) Valid() bool {
	switch m {
	case PermissionRequestApproval, PermissionAutoApprove, PermissionFullAccess:
		return true
	default:
		return false
	}
}

type SessionStatus string

const (
	SessionActive SessionStatus = "active"
)

// Session is an AI Studio conversation and is always owned by one console
// account. It deliberately does not reuse the messaging-channel Session.
type Session struct {
	ID                             string
	AccountID                      string
	Title                          string
	PermissionMode                 PermissionMode
	ModelConfigID                  string
	ContextSummary                 string
	ContextSummaryThroughMessageID string
	Status                         SessionStatus
	CreatedAt                      time.Time
	UpdatedAt                      time.Time
}

func NewSession(id, accountID string, now time.Time) (*Session, error) {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(accountID) == "" {
		return nil, fmt.Errorf("%w: session id and account id are required", ErrInvalid)
	}
	now = now.UTC()
	return &Session{
		ID:             id,
		AccountID:      accountID,
		Title:          DefaultSessionTitle,
		PermissionMode: PermissionRequestApproval,
		Status:         SessionActive,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

func (s *Session) Rename(title string, now time.Time) error {
	title = strings.TrimSpace(title)
	if title == "" {
		return fmt.Errorf("%w: title is required", ErrInvalid)
	}
	s.Title = title
	s.UpdatedAt = now.UTC()
	return nil
}

func (s *Session) Configure(modelConfigID string, mode PermissionMode, now time.Time) error {
	if !mode.Valid() {
		return fmt.Errorf("%w: invalid permission mode", ErrInvalid)
	}
	s.ModelConfigID = strings.TrimSpace(modelConfigID)
	s.PermissionMode = mode
	s.UpdatedAt = now.UTC()
	return nil
}

// UpdateContextSummary records the oldest message represented by a compacted
// summary. The boundary is deliberately a message id instead of an array
// offset, because new turns can be appended while an Agent run is executing.
func (s *Session) UpdateContextSummary(summary, throughMessageID string, now time.Time) error {
	if strings.TrimSpace(summary) == "" || strings.TrimSpace(throughMessageID) == "" {
		return fmt.Errorf("%w: context summary and boundary are required", ErrInvalid)
	}
	trimmedSummary := strings.TrimSpace(summary)
	if len([]rune(trimmedSummary)) > 12000 {
		trimmedSummary = string([]rune(trimmedSummary)[:12000])
	}
	s.ContextSummary = trimmedSummary
	s.ContextSummaryThroughMessageID = strings.TrimSpace(throughMessageID)
	s.UpdatedAt = now.UTC()
	return nil
}

type MessageRole string

const (
	MessageRoleUser      MessageRole = "user"
	MessageRoleAssistant MessageRole = "assistant"
	MessageRoleSystem    MessageRole = "system"
	MessageRoleTool      MessageRole = "tool"
)

type Message struct {
	ID          string
	SessionID   string
	AccountID   string
	RunID       string
	Role        MessageRole
	ContentJSON json.RawMessage
	CreatedAt   time.Time
}

type RunStatus string

const (
	RunQueued          RunStatus = "queued"
	RunRunning         RunStatus = "running"
	RunWaitingApproval RunStatus = "waiting_approval"
	RunSucceeded       RunStatus = "succeeded"
	RunFailed          RunStatus = "failed"
	RunCancelled       RunStatus = "cancelled"
)

func (s RunStatus) Terminal() bool {
	return s == RunSucceeded || s == RunFailed || s == RunCancelled
}

type Run struct {
	ID               string
	SessionID        string
	AccountID        string
	TriggerMessageID string
	Status           RunStatus
	ModelConfigID    string
	SkillIDs         []string
	AssetIDs         []string
	AssetReferences  []AssetReference
	ErrorCode        string
	ErrorMessage     string
	CreatedAt        time.Time
	StartedAt        time.Time
	CompletedAt      time.Time
	UpdatedAt        time.Time
}

// AssetReference identifies the immutable asset version consumed by a Run.
// AssetID is retained separately on Run for compatibility with historical
// records created before version snapshots were introduced.
type AssetReference struct {
	AssetID        string `json:"asset_id"`
	AssetVersionID string `json:"asset_version_id"`
}

func NewRun(id, sessionID, accountID, triggerMessageID string, now time.Time) (*Run, error) {
	if anyBlank(id, sessionID, accountID, triggerMessageID) {
		return nil, fmt.Errorf("%w: run ownership and trigger are required", ErrInvalid)
	}
	now = now.UTC()
	return &Run{
		ID:               id,
		SessionID:        sessionID,
		AccountID:        accountID,
		TriggerMessageID: triggerMessageID,
		Status:           RunQueued,
		CreatedAt:        now,
		UpdatedAt:        now,
	}, nil
}

func (r *Run) Start(now time.Time) error {
	if r.Status != RunQueued {
		return ErrInvalidTransition
	}
	now = now.UTC()
	r.Status = RunRunning
	r.StartedAt = now
	r.UpdatedAt = now
	return nil
}

func (r *Run) WaitForApproval(now time.Time) error {
	if r.Status != RunRunning {
		return ErrInvalidTransition
	}
	r.Status = RunWaitingApproval
	r.UpdatedAt = now.UTC()
	return nil
}

func (r *Run) Resume(now time.Time) error {
	if r.Status != RunWaitingApproval {
		return ErrInvalidTransition
	}
	r.Status = RunRunning
	r.UpdatedAt = now.UTC()
	return nil
}

func (r *Run) Succeed(now time.Time) error {
	if r.Status != RunRunning {
		return ErrInvalidTransition
	}
	r.finish(RunSucceeded, now)
	return nil
}

func (r *Run) Fail(code, message string, now time.Time) error {
	if r.Status != RunQueued && r.Status != RunRunning && r.Status != RunWaitingApproval {
		return ErrInvalidTransition
	}
	r.ErrorCode = strings.TrimSpace(code)
	r.ErrorMessage = strings.TrimSpace(message)
	r.finish(RunFailed, now)
	return nil
}

func (r *Run) Cancel(now time.Time) error {
	if r.Status != RunQueued && r.Status != RunRunning && r.Status != RunWaitingApproval {
		return ErrInvalidTransition
	}
	r.finish(RunCancelled, now)
	return nil
}

func (r *Run) finish(status RunStatus, now time.Time) {
	now = now.UTC()
	r.Status = status
	r.CompletedAt = now
	r.UpdatedAt = now
}

type Event struct {
	ID        string
	RunID     string
	SessionID string
	AccountID string
	Sequence  uint64
	Type      string
	Payload   json.RawMessage
	CreatedAt time.Time
}

type ApprovalStatus string

const (
	ApprovalPending  ApprovalStatus = "pending"
	ApprovalApproved ApprovalStatus = "approved"
	ApprovalRejected ApprovalStatus = "rejected"
)

type Approval struct {
	ID         string
	RunID      string
	SessionID  string
	AccountID  string
	ToolCallID string
	Action     string
	Status     ApprovalStatus
	ResolvedBy string
	CreatedAt  time.Time
	ResolvedAt time.Time
	UpdatedAt  time.Time
}

func NewApproval(id, runID, sessionID, accountID, toolCallID, action string, now time.Time) (*Approval, error) {
	if anyBlank(id, runID, sessionID, accountID, toolCallID, action) {
		return nil, fmt.Errorf("%w: approval fields are required", ErrInvalid)
	}
	now = now.UTC()
	return &Approval{
		ID: id, RunID: runID, SessionID: sessionID, AccountID: accountID,
		ToolCallID: toolCallID, Action: action, Status: ApprovalPending,
		CreatedAt: now, UpdatedAt: now,
	}, nil
}

func (a *Approval) Approve(actorID string, now time.Time) error {
	return a.resolve(ApprovalApproved, actorID, now)
}

func (a *Approval) Reject(actorID string, now time.Time) error {
	return a.resolve(ApprovalRejected, actorID, now)
}

func (a *Approval) resolve(status ApprovalStatus, actorID string, now time.Time) error {
	if a.Status != ApprovalPending {
		return ErrInvalidTransition
	}
	if strings.TrimSpace(actorID) == "" || actorID != a.AccountID {
		return fmt.Errorf("%w: approval actor must own the session", ErrInvalid)
	}
	now = now.UTC()
	a.Status = status
	a.ResolvedBy = actorID
	a.ResolvedAt = now
	a.UpdatedAt = now
	return nil
}

type AssetKind string

const (
	AssetDocument AssetKind = "document"
	AssetImage    AssetKind = "image"
	AssetVideo    AssetKind = "video"
	AssetAudio    AssetKind = "audio"
	AssetData     AssetKind = "data"
	AssetFile     AssetKind = "file"
)

func (k AssetKind) Valid() bool {
	switch k {
	case AssetDocument, AssetImage, AssetVideo, AssetAudio, AssetData, AssetFile:
		return true
	default:
		return false
	}
}

type AssetOrigin string

const (
	AssetOriginUser     AssetOrigin = "user"
	AssetOriginAgent    AssetOrigin = "agent"
	AssetOriginModel    AssetOrigin = "model"
	AssetOriginWorkflow AssetOrigin = "workflow"
	AssetOriginLibrary  AssetOrigin = "library"
)

type Asset struct {
	ID             string
	SessionID      string
	AccountID      string
	Name           string
	Kind           AssetKind
	Origin         AssetOrigin
	SourceRunID    string
	CurrentVersion int
	LibrarySavedAt time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Versions       []AssetVersion
}

type AssetVersion struct {
	ID        string
	AssetID   string
	AccountID string
	Version   int
	MIMEType  string
	BlobKey   string
	SizeBytes int64
	Metadata  json.RawMessage
	CreatedAt time.Time
}

type LibraryFolder struct {
	ID        string
	AccountID string
	ParentID  string
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewLibraryFolder(id, accountID, parentID, name string, now time.Time) (*LibraryFolder, error) {
	if anyBlank(id, accountID, name) {
		return nil, fmt.Errorf("%w: invalid library folder", ErrInvalid)
	}
	now = now.UTC()
	return &LibraryFolder{ID: id, AccountID: accountID, ParentID: strings.TrimSpace(parentID), Name: strings.TrimSpace(name), CreatedAt: now, UpdatedAt: now}, nil
}

func NewAsset(id, sessionID, accountID, name string, kind AssetKind, origin AssetOrigin, now time.Time) (*Asset, error) {
	if anyBlank(id, accountID, name) || (!kind.Valid()) || strings.TrimSpace(string(origin)) == "" {
		return nil, fmt.Errorf("%w: invalid asset", ErrInvalid)
	}
	now = now.UTC()
	return &Asset{
		ID: id, SessionID: sessionID, AccountID: accountID, Name: strings.TrimSpace(name),
		Kind: kind, Origin: origin, CreatedAt: now, UpdatedAt: now,
	}, nil
}

func (a *Asset) AppendVersion(id, mimeType, blobKey string, sizeBytes int64, now time.Time) (AssetVersion, error) {
	if anyBlank(id, mimeType, blobKey) || sizeBytes < 0 {
		return AssetVersion{}, fmt.Errorf("%w: invalid asset version", ErrInvalid)
	}
	now = now.UTC()
	version := AssetVersion{
		ID: id, AssetID: a.ID, AccountID: a.AccountID, Version: a.CurrentVersion + 1,
		MIMEType: mimeType, BlobKey: blobKey, SizeBytes: sizeBytes, CreatedAt: now,
	}
	a.Versions = append(a.Versions, version)
	a.CurrentVersion = version.Version
	a.UpdatedAt = now
	return version, nil
}

type FlowNodeType string

const (
	FlowNodeStage     FlowNodeType = "stage"
	FlowNodePlan      FlowNodeType = "plan"
	FlowNodeOperation FlowNodeType = "operation"
	FlowNodeAsset     FlowNodeType = "asset"
)

func (t FlowNodeType) Valid() bool {
	switch t {
	case FlowNodeStage, FlowNodePlan, FlowNodeOperation, FlowNodeAsset:
		return true
	default:
		return false
	}
}

type FlowNode struct {
	ID             string
	SessionID      string
	AccountID      string
	Type           FlowNodeType
	Title          string
	Body           string
	AssetID        string
	AssetVersionID string
	AssetVersion   int
	RunID          string
	PositionX      float64
	PositionY      float64
	SortOrder      int
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func NewFlowNode(id, sessionID, accountID string, nodeType FlowNodeType, title string, sortOrder int, now time.Time) (*FlowNode, error) {
	if anyBlank(id, sessionID, accountID, title) || !nodeType.Valid() || sortOrder < 0 {
		return nil, fmt.Errorf("%w: invalid flow node", ErrInvalid)
	}
	now = now.UTC()
	return &FlowNode{
		ID: id, SessionID: sessionID, AccountID: accountID, Type: nodeType,
		Title: strings.TrimSpace(title), SortOrder: sortOrder, CreatedAt: now, UpdatedAt: now,
	}, nil
}

type FlowEdge struct {
	ID           string
	SessionID    string
	AccountID    string
	SourceNodeID string
	TargetNodeID string
	Label        string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func NewFlowEdge(id, sessionID, accountID, sourceNodeID, targetNodeID string, now time.Time) (*FlowEdge, error) {
	if anyBlank(id, sessionID, accountID, sourceNodeID, targetNodeID) || sourceNodeID == targetNodeID {
		return nil, fmt.Errorf("%w: invalid flow edge", ErrInvalid)
	}
	now = now.UTC()
	return &FlowEdge{
		ID: id, SessionID: sessionID, AccountID: accountID,
		SourceNodeID: sourceNodeID, TargetNodeID: targetNodeID,
		CreatedAt: now, UpdatedAt: now,
	}, nil
}

func anyBlank(values ...string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return true
		}
	}
	return false
}
