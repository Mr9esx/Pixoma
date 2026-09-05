package livedemo

import (
	"context"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"gorm.io/gorm"

	"github.com/mr9esx/comfyui_tgbot/apps/pixoma/internal/webembed"
	casepersist "github.com/mr9esx/comfyui_tgbot/internal/catalog/infrastructure/persistence"
	channelapp "github.com/mr9esx/comfyui_tgbot/internal/channel/application"
	channelpersist "github.com/mr9esx/comfyui_tgbot/internal/channel/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/channel/text"
	consolepersist "github.com/mr9esx/comfyui_tgbot/internal/consoleuser/persistence"
	sesspersist "github.com/mr9esx/comfyui_tgbot/internal/conversation/infrastructure/persistence"
	adminhost "github.com/mr9esx/comfyui_tgbot/internal/httpapi/adminhost"
	adminusersapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/adminusers"
	casesapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/cases"
	channelsapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/channels"
	channeltextapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/channeltext"
	edgesapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/edges"
	menucardsapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/menucards"
	routingapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/routing"
	sessionsapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/sessions"
	statsapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/stats"
	tasksapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/tasks"
	topicsapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/topics"
	usersapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/users"
	userpersist "github.com/mr9esx/comfyui_tgbot/internal/identity/infrastructure/persistence"
	mencardpersist "github.com/mr9esx/comfyui_tgbot/internal/menucard/infrastructure/persistence"
	instpersist "github.com/mr9esx/comfyui_tgbot/internal/platform/edge/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/presence"
	taskstatspersist "github.com/mr9esx/comfyui_tgbot/internal/platform/taskstats/persistence"
	topicpersist "github.com/mr9esx/comfyui_tgbot/internal/platform/topic/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/domain/condition"
	taskpersist "github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/persistence"
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
	conditionReg.Register(&condition.CaseProvider{})
	textStore, err := text.NewStore(gdb)
	if err != nil {
		return nil, err
	}

	return adminhost.NewHandler(adminhost.Options{
		Instances:  &edgesapi.Handler{Repo: instRepo, Tasks: taskRepo, Metrics: instpersist.NewMetricsRepository(gdb, 24*time.Hour), Presence: presence.NewStore()},
		Cases:      &casesapi.Handler{Repo: caseRepo},
		AdminUsers: &adminusersapi.Handler{Repo: consoleRepo},
		Users:      &usersapi.Handler{Repo: userRepo, Channels: channelStore},
		Sessions:   &sessionsapi.Handler{Repo: sessionRepo, Channels: channelStore, Context: sesspersist.NewSessionAdminProjection(gdb)},
		Tasks:      &tasksapi.Handler{Tasks: taskRepo, Context: taskpersist.NewTaskAdminProjection(gdb)},
		Stats:      &statsapi.Handler{Repo: statsRepo, Loc: time.UTC, Metrics: instpersist.NewMetricsRepository(gdb, 24*time.Hour)},
		Channels:   &channelsapi.Handler{Svc: &channelapp.Service{Store: channelStore, Key: demoEncryptionKey()}},
		MenuCards:  menucardsapi.NewHandler(menuRepo),
		Topics:     &topicsapi.Handler{Repo: topicpersist.NewTopicRepository(gdb), Tasks: taskRepo},
		Routing:    &routingapi.Handler{Registry: conditionReg},
		ChannelText: &channeltextapi.Handler{
			Store: textStore,
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
