package linkhealth

import (
	"net/http"

	"github.com/Mr9esx/Pixoma/internal/apierr"
	"github.com/Mr9esx/Pixoma/internal/packaging/linkhealth"
	"github.com/Mr9esx/Pixoma/internal/response"
)

// Handler serves GET /api/v1/link-health.
type Handler struct {
	Snapshot func(*http.Request) (linkhealth.Snapshot, error)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.Snapshot == nil {
		response.Fail(w, apierr.ErrLinkHealthGetNotConfigured, "link health not configured")
		return
	}
	snap, err := h.Snapshot(r)
	if err != nil {
		response.FailErr(w, apierr.ErrLinkHealthGetFailed, err)
		return
	}
	response.OKStatus(w, http.StatusOK, linkhealth.Assemble(snap))
}
