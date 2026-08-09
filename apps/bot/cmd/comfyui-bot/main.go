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

	"github.com/go-telegram/bot"
	"github.com/google/uuid"

	botserver "github.com/mr9esx/comfyui_tgbot/apps/bot/internal/server"
	catalogdomain "github.com/mr9esx/comfyui_tgbot/internal/catalog/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/catalog/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/catalog/infrastructure/validation"
	"github.com/mr9esx/comfyui_tgbot/internal/channel/tg"
	"github.com/mr9esx/comfyui_tgbot/internal/channel/tg/notifybridge"
	convdomain "github.com/mr9esx/comfyui_tgbot/internal/conversation/domain"
	convpersist "github.com/mr9esx/comfyui_tgbot/internal/conversation/infrastructure/persistence"
	identitypersist "github.com/mr9esx/comfyui_tgbot/internal/identity/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/packaging/botapp"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/appboot"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/blob/localfs"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/botconfig"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/instance"
	instpersist "github.com/mr9esx/comfyui_tgbot/internal/platform/instance/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/queue"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/queue/memory"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/application/orchestrator"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/actuator"
	taskpersist "github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
	tgmenuapp "github.com/mr9esx/comfyui_tgbot/internal/tgmenu/application"
	tgmenudomain "github.com/mr9esx/comfyui_tgbot/internal/tgmenu/domain"
	tgmenupersist "github.com/mr9esx/comfyui_tgbot/internal/tgmenu/infrastructure/persistence"
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

	dataDir := envOr("DATA_DIR", "data")
	_ = os.MkdirAll(dataDir, 0o755)

	dsn := resolveDSN(cfg.DatabaseDSN)
	gdb, cleanup, err := appboot.Bootstrap(ctx, appboot.Options{
		DSN:              dsn,
		MigrateInstances: true,
		Models: []any{
			&persistence.CaseRow{},
			&identitypersist.UserRow{},
			&convpersist.SessionRow{},
			&taskpersist.TaskRow{},
			&tgmenupersist.MenuRow{},
		},
	})
	if err != nil {
		return err
	}
	defer func() { _ = cleanup() }()

	caseRepo := persistence.NewGormRepository(gdb)
	userRepo := identitypersist.NewUserRepository(gdb)
	menuStore := tgmenupersist.NewGormRepository(gdb)
	menuSvc := &tgmenuapp.Service{
		Store: menuStore,
		Cases: tgmenuapp.CatalogCaseChecker{Repo: caseRepo},
	}
	menuReader := tgMenuReader{svc: menuSvc}
	if n, err := seedCasesDir(ctx, caseRepo, cfg.CaseSeedDir); err != nil {
		slog.Warn("seed cases", "err", err)
	} else {
		slog.Info("seed cases loaded", "count", n)
	}

	blobRoot := cfg.BlobRoot
	if blobRoot == "" {
		blobRoot = filepath.Join(dataDir, "blob")
	}
	blobStore, err := localfs.New(blobRoot)
	if err != nil {
		return err
	}

	bus := memory.New()
	tasks := taskpersist.NewTaskRepository(gdb)
	sessRepo := convpersist.NewSessionRepository(gdb)
	sessSvc := convdomain.NewService(sessRepo, func() sharedkernel.SessionID {
		return sharedkernel.SessionID(uuid.NewString())
	}, nil)

	instRepo := instpersist.NewInstanceRepository(gdb)
	seedCfg := instance.SeedConfig{
		DefaultInstanceID: cfg.DefaultInstanceID,
		ComfyUIBaseURL:    cfg.ComfyUIBaseURL,
		ComfyMock:         cfg.ComfyMock,
	}
	for _, s := range cfg.ComfyInstances {
		seedCfg.ComfyInstances = append(seedCfg.ComfyInstances, instance.SeedInstance{
			ID:           s.ID,
			BaseURL:      s.BaseURL,
			Enabled:      s.Enabled,
			Capabilities: s.Capabilities,
		})
	}
	if n, err := instance.SeedFromConfig(ctx, instRepo, seedCfg); err != nil {
		return err
	} else {
		slog.Info("comfy instances seeded", "count", n)
	}

	pool := instance.NewPool(instRepo, instance.PoolOptions{Mock: cfg.ComfyMock})
	if err := pool.Refresh(ctx); err != nil {
		return err
	}

	instID := sharedkernel.InstanceID(cfg.DefaultInstanceID)
	comfy, err := pool.Client(instID)
	if err != nil {
		// Fall back to first healthy/enabled instance when default id is absent.
		healthy, listErr := pool.ListHealthy(ctx, instance.CapabilityFilter{})
		if listErr != nil || len(healthy) == 0 {
			return fmt.Errorf("comfy client for %s: %w", instID, err)
		}
		instID = healthy[0].ID
		comfy, err = pool.Client(instID)
		if err != nil {
			return err
		}
	}
	slog.Info("comfyui client ready", "mock", cfg.ComfyMock, "instance_id", instID)

	tgAdapter := tg.New(nil, nil)
	notifyPub := &notifybridge.Publisher{Adapter: tgAdapter}
	orch := orchestrator.New(tasks, pool, bus, notifyPub)
	orch.Sessions = sessRepo

	snap := &actuator.CaseSnapshot{
		Tasks:    tasks,
		Cases:    caseRepo,
		Blob:     blobStore,
		Uploader: comfy,
	}
	worker := &actuator.Worker{
		InstanceID:    instID,
		Comfy:         comfy,
		ResolveClient: pool.Client,
		Blob:          blobStore,
		Status:        bus,
		Workflows:     snap,
	}
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

	var messenger tg.Messenger = logMessenger{}
	token := cfg.TelegramBotToken
	var tgBot *bot.Bot
	if token != "" {
		tgBot, err = bot.New(token)
		if err != nil {
			return err
		}
		messenger = &tg.BotMessenger{Bot: tgBot, Blob: blobStore, Menu: menuReader}
	}
	*tgAdapter = *tg.New(facade, messenger)
	tgAdapter.Users = userRepo
	tgAdapter.Menu = menuReader

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
	ensureDispatchSubs := func(instances []instance.Instance) {
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
	pool.SetAfterRefresh(ensureDispatchSubs)
	ensureDispatchSubs(pool.List())
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

	if tgBot != nil {
		tg.RegisterHandlers(tgBot, tgAdapter)
		go tgBot.Start(ctx)
		slog.Info("telegram bot started")
	} else {
		slog.Info("TG_BOT_TOKEN empty; telegram polling disabled")
	}

	probeEvery := 30 * time.Second
	if cfg.HealthProbeInterval != "" {
		if d, err := time.ParseDuration(cfg.HealthProbeInterval); err == nil && d > 0 {
			probeEvery = d
		}
	}
	go func() {
		t := time.NewTicker(probeEvery)
		defer t.Stop()
		tickPool(ctx, pool)
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
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

type logMessenger struct{}

func (logMessenger) SendText(_ context.Context, chatID int64, text string) error {
	slog.Info("tg out text", "chat_id", chatID, "text", text)
	return nil
}
func (logMessenger) SendMenu(_ context.Context, chatID int64, text string) error {
	slog.Info("tg out menu", "chat_id", chatID, "text", text)
	return nil
}
func (logMessenger) SendInline(_ context.Context, chatID int64, text string, rows [][]tg.InlineButton) error {
	slog.Info("tg out inline", "chat_id", chatID, "text", text, "rows", len(rows))
	return nil
}
func (logMessenger) SendPhoto(_ context.Context, chatID int64, ref sharedkernel.BlobRef, caption string) error {
	slog.Info("tg out photo", "chat_id", chatID, "blob", ref.Key, "caption", caption)
	return nil
}
func (logMessenger) SendPhotoURL(_ context.Context, chatID int64, imageURL, caption string) error {
	slog.Info("tg out photo_url", "chat_id", chatID, "url", imageURL, "caption", caption)
	return nil
}
func (logMessenger) AnswerCallback(_ context.Context, callbackID, text string) error {
	slog.Info("tg answer callback", "id", callbackID, "text", text)
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

type tgMenuReader struct {
	svc *tgmenuapp.Service
}

func (m tgMenuReader) GetMenu(ctx context.Context) (tgmenudomain.MenuDocument, error) {
	return m.svc.Get(ctx)
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
