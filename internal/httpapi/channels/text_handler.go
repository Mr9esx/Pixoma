// Package channels serves the admin APIs for configuring the Telegram copy
// templates rendered by bot adapters. Templates are scoped: the platform
// default lives at /api/v1/text-templates, per-channel copies live under
// /api/v1/channels/{id}/text-templates.
package channels

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	templates "github.com/Mr9esx/Pixoma/internal/channels/domain/templates"
	textpersist "github.com/Mr9esx/Pixoma/internal/channels/infrastructure/persistence"
)

// Handler serves both the platform-default templates and per-channel overrides.
type TextHandler struct {
	Store *textpersist.Store
}

// Mount registers the platform-default routes (Settings → 默认文案).
func (h *TextHandler) Mount(r chi.Router) {
	h.mount(r, "/", func(*http.Request) string { return templates.GlobalDefaultID })
}

// MountChannel registers per-channel routes on an {id} channel sub-router.
func (h *TextHandler) MountChannel(r chi.Router) {
	h.mount(r, "/text-templates", func(req *http.Request) string {
		return chi.URLParam(req, "id")
	})
}

// mount wires CRUD routes under base, resolving the storage scope per request.
func (h *TextHandler) mount(r chi.Router, base string, scopeOf func(*http.Request) string) {
	r.Get(base, func(w http.ResponseWriter, req *http.Request) {
		h.list(w, req, scopeOf(req))
	})
	r.Put(base, func(w http.ResponseWriter, req *http.Request) {
		h.save(w, req, scopeOf(req))
	})
	r.Post(base+"/reset", func(w http.ResponseWriter, req *http.Request) {
		h.reset(w, req, scopeOf(req))
	})
}

type itemDTO struct {
	Key         string   `json:"key"`
	Group       string   `json:"group"`
	Description string   `json:"description"`
	Default     string   `json:"default"`
	Value       string   `json:"value"`
	Variables   []string `json:"variables,omitempty"`
}

func (h *TextHandler) list(w http.ResponseWriter, r *http.Request, channelID string) {
	stored := map[string]string{}
	if h.Store != nil {
		var err error
		stored, err = h.Store.Load(r.Context(), channelID)
		if err != nil {
			textWriteErr(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	specs := templates.Specs()
	out := make([]itemDTO, 0, len(specs))
	for _, s := range specs {
		out = append(out, itemDTO{
			Key:         s.Key,
			Group:       s.Group,
			Description: s.Description,
			Default:     s.Default,
			Value:       stored[s.Key],
			Variables:   s.Variables,
		})
	}
	textWriteJSON(w, http.StatusOK, out)
}

type saveRequest struct {
	Templates map[string]string `json:"templates"`
}

func (h *TextHandler) save(w http.ResponseWriter, r *http.Request, channelID string) {
	if h.Store == nil {
		textWriteErr(w, http.StatusServiceUnavailable, "text templates unavailable")
		return
	}
	var req saveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		textWriteErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.Templates == nil {
		textWriteErr(w, http.StatusBadRequest, "templates required")
		return
	}
	if err := h.Store.Save(r.Context(), channelID, req.Templates); err != nil {
		textWriteErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.list(w, r, channelID)
}

type resetRequest struct {
	Keys []string `json:"keys"`
}

func (h *TextHandler) reset(w http.ResponseWriter, r *http.Request, channelID string) {
	if h.Store == nil {
		textWriteErr(w, http.StatusServiceUnavailable, "text templates unavailable")
		return
	}
	var req resetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		textWriteErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := h.Store.Reset(r.Context(), channelID, req.Keys); err != nil {
		textWriteErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.list(w, r, channelID)
}

func textWriteJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func textWriteErr(w http.ResponseWriter, code int, msg string) {
	textWriteJSON(w, code, map[string]string{"error": msg})
}
