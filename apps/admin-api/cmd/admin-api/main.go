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
	"github.com/mr9esx/comfyui_tgbot/internal/platform/adminconfig"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/appboot"
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

	_, cleanup, err := appboot.Bootstrap(ctx, appboot.Options{
		DSN:              dsn,
		MigrateInstances: true,
	})
	if err != nil {
		return err
	}
	defer func() { _ = cleanup() }()

	h := server.NewHandler(server.Options{CORSOrigins: cfg.CORSOrigins})
	addr := cfg.HTTPAddr
	srv := &http.Server{
		Addr:              addr,
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("admin-api listening", "addr", addr, "dsn", dsn)
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
