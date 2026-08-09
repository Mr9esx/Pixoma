package tgmenu

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	tgmenuapp "github.com/mr9esx/comfyui_tgbot/internal/tgmenu/application"
	"github.com/mr9esx/comfyui_tgbot/internal/tgmenu/domain"
)

// Handler serves TG menu admin GET/PUT under /api/v1/tg-menu.
type Handler struct {
	Svc *tgmenuapp.Service
}

// Mount registers chi routes on r (caller mounts under /api/v1/tg-menu).
func (h *Handler) Mount(r chi.Router) {
	r.Get("/", h.get)
	r.Put("/", h.put)
}

type menuDTO struct {
	ID        string            `json:"id"`
	Items     []domain.MenuItem `json:"items"`
	UpdatedAt time.Time         `json:"updated_at"`
}

type putBody struct {
	Items []domain.MenuItem `json:"items"`
}

func toDTO(doc domain.MenuDocument) menuDTO {
	return menuDTO{ID: doc.ID, Items: doc.Items, UpdatedAt: doc.UpdatedAt}
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.Svc == nil {
		writeErr(w, http.StatusInternalServerError, "tg menu service not configured")
		return
	}
	doc, err := h.Svc.Get(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toDTO(doc))
}

func (h *Handler) put(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.Svc == nil {
		writeErr(w, http.StatusInternalServerError, "tg menu service not configured")
		return
	}
	var body putBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	doc, err := h.Svc.Replace(r.Context(), body.Items)
	if err != nil {
		if errors.Is(err, domain.ErrValidation) {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toDTO(doc))
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
