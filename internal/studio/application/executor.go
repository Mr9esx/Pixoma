package application

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Mr9esx/Pixoma/internal/platform/blob"
	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

const (
	EventRunStarted         = "RUN_STARTED"
	EventTextMessageStart   = "TEXT_MESSAGE_START"
	EventTextMessageContent = "TEXT_MESSAGE_CONTENT"
	EventTextMessageEnd     = "TEXT_MESSAGE_END"
	EventToolCallStart      = "TOOL_CALL_START"
	EventToolCallEnd        = "TOOL_CALL_END"
	EventAssetCreated       = "ASSET_CREATED"
	EventFlowUpdated        = "FLOW_UPDATED"
	EventApprovalRequired   = "APPROVAL_REQUIRED"
	EventApprovalResolved   = "APPROVAL_RESOLVED"
	EventRunFinished        = "RUN_FINISHED"
)

var ErrApprovalRequired = errors.New("studio: approval required")

type AgentRepository interface {
	domain.Repository
}

type AgentRequest struct {
	Run       *domain.Run
	Session   *domain.Session
	UserText  string
	Skills    []domain.Skill
	Assets    []*domain.Asset
	Approvals []*domain.Approval
}

type GeneratedAsset struct {
	Name     string
	Kind     domain.AssetKind
	Origin   domain.AssetOrigin
	MIMEType string
	Content  []byte
	Metadata map[string]any
}

type FlowNodeInput struct {
	Type      domain.FlowNodeType
	Title     string
	Body      string
	AssetID   string
	SortOrder int
}

type AgentSink interface {
	Emit(ctx context.Context, eventType string, payload any) error
	AssistantMessage(ctx context.Context, text string) (*domain.Message, error)
	CreateAsset(ctx context.Context, input GeneratedAsset) (*domain.Asset, error)
	CreateFlowNode(ctx context.Context, input FlowNodeInput) (*domain.FlowNode, error)
	CreateFlowEdge(ctx context.Context, sourceNodeID, targetNodeID, label string) (*domain.FlowEdge, error)
	RequestApproval(ctx context.Context, toolCallID, action string) (*domain.Approval, error)
}

type AgentEngine interface {
	Execute(ctx context.Context, request AgentRequest, sink AgentSink) error
}

type AgentExecutorOptions struct {
	Repo   AgentRepository
	Blob   blob.Store
	Engine AgentEngine
	IDs    func() string
	Now    func() time.Time
}

type AgentExecutor struct {
	repo   AgentRepository
	blob   blob.Store
	engine AgentEngine
	ids    func() string
	now    func() time.Time
}

func NewAgentExecutor(options AgentExecutorOptions) *AgentExecutor {
	ids := options.IDs
	if ids == nil {
		ids = uuid.NewString
	}
	now := options.Now
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &AgentExecutor{repo: options.Repo, blob: options.Blob, engine: options.Engine, ids: ids, now: now}
}

func (e *AgentExecutor) Execute(ctx context.Context, run *domain.Run) error {
	if e == nil || e.repo == nil || e.blob == nil || e.engine == nil || run == nil {
		return fmt.Errorf("studio: agent executor is not configured")
	}
	session, err := e.repo.GetSession(ctx, run.AccountID, run.SessionID)
	if err != nil {
		return err
	}
	message, err := e.repo.GetMessage(ctx, run.AccountID, run.TriggerMessageID)
	if err != nil {
		return err
	}
	text, err := messageText(message.ContentJSON)
	if err != nil {
		return err
	}
	sink := &executionWriter{executor: e, run: run}
	approvals, err := e.repo.ListApprovals(ctx, run.AccountID, run.ID)
	if err != nil {
		return err
	}
	events, err := e.repo.ListEventsAfter(ctx, run.AccountID, run.ID, 0, 200)
	if err != nil {
		return err
	}
	for _, event := range events {
		if event.Sequence > sink.sequence {
			sink.sequence = event.Sequence
		}
	}
	skills, err := e.selectedSkills(ctx, run)
	if err != nil {
		return err
	}
	assetReferences := run.AssetReferences
	if len(assetReferences) == 0 {
		assetReferences = make([]domain.AssetReference, 0, len(run.AssetIDs))
		for _, assetID := range run.AssetIDs {
			assetReferences = append(assetReferences, domain.AssetReference{AssetID: assetID})
		}
	}
	assets := make([]*domain.Asset, 0, len(assetReferences))
	for _, reference := range assetReferences {
		asset, err := e.repo.GetAsset(ctx, run.AccountID, reference.AssetID)
		if err != nil {
			return err
		}
		if reference.AssetVersionID != "" {
			if err := retainAssetVersion(asset, reference.AssetVersionID); err != nil {
				return err
			}
		}
		assets = append(assets, asset)
	}
	return e.engine.Execute(ctx, AgentRequest{Run: run, Session: session, UserText: text, Skills: skills, Assets: assets, Approvals: approvals}, sink)
}

func retainAssetVersion(asset *domain.Asset, versionID string) error {
	for _, version := range asset.Versions {
		if version.ID == versionID {
			asset.CurrentVersion = version.Version
			asset.Versions = []domain.AssetVersion{version}
			return nil
		}
	}
	return fmt.Errorf("%w: run asset version is no longer available", domain.ErrNotFound)
}

func (e *AgentExecutor) selectedSkills(ctx context.Context, run *domain.Run) ([]domain.Skill, error) {
	if len(run.SkillIDs) == 0 {
		return nil, nil
	}
	available, err := e.repo.ListSkills(ctx, run.AccountID)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]*domain.Skill, len(available))
	for _, skill := range available {
		byID[skill.ID] = skill
	}
	selected := make([]domain.Skill, 0, len(run.SkillIDs))
	for _, id := range run.SkillIDs {
		skill, ok := byID[id]
		if !ok {
			return nil, fmt.Errorf("%w: selected Skill no longer exists", domain.ErrNotFound)
		}
		selected = append(selected, *skill)
	}
	return selected, nil
}

type executionWriter struct {
	executor *AgentExecutor
	run      *domain.Run
	sequence uint64
}

func (w *executionWriter) Emit(ctx context.Context, eventType string, payload any) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	w.sequence++
	event := &domain.Event{
		ID: w.executor.ids(), RunID: w.run.ID, SessionID: w.run.SessionID,
		AccountID: w.run.AccountID, Sequence: w.sequence, Type: eventType,
		Payload: raw, CreatedAt: w.executor.now().UTC(),
	}
	return w.executor.repo.AppendEvent(ctx, event)
}

func (w *executionWriter) AssistantMessage(ctx context.Context, text string) (*domain.Message, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, fmt.Errorf("%w: assistant text is required", domain.ErrInvalid)
	}
	content, err := json.Marshal([]messagePart{{Type: "text", Text: text}})
	if err != nil {
		return nil, err
	}
	message := &domain.Message{
		ID: w.executor.ids(), SessionID: w.run.SessionID, AccountID: w.run.AccountID,
		RunID: w.run.ID, Role: domain.MessageRoleAssistant, ContentJSON: content,
		CreatedAt: w.executor.now().UTC(),
	}
	if err := w.executor.repo.AppendMessage(ctx, message); err != nil {
		return nil, err
	}
	if err := w.Emit(ctx, EventTextMessageStart, map[string]any{"message_id": message.ID, "role": "assistant"}); err != nil {
		return nil, err
	}
	if err := w.Emit(ctx, EventTextMessageContent, map[string]any{"message_id": message.ID, "delta": text}); err != nil {
		return nil, err
	}
	if err := w.Emit(ctx, EventTextMessageEnd, map[string]any{"message_id": message.ID}); err != nil {
		return nil, err
	}
	return message, nil
}

func (w *executionWriter) CreateAsset(ctx context.Context, input GeneratedAsset) (*domain.Asset, error) {
	if len(input.Content) == 0 || strings.TrimSpace(input.MIMEType) == "" {
		return nil, fmt.Errorf("%w: generated asset content and MIME type are required", domain.ErrInvalid)
	}
	now := w.executor.now().UTC()
	assetID := w.executor.ids()
	asset, err := domain.NewAsset(assetID, w.run.SessionID, w.run.AccountID, input.Name, input.Kind, input.Origin, now)
	if err != nil {
		return nil, err
	}
	asset.SourceRunID = w.run.ID
	extension := extensionForMIME(input.MIMEType)
	key := filepath.ToSlash(filepath.Join("studio", w.run.AccountID, w.run.SessionID, assetID, "v1"+extension))
	ref, err := w.executor.blob.Put(ctx, key, bytes.NewReader(input.Content), blob.PutOptions{MIME: input.MIMEType})
	if err != nil {
		return nil, err
	}
	version, err := asset.AppendVersion(w.executor.ids(), input.MIMEType, ref.Key, ref.Size, now)
	if err != nil {
		return nil, err
	}
	if len(input.Metadata) > 0 {
		version.Metadata, err = json.Marshal(input.Metadata)
		if err != nil {
			return nil, err
		}
		asset.Versions[len(asset.Versions)-1] = version
	}
	if err := w.executor.repo.CreateAsset(ctx, asset); err != nil {
		return nil, err
	}
	if err := w.Emit(ctx, EventAssetCreated, map[string]any{"asset_id": asset.ID, "name": asset.Name, "kind": asset.Kind}); err != nil {
		return nil, err
	}
	return asset, nil
}

func (w *executionWriter) CreateFlowNode(ctx context.Context, input FlowNodeInput) (*domain.FlowNode, error) {
	node, err := domain.NewFlowNode(w.executor.ids(), w.run.SessionID, w.run.AccountID, input.Type, input.Title, input.SortOrder, w.executor.now())
	if err != nil {
		return nil, err
	}
	node.Body = input.Body
	node.AssetID = input.AssetID
	node.RunID = w.run.ID
	if err := w.executor.repo.SaveFlowNode(ctx, node); err != nil {
		return nil, err
	}
	if err := w.Emit(ctx, EventFlowUpdated, map[string]any{"node_id": node.ID, "action": "created"}); err != nil {
		return nil, err
	}
	return node, nil
}

func (w *executionWriter) CreateFlowEdge(ctx context.Context, sourceNodeID, targetNodeID, label string) (*domain.FlowEdge, error) {
	edge, err := domain.NewFlowEdge(w.executor.ids(), w.run.SessionID, w.run.AccountID, sourceNodeID, targetNodeID, w.executor.now())
	if err != nil {
		return nil, err
	}
	edge.Label = label
	if err := w.executor.repo.SaveFlowEdge(ctx, edge); err != nil {
		return nil, err
	}
	if err := w.Emit(ctx, EventFlowUpdated, map[string]any{"edge_id": edge.ID, "action": "created"}); err != nil {
		return nil, err
	}
	return edge, nil
}

func (w *executionWriter) RequestApproval(ctx context.Context, toolCallID, action string) (*domain.Approval, error) {
	approval, err := domain.NewApproval(
		w.executor.ids(), w.run.ID, w.run.SessionID, w.run.AccountID,
		toolCallID, action, w.executor.now(),
	)
	if err != nil {
		return nil, err
	}
	if err := w.executor.repo.CreateApproval(ctx, approval); err != nil {
		return nil, err
	}
	if err := w.run.WaitForApproval(w.executor.now()); err != nil {
		return nil, err
	}
	if err := w.executor.repo.UpdateRun(ctx, w.run); err != nil {
		return nil, err
	}
	if err := w.Emit(ctx, EventApprovalRequired, map[string]any{
		"approval_id": approval.ID, "tool_call_id": toolCallID, "action": action,
	}); err != nil {
		return nil, err
	}
	return approval, nil
}

func messageText(raw json.RawMessage) (string, error) {
	var parts []messagePart
	if err := json.Unmarshal(raw, &parts); err != nil {
		return "", fmt.Errorf("studio: decode message content: %w", err)
	}
	var text strings.Builder
	for _, part := range parts {
		if part.Type == "text" {
			text.WriteString(part.Text)
		}
	}
	return text.String(), nil
}

func extensionForMIME(mime string) string {
	switch strings.ToLower(strings.TrimSpace(mime)) {
	case "text/markdown":
		return ".md"
	case "image/svg+xml":
		return ".svg"
	case "image/png":
		return ".png"
	case "image/jpeg":
		return ".jpg"
	case "application/json":
		return ".json"
	default:
		return ".bin"
	}
}

var _ Executor = (*AgentExecutor)(nil)
