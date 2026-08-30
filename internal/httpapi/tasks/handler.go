package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	runtimedomain "github.com/mr9esx/comfyui_tgbot/internal/runtime/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

// Canceller cancels a task using runtime domain rules (e.g. orchestrator.RequestCancel).
type Canceller interface {
	RequestCancel(ctx context.Context, taskID sharedkernel.TaskID) error
}

// Handler serves task admin list/get/cancel under /api/v1/tasks.
type Handler struct {
	Tasks  runtimedomain.TaskRepository
	Cancel Canceller
}

// Mount registers list/get/cancel. Does not register task creation.
func (h *Handler) Mount(r chi.Router) {
	r.Get("/", h.list)
	r.Get("/{id}", h.get)
	r.Post("/{id}/cancel", h.cancel)
}

type taskDTO struct {
	ID            string     `json:"id"`
	SessionID     string     `json:"session_id"`
	ChatID        string     `json:"chat_id,omitempty"`
	CaseID        uint64     `json:"case_id"`
	Status        string     `json:"status"`
	EdgeID        string     `json:"instance_id,omitempty"`
	DispatchTopic string     `json:"dispatch_topic"`
	PromptID      string     `json:"prompt_id,omitempty"`
	ErrorCode     string     `json:"error_code,omitempty"`
	ErrorMessage  string     `json:"error_message,omitempty"`
	StartedAt     *time.Time `json:"started_at,omitempty"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

func toDTO(t *runtimedomain.Task) taskDTO {
	return taskDTO{
		ID:            string(t.ID),
		SessionID:     string(t.SessionID),
		ChatID:        string(t.ChatID),
		CaseID:        uint64(t.CaseID),
		Status:        string(t.Status),
		EdgeID:        string(t.EdgeID),
		DispatchTopic: t.DispatchTopic,
		PromptID:      t.PromptID,
		ErrorCode:     t.ErrorCode,
		ErrorMessage:  t.ErrorMessage,
		StartedAt:     timePtr(t.StartedAt),
		CompletedAt:   timePtr(t.CompletedAt),
		CreatedAt:     t.CreatedAt,
		UpdatedAt:     t.UpdatedAt,
	}
}

func timePtr(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	q, err := parseAdminListQuery(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	list, err := h.Tasks.List(r.Context(), q)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]taskDTO, 0, len(list))
	for _, t := range list {
		if t == nil {
			continue
		}
		out = append(out, toDTO(t))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id := sharedkernel.TaskID(chi.URLParam(r, "id"))
	t, err := h.Tasks.Get(r.Context(), id)
	if errors.Is(err, runtimedomain.ErrTaskNotFound) {
		writeErr(w, http.StatusNotFound, "task not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toDTO(t))
}

func (h *Handler) cancel(w http.ResponseWriter, r *http.Request) {
	if h.Cancel == nil {
		writeErr(w, http.StatusInternalServerError, "cancel not configured")
		return
	}
	id := sharedkernel.TaskID(chi.URLParam(r, "id"))
	err := h.Cancel.RequestCancel(r.Context(), id)
	if errors.Is(err, runtimedomain.ErrTaskNotFound) {
		writeErr(w, http.StatusNotFound, "task not found")
		return
	}
	if errors.Is(err, runtimedomain.ErrCancelNotAllowed) {
		writeErr(w, http.StatusConflict, "cancel not allowed")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	t, err := h.Tasks.Get(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toDTO(t))
}

func parseAdminListQuery(r *http.Request) (runtimedomain.AdminListQuery, error) {
	q := runtimedomain.AdminListQuery{Q: r.URL.Query().Get("q")}
	if v := r.URL.Query().Get("status"); v != "" {
		q.Status = sharedkernel.TaskStatus(v)
	}
	if v := r.URL.Query().Get("edge_id"); v != "" {
		q.EdgeID = sharedkernel.EdgeID(v)
	}
	if v := r.URL.Query().Get("session_id"); v != "" {
		q.SessionID = sharedkernel.SessionID(v)
	}
	if v := r.URL.Query().Get("case_id"); v != "" {
		if parsed, perr := sharedkernel.ParseCaseID(v); perr == nil {
			q.CaseID = parsed
		}
	}
	if v := r.URL.Query().Get("channel_id"); v != "" {
		q.ChannelID = v
	}
	if v := r.URL.Query().Get("chat_id"); v != "" {
		q.ChatID = sharedkernel.ChatID(v)
	}
	if v := r.URL.Query().Get("dispatch_topic"); v != "" {
		q.DispatchTopic = v
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

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
