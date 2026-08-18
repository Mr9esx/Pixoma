package agent

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/edge"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/presence"
	runtimedomain "github.com/mr9esx/comfyui_tgbot/internal/runtime/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

const defaultLease = 90 * time.Second

// StatusApplier applies Edge-reported task status (orchestrator OnStatus).
type StatusApplier interface {
	OnStatus(ctx context.Context, ev sharedkernel.TaskStatusEvent) error
}

// Handler serves Edge Agent pull APIs under /agent/v1.
type Handler struct {
	Token    string
	Verify   func(ctx context.Context, edgeID sharedkernel.EdgeID, token string) bool
	Tasks    runtimedomain.TaskRepository
	Status   StatusApplier
	Presence *presence.Store
	Edges    edge.Repository
	Metrics  edge.MetricsRepository
	Lease    time.Duration
	Now      func() time.Time
}

// Mount registers claim / heartbeat / status routes.
func (h *Handler) Mount(r chi.Router) {
	r.Post("/presence", h.presence)
	r.Get("/jobs/claim", h.claim)
	r.Post("/jobs/{id}/heartbeat", h.heartbeat)
	r.Post("/jobs/{id}/status", h.status)
}

func (h *Handler) lease() time.Duration {
	if h.Lease > 0 {
		return h.Lease
	}
	return defaultLease
}

func (h *Handler) now() time.Time {
	if h.Now != nil {
		return h.Now()
	}
	return time.Now().UTC()
}

func (h *Handler) authorize(r *http.Request, edgeID sharedkernel.EdgeID) bool {
	hdr := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if !strings.HasPrefix(hdr, prefix) {
		return false
	}
	got := strings.TrimSpace(strings.TrimPrefix(hdr, prefix))
	if got == "" || edgeID == "" {
		return false
	}
	if h.Verify != nil {
		return h.Verify(r.Context(), edgeID, got)
	}
	want := strings.TrimSpace(h.Token)
	if want == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}

func (h *Handler) claim(w http.ResponseWriter, r *http.Request) {
	edgeID := sharedkernel.EdgeID(strings.TrimSpace(r.URL.Query().Get("edge_id")))
	if edgeID == "" {
		writeErr(w, http.StatusBadRequest, "edge_id required")
		return
	}
	if !h.authorize(r, edgeID) {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	wait := parseWait(r.URL.Query().Get("wait"))
	deadline := h.now().Add(wait)
	for {
		h.touch(edgeID)
		_, _ = h.Tasks.RequeueExpiredLeases(r.Context(), h.now())
		claimed, err := h.Tasks.ClaimNextWithLease(r.Context(), edgeID, h.lease(), h.now())
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if claimed != nil {
			writeJSON(w, http.StatusOK, map[string]any{
				"task_id":     string(claimed.ID),
				"edge_id":     string(claimed.EdgeID),
				"job_ref":     claimed.JobRef,
				"lease_until": claimed.LeaseUntil,
			})
			return
		}
		if wait <= 0 || !h.now().Before(deadline) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		select {
		case <-r.Context().Done():
			writeErr(w, http.StatusRequestTimeout, "canceled")
			return
		case <-time.After(200 * time.Millisecond):
		}
	}
}

func (h *Handler) heartbeat(w http.ResponseWriter, r *http.Request) {
	id := sharedkernel.TaskID(chi.URLParam(r, "id"))
	var body struct {
		EdgeID string `json:"edge_id"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	edgeID := sharedkernel.EdgeID(strings.TrimSpace(body.EdgeID))
	if edgeID == "" {
		writeErr(w, http.StatusBadRequest, "edge_id required")
		return
	}
	if !h.authorize(r, edgeID) {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ok, err := h.Tasks.HeartbeatLease(r.Context(), id, edgeID, h.lease(), h.now())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !ok {
		writeErr(w, http.StatusConflict, "heartbeat rejected")
		return
	}
	h.touch(edgeID)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) presence(w http.ResponseWriter, r *http.Request) {
	if h.Presence == nil {
		writeErr(w, http.StatusInternalServerError, "presence not configured")
		return
	}
	var body struct {
		EdgeID       string         `json:"edge_id"`
		ComfyRunning bool           `json:"comfy_running"`
		StartedAt    *time.Time     `json:"started_at"`
		ComfyVersion string         `json:"comfy_version"`
		Hardware     *edge.Hardware `json:"hardware"`
		Metrics      *edge.Metrics  `json:"metrics"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	edgeID := sharedkernel.EdgeID(strings.TrimSpace(body.EdgeID))
	if edgeID == "" {
		writeErr(w, http.StatusBadRequest, "edge_id required")
		return
	}
	if !h.authorize(r, edgeID) {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	h.Presence.Report(edgeID, body.ComfyRunning)
	if h.Metrics != nil && body.Metrics != nil && !body.Metrics.CollectedAt.IsZero() {
		if err := h.Metrics.Append(r.Context(), edgeID, *body.Metrics); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	refresh := false
	if h.Edges != nil {
		rec, err := h.Edges.Get(r.Context(), edgeID)
		if err != nil && !errors.Is(err, edge.ErrNotFound) {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if rec != nil {
			if body.Hardware != nil && edge.ShouldWriteHardware(rec.Hardware, rec.HardwareRefreshRequested) {
				if err := h.Edges.UpdateHardware(r.Context(), edgeID, *body.Hardware); err != nil {
					writeErr(w, http.StatusInternalServerError, err.Error())
					return
				}
			}
			if body.StartedAt != nil || body.ComfyVersion != "" {
				if err := h.Edges.UpdatePresenceInfo(
					r.Context(),
					edgeID,
					body.StartedAt,
					body.ComfyVersion,
				); err != nil {
					writeErr(w, http.StatusInternalServerError, err.Error())
					return
				}
			}
			if latest, err := h.Edges.Get(r.Context(), edgeID); err == nil && latest != nil {
				refresh = latest.HardwareRefreshRequested
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"refresh_hardware": refresh})
}

func (h *Handler) touch(id sharedkernel.EdgeID) {
	if h.Presence == nil {
		return
	}
	h.Presence.Touch(id)
}

func (h *Handler) status(w http.ResponseWriter, r *http.Request) {
	if h.Status == nil {
		writeErr(w, http.StatusInternalServerError, "status not configured")
		return
	}
	id := sharedkernel.TaskID(chi.URLParam(r, "id"))
	var body struct {
		EdgeID    string                 `json:"edge_id"`
		Status    string                 `json:"status"`
		PromptID  string                 `json:"prompt_id"`
		Outputs   []sharedkernel.BlobRef `json:"outputs"`
		ErrorCode string                 `json:"error_code"`
		ErrorMsg  string                 `json:"error_msg"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 4<<20)).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if strings.TrimSpace(body.EdgeID) == "" {
		writeErr(w, http.StatusBadRequest, "edge_id required")
		return
	}
	if !h.authorize(r, sharedkernel.EdgeID(body.EdgeID)) {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	cur, err := h.Tasks.Get(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusConflict, "task not found")
		return
	}
	if cur.EdgeID != "" && cur.EdgeID != sharedkernel.EdgeID(body.EdgeID) {
		writeErr(w, http.StatusConflict, "stale holder")
		return
	}
	ev := sharedkernel.TaskStatusEvent{
		TaskID:    id,
		EdgeID:    sharedkernel.EdgeID(body.EdgeID),
		Status:    sharedkernel.TaskStatus(body.Status),
		PromptID:  body.PromptID,
		Outputs:   body.Outputs,
		ErrorCode: body.ErrorCode,
		ErrorMsg:  body.ErrorMsg,
		At:        h.now(),
	}
	if err := h.Status.OnStatus(r.Context(), ev); err != nil {
		writeErr(w, http.StatusConflict, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func parseWait(raw string) time.Duration {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d < 0 {
		return 0
	}
	if d > 60*time.Second {
		return 60 * time.Second
	}
	return d
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
