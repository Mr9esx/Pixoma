package menucards

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	mcdomain "github.com/mr9esx/comfyui_tgbot/internal/menucard/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/menucard/infrastructure/persistence"
)

// Handler serves menu tree management under /api/v1/channels/{id}.
type Handler struct {
	Repo persistence.CardRepository
}

func NewHandler(repo persistence.CardRepository) *Handler {
	return &Handler{Repo: repo}
}

func (h *Handler) Mount(r chi.Router) {
	r.Get("/menu", h.getMenu)
	r.Put("/menu", h.putMenu)
	r.Get("/cards", h.goneCards)
	r.Post("/cards", h.goneCards)
	r.Patch("/cards/{cardID}", h.goneCards)
	r.Delete("/cards/{cardID}", h.goneCards)
	r.Get("/cards/{cardID}/references", h.goneCards)
}

func (h *Handler) getMenu(w http.ResponseWriter, r *http.Request) {
	channelID := chi.URLParam(r, "id")
	tree, err := h.Repo.GetTree(r.Context(), channelID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		writeJSON(w, http.StatusOK, mcdomain.DefaultMenuTree(channelID))
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, tree)
}

func (h *Handler) putMenu(w http.ResponseWriter, r *http.Request) {
	channelID := chi.URLParam(r, "id")
	var tree mcdomain.MenuTree
	if err := json.NewDecoder(r.Body).Decode(&tree); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	tree.ID = channelID
	if err := mcdomain.ValidateTree(tree, nil); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.Repo.PutTree(r.Context(), channelID, tree); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, tree)
}

func (h *Handler) goneCards(w http.ResponseWriter, r *http.Request) {
	writeErr(w, http.StatusGone, "cards are nested in the menu tree")
}

// ListWorkflowPlacements returns where a workflow is referenced in menu trees.
func (h *Handler) ListWorkflowPlacements(w http.ResponseWriter, r *http.Request) {
	placements, err := h.Repo.WorkflowPlacements(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	type step struct {
		ID    string `json:"id"`
		Label string `json:"label"`
	}
	type placementDTO struct {
		ChannelID   string `json:"channel_id"`
		ChannelName string `json:"channel_name"`
		ItemID      string `json:"item_id"`
		Kind        string `json:"kind"`
		Path        []step `json:"path"`
	}
	out := make([]placementDTO, 0, len(placements))
	for _, p := range placements {
		path := make([]step, 0, len(p.Labels))
		for _, lab := range p.Labels {
			path = append(path, step{ID: p.ItemID, Label: lab})
		}
		if len(path) == 0 {
			path = []step{{ID: p.ItemID, Label: p.Label}}
		}
		out = append(out, placementDTO{
			ChannelID:   p.ChannelID,
			ChannelName: p.ChannelName,
			ItemID:      p.ItemID,
			Kind:        p.Kind,
			Path:        path,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}