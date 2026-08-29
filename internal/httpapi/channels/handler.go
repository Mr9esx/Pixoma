package channels

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/mr9esx/comfyui_tgbot/internal/channel/application"
	"github.com/mr9esx/comfyui_tgbot/internal/channel/domain"
)

// Handler serves channel management endpoints under /api/v1/channels.
type Handler struct {
	Svc *application.Service
	// OnCreated, when set, is invoked after a channel is created so channel-
	// scoped data (e.g. an initial copy of the platform default templates) can
	// be seeded. Failures are logged but do not fail channel creation.
	OnCreated func(ctx context.Context, id string) error
}

func (h *Handler) Mount(r chi.Router) {
	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Get("/{id}", h.Get)
	r.Put("/{id}", h.Update)
	r.Post("/{id}/disable", h.Disable)
	r.Post("/{id}/enable", h.Enable)
	r.Post("/{id}/check", h.CheckReachability)
	r.Delete("/{id}", h.Delete)
}

type channelDTO struct {
	ID          string          `json:"id"`
	Platform    string          `json:"platform"`
	Name        string          `json:"name"`
	ExtraInfo   json.RawMessage `json:"extra_info"`
	TokenMasked string          `json:"token_masked"`
	Enabled     bool            `json:"enabled"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

func (h *Handler) toDTO(ctx *http.Request, ch domain.Channel) (channelDTO, error) {
	masked, err := h.Svc.Masked(ctx.Context(), ch.ID)
	if err != nil {
		return channelDTO{}, err
	}
	var extra json.RawMessage
	if ch.ExtraInfo != "" {
		extra = json.RawMessage(ch.ExtraInfo)
	}
	return channelDTO{
		ID: ch.ID, Platform: ch.Platform, Name: ch.Name,
		ExtraInfo:   extra,
		TokenMasked: masked, Enabled: ch.Enabled,
		CreatedAt: ch.CreatedAt, UpdatedAt: ch.UpdatedAt,
	}, nil
}

type createBody struct {
	ID        string `json:"id"`
	Platform  string `json:"platform"`
	Name      string `json:"name"`
	Token     string `json:"token"`
	ExtraInfo string `json:"extra_info"`
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.Svc == nil {
		writeErr(w, http.StatusInternalServerError, "channel service not configured")
		return
	}
	var body createBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	id := body.ID
	if id == "" {
		id = uuid.NewString()
	}
	ch, err := h.Svc.Create(r.Context(), id, domain.Platform(body.Platform), body.Name, body.Token, body.ExtraInfo)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if h.OnCreated != nil {
		if serr := h.OnCreated(r.Context(), id); serr != nil {
			slog.Error("seed channel text templates", "err", serr, "channel_id", id)
		}
	}
	dto, err := h.toDTO(r, ch)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, dto)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.Svc == nil {
		writeErr(w, http.StatusInternalServerError, "channel service not configured")
		return
	}
	chs, err := h.Svc.List(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]channelDTO, 0, len(chs))
	for _, ch := range chs {
		dto, err := h.toDTO(r, ch)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		out = append(out, dto)
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ch, err := h.Svc.Get(r.Context(), id)
	if errors.Is(err, domain.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "channel not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	dto, err := h.toDTO(r, ch)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, dto)
}

type updateBody struct {
	Name  string  `json:"name"`
	Token *string `json:"token"`
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body updateBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	ch, err := h.Svc.Update(r.Context(), id, body.Name, body.Token)
	if errors.Is(err, domain.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "channel not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	dto, err := h.toDTO(r, ch)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, dto)
}

func (h *Handler) Disable(w http.ResponseWriter, r *http.Request) {
	if err := h.Svc.Disable(r.Context(), chi.URLParam(r, "id")); errors.Is(err, domain.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "channel not found")
		return
	} else if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"enabled": false})
}

func (h *Handler) Enable(w http.ResponseWriter, r *http.Request) {
	if err := h.Svc.Enable(r.Context(), chi.URLParam(r, "id")); errors.Is(err, domain.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "channel not found")
		return
	} else if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"enabled": true})
}

func (h *Handler) CheckReachability(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.Svc == nil {
		writeErr(w, http.StatusInternalServerError, "channel service not configured")
		return
	}
	id := chi.URLParam(r, "id")
	res, err := h.Svc.CheckReachability(r.Context(), id)
	if errors.Is(err, domain.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "channel not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	dto := struct {
		application.ReachabilityResult
		CheckedAt    time.Time `json:"checked_at"`
		AdapterState string    `json:"adapter_state,omitempty"`
		AdapterError string    `json:"adapter_error,omitempty"`
	}{
		ReachabilityResult: res,
		CheckedAt:          time.Now().UTC(),
	}
	if h.Svc.AdapterStatus != nil {
		if state, lastErr, found := h.Svc.AdapterStatus(r.Context(), id); found {
			dto.AdapterState = state
			dto.AdapterError = lastErr
		}
	}
	writeJSON(w, http.StatusOK, dto)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	err := h.Svc.Delete(r.Context(), chi.URLParam(r, "id"))
	if errors.Is(err, domain.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "channel not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
