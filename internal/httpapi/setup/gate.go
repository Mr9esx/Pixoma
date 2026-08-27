package setup

import (
	"context"
	"net/http"
	"strings"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/bootstrap"
)

type accountCtxKey struct{}

// WithAccount attaches the authenticated AccountSession to the request context.
func WithAccount(ctx context.Context, acct AccountSession) context.Context {
	return context.WithValue(ctx, accountCtxKey{}, acct)
}

// AccountFromContext returns the authenticated AccountSession, if any.
func AccountFromContext(ctx context.Context) (AccountSession, bool) {
	acct, ok := ctx.Value(accountCtxKey{}).(AccountSession)
	return acct, ok
}

// Gate rejects business admin APIs until initialized + authenticated.
type Gate struct {
	Boot     *bootstrap.Store
	Sessions *Sessions
}

func (g *Gate) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if g == nil || g.Boot == nil {
			next.ServeHTTP(w, r)
			return
		}
		path := r.URL.Path
		if path == "/healthz" || strings.HasPrefix(path, "/agent/") {
			next.ServeHTTP(w, r)
			return
		}
		if !strings.HasPrefix(path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}
		if path == "/api/v1/setup/status" || path == "/api/v1/setup/login" {
			next.ServeHTTP(w, r)
			return
		}
		// Self-registration is intentionally public; the handler enforces the
		// "open registration" setting and rejects disabled registration.
		if path == "/api/v1/auth/register" {
			next.ServeHTTP(w, r)
			return
		}
		if strings.HasPrefix(path, "/api/v1/setup/") {
			if g.Sessions == nil {
				writeErr(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			if _, ok := g.Sessions.Lookup(TokenFromRequest(r)); !ok {
				writeErr(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			next.ServeHTTP(w, r)
			return
		}
		if !g.Boot.Initialized() || g.Boot.RestartRequired() {
			code := "not_initialized"
			msg := "platform not initialized"
			if g.Boot.Initialized() {
				code = "restart_required"
				msg = "settings saved; restart pixoma for them to take effect"
			}
			writeJSON(w, http.StatusForbidden, map[string]string{
				"error": msg,
				"code":  code,
			})
			return
		}
		if g.Sessions == nil {
			writeErr(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		acct, ok := g.Sessions.LookupAccount(TokenFromRequest(r))
		if !ok {
			writeErr(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		next.ServeHTTP(w, r.WithContext(WithAccount(r.Context(), acct)))
	})
}
