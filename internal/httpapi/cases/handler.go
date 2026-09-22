package cases

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/Mr9esx/Pixoma/internal/apierr"
	caseapp "github.com/Mr9esx/Pixoma/internal/cases/application"
	domain "github.com/Mr9esx/Pixoma/internal/cases/domain"
	"github.com/Mr9esx/Pixoma/internal/response"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
)

// Handler serves Case admin CRUD under /api/v1/cases.
type Handler struct {
	Repo     domain.Repository
	Validate func(domain.CaseDocument) error
	// DeleteWithCleanup performs the cleanup delete (see internal/cases/application).
	DeleteWithCleanup func(ctx context.Context, id sharedkernel.CaseID, ack bool) (caseapp.DeleteSummary, error)
}

// Mount registers chi routes on r (caller mounts under /api/v1/cases).
func (h *Handler) Mount(r chi.Router) {
	r.Get("/", h.list)
	r.Post("/", h.create)
	r.Get("/{id}", h.get)
	r.Patch("/{id}", h.patch)
	r.Delete("/{id}", h.delete)
	r.Post("/{id}/disable", h.disable)
	r.Post("/{id}/enable", h.enable)
}

type caseDTO struct {
	domain.CaseDocument
	Enabled bool `json:"enabled"`
}

type writeBody struct {
	domain.CaseDocument
	Enabled *bool `json:"enabled"`
}

// mergeDocument 按 PATCH 语义合并：请求里显式提供的字段覆盖现有值，
// 省略的字段（name / description / input_schema 等）保留原文档，避免
// 工作流区块只保存 inputs/outputs/bindings 时把其它字段冲掉。
func mergeDocument(existing, patch domain.CaseDocument) domain.CaseDocument {
	doc := existing
	if patch.Name != "" {
		doc.Name = patch.Name
	}
	if patch.Description != "" {
		doc.Description = patch.Description
	}
	if patch.Preview != "" {
		doc.Preview = patch.Preview
	}
	if patch.Tags != nil {
		doc.Tags = patch.Tags
	}
	if patch.Categories != nil {
		doc.Categories = patch.Categories
	}
	if patch.Routing != nil {
		doc.Routing = patch.Routing
	}
	if patch.Inputs != nil {
		doc.Inputs = patch.Inputs
	}
	if patch.Outputs != nil {
		doc.Outputs = patch.Outputs
	}
	if patch.Bindings.WorkflowJSON != nil {
		doc.Bindings.WorkflowJSON = patch.Bindings.WorkflowJSON
	}
	if patch.Bindings.Inputs != nil {
		doc.Bindings.Inputs = patch.Bindings.Inputs
	}
	if patch.Bindings.Outputs != nil {
		doc.Bindings.Outputs = patch.Bindings.Outputs
	}
	if patch.InputSchema != nil {
		doc.InputSchema = patch.InputSchema
	}
	if patch.WorkflowFilename != "" {
		doc.WorkflowFilename = patch.WorkflowFilename
	}
	return doc
}

// schemaFromInputs 由输入字段生成 JSON Schema（与前端 case-form 同构）。
func schemaFromInputs(inputs []domain.InputField) map[string]any {
	props := make(map[string]any, len(inputs))
	for _, in := range inputs {
		props[in.Key] = map[string]any{"type": inputSchemaType(in.Type)}
	}
	return map[string]any{
		"type":       "object",
		"properties": props,
		"required":   []any{},
	}
}

// inputSchemaType 把业务输入类型映射成合法 JSON Schema 类型：
// 图片/视频等媒体字段在 values 里存的是 Blob 引用字符串，schema 用 string。
func inputSchemaType(t string) string {
	switch t {
	case "number":
		return "number"
	case "boolean":
		return "boolean"
	default:
		return "string"
	}
}

func toDTO(c *domain.Case) caseDTO {
	doc := c.Document
	if doc.Inputs == nil {
		doc.Inputs = []domain.InputField{}
	}
	if doc.Outputs == nil {
		doc.Outputs = []domain.OutputField{}
	}
	if doc.Bindings.WorkflowJSON == nil {
		doc.Bindings.WorkflowJSON = map[string]any{}
	}
	if doc.Bindings.Inputs == nil {
		doc.Bindings.Inputs = []domain.InputBinding{}
	}
	if doc.Bindings.Outputs == nil {
		doc.Bindings.Outputs = []domain.OutputBinding{}
	}
	if doc.InputSchema == nil {
		doc.InputSchema = map[string]any{}
	}
	return caseDTO{CaseDocument: doc, Enabled: c.Enabled}
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	q, err := parseListQuery(r)
	if err != nil {
		response.FailErr(w, apierr.ErrCaseListInvalidQuery, err)
		return
	}
	list, err := h.Repo.List(r.Context(), q)
	if err != nil {
		response.FailErr(w, apierr.ErrCaseListFailed, err)
		return
	}
	out := make([]caseDTO, 0, len(list))
	for _, c := range list {
		if c == nil {
			continue
		}
		out = append(out, toDTO(c))
	}
	response.OKStatus(w, http.StatusOK, out)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var body writeBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Fail(w, apierr.ErrCaseCreateInvalidJSON, "invalid json")
		return
	}
	if err := h.validate(body.CaseDocument); err != nil {
		response.FailErr(w, apierr.ErrCaseCreateInvalid, err)
		return
	}
	enabled := true
	if body.Enabled != nil {
		enabled = *body.Enabled
	}
	c := &domain.Case{Document: body.CaseDocument, Enabled: enabled}
	if err := h.Repo.Create(r.Context(), c); errors.Is(err, domain.ErrAlreadyExists) {
		response.Fail(w, apierr.ErrCaseCreateAlreadyExists, "case already exists")
		return
	} else if err != nil {
		response.FailErr(w, apierr.ErrCaseCreateFailed, err)
		return
	}
	got, err := h.Repo.Get(r.Context(), c.Document.ID)
	if err != nil {
		response.FailErr(w, apierr.ErrCaseCreateFailed, err)
		return
	}
	response.OKStatus(w, http.StatusCreated, toDTO(got))
}

func parseCaseID(s string) (sharedkernel.CaseID, error) {
	n, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return sharedkernel.CaseID(n), nil
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, err := parseCaseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Fail(w, apierr.ErrCaseGetInvalidCaseID, "invalid case id")
		return
	}
	c, err := h.Repo.Get(r.Context(), id)
	if errors.Is(err, domain.ErrNotFound) {
		response.Fail(w, apierr.ErrCaseGetNotFound, "case not found")
		return
	}
	if err != nil {
		response.FailErr(w, apierr.ErrCaseListFailed, err)
		return
	}
	response.OKStatus(w, http.StatusOK, toDTO(c))
}

// delete removes a case with cleanup: pending tasks are failed with a recorded
// reason, active sessions are terminated, menu/card references are unlinked,
// and the case row is deleted. running/queued tasks keep their job_ref
// snapshots and finish normally; historical task rows remain queryable.
func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseCaseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Fail(w, apierr.ErrCaseGetInvalidCaseID, "invalid case id")
		return
	}
	var body struct {
		AckReferences bool `json:"ack_references"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if h.DeleteWithCleanup == nil {
		response.Fail(w, apierr.ErrCaseDeleteNotConfigured, "delete cleanup not configured")
		return
	}
	summary, err := h.DeleteWithCleanup(r.Context(), id, body.AckReferences)
	if errors.Is(err, caseapp.ErrNeedsAck) {
		response.Fail(w, apierr.ErrCaseDeleteConflict, "case is referenced by menu or card entries; confirm with ack_references to remove references")
		return
	}
	if errors.Is(err, domain.ErrNotFound) {
		response.Fail(w, apierr.ErrCaseGetNotFound, "case not found")
		return
	}
	if err != nil {
		response.FailErr(w, apierr.ErrCaseDeleteFailed, err)
		return
	}
	response.OKStatus(w, http.StatusOK, map[string]any{
		"deleted":             true,
		"removed_placements":  summary.RemovedPlacements,
		"failed_tasks":        summary.FailedTasks,
		"terminated_sessions": summary.TerminatedSessions,
	})
}

func (h *Handler) patch(w http.ResponseWriter, r *http.Request) {
	id, err := parseCaseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Fail(w, apierr.ErrCaseGetInvalidCaseID, "invalid case id")
		return
	}
	existing, err := h.Repo.Get(r.Context(), id)
	if errors.Is(err, domain.ErrNotFound) {
		response.Fail(w, apierr.ErrCaseGetNotFound, "case not found")
		return
	}
	if err != nil {
		response.FailErr(w, apierr.ErrCaseUpdateFailed, err)
		return
	}
	var body writeBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Fail(w, apierr.ErrCaseCreateInvalidJSON, "invalid json")
		return
	}
	doc := mergeDocument(existing.Document, body.CaseDocument)
	if body.Inputs != nil && body.InputSchema == nil {
		doc.InputSchema = schemaFromInputs(body.Inputs)
	}
	doc.ID = id
	if err := h.validate(doc); err != nil {
		response.FailErr(w, apierr.ErrCaseUpdateInvalid, err)
		return
	}
	existing.Document = doc
	if body.Enabled != nil {
		existing.Enabled = *body.Enabled
	}
	if err := h.Repo.Save(r.Context(), existing); err != nil {
		response.FailErr(w, apierr.ErrCaseUpdateFailed, err)
		return
	}
	got, err := h.Repo.Get(r.Context(), id)
	if err != nil {
		response.FailErr(w, apierr.ErrCaseUpdateFailed, err)
		return
	}
	response.OKStatus(w, http.StatusOK, toDTO(got))
}

func (h *Handler) disable(w http.ResponseWriter, r *http.Request) {
	id, err := parseCaseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Fail(w, apierr.ErrCaseGetInvalidCaseID, "invalid case id")
		return
	}
	if err := h.Repo.Disable(r.Context(), id); errors.Is(err, domain.ErrNotFound) {
		response.Fail(w, apierr.ErrCaseGetNotFound, "case not found")
		return
	} else if err != nil {
		response.FailErr(w, apierr.ErrCaseDisableFailed, err)
		return
	}
	got, err := h.Repo.Get(r.Context(), id)
	if err != nil {
		response.FailErr(w, apierr.ErrCaseDisableFailed, err)
		return
	}
	response.OKStatus(w, http.StatusOK, toDTO(got))
}

func (h *Handler) enable(w http.ResponseWriter, r *http.Request) {
	id, err := parseCaseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Fail(w, apierr.ErrCaseGetInvalidCaseID, "invalid case id")
		return
	}
	if err := h.Repo.Enable(r.Context(), id); errors.Is(err, domain.ErrNotFound) {
		response.Fail(w, apierr.ErrCaseGetNotFound, "case not found")
		return
	} else if err != nil {
		response.FailErr(w, apierr.ErrCaseEnableFailed, err)
		return
	}
	got, err := h.Repo.Get(r.Context(), id)
	if err != nil {
		response.FailErr(w, apierr.ErrCaseEnableFailed, err)
		return
	}
	response.OKStatus(w, http.StatusOK, toDTO(got))
}

func (h *Handler) validate(doc domain.CaseDocument) error {
	if h.Validate == nil {
		return nil
	}
	return h.Validate(doc)
}

func parseListQuery(r *http.Request) (domain.ListQuery, error) {
	q := domain.ListQuery{
		Q:        r.URL.Query().Get("q"),
		Category: r.URL.Query().Get("category"),
		Tag:      r.URL.Query().Get("tag"),
	}
	if v := r.URL.Query().Get("enabled"); v != "" {
		b, err := parseBoolQuery(v)
		if err != nil {
			return q, err
		}
		q.Enabled = &b
	}
	if v := r.URL.Query().Get("created_from"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return q, errors.New("invalid created_from")
		}
		q.CreatedFrom = &t
	}
	if v := r.URL.Query().Get("created_to"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return q, errors.New("invalid created_to")
		}
		q.CreatedTo = &t
	}
	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			return q, errors.New("invalid limit")
		}
		if n > 200 {
			n = 200
		}
		q.Limit = n
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			return q, errors.New("invalid offset")
		}
		q.Offset = n
	}
	return q, nil
}

func parseBoolQuery(v string) (bool, error) {
	switch v {
	case "true", "1":
		return true, nil
	case "false", "0":
		return false, nil
	default:
		return false, errors.New("invalid enabled")
	}
}
