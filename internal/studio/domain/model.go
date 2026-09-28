package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

var (
	ErrInvalid           = errors.New("studio: invalid value")
	ErrInvalidTransition = errors.New("studio: invalid state transition")
	ErrNotFound          = errors.New("studio: not found")
	ErrAlreadyExists     = errors.New("studio: already exists")
	ErrConflict          = errors.New("studio: content changed")
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
	ProjectID                      string
	Title                          string
	PermissionMode                 PermissionMode
	ModelConfigID                  string
	ContextSummary                 string
	ContextSummaryThroughMessageID string
	Status                         SessionStatus
	CreatedAt                      time.Time
	UpdatedAt                      time.Time
}

type Project struct {
	ID        string
	AccountID string
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewProject(id, accountID, name string, now time.Time) (*Project, error) {
	name = strings.TrimSpace(name)
	if anyBlank(id, accountID, name) || len([]rune(name)) > 80 {
		return nil, fmt.Errorf("%w: invalid project name", ErrInvalid)
	}
	return &Project{ID: id, AccountID: accountID, Name: name, CreatedAt: now.UTC(), UpdatedAt: now.UTC()}, nil
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
	RunQueued               RunStatus = "queued"
	RunRunning              RunStatus = "running"
	RunWaitingApproval      RunStatus = "waiting_approval"
	RunWaitingClarification RunStatus = "waiting_clarification"
	RunSucceeded            RunStatus = "succeeded"
	RunFailed               RunStatus = "failed"
	RunCancelled            RunStatus = "cancelled"
)

func (s RunStatus) Terminal() bool {
	return s == RunSucceeded || s == RunFailed || s == RunCancelled
}

type Run struct {
	ID               string
	RequestID        string
	SessionID        string
	AccountID        string
	TriggerMessageID string
	Status           RunStatus
	ModelConfigID    string
	Locale           string
	SkillIDs         []string
	SkillSnapshot    []RunSkill
	AssetIDs         []string
	AssetReferences  []AssetReference
	ErrorCode        string
	ErrorMessage     string
	CreatedAt        time.Time
	StartedAt        time.Time
	CompletedAt      time.Time
	UpdatedAt        time.Time
}

type RunSkill struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Prompt      string      `json:"prompt"`
	Files       []SkillFile `json:"files,omitempty"`
}

// RunProgress is the latest recoverable snapshot for a non-terminal Run.
// It complements the durable event ledger: high-frequency stream deltas can
// be restored without writing one database row per token.
type RunProgress struct {
	RunID              string
	SessionID          string
	AccountID          string
	AssistantMessageID string
	AssistantText      string
	ReasoningText      string
	ToolCallsJSON      json.RawMessage
	LastSequence       uint64
	UpdatedAt          time.Time
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

func (r *Run) WaitForClarification(now time.Time) error {
	if r.Status != RunRunning {
		return ErrInvalidTransition
	}
	r.Status = RunWaitingClarification
	r.UpdatedAt = now.UTC()
	return nil
}

func (r *Run) Resume(now time.Time) error {
	if r.Status != RunWaitingApproval && r.Status != RunWaitingClarification {
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
	if r.Status != RunQueued && r.Status != RunRunning && r.Status != RunWaitingApproval && r.Status != RunWaitingClarification {
		return ErrInvalidTransition
	}
	r.ErrorCode = strings.TrimSpace(code)
	r.ErrorMessage = strings.TrimSpace(message)
	r.finish(RunFailed, now)
	return nil
}

func (r *Run) Cancel(now time.Time) error {
	if r.Status != RunQueued && r.Status != RunRunning && r.Status != RunWaitingApproval && r.Status != RunWaitingClarification {
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

type ClarificationStatus string

const (
	ClarificationPending  ClarificationStatus = "pending"
	ClarificationAnswered ClarificationStatus = "answered"
	ClarificationSkipped  ClarificationStatus = "skipped"
)

type Clarification struct {
	ID         string
	RunID      string
	SessionID  string
	AccountID  string
	Question   string
	Options    []string
	Workflow   *WorkflowRequest
	Status     ClarificationStatus
	Selected   string
	Answer     string
	ResolvedBy string
	CreatedAt  time.Time
	ResolvedAt time.Time
	UpdatedAt  time.Time
}

type WorkflowRequest struct {
	ID              string               `json:"id"`
	Name            string               `json:"name"`
	Description     string               `json:"description,omitempty"`
	Preview         string               `json:"preview,omitempty"`
	InputSchema     json.RawMessage      `json:"input_schema"`
	SuggestedInputs map[string]any       `json:"suggested_inputs"`
	SubmittedInputs map[string]any       `json:"submitted_inputs,omitempty"`
	AnthropicOutput []json.RawMessage    `json:"anthropic_output,omitempty"`
	InputFields     []WorkflowInputField `json:"input_fields"`
}

func (w WorkflowRequest) ForClient() WorkflowRequest {
	w.AnthropicOutput = nil
	w.SubmittedInputs = nil
	return w
}

type WorkflowInputField struct {
	Key         string `json:"key"`
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Description string `json:"description,omitempty"`
}

func NewWorkflowClarification(id, runID, sessionID, accountID string, workflow WorkflowRequest, now time.Time) (*Clarification, error) {
	if anyBlank(id, runID, sessionID, accountID, workflow.ID, workflow.Name) || !json.Valid(workflow.InputSchema) {
		return nil, fmt.Errorf("%w: invalid workflow request", ErrInvalid)
	}
	now = now.UTC()
	return &Clarification{
		ID: id, RunID: runID, SessionID: sessionID, AccountID: accountID,
		Question: workflow.Name, Options: []string{}, Workflow: &workflow, Status: ClarificationPending,
		CreatedAt: now, UpdatedAt: now,
	}, nil
}

func (c *Clarification) ResolveWorkflow(actorID string, inputs map[string]any, skip bool, now time.Time) error {
	if c.Status != ClarificationPending {
		return ErrInvalidTransition
	}
	if c.Workflow == nil || strings.TrimSpace(actorID) == "" || actorID != c.AccountID {
		return fmt.Errorf("%w: invalid workflow response", ErrInvalid)
	}
	if skip {
		c.Selected = "skip"
		c.Answer = "已跳过"
		c.Status = ClarificationSkipped
	} else {
		if inputs == nil {
			return fmt.Errorf("%w: workflow inputs are required", ErrInvalid)
		}
		c.Selected = "submit"
		c.Answer = "已提交"
		c.Workflow.SubmittedInputs = inputs
		c.Status = ClarificationAnswered
	}
	now = now.UTC()
	c.ResolvedBy = actorID
	c.ResolvedAt = now
	c.UpdatedAt = now
	return nil
}

func NewClarification(id, runID, sessionID, accountID, question string, options []string, now time.Time) (*Clarification, error) {
	question = strings.TrimSpace(question)
	if anyBlank(id, runID, sessionID, accountID, question) || len(options) < 2 || len(options) > 5 || len([]rune(question)) > 300 {
		return nil, fmt.Errorf("%w: invalid clarification question", ErrInvalid)
	}
	seen := make(map[string]bool, len(options))
	cleaned := make([]string, 0, len(options))
	for _, option := range options {
		option = strings.TrimSpace(option)
		if option == "" || option == "其他" || option == "其它" || strings.EqualFold(option, "other") || len([]rune(option)) > 100 || seen[option] {
			return nil, fmt.Errorf("%w: invalid clarification option", ErrInvalid)
		}
		seen[option] = true
		cleaned = append(cleaned, option)
	}
	now = now.UTC()
	return &Clarification{
		ID: id, RunID: runID, SessionID: sessionID, AccountID: accountID,
		Question: question, Options: cleaned, Status: ClarificationPending,
		CreatedAt: now, UpdatedAt: now,
	}, nil
}

func (c *Clarification) Resolve(actorID, selected, custom string, now time.Time) error {
	if c.Status != ClarificationPending {
		return ErrInvalidTransition
	}
	if strings.TrimSpace(actorID) == "" || actorID != c.AccountID {
		return fmt.Errorf("%w: clarification actor must own the session", ErrInvalid)
	}
	status := ClarificationAnswered
	if selected == "skip" {
		if strings.TrimSpace(custom) != "" {
			return fmt.Errorf("%w: invalid clarification skip", ErrInvalid)
		}
		c.Answer = ""
		status = ClarificationSkipped
	} else if selected == "other" {
		custom = strings.TrimSpace(custom)
		if custom == "" || len([]rune(custom)) > 1000 {
			return fmt.Errorf("%w: invalid custom clarification answer", ErrInvalid)
		}
		c.Answer = custom
	} else {
		index, err := strconv.Atoi(selected)
		if err != nil || index < 0 || index >= len(c.Options) || strings.TrimSpace(custom) != "" {
			return fmt.Errorf("%w: invalid clarification selection", ErrInvalid)
		}
		c.Answer = c.Options[index]
	}
	now = now.UTC()
	c.Selected = selected
	c.Status = status
	c.ResolvedBy = actorID
	c.ResolvedAt = now
	c.UpdatedAt = now
	return nil
}

const (
	ApprovalPending  ApprovalStatus = "pending"
	ApprovalApproved ApprovalStatus = "approved"
	ApprovalRejected ApprovalStatus = "rejected"
)

type Approval struct {
	ID          string
	RunID       string
	SessionID   string
	AccountID   string
	ToolCallID  string
	Action      string
	Description string
	Status      ApprovalStatus
	ResolvedBy  string
	CreatedAt   time.Time
	ResolvedAt  time.Time
	UpdatedAt   time.Time
}

func NewApproval(id, runID, sessionID, accountID, toolCallID, action, description string, now time.Time) (*Approval, error) {
	if anyBlank(id, runID, sessionID, accountID, toolCallID, action, description) {
		return nil, fmt.Errorf("%w: approval fields are required", ErrInvalid)
	}
	now = now.UTC()
	return &Approval{
		ID: id, RunID: runID, SessionID: sessionID, AccountID: accountID,
		ToolCallID: toolCallID, Action: action, Description: description, Status: ApprovalPending,
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

type LibraryCategory struct {
	ID        string
	AccountID string
	ParentID  string
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewLibraryCategory(id, accountID, parentID, name string, now time.Time) (*LibraryCategory, error) {
	if anyBlank(id, accountID, name) {
		return nil, fmt.Errorf("%w: invalid library category", ErrInvalid)
	}
	now = now.UTC()
	return &LibraryCategory{ID: id, AccountID: accountID, ParentID: strings.TrimSpace(parentID), Name: strings.TrimSpace(name), CreatedAt: now, UpdatedAt: now}, nil
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
