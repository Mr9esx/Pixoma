package comfyinstances

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/instance"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/comfyui"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

// Handler serves /api/v1/comfy-instances CRUD and observation endpoints.
type Handler struct {
	Repo  instance.Repository
	Pool  *instance.Pool
	Tasks domain.TaskRepository
	Mock  bool
}

// Mount registers chi routes on r (caller should mount under /api/v1/comfy-instances).
func (h *Handler) Mount(r chi.Router) {
	r.Get("/", h.list)
	r.Post("/", h.create)
	r.Get("/{id}", h.get)
	r.Patch("/{id}", h.patch)
	r.Delete("/{id}", h.delete)
	r.Get("/{id}/system", h.system)
	r.Get("/{id}/queue", h.queue)
	r.Get("/{id}/tasks", h.listTasks)
}

type instanceDTO struct {
	ID           string    `json:"id"`
	BaseURL      string    `json:"base_url"`
	Enabled      bool      `json:"enabled"`
	Capabilities []string  `json:"capabilities"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type createRequest struct {
	ID           string   `json:"id"`
	BaseURL      string   `json:"base_url"`
	Enabled      *bool    `json:"enabled"`
	Capabilities []string `json:"capabilities"`
}

type patchRequest struct {
	BaseURL      *string  `json:"base_url"`
	Enabled      *bool    `json:"enabled"`
	Capabilities []string `json:"capabilities"`
}

type taskDTO struct {
	ID           string    `json:"id"`
	SessionID    string    `json:"session_id"`
	CaseID       string    `json:"case_id"`
	Status       string    `json:"status"`
	InstanceID   string    `json:"instance_id,omitempty"`
	PromptID     string    `json:"prompt_id,omitempty"`
	ErrorCode    string    `json:"error_code,omitempty"`
	ErrorMessage string    `json:"error_message,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func toDTO(rec *instance.Record) instanceDTO {
	caps := rec.Capabilities
	if caps == nil {
		caps = []string{}
	}
	return instanceDTO{
		ID:           string(rec.ID),
		BaseURL:      rec.BaseURL,
		Enabled:      rec.Enabled,
		Capabilities: caps,
		CreatedAt:    rec.CreatedAt,
		UpdatedAt:    rec.UpdatedAt,
	}
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	list, err := h.Repo.List(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]instanceDTO, 0, len(list))
	for _, rec := range list {
		if rec == nil {
			continue
		}
		out = append(out, toDTO(rec))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.ID == "" || req.BaseURL == "" {
		writeErr(w, http.StatusBadRequest, "id and base_url required")
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	now := time.Now().UTC()
	rec := &instance.Record{
		ID:           sharedkernel.InstanceID(req.ID),
		BaseURL:      req.BaseURL,
		Enabled:      enabled,
		Capabilities: append([]string(nil), req.Capabilities...),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if _, err := h.Repo.Get(r.Context(), rec.ID); err == nil {
		writeErr(w, http.StatusConflict, "instance already exists")
		return
	} else if !errors.Is(err, instance.ErrNotFound) {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := h.Repo.Upsert(r.Context(), rec); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := h.refreshPool(r); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	got, err := h.Repo.Get(r.Context(), rec.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, toDTO(got))
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id := sharedkernel.InstanceID(chi.URLParam(r, "id"))
	rec, err := h.Repo.Get(r.Context(), id)
	if errors.Is(err, instance.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "instance not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toDTO(rec))
}

func (h *Handler) patch(w http.ResponseWriter, r *http.Request) {
	id := sharedkernel.InstanceID(chi.URLParam(r, "id"))
	rec, err := h.Repo.Get(r.Context(), id)
	if errors.Is(err, instance.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "instance not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	var req patchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.BaseURL != nil {
		if *req.BaseURL == "" {
			writeErr(w, http.StatusBadRequest, "base_url required")
			return
		}
		rec.BaseURL = *req.BaseURL
	}
	if req.Enabled != nil {
		rec.Enabled = *req.Enabled
	}
	if req.Capabilities != nil {
		rec.Capabilities = append([]string(nil), req.Capabilities...)
	}
	rec.UpdatedAt = time.Now().UTC()
	if err := h.Repo.Upsert(r.Context(), rec); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := h.refreshPool(r); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	got, err := h.Repo.Get(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toDTO(got))
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id := sharedkernel.InstanceID(chi.URLParam(r, "id"))
	if err := h.Repo.Delete(r.Context(), id); errors.Is(err, instance.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "instance not found")
		return
	} else if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := h.refreshPool(r); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) system(w http.ResponseWriter, r *http.Request) {
	id := sharedkernel.InstanceID(chi.URLParam(r, "id"))
	cli, err := h.clientFor(r, id)
	if err != nil {
		if errors.Is(err, instance.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "instance not found")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	st, err := cli.SystemStats(r.Context())
	if err != nil {
		writeJSON(w, http.StatusOK, comfyui.SystemStats{Reachable: false, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, st)
}

func (h *Handler) queue(w http.ResponseWriter, r *http.Request) {
	id := sharedkernel.InstanceID(chi.URLParam(r, "id"))
	cli, err := h.clientFor(r, id)
	if err != nil {
		if errors.Is(err, instance.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "instance not found")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	q, err := cli.Queue(r.Context())
	if err != nil {
		writeJSON(w, http.StatusOK, comfyui.QueueView{
			Reachable: false,
			Error:     err.Error(),
			Running:   []any{},
			Pending:   []any{},
		})
		return
	}
	writeJSON(w, http.StatusOK, q)
}

func (h *Handler) listTasks(w http.ResponseWriter, r *http.Request) {
	id := sharedkernel.InstanceID(chi.URLParam(r, "id"))
	if _, err := h.Repo.Get(r.Context(), id); errors.Is(err, instance.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "instance not found")
		return
	} else if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	q := domain.ListByInstanceQuery{}
	if s := r.URL.Query().Get("status"); s != "" {
		q.Status = sharedkernel.TaskStatus(s)
	}
	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			writeErr(w, http.StatusBadRequest, "invalid limit")
			return
		}
		q.Limit = n
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			writeErr(w, http.StatusBadRequest, "invalid offset")
			return
		}
		q.Offset = n
	}
	list, err := h.Tasks.ListByInstance(r.Context(), id, q)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]taskDTO, 0, len(list))
	for _, t := range list {
		if t == nil {
			continue
		}
		out = append(out, taskDTO{
			ID:           string(t.ID),
			SessionID:    string(t.SessionID),
			CaseID:       string(t.CaseID),
			Status:       string(t.Status),
			InstanceID:   string(t.InstanceID),
			PromptID:     t.PromptID,
			ErrorCode:    t.ErrorCode,
			ErrorMessage: t.ErrorMessage,
			CreatedAt:    t.CreatedAt,
			UpdatedAt:    t.UpdatedAt,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) clientFor(r *http.Request, id sharedkernel.InstanceID) (comfyui.Client, error) {
	rec, err := h.Repo.Get(r.Context(), id)
	if err != nil {
		return nil, err
	}
	if h.Pool != nil {
		if cli, err := h.Pool.Client(id); err == nil {
			return cli, nil
		}
	}
	return comfyui.NewClient(comfyui.Options{Mock: h.Mock, BaseURL: rec.BaseURL})
}

func (h *Handler) refreshPool(r *http.Request) error {
	if h.Pool == nil {
		return nil
	}
	return h.Pool.Refresh(r.Context())
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
