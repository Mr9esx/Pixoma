package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Mr9esx/Pixoma/apps/pixoma/internal/config"
	"github.com/Mr9esx/Pixoma/apps/pixoma/internal/demo"
	"github.com/Mr9esx/Pixoma/apps/pixoma/internal/scheduling"
	"github.com/Mr9esx/Pixoma/apps/pixoma/internal/telegram"
	"github.com/Mr9esx/Pixoma/apps/pixoma/internal/webembed"
	consolepersist "github.com/Mr9esx/Pixoma/internal/adminusers/infrastructure/persistence"
	caseapp "github.com/Mr9esx/Pixoma/internal/cases/application"
	catalogdomain "github.com/Mr9esx/Pixoma/internal/cases/domain"
	casepersist "github.com/Mr9esx/Pixoma/internal/cases/infrastructure/persistence"
	"github.com/Mr9esx/Pixoma/internal/cases/infrastructure/validation"
	channelapp "github.com/Mr9esx/Pixoma/internal/channels/application"
	channelpersist "github.com/Mr9esx/Pixoma/internal/channels/infrastructure/persistence"
	textpersist "github.com/Mr9esx/Pixoma/internal/channels/infrastructure/persistence"
	edgeapp "github.com/Mr9esx/Pixoma/internal/edge/application"
	edgedomain "github.com/Mr9esx/Pixoma/internal/edge/domain"
	instpersist "github.com/Mr9esx/Pixoma/internal/edge/infrastructure/persistence"
	"github.com/Mr9esx/Pixoma/internal/edge/infrastructure/presence"
	"github.com/Mr9esx/Pixoma/internal/httpapi/adminhost"
	adminusersapi "github.com/Mr9esx/Pixoma/internal/httpapi/adminusers"
	agentapi "github.com/Mr9esx/Pixoma/internal/httpapi/agent"
	casesapi "github.com/Mr9esx/Pixoma/internal/httpapi/cases"
	channelsapi "github.com/Mr9esx/Pixoma/internal/httpapi/channels"
	"github.com/Mr9esx/Pixoma/internal/httpapi/edges"
	linkhealthapi "github.com/Mr9esx/Pixoma/internal/httpapi/linkhealth"
	"github.com/Mr9esx/Pixoma/internal/httpapi/media"
	routingapi "github.com/Mr9esx/Pixoma/internal/httpapi/routing"
	sessionsapi "github.com/Mr9esx/Pixoma/internal/httpapi/sessions"
	setupapi "github.com/Mr9esx/Pixoma/internal/httpapi/setup"
	statsapi "github.com/Mr9esx/Pixoma/internal/httpapi/stats"
	tasksapi "github.com/Mr9esx/Pixoma/internal/httpapi/tasks"
	topicsapi "github.com/Mr9esx/Pixoma/internal/httpapi/topics"
	usersapi "github.com/Mr9esx/Pixoma/internal/httpapi/users"
	mencardpersist "github.com/Mr9esx/Pixoma/internal/menus/infrastructure/persistence"
	packlink "github.com/Mr9esx/Pixoma/internal/packaging/linkhealth"
	"github.com/Mr9esx/Pixoma/internal/platform/appboot"
	"github.com/Mr9esx/Pixoma/internal/platform/blob/factory"
	"github.com/Mr9esx/Pixoma/internal/platform/bootstrap"
	"github.com/Mr9esx/Pixoma/internal/platform/botconfig"
	"github.com/Mr9esx/Pixoma/internal/platform/queue/memory"
	convdomain "github.com/Mr9esx/Pixoma/internal/sessions/domain"
	sesspersist "github.com/Mr9esx/Pixoma/internal/sessions/infrastructure/persistence"
	settingsdomain "github.com/Mr9esx/Pixoma/internal/settings/domain"
	settingsinfra "github.com/Mr9esx/Pixoma/internal/settings/infrastructure"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
	taskstatspersist "github.com/Mr9esx/Pixoma/internal/stats/infrastructure/persistence"
	"github.com/Mr9esx/Pixoma/internal/tasks/application/orchestrator"
	"github.com/Mr9esx/Pixoma/internal/tasks/domain/condition"
	"github.com/Mr9esx/Pixoma/internal/tasks/infrastructure/actuator"
	taskpersist "github.com/Mr9esx/Pixoma/internal/tasks/infrastructure/persistence"
	topicapp "github.com/Mr9esx/Pixoma/internal/topics/application"
	topicpersist "github.com/Mr9esx/Pixoma/internal/topics/infrastructure/persistence"
	identitydomain "github.com/Mr9esx/Pixoma/internal/users/domain"
	userpersist "github.com/Mr9esx/Pixoma/internal/users/infrastructure/persistence"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// OptionsFromEnv reads process env into production server options.
func OptionsFromEnv() Options {
	dataDir := envOr("DATA_DIR", "data")
	addr := envOr("HTTP_ADDR", "127.0.0.1:8082")
	return Options{
		DataDir:   dataDir,
		HTTPAddr:  addr,
		PublicURL: envOr("PUBLIC_URL", "http://"+addr),
	}
}

// Options configures a production Pixoma server run.
type Options struct {
	DataDir   string
	HTTPAddr  string
	PublicURL string
}

// ErrRestart reports that setup requested a clean reload.
var ErrRestart = errors.New("setup restart requested")

// Run starts the production server and performs the setup-triggered reload
// loop.
func Run(ctx context.Context, opts Options) error {
	if opts.DataDir == "" {
		opts.DataDir = envOr("DATA_DIR", "data")
	}
	if opts.HTTPAddr == "" {
		opts.HTTPAddr = envOr("HTTP_ADDR", "127.0.0.1:8082")
	}
	if opts.PublicURL == "" {
		opts.PublicURL = "http://" + opts.HTTPAddr
	}
	sess := setupapi.NewSessions(filepath.Join(opts.DataDir, setupapi.SessionStoreFile()))
	var listenRetries int
	for {
		err := run(ctx, sess, opts)
		if errors.Is(err, ErrRestart) && ctx.Err() == nil {
			listenRetries = 0
			slog.Info("reloading pixoma after setup")
			continue
		}
		if ctx.Err() == nil && isListenBusy(err) && listenRetries < 5 {
			listenRetries++
			slog.Error("pixoma listen failed; retrying", "err", err, "attempt", listenRetries)
			time.Sleep(200 * time.Millisecond)
			continue
		}
		return err
	}
}

func RunLiveDemo(ctx context.Context, opts Options) error {
	addr := opts.HTTPAddr
	listenURL := opts.PublicURL
	if listenURL == "" {
		listenURL = "http://" + addr
	}
	server, err := demo.New(ctx, addr, listenURL)
	if err != nil {
		return err
	}
	slog.Info("pixoma listening", "addr", addr, "mode", "live-demo")

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	}
}

func run(ctx context.Context, sess *setupapi.Sessions, opts Options) error {
	dataDir := opts.DataDir
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
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

	addr := opts.HTTPAddr
	listenURL := opts.PublicURL
	if listenURL == "" {
		listenURL = "http://" + addr
	}
	bannerPass := creds.Password
	if boot.MustChangePassword() && bannerPass == "" {
		// 重启后明文密码不随 Open 返回（只存 bcrypt 哈希），
		// 未完成初始化时从落盘文件恢复，便于再次展示默认密码。
		if p, err := bootstrap.ReadStoredPassword(bootPath); err == nil {
			bannerPass = p
		}
	}
	if !boot.MustChangePassword() {
		bannerPass = ""
	}
	fmt.Print(config.StartupBanner(config.BannerInput{
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
		settingsdomain.ApplyEnv(&loaded)
		if err := loaded.Validate(); err != nil {
			return err
		}
		cfg = loaded
		if err := boot.SetRestartRequired(false); err != nil {
			return err
		}
	} else {
		settingsdomain.ApplyEnv(&cfg)
	}

	gdb, cleanup, err := appboot.Bootstrap(ctx, appboot.Options{
		Driver:       cfg.DBDriver,
		DSN:          cfg.DBDSN,
		MigrateEdges: true,
		Models: []any{
			&casepersist.CaseRow{},
			&userpersist.UserRow{},
			&userpersist.UserExternalIdentityRow{},
			&consolepersist.ConsoleUserRow{},
			&sesspersist.SessionRow{},
			&taskpersist.TaskRow{},
			&taskstatspersist.DailyStatsRow{},
			&taskstatspersist.EdgeDailyStatsRow{},
			&taskstatspersist.ErrorDailyStatsRow{},
			&taskstatspersist.CaseDailyStatsRow{},
			&topicpersist.TopicRow{},
			&channelpersist.ChannelRow{},
			&mencardpersist.MainMenuRow{},
			&mencardpersist.CardRow{},
		},
	})
	if err != nil {
		return err
	}
	defer func() { _ = cleanup() }()

	runCtx, runCancel := context.WithCancel(ctx)
	defer runCancel()

	if err := bootstrap.MigrateBootstrapAdmin(ctx, gdb, boot); err != nil {
		return err
	}

	config.ApplyBlobEnv(cfg)
	config.ApplyHTTPProxy(cfg)
	textStore, err := textpersist.NewStore(gdb)
	if err != nil {
		return err
	}

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
	if err := edgedomain.EnsureAgentTokens(ctx, instRepo, encKey); err != nil {
		return err
	}
	topicRepo := topicpersist.NewTopicRepository(gdb)
	if err := topicapp.EnsureDefaultTopic(ctx, topicRepo); err != nil {
		return err
	}
	pool := edgeapp.NewPool(instRepo, edgeapp.PoolOptions{})
	if err := pool.Refresh(ctx); err != nil {
		return err
	}
	caseRepo := casepersist.NewGormRepository(gdb)
	userRepo := userpersist.NewUserRepository(gdb)
	userRepo.SetDefaultAccess(func() identitydomain.UserAccess {
		return identitydomain.NormalizeUserAccess(cfg.DefaultUserAccess)
	})
	consoleRepo := consolepersist.NewConsoleUserRepository(gdb)
	sessionRepo := sesspersist.NewSessionRepository(gdb)
	sessSvc := convdomain.NewService(sessionRepo, func() sharedkernel.SessionID {
		return sharedkernel.SessionID(uuid.NewString())
	}, nil)
	taskRepo := taskpersist.NewTaskRepository(gdb)
	channelStore := channelpersist.NewGormRepository(gdb)
	chSvc := &channelapp.Service{
		Store:             channelStore,
		Key:               encKey,
		DeleteWithCleanup: channelapp.DeleteWithCleanup(gdb),
	}
	bus := memory.New()
	defer func() { _ = bus.Close() }()
	snap := &actuator.CaseSnapshot{
		Tasks: taskRepo,
		Cases: caseRepo,
		Blob:  blobStore,
	}
	botRT, err := telegram.StartBotRuntime(runCtx, telegram.BotDeps{
		Channels:     chSvc,
		Cases:        caseRepo,
		Sessions:     sessSvc,
		SessionStore: sessionRepo,
		Tasks:        taskRepo,
		Users:        userRepo,
		MenuCards:    mencardpersist.NewGormCardRepository(gdb),
		Blob:         blobStore,
		Bus:          bus,
		Texts:        textStore,
	})
	if err != nil {
		return err
	}
	chSvc.Notify = botRT.Notify
	chSvc.AdapterStatus = func(_ context.Context, id string) (state string, lastErr string, found bool) {
		return botRT.ChannelStatus(id)
	}
	edgeDeleteSvc := edgeapp.NewService(gdb, botRT.Notify)
	topicDeleteSvc := topicapp.NewService(gdb, botRT.Notify)
	conditionReg := condition.NewRegistry()

	orch := orchestrator.New(taskRepo, pool, nil, botRT.Notify)
	orch.Sessions = sessionRepo
	orch.Prep = snap
	orch.Now = func() time.Time { return time.Now().UTC() }
	orch.Cases = caseDocReader{repo: caseRepo}
	orch.Condition = conditionReg
	if err := scheduling.SubscribeTaskCreated(runCtx, bus, orch); err != nil {
		return err
	}
	scheduling.RunScheduler(runCtx, orch)

	restartCh := make(chan struct{})
	var restartOnce sync.Once
	setupH := &setupapi.Handler{
		Boot:         boot,
		Sessions:     sess,
		DataDir:      dataDir,
		PublicURL:    listenURL,
		ConsoleUsers: consoleRepo,
		Restart: func() {
			restartOnce.Do(func() { close(restartCh) })
		},
	}
	gate := &setupapi.Gate{Boot: boot, Sessions: sess, ConsoleUsers: consoleRepo}
	pres := presence.NewStore()
	metricsRepo := instpersist.NewMetricsRepository(gdb, metricsRetention())
	statsRepo := taskstatspersist.NewGormStatsRepository(gdb, statsRetention(), statsLocation())
	orch.Stats = statsRepo

	validator := validation.New()
	menuRepo := mencardpersist.NewGormCardRepository(gdb)
	caseDeleteSvc := caseapp.NewService(gdb, botRT.Notify)
	adminH := adminhost.NewHandler(adminhost.Options{
		CORSOrigins: corsOrigins(),
		Instances:   &edges.Handler{Repo: instRepo, Pool: pool, Tasks: taskRepo, Metrics: metricsRepo, EncKey: encKey, Presence: pres, Topics: topicRepo, DeleteWithCleanup: edgeDeleteSvc.DeleteEdge},
		Cases: &casesapi.Handler{Repo: caseRepo, Validate: func(doc catalogdomain.CaseDocument) error {
			if err := validator.ValidateDocument(doc); err != nil {
				return err
			}
			return validation.ValidateRouting(context.Background(), doc.Routing, topicRepo, conditionReg)
		}, DeleteWithCleanup: caseDeleteSvc.DeleteCase},
		AdminUsers: &adminusersapi.Handler{Repo: consoleRepo},
		Users:      &usersapi.Handler{Repo: userRepo, Channels: channelStore},
		Sessions:   &sessionsapi.Handler{Repo: sessionRepo, Channels: channelStore, Context: sesspersist.NewSessionAdminProjection(gdb)},
		Tasks:      &tasksapi.Handler{Tasks: taskRepo, Cancel: orch, Context: taskpersist.NewTaskAdminProjection(gdb)},
		Stats:      &statsapi.Handler{Repo: statsRepo, Loc: statsLocation(), Metrics: metricsRepo},
		Channels: &channelsapi.Handler{
			Svc: chSvc,
			OnCreated: func(ctx context.Context, id string) error {
				return textStore.Seed(ctx, id)
			},
			Probe:    botRT.Probe,
			ProbeCtx: runCtx,
		},
		MenuHandler: channelsapi.NewMenuHandler(menuRepo),
		Topics: &topicsapi.Handler{
			Repo:              topicRepo,
			Tasks:             taskRepo,
			DeleteWithCleanup: topicDeleteSvc.DeleteTopic,
		},
		Routing:     &routingapi.Handler{Registry: conditionReg},
		Media:       &media.Handler{Blob: blobStore, MaxBytes: mediaMediaMaxBytes(cfg)},
		TextHandler: &channelsapi.TextHandler{Store: textStore},
		LinkHealth: &linkhealthapi.Handler{
			Snapshot: func(r *http.Request) (packlink.Snapshot, error) {
				return linkhealthapi.Collect(r.Context(), linkhealthapi.Source{
					Channels: channelStore,
					Adapter:  chSvc.AdapterStatus,
					Menus:    menuRepo,
					Cases:    caseRepo,
					Topics:   topicRepo,
					Edges:    instRepo,
					Presence: pres,
				})
			},
		},
		NotFound: webembed.Handler(),
	})

	agentH := &agentapi.Handler{
		Verify: func(ctx context.Context, id sharedkernel.EdgeID, tok string) bool {
			return edgedomain.VerifyAgentToken(ctx, instRepo, encKey, id, tok)
		},
		Tasks:    taskRepo,
		Status:   orch,
		Presence: pres,
		Edges:    instRepo,
		Metrics:  metricsRepo,
		Lease:    leaseDuration(cfg),
	}

	r := chi.NewRouter()
	r.Use(adminhost.SecurityHeaders)
	r.Use(adminhost.RequestBodyLimit(requestBodyLimit(cfg)))
	r.Use(adminhost.CORS(corsOrigins()))
	r.Use(gate.Middleware)
	r.Route("/api/v1/setup", setupH.Mount)
	r.Route("/api/v1/auth", setupH.MountAuth)
	r.Mount("/", adminH)
	r.Route("/agent/v1", agentH.Mount)

	ln, err := listenWithRetry(addr, 50, 100*time.Millisecond)
	if err != nil {
		return err
	}
	srv := &http.Server{Handler: r, ReadHeaderTimeout: 5 * time.Second}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("pixoma listening", "addr", addr, "data_dir", dataDir, "initialized", boot.Initialized())
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case <-ctx.Done():
		runCancel()
		waitChannelRuntime(botRT)
		return shutdownServer(srv)
	case <-restartCh:
		return finishReload(srv, runCancel, botRT)
	case err := <-errCh:
		runCancel()
		waitChannelRuntime(botRT)
		return err
	}
}

func finishReload(srv *http.Server, runCancel context.CancelFunc, rt *telegram.BotRuntime) error {
	_ = shutdownServer(srv)
	if runCancel != nil {
		runCancel()
	}
	waitChannelRuntime(rt)
	return ErrRestart
}

func waitChannelRuntime(rt *telegram.BotRuntime) {
	if rt == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	if err := rt.Wait(ctx); err != nil {
		slog.Warn("channel runtime stop timed out", "err", err)
	}
}

func shutdownServer(srv *http.Server) error {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err := srv.Shutdown(shutdownCtx)
	_ = srv.Close()
	return err
}

func isListenBusy(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, syscall.EADDRINUSE) {
		return true
	}
	return strings.Contains(err.Error(), "address already in use")
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

func defaultRuntimeSettings(dataDir string) settingsdomain.Settings {
	return settingsdomain.Settings{
		Placement:      settingsdomain.PlacementLocal,
		DBDriver:       settingsdomain.DriverSQLite,
		DBDSN:          filepath.Join(dataDir, "app.db"),
		BlobDriver:     botconfig.BlobDriverLocalFS,
		BlobRoot:       filepath.Join(dataDir, "blob"),
		ComfyUIBaseURL: envOr("COMFYUI_BASE_URL", "http://127.0.0.1:8188"),
	}
}

func loadSavedSettings(boot *bootstrap.Store) (settingsdomain.Settings, error) {
	driver, dsn, err := boot.AppDB()
	if err != nil {
		return settingsdomain.Settings{}, err
	}
	if strings.TrimSpace(dsn) == "" {
		return settingsdomain.Settings{}, fmt.Errorf("initialized but app db dsn is empty")
	}
	key, err := boot.EncKey()
	if err != nil {
		return settingsdomain.Settings{}, err
	}
	gdb, cleanup, err := appboot.Bootstrap(context.Background(), appboot.Options{
		Driver: driver,
		DSN:    dsn,
	})
	if err != nil {
		return settingsdomain.Settings{}, err
	}
	defer func() { _ = cleanup() }()
	st, err := settingsinfra.NewStore(gdb, key)
	if err != nil {
		return settingsdomain.Settings{}, err
	}
	return st.Load()
}

func leaseDuration(cfg settingsdomain.Settings) time.Duration {
	if cfg.LeaseSeconds > 0 {
		return time.Duration(cfg.LeaseSeconds) * time.Second
	}
	return 90 * time.Second
}

// mediaMediaMaxBytes returns the configured per-upload media cap, falling back
// to media.DefaultMaxBytes when the value is unset or below the lower bound.
func mediaMediaMaxBytes(cfg settingsdomain.Settings) int64 {
	if cfg.MediaMaxBytes >= settingsdomain.MinMediaMaxBytes {
		return cfg.MediaMaxBytes
	}
	return media.DefaultMaxBytes
}

// requestBodyLimit returns the HTTP request body limit for admin routes. It
// tracks cfg.MediaMaxBytes so the body cap never sits below the media upload
// cap. A small overhead is added for JSON envelopes and multipart framing.
func requestBodyLimit(cfg settingsdomain.Settings) int64 {
	const overhead int64 = 1 << 20 // 1 MiB slack for headers / multipart envelope
	limit := mediaMediaMaxBytes(cfg) + overhead
	const floor int64 = 32 << 20 // legacy default — never go below 32 MiB
	if limit < floor {
		return floor
	}
	return limit
}

func metricsRetention() time.Duration {
	if v := strings.TrimSpace(os.Getenv("METRICS_RETENTION")); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			return d
		}
	}
	return 24 * time.Hour
}

func statsRetention() time.Duration {
	if v := strings.TrimSpace(os.Getenv("TASK_STATS_RETENTION")); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			return d
		}
	}
	return 365 * 24 * time.Hour
}

func statsLocation() *time.Location {
	v := strings.TrimSpace(os.Getenv("STATS_TIMEZONE"))
	if v == "" {
		v = "Asia/Shanghai"
	}
	loc, err := time.LoadLocation(v)
	if err != nil {
		return time.FixedZone("Asia/Shanghai", 8*3600)
	}
	return loc
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

// caseDocReader adapts the catalog repository to the orchestrator CaseReader
// port (routing evaluation only needs the protocol document).
type caseDocReader struct {
	repo *casepersist.GormRepository
}

func (c caseDocReader) GetCase(ctx context.Context, id sharedkernel.CaseID) (*catalogdomain.CaseDocument, error) {
	got, err := c.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return &got.Document, nil
}
