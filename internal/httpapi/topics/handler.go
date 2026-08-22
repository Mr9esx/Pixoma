package topics

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/topic"
	runtimedomain "github.com/mr9esx/comfyui_tgbot/internal/runtime/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

// KeyPattern is the shared topic key format.
var KeyPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,62}[a-z0-9])?$`)

// Handler serves /api/v1/topics CRUD.
type Handler struct {
	Repo topic.Repository
	// Tasks optionally enables /{key}/stats aggregation.
	Tasks runtimedomain.TaskRepository
	// CountCaseRefs counts Case routing rules referencing a topic key.
	CountCaseRefs func(ctx context.Context, key string) (int, error)
	// CountEdgeRefs counts Edge subscriptions referencing a topic key.
	CountEdgeRefs func(ctx context.Context, key string) (int, error)
}

// Mount registers chi routes (caller mounts under /api/v1/topics).
func (h *Handler) Mount(r chi.Router) {
	r.Get("/", h.list)
	r.Post("/", h.create)
	r.Get("/{key}", h.get)
	r.Get("/{key}/stats", h.stats)
	r.Put("/{key}", h.update)
	r.Delete("/{key}", h.delete)
}

type topicDTO struct {
	Key       string    `json:"key"`
	Name      string    `json:"name"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func toDTO(t topic.Topic) topicDTO {
	return topicDTO{
		Key:       t.Key,
		Name:      t.Name,
		Enabled:   t.Enabled,
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
	}
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	var enabled *bool
	if raw := strings.TrimSpace(r.URL.Query().Get("enabled")); raw != "" {
		v := raw == "true"
		enabled = &v
	}
	list, err := h.Repo.List(r.Context(), enabled)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]topicDTO, 0, len(list))
	for _, t := range list {
		out = append(out, toDTO(t))
	}
	writeJSON(w, http.StatusOK, out)
}

type createRequest struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	req.Key = strings.TrimSpace(req.Key)
	if !KeyPattern.MatchString(req.Key) {
		writeErr(w, http.StatusBadRequest, "invalid topic key (lowercase letters/digits/hyphens)")
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeErr(w, http.StatusBadRequest, "name required")
		return
	}
	now := time.Now().UTC()
	if err := h.Repo.Create(r.Context(), topic.Topic{
		Key:       req.Key,
		Name:      strings.TrimSpace(req.Name),
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		if errors.Is(err, topic.ErrTopicConflict) {
			writeErr(w, http.StatusConflict, "topic key already exists")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	got, err := h.Repo.Get(r.Context(), req.Key)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, toDTO(*got))
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	got, err := h.Repo.Get(r.Context(), key)
	if err != nil {
		if errors.Is(err, topic.ErrTopicNotFound) {
			writeErr(w, http.StatusNotFound, "topic not found")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toDTO(*got))
}

type updateRequest struct {
	Name    *string `json:"name"`
	Enabled *bool   `json:"enabled"`
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	var req updateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	got, err := h.Repo.Get(r.Context(), key)
	if err != nil {
		writeErr(w, http.StatusNotFound, "topic not found")
		return
	}
	if req.Name != nil {
		got.Name = strings.TrimSpace(*req.Name)
	}
	if req.Enabled != nil {
		if key == topic.DefaultKey && !*req.Enabled {
			writeErr(w, http.StatusConflict, "default topic cannot be disabled")
			return
		}
		got.Enabled = *req.Enabled
	}
	got.UpdatedAt = time.Now().UTC()
	if err := h.Repo.Update(r.Context(), *got); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toDTO(*got))
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	if key == topic.DefaultKey {
		writeErr(w, http.StatusConflict, "default topic cannot be deleted")
		return
	}
	var refs int
	if h.CountCaseRefs != nil {
		n, err := h.CountCaseRefs(r.Context(), key)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		refs += n
	}
	if h.CountEdgeRefs != nil {
		n, err := h.CountEdgeRefs(r.Context(), key)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		refs += n
	}
	if refs > 0 {
		writeErr(w, http.StatusConflict, "topic is referenced by cases or edges")
		return
	}
	if err := h.Repo.Delete(r.Context(), key); err != nil {
		if errors.Is(err, topic.ErrTopicNotFound) {
			writeErr(w, http.StatusNotFound, "topic not found")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type topicStatsDTO struct {
	TaskCount   int                  `json:"task_count"`
	Status      map[string]int       `json:"status"`
	SuccessRate *float64             `json:"success_rate"`
	ErrorCodes  []errorCodeCountDTO  `json:"error_codes"`
	RuntimeMS   runtimeStatsDTO      `json:"runtime_ms"`
	Throughput  []throughputPointDTO `json:"throughput"`
	From        time.Time            `json:"from"`
	To          time.Time            `json:"to"`
}

type errorCodeCountDTO struct {
	Code  string `json:"code"`
	Count int    `json:"count"`
}

type runtimeStatsDTO struct {
	SumMS int64    `json:"sum_ms"`
	AvgMS *float64 `json:"avg_ms"`
	Count int      `json:"count"`
}

type throughputPointDTO struct {
	Ts    time.Time `json:"ts"`
	Count int       `json:"count"`
}

// stats aggregates task-level metrics for a topic. When the optional Tasks
// repository is not wired, it returns an empty-but-valid payload.
func (h *Handler) stats(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	if _, err := h.Repo.Get(r.Context(), key); err != nil {
		if errors.Is(err, topic.ErrTopicNotFound) {
			writeErr(w, http.StatusNotFound, "topic not found")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if h.Tasks == nil {
		writeJSON(w, http.StatusOK, topicStatsDTO{
			Status:     map[string]int{},
			ErrorCodes: []errorCodeCountDTO{},
			Throughput: []throughputPointDTO{},
		})
		return
	}
	from, to := statsWindow(r)
	list, err := h.Tasks.ListByTopic(r.Context(), key, runtimedomain.ListByTopicQuery{
		CreatedFrom: &from,
		CreatedTo:   &to,
		Limit:       100000,
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, aggregateTopicStats(list, from, to))
}

func statsWindow(r *http.Request) (time.Time, time.Time) {
	now := time.Now().UTC()
	from := now.Add(-7 * 24 * time.Hour)
	to := now
	if v := strings.TrimSpace(r.URL.Query().Get("from")); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			from = t.UTC()
		}
	}
	if v := strings.TrimSpace(r.URL.Query().Get("to")); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			to = t.UTC()
		}
	}
	if to.Before(from) {
		to = from
	}
	return from, to
}

func aggregateTopicStats(tasks []*runtimedomain.Task, from, to time.Time) topicStatsDTO {
	out := topicStatsDTO{
		Status:     map[string]int{},
		ErrorCodes: []errorCodeCountDTO{},
		Throughput: []throughputPointDTO{},
		From:       from,
		To:         to,
	}
	statusCounts := map[string]int{}
	errCounts := map[string]int{}
	var okN, failN int
	var runtimeSum int64
	var runtimeN int

	for _, t := range tasks {
		if t == nil {
			continue
		}
		out.TaskCount++
		statusCounts[string(t.Status)]++
		if rt, ok := taskRuntimeMS(t); ok {
			runtimeSum += rt
			runtimeN++
		}
		switch t.Status {
		case sharedkernel.TaskSucceeded:
			okN++
		case sharedkernel.TaskFailed:
			failN++
			code := t.ErrorCode
			if code == "" {
				code = "unknown"
			}
			errCounts[code]++
		}
	}
	out.Status = statusCounts
	if okN+failN > 0 {
		rate := float64(okN) / float64(okN+failN)
		out.SuccessRate = &rate
	}
	out.ErrorCodes = topErrorCodes(errCounts, 10)
	if runtimeN > 0 {
		avg := float64(runtimeSum) / float64(runtimeN)
		out.RuntimeMS = runtimeStatsDTO{SumMS: runtimeSum, AvgMS: &avg, Count: runtimeN}
	}
	out.Throughput = throughputSeries(tasks, from, to)
	return out
}

func taskRuntimeMS(t *runtimedomain.Task) (int64, bool) {
	switch t.Status {
	case sharedkernel.TaskSucceeded, sharedkernel.TaskFailed, sharedkernel.TaskCancelled:
	default:
		return 0, false
	}
	if !t.StartedAt.IsZero() && !t.CompletedAt.IsZero() {
		ms := t.CompletedAt.Sub(t.StartedAt).Milliseconds()
		if ms < 0 {
			ms = 0
		}
		return ms, true
	}
	ms := t.UpdatedAt.Sub(t.CreatedAt).Milliseconds()
	if ms < 0 {
		ms = 0
	}
	return ms, true
}

func topErrorCodes(counts map[string]int, n int) []errorCodeCountDTO {
	type kv struct {
		code  string
		count int
	}
	items := make([]kv, 0, len(counts))
	for code, count := range counts {
		items = append(items, kv{code, count})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].count != items[j].count {
			return items[i].count > items[j].count
		}
		return items[i].code < items[j].code
	})
	if n > len(items) {
		n = len(items)
	}
	out := make([]errorCodeCountDTO, 0, n)
	for _, it := range items[:n] {
		out = append(out, errorCodeCountDTO{Code: it.code, Count: it.count})
	}
	return out
}

func throughputSeries(tasks []*runtimedomain.Task, from, to time.Time) []throughputPointDTO {
	window := to.Sub(from)
	bucket := 24 * time.Hour
	if window <= 48*time.Hour {
		bucket = time.Hour
	}
	n := int(window/bucket) + 1
	if n <= 0 {
		n = 1
	}
	if n > 2000 {
		n = 2000
	}
	counts := make([]int, n)
	for _, t := range tasks {
		if t == nil {
			continue
		}
		idx := int(t.CreatedAt.Sub(from) / bucket)
		if idx < 0 {
			idx = 0
		}
		if idx >= n {
			idx = n - 1
		}
		counts[idx]++
	}
	out := make([]throughputPointDTO, 0, n)
	for i, c := range counts {
		out = append(out, throughputPointDTO{Ts: from.Add(time.Duration(i) * bucket), Count: c})
	}
	return out
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
