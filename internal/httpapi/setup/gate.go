package setup

import (
	"net/http"
	"strings"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/bootstrap"
)

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
		if _, ok := g.Sessions.Lookup(TokenFromRequest(r)); !ok {
			writeErr(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		next.ServeHTTP(w, r)
	})
}
