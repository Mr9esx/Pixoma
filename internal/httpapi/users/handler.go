package users

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/Mr9esx/Pixoma/internal/apierr"
	channeldomain "github.com/Mr9esx/Pixoma/internal/channels/domain"
	edgedomain "github.com/Mr9esx/Pixoma/internal/edge/domain"
	"github.com/Mr9esx/Pixoma/internal/httpapi/setup"
	pixmcp "github.com/Mr9esx/Pixoma/internal/mcp"
	"github.com/Mr9esx/Pixoma/internal/response"
	domain "github.com/Mr9esx/Pixoma/internal/users/domain"
)

// Handler serves user admin HTTP under /api/v1/users.
type Handler struct {
	Repo       domain.Repository
	Channels   ChannelNamesReader
	Tokens     pixmcp.TokenStore
	Key        []byte
	ChannelGet ChannelGetter
}

type ChannelNamesReader interface {
	ChannelNames(ctx context.Context, ids []string) (map[string]string, error)
}

type ChannelGetter interface {
	Get(ctx context.Context, id string) (channeldomain.Channel, error)
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
		response.FailErr(w, apierr.ErrUserListInvalidQuery, err)
		return
	}
	list, err := h.Repo.List(r.Context(), q)
	if err != nil {
		response.FailErr(w, apierr.ErrUserListFailed, err)
		return
	}
	if err := h.attachChannelNames(r.Context(), list); err != nil {
		response.FailErr(w, apierr.ErrUserListFailed, err)
		return
	}
	out := make([]userDTO, 0, len(list))
	for _, u := range list {
		if u == nil {
			continue
		}
		out = append(out, toDTO(u))
	}
	response.OKStatus(w, http.StatusOK, out)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	u, err := h.Repo.GetByID(r.Context(), id)
	if errors.Is(err, domain.ErrNotFound) {
		response.Fail(w, apierr.ErrUserGetNotFound, "user not found")
		return
	}
	if err != nil {
		response.FailErr(w, apierr.ErrUserListFailed, err)
		return
	}
	if err := h.attachChannelNames(r.Context(), []*domain.User{u}); err != nil {
		response.FailErr(w, apierr.ErrUserListFailed, err)
		return
	}
	response.OKStatus(w, http.StatusOK, toDTO(u))
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
		response.Fail(w, apierr.ErrUserUpdateAccessInvalidJSON, "invalid json")
		return
	}
	access := domain.NormalizeUserAccess(body.Access)
	if body.Access != string(access) {
		response.Fail(w, apierr.ErrUserUpdateAccessInvalidAccess, "invalid access")
		return
	}
	u, err := h.Repo.SetAccess(r.Context(), id, access)
	if errors.Is(err, domain.ErrNotFound) {
		response.Fail(w, apierr.ErrUserGetNotFound, "user not found")
		return
	}
	if err != nil {
		response.FailErr(w, apierr.ErrUserUpdateAccessFailed, err)
		return
	}
	response.OKStatus(w, http.StatusOK, toDTO(u))
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

func (h *Handler) GetMCPToken(w http.ResponseWriter, r *http.Request) {
	u, rec, ok := h.loadMCPToken(w, r)
	if !ok {
		return
	}
	_ = u
	out := map[string]any{}
	if canRevealMCPToken(r) {
		plain, err := pixmcp.DecryptToken(h.Key, rec.TokenCipher)
		if err != nil {
			response.FailErr(w, apierr.ErrUserGetMCPTokenFailed, err)
			return
		}
		out["token"] = plain
	}
	response.OKStatus(w, http.StatusOK, out)
}

func (h *Handler) RotateMCPToken(w http.ResponseWriter, r *http.Request) {
	if !canRevealMCPToken(r) {
		response.Fail(w, apierr.ErrUserRotateMCPTokenForbidden, "forbidden")
		return
	}
	u, _, ok := h.loadMCPToken(w, r)
	if !ok {
		return
	}
	plain, err := edgedomain.MintToken()
	if err != nil {
		response.FailErr(w, apierr.ErrUserRotateMCPTokenFailed, err)
		return
	}
	cipher, err := pixmcp.EncryptToken(h.Key, plain)
	if err != nil {
		response.FailErr(w, apierr.ErrUserRotateMCPTokenFailed, err)
		return
	}
	if err := h.Tokens.Put(r.Context(), pixmcp.TokenRecord{
		UserID: u.ID, TokenHash: pixmcp.HashToken(plain), TokenCipher: cipher,
	}); err != nil {
		response.FailErr(w, apierr.ErrUserRotateMCPTokenFailed, err)
		return
	}
	response.OKStatus(w, http.StatusOK, map[string]any{"token": plain})
}

func (h *Handler) loadMCPToken(w http.ResponseWriter, r *http.Request) (*domain.User, pixmcp.TokenRecord, bool) {
	if h == nil || h.Repo == nil || h.Tokens == nil || h.ChannelGet == nil {
		response.Fail(w, apierr.ErrUserLoadMCPTokenNotConfigured, "mcp token not configured")
		return nil, pixmcp.TokenRecord{}, false
	}
	id := chi.URLParam(r, "id")
	u, err := h.Repo.GetByID(r.Context(), id)
	if errors.Is(err, domain.ErrNotFound) {
		response.Fail(w, apierr.ErrUserGetNotFound, "user not found")
		return nil, pixmcp.TokenRecord{}, false
	}
	if err != nil {
		response.FailErr(w, apierr.ErrUserGetMCPTokenFailed, err)
		return nil, pixmcp.TokenRecord{}, false
	}
	ch, err := h.ChannelGet.Get(r.Context(), u.ChannelID)
	if err != nil || ch.Platform != string(channeldomain.PlatformMCP) {
		response.Fail(w, apierr.ErrUserLoadMCPTokenNotFound, "mcp token not found")
		return nil, pixmcp.TokenRecord{}, false
	}
	rec, err := h.Tokens.GetByUserID(r.Context(), u.ID)
	if errors.Is(err, pixmcp.ErrTokenNotFound) {
		response.Fail(w, apierr.ErrUserLoadMCPTokenNotFound, "mcp token not found")
		return nil, pixmcp.TokenRecord{}, false
	}
	if err != nil {
		response.FailErr(w, apierr.ErrUserGetMCPTokenFailed, err)
		return nil, pixmcp.TokenRecord{}, false
	}
	return u, rec, true
}

func canRevealMCPToken(r *http.Request) bool {
	acct, ok := setup.AccountFromContext(r.Context())
	if !ok {
		return false
	}
	return acct.Role == "admin" || acct.Role == "operator"
}
