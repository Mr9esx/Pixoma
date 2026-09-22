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

	"github.com/Mr9esx/Pixoma/internal/apierr"
	"github.com/Mr9esx/Pixoma/internal/channels/application"
	"github.com/Mr9esx/Pixoma/internal/channels/domain"
	pixmcp "github.com/Mr9esx/Pixoma/internal/mcp"
	"github.com/Mr9esx/Pixoma/internal/response"
	identitydomain "github.com/Mr9esx/Pixoma/internal/users/domain"
)

// Handler serves channel management endpoints under /api/v1/channels.
type Handler struct {
	Svc *application.Service
	// OnCreated, when set, is invoked after a channel is created so channel-
	// scoped data (e.g. an initial copy of the platform default templates) can
	// be seeded. Failures are logged but do not fail channel creation.
	OnCreated func(ctx context.Context, id string) error
	// Probe, when set, is kicked asynchronously by POST /probe.
	Probe *application.ReachabilityProbe
	// ProbeCtx is the process context for kicked ProbeOnce calls; request
	// cancel must not abort the probe. Nil falls back to context.Background.
	ProbeCtx context.Context
	Users    identitydomain.Repository
	Tokens   pixmcp.TokenStore
	Key      []byte
}

func (h *Handler) Mount(r chi.Router) {
	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Post("/probe", h.KickProbe)
	r.Get("/{id}", h.Get)
	r.Put("/{id}", h.Update)
	r.Post("/{id}/disable", h.Disable)
	r.Post("/{id}/enable", h.Enable)
	r.Delete("/{id}", h.Delete)
	r.Get("/{id}/mcp-users", h.ListMCPUsers)
	r.Post("/{id}/mcp-users", h.CreateMCPUser)
	r.Delete("/{id}/mcp-users/{userId}", h.DeleteMCPUser)
}

type channelDTO struct {
	ID               string          `json:"id"`
	Platform         string          `json:"platform"`
	Name             string          `json:"name"`
	AppID            string          `json:"app_id,omitempty"`
	ExtraInfo        json.RawMessage `json:"extra_info"`
	TokenMasked      string          `json:"token_masked"`
	Enabled          bool            `json:"enabled"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
	AdapterState     string          `json:"adapter_state,omitempty"`
	AdapterError     string          `json:"adapter_error,omitempty"`
	LastCheckKind    string          `json:"last_check_kind,omitempty"`
	LastCheckMessage string          `json:"last_check_message,omitempty"`
	LastCheckAt      *time.Time      `json:"last_check_at,omitempty"`
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
	dto := channelDTO{
		ID: ch.ID, Platform: ch.Platform, Name: ch.Name,
		ExtraInfo:   extra,
		TokenMasked: masked, Enabled: ch.Enabled,
		CreatedAt: ch.CreatedAt, UpdatedAt: ch.UpdatedAt,
		LastCheckKind:    ch.LastCheckKind,
		LastCheckMessage: ch.LastCheckMessage,
		LastCheckAt:      ch.LastCheckAt,
	}
	if ch.Platform == string(domain.PlatformFeishu) && h.Svc != nil {
		if cred, derr := domain.DecryptCredential(h.Svc.Key, ch.CredentialCiphertext); derr == nil {
			dto.AppID = cred.AppID
		}
	}
	if h.Svc != nil && h.Svc.AdapterStatus != nil {
		if state, lastErr, found := h.Svc.AdapterStatus(ctx.Context(), ch.ID); found {
			dto.AdapterState = state
			dto.AdapterError = lastErr
		}
	}
	return dto, nil
}

type createBody struct {
	ID        string `json:"id"`
	Platform  string `json:"platform"`
	Name      string `json:"name"`
	Token     string `json:"token"`
	AppID     string `json:"app_id"`
	AppSecret string `json:"app_secret"`
	ExtraInfo string `json:"extra_info"`
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.Svc == nil {
		response.Fail(w, apierr.ErrChannelCreateNotConfigured, "channel service not configured")
		return
	}
	var body createBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Fail(w, apierr.ErrChannelCreateInvalidJSON, "invalid json")
		return
	}
	id := body.ID
	if id == "" {
		id = uuid.NewString()
	}
	platform := domain.Platform(body.Platform)
	cred := domain.Credential{BotToken: body.Token}
	switch platform {
	case domain.PlatformFeishu:
		cred = domain.Credential{AppID: body.AppID, AppSecret: body.AppSecret}
	}
	ch, err := h.Svc.CreateWithCredential(r.Context(), id, platform, body.Name, cred, body.ExtraInfo)
	if err != nil {
		response.FailErr(w, apierr.ErrChannelCreateInvalid, err)
		return
	}
	if h.OnCreated != nil {
		if serr := h.OnCreated(r.Context(), id); serr != nil {
			slog.Error("seed channel text templates", "err", serr, "channel_id", id)
		}
	}
	dto, err := h.toDTO(r, ch)
	if err != nil {
		response.FailErr(w, apierr.ErrChannelCreateFailed, err)
		return
	}
	response.OKStatus(w, http.StatusOK, dto)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.Svc == nil {
		response.Fail(w, apierr.ErrChannelCreateNotConfigured, "channel service not configured")
		return
	}
	chs, err := h.Svc.List(r.Context())
	if err != nil {
		response.FailErr(w, apierr.ErrChannelListFailed, err)
		return
	}
	out := make([]channelDTO, 0, len(chs))
	for _, ch := range chs {
		dto, err := h.toDTO(r, ch)
		if err != nil {
			response.FailErr(w, apierr.ErrChannelListFailed, err)
			return
		}
		out = append(out, dto)
	}
	response.OKStatus(w, http.StatusOK, out)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ch, err := h.Svc.Get(r.Context(), id)
	if errors.Is(err, domain.ErrNotFound) {
		response.Fail(w, apierr.ErrChannelGetNotFound, "channel not found")
		return
	}
	if err != nil {
		response.FailErr(w, apierr.ErrChannelListFailed, err)
		return
	}
	dto, err := h.toDTO(r, ch)
	if err != nil {
		response.FailErr(w, apierr.ErrChannelListFailed, err)
		return
	}
	response.OKStatus(w, http.StatusOK, dto)
}

type updateBody struct {
	Name      string  `json:"name"`
	Token     *string `json:"token"`
	AppSecret *string `json:"app_secret"`
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body updateBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Fail(w, apierr.ErrChannelCreateInvalidJSON, "invalid json")
		return
	}
	current, err := h.Svc.Get(r.Context(), id)
	if errors.Is(err, domain.ErrNotFound) {
		response.Fail(w, apierr.ErrChannelGetNotFound, "channel not found")
		return
	}
	if err != nil {
		response.FailErr(w, apierr.ErrChannelUpdateFailed, err)
		return
	}
	var ch domain.Channel
	switch domain.Platform(current.Platform) {
	case domain.PlatformFeishu:
		// Feishu rotates the App Secret in place; App ID is immutable.
		var updCred *domain.Credential
		if body.AppSecret != nil && *body.AppSecret != "" {
			cred := domain.Credential{}
			if cur, derr := domain.DecryptCredential(h.Svc.Key, current.CredentialCiphertext); derr == nil {
				cred = cur
			}
			cred.AppSecret = *body.AppSecret
			updCred = &cred
		}
		ch, err = h.Svc.UpdateCredential(r.Context(), id, body.Name, updCred)
	default:
		ch, err = h.Svc.Update(r.Context(), id, body.Name, body.Token)
	}
	if errors.Is(err, domain.ErrNotFound) {
		response.Fail(w, apierr.ErrChannelGetNotFound, "channel not found")
		return
	}
	if err != nil {
		response.FailErr(w, apierr.ErrChannelUpdateFailed, err)
		return
	}
	dto, err := h.toDTO(r, ch)
	if err != nil {
		response.FailErr(w, apierr.ErrChannelUpdateFailed, err)
		return
	}
	response.OKStatus(w, http.StatusOK, dto)
}

func (h *Handler) Disable(w http.ResponseWriter, r *http.Request) {
	if err := h.Svc.Disable(r.Context(), chi.URLParam(r, "id")); errors.Is(err, domain.ErrNotFound) {
		response.Fail(w, apierr.ErrChannelGetNotFound, "channel not found")
		return
	} else if err != nil {
		response.FailErr(w, apierr.ErrChannelDisableFailed, err)
		return
	}
	response.OKStatus(w, http.StatusOK, map[string]bool{"enabled": false})
}

func (h *Handler) Enable(w http.ResponseWriter, r *http.Request) {
	if err := h.Svc.Enable(r.Context(), chi.URLParam(r, "id")); errors.Is(err, domain.ErrNotFound) {
		response.Fail(w, apierr.ErrChannelGetNotFound, "channel not found")
		return
	} else if err != nil {
		response.FailErr(w, apierr.ErrChannelEnableFailed, err)
		return
	}
	response.OKStatus(w, http.StatusOK, map[string]bool{"enabled": true})
}

func (h *Handler) KickProbe(w http.ResponseWriter, r *http.Request) {
	if h == nil {
		response.Fail(w, apierr.ErrChannelCreateNotConfigured, "channel service not configured")
		return
	}
	if h.Probe != nil {
		ctx := h.ProbeCtx
		if ctx == nil {
			ctx = context.Background()
		}
		go func() {
			if err := h.Probe.ProbeOnce(ctx); err != nil && ctx.Err() == nil {
				slog.Error("channel reachability probe kick", "err", err)
			}
		}()
	}
	response.OKStatus(w, http.StatusAccepted, nil)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	err := h.Svc.Delete(r.Context(), chi.URLParam(r, "id"))
	if errors.Is(err, domain.ErrNotFound) {
		response.Fail(w, apierr.ErrChannelGetNotFound, "channel not found")
		return
	}
	if err != nil {
		response.FailErr(w, apierr.ErrChannelDeleteFailed, err)
		return
	}
	response.OKStatus(w, http.StatusOK, map[string]bool{"deleted": true})
}
