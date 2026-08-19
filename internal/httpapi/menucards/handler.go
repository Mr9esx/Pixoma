package menucards

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	mcdomain "github.com/mr9esx/comfyui_tgbot/internal/menucard/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/menucard/infrastructure/persistence"
)

// Handler serves the new menu/card management API under /api/v1/channels/{id}.
type Handler struct {
	Repo persistence.CardRepository
}

func NewHandler(repo persistence.CardRepository) *Handler {
	return &Handler{Repo: repo}
}

func (h *Handler) Mount(r chi.Router) {
	r.Get("/menu", h.getMenu)
	r.Put("/menu", h.putMenu)
	r.Get("/cards", h.listCards)
	r.Post("/cards", h.createCard)
	r.Patch("/cards/{cardID}", h.updateCard)
	r.Delete("/cards/{cardID}", h.deleteCard)
	r.Get("/cards/{cardID}/references", h.cardReferences)
}

func (h *Handler) getMenu(w http.ResponseWriter, r *http.Request) {
	menu, err := h.Repo.GetMenu(r.Context(), chi.URLParam(r, "id"))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		writeJSON(w, http.StatusOK, mcdomain.Menu{ID: chi.URLParam(r, "id")})
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, menu)
}

func (h *Handler) putMenu(w http.ResponseWriter, r *http.Request) {
	var menu mcdomain.Menu
	if err := json.NewDecoder(r.Body).Decode(&menu); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := mcdomain.ValidateMenu(menu); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.Repo.PutMenu(r.Context(), chi.URLParam(r, "id"), menu); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, menu)
}

func (h *Handler) listCards(w http.ResponseWriter, r *http.Request) {
	cards, err := h.Repo.ListCards(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q != "" {
		filtered := cards[:0]
		for _, c := range cards {
			if strings.Contains(c.Name, q) {
				filtered = append(filtered, c)
			}
		}
		cards = filtered
	}
	writeJSON(w, http.StatusOK, cards)
}

func (h *Handler) createCard(w http.ResponseWriter, r *http.Request) {
	var card mcdomain.Card
	if err := json.NewDecoder(r.Body).Decode(&card); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := mcdomain.ValidateCard(card); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.Repo.CreateCard(r.Context(), chi.URLParam(r, "id"), card); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, card)
}

func (h *Handler) updateCard(w http.ResponseWriter, r *http.Request) {
	var card mcdomain.Card
	if err := json.NewDecoder(r.Body).Decode(&card); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	card.ID = chi.URLParam(r, "cardID")
	if err := mcdomain.ValidateCard(card); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.Repo.UpdateCard(r.Context(), chi.URLParam(r, "id"), card); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, card)
}

func (h *Handler) deleteCard(w http.ResponseWriter, r *http.Request) {
	channelID := chi.URLParam(r, "id")
	cardID := chi.URLParam(r, "cardID")
	refs, err := h.Repo.CardReferences(r.Context(), channelID, cardID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if len(refs) > 0 {
		writeJSON(w, http.StatusConflict, map[string]any{
			"error":      "card is referenced",
			"references": refs,
		})
		return
	}
	if err := h.Repo.DeleteCard(r.Context(), channelID, cardID); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) cardReferences(w http.ResponseWriter, r *http.Request) {
	refs, err := h.Repo.CardReferences(r.Context(), chi.URLParam(r, "id"), chi.URLParam(r, "cardID"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, refs)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
