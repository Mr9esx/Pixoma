package demo

import (
	"context"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"gorm.io/gorm"

	"github.com/Mr9esx/Pixoma/apps/pixoma/internal/webembed"
	consolepersist "github.com/Mr9esx/Pixoma/internal/adminusers/infrastructure/persistence"
	casepersist "github.com/Mr9esx/Pixoma/internal/cases/infrastructure/persistence"
	channelapp "github.com/Mr9esx/Pixoma/internal/channels/application"
	channelpersist "github.com/Mr9esx/Pixoma/internal/channels/infrastructure/persistence"
	textpersist "github.com/Mr9esx/Pixoma/internal/channels/infrastructure/persistence"
	instpersist "github.com/Mr9esx/Pixoma/internal/edge/infrastructure/persistence"
	"github.com/Mr9esx/Pixoma/internal/edge/infrastructure/presence"
	adminhost "github.com/Mr9esx/Pixoma/internal/httpapi/adminhost"
	adminusersapi "github.com/Mr9esx/Pixoma/internal/httpapi/adminusers"
	casesapi "github.com/Mr9esx/Pixoma/internal/httpapi/cases"
	channelsapi "github.com/Mr9esx/Pixoma/internal/httpapi/channels"
	edgesapi "github.com/Mr9esx/Pixoma/internal/httpapi/edges"
	linkhealthapi "github.com/Mr9esx/Pixoma/internal/httpapi/linkhealth"
	routingapi "github.com/Mr9esx/Pixoma/internal/httpapi/routing"
	sessionsapi "github.com/Mr9esx/Pixoma/internal/httpapi/sessions"
	statsapi "github.com/Mr9esx/Pixoma/internal/httpapi/stats"
	tasksapi "github.com/Mr9esx/Pixoma/internal/httpapi/tasks"
	topicsapi "github.com/Mr9esx/Pixoma/internal/httpapi/topics"
	usersapi "github.com/Mr9esx/Pixoma/internal/httpapi/users"
	mencardpersist "github.com/Mr9esx/Pixoma/internal/menus/infrastructure/persistence"
	packlink "github.com/Mr9esx/Pixoma/internal/packaging/linkhealth"
	sesspersist "github.com/Mr9esx/Pixoma/internal/sessions/infrastructure/persistence"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
	taskstatspersist "github.com/Mr9esx/Pixoma/internal/stats/infrastructure/persistence"
	"github.com/Mr9esx/Pixoma/internal/tasks/domain/condition"
	taskpersist "github.com/Mr9esx/Pixoma/internal/tasks/infrastructure/persistence"
	topicpersist "github.com/Mr9esx/Pixoma/internal/topics/infrastructure/persistence"
	userpersist "github.com/Mr9esx/Pixoma/internal/users/infrastructure/persistence"
)

type Server struct {
	server *http.Server
	db     *gorm.DB
	clean  func() error
}

func New(ctx context.Context, addr, publicURL string) (*Server, error) {
	gdb, cleanup, err := OpenDatabase()
	if err != nil {
		return nil, err
	}
	if err := Seed(ctx, gdb); err != nil {
		_ = cleanup()
		return nil, err
	}

	auth := NewAuthStore()
	adminHandler, err := newAdminHandler(gdb)
	if err != nil {
		_ = cleanup()
		return nil, err
	}
	router := chi.NewRouter()
	router.Use(middleware.RequestID, middleware.RealIP, middleware.Recoverer)
	router.Route("/api/v1/setup", func(setup chi.Router) {
		Mount(setup, auth)
	})
	router.Post("/api/v1/auth/register", writeReadonly)
	router.Get("/api/v1/auth/registration", writeReadonly)
	router.Group(func(admin chi.Router) {
		admin.Use(publicFrontend(auth))
		admin.Use(Readonly)
		admin.Mount("/", adminHandler)
	})

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		_ = cleanup()
		return nil, err
	}
	server := &http.Server{Handler: router, ReadHeaderTimeout: 5 * time.Second}
	errorCh := make(chan error, 1)
	go func() {
		if serveErr := server.Serve(listener); serveErr != nil && serveErr != http.ErrServerClosed {
			errorCh <- serveErr
		}
		close(errorCh)
	}()
	_ = publicURL
	return &Server{server: server, db: gdb, clean: cleanup}, nil
}

func (s *Server) Handler() http.Handler {
	return s.server.Handler
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s == nil || s.server == nil {
		return nil
	}
	err := s.server.Shutdown(ctx)
	if s.clean != nil {
		_ = s.clean()
		s.clean = nil
	}
	return err
}

func newAdminHandler(gdb *gorm.DB) (http.Handler, error) {
	instRepo := instpersist.NewEdgeRepository(gdb)
	caseRepo := casepersist.NewGormRepository(gdb)
	userRepo := userpersist.NewUserRepository(gdb)
	consoleRepo := consolepersist.NewConsoleUserRepository(gdb)
	sessionRepo := sesspersist.NewSessionRepository(gdb)
	taskRepo := taskpersist.NewTaskRepository(gdb)
	channelStore := channelpersist.NewGormRepository(gdb)
	statsRepo := taskstatspersist.NewGormStatsRepository(gdb, 365*24*time.Hour, time.UTC)
	menuRepo := mencardpersist.NewGormCardRepository(gdb)
	conditionReg := condition.NewRegistry()
	textStore, err := textpersist.NewStore(gdb)
	if err != nil {
		return nil, err
	}

	topicRepo := topicpersist.NewTopicRepository(gdb)
	pres := presence.NewStore()
	pres.Report(sharedkernel.EdgeID("edge-demo-1"), true)
	pres.Report(sharedkernel.EdgeID("edge-demo-2"), true)
	adapter := func(_ context.Context, _ string) (state string, lastErr string, found bool) {
		return "running", "", true
	}
	chSvc := &channelapp.Service{Store: channelStore, Key: demoEncryptionKey(), AdapterStatus: adapter}
	probe := &channelapp.ReachabilityProbe{Svc: chSvc}

	return adminhost.NewHandler(adminhost.Options{
		Instances:   &edgesapi.Handler{Repo: instRepo, Tasks: taskRepo, Metrics: instpersist.NewMetricsRepository(gdb, 24*time.Hour), Presence: pres},
		Cases:       &casesapi.Handler{Repo: caseRepo},
		AdminUsers:  &adminusersapi.Handler{Repo: consoleRepo},
		Users:       &usersapi.Handler{Repo: userRepo, Channels: channelStore},
		Sessions:    &sessionsapi.Handler{Repo: sessionRepo, Channels: channelStore, Context: sesspersist.NewSessionAdminProjection(gdb)},
		Tasks:       &tasksapi.Handler{Tasks: taskRepo, Context: taskpersist.NewTaskAdminProjection(gdb)},
		Stats:       &statsapi.Handler{Repo: statsRepo, Loc: time.UTC, Metrics: instpersist.NewMetricsRepository(gdb, 24*time.Hour)},
		Channels:    &channelsapi.Handler{Svc: chSvc, Probe: probe},
		MenuHandler: channelsapi.NewMenuHandler(menuRepo),
		Topics:      &topicsapi.Handler{Repo: topicRepo, Tasks: taskRepo},
		Routing:     &routingapi.Handler{Registry: conditionReg},
		TextHandler: &channelsapi.TextHandler{
			Store: textStore,
		},
		LinkHealth: &linkhealthapi.Handler{
			Snapshot: func(r *http.Request) (packlink.Snapshot, error) {
				return linkhealthapi.Collect(r.Context(), linkhealthapi.Source{
					Channels: channelStore,
					Adapter:  adapter,
					Menus:    menuRepo,
					Cases:    caseRepo,
					Topics:   topicRepo,
					Edges:    instRepo,
					Presence: pres,
				})
			},
		},
		NotFound: webembed.Handler(),
	}), nil
}

func publicFrontend(auth *AuthStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasPrefix(r.URL.Path, "/api/") {
				next.ServeHTTP(w, r)
				return
			}
			auth.Handler(next).ServeHTTP(w, r)
		})
	}
}
