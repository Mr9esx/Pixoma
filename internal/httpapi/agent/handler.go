package agent

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

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
	Token  string
	Tasks  runtimedomain.TaskRepository
	Status StatusApplier
	Lease  time.Duration
	Now    func() time.Time
}

// Mount registers claim / heartbeat / status routes.
func (h *Handler) Mount(r chi.Router) {
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

func (h *Handler) authorize(r *http.Request) bool {
	hdr := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if !strings.HasPrefix(hdr, prefix) {
		return false
	}
	got := strings.TrimSpace(strings.TrimPrefix(hdr, prefix))
	want := strings.TrimSpace(h.Token)
	if want == "" || got == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}

func (h *Handler) claim(w http.ResponseWriter, r *http.Request) {
	if !h.authorize(r) {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	instanceID := sharedkernel.InstanceID(strings.TrimSpace(r.URL.Query().Get("instance_id")))
	if instanceID == "" {
		writeErr(w, http.StatusBadRequest, "instance_id required")
		return
	}
	wait := parseWait(r.URL.Query().Get("wait"))
	deadline := h.now().Add(wait)
	for {
		_, _ = h.Tasks.RequeueExpiredLeases(r.Context(), h.now())
		claimed, err := h.Tasks.ClaimNextWithLease(r.Context(), instanceID, h.lease(), h.now())
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if claimed != nil {
			writeJSON(w, http.StatusOK, map[string]any{
				"task_id":     string(claimed.ID),
				"instance_id": string(claimed.InstanceID),
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
	if !h.authorize(r) {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := sharedkernel.TaskID(chi.URLParam(r, "id"))
	var body struct {
		InstanceID string `json:"instance_id"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	ok, err := h.Tasks.HeartbeatLease(r.Context(), id, sharedkernel.InstanceID(body.InstanceID), h.lease(), h.now())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !ok {
		writeErr(w, http.StatusConflict, "heartbeat rejected")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) status(w http.ResponseWriter, r *http.Request) {
	if !h.authorize(r) {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if h.Status == nil {
		writeErr(w, http.StatusInternalServerError, "status not configured")
		return
	}
	id := sharedkernel.TaskID(chi.URLParam(r, "id"))
	var body struct {
		InstanceID string                 `json:"instance_id"`
		Status     string                 `json:"status"`
		PromptID   string                 `json:"prompt_id"`
		Outputs    []sharedkernel.BlobRef `json:"outputs"`
		ErrorCode  string                 `json:"error_code"`
		ErrorMsg   string                 `json:"error_msg"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 4<<20)).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if strings.TrimSpace(body.InstanceID) == "" {
		writeErr(w, http.StatusBadRequest, "instance_id required")
		return
	}
	cur, err := h.Tasks.Get(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusConflict, "task not found")
		return
	}
	if cur.InstanceID != "" && cur.InstanceID != sharedkernel.InstanceID(body.InstanceID) {
		writeErr(w, http.StatusConflict, "stale holder")
		return
	}
	ev := sharedkernel.TaskStatusEvent{
		TaskID:     id,
		InstanceID: sharedkernel.InstanceID(body.InstanceID),
		Status:     sharedkernel.TaskStatus(body.Status),
		PromptID:   body.PromptID,
		Outputs:    body.Outputs,
		ErrorCode:  body.ErrorCode,
		ErrorMsg:   body.ErrorMsg,
		At:         h.now(),
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
