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
	casesapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/cases"
	"github.com/mr9esx/comfyui_tgbot/internal/httpapi/comfyinstances"
	sessionsapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/sessions"
	tasksapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/tasks"
	tgmenuapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/tgmenu"
	usersapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/users"
	userpersist "github.com/mr9esx/comfyui_tgbot/internal/identity/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/adminconfig"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/appboot"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/instance"
	instpersist "github.com/mr9esx/comfyui_tgbot/internal/platform/instance/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/notify"
	sesspersist "github.com/mr9esx/comfyui_tgbot/internal/conversation/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/application/orchestrator"
	taskpersist "github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/persistence"
	tgmenuapp "github.com/mr9esx/comfyui_tgbot/internal/tgmenu/application"
	tgmenupersist "github.com/mr9esx/comfyui_tgbot/internal/tgmenu/infrastructure/persistence"
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
		DSN:              dsn,
		MigrateInstances: true,
		Models: []any{
			&casepersist.CaseRow{},
			&userpersist.UserRow{},
			&sesspersist.SessionRow{},
			&taskpersist.TaskRow{},
			&tgmenupersist.MenuHeaderRow{},
			&tgmenupersist.MenuItemRow{},
			&tgmenupersist.MenuItemCaseRow{},
		},
	})
	if err != nil {
		return err
	}
	defer func() { _ = cleanup() }()

	instRepo := instpersist.NewInstanceRepository(gdb)
	pool := instance.NewPool(instRepo, instance.PoolOptions{Mock: cfg.ComfyMock})
	if err := pool.Refresh(ctx); err != nil {
		return err
	}
	caseRepo := casepersist.NewGormRepository(gdb)
	userRepo := userpersist.NewUserRepository(gdb)
	sessionRepo := sesspersist.NewSessionRepository(gdb)
	taskRepo := taskpersist.NewTaskRepository(gdb)
	menuStore := tgmenupersist.NewGormRepository(gdb)
	menuSvc := &tgmenuapp.Service{
		Store: menuStore,
		Cases: tgmenuapp.CatalogCaseChecker{Repo: caseRepo},
	}
	orch := orchestrator.New(taskRepo, pool, nil, notify.Nop{})
	orch.Sessions = sessionRepo

	instAPI := &comfyinstances.Handler{
		Repo:  instRepo,
		Pool:  pool,
		Tasks: taskRepo,
		Mock:  cfg.ComfyMock,
	}
	casesAPI := &casesapi.Handler{
		Repo:     caseRepo,
		Validate: validation.New().ValidateDocument,
	}
	usersAPI := &usersapi.Handler{Repo: userRepo}
	sessionsAPI := &sessionsapi.Handler{Repo: sessionRepo}
	tasksAPI := &tasksapi.Handler{Tasks: taskRepo, Cancel: orch}
	tgMenuAPI := &tgmenuapi.Handler{Svc: menuSvc}

	h := server.NewHandler(server.Options{
		CORSOrigins: cfg.CORSOrigins,
		Instances:   instAPI,
		Cases:       casesAPI,
		Users:       usersAPI,
		Sessions:    sessionsAPI,
		Tasks:       tasksAPI,
		TGMenu:      tgMenuAPI,
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
