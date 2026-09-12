package mcp

import (
	"bytes"
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"net/http"
	"strings"

	channeldomain "github.com/Mr9esx/Pixoma/internal/channels/domain"
	identitydomain "github.com/Mr9esx/Pixoma/internal/users/domain"
	"github.com/modelcontextprotocol/go-sdk/auth"
)

var (
	errUnauthorized = errors.New("unauthorized")
	errUnavailable  = errors.New("mcp channel or user unavailable")
)

type ChannelGetter interface {
	Get(ctx context.Context, id string) (channeldomain.Channel, error)
}

type Resolver struct {
	Tokens   TokenStore
	Users    identitydomain.Repository
	Channels ChannelGetter
	Key      []byte
}

func NewResolver(tokens TokenStore, users identitydomain.Repository, channels ChannelGetter, key []byte) *Resolver {
	return &Resolver{Tokens: tokens, Users: users, Channels: channels, Key: key}
}

func (r *Resolver) Resolve(ctx context.Context, bearer string) (Identity, error) {
	if r == nil || r.Tokens == nil || r.Users == nil || r.Channels == nil {
		return Identity{}, errUnauthorized
	}
	bearer = strings.TrimSpace(bearer)
	if bearer == "" {
		return Identity{}, errUnauthorized
	}
	rec, err := r.Tokens.GetByHash(ctx, HashToken(bearer))
	if err != nil {
		return Identity{}, errUnauthorized
	}
	plain, err := DecryptToken(r.Key, rec.TokenCipher)
	if err != nil || subtle.ConstantTimeCompare([]byte(plain), []byte(bearer)) != 1 {
		return Identity{}, errUnauthorized
	}
	user, err := r.Users.GetByID(ctx, rec.UserID)
	if err != nil || user == nil {
		return Identity{}, errUnavailable
	}
	ch, err := r.Channels.Get(ctx, user.ChannelID)
	if err != nil || !ch.Enabled || ch.Platform != string(channeldomain.PlatformMCP) {
		return Identity{}, errUnavailable
	}
	return Identity{
		UserID:         user.ID,
		ChannelID:      user.ChannelID,
		ExternalUserID: user.ExternalUserID,
	}, nil
}

func RequireBearer(res *Resolver) func(http.Handler) http.Handler {
	verifier := func(ctx context.Context, token string, _ *http.Request) (*auth.TokenInfo, error) {
		id, err := res.Resolve(ctx, token)
		if err != nil {
			msg := GuideUnauthorized
			if errors.Is(err, errUnavailable) {
				msg = GuideUnavailable
			}
			return nil, fmt.Errorf("%s: %w", msg, auth.ErrInvalidToken)
		}
		return &auth.TokenInfo{
			UserID: id.UserID,
			Extra:  map[string]any{"identity": id},
		}, nil
	}
	mw := auth.RequireBearerToken(verifier, &auth.RequireBearerTokenOptions{
		AllowMissingExpiration: true,
	})
	return func(next http.Handler) http.Handler {
		return rewriteMCPAuthBody(mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			info := auth.TokenInfoFromContext(r.Context())
			if info == nil {
				http.Error(w, GuideUnauthorized, http.StatusUnauthorized)
				return
			}
			id, _ := info.Extra["identity"].(Identity)
			if id.UserID == "" {
				http.Error(w, GuideUnauthorized, http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r.WithContext(WithIdentity(r.Context(), id)))
		})))
	}
}

func rewriteMCPAuthBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(&authGuideWriter{ResponseWriter: w}, r)
	})
}

type authGuideWriter struct {
	http.ResponseWriter
	code int
}

func (w *authGuideWriter) WriteHeader(code int) {
	w.code = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *authGuideWriter) Write(p []byte) (int, error) {
	code := w.code
	if code == 0 {
		code = http.StatusOK
	}
	if code == http.StatusUnauthorized && !bytes.Contains(p, []byte("连接器")) {
		p = []byte(GuideUnauthorized + "\n")
	}
	if code == http.StatusForbidden && bytes.Contains(p, []byte("session user mismatch")) {
		p = []byte(GuideForbidden + "\n")
	}
	return w.ResponseWriter.Write(p)
}

func (w *authGuideWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func (w *authGuideWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
