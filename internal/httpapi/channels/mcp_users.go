package channels

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/Mr9esx/Pixoma/internal/apierr"
	"github.com/Mr9esx/Pixoma/internal/channels/domain"
	edgedomain "github.com/Mr9esx/Pixoma/internal/edge/domain"
	pixmcp "github.com/Mr9esx/Pixoma/internal/mcp"
	"github.com/Mr9esx/Pixoma/internal/response"
	identitydomain "github.com/Mr9esx/Pixoma/internal/users/domain"
)

func (h *Handler) CreateMCPUser(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.Svc == nil || h.Users == nil || h.Tokens == nil {
		response.Fail(w, apierr.ErrChannelCreateMCPUserNotConfigured, "mcp users not configured")
		return
	}
	id := chi.URLParam(r, "id")
	ch, err := h.Svc.Get(r.Context(), id)
	if errors.Is(err, domain.ErrNotFound) {
		response.Fail(w, apierr.ErrChannelGetNotFound, "channel not found")
		return
	}
	if err != nil {
		response.FailErr(w, apierr.ErrChannelCreateMCPUserFailed, err)
		return
	}
	if ch.Platform != string(domain.PlatformMCP) {
		response.Fail(w, apierr.ErrChannelCreateMCPUserNotMCP, "channel is not mcp")
		return
	}
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Fail(w, apierr.ErrChannelCreateInvalidJSON, "invalid json")
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		response.Fail(w, apierr.ErrChannelCreateMCPUserNameRequired, "name required")
		return
	}
	now := time.Now().UTC()
	u, err := h.Users.UpsertByChannelExternal(r.Context(), identitydomain.UpsertFrom{
		ChannelID:      ch.ID,
		ExternalUserID: "mcp-" + uuid.NewString(),
		Username:       name,
		LastSeenAt:     now,
	})
	if err != nil {
		response.FailErr(w, apierr.ErrChannelCreateMCPUserFailed, err)
		return
	}
	u, err = h.Users.SetAccess(r.Context(), u.ID, identitydomain.UserAccessAlwaysAllowed)
	if err != nil {
		response.FailErr(w, apierr.ErrChannelCreateMCPUserFailed, err)
		return
	}
	plain, err := mintAndStoreMCPToken(r.Context(), h.Tokens, h.Key, u.ID)
	if err != nil {
		_ = h.Users.Delete(r.Context(), u.ID)
		response.FailErr(w, apierr.ErrChannelCreateMCPUserFailed, err)
		return
	}
	response.OKStatus(w, http.StatusCreated, map[string]any{
		"user":  mcpUserDTO(u),
		"token": plain,
	})
}

func (h *Handler) ListMCPUsers(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.Users == nil {
		response.Fail(w, apierr.ErrChannelCreateMCPUserNotConfigured, "mcp users not configured")
		return
	}
	id := chi.URLParam(r, "id")
	ch, err := h.Svc.Get(r.Context(), id)
	if errors.Is(err, domain.ErrNotFound) {
		response.Fail(w, apierr.ErrChannelGetNotFound, "channel not found")
		return
	}
	if err != nil {
		response.FailErr(w, apierr.ErrChannelListMCPUsersFailed, err)
		return
	}
	if ch.Platform != string(domain.PlatformMCP) {
		response.Fail(w, apierr.ErrChannelCreateMCPUserNotMCP, "channel is not mcp")
		return
	}
	list, err := h.Users.List(r.Context(), identitydomain.ListQuery{ChannelID: &id, Limit: 200})
	if err != nil {
		response.FailErr(w, apierr.ErrChannelListMCPUsersFailed, err)
		return
	}
	out := make([]map[string]any, 0, len(list))
	for _, u := range list {
		if u == nil {
			continue
		}
		out = append(out, mcpUserDTO(u))
	}
	response.OKStatus(w, http.StatusOK, out)
}

func (h *Handler) DeleteMCPUser(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.Svc == nil || h.Users == nil || h.Tokens == nil {
		response.Fail(w, apierr.ErrChannelCreateMCPUserNotConfigured, "mcp users not configured")
		return
	}
	chID := chi.URLParam(r, "id")
	userID := chi.URLParam(r, "userId")
	ch, err := h.Svc.Get(r.Context(), chID)
	if errors.Is(err, domain.ErrNotFound) {
		response.Fail(w, apierr.ErrChannelGetNotFound, "channel not found")
		return
	}
	if err != nil {
		response.FailErr(w, apierr.ErrChannelDeleteMCPUserFailed, err)
		return
	}
	if ch.Platform != string(domain.PlatformMCP) {
		response.Fail(w, apierr.ErrChannelCreateMCPUserNotMCP, "channel is not mcp")
		return
	}
	u, err := h.Users.GetByID(r.Context(), userID)
	if errors.Is(err, identitydomain.ErrNotFound) {
		response.Fail(w, apierr.ErrChannelGetNotFound, "user not found")
		return
	}
	if err != nil {
		response.FailErr(w, apierr.ErrChannelDeleteMCPUserFailed, err)
		return
	}
	if u.ChannelID != ch.ID {
		response.Fail(w, apierr.ErrChannelGetNotFound, "user not found")
		return
	}
	if err := h.Tokens.DeleteByUserID(r.Context(), u.ID); err != nil {
		response.FailErr(w, apierr.ErrChannelDeleteMCPUserFailed, err)
		return
	}
	if err := h.Users.Delete(r.Context(), u.ID); err != nil {
		if errors.Is(err, identitydomain.ErrNotFound) {
			response.Fail(w, apierr.ErrChannelGetNotFound, "user not found")
			return
		}
		response.FailErr(w, apierr.ErrChannelDeleteMCPUserFailed, err)
		return
	}
	response.OKStatus(w, http.StatusOK, map[string]bool{"deleted": true})
}

func mintAndStoreMCPToken(ctx context.Context, store pixmcp.TokenStore, key []byte, userID string) (string, error) {
	plain, err := edgedomain.MintToken()
	if err != nil {
		return "", err
	}
	cipher, err := pixmcp.EncryptToken(key, plain)
	if err != nil {
		return "", err
	}
	if err := store.Put(ctx, pixmcp.TokenRecord{
		UserID: userID, TokenHash: pixmcp.HashToken(plain), TokenCipher: cipher,
	}); err != nil {
		return "", err
	}
	return plain, nil
}

func mcpUserDTO(u *identitydomain.User) map[string]any {
	return map[string]any{
		"id":               u.ID,
		"channel_id":       u.ChannelID,
		"external_user_id": u.ExternalUserID,
		"username":         u.Username,
		"access":           string(u.Access),
		"created_at":       u.CreatedAt,
		"updated_at":       u.UpdatedAt,
	}
}
