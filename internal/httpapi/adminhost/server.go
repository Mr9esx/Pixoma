package adminhost

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	casesapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/cases"
	channelapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/channels"
	"github.com/mr9esx/comfyui_tgbot/internal/httpapi/edges"
	menucardsapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/menucards"
	routingapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/routing"
	sessionsapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/sessions"
	statsapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/stats"
	tasksapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/tasks"
	topicsapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/topics"
	usersapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/users"
)

// Options configures the admin-api HTTP handler.
type Options struct {
	CORSOrigins []string
	// Instances, when non-nil, is mounted at /api/v1/edges.
	Instances *edges.Handler
	Cases     *casesapi.Handler
	Users     *usersapi.Handler
	Sessions  *sessionsapi.Handler
	Tasks     *tasksapi.Handler
	Stats     *statsapi.Handler
	Channels  *channelapi.Handler
	MenuCards *menucardsapi.Handler
	Topics    *topicsapi.Handler
	Routing   *routingapi.Handler
	// NotFound handles unmatched paths (SPA embed).
	NotFound http.Handler
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

	r.Route("/api/v1/edges", func(r chi.Router) {
		if opts.Instances != nil {
			opts.Instances.Mount(r)
		}
	})
	r.Route("/api/v1/cases", func(r chi.Router) {
		if opts.Cases != nil {
			opts.Cases.Mount(r)
		}
		if opts.MenuCards != nil {
			r.Get("/{id}/menu-placements", opts.MenuCards.ListWorkflowPlacements)
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
	r.Route("/api/v1/stats", func(r chi.Router) {
		if opts.Stats != nil {
			opts.Stats.Mount(r)
		}
	})
	r.Route("/api/v1/channels", func(r chi.Router) {
		if opts.Channels != nil {
			r.Get("/", opts.Channels.List)
			r.Post("/", opts.Channels.Create)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", opts.Channels.Get)
				r.Put("/", opts.Channels.Update)
				r.Post("/disable", opts.Channels.Disable)
				r.Post("/enable", opts.Channels.Enable)
				r.Delete("/", opts.Channels.Delete)
				if opts.MenuCards != nil {
					opts.MenuCards.Mount(r)
				}
			})
		}
	})
	r.Route("/api/v1/topics", func(r chi.Router) {
		if opts.Topics != nil {
			opts.Topics.Mount(r)
		}
	})
	r.Route("/api/v1/routing", func(r chi.Router) {
		if opts.Routing != nil {
			opts.Routing.Mount(r)
		}
	})

	if opts.NotFound != nil {
		r.NotFound(opts.NotFound.ServeHTTP)
	}

	return r
}
