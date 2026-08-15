package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/mr9esx/comfyui_tgbot/apps/pixoma/internal/app"
	"github.com/mr9esx/comfyui_tgbot/apps/pixoma/internal/webembed"
	casepersist "github.com/mr9esx/comfyui_tgbot/internal/catalog/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/catalog/infrastructure/validation"
	sesspersist "github.com/mr9esx/comfyui_tgbot/internal/conversation/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/httpapi/adminhost"
	agentapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/agent"
	casesapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/cases"
	"github.com/mr9esx/comfyui_tgbot/internal/httpapi/comfyinstances"
	sessionsapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/sessions"
	setupapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/setup"
	tasksapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/tasks"
	tgmenuapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/tgmenu"
	usersapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/users"
	userpersist "github.com/mr9esx/comfyui_tgbot/internal/identity/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/appboot"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/blob/factory"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/bootstrap"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/botconfig"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/instance"
	instpersist "github.com/mr9esx/comfyui_tgbot/internal/platform/instance/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/notify"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/settings"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/application/orchestrator"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/actuator"
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
		slog.Error("pixoma failed", "err", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	dataDir := envOr("DATA_DIR", "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return err
	}

	bootPath := filepath.Join(dataDir, "bootstrap.db")
	boot, creds, err := bootstrap.Open(bootPath)
	if err != nil {
		return err
	}
	defer boot.Close()

	agentTok, minted, err := boot.EnsureAgentToken()
	if err != nil {
		return err
	}
	tokenFile := filepath.Join(dataDir, "agent.token")
	if minted {
		if err := app.WriteAgentTokenFile(tokenFile, agentTok); err != nil {
			return err
		}
		slog.Info("minted agent token", "path", tokenFile)
	} else {
		agentTok, err = app.ReadAgentTokenFile(tokenFile)
		if err != nil {
			return fmt.Errorf("agent token hash exists but %s unreadable: %w (delete bootstrap to remint)", tokenFile, err)
		}
	}

	addr := envOr("HTTP_ADDR", "127.0.0.1:8080")
	listenURL := envOr("PUBLIC_URL", "http://"+addr)
	bannerPass := creds.Password
	if !boot.MustChangePassword() {
		bannerPass = ""
	}
	fmt.Print(app.StartupBanner(app.BannerInput{
		ListenURL: listenURL,
		Username:  creds.Username,
		Password:  bannerPass,
	}))

	cfg := defaultRuntimeSettings(dataDir)
	if boot.Initialized() {
		loaded, err := loadSavedSettings(boot)
		if err != nil {
			return err
		}
		settings.ApplyEnv(&loaded)
		if err := loaded.Validate(); err != nil {
			return err
		}
		cfg = loaded
	} else {
		settings.ApplyEnv(&cfg)
	}

	gdb, cleanup, err := appboot.Bootstrap(ctx, appboot.Options{
		Driver:           cfg.DBDriver,
		DSN:              cfg.DBDSN,
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
		Seed: &instance.SeedConfig{
			DefaultInstanceID: cfg.DefaultInstanceID,
			ComfyUIBaseURL:    cfg.ComfyUIBaseURL,
			ComfyMock:         cfg.ComfyMock,
		},
	})
	if err != nil {
		return err
	}
	defer func() { _ = cleanup() }()

	applyBlobSecrets(cfg)
	blobStore, err := factory.NewFromConfig(botconfig.Config{
		Blob: botconfig.BlobConfig{
			Driver: cfg.BlobDriver,
			TOS: botconfig.BlobTOSConfig{
				Endpoint: cfg.BlobEndpoint,
				Region:   cfg.BlobRegion,
				Bucket:   cfg.BlobBucket,
			},
		},
	}, cfg.BlobRoot)
	if err != nil {
		return err
	}

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
		Store:            menuStore,
		Cases:            tgmenuapp.CatalogCaseChecker{Repo: caseRepo},
		ListImageCaseIDs: tgmenuapp.CatalogImageCaseIDs(caseRepo),
	}

	snap := &actuator.CaseSnapshot{
		Tasks: taskRepo,
		Cases: caseRepo,
		Blob:  blobStore,
	}
	orch := orchestrator.New(taskRepo, pool, nil, notify.Nop{})
	orch.Sessions = sessionRepo
	orch.Prep = snap
	orch.Now = func() time.Time { return time.Now().UTC() }

	sess := setupapi.NewSessions()
	setupH := &setupapi.Handler{Boot: boot, Sessions: sess, DataDir: dataDir}
	gate := &setupapi.Gate{Boot: boot, Sessions: sess}

	adminH := adminhost.NewHandler(adminhost.Options{
		CORSOrigins: corsOrigins(),
		Instances:   &comfyinstances.Handler{Repo: instRepo, Pool: pool, Tasks: taskRepo, Mock: cfg.ComfyMock},
		Cases:       &casesapi.Handler{Repo: caseRepo, Validate: validation.New().ValidateDocument},
		Users:       &usersapi.Handler{Repo: userRepo},
		Sessions:    &sessionsapi.Handler{Repo: sessionRepo},
		Tasks:       &tasksapi.Handler{Tasks: taskRepo, Cancel: orch},
		TGMenu:      &tgmenuapi.Handler{Svc: menuSvc},
		NotFound:    webembed.Handler(),
	})

	agentH := &agentapi.Handler{
		Token:  agentTok,
		Tasks:  taskRepo,
		Status: orch,
		Lease:  leaseDuration(cfg),
	}

	r := chi.NewRouter()
	r.Use(adminhost.CORS(corsOrigins()))
	r.Use(gate.Middleware)
	r.Route("/api/v1/setup", setupH.Mount)
	r.Mount("/", adminH)
	r.Route("/agent/v1", agentH.Mount)

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	srv := &http.Server{Handler: r, ReadHeaderTimeout: 5 * time.Second}

	var edgeCmd = (*os.Process)(nil)
	if shouldSpawnEdge(boot.Initialized(), cfg) {
		cmd := app.EdgeCommand(app.EdgeSpawnConfig{
			Binary:          app.ResolveEdgeBinary(),
			ControlPlaneURL: listenURL,
			AgentToken:      agentTok,
			InstanceID:      cfg.DefaultInstanceID,
			BlobDriver:      cfg.BlobDriver,
			BlobRoot:        cfg.BlobRoot,
			ComfyMock:       cfg.ComfyMock,
		})
		if err := cmd.Start(); err != nil {
			slog.Warn("edge auto-spawn failed (install pixoma-edge-agent or set EDGE_AGENT_BIN)", "err", err)
		} else {
			edgeCmd = cmd.Process
			slog.Info("spawned local edge", "pid", cmd.Process.Pid, "bin", cmd.Path)
			go func() { _ = cmd.Wait() }()
		}
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("pixoma listening", "addr", addr, "data_dir", dataDir, "comfy_mock", cfg.ComfyMock, "initialized", boot.Initialized())
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case <-ctx.Done():
		if edgeCmd != nil {
			_ = edgeCmd.Signal(syscall.SIGTERM)
			_, _ = edgeCmd.Wait()
		}
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	case err := <-errCh:
		if edgeCmd != nil {
			_ = edgeCmd.Signal(syscall.SIGTERM)
		}
		return err
	}
}

func defaultRuntimeSettings(dataDir string) settings.Settings {
	return settings.Settings{
		Placement:         settings.PlacementLocal,
		DBDriver:          settings.DriverSQLite,
		DBDSN:             filepath.Join(dataDir, "app.db"),
		BlobDriver:        botconfig.BlobDriverLocalFS,
		BlobRoot:          filepath.Join(dataDir, "blob"),
		ComfyMock:         envBool("COMFY_MOCK", true),
		ComfyUIBaseURL:    envOr("COMFYUI_BASE_URL", "http://127.0.0.1:8188"),
		DefaultInstanceID: envOr("INSTANCE_ID", "local"),
		AutoSpawnEdge:     true,
	}
}

func loadSavedSettings(boot *bootstrap.Store) (settings.Settings, error) {
	driver, dsn, err := boot.AppDB()
	if err != nil {
		return settings.Settings{}, err
	}
	if strings.TrimSpace(dsn) == "" {
		return settings.Settings{}, fmt.Errorf("initialized but app db dsn is empty")
	}
	key, err := boot.EncKey()
	if err != nil {
		return settings.Settings{}, err
	}
	gdb, cleanup, err := appboot.Bootstrap(context.Background(), appboot.Options{
		Driver: driver,
		DSN:    dsn,
	})
	if err != nil {
		return settings.Settings{}, err
	}
	defer func() { _ = cleanup() }()
	st, err := settings.NewStore(gdb, key)
	if err != nil {
		return settings.Settings{}, err
	}
	return st.Load()
}

func applyBlobSecrets(cfg settings.Settings) {
	if cfg.BlobDriver == botconfig.BlobDriverTOS {
		if os.Getenv("TOS_ACCESS_KEY") == "" && cfg.BlobAccessKey != "" {
			_ = os.Setenv("TOS_ACCESS_KEY", cfg.BlobAccessKey)
		}
		if os.Getenv("TOS_SECRET_KEY") == "" && cfg.BlobSecretKey != "" {
			_ = os.Setenv("TOS_SECRET_KEY", cfg.BlobSecretKey)
		}
		return
	}
	if os.Getenv("S3_ACCESS_KEY") == "" && cfg.BlobAccessKey != "" {
		_ = os.Setenv("S3_ACCESS_KEY", cfg.BlobAccessKey)
	}
	if os.Getenv("S3_SECRET_KEY") == "" && cfg.BlobSecretKey != "" {
		_ = os.Setenv("S3_SECRET_KEY", cfg.BlobSecretKey)
	}
}

func shouldSpawnEdge(initialized bool, cfg settings.Settings) bool {
	if !envBool("EDGE_AUTO_SPAWN", true) {
		return false
	}
	if !initialized {
		return true
	}
	return cfg.Placement == settings.PlacementLocal && cfg.AutoSpawnEdge
}

func leaseDuration(cfg settings.Settings) time.Duration {
	if cfg.LeaseSeconds > 0 {
		return time.Duration(cfg.LeaseSeconds) * time.Second
	}
	return 90 * time.Second
}

func corsOrigins() []string {
	raw := strings.TrimSpace(os.Getenv("CORS_ORIGINS"))
	if raw != "" {
		parts := strings.Split(raw, ",")
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			if s := strings.TrimSpace(p); s != "" {
				out = append(out, s)
			}
		}
		return out
	}
	return []string{
		"http://127.0.0.1:5173",
		"http://localhost:5173",
	}
}

func envOr(k, def string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return def
}

func envBool(k string, def bool) bool {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		return def
	}
	switch strings.ToLower(v) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return def
	}
}
