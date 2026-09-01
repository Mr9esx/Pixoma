package users

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/mr9esx/comfyui_tgbot/internal/identity/domain"
)

// Handler serves user admin HTTP under /api/v1/users.
type Handler struct {
	Repo     domain.Repository
	Channels ChannelNamesReader
}

type ChannelNamesReader interface {
	ChannelNames(ctx context.Context, ids []string) (map[string]string, error)
}

// Mount registers read routes and the admin-only access update route.
func (h *Handler) Mount(r chi.Router) {
	r.Get("/", h.list)
	r.Get("/{id}", h.get)
	r.Put("/{id}/access", h.updateAccess)
}

type userDTO struct {
	ID             string    `json:"id"`
	ChannelID      string    `json:"channel_id"`
	ChannelName    string    `json:"channel_name,omitempty"`
	ExternalUserID string    `json:"external_user_id"`
	Username       string    `json:"username"`
	FirstName      string    `json:"first_name"`
	LastName       string    `json:"last_name"`
	LanguageCode   string    `json:"language_code"`
	Access         string    `json:"access"`
	LastSeenAt     time.Time `json:"last_seen_at"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func toDTO(u *domain.User) userDTO {
	return userDTO{
		ID:             u.ID,
		ChannelID:      u.ChannelID,
		ChannelName:    u.ChannelName,
		ExternalUserID: u.ExternalUserID,
		Username:       u.Username,
		FirstName:      u.FirstName,
		LastName:       u.LastName,
		LanguageCode:   u.LanguageCode,
		Access:         string(u.Access),
		LastSeenAt:     u.LastSeenAt,
		CreatedAt:      u.CreatedAt,
		UpdatedAt:      u.UpdatedAt,
	}
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	q, err := parseListQuery(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	list, err := h.Repo.List(r.Context(), q)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := h.attachChannelNames(r.Context(), list); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]userDTO, 0, len(list))
	for _, u := range list {
		if u == nil {
			continue
		}
		out = append(out, toDTO(u))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	u, err := h.Repo.GetByID(r.Context(), id)
	if errors.Is(err, domain.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "user not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := h.attachChannelNames(r.Context(), []*domain.User{u}); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toDTO(u))
}

func (h *Handler) attachChannelNames(ctx context.Context, users []*domain.User) error {
	if h.Channels == nil || len(users) == 0 {
		return nil
	}
	channelIDs := make([]string, 0, len(users))
	seen := make(map[string]struct{}, len(users))
	for _, user := range users {
		if user == nil || user.ChannelID == "" {
			continue
		}
		if _, exists := seen[user.ChannelID]; !exists {
			channelIDs = append(channelIDs, user.ChannelID)
			seen[user.ChannelID] = struct{}{}
		}
	}
	if len(channelIDs) == 0 {
		return nil
	}
	names, err := h.Channels.ChannelNames(ctx, channelIDs)
	if err != nil {
		return err
	}
	for _, user := range users {
		if user != nil {
			user.ChannelName = names[user.ChannelID]
		}
	}
	return nil
}

func (h *Handler) updateAccess(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Access string `json:"access"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	access := domain.NormalizeUserAccess(body.Access)
	if body.Access != string(access) {
		writeErr(w, http.StatusBadRequest, "invalid access")
		return
	}
	u, err := h.Repo.SetAccess(r.Context(), id, access)
	if errors.Is(err, domain.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "user not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toDTO(u))
}

func parseListQuery(r *http.Request) (domain.ListQuery, error) {
	q := domain.ListQuery{Q: r.URL.Query().Get("q")}
	if v := r.URL.Query().Get("channel_id"); v != "" {
		q.ChannelID = &v
	}
	if v := r.URL.Query().Get("external_user_id"); v != "" {
		q.ExternalUserID = &v
	}
	if v := r.URL.Query().Get("created_from"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return q, errors.New("invalid created_from")
		}
		q.CreatedFrom = &t
	}
	if v := r.URL.Query().Get("created_to"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return q, errors.New("invalid created_to")
		}
		q.CreatedTo = &t
	}
	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			return q, errors.New("invalid limit")
		}
		if n > 200 {
			n = 200
		}
		q.Limit = n
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			return q, errors.New("invalid offset")
		}
		q.Offset = n
	}
	return q, nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
