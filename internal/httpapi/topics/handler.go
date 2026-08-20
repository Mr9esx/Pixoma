package topics

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/topic"
)

// KeyPattern is the shared topic key format.
var KeyPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,62}[a-z0-9])?$`)

// Handler serves /api/v1/topics CRUD.
type Handler struct {
	Repo topic.Repository
	// CountCaseRefs counts Case routing rules referencing a topic key.
	CountCaseRefs func(ctx context.Context, key string) (int, error)
	// CountEdgeRefs counts Edge subscriptions referencing a topic key.
	CountEdgeRefs func(ctx context.Context, key string) (int, error)
}

// Mount registers chi routes (caller mounts under /api/v1/topics).
func (h *Handler) Mount(r chi.Router) {
	r.Get("/", h.list)
	r.Post("/", h.create)
	r.Get("/{key}", h.get)
	r.Put("/{key}", h.update)
	r.Delete("/{key}", h.delete)
}

type topicDTO struct {
	Key       string    `json:"key"`
	Name      string    `json:"name"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func toDTO(t topic.Topic) topicDTO {
	return topicDTO{
		Key:       t.Key,
		Name:      t.Name,
		Enabled:   t.Enabled,
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
	}
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	var enabled *bool
	if raw := strings.TrimSpace(r.URL.Query().Get("enabled")); raw != "" {
		v := raw == "true"
		enabled = &v
	}
	list, err := h.Repo.List(r.Context(), enabled)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]topicDTO, 0, len(list))
	for _, t := range list {
		out = append(out, toDTO(t))
	}
	writeJSON(w, http.StatusOK, out)
}

type createRequest struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	req.Key = strings.TrimSpace(req.Key)
	if !KeyPattern.MatchString(req.Key) {
		writeErr(w, http.StatusBadRequest, "invalid topic key (lowercase letters/digits/hyphens)")
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeErr(w, http.StatusBadRequest, "name required")
		return
	}
	now := time.Now().UTC()
	if err := h.Repo.Create(r.Context(), topic.Topic{
		Key:       req.Key,
		Name:      strings.TrimSpace(req.Name),
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		if errors.Is(err, topic.ErrTopicConflict) {
			writeErr(w, http.StatusConflict, "topic key already exists")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	got, err := h.Repo.Get(r.Context(), req.Key)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, toDTO(*got))
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	got, err := h.Repo.Get(r.Context(), key)
	if err != nil {
		if errors.Is(err, topic.ErrTopicNotFound) {
			writeErr(w, http.StatusNotFound, "topic not found")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toDTO(*got))
}

type updateRequest struct {
	Name    *string `json:"name"`
	Enabled *bool   `json:"enabled"`
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	var req updateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	got, err := h.Repo.Get(r.Context(), key)
	if err != nil {
		writeErr(w, http.StatusNotFound, "topic not found")
		return
	}
	if req.Name != nil {
		got.Name = strings.TrimSpace(*req.Name)
	}
	if req.Enabled != nil {
		if key == topic.DefaultKey && !*req.Enabled {
			writeErr(w, http.StatusConflict, "default topic cannot be disabled")
			return
		}
		got.Enabled = *req.Enabled
	}
	got.UpdatedAt = time.Now().UTC()
	if err := h.Repo.Update(r.Context(), *got); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toDTO(*got))
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	if key == topic.DefaultKey {
		writeErr(w, http.StatusConflict, "default topic cannot be deleted")
		return
	}
	var refs int
	if h.CountCaseRefs != nil {
		n, err := h.CountCaseRefs(r.Context(), key)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		refs += n
	}
	if h.CountEdgeRefs != nil {
		n, err := h.CountEdgeRefs(r.Context(), key)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		refs += n
	}
	if refs > 0 {
		writeErr(w, http.StatusConflict, "topic is referenced by cases or edges")
		return
	}
	if err := h.Repo.Delete(r.Context(), key); err != nil {
		if errors.Is(err, topic.ErrTopicNotFound) {
			writeErr(w, http.StatusNotFound, "topic not found")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
