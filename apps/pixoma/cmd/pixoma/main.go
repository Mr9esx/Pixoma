package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/mr9esx/comfyui_tgbot/apps/pixoma/internal/app"
	"github.com/mr9esx/comfyui_tgbot/apps/pixoma/internal/webembed"
	casepersist "github.com/mr9esx/comfyui_tgbot/internal/catalog/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/catalog/infrastructure/validation"
	channelapp "github.com/mr9esx/comfyui_tgbot/internal/channel/application"
	channelpersist "github.com/mr9esx/comfyui_tgbot/internal/channel/infrastructure/persistence"
	convdomain "github.com/mr9esx/comfyui_tgbot/internal/conversation/domain"
	sesspersist "github.com/mr9esx/comfyui_tgbot/internal/conversation/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/httpapi/adminhost"
	agentapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/agent"
	casesapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/cases"
	channelsapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/channels"
	"github.com/mr9esx/comfyui_tgbot/internal/httpapi/edges"
	menucardsapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/menucards"
	sessionsapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/sessions"
	setupapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/setup"
	tasksapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/tasks"
	usersapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/users"
	userpersist "github.com/mr9esx/comfyui_tgbot/internal/identity/infrastructure/persistence"
	menuapp "github.com/mr9esx/comfyui_tgbot/internal/menu/application"
	tgmenupersist "github.com/mr9esx/comfyui_tgbot/internal/menu/infrastructure/persistence"
	mencardpersist "github.com/mr9esx/comfyui_tgbot/internal/menucard/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/appboot"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/blob/factory"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/bootstrap"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/botconfig"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/edge"
	instpersist "github.com/mr9esx/comfyui_tgbot/internal/platform/edge/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/presence"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/queue/memory"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/settings"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/application/orchestrator"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/actuator"
	taskpersist "github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

var errRestart = errors.New("setup restart requested")

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	sess := setupapi.NewSessions()
	for {
		err := run(ctx, sess)
		if errors.Is(err, errRestart) && ctx.Err() == nil {
			slog.Info("reloading pixoma after setup")
			continue
		}
		if err != nil {
			slog.Error("pixoma failed", "err", err)
			os.Exit(1)
		}
		return
	}
}

func run(ctx context.Context, sess *setupapi.Sessions) error {
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

	encKey, err := boot.EncKey()
	if err != nil {
		return err
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
		if err := boot.SetRestartRequired(false); err != nil {
			return err
		}
	} else {
		settings.ApplyEnv(&cfg)
	}

	gdb, cleanup, err := appboot.Bootstrap(ctx, appboot.Options{
		Driver:       cfg.DBDriver,
		DSN:          cfg.DBDSN,
		MigrateEdges: true,
		Models: []any{
			&casepersist.CaseRow{},
			&userpersist.UserRow{},
			&userpersist.UserExternalIdentityRow{},
			&sesspersist.SessionRow{},
			&taskpersist.TaskRow{},
			&channelpersist.ChannelRow{},
			&tgmenupersist.ChannelMenuRow{},
			&tgmenupersist.ChannelMenuItemRow{},
			&tgmenupersist.ChannelMenuItemCaseRow{},
			&tgmenupersist.ChannelMenuExtraRow{},
			&mencardpersist.MainMenuRow{},
			&mencardpersist.CardRow{},
		},
	})
	if err != nil {
		return err
	}
	defer func() { _ = cleanup() }()

	app.ApplyBlobEnv(cfg)
	app.ApplyHTTPProxy(cfg)
	blobStore, err := factory.NewFromConfig(botconfig.Config{
		Blob: botconfig.BlobConfig{
			Driver: cfg.BlobDriver,
			TOS: botconfig.BlobTOSConfig{
				Endpoint: cfg.BlobEndpoint,
				Region:   cfg.BlobRegion,
				Bucket:   cfg.BlobBucket,
			},
			S3: botconfig.BlobTOSConfig{
				Endpoint: cfg.BlobEndpoint,
				Region:   cfg.BlobRegion,
				Bucket:   cfg.BlobBucket,
			},
		},
	}, cfg.BlobRoot)
	if err != nil {
		return err
	}

	instRepo := instpersist.NewEdgeRepository(gdb)
	if err := edge.EnsureAgentTokens(ctx, instRepo, encKey); err != nil {
		return err
	}
	pool := edge.NewPool(instRepo, edge.PoolOptions{})
	if err := pool.Refresh(ctx); err != nil {
		return err
	}
	caseRepo := casepersist.NewGormRepository(gdb)
	if n, err := app.SeedCasesDir(ctx, caseRepo, envOr("CASE_SEED_DIR", "configs/cases")); err != nil {
		slog.Warn("seed cases", "err", err)
	} else if n > 0 {
		slog.Info("seed cases loaded", "count", n)
	}
	userRepo := userpersist.NewUserRepository(gdb)
	sessionRepo := sesspersist.NewSessionRepository(gdb)
	sessSvc := convdomain.NewService(sessionRepo, func() sharedkernel.SessionID {
		return sharedkernel.SessionID(uuid.NewString())
	}, nil)
	taskRepo := taskpersist.NewTaskRepository(gdb)
	channelStore := channelpersist.NewGormRepository(gdb)
	chSvc := &channelapp.Service{
		Store: channelStore,
		Key:   encKey,
		HasActiveRefs: func(ctx context.Context, channelID string) (bool, error) {
			n, err := sessionRepo.CountByChannel(ctx, channelID)
			if err != nil {
				return false, err
			}
			return n > 0, nil
		},
	}
	menuStore := tgmenupersist.NewGormRepository(gdb)
	menuSvc := &menuapp.Service{
		Store:            menuStore,
		Cases:            menuapp.CatalogCaseChecker{Repo: caseRepo},
		ListImageCaseIDs: menuapp.CatalogImageCaseIDs(caseRepo),
	}

	bus := memory.New()
	defer func() { _ = bus.Close() }()
	snap := &actuator.CaseSnapshot{
		Tasks: taskRepo,
		Cases: caseRepo,
		Blob:  blobStore,
	}
	botRT, err := app.StartBotRuntime(ctx, app.BotDeps{
		Channels:     chSvc,
		Cases:        caseRepo,
		Sessions:     sessSvc,
		SessionStore: sessionRepo,
		Tasks:        taskRepo,
		Users:        userRepo,
		Menu:         menuSvc,
		Blob:         blobStore,
		Bus:          bus,
	})
	if err != nil {
		return err
	}
	menuSvc.Capabilities = app.MenuCapabilityChecker{Reg: botRT.Capabilities}
	orch := orchestrator.New(taskRepo, pool, nil, botRT.Notify)
	orch.Sessions = sessionRepo
	orch.Prep = snap
	orch.Now = func() time.Time { return time.Now().UTC() }
	if err := app.SubscribeTaskCreated(ctx, bus, orch); err != nil {
		return err
	}
	app.RunScheduler(ctx, orch)

	restartCh := make(chan struct{})
	var restartOnce sync.Once
	setupH := &setupapi.Handler{
		Boot:     boot,
		Sessions: sess,
		DataDir:  dataDir,
		Restart: func() {
			restartOnce.Do(func() { close(restartCh) })
		},
	}
	gate := &setupapi.Gate{Boot: boot, Sessions: sess}
	pres := presence.NewStore()
	metricsRepo := instpersist.NewMetricsRepository(gdb, metricsRetention())

	adminH := adminhost.NewHandler(adminhost.Options{
		CORSOrigins: corsOrigins(),
		Instances:   &edges.Handler{Repo: instRepo, Pool: pool, Tasks: taskRepo, Metrics: metricsRepo, EncKey: encKey, Presence: pres},
		Cases:       &casesapi.Handler{Repo: caseRepo, Validate: validation.New().ValidateDocument},
		Users:       &usersapi.Handler{Repo: userRepo},
		Sessions:    &sessionsapi.Handler{Repo: sessionRepo},
		Tasks:       &tasksapi.Handler{Tasks: taskRepo, Cancel: orch},
		Channels:    &channelsapi.Handler{Svc: chSvc},
		MenuCards:   menucardsapi.NewHandler(mencardpersist.NewGormCardRepository(gdb)),
		NotFound:    webembed.Handler(),
	})

	agentH := &agentapi.Handler{
		Verify: func(ctx context.Context, id sharedkernel.EdgeID, tok string) bool {
			return edge.VerifyAgentToken(ctx, instRepo, encKey, id, tok)
		},
		Tasks:    taskRepo,
		Status:   orch,
		Presence: pres,
		Edges:    instRepo,
		Metrics:  metricsRepo,
		Lease:    leaseDuration(cfg),
	}

	r := chi.NewRouter()
	r.Use(adminhost.CORS(corsOrigins()))
	r.Use(gate.Middleware)
	r.Route("/api/v1/setup", setupH.Mount)
	r.Mount("/", adminH)
	r.Route("/agent/v1", agentH.Mount)

	ln, err := listenWithRetry(addr, 20, 100*time.Millisecond)
	if err != nil {
		return err
	}
	srv := &http.Server{Handler: r, ReadHeaderTimeout: 5 * time.Second}

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
		return shutdownServer(srv)
	case <-restartCh:
		if err := shutdownServer(srv); err != nil {
			return err
		}
		return errRestart
	case err := <-errCh:
		return err
	}
}

func shutdownServer(srv *http.Server) error {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

func listenWithRetry(addr string, attempts int, pause time.Duration) (net.Listener, error) {
	var last error
	for i := 0; i < attempts; i++ {
		ln, err := net.Listen("tcp", addr)
		if err == nil {
			return ln, nil
		}
		last = err
		time.Sleep(pause)
	}
	if last == nil {
		return nil, fmt.Errorf("listen %s: no attempts", addr)
	}
	return nil, last
}

func defaultRuntimeSettings(dataDir string) settings.Settings {
	return settings.Settings{
		Placement:      settings.PlacementLocal,
		DBDriver:       settings.DriverSQLite,
		DBDSN:          filepath.Join(dataDir, "app.db"),
		BlobDriver:     botconfig.BlobDriverLocalFS,
		BlobRoot:       filepath.Join(dataDir, "blob"),
		ComfyMock:      envBool("COMFY_MOCK", true),
		ComfyUIBaseURL: envOr("COMFYUI_BASE_URL", "http://127.0.0.1:8188"),
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

func leaseDuration(cfg settings.Settings) time.Duration {
	if cfg.LeaseSeconds > 0 {
		return time.Duration(cfg.LeaseSeconds) * time.Second
	}
	return 90 * time.Second
}

func metricsRetention() time.Duration {
	if v := strings.TrimSpace(os.Getenv("METRICS_RETENTION")); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			return d
		}
	}
	return 24 * time.Hour
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
