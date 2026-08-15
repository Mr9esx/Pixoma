package adminhost

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	casesapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/cases"
	"github.com/mr9esx/comfyui_tgbot/internal/httpapi/comfyinstances"
	sessionsapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/sessions"
	tasksapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/tasks"
	tgmenuapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/tgmenu"
	usersapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/users"
)

// Options configures the admin-api HTTP handler.
type Options struct {
	CORSOrigins []string
	// Instances, when non-nil, is mounted at /api/v1/comfy-instances.
	Instances *comfyinstances.Handler
	Cases     *casesapi.Handler
	Users     *usersapi.Handler
	Sessions  *sessionsapi.Handler
	Tasks     *tasksapi.Handler
	TGMenu    *tgmenuapi.Handler
}

// NewHandler returns the admin-api chi router (health, CORS, resource APIs).
func NewHandler(opts Options) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.RealIP, middleware.Recoverer)
	r.Use(corsMiddleware(opts.CORSOrigins))

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	r.Route("/api/v1/comfy-instances", func(r chi.Router) {
		if opts.Instances != nil {
			opts.Instances.Mount(r)
		}
	})
	r.Route("/api/v1/cases", func(r chi.Router) {
		if opts.Cases != nil {
			opts.Cases.Mount(r)
		}
		if opts.TGMenu != nil {
			r.Get("/{id}/menu-placements", opts.TGMenu.ListPlacements)
		}
	})
	r.Route("/api/v1/users", func(r chi.Router) {
		if opts.Users != nil {
			opts.Users.Mount(r)
		}
	})
	r.Route("/api/v1/sessions", func(r chi.Router) {
		if opts.Sessions != nil {
			opts.Sessions.Mount(r)
		}
	})
	r.Route("/api/v1/tasks", func(r chi.Router) {
		if opts.Tasks != nil {
			opts.Tasks.Mount(r)
		}
	})
	r.Route("/api/v1/tg-menu", func(r chi.Router) {
		if opts.TGMenu != nil {
			opts.TGMenu.Mount(r)
		}
	})

	return r
}
