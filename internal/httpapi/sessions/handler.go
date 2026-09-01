package sessions

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/mr9esx/comfyui_tgbot/internal/conversation/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

// Handler serves read-only session admin HTTP under /api/v1/sessions.
type Handler struct {
	Repo     domain.Repository
	Channels ChannelNamesReader
	Context  domain.SessionAdminReader
}

type ChannelNamesReader interface {
	ChannelNames(ctx context.Context, ids []string) (map[string]string, error)
}

// Mount registers GET / and GET /{id} only.
func (h *Handler) Mount(r chi.Router) {
	r.Get("/", h.list)
	r.Get("/{id}", h.get)
}

type draftDTO struct {
	Key     string                `json:"key"`
	Text    *string               `json:"text,omitempty"`
	Number  *float64              `json:"number,omitempty"`
	Bool    *bool                 `json:"bool,omitempty"`
	Blob    *sharedkernel.BlobRef `json:"blob,omitempty"`
	Skipped bool                  `json:"skipped,omitempty"`
}

type sessionDTO struct {
	ID                string              `json:"id"`
	UserID            string              `json:"user_id"`
	User              *sessionUserDTO     `json:"user,omitempty"`
	ChannelID         string              `json:"channel_id"`
	ChannelName       string              `json:"channel_name,omitempty"`
	ChatID            string              `json:"chat_id"`
	CaseID            uint64              `json:"case_id"`
	Status            string              `json:"status"`
	CurrentInputIndex int                 `json:"current_input_index"`
	InputKeys         []string            `json:"input_keys"`
	Draft             map[string]draftDTO `json:"draft"`
	CreatedAt         time.Time           `json:"created_at"`
	UpdatedAt         time.Time           `json:"updated_at"`
}

type sessionUserDTO struct {
	ID             string `json:"id"`
	ChannelID      string `json:"channel_id"`
	ExternalUserID string `json:"external_user_id"`
	Username       string `json:"username"`
	FirstName      string `json:"first_name"`
	LastName       string `json:"last_name"`
}

func (h *Handler) toDTO(ctx context.Context, s *domain.Session) (sessionDTO, error) {
	draft := map[string]draftDTO{}
	for k, v := range s.Draft {
		draft[k] = draftDTO{
			Key:     v.Key,
			Text:    v.Text,
			Number:  v.Number,
			Bool:    v.Bool,
			Blob:    v.Blob,
			Skipped: v.Skipped,
		}
	}
	keys := s.InputKeys
	if keys == nil {
		keys = []string{}
	}
	channelName := ""
	if h.Channels != nil && s.ChannelID != "" {
		names, err := h.Channels.ChannelNames(ctx, []string{s.ChannelID})
		if err != nil {
			return sessionDTO{}, err
		}
		channelName = names[s.ChannelID]
	}
	return sessionDTO{
		ID:                string(s.ID),
		UserID:            s.UserID,
		ChannelID:         s.ChannelID,
		ChannelName:       channelName,
		ChatID:            string(s.ChatID),
		CaseID:            uint64(s.CaseID),
		Status:            string(s.Status),
		CurrentInputIndex: s.CurrentInputIndex,
		InputKeys:         keys,
		Draft:             draft,
		CreatedAt:         s.CreatedAt,
		UpdatedAt:         s.UpdatedAt,
	}, nil
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	q, err := parseListQuery(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if h.Context != nil {
		contexts, err := h.Context.List(r.Context(), q)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		out := make([]sessionDTO, 0, len(contexts))
		for _, context := range contexts {
			if context == nil || context.Session == nil {
				continue
			}
			dto, err := h.toContextDTO(r.Context(), context)
			if err != nil {
				writeErr(w, http.StatusInternalServerError, err.Error())
				return
			}
			out = append(out, dto)
		}
		writeJSON(w, http.StatusOK, out)
		return
	}
	list, err := h.Repo.List(r.Context(), q)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]sessionDTO, 0, len(list))
	for _, s := range list {
		if s == nil {
			continue
		}
		dto, err := h.toDTO(r.Context(), s)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		out = append(out, dto)
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id := sharedkernel.SessionID(chi.URLParam(r, "id"))
	if h.Context != nil {
		context, err := h.Context.Get(r.Context(), id)
		if errors.Is(err, domain.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "session not found")
			return
		}
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if context == nil || context.Session == nil {
			writeErr(w, http.StatusNotFound, "session not found")
			return
		}
		dto, err := h.toContextDTO(r.Context(), context)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, dto)
		return
	}
	s, err := h.Repo.GetByID(r.Context(), id)
	if errors.Is(err, domain.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "session not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	dto, err := h.toDTO(r.Context(), s)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, dto)
}

func (h *Handler) toContextDTO(ctx context.Context, context *domain.SessionAdminContext) (sessionDTO, error) {
	dto, err := h.toDTO(ctx, context.Session)
	if err != nil {
		return sessionDTO{}, err
	}
	dto.ChannelName = context.ChannelName
	if context.User != nil {
		dto.User = &sessionUserDTO{
			ID:             context.User.ID,
			ChannelID:      context.User.ChannelID,
			ExternalUserID: context.User.ExternalUserID,
			Username:       context.User.Username,
			FirstName:      context.User.FirstName,
			LastName:       context.User.LastName,
		}
	}
	return dto, nil
}

func parseListQuery(r *http.Request) (domain.ListQuery, error) {
	q := domain.ListQuery{
		Q:      r.URL.Query().Get("q"),
		UserID: r.URL.Query().Get("user_id"),
	}
	if v := r.URL.Query().Get("status"); v != "" {
		q.Status = domain.Status(v)
	}
	if v := r.URL.Query().Get("chat_id"); v != "" {
		chat := sharedkernel.ChatID(v)
		q.ChatID = &chat
	}
	if v := r.URL.Query().Get("case_id"); v != "" {
		n, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			return q, errors.New("invalid case_id")
		}
		q.CaseID = sharedkernel.CaseID(n)
	}
	if v := r.URL.Query().Get("channel_id"); v != "" {
		q.ChannelID = v
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
