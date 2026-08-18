package channelmenu

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	channelapp "github.com/mr9esx/comfyui_tgbot/internal/channel/application"
	channeldomain "github.com/mr9esx/comfyui_tgbot/internal/channel/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/menu/application"
	"github.com/mr9esx/comfyui_tgbot/internal/menu/domain"
)

// Handler serves channel-scoped menu endpoints under /api/v1/channels/{id}/menu
// and case menu placements under /api/v1/cases/{id}/menu-placements.
type Handler struct {
	Channels *channelapp.Service
	Svc      *application.Service
}

func (h *Handler) MountMenu(r chi.Router) {
	r.Get("/", h.get)
	r.Put("/", h.put)
	r.Get("/extras", h.getExtras)
	r.Put("/extras", h.putExtras)
}

func (h *Handler) requireChannel(w http.ResponseWriter, r *http.Request) bool {
	if h == nil || h.Channels == nil || h.Svc == nil {
		writeErr(w, http.StatusInternalServerError, "channel menu service not configured")
		return false
	}
	if _, err := h.Channels.Get(r.Context(), chi.URLParam(r, "id")); errors.Is(err, channeldomain.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "channel not found")
		return false
	} else if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return false
	}
	return true
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	if !h.requireChannel(w, r) {
		return
	}
	tree, err := h.Svc.Get(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, tree)
}

func (h *Handler) put(w http.ResponseWriter, r *http.Request) {
	if !h.requireChannel(w, r) {
		return
	}
	var body struct {
		Items []domain.MenuNode `json:"items"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	tree, err := h.Svc.Replace(r.Context(), chi.URLParam(r, "id"), domain.MenuTree{Items: body.Items})
	if errors.Is(err, domain.ErrValidation) {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, tree)
}

func (h *Handler) getExtras(w http.ResponseWriter, r *http.Request) {
	if !h.requireChannel(w, r) {
		return
	}
	extras, err := h.Svc.ListExtras(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, extras)
}

func (h *Handler) putExtras(w http.ResponseWriter, r *http.Request) {
	if !h.requireChannel(w, r) {
		return
	}
	var body map[string][]domain.Extra
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	for itemID, extras := range body {
		for _, extra := range extras {
			if extra.ExtraType == "" {
				writeErr(w, http.StatusBadRequest, "extra_type required")
				return
			}
			var probe any
			if err := json.Unmarshal([]byte(extra.ExtraJSON), &probe); err != nil {
				writeErr(w, http.StatusBadRequest, "extra_json must be valid JSON")
				return
			}
			extra.MenuItemID = itemID
		}
	}
	if err := h.Svc.SaveExtras(r.Context(), chi.URLParam(r, "id"), body); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, body)
}

// ListPlacements returns menu paths where a case is mounted.
func (h *Handler) ListPlacements(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.Svc == nil {
		writeErr(w, http.StatusInternalServerError, "channel menu service not configured")
		return
	}
	placements, err := h.Svc.ListPlacementsByCase(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if placements == nil {
		placements = []domain.MenuPlacement{}
	}
	writeJSON(w, http.StatusOK, placements)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
