package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"

	botserver "github.com/mr9esx/comfyui_tgbot/apps/bot/internal/server"
	catalogdomain "github.com/mr9esx/comfyui_tgbot/internal/catalog/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/catalog/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/catalog/infrastructure/validation"
	channelapp "github.com/mr9esx/comfyui_tgbot/internal/channel/application"
	channelpersist "github.com/mr9esx/comfyui_tgbot/internal/channel/infrastructure/persistence"
	channelruntime "github.com/mr9esx/comfyui_tgbot/internal/channel/runtime"
	convdomain "github.com/mr9esx/comfyui_tgbot/internal/conversation/domain"
	convpersist "github.com/mr9esx/comfyui_tgbot/internal/conversation/infrastructure/persistence"
	identitypersist "github.com/mr9esx/comfyui_tgbot/internal/identity/infrastructure/persistence"
	mencardpersist "github.com/mr9esx/comfyui_tgbot/internal/menucard/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/packaging/botapp"
	goredis "github.com/redis/go-redis/v9"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/appboot"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/blob"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/blob/factory"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/bootstrap"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/botconfig"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/edge"
	instpersist "github.com/mr9esx/comfyui_tgbot/internal/platform/edge/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/edgeonline"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/queue"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/queue/memory"
	queueredis "github.com/mr9esx/comfyui_tgbot/internal/platform/queue/redis"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/application/orchestrator"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/actuator"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/comfyui"
	taskpersist "github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := run(ctx); err != nil {
		slog.Error("bot failed", "err", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	cfg, err := botconfig.Load("")
	if err != nil {
		return err
	}
	if err := cfg.ValidateRuntimeDrivers(); err != nil {
		return err
	}

	dataDir := envOr("DATA_DIR", "data")
	_ = os.MkdirAll(dataDir, 0o755)

	dsn := resolveDSN(cfg.DatabaseDSN)
	bootMeta, _, err := bootstrap.Open(filepath.Join(dataDir, "bootstrap.db"))
	if err != nil {
		return err
	}
	encKey, err := bootMeta.EncKey()
	if err != nil {
		return err
	}
	gdb, cleanup, err := appboot.Bootstrap(ctx, appboot.Options{
		DSN:          dsn,
		MigrateEdges: true,
		Models: []any{
			&persistence.CaseRow{},
			&identitypersist.UserRow{},
			&identitypersist.UserExternalIdentityRow{},
			&convpersist.SessionRow{},
			&taskpersist.TaskRow{},
			&channelpersist.ChannelRow{},
			&mencardpersist.MainMenuRow{},
			&mencardpersist.CardRow{},
		},
	})
	if err != nil {
		return err
	}
	if err := gdb.Migrator().DropTable(
		"channel_menus",
		"channel_menu_items",
		"channel_menu_item_cases",
		"channel_menu_item_extras",
	); err != nil {
		return err
	}
	defer func() { _ = cleanup() }()

	caseRepo := persistence.NewGormRepository(gdb)
	userRepo := identitypersist.NewUserRepository(gdb)
	if n, err := seedCasesDir(ctx, caseRepo, cfg.CaseSeedDir); err != nil {
		slog.Warn("seed cases", "err", err)
	} else {
		slog.Info("seed cases loaded", "count", n)
	}

	blobStore, err := openBlobStore(cfg, dataDir)
	if err != nil {
		return err
	}

	bus, redisClient, err := openQueueBus(cfg)
	if err != nil {
		return err
	}
	if redisClient != nil {
		defer func() { _ = redisClient.Close() }()
	}
	tasks := taskpersist.NewTaskRepository(gdb)
	sessRepo := convpersist.NewSessionRepository(gdb)
	sessSvc := convdomain.NewService(sessRepo, func() sharedkernel.SessionID {
		return sharedkernel.SessionID(uuid.NewString())
	}, nil)

	instRepo := instpersist.NewEdgeRepository(gdb)
	seedCfg := edge.SeedConfig{}
	for _, s := range cfg.Edges {
		seedCfg.Edges = append(seedCfg.Edges, edge.SeedInstance{
			ID:           s.ID,
			Enabled:      s.Enabled,
			Capabilities: s.Capabilities,
		})
	}
	if n, err := edge.SeedFromConfig(ctx, instRepo, seedCfg); err != nil {
		return err
	} else {
		slog.Info("comfy instances seeded", "count", n)
	}

	pool := edge.NewPool(instRepo, edge.PoolOptions{})
	if err := pool.Refresh(ctx); err != nil {
		return err
	}

	// Edge records no longer carry a Comfy base URL; the legacy bot builds its
	// own clients from bot YAML (comfy_instances / comfyui_base_url).
	clients := map[sharedkernel.EdgeID]comfyui.Client{}
	for _, s := range cfg.Edges {
		cli, err := comfyui.NewClient(comfyui.Options{Mock: cfg.ComfyMock, BaseURL: s.BaseURL})
		if err != nil {
			return err
		}
		clients[sharedkernel.EdgeID(s.ID)] = cli
	}
	if len(clients) == 0 {
		id := "local"
		baseURL := cfg.ComfyUIBaseURL
		if baseURL == "" {
			baseURL = "http://127.0.0.1:8188"
		}
		cli, err := comfyui.NewClient(comfyui.Options{Mock: cfg.ComfyMock, BaseURL: baseURL})
		if err != nil {
			return err
		}
		clients[sharedkernel.EdgeID(id)] = cli
	}
	resolveClient := func(id sharedkernel.EdgeID) (comfyui.Client, error) {
		cli, ok := clients[id]
		if !ok || cli == nil {
			return nil, fmt.Errorf("bot: no comfy client for %s", id)
		}
		return cli, nil
	}

	instID := sharedkernel.EdgeID("local")
	comfy, err := resolveClient(instID)
	if err != nil {
		// Fall back to first healthy/enabled instance when default id is absent.
		healthy, listErr := pool.ListHealthy(ctx, edge.CapabilityFilter{})
		if listErr != nil || len(healthy) == 0 {
			return fmt.Errorf("comfy client for %s: %w", instID, err)
		}
		instID = healthy[0].ID
		comfy, err = resolveClient(instID)
		if err != nil {
			return err
		}
	}
	slog.Info("comfyui client ready", "mock", cfg.ComfyMock, "edge_id", instID)

	chSvc := &channelapp.Service{
		Store: channelpersist.NewGormRepository(gdb),
		Key:   encKey,
		HasActiveRefs: func(ctx context.Context, channelID string) (bool, error) {
			n, err := sessRepo.CountByChannel(ctx, channelID)
			if err != nil {
				return false, err
			}
			return n > 0, nil
		},
	}
	notifyRegistry := &botNotifyRegistry{handlers: map[string]channelruntime.NotifyHandler{}}
	notifyRouter := &channelruntime.NotifyRouter{HandlerByChannel: notifyRegistry.lookup}
	orch := orchestrator.New(tasks, pool, bus, notifyRouter)
	orch.Sessions = sessRepo
	if cfg.RuntimeMode == botconfig.RuntimeModeSplit {
		if redisClient == nil {
			return fmt.Errorf("split mode requires redis queue client")
		}
		orch.Online = edgeonline.Checker(redisClient)
	}

	snap := &actuator.CaseSnapshot{
		Tasks:    tasks,
		Cases:    caseRepo,
		Blob:     blobStore,
		Uploader: comfy,
	}
	worker := &actuator.Worker{
		EdgeID:        instID,
		Comfy:         comfy,
		ResolveClient: resolveClient,
		Blob:          blobStore,
		Status:        bus,
		Workflows:     snap,
	}
	orch.Prep = snap
	orch.Query = &actuator.QueryAdapter{Tasks: tasks}

	facade := &botapp.Facade{
		Cases:        caseRepo,
		Validator:    validation.New(),
		Sessions:     sessSvc,
		SessionStore: sessRepo,
		Tasks:        tasks,
		Blob:         blobStore,
		Publisher:    bus,
		NewTaskID: func() sharedkernel.TaskID {
			return sharedkernel.TaskID(uuid.NewString())
		},
	}

	caps := botCapabilities(facade)
	assembler := &channelruntime.Assembler{
		Store: &botChannelSnapshotStore{svc: chSvc},
		Factory: &botTGAdapterFactory{
			facade:   facade,
			cards:    mencardpersist.NewGormCardRepository(gdb),
			blob:     blobStore,
			users:    botIdentityResolver{users: userRepo},
			registry: notifyRegistry,
			caps:     caps,
		},
		Interval: 5 * time.Second,
	}
	go func() {
		if err := assembler.Run(ctx); err != nil && ctx.Err() == nil {
			slog.Error("channel assembler stopped", "err", err)
		}
	}()

	_ = bus.Subscribe(ctx, sharedkernel.TopicTaskCreated, func(ctx context.Context, msg queue.Message) error {
		var ev sharedkernel.TaskCreated
		if err := json.Unmarshal(msg.Payload, &ev); err != nil {
			return err
		}
		return orch.OnTaskCreated(ctx, ev)
	})
	dispatchHandler := func(ctx context.Context, msg queue.Message) error {
		var cmd sharedkernel.DispatchCommand
		if err := json.Unmarshal(msg.Payload, &cmd); err != nil {
			return err
		}
		return worker.HandleDispatch(ctx, cmd)
	}
	var dispatchSubs queue.SubscriptionSet
	ensureDispatchSubs := func(instances []edge.Instance) {
		for _, inst := range instances {
			topic := inst.DispatchTopic
			if topic == "" {
				topic = sharedkernel.TopicDispatch(inst.ID)
			}
			if err := dispatchSubs.Ensure(ctx, bus, topic, dispatchHandler); err != nil {
				slog.Warn("dispatch subscribe", "topic", topic, "err", err)
			}
		}
	}
	if cfg.RuntimeMode == botconfig.RuntimeModeAllinone {
		pool.SetAfterRefresh(ensureDispatchSubs)
		ensureDispatchSubs(pool.List())
	} else {
		slog.Info("split mode: bot does not subscribe production dispatch topics")
	}
	_ = bus.Subscribe(ctx, sharedkernel.TopicTaskStatus, func(ctx context.Context, msg queue.Message) error {
		var ev sharedkernel.TaskStatusEvent
		if err := json.Unmarshal(msg.Payload, &ev); err != nil {
			return err
		}
		return orch.OnStatus(ctx, ev)
	})

	addr := cfg.HTTPAddr
	// Instance management HTTP moved to admin-api (:8081).
	srv := &http.Server{Addr: addr, Handler: botserver.NewHandler(), ReadHeaderTimeout: 5 * time.Second}
	go func() {
		slog.Info("bot http listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("http server failed", "err", err)
		}
	}()

	probeEvery := 30 * time.Second
	if cfg.HealthProbeInterval != "" {
		if d, err := time.ParseDuration(cfg.HealthProbeInterval); err == nil && d > 0 {
			probeEvery = d
		}
	}
	go func() {
		t := time.NewTicker(probeEvery)
		defer t.Stop()
		if cfg.RuntimeMode == botconfig.RuntimeModeAllinone {
			tickPool(ctx, pool)
		}
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				if cfg.RuntimeMode == botconfig.RuntimeModeSplit {
					// split: Edge heartbeat is presence; skip cloud→Comfy Probe.
					_ = pool.Refresh(ctx)
					continue
				}
				tickPool(ctx, pool)
			}
		}
	}()

	go func() {
		t := time.NewTicker(30 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				orch.Storm.ResetTick()
				if err := orch.SchedulePending(ctx, 32); err != nil {
					slog.Warn("schedule pending", "err", err)
				}
				if err := orch.ReconcileStale(ctx, 2*time.Minute, 16); err != nil {
					slog.Warn("reconcile stale", "err", err)
				}
			}
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	_ = bus.Close()
	slog.Info("bot stopped")
	return nil
}

func seedCasesDir(ctx context.Context, repo catalogdomain.Repository, dir string) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		if err := seedCaseFile(ctx, repo, filepath.Join(dir, e.Name())); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

func seedCaseFile(ctx context.Context, repo catalogdomain.Repository, path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var doc catalogdomain.CaseDocument
	if err := json.Unmarshal(raw, &doc); err != nil {
		return err
	}
	c := &catalogdomain.Case{Document: doc, Enabled: true}
	if _, err := repo.Get(ctx, doc.ID); err == nil {
		return repo.Save(ctx, c)
	}
	return repo.Create(ctx, c)
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
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

func openBlobStore(cfg botconfig.Config, dataDir string) (blob.Store, error) {
	blobRoot := cfg.BlobRoot
	if blobRoot == "" {
		blobRoot = filepath.Join(dataDir, "blob")
	}
	return factory.NewFromConfig(cfg, blobRoot)
}

func openQueueBus(cfg botconfig.Config) (queue.Bus, *goredis.Client, error) {
	switch strings.TrimSpace(cfg.Queue.Driver) {
	case botconfig.QueueDriverRedis:
		rdb := goredis.NewClient(&goredis.Options{Addr: envOr("REDIS_ADDR", "127.0.0.1:6379")})
		if err := rdb.Ping(context.Background()).Err(); err != nil {
			_ = rdb.Close()
			return nil, nil, fmt.Errorf("redis ping: %w", err)
		}
		bus, err := queueredis.New(queueredis.Options{
			Client:        rdb,
			ConsumerGroup: envOr("QUEUE_GROUP", "bot"),
			ConsumerName:  envOr("QUEUE_CONSUMER", "bot"),
		})
		if err != nil {
			_ = rdb.Close()
			return nil, nil, err
		}
		return bus, rdb, nil
	default:
		return memory.New(), nil, nil
	}
}
