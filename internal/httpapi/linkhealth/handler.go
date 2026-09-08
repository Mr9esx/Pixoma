package linkhealth

import (
	"encoding/json"
	"net/http"

	"github.com/Mr9esx/Pixoma/internal/packaging/linkhealth"
)

// Handler serves GET /api/v1/link-health.
type Handler struct {
	Snapshot func(*http.Request) (linkhealth.Snapshot, error)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.Snapshot == nil {
		writeErr(w, http.StatusInternalServerError, "link health not configured")
		return
	}
	snap, err := h.Snapshot(r)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, linkhealth.Assemble(snap))
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
