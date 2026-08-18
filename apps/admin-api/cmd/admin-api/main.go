package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/mr9esx/comfyui_tgbot/apps/admin-api/internal/server"
	casepersist "github.com/mr9esx/comfyui_tgbot/internal/catalog/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/catalog/infrastructure/validation"
	channelapp "github.com/mr9esx/comfyui_tgbot/internal/channel/application"
	channelpersist "github.com/mr9esx/comfyui_tgbot/internal/channel/infrastructure/persistence"
	sesspersist "github.com/mr9esx/comfyui_tgbot/internal/conversation/infrastructure/persistence"
	casesapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/cases"
	channelsapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/channels"
	channelmenuapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/channelmenu"
	"github.com/mr9esx/comfyui_tgbot/internal/httpapi/edges"
	sessionsapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/sessions"
	tasksapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/tasks"
	usersapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/users"
	userpersist "github.com/mr9esx/comfyui_tgbot/internal/identity/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/adminconfig"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/appboot"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/bootstrap"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/edge"
	instpersist "github.com/mr9esx/comfyui_tgbot/internal/platform/edge/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/notify"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/application/orchestrator"
	taskpersist "github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/persistence"
	menuapp "github.com/mr9esx/comfyui_tgbot/internal/menu/application"
	menupersist "github.com/mr9esx/comfyui_tgbot/internal/menu/infrastructure/persistence"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := run(ctx); err != nil {
		slog.Error("admin-api failed", "err", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	cfg, err := adminconfig.Load("")
	if err != nil {
		return err
	}

	dsn := resolveDSN(cfg.DatabaseDSN)
	if err := ensureDSNParent(dsn); err != nil {
		return err
	}

	gdb, cleanup, err := appboot.Bootstrap(ctx, appboot.Options{
		DSN:          dsn,
		MigrateEdges: true,
		Models: []any{
			&casepersist.CaseRow{},
			&userpersist.UserRow{},
			&userpersist.UserExternalIdentityRow{},
			&sesspersist.SessionRow{},
			&taskpersist.TaskRow{},
			&channelpersist.ChannelRow{},
			&menupersist.ChannelMenuRow{},
			&menupersist.ChannelMenuItemRow{},
			&menupersist.ChannelMenuItemCaseRow{},
			&menupersist.ChannelMenuExtraRow{},
		},
	})
	if err != nil {
		return err
	}
	defer func() { _ = cleanup() }()

	instRepo := instpersist.NewEdgeRepository(gdb)
	metricsRepo := instpersist.NewMetricsRepository(gdb, metricsRetention())
	pool := edge.NewPool(instRepo, edge.PoolOptions{})
	if err := pool.Refresh(ctx); err != nil {
		return err
	}
	caseRepo := casepersist.NewGormRepository(gdb)
	userRepo := userpersist.NewUserRepository(gdb)
	sessionRepo := sesspersist.NewSessionRepository(gdb)
	taskRepo := taskpersist.NewTaskRepository(gdb)
	menuStore := menupersist.NewGormRepository(gdb)
	menuSvc := &menuapp.Service{
		Store:            menuStore,
		Cases:            menuapp.CatalogCaseChecker{Repo: caseRepo},
		ListImageCaseIDs: menuapp.CatalogImageCaseIDs(caseRepo),
	}
	bootPath := bootstrapDBPath(dsn)
	bootMeta, _, err := bootstrap.Open(bootPath)
	if err != nil {
		return err
	}
	encKey, err := bootMeta.EncKey()
	if err != nil {
		return err
	}
	orch := orchestrator.New(taskRepo, pool, nil, notify.Nop{})
	orch.Sessions = sessionRepo

	instAPI := &edges.Handler{
		Repo:    instRepo,
		Pool:    pool,
		Tasks:   taskRepo,
		Metrics: metricsRepo,
	}
	casesAPI := &casesapi.Handler{
		Repo:     caseRepo,
		Validate: validation.New().ValidateDocument,
	}
	usersAPI := &usersapi.Handler{Repo: userRepo}
	sessionsAPI := &sessionsapi.Handler{Repo: sessionRepo}
	tasksAPI := &tasksapi.Handler{Tasks: taskRepo, Cancel: orch}
	chSvc := &channelapp.Service{
		Store: channelpersist.NewGormRepository(gdb),
		Key:   encKey,
		HasActiveRefs: func(ctx context.Context, channelID string) (bool, error) {
			n, err := sessionRepo.CountByChannel(ctx, channelID)
			if err != nil {
				return false, err
			}
			return n > 0, nil
		},
	}
	channelsAPI := &channelsapi.Handler{Svc: chSvc}
	channelMenuAPI := &channelmenuapi.Handler{Channels: chSvc, Svc: menuSvc}

	h := server.NewHandler(server.Options{
		CORSOrigins: cfg.CORSOrigins,
		Instances:   instAPI,
		Cases:       casesAPI,
		Users:       usersAPI,
		Sessions:    sessionsAPI,
		Tasks:       tasksAPI,
		Channels:    channelsAPI,
		ChannelMenu: channelMenuAPI,
	})
	addr := cfg.HTTPAddr
	srv := &http.Server{
		Addr:              addr,
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("admin-api listening", "addr", addr, "dsn", dsn, "comfy_mock", cfg.ComfyMock)
		slog.Warn("admin-api has no auth; do not expose to the public internet")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}

func metricsRetention() time.Duration {
	if v := strings.TrimSpace(os.Getenv("METRICS_RETENTION")); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			return d
		}
	}
	return 24 * time.Hour
}

func resolveDSN(configured string) string {
	if configured != "" {
		return configured
	}
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "data"
	}
	return filepath.Join(dataDir, "app.db")
}

func ensureDSNParent(dsn string) error {
	path := dsn
	if strings.HasPrefix(dsn, "file:") {
		path = strings.TrimPrefix(dsn, "file:")
		if i := strings.IndexAny(path, "?#"); i >= 0 {
			path = path[:i]
		}
	}
	if path == "" || path == ":memory:" || strings.HasPrefix(path, ":memory:") {
		return nil
	}
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}

func bootstrapDBPath(dsn string) string {
	p := dsn
	p = strings.TrimPrefix(p, "file:")
	if i := strings.Index(p, "?"); i >= 0 {
		p = p[:i]
	}
	if p == "" {
		p = "data/app.db"
	}
	return filepath.Join(filepath.Dir(p), "bootstrap.db")
}
