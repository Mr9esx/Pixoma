package edges

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/Mr9esx/Pixoma/internal/apierr"
	edgeapp "github.com/Mr9esx/Pixoma/internal/edge/application"
	edge "github.com/Mr9esx/Pixoma/internal/edge/domain"
	"github.com/Mr9esx/Pixoma/internal/edge/infrastructure/presence"
	"github.com/Mr9esx/Pixoma/internal/response"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
	"github.com/Mr9esx/Pixoma/internal/tasks/domain"
	topicdomain "github.com/Mr9esx/Pixoma/internal/topics/domain"
)

// Handler serves /api/v1/edges CRUD and observation endpoints.
type Handler struct {
	Repo     edge.Repository
	Pool     *edgeapp.Pool
	Tasks    domain.TaskRepository
	Metrics  edge.MetricsRepository
	EncKey   []byte
	Presence *presence.Store
	Topics   topicdomain.Repository
	// DeleteWithCleanup performs the cleanup delete (see internal/edge/application).
	DeleteWithCleanup func(ctx context.Context, id sharedkernel.EdgeID, ack bool) (edgeapp.DeleteSummary, error)
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
	ID              string         `json:"id"`
	Name            string         `json:"name"`
	Description     string         `json:"description,omitempty"`
	Enabled         bool           `json:"enabled"`
	Capabilities    []string       `json:"capabilities"`
	SubscribeTopics []string       `json:"subscribe_topics"`
	EffectiveTopics []string       `json:"effective_topics"`
	AgentToken      string         `json:"agent_token,omitempty"`
	Hardware        *edge.Hardware `json:"hardware,omitempty"`
	StartedAt       *time.Time     `json:"started_at,omitempty"`
	ComfyVersion    string         `json:"comfy_version,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
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
	SubscribeTopics []string       `json:"subscribe_topics"`
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
		ID:              string(rec.ID),
		Name:            name,
		Description:     rec.Description,
		Enabled:         rec.Enabled,
		Capabilities:    caps,
		SubscribeTopics: append([]string(nil), rec.SubscribeTopics...),
		EffectiveTopics: rec.EffectiveTopics(),
		AgentToken:      token,
		Hardware:        hw,
		StartedAt:       rec.StartedAt,
		ComfyVersion:    rec.ComfyVersion,
		CreatedAt:       rec.CreatedAt,
		UpdatedAt:       rec.UpdatedAt,
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
		response.FailErr(w, apierr.ErrEdgeListFailed, err)
		return
	}
	out := make([]instanceDTO, 0, len(list))
	for _, rec := range list {
		if rec == nil {
			continue
		}
		out = append(out, toDTO(rec, ""))
	}
	response.OKStatus(w, http.StatusOK, out)
}

func (h *Handler) listPresence(w http.ResponseWriter, r *http.Request) {
	list, err := h.Repo.List(r.Context())
	if err != nil {
		response.FailErr(w, apierr.ErrEdgeListPresenceFailed, err)
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
	response.OKStatus(w, http.StatusOK, out)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Fail(w, apierr.ErrEdgeCreateInvalidJSON, "invalid json")
		return
	}
	name := strings.TrimSpace(req.Name)
	id := strings.TrimSpace(req.ID)
	if id == "" {
		gen, err := edge.NewID()
		if err != nil {
			response.FailErr(w, apierr.ErrEdgeCreateFailed, err)
			return
		}
		id = string(gen)
	}
	if name == "" {
		response.Fail(w, apierr.ErrEdgeCreateNameRequired, "name required")
		return
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
		response.Fail(w, apierr.ErrEdgeCreateAlreadyExists, "instance already exists")
		return
	} else if !errors.Is(err, edge.ErrNotFound) {
		response.FailErr(w, apierr.ErrEdgeCreateFailed, err)
		return
	}
	plain, err := h.mintAndStoreToken(rec)
	if err != nil {
		response.FailErr(w, apierr.ErrEdgeCreateFailed, err)
		return
	}
	if err := h.Repo.Upsert(r.Context(), rec); err != nil {
		response.FailErr(w, apierr.ErrEdgeCreateFailed, err)
		return
	}
	if err := h.refreshPool(r); err != nil {
		response.FailErr(w, apierr.ErrEdgeCreateFailed, err)
		return
	}
	got, err := h.Repo.Get(r.Context(), rec.ID)
	if err != nil {
		response.FailErr(w, apierr.ErrEdgeCreateFailed, err)
		return
	}
	response.OKStatus(w, http.StatusCreated, toDTO(got, plain))
}

func (h *Handler) writeInstance(w http.ResponseWriter, status int, rec *edge.Record) {
	tok, err := h.decryptToken(rec)
	if err != nil {
		response.FailErr(w, apierr.ErrEdgeWriteFailed, err)
		return
	}
	response.OKStatus(w, status, toDTO(rec, tok))
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id := sharedkernel.EdgeID(chi.URLParam(r, "id"))
	rec, err := h.Repo.Get(r.Context(), id)
	if errors.Is(err, edge.ErrNotFound) {
		response.Fail(w, apierr.ErrEdgeGetNotFound, "instance not found")
		return
	}
	if err != nil {
		response.FailErr(w, apierr.ErrEdgeListFailed, err)
		return
	}
	h.writeInstance(w, http.StatusOK, rec)
}

func (h *Handler) rotateToken(w http.ResponseWriter, r *http.Request) {
	id := sharedkernel.EdgeID(chi.URLParam(r, "id"))
	rec, err := h.Repo.Get(r.Context(), id)
	if errors.Is(err, edge.ErrNotFound) {
		response.Fail(w, apierr.ErrEdgeGetNotFound, "instance not found")
		return
	}
	if err != nil {
		response.FailErr(w, apierr.ErrEdgeRotateTokenFailed, err)
		return
	}
	plain, err := h.mintAndStoreToken(rec)
	if err != nil {
		response.FailErr(w, apierr.ErrEdgeRotateTokenFailed, err)
		return
	}
	if err := h.Repo.UpdateAgentTokenEnc(r.Context(), rec.ID, rec.AgentTokenEnc); err != nil {
		response.FailErr(w, apierr.ErrEdgeRotateTokenFailed, err)
		return
	}
	rec.UpdatedAt = time.Now().UTC()
	response.OKStatus(w, http.StatusOK, toDTO(rec, plain))
}

func (h *Handler) patch(w http.ResponseWriter, r *http.Request) {
	id := sharedkernel.EdgeID(chi.URLParam(r, "id"))
	rec, err := h.Repo.Get(r.Context(), id)
	if errors.Is(err, edge.ErrNotFound) {
		response.Fail(w, apierr.ErrEdgeGetNotFound, "instance not found")
		return
	}
	if err != nil {
		response.FailErr(w, apierr.ErrEdgeUpdateFailed, err)
		return
	}
	var req patchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Fail(w, apierr.ErrEdgeCreateInvalidJSON, "invalid json")
		return
	}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			response.Fail(w, apierr.ErrEdgeCreateNameRequired, "name required")
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
	if req.SubscribeTopics != nil {
		topics := topicdomain.NormalizeTopics(req.SubscribeTopics)
		if h.Topics == nil {
			response.Fail(w, apierr.ErrEdgeUpdateNotConfigured, "topics repository not configured")
			return
		}
		for _, key := range topics {
			got, err := h.Topics.Get(r.Context(), key)
			if err != nil || !got.Enabled {
				response.Fail(w, apierr.ErrEdgeUpdateTopicUnknown, "unknown or disabled topic: "+key)
				return
			}
		}
		rec.SubscribeTopics = topics
	}
	if req.Hardware != nil {
		hw := *req.Hardware
		hw.CollectedAt = time.Now().UTC()
		rec.Hardware = hw
	}
	rec.UpdatedAt = time.Now().UTC()
	if err := h.Repo.Upsert(r.Context(), rec); err != nil {
		response.FailErr(w, apierr.ErrEdgeUpdateFailed, err)
		return
	}
	if req.Hardware != nil {
		if err := h.Repo.UpdateHardware(r.Context(), rec.ID, rec.Hardware); err != nil {
			response.FailErr(w, apierr.ErrEdgeUpdateFailed, err)
			return
		}
	}
	if req.SubscribeTopics != nil {
		if err := h.Repo.UpdateSubscribeTopics(r.Context(), rec.ID, rec.SubscribeTopics); err != nil {
			response.FailErr(w, apierr.ErrEdgeUpdateFailed, err)
			return
		}
	}
	if req.RefreshHardware != nil {
		if err := h.Repo.SetHardwareRefreshRequested(r.Context(), rec.ID, *req.RefreshHardware); err != nil {
			response.FailErr(w, apierr.ErrEdgeUpdateFailed, err)
			return
		}
	}
	if err := h.refreshPool(r); err != nil {
		response.FailErr(w, apierr.ErrEdgeUpdateFailed, err)
		return
	}
	got, err := h.Repo.Get(r.Context(), id)
	if err != nil {
		response.FailErr(w, apierr.ErrEdgeUpdateFailed, err)
		return
	}
	h.writeInstance(w, http.StatusOK, got)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id := sharedkernel.EdgeID(chi.URLParam(r, "id"))
	var body struct {
		AckReferences bool `json:"ack_references"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if h.DeleteWithCleanup == nil {
		response.Fail(w, apierr.ErrEdgeDeleteNotConfigured, "delete cleanup not configured")
		return
	}
	summary, err := h.DeleteWithCleanup(r.Context(), id, body.AckReferences)
	if errors.Is(err, edgeapp.ErrNeedsAck) {
		response.Fail(w, apierr.ErrEdgeDeleteConflict, "edge has running tasks; confirm with ack_references to mark them failed")
		return
	}
	if errors.Is(err, edge.ErrNotFound) {
		response.Fail(w, apierr.ErrEdgeGetNotFound, "instance not found")
		return
	}
	if err != nil {
		response.FailErr(w, apierr.ErrEdgeDeleteFailed, err)
		return
	}
	if h.Presence != nil {
		h.Presence.Remove(id)
	}
	if err := h.refreshPool(r); err != nil {
		response.FailErr(w, apierr.ErrEdgeDeleteFailed, err)
		return
	}
	response.OKStatus(w, http.StatusOK, map[string]any{
		"deleted":      true,
		"failed_tasks": summary.FailedTasks,
	})
}

func (h *Handler) listTasks(w http.ResponseWriter, r *http.Request) {
	id := sharedkernel.EdgeID(chi.URLParam(r, "id"))
	if _, err := h.Repo.Get(r.Context(), id); errors.Is(err, edge.ErrNotFound) {
		response.Fail(w, apierr.ErrEdgeGetNotFound, "instance not found")
		return
	} else if err != nil {
		response.FailErr(w, apierr.ErrEdgeListTasksFailed, err)
		return
	}
	q := domain.ListByInstanceQuery{}
	if s := r.URL.Query().Get("status"); s != "" {
		q.Status = sharedkernel.TaskStatus(s)
	}
	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			response.Fail(w, apierr.ErrEdgeListTasksInvalidLimit, "invalid limit")
			return
		}
		q.Limit = n
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			response.Fail(w, apierr.ErrEdgeListTasksInvalidOffset, "invalid offset")
			return
		}
		q.Offset = n
	}
	list, err := h.Tasks.ListByInstance(r.Context(), id, q)
	if err != nil {
		response.FailErr(w, apierr.ErrEdgeListTasksFailed, err)
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
	response.OKStatus(w, http.StatusOK, out)
}

type statsDTO struct {
	TaskCount   int      `json:"task_count"`
	RuntimeMS   int64    `json:"runtime_ms"`
	SuccessRate *float64 `json:"success_rate"`
}

func (h *Handler) stats(w http.ResponseWriter, r *http.Request) {
	id := sharedkernel.EdgeID(chi.URLParam(r, "id"))
	if _, err := h.Repo.Get(r.Context(), id); errors.Is(err, edge.ErrNotFound) {
		response.Fail(w, apierr.ErrEdgeGetNotFound, "instance not found")
		return
	} else if err != nil {
		response.FailErr(w, apierr.ErrEdgeStatsFailed, err)
		return
	}
	out := statsDTO{}
	if h.Tasks == nil {
		response.OKStatus(w, http.StatusOK, out)
		return
	}
	list, err := h.Tasks.ListByInstance(r.Context(), id, domain.ListByInstanceQuery{})
	if err != nil {
		response.FailErr(w, apierr.ErrEdgeStatsFailed, err)
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
	response.OKStatus(w, http.StatusOK, out)
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
		response.Fail(w, apierr.ErrEdgeMetricsNotConfigured, "metrics not configured")
		return
	}
	if _, err := h.Repo.Get(r.Context(), id); errors.Is(err, edge.ErrNotFound) {
		response.Fail(w, apierr.ErrEdgeGetNotFound, "instance not found")
		return
	} else if err != nil {
		response.FailErr(w, apierr.ErrEdgeMetricsFailed, err)
		return
	}
	since, until, err := parseMetricsRange(r)
	if err != nil {
		response.FailErr(w, apierr.ErrEdgeMetricsInvalidQuery, err)
		return
	}
	series, err := h.Metrics.ListSince(r.Context(), id, since, 100000)
	if err != nil {
		response.FailErr(w, apierr.ErrEdgeMetricsFailed, err)
		return
	}
	var latest *edge.Metrics
	if len(series) > 0 {
		cp := series[len(series)-1]
		latest = &cp
	}
	filled := fillMetricsGaps(series, since, until, metricsBucketSize(until.Sub(since)))
	if filled == nil {
		filled = []edge.Metrics{}
	}
	response.OKStatus(w, http.StatusOK, map[string]any{"latest": latest, "series": filled})
}

const maxMetricsRange = 90 * 24 * time.Hour

// parseMetricsRange resolves the requested window. Presets (window=1h/6h/24h)
// are relative to now; a custom range is expressed as explicit RFC3339 from/to.
func parseMetricsRange(r *http.Request) (time.Time, time.Time, error) {
	fromRaw := strings.TrimSpace(r.URL.Query().Get("from"))
	toRaw := strings.TrimSpace(r.URL.Query().Get("to"))
	now := time.Now().UTC()
	if fromRaw == "" && toRaw == "" {
		window, err := parseMetricsWindow(r.URL.Query().Get("window"))
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		return now.Add(-window), now, nil
	}
	from, err := time.Parse(time.RFC3339, fromRaw)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid from %q", fromRaw)
	}
	to, err := time.Parse(time.RFC3339, toRaw)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid to %q", toRaw)
	}
	if !from.Before(to) {
		return time.Time{}, time.Time{}, errors.New("from must be before to")
	}
	if to.After(now.Add(time.Minute)) {
		return time.Time{}, time.Time{}, errors.New("to cannot be in the future")
	}
	if to.Sub(from) > maxMetricsRange {
		return time.Time{}, time.Time{}, fmt.Errorf("range exceeds %s", maxMetricsRange)
	}
	return from, to, nil
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

// metricsBucketSize picks a fixed bucket so the returned series stays small
// enough for charts while keeping the whole window covered.
func metricsBucketSize(window time.Duration) time.Duration {
	switch {
	case window <= time.Hour:
		return time.Minute
	case window <= 6*time.Hour:
		return time.Minute
	case window <= 24*time.Hour:
		return 5 * time.Minute
	case window <= 7*24*time.Hour:
		return 30 * time.Minute
	case window <= 30*24*time.Hour:
		return 2 * time.Hour
	default:
		return 6 * time.Hour
	}
}

var zeroDiskRate = 0.0

func zeroMetrics(at time.Time) edge.Metrics {
	return edge.Metrics{
		CollectedAt:          at,
		DiskReadBytesPerSec:  &zeroDiskRate,
		DiskWriteBytesPerSec: &zeroDiskRate,
	}
}

// fillMetricsGaps buckets the window into fixed intervals and zero-fills
// buckets without samples, so charts render the full timeline instead of
// connecting the last pre-downtime sample straight to the first post-boot one.
func fillMetricsGaps(series []edge.Metrics, since, until time.Time, bucket time.Duration) []edge.Metrics {
	if len(series) == 0 {
		return nil
	}
	buckets := int(until.Sub(since) / bucket)
	if buckets <= 0 {
		return series
	}
	out := make([]edge.Metrics, 0, buckets)
	idx := 0
	for i := 0; i < buckets; i++ {
		start := since.Add(time.Duration(i) * bucket)
		end := start.Add(bucket)
		var sample *edge.Metrics
		for idx < len(series) && series[idx].CollectedAt.Before(end) {
			if !series[idx].CollectedAt.Before(start) {
				cp := series[idx]
				sample = &cp
			}
			idx++
		}
		if sample == nil {
			out = append(out, zeroMetrics(start.Add(bucket/2)))
			continue
		}
		out = append(out, *sample)
	}
	return out
}
