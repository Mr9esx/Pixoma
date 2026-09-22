package studio

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	setupapi "github.com/Mr9esx/Pixoma/internal/httpapi/setup"
	"github.com/Mr9esx/Pixoma/internal/platform/blob"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
	studioapp "github.com/Mr9esx/Pixoma/internal/studio/application"
	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

type Handler struct {
	Repo         domain.Repository
	Service      *studioapp.Service
	Runner       *studioapp.BackgroundRunner
	Approvals    *studioapp.ApprovalService
	Models       *studioapp.ModelConfigService
	Capabilities *studioapp.CapabilityConfigService
	Blob         blob.Store
}

func (h *Handler) Mount(r chi.Router) {
	r.Post("/agui", h.streamAGUI)
	r.Get("/sessions", h.listSessions)
	r.Post("/sessions", h.createSession)
	r.Get("/sessions/{sessionID}", h.getSession)
	r.Get("/sessions/{sessionID}/runs", h.listSessionRuns)
	r.Patch("/sessions/{sessionID}/flow", h.updateFlow)
	r.Post("/sessions/{sessionID}/flow/nodes", h.createFlowNode)
	r.Delete("/sessions/{sessionID}/flow/nodes/{nodeID}", h.deleteFlowNode)
	r.Post("/sessions/{sessionID}/flow/edges", h.createFlowEdge)
	r.Delete("/sessions/{sessionID}/flow/edges/{edgeID}", h.deleteFlowEdge)
	r.Post("/messages", h.sendMessage)
	r.Get("/runs/{runID}", h.getRun)
	r.Get("/runs/{runID}/events", h.listEvents)
	r.Post("/runs/{runID}/cancel", h.cancelRun)
	r.Post("/runs/{runID}/retry", h.retryRun)
	r.Post("/approvals/{approvalID}", h.resolveApproval)
	r.Get("/assets/{assetID}/content", h.assetContent)
	r.Post("/assets/text", h.createTextAsset)
	r.Patch("/assets/{assetID}/text", h.updateTextAsset)
	r.Post("/assets/upload", h.uploadAsset)
	r.Post("/assets/{assetID}/save-to-library", h.saveAssetToLibrary)
	r.Post("/sessions/{sessionID}/assets/import", h.importLibraryAsset)
	r.Get("/library/assets", h.listLibraryAssets)
	r.Get("/library/folders", h.listLibraryFolders)
	r.Post("/library/folders", h.createLibraryFolder)
	r.Get("/models", h.listModels)
	r.Post("/models", h.createModel)
	r.Post("/models/{modelID}/test", h.testModelConnection)
	r.Get("/skills", h.listSkills)
	r.Post("/skills", h.createSkill)
	r.Patch("/skills/{skillID}", h.updateSkill)
	r.Get("/connectors", h.listConnectors)
	r.Post("/connectors", h.createConnector)
	r.Patch("/connectors/{connectorID}", h.updateConnector)
	r.Post("/connectors/{connectorID}/probe", h.probeConnector)
	r.Get("/workflows", h.listAgentWorkflows)
	r.Patch("/workflows/{workflowID}", h.updateAgentWorkflow)
}

func (h *Handler) createTextAsset(w http.ResponseWriter, r *http.Request) {
	accountID, ok := accountID(w, r)
	if !ok {
		return
	}
	if h.Blob == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "资产存储服务不可用"})
		return
	}
	var body struct {
		SessionID string `json:"session_id"`
		Name      string `json:"name"`
		Content   string `json:"content"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求内容格式不正确"})
		return
	}
	asset, err := h.Service.CreateManualTextAsset(r.Context(), studioapp.CreateManualTextAssetInput{
		AccountID: accountID, SessionID: body.SessionID, Name: body.Name, Content: body.Content,
	}, h.Blob)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, assetsToViews([]*domain.Asset{asset})[0])
}

func (h *Handler) updateTextAsset(w http.ResponseWriter, r *http.Request) {
	accountID, ok := accountID(w, r)
	if !ok {
		return
	}
	if h.Blob == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "资产存储服务不可用"})
		return
	}
	var body struct {
		Content string `json:"content"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求内容格式不正确"})
		return
	}
	asset, err := h.Service.UpdateManualTextAsset(r.Context(), studioapp.UpdateManualTextAssetInput{
		AccountID: accountID, AssetID: chi.URLParam(r, "assetID"), Content: body.Content,
	}, h.Blob)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, assetsToViews([]*domain.Asset{asset})[0])
}

func (h *Handler) uploadAsset(w http.ResponseWriter, r *http.Request) {
	accountID, ok := accountID(w, r)
	if !ok {
		return
	}
	if h.Blob == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "资产存储服务不可用"})
		return
	}
	if err := r.ParseMultipartForm(50 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "上传文件读取失败"})
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请选择上传文件"})
		return
	}
	defer file.Close()
	asset, err := h.Service.UploadAsset(r.Context(), studioapp.UploadAssetInput{
		AccountID: accountID, SessionID: r.FormValue("session_id"), Name: header.Filename,
		MIMEType: header.Header.Get("Content-Type"), Content: file,
	}, h.Blob)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, assetsToViews([]*domain.Asset{asset})[0])
}

func (h *Handler) updateFlow(w http.ResponseWriter, r *http.Request) {
	accountID, ok := accountID(w, r)
	if !ok {
		return
	}
	var body struct {
		Nodes []studioapp.UpdateFlowNodeInput `json:"nodes"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求内容格式不正确"})
		return
	}
	if err := h.Service.UpdateFlowNodes(r.Context(), accountID, chi.URLParam(r, "sessionID"), body.Nodes); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) createFlowNode(w http.ResponseWriter, r *http.Request) {
	accountID, ok := accountID(w, r)
	if !ok {
		return
	}
	var body studioapp.CreateFlowNodeInput
	if err := decodeJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求内容格式不正确"})
		return
	}
	node, err := h.Service.CreateFlowNode(r.Context(), accountID, chi.URLParam(r, "sessionID"), body)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, flowNodesToViews([]*domain.FlowNode{node})[0])
}

func (h *Handler) deleteFlowNode(w http.ResponseWriter, r *http.Request) {
	accountID, ok := accountID(w, r)
	if !ok {
		return
	}
	if err := h.Service.DeleteFlowNode(r.Context(), accountID, chi.URLParam(r, "sessionID"), chi.URLParam(r, "nodeID")); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) createFlowEdge(w http.ResponseWriter, r *http.Request) {
	accountID, ok := accountID(w, r)
	if !ok {
		return
	}
	var body studioapp.CreateFlowEdgeInput
	if err := decodeJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求内容格式不正确"})
		return
	}
	edge, err := h.Service.CreateFlowEdge(r.Context(), accountID, chi.URLParam(r, "sessionID"), body)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, flowEdgesToViews([]*domain.FlowEdge{edge})[0])
}

func (h *Handler) deleteFlowEdge(w http.ResponseWriter, r *http.Request) {
	accountID, ok := accountID(w, r)
	if !ok {
		return
	}
	if err := h.Service.DeleteFlowEdge(r.Context(), accountID, chi.URLParam(r, "sessionID"), chi.URLParam(r, "edgeID")); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type sessionView struct {
	ID             string                `json:"id"`
	Title          string                `json:"title"`
	PermissionMode domain.PermissionMode `json:"permission_mode"`
	ModelConfigID  string                `json:"model_config_id,omitempty"`
	Status         domain.SessionStatus  `json:"status"`
	CreatedAt      time.Time             `json:"created_at"`
	UpdatedAt      time.Time             `json:"updated_at"`
}

type messageView struct {
	ID        string             `json:"id"`
	SessionID string             `json:"session_id"`
	RunID     string             `json:"run_id,omitempty"`
	Role      domain.MessageRole `json:"role"`
	Content   json.RawMessage    `json:"content"`
	CreatedAt time.Time          `json:"created_at"`
}

type runView struct {
	ID               string           `json:"id"`
	SessionID        string           `json:"session_id"`
	TriggerMessageID string           `json:"trigger_message_id"`
	Status           domain.RunStatus `json:"status"`
	ModelConfigID    string           `json:"model_config_id,omitempty"`
	ErrorCode        string           `json:"error_code,omitempty"`
	ErrorMessage     string           `json:"error_message,omitempty"`
	CreatedAt        time.Time        `json:"created_at"`
	StartedAt        time.Time        `json:"started_at,omitempty"`
	CompletedAt      time.Time        `json:"completed_at,omitempty"`
	UpdatedAt        time.Time        `json:"updated_at"`
}

type assetVersionView struct {
	ID         string          `json:"id"`
	Version    int             `json:"version"`
	MIMEType   string          `json:"mime_type"`
	SizeBytes  int64           `json:"size_bytes"`
	Metadata   json.RawMessage `json:"metadata,omitempty"`
	ContentURL string          `json:"content_url"`
	CreatedAt  time.Time       `json:"created_at"`
}

type assetView struct {
	ID             string             `json:"id"`
	SessionID      string             `json:"session_id"`
	Name           string             `json:"name"`
	Kind           domain.AssetKind   `json:"kind"`
	Origin         domain.AssetOrigin `json:"origin"`
	SourceRunID    string             `json:"source_run_id,omitempty"`
	CurrentVersion int                `json:"current_version"`
	SavedToLibrary bool               `json:"saved_to_library"`
	Versions       []assetVersionView `json:"versions"`
	CreatedAt      time.Time          `json:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at"`
}

type flowNodeView struct {
	ID             string              `json:"id"`
	Type           domain.FlowNodeType `json:"type"`
	Title          string              `json:"title"`
	Body           string              `json:"body,omitempty"`
	AssetID        string              `json:"asset_id,omitempty"`
	AssetVersionID string              `json:"asset_version_id,omitempty"`
	AssetVersion   int                 `json:"asset_version,omitempty"`
	RunID          string              `json:"run_id,omitempty"`
	Position       positionView        `json:"position"`
	SortOrder      int                 `json:"sort_order"`
	UpdatedAt      time.Time           `json:"updated_at"`
}

type positionView struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type flowEdgeView struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
	Label  string `json:"label,omitempty"`
}

type eventView struct {
	ID        string          `json:"id"`
	RunID     string          `json:"run_id"`
	Sequence  uint64          `json:"sequence"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	CreatedAt time.Time       `json:"created_at"`
}

type libraryFolderView struct {
	ID        string    `json:"id"`
	ParentID  string    `json:"parent_id,omitempty"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (h *Handler) listSessions(w http.ResponseWriter, r *http.Request) {
	accountID, ok := accountID(w, r)
	if !ok {
		return
	}
	limit := queryInt(r, "limit", 50)
	offset := queryInt(r, "offset", 0)
	sessions, err := h.Repo.ListSessions(r.Context(), accountID, domain.SessionListQuery{Limit: limit, Offset: offset})
	if err != nil {
		writeError(w, err)
		return
	}
	out := make([]sessionView, 0, len(sessions))
	for _, session := range sessions {
		out = append(out, toSessionView(session))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) createSession(w http.ResponseWriter, r *http.Request) {
	accountID, ok := accountID(w, r)
	if !ok {
		return
	}
	session, err := h.Service.CreateSession(r.Context(), accountID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toSessionView(session))
}

func (h *Handler) getSession(w http.ResponseWriter, r *http.Request) {
	accountID, ok := accountID(w, r)
	if !ok {
		return
	}
	sessionID := chi.URLParam(r, "sessionID")
	session, err := h.Repo.GetSession(r.Context(), accountID, sessionID)
	if err != nil {
		writeError(w, err)
		return
	}
	messages, err := h.Repo.ListMessages(r.Context(), accountID, sessionID, 200)
	if err != nil {
		writeError(w, err)
		return
	}
	assets, err := h.Repo.ListSessionAssets(r.Context(), accountID, sessionID, 200)
	if err != nil {
		writeError(w, err)
		return
	}
	nodes, edges, err := h.Repo.GetFlow(r.Context(), accountID, sessionID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"session":  toSessionView(session),
		"messages": messagesToViews(messages),
		"assets":   assetsToViews(assets),
		"flow": map[string]any{
			"nodes": flowNodesToViews(nodes),
			"edges": flowEdgesToViews(edges),
		},
	})
}

func (h *Handler) listSessionRuns(w http.ResponseWriter, r *http.Request) {
	accountID, ok := accountID(w, r)
	if !ok {
		return
	}
	sessionID := chi.URLParam(r, "sessionID")
	if _, err := h.Repo.GetSession(r.Context(), accountID, sessionID); err != nil {
		writeError(w, err)
		return
	}
	runs, err := h.Repo.ListSessionRuns(r.Context(), accountID, sessionID, queryInt(r, "limit", 100))
	if err != nil {
		writeError(w, err)
		return
	}
	out := make([]runView, 0, len(runs))
	for _, run := range runs {
		out = append(out, toRunView(run))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) sendMessage(w http.ResponseWriter, r *http.Request) {
	accountID, ok := accountID(w, r)
	if !ok {
		return
	}
	var body struct {
		SessionID      string                `json:"session_id"`
		Text           string                `json:"text"`
		ModelConfigID  string                `json:"model_config_id"`
		PermissionMode domain.PermissionMode `json:"permission_mode"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求内容格式不正确"})
		return
	}
	result, err := h.Service.SendMessage(r.Context(), studioapp.SendMessageInput{
		AccountID: accountID, SessionID: body.SessionID, Text: body.Text,
		ModelConfigID: body.ModelConfigID, PermissionMode: body.PermissionMode,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{
		"session": toSessionView(result.Session),
		"message": toMessageView(result.Message),
		"run":     toRunView(result.Run),
	})
}

func (h *Handler) getRun(w http.ResponseWriter, r *http.Request) {
	accountID, ok := accountID(w, r)
	if !ok {
		return
	}
	run, err := h.Repo.GetRun(r.Context(), accountID, chi.URLParam(r, "runID"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toRunView(run))
}

func (h *Handler) listEvents(w http.ResponseWriter, r *http.Request) {
	accountID, ok := accountID(w, r)
	if !ok {
		return
	}
	after, _ := strconv.ParseUint(r.URL.Query().Get("after"), 10, 64)
	events, err := h.Repo.ListEventsAfter(r.Context(), accountID, chi.URLParam(r, "runID"), after, queryInt(r, "limit", 200))
	if err != nil {
		writeError(w, err)
		return
	}
	out := make([]eventView, 0, len(events))
	for _, event := range events {
		out = append(out, eventView{ID: event.ID, RunID: event.RunID, Sequence: event.Sequence, Type: event.Type, Payload: event.Payload, CreatedAt: event.CreatedAt})
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) cancelRun(w http.ResponseWriter, r *http.Request) {
	accountID, ok := accountID(w, r)
	if !ok {
		return
	}
	if err := h.Runner.Cancel(r.Context(), accountID, chi.URLParam(r, "runID")); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) retryRun(w http.ResponseWriter, r *http.Request) {
	accountID, ok := accountID(w, r)
	if !ok {
		return
	}
	run, err := h.Service.RetryRun(r.Context(), accountID, chi.URLParam(r, "runID"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, toRunView(run))
}

func (h *Handler) resolveApproval(w http.ResponseWriter, r *http.Request) {
	accountID, ok := accountID(w, r)
	if !ok {
		return
	}
	var body struct {
		Approved bool `json:"approved"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求内容格式不正确"})
		return
	}
	if err := h.Approvals.Resolve(r.Context(), studioapp.ResolveApprovalInput{AccountID: accountID, ApprovalID: chi.URLParam(r, "approvalID"), Approved: body.Approved}); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) assetContent(w http.ResponseWriter, r *http.Request) {
	accountID, ok := accountID(w, r)
	if !ok {
		return
	}
	asset, err := h.Repo.GetAsset(r.Context(), accountID, chi.URLParam(r, "assetID"))
	if err != nil {
		writeError(w, err)
		return
	}
	if len(asset.Versions) == 0 || h.Blob == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "资产内容不存在"})
		return
	}
	version := asset.Versions[len(asset.Versions)-1]
	if versionID := strings.TrimSpace(r.URL.Query().Get("version_id")); versionID != "" {
		found := false
		for _, candidate := range asset.Versions {
			if candidate.ID == versionID {
				version = candidate
				found = true
				break
			}
		}
		if !found {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "资产版本不存在"})
			return
		}
	}
	reader, err := h.Blob.Get(r.Context(), sharedkernel.BlobRef{Key: version.BlobKey, MIME: version.MIMEType, Size: version.SizeBytes})
	if err != nil {
		writeError(w, err)
		return
	}
	defer reader.Close()
	w.Header().Set("Content-Type", version.MIMEType)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = io.Copy(w, reader)
}

func (h *Handler) saveAssetToLibrary(w http.ResponseWriter, r *http.Request) {
	accountID, ok := accountID(w, r)
	if !ok {
		return
	}
	var body struct {
		FolderID string `json:"folder_id"`
	}
	if r.ContentLength > 0 {
		if err := decodeJSON(r, &body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求内容格式不正确"})
			return
		}
	}
	if err := h.Repo.SaveAssetToLibrary(r.Context(), accountID, chi.URLParam(r, "assetID"), body.FolderID, time.Now().UTC()); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) importLibraryAsset(w http.ResponseWriter, r *http.Request) {
	accountID, ok := accountID(w, r)
	if !ok {
		return
	}
	var body struct {
		AssetID        string `json:"asset_id"`
		AssetVersionID string `json:"asset_version_id"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求内容格式不正确"})
		return
	}
	asset, err := h.Service.ImportLibraryAsset(r.Context(), studioapp.ImportLibraryAssetInput{AccountID: accountID, SessionID: chi.URLParam(r, "sessionID"), SourceAssetID: body.AssetID, SourceVersionID: body.AssetVersionID})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, assetsToViews([]*domain.Asset{asset})[0])
}

func (h *Handler) listLibraryAssets(w http.ResponseWriter, r *http.Request) {
	accountID, ok := accountID(w, r)
	if !ok {
		return
	}
	assets, err := h.Repo.ListLibraryAssets(r.Context(), accountID, r.URL.Query().Get("folder_id"), queryInt(r, "limit", 100))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, assetsToViews(assets))
}

func (h *Handler) listLibraryFolders(w http.ResponseWriter, r *http.Request) {
	accountID, ok := accountID(w, r)
	if !ok {
		return
	}
	folders, err := h.Service.ListLibraryFolders(r.Context(), accountID)
	if err != nil {
		writeError(w, err)
		return
	}
	out := make([]libraryFolderView, 0, len(folders))
	for _, folder := range folders {
		out = append(out, libraryFolderView{ID: folder.ID, ParentID: folder.ParentID, Name: folder.Name, CreatedAt: folder.CreatedAt, UpdatedAt: folder.UpdatedAt})
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) createLibraryFolder(w http.ResponseWriter, r *http.Request) {
	accountID, ok := accountID(w, r)
	if !ok {
		return
	}
	var body struct {
		ParentID string `json:"parent_id"`
		Name     string `json:"name"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求内容格式不正确"})
		return
	}
	folder, err := h.Service.CreateLibraryFolder(r.Context(), studioapp.CreateLibraryFolderInput{AccountID: accountID, ParentID: body.ParentID, Name: body.Name})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, libraryFolderView{ID: folder.ID, ParentID: folder.ParentID, Name: folder.Name, CreatedAt: folder.CreatedAt, UpdatedAt: folder.UpdatedAt})
}

func (h *Handler) listModels(w http.ResponseWriter, r *http.Request) {
	accountID, ok := accountID(w, r)
	if !ok {
		return
	}
	if h.Models == nil {
		writeJSON(w, http.StatusOK, []any{})
		return
	}
	models, err := h.Models.List(r.Context(), accountID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, models)
}

func (h *Handler) createModel(w http.ResponseWriter, r *http.Request) {
	accountID, ok := accountID(w, r)
	if !ok {
		return
	}
	if h.Models == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "模型配置服务不可用"})
		return
	}
	var body studioapp.CreateModelConfigInput
	if err := decodeJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求内容格式不正确"})
		return
	}
	body.AccountID = accountID
	model, err := h.Models.Create(r.Context(), body)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, model)
}

func (h *Handler) testModelConnection(w http.ResponseWriter, r *http.Request) {
	accountID, ok := accountID(w, r)
	if !ok {
		return
	}
	if h.Models == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "模型配置服务不可用"})
		return
	}
	result, err := h.Models.TestConnection(r.Context(), accountID, chi.URLParam(r, "modelID"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) listSkills(w http.ResponseWriter, r *http.Request) {
	accountID, ok := accountID(w, r)
	if !ok {
		return
	}
	if h.Capabilities == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "能力配置服务不可用"})
		return
	}
	skills, err := h.Capabilities.ListSkills(r.Context(), accountID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, skills)
}

func (h *Handler) createSkill(w http.ResponseWriter, r *http.Request) {
	accountID, ok := accountID(w, r)
	if !ok {
		return
	}
	if h.Capabilities == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "能力配置服务不可用"})
		return
	}
	var body studioapp.CreateSkillInput
	if err := decodeJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求内容格式不正确"})
		return
	}
	body.AccountID = accountID
	skill, err := h.Capabilities.CreateSkill(r.Context(), body)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, skill)
}

func (h *Handler) updateSkill(w http.ResponseWriter, r *http.Request) {
	accountID, ok := accountID(w, r)
	if !ok {
		return
	}
	if h.Capabilities == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "能力配置服务不可用"})
		return
	}
	var body studioapp.UpdateSkillInput
	if err := decodeJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求内容格式不正确"})
		return
	}
	body.AccountID = accountID
	body.SkillID = chi.URLParam(r, "skillID")
	skill, err := h.Capabilities.UpdateSkill(r.Context(), body)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, skill)
}

func (h *Handler) listConnectors(w http.ResponseWriter, r *http.Request) {
	accountID, ok := accountID(w, r)
	if !ok {
		return
	}
	if h.Capabilities == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "能力配置服务不可用"})
		return
	}
	connectors, err := h.Capabilities.ListConnectors(r.Context(), accountID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, connectors)
}

func (h *Handler) createConnector(w http.ResponseWriter, r *http.Request) {
	accountID, ok := accountID(w, r)
	if !ok {
		return
	}
	if h.Capabilities == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "能力配置服务不可用"})
		return
	}
	var body studioapp.CreateConnectorInput
	if err := decodeJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求内容格式不正确"})
		return
	}
	body.AccountID = accountID
	connector, err := h.Capabilities.CreateConnector(r.Context(), body)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, connector)
}

func (h *Handler) updateConnector(w http.ResponseWriter, r *http.Request) {
	accountID, ok := accountID(w, r)
	if !ok {
		return
	}
	if h.Capabilities == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "能力配置服务不可用"})
		return
	}
	var body studioapp.UpdateConnectorInput
	if err := decodeJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求内容格式不正确"})
		return
	}
	body.AccountID = accountID
	body.ConnectorID = chi.URLParam(r, "connectorID")
	connector, err := h.Capabilities.UpdateConnector(r.Context(), body)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, connector)
}

func (h *Handler) probeConnector(w http.ResponseWriter, r *http.Request) {
	accountID, ok := accountID(w, r)
	if !ok {
		return
	}
	if h.Capabilities == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "能力配置服务不可用"})
		return
	}
	connector, err := h.Capabilities.ProbeConnector(r.Context(), accountID, chi.URLParam(r, "connectorID"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, connector)
}

func (h *Handler) listAgentWorkflows(w http.ResponseWriter, r *http.Request) {
	accountID, ok := accountID(w, r)
	if !ok {
		return
	}
	if h.Capabilities == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "能力配置服务不可用"})
		return
	}
	workflows, err := h.Capabilities.ListAgentWorkflows(r.Context(), accountID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, workflows)
}

func (h *Handler) updateAgentWorkflow(w http.ResponseWriter, r *http.Request) {
	accountID, ok := accountID(w, r)
	if !ok {
		return
	}
	if h.Capabilities == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "能力配置服务不可用"})
		return
	}
	var body struct {
		AgentEnabled *bool `json:"agent_enabled"`
	}
	if err := decodeJSON(r, &body); err != nil || body.AgentEnabled == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求内容格式不正确"})
		return
	}
	workflow, err := h.Capabilities.SetAgentWorkflowEnabled(r.Context(), accountID, chi.URLParam(r, "workflowID"), *body.AgentEnabled)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, workflow)
}

func accountID(w http.ResponseWriter, r *http.Request) (string, bool) {
	account, ok := setupapi.AccountFromContext(r.Context())
	if !ok || strings.TrimSpace(account.AccountID) == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "登录状态已失效"})
		return "", false
	}
	return account.AccountID, true
}

func decodeJSON(r *http.Request, out any) error {
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	return decoder.Decode(out)
}

func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	message := "服务暂时不可用"
	switch {
	case errors.Is(err, domain.ErrNotFound):
		status, message = http.StatusNotFound, "内容不存在或无权访问"
	case errors.Is(err, domain.ErrAlreadyExists):
		status, message = http.StatusConflict, "内容已存在"
	case errors.Is(err, domain.ErrInvalid), errors.Is(err, domain.ErrInvalidTransition):
		status, message = http.StatusBadRequest, err.Error()
	}
	writeJSON(w, status, map[string]string{"error": message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func queryInt(r *http.Request, key string, fallback int) int {
	value, err := strconv.Atoi(r.URL.Query().Get(key))
	if err != nil || value < 0 {
		return fallback
	}
	return value
}

func toSessionView(session *domain.Session) sessionView {
	return sessionView{ID: session.ID, Title: session.Title, PermissionMode: session.PermissionMode, ModelConfigID: session.ModelConfigID, Status: session.Status, CreatedAt: session.CreatedAt, UpdatedAt: session.UpdatedAt}
}

func toMessageView(message *domain.Message) messageView {
	return messageView{ID: message.ID, SessionID: message.SessionID, RunID: message.RunID, Role: message.Role, Content: message.ContentJSON, CreatedAt: message.CreatedAt}
}

func messagesToViews(messages []*domain.Message) []messageView {
	out := make([]messageView, 0, len(messages))
	for _, message := range messages {
		out = append(out, toMessageView(message))
	}
	return out
}

func toRunView(run *domain.Run) runView {
	return runView{ID: run.ID, SessionID: run.SessionID, TriggerMessageID: run.TriggerMessageID, Status: run.Status, ModelConfigID: run.ModelConfigID, ErrorCode: run.ErrorCode, ErrorMessage: run.ErrorMessage, CreatedAt: run.CreatedAt, StartedAt: run.StartedAt, CompletedAt: run.CompletedAt, UpdatedAt: run.UpdatedAt}
}

func assetsToViews(assets []*domain.Asset) []assetView {
	out := make([]assetView, 0, len(assets))
	for _, asset := range assets {
		versions := make([]assetVersionView, 0, len(asset.Versions))
		for _, version := range asset.Versions {
			versions = append(versions, assetVersionView{ID: version.ID, Version: version.Version, MIMEType: version.MIMEType, SizeBytes: version.SizeBytes, Metadata: version.Metadata, ContentURL: "/api/v1/studio/assets/" + asset.ID + "/content?version_id=" + url.QueryEscape(version.ID), CreatedAt: version.CreatedAt})
		}
		out = append(out, assetView{ID: asset.ID, SessionID: asset.SessionID, Name: asset.Name, Kind: asset.Kind, Origin: asset.Origin, SourceRunID: asset.SourceRunID, CurrentVersion: asset.CurrentVersion, SavedToLibrary: !asset.LibrarySavedAt.IsZero(), Versions: versions, CreatedAt: asset.CreatedAt, UpdatedAt: asset.UpdatedAt})
	}
	return out
}

func flowNodesToViews(nodes []*domain.FlowNode) []flowNodeView {
	out := make([]flowNodeView, 0, len(nodes))
	for _, node := range nodes {
		out = append(out, flowNodeView{ID: node.ID, Type: node.Type, Title: node.Title, Body: node.Body, AssetID: node.AssetID, AssetVersionID: node.AssetVersionID, AssetVersion: node.AssetVersion, RunID: node.RunID, Position: positionView{X: node.PositionX, Y: node.PositionY}, SortOrder: node.SortOrder, UpdatedAt: node.UpdatedAt})
	}
	return out
}

func flowEdgesToViews(edges []*domain.FlowEdge) []flowEdgeView {
	out := make([]flowEdgeView, 0, len(edges))
	for _, edge := range edges {
		out = append(out, flowEdgeView{ID: edge.ID, Source: edge.SourceNodeID, Target: edge.TargetNodeID, Label: edge.Label})
	}
	return out
}
