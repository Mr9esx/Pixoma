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

	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"

	"github.com/Mr9esx/Pixoma/internal/platform/blob"
	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

const (
	EventRunStarted              = "RUN_STARTED"
	EventTextMessageStart        = "TEXT_MESSAGE_START"
	EventTextMessageContent      = "TEXT_MESSAGE_CONTENT"
	EventTextMessageEnd          = "TEXT_MESSAGE_END"
	EventReasoningStart          = "REASONING_START"
	EventReasoningMessageStart   = "REASONING_MESSAGE_START"
	EventReasoningMessageContent = "REASONING_MESSAGE_CONTENT"
	EventReasoningMessageEnd     = "REASONING_MESSAGE_END"
	EventReasoningEnd            = "REASONING_END"
	EventToolCallStart           = "TOOL_CALL_START"
	EventToolCallArgs            = "TOOL_CALL_ARGS"
	EventToolCallEnd             = "TOOL_CALL_END"
	EventToolCallResult          = "TOOL_CALL_RESULT"
	EventModelRequestStarted     = "MODEL_REQUEST_STARTED"
	EventModelFirstToken         = "MODEL_FIRST_TOKEN"
	EventModelRequestFinished    = "MODEL_REQUEST_FINISHED"
	EventModelRequestFailed      = "MODEL_REQUEST_FAILED"
	EventContextCompacted        = "CONTEXT_COMPACTED"
	EventAssetCreated            = "ASSET_CREATED"
	EventFlowUpdated             = "FLOW_UPDATED"
	EventApprovalRequired        = "APPROVAL_REQUIRED"
	EventApprovalResolved        = "APPROVAL_RESOLVED"
	EventRunFinished             = "RUN_FINISHED"
)

var ErrApprovalRequired = errors.New("studio: approval required")

type AgentRepository interface {
	domain.Repository
}

type AgentRequest struct {
	Run      *domain.Run
	Session  *domain.Session
	UserText string
	History  []*schema.Message
	// HistoryMessageIDs is aligned with History. Each entry is the last
	// durable message that may safely be summarized before that model message;
	// event-only tool messages inherit their run trigger boundary.
	HistoryMessageIDs  []string
	ContextSummary     string
	SaveContextSummary func(context.Context, string, string) error
	Skills             []domain.Skill
	AvailableSkills    []domain.RunSkill
	Assets             []*domain.Asset
	Approvals          []*domain.Approval
}

type GeneratedAsset struct {
	ActionID string
	Name     string
	Kind     domain.AssetKind
	Origin   domain.AssetOrigin
	MIMEType string
	Content  []byte
	Metadata map[string]any
}

type FlowNodeInput struct {
	ActionID       string
	Type           domain.FlowNodeType
	Title          string
	Body           string
	AssetID        string
	AssetVersionID string
	SortOrder      int
}

type AgentSink interface {
	Emit(ctx context.Context, eventType string, payload any) error
	AssistantMessage(ctx context.Context, text string) (*domain.Message, error)
	CreateAsset(ctx context.Context, input GeneratedAsset) (*domain.Asset, error)
	CreateFlowNode(ctx context.Context, input FlowNodeInput) (*domain.FlowNode, error)
	CreateFlowEdge(ctx context.Context, sourceNodeID, targetNodeID, label string) (*domain.FlowEdge, error)
	RequestApproval(ctx context.Context, toolCallID, action, description string) (*domain.Approval, error)
}

// AssistantStreamSink is optional so existing workflow/test sinks can keep the
// small AgentSink contract while the production persistence writer can publish
// text deltas without creating one database message per delta.
type AssistantStreamSink interface {
	BeginAssistantMessage(ctx context.Context) (string, error)
	AppendAssistantMessage(ctx context.Context, messageID, delta string) error
	EndAssistantMessage(ctx context.Context, messageID, text string) (*domain.Message, error)
}

type AgentEngine interface {
	Execute(ctx context.Context, request AgentRequest, sink AgentSink) error
}

type AgentExecutorOptions struct {
	Repo   AgentRepository
	Blob   blob.Store
	Engine AgentEngine
	Events EventStream
	IDs    func() string
	Now    func() time.Time
}

type AgentExecutor struct {
	repo   AgentRepository
	blob   blob.Store
	engine AgentEngine
	events EventStream
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
	return &AgentExecutor{repo: options.Repo, blob: options.Blob, engine: options.Engine, events: options.Events, ids: ids, now: now}
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
	transcriptData, err := e.repo.ListSessionTranscript(ctx, run.AccountID, run.SessionID)
	if err != nil {
		return err
	}
	history, err := ProjectModelHistoryWithBoundaries(
		transcriptData.Messages,
		transcriptData.Runs,
		transcriptData.Events,
		session.ContextSummaryThroughMessageID,
		run.TriggerMessageID,
	)
	if err != nil {
		return err
	}
	text, err := messageText(message.ContentJSON)
	if err != nil {
		return err
	}
	sink := &executionWriter{executor: e, run: run, events: e.events}
	approvals, err := e.repo.ListApprovals(ctx, run.AccountID, run.ID)
	if err != nil {
		return err
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
	saveSummary := func(summaryCtx context.Context, throughMessageID, summary string) error {
		if err := session.UpdateContextSummary(summary, throughMessageID, e.now()); err != nil {
			return err
		}
		return e.repo.UpdateSession(summaryCtx, session)
	}
	err = e.engine.Execute(ctx, AgentRequest{
		Run: run, Session: session, UserText: text,
		History: history.Messages, HistoryMessageIDs: history.BoundaryMessageIDs,
		ContextSummary: session.ContextSummary, SaveContextSummary: saveSummary,
		Skills: skills, AvailableSkills: run.SkillSnapshot, Assets: assets, Approvals: approvals,
	}, sink)
	if flushErr := sink.FlushOutput(ctx); flushErr != nil {
		return flushErr
	}
	if errors.Is(err, ErrApprovalRequired) {
		if waitErr := run.WaitForApproval(e.now()); waitErr != nil {
			return waitErr
		}
		if updateErr := e.repo.UpdateRun(ctx, run); updateErr != nil {
			return updateErr
		}
	}
	return err
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
	if run.SkillSnapshot != nil {
		byID := make(map[string]domain.RunSkill, len(run.SkillSnapshot))
		for _, skill := range run.SkillSnapshot {
			byID[skill.ID] = skill
		}
		selected := make([]domain.Skill, 0, len(run.SkillIDs))
		for _, id := range run.SkillIDs {
			skill, ok := byID[id]
			if !ok {
				return nil, fmt.Errorf("%w: selected Skill is absent from Run snapshot", domain.ErrNotFound)
			}
			selected = append(selected, domain.Skill{ID: skill.ID, AccountID: run.AccountID, Name: skill.Name, Description: skill.Description, Prompt: skill.Prompt, Enabled: true})
		}
		return selected, nil
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
	executor             *AgentExecutor
	run                  *domain.Run
	events               EventStream
	sequence             uint64
	pendingTextID        string
	pendingText          string
	textLastFlushed      time.Time
	pendingReasoningID   string
	pendingReasoning     string
	reasoningLastFlushed time.Time
	progress             *domain.RunProgress
	progressLastSaved    time.Time
	progressDirty        bool
	progressBytes        int
	toolCalls            map[string]runProgressToolCall
}

const runProgressPersistInterval = 200 * time.Millisecond
const textBatchPersistInterval = 50 * time.Millisecond

type runProgressToolCall struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Args    string `json:"args,omitempty"`
	Result  string `json:"result,omitempty"`
	IsError bool   `json:"is_error,omitempty"`
}

func (w *executionWriter) Emit(ctx context.Context, eventType string, payload any) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if eventType == EventReasoningMessageContent {
		return w.bufferReasoning(ctx, payload)
	}
	if eventType != EventTextMessageContent {
		if err := w.FlushOutput(ctx); err != nil {
			return err
		}
	}
	return w.emitStored(ctx, eventType, payload)
}

func (w *executionWriter) emitStored(ctx context.Context, eventType string, payload any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	event := &domain.Event{
		ID: w.executor.ids(), RunID: w.run.ID, SessionID: w.run.SessionID,
		AccountID: w.run.AccountID, Type: eventType,
		Payload: raw, CreatedAt: w.executor.now().UTC(),
	}
	stored, err := w.executor.repo.AppendRunEvent(ctx, event)
	if err != nil {
		return err
	}
	w.sequence = stored.Sequence
	w.updateProgressFromEvent(ctx, eventType, raw, stored.Sequence)
	if w.events == nil {
		return nil
	}
	return w.events.Publish(ctx, LiveEvent{
		RunID: w.run.ID, Sequence: stored.Sequence, Type: event.Type, Payload: raw,
	})
}

func (w *executionWriter) AssistantMessage(ctx context.Context, text string) (*domain.Message, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, fmt.Errorf("%w: assistant text is required", domain.ErrInvalid)
	}
	messageID, err := w.BeginAssistantMessage(ctx)
	if err != nil {
		return nil, err
	}
	if err := w.AppendAssistantMessage(ctx, messageID, text); err != nil {
		return nil, err
	}
	return w.EndAssistantMessage(ctx, messageID, text)
}

func (w *executionWriter) BeginAssistantMessage(ctx context.Context) (string, error) {
	messageID := w.executor.ids()
	if err := w.Emit(ctx, EventTextMessageStart, map[string]any{"message_id": messageID, "role": "assistant"}); err != nil {
		return "", err
	}
	return messageID, nil
}

func (w *executionWriter) AppendAssistantMessage(ctx context.Context, messageID, delta string) error {
	if strings.TrimSpace(delta) == "" {
		return nil
	}
	if w.pendingReasoning != "" {
		if err := w.FlushOutput(ctx); err != nil {
			return err
		}
	}
	if w.pendingTextID != "" && w.pendingTextID != messageID {
		if err := w.FlushOutput(ctx); err != nil {
			return err
		}
	}
	w.ensureProgress()
	w.progress.AssistantMessageID = messageID
	w.progress.AssistantText += delta
	w.progressDirty = true
	w.progressBytes += len(delta)
	w.pendingTextID = messageID
	w.pendingText += delta
	if w.textLastFlushed.IsZero() || w.executor.now().Sub(w.textLastFlushed) >= textBatchPersistInterval || len(w.pendingText) >= 1024 {
		return w.FlushOutput(ctx)
	}
	return nil
}

func (w *executionWriter) FlushOutput(ctx context.Context) error {
	if w.pendingText != "" {
		if err := w.emitStored(ctx, EventTextMessageContent, map[string]any{
			"message_id": w.pendingTextID, "delta": w.pendingText,
		}); err != nil {
			return err
		}
		w.pendingText = ""
		w.pendingTextID = ""
		w.textLastFlushed = w.executor.now()
	}
	if w.pendingReasoning != "" {
		if err := w.emitStored(ctx, EventReasoningMessageContent, map[string]any{
			"message_id": w.pendingReasoningID, "delta": w.pendingReasoning,
		}); err != nil {
			return err
		}
		w.pendingReasoning = ""
		w.pendingReasoningID = ""
		w.reasoningLastFlushed = w.executor.now()
	}
	return nil
}

func (w *executionWriter) bufferReasoning(ctx context.Context, payload any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	var value struct {
		MessageID string `json:"message_id"`
		Delta     string `json:"delta"`
	}
	if err := json.Unmarshal(raw, &value); err != nil {
		return err
	}
	if value.Delta == "" {
		return nil
	}
	if (w.pendingReasoningID != "" && w.pendingReasoningID != value.MessageID) || w.pendingText != "" {
		if err := w.FlushOutput(ctx); err != nil {
			return err
		}
	}
	w.pendingReasoningID = value.MessageID
	w.pendingReasoning += value.Delta
	if w.reasoningLastFlushed.IsZero() || w.executor.now().Sub(w.reasoningLastFlushed) >= textBatchPersistInterval || len(w.pendingReasoning) >= 1024 {
		return w.FlushOutput(ctx)
	}
	return nil
}

func (w *executionWriter) EndAssistantMessage(ctx context.Context, messageID, text string) (*domain.Message, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, fmt.Errorf("%w: assistant text is required", domain.ErrInvalid)
	}
	if err := w.FlushOutput(ctx); err != nil {
		return nil, err
	}
	content, err := json.Marshal([]MessagePart{{Type: "text", Text: text}})
	if err != nil {
		return nil, err
	}
	message := &domain.Message{
		ID: messageID, SessionID: w.run.SessionID, AccountID: w.run.AccountID,
		RunID: w.run.ID, Role: domain.MessageRoleAssistant, ContentJSON: content,
		CreatedAt: w.executor.now().UTC(),
	}
	if err := w.executor.repo.AppendMessage(ctx, message); err != nil {
		return nil, err
	}
	if err := w.Emit(ctx, EventTextMessageEnd, map[string]any{
		"message_id": messageID,
		"content":    text,
	}); err != nil {
		return nil, err
	}
	w.deleteProgress(ctx)
	return message, nil
}

func (w *executionWriter) ensureProgress() {
	if w.progress != nil {
		return
	}
	w.progress = &domain.RunProgress{
		RunID: w.run.ID, SessionID: w.run.SessionID, AccountID: w.run.AccountID,
	}
	if w.toolCalls == nil {
		w.toolCalls = make(map[string]runProgressToolCall)
	}
}

func (w *executionWriter) updateProgressFromEvent(ctx context.Context, eventType string, raw json.RawMessage, sequence uint64) {
	if eventType == EventRunFinished {
		w.deleteProgress(ctx)
		return
	}
	var value map[string]any
	if json.Unmarshal(raw, &value) != nil {
		return
	}
	w.ensureProgress()
	w.progress.LastSequence = sequence
	switch eventType {
	case EventTextMessageStart:
		if messageID, _ := value["message_id"].(string); messageID != "" {
			w.progress.AssistantMessageID = messageID
		}
	case EventReasoningMessageStart:
		if w.progress.ReasoningText != "" {
			w.progress.ReasoningText += "\n"
		}
	case EventReasoningMessageContent:
		if delta, _ := value["delta"].(string); delta != "" {
			w.progress.ReasoningText += delta
			w.progressBytes += len(delta)
		}
	case EventToolCallStart:
		callID, _ := value["tool_call_id"].(string)
		if callID != "" {
			w.toolCalls[callID] = runProgressToolCall{ID: callID, Name: stringValue(value, "tool_name")}
		}
	case EventToolCallArgs:
		callID, _ := value["tool_call_id"].(string)
		call := w.toolCalls[callID]
		call.ID = callID
		call.Args += stringValue(value, "delta")
		w.toolCalls[callID] = call
		w.progressBytes += len(stringValue(value, "delta"))
	case EventToolCallResult:
		callID, _ := value["tool_call_id"].(string)
		call := w.toolCalls[callID]
		call.ID = callID
		call.Result = stringValue(value, "content")
		call.IsError, _ = value["is_error"].(bool)
		w.toolCalls[callID] = call
	}
	w.progressDirty = true
	w.maybePersistProgress(ctx, eventType == EventTextMessageStart || eventType == EventReasoningMessageStart || eventType == EventToolCallStart || eventType == EventToolCallEnd)
}

func stringValue(value map[string]any, key string) string {
	result, _ := value[key].(string)
	return result
}

func (w *executionWriter) maybePersistProgress(ctx context.Context, force bool) {
	if w.progress == nil || !w.progressDirty {
		return
	}
	now := w.executor.now().UTC()
	if !force && !w.progressLastSaved.IsZero() && now.Sub(w.progressLastSaved) < runProgressPersistInterval && w.progressBytes < 1024 {
		return
	}
	if len(w.toolCalls) > 0 {
		calls, err := json.Marshal(w.toolCalls)
		if err != nil {
			return
		}
		w.progress.ToolCallsJSON = calls
	}
	w.progress.UpdatedAt = now
	if err := w.executor.repo.UpsertRunProgress(ctx, w.progress); err != nil {
		return
	}
	w.progressLastSaved = now
	w.progressDirty = false
	w.progressBytes = 0
}

func (w *executionWriter) deleteProgress(ctx context.Context) {
	if w.progress == nil {
		return
	}
	_ = w.executor.repo.DeleteRunProgress(ctx, w.run.AccountID, w.run.ID)
	w.progress = nil
	w.progressDirty = false
	w.progressBytes = 0
}

func (w *executionWriter) CreateAsset(ctx context.Context, input GeneratedAsset) (*domain.Asset, error) {
	if len(input.Content) == 0 || strings.TrimSpace(input.MIMEType) == "" {
		return nil, fmt.Errorf("%w: generated asset content and MIME type are required", domain.ErrInvalid)
	}
	now := w.executor.now().UTC()
	assetID := w.executor.ids()
	if input.ActionID != "" {
		assetID = uuid.NewSHA1(uuid.NameSpaceOID, []byte(w.run.ID+"\x00asset\x00"+input.ActionID)).String()
		if existing, err := w.executor.repo.GetAsset(ctx, w.run.AccountID, assetID); err == nil {
			return existing, nil
		} else if !errors.Is(err, domain.ErrNotFound) {
			return nil, err
		}
	}
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
	versionID := w.executor.ids()
	if input.ActionID != "" {
		versionID = uuid.NewSHA1(uuid.NameSpaceOID, []byte(w.run.ID+"\x00version\x00"+input.ActionID)).String()
	}
	version, err := asset.AppendVersion(versionID, input.MIMEType, ref.Key, ref.Size, now)
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
		if input.ActionID != "" && errors.Is(err, domain.ErrAlreadyExists) {
			return w.executor.repo.GetAsset(ctx, w.run.AccountID, assetID)
		}
		return nil, err
	}
	if err := w.Emit(ctx, EventAssetCreated, map[string]any{"asset_id": asset.ID, "name": asset.Name, "kind": asset.Kind}); err != nil {
		return nil, err
	}
	return asset, nil
}

func (w *executionWriter) CreateFlowNode(ctx context.Context, input FlowNodeInput) (*domain.FlowNode, error) {
	nodeID := w.executor.ids()
	if input.ActionID != "" {
		nodeID = uuid.NewSHA1(uuid.NameSpaceOID, []byte(w.run.ID+"\x00flow-node\x00"+input.ActionID)).String()
	}
	node, err := domain.NewFlowNode(nodeID, w.run.SessionID, w.run.AccountID, input.Type, input.Title, input.SortOrder, w.executor.now())
	if err != nil {
		return nil, err
	}
	node.Body = input.Body
	node.AssetID = input.AssetID
	node.AssetVersionID = input.AssetVersionID
	if node.AssetID != "" {
		asset, err := w.executor.repo.GetAsset(ctx, w.run.AccountID, node.AssetID)
		if err != nil {
			return nil, err
		}
		if len(asset.Versions) == 0 {
			return nil, fmt.Errorf("%w: flow asset has no versions", domain.ErrInvalid)
		}
		if node.AssetVersionID == "" {
			node.AssetVersionID = asset.Versions[len(asset.Versions)-1].ID
		}
		assetVersion, ok := assetVersionByID(asset, node.AssetVersionID)
		if !ok {
			return nil, fmt.Errorf("%w: flow asset version is not available", domain.ErrInvalid)
		}
		node.AssetVersion = assetVersion.Version
	}
	node.RunID = w.run.ID
	if err := w.executor.repo.SaveFlowNode(ctx, node); err != nil {
		return nil, err
	}
	if err := w.Emit(ctx, EventFlowUpdated, map[string]any{"node_id": node.ID, "action": "created"}); err != nil {
		return nil, err
	}
	return node, nil
}

func assetVersionByID(asset *domain.Asset, versionID string) (domain.AssetVersion, bool) {
	for _, version := range asset.Versions {
		if version.ID == versionID {
			return version, true
		}
	}
	return domain.AssetVersion{}, false
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

func (w *executionWriter) RequestApproval(ctx context.Context, toolCallID, action, description string) (*domain.Approval, error) {
	approval, err := domain.NewApproval(
		w.executor.ids(), w.run.ID, w.run.SessionID, w.run.AccountID,
		toolCallID, action, description, w.executor.now(),
	)
	if err != nil {
		return nil, err
	}
	if err := w.executor.repo.CreateApproval(ctx, approval); err != nil {
		return nil, err
	}
	if err := w.Emit(ctx, EventApprovalRequired, map[string]any{
		"approval_id": approval.ID, "tool_call_id": toolCallID, "action": action, "description": description,
	}); err != nil {
		return nil, err
	}
	return approval, nil
}

func messageText(raw json.RawMessage) (string, error) {
	var parts []MessagePart
	if err := json.Unmarshal(raw, &parts); err != nil {
		return "", fmt.Errorf("studio: decode message content: %w", err)
	}
	return messagePartsText(parts)
}

func messagePartsText(parts []MessagePart) (string, error) {
	var text strings.Builder
	for _, part := range parts {
		switch part.Type {
		case "text":
			text.WriteString(part.Text)
		case "skill_ref":
			if part.SkillID == "" || part.Name == "" {
				return "", fmt.Errorf("studio: invalid Skill reference")
			}
			text.WriteString("「" + part.Name + "」Skill")
		case "asset_ref":
			if part.AssetID == "" || part.AssetVersionID == "" || part.Name == "" {
				return "", fmt.Errorf("studio: invalid asset reference")
			}
			text.WriteString("「" + part.Name + "」资产")
		case "reasoning", "image", "file":
		default:
			return "", fmt.Errorf("studio: unsupported message part %q", part.Type)
		}
	}
	return text.String(), nil
}

func historyBeforeMessage(messages []*domain.Message, currentMessageID, summaryThroughMessageID string) ([]*schema.Message, []string, error) {
	history := make([]*schema.Message, 0, len(messages))
	ids := make([]string, 0, len(messages))
	boundaryFound := strings.TrimSpace(summaryThroughMessageID) == ""
	for _, message := range messages {
		if message == nil {
			continue
		}
		if !boundaryFound {
			if message.ID == summaryThroughMessageID {
				boundaryFound = true
			}
			continue
		}
		if message.ID == currentMessageID {
			break
		}
		converted, err := schemaMessageFromDomain(message)
		if err != nil {
			return nil, nil, err
		}
		if converted == nil {
			continue
		}
		history = append(history, converted)
		ids = append(ids, message.ID)
	}
	return history, ids, nil
}

func schemaMessageFromDomain(message *domain.Message) (*schema.Message, error) {
	text, err := messageText(message.ContentJSON)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(text) == "" {
		return nil, nil
	}
	role := schema.User
	switch message.Role {
	case domain.MessageRoleAssistant:
		role = schema.Assistant
	case domain.MessageRoleSystem:
		role = schema.System
	case domain.MessageRoleUser:
		role = schema.User
	default:
		return nil, nil
	}
	return &schema.Message{Role: role, Content: text}, nil
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
