package routing

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/Mr9esx/Pixoma/internal/apierr"
	"github.com/Mr9esx/Pixoma/internal/response"
	"github.com/Mr9esx/Pixoma/internal/tasks/domain/condition"
)

// Handler serves the condition attribute catalog for the admin editor.
type Handler struct {
	Registry *condition.Registry
}

// Mount registers chi routes (caller mounts under /api/v1/routing).
func (h *Handler) Mount(r chi.Router) {
	r.Get("/attributes", h.attributes)
}

func (h *Handler) attributes(w http.ResponseWriter, _ *http.Request) {
	if h.Registry == nil {
		response.Fail(w, apierr.ErrRoutingAttributesNotConfigured, "registry not configured")
		return
	}
	attrs := h.Registry.Attributes()
	response.OKStatus(w, http.StatusOK, map[string]any{"attributes": attrs})
}
