package edges

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/edge"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/presence"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

// Handler serves /api/v1/edges CRUD and observation endpoints.
type Handler struct {
	Repo     edge.Repository
	Pool     *edge.Pool
	Tasks    domain.TaskRepository
	Metrics  edge.MetricsRepository
	EncKey   []byte
	Presence *presence.Store
}

// Mount registers chi routes on r (caller should mount under /api/v1/edges).
func (h *Handler) Mount(r chi.Router) {
	r.Get("/", h.list)
	r.Post("/", h.create)
	r.Get("/presence", h.listPresence)
	r.Post("/{id}/rotate-token", h.rotateToken)
	r.Get("/{id}", h.get)
	r.Patch("/{id}", h.patch)
	r.Delete("/{id}", h.delete)
	r.Get("/{id}/metrics", h.metrics)
	r.Get("/{id}/tasks", h.listTasks)
	r.Get("/{id}/stats", h.stats)
}

type instanceDTO struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Description  string         `json:"description,omitempty"`
	Enabled      bool           `json:"enabled"`
	Capabilities []string       `json:"capabilities"`
	AgentToken   string         `json:"agent_token,omitempty"`
	Hardware     *edge.Hardware `json:"hardware,omitempty"`
	StartedAt    *time.Time     `json:"started_at,omitempty"`
	ComfyVersion string         `json:"comfy_version,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

type createRequest struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Enabled      *bool    `json:"enabled"`
	Capabilities []string `json:"capabilities"`
}

type patchRequest struct {
	Name            *string        `json:"name"`
	Description     *string        `json:"description"`
	Enabled         *bool          `json:"enabled"`
	Capabilities    []string       `json:"capabilities"`
	RefreshHardware *bool          `json:"refresh_hardware"`
	Hardware        *edge.Hardware `json:"hardware"`
}

type taskDTO struct {
	ID           string    `json:"id"`
	SessionID    string    `json:"session_id"`
	CaseID       uint64    `json:"case_id"`
	Status       string    `json:"status"`
	EdgeID       string    `json:"edge_id,omitempty"`
	PromptID     string    `json:"prompt_id,omitempty"`
	ErrorCode    string    `json:"error_code,omitempty"`
	ErrorMessage string    `json:"error_message,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func toDTO(rec *edge.Record, token string) instanceDTO {
	caps := rec.Capabilities
	if caps == nil {
		caps = []string{}
	}
	name := rec.Name
	if name == "" {
		name = string(rec.ID)
	}
	var hw *edge.Hardware
	if !edge.HardwareEmpty(rec.Hardware) {
		cp := rec.Hardware
		hw = &cp
	}
	return instanceDTO{
		ID:           string(rec.ID),
		Name:         name,
		Description:  rec.Description,
		Enabled:      rec.Enabled,
		Capabilities: caps,
		AgentToken:   token,
		Hardware:     hw,
		StartedAt:    rec.StartedAt,
		ComfyVersion: rec.ComfyVersion,
		CreatedAt:    rec.CreatedAt,
		UpdatedAt:    rec.UpdatedAt,
	}
}

func (h *Handler) decryptToken(rec *edge.Record) (string, error) {
	if rec == nil || rec.AgentTokenEnc == "" {
		return "", nil
	}
	return edge.DecryptToken(h.EncKey, rec.AgentTokenEnc)
}

func (h *Handler) mintAndStoreToken(rec *edge.Record) (string, error) {
	plain, err := edge.MintToken()
	if err != nil {
		return "", err
	}
	enc, err := edge.EncryptToken(h.EncKey, plain)
	if err != nil {
		return "", err
	}
	rec.AgentTokenEnc = enc
	return plain, nil
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
		out = append(out, toDTO(rec, ""))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) listPresence(w http.ResponseWriter, r *http.Request) {
	list, err := h.Repo.List(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]presence.Snapshot, 0, len(list))
	for _, rec := range list {
		if rec == nil {
			continue
		}
		if h.Presence == nil {
			out = append(out, presence.Snapshot{ID: string(rec.ID)})
			continue
		}
		out = append(out, h.Presence.Snapshot(rec.ID))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	name := strings.TrimSpace(req.Name)
	id := strings.TrimSpace(req.ID)
	if id == "" {
		gen, err := edge.NewID()
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		id = string(gen)
	}
	if name == "" {
		name = id
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	now := time.Now().UTC()
	rec := &edge.Record{
		ID:           sharedkernel.EdgeID(id),
		Name:         name,
		Description:  strings.TrimSpace(req.Description),
		Enabled:      enabled,
		Capabilities: append([]string(nil), req.Capabilities...),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if _, err := h.Repo.Get(r.Context(), rec.ID); err == nil {
		writeErr(w, http.StatusConflict, "instance already exists")
		return
	} else if !errors.Is(err, edge.ErrNotFound) {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	plain, err := h.mintAndStoreToken(rec)
	if err != nil {
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
	writeJSON(w, http.StatusCreated, toDTO(got, plain))
}

func (h *Handler) writeInstance(w http.ResponseWriter, status int, rec *edge.Record) {
	tok, err := h.decryptToken(rec)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, status, toDTO(rec, tok))
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id := sharedkernel.EdgeID(chi.URLParam(r, "id"))
	rec, err := h.Repo.Get(r.Context(), id)
	if errors.Is(err, edge.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "instance not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.writeInstance(w, http.StatusOK, rec)
}

func (h *Handler) rotateToken(w http.ResponseWriter, r *http.Request) {
	id := sharedkernel.EdgeID(chi.URLParam(r, "id"))
	rec, err := h.Repo.Get(r.Context(), id)
	if errors.Is(err, edge.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "instance not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	plain, err := h.mintAndStoreToken(rec)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := h.Repo.UpdateAgentTokenEnc(r.Context(), rec.ID, rec.AgentTokenEnc); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	rec.UpdatedAt = time.Now().UTC()
	writeJSON(w, http.StatusOK, toDTO(rec, plain))
}

func (h *Handler) patch(w http.ResponseWriter, r *http.Request) {
	id := sharedkernel.EdgeID(chi.URLParam(r, "id"))
	rec, err := h.Repo.Get(r.Context(), id)
	if errors.Is(err, edge.ErrNotFound) {
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
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			writeErr(w, http.StatusBadRequest, "name required")
			return
		}
		rec.Name = name
	}
	if req.Description != nil {
		rec.Description = strings.TrimSpace(*req.Description)
	}
	if req.Enabled != nil {
		rec.Enabled = *req.Enabled
	}
	if req.Capabilities != nil {
		rec.Capabilities = append([]string(nil), req.Capabilities...)
	}
	if req.Hardware != nil {
		hw := *req.Hardware
		hw.CollectedAt = time.Now().UTC()
		rec.Hardware = hw
	}
	rec.UpdatedAt = time.Now().UTC()
	if err := h.Repo.Upsert(r.Context(), rec); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if req.Hardware != nil {
		if err := h.Repo.UpdateHardware(r.Context(), rec.ID, rec.Hardware); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	if req.RefreshHardware != nil {
		if err := h.Repo.SetHardwareRefreshRequested(r.Context(), rec.ID, *req.RefreshHardware); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
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
	h.writeInstance(w, http.StatusOK, got)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id := sharedkernel.EdgeID(chi.URLParam(r, "id"))
	if err := h.Repo.Delete(r.Context(), id); errors.Is(err, edge.ErrNotFound) {
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

func (h *Handler) listTasks(w http.ResponseWriter, r *http.Request) {
	id := sharedkernel.EdgeID(chi.URLParam(r, "id"))
	if _, err := h.Repo.Get(r.Context(), id); errors.Is(err, edge.ErrNotFound) {
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
			CaseID:       uint64(t.CaseID),
			Status:       string(t.Status),
			EdgeID:       string(t.EdgeID),
			PromptID:     t.PromptID,
			ErrorCode:    t.ErrorCode,
			ErrorMessage: t.ErrorMessage,
			CreatedAt:    t.CreatedAt,
			UpdatedAt:    t.UpdatedAt,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

type statsDTO struct {
	TaskCount   int      `json:"task_count"`
	RuntimeMS   int64    `json:"runtime_ms"`
	SuccessRate *float64 `json:"success_rate"`
}

func (h *Handler) stats(w http.ResponseWriter, r *http.Request) {
	id := sharedkernel.EdgeID(chi.URLParam(r, "id"))
	if _, err := h.Repo.Get(r.Context(), id); errors.Is(err, edge.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "instance not found")
		return
	} else if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := statsDTO{}
	if h.Tasks == nil {
		writeJSON(w, http.StatusOK, out)
		return
	}
	list, err := h.Tasks.ListByInstance(r.Context(), id, domain.ListByInstanceQuery{})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	var okN, failN int
	for _, t := range list {
		if t == nil {
			continue
		}
		out.TaskCount++
		switch t.Status {
		case sharedkernel.TaskSucceeded, sharedkernel.TaskFailed, sharedkernel.TaskCancelled:
			ms := t.UpdatedAt.Sub(t.CreatedAt).Milliseconds()
			if ms < 0 {
				ms = 0
			}
			out.RuntimeMS += ms
		}
		switch t.Status {
		case sharedkernel.TaskSucceeded:
			okN++
		case sharedkernel.TaskFailed:
			failN++
		}
	}
	if den := okN + failN; den > 0 {
		rate := float64(okN) / float64(den)
		out.SuccessRate = &rate
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) refreshPool(r *http.Request) error {
	if h.Pool == nil {
		return nil
	}
	return h.Pool.Refresh(r.Context())
}

func (h *Handler) metrics(w http.ResponseWriter, r *http.Request) {
	id := sharedkernel.EdgeID(chi.URLParam(r, "id"))
	if h.Metrics == nil {
		writeErr(w, http.StatusInternalServerError, "metrics not configured")
		return
	}
	if _, err := h.Repo.Get(r.Context(), id); errors.Is(err, edge.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "instance not found")
		return
	} else if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	window, err := parseMetricsWindow(r.URL.Query().Get("window"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	since := time.Now().UTC().Add(-window)
	series, err := h.Metrics.ListSince(r.Context(), id, since, 720)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	var latest *edge.Metrics
	if len(series) > 0 {
		cp := series[len(series)-1]
		latest = &cp
	}
	writeJSON(w, http.StatusOK, map[string]any{"latest": latest, "series": series})
}

func parseMetricsWindow(raw string) (time.Duration, error) {
	switch strings.TrimSpace(raw) {
	case "", "1h":
		return time.Hour, nil
	case "6h":
		return 6 * time.Hour, nil
	case "24h":
		return 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("invalid window %q", raw)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
