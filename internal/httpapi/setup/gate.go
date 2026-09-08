package setup

import (
	"context"
	"net/http"
	"strings"

	consoledomain "github.com/Mr9esx/Pixoma/internal/adminusers/domain"
	"github.com/Mr9esx/Pixoma/internal/platform/bootstrap"
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
	Boot         *bootstrap.Store
	Sessions     *Sessions
	ConsoleUsers consoledomain.Repository
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
		if path == "/api/v1/auth/register" || path == "/api/v1/auth/registration" {
			next.ServeHTTP(w, r)
			return
		}
		if strings.HasPrefix(path, "/api/v1/setup/") && !g.Boot.Initialized() {
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
		// Media upload/preview stays reachable during first-run setup so the
		// admin can attach an avatar before the platform is initialized. Once
		// initialized, requests fall through to the account auth below.
		if strings.HasPrefix(path, "/api/v1/media/") && !g.Boot.Initialized() {
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
		if !g.Boot.Initialized() {
			code := "not_initialized"
			msg := "platform not initialized"
			writeJSON(w, http.StatusForbidden, map[string]string{
				"error": msg,
				"code":  code,
			})
			return
		}
		if g.Boot.RestartRequired() && !strings.HasPrefix(path, "/api/v1/setup/") {
			writeJSON(w, http.StatusForbidden, map[string]string{
				"error": "settings saved; restart pixoma for them to take effect",
				"code":  "restart_required",
			})
			return
		}
		if g.Sessions == nil {
			writeErr(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		acct, ok := resolveAccount(r.Context(), g.Sessions, g.ConsoleUsers, TokenFromRequest(r))
		if !ok {
			writeErr(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		if isSetupAdministration(path) && acct.Role != consoledomain.RoleAdmin {
			writeErr(w, http.StatusForbidden, "forbidden")
			return
		}
		if !isSetupSelfService(path) && isWriteMethod(r.Method) &&
			acct.Role != consoledomain.RoleAdmin && acct.Role != consoledomain.RoleOperator {
			writeErr(w, http.StatusForbidden, "forbidden")
			return
		}
		next.ServeHTTP(w, r.WithContext(WithAccount(r.Context(), acct)))
	})
}

func resolveAccount(
	ctx context.Context,
	sessions *Sessions,
	users consoledomain.Repository,
	token string,
) (AccountSession, bool) {
	acct, ok := sessions.LookupAccount(token)
	if !ok {
		return AccountSession{}, false
	}
	if users == nil {
		if acct.AccountID == "" {
			acct.Role = consoledomain.RoleAdmin
		}
		return acct, true
	}

	if acct.AccountID != "" {
		user, err := users.GetByID(ctx, acct.AccountID)
		if err != nil || user == nil || !user.Enabled {
			return AccountSession{}, false
		}
		return AccountSession{
			AccountID: user.ID,
			Username:  user.Username,
			Role:      user.Role,
		}, true
	}

	// Bootstrap-era sessions can survive a restart before the console account
	// store exists. Rebind them now so status and business APIs agree.
	user, err := users.GetByUsername(ctx, acct.Username)
	if err != nil || user == nil || !user.Enabled {
		return AccountSession{}, false
	}
	if !sessions.BindAccount(token, user.ID, user.Role) {
		return AccountSession{}, false
	}
	return AccountSession{
		AccountID: user.ID,
		Username:  user.Username,
		Role:      user.Role,
	}, true
}

func isWriteMethod(method string) bool {
	return method != http.MethodGet && method != http.MethodHead && method != http.MethodOptions
}

func isSetupSelfService(path string) bool {
	switch path {
	case "/api/v1/setup/logout", "/api/v1/setup/me", "/api/v1/setup/password", "/api/v1/setup/profile":
		return true
	default:
		return false
	}
}

func isSetupAdministration(path string) bool {
	return strings.HasPrefix(path, "/api/v1/setup/") && !isSetupSelfService(path)
}
