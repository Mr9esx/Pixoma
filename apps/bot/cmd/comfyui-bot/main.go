package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-telegram/bot"
	"github.com/google/uuid"

	catalogdomain "github.com/mr9esx/comfyui_tgbot/internal/catalog/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/catalog/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/catalog/infrastructure/validation"
	"github.com/mr9esx/comfyui_tgbot/internal/channel/tg"
	"github.com/mr9esx/comfyui_tgbot/internal/channel/tg/notifybridge"
	convdomain "github.com/mr9esx/comfyui_tgbot/internal/conversation/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/packaging/botapp"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/blob/localfs"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/db"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/instance"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/instance/static"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/queue"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/queue/memory"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/application/orchestrator"
	runtimedomain "github.com/mr9esx/comfyui_tgbot/internal/runtime/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/actuator"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/comfyui"
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
	dataDir := envOr("DATA_DIR", "data")
	_ = os.MkdirAll(dataDir, 0o755)

	gdb, err := db.Open(db.Options{DSN: filepath.Join(dataDir, "app.db")})
	if err != nil {
		return err
	}
	if err := db.AutoMigrate(gdb, &persistence.CaseRow{}); err != nil {
		return err
	}
	caseRepo := persistence.NewGormRepository(gdb)
	if n, err := seedCasesDir(ctx, caseRepo, envOr("CASE_SEED_DIR", "configs/cases")); err != nil {
		slog.Warn("seed cases", "err", err)
	} else {
		slog.Info("seed cases loaded", "count", n)
	}

	blobStore, err := localfs.New(filepath.Join(dataDir, "blob"))
	if err != nil {
		return err
	}

	bus := memory.New()
	tasks := runtimedomain.NewMemoryTaskRepository()
	sessRepo := convdomain.NewMemoryRepository()
	sessSvc := convdomain.NewService(sessRepo, func() sharedkernel.SessionID {
		return sharedkernel.SessionID(uuid.NewString())
	}, nil)

	instID := sharedkernel.InstanceID(envOr("INSTANCE_ID", "local"))
	reg := static.New(instance.Instance{ID: instID, DispatchTopic: sharedkernel.TopicDispatch(instID)})

	tgAdapter := tg.New(nil, nil)
	notifyPub := &notifybridge.Publisher{Adapter: tgAdapter}
	orch := orchestrator.New(tasks, reg, bus, notifyPub)

	worker := &actuator.Worker{
		InstanceID: instID,
		Comfy:      &comfyui.Mock{},
		Blob:       blobStore,
		Status:     bus,
		Ledger:     actuator.NewMemoryLedger(),
		Workflows:  actuator.StaticWorkflows{},
	}
	orch.Query = &actuator.QueryAdapter{Worker: worker}

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
	token := os.Getenv("TG_BOT_TOKEN")
	var tgBot *bot.Bot
	if token != "" {
		tgBot, err = bot.New(token)
		if err != nil {
			return err
		}
		messenger = &tg.BotMessenger{Bot: tgBot, Blob: blobStore}
	}
	*tgAdapter = *tg.New(facade, messenger)

	_ = bus.Subscribe(ctx, sharedkernel.TopicTaskCreated, func(ctx context.Context, msg queue.Message) error {
		var ev sharedkernel.TaskCreated
		if err := json.Unmarshal(msg.Payload, &ev); err != nil {
			return err
		}
		return orch.OnTaskCreated(ctx, ev)
	})
	_ = bus.Subscribe(ctx, sharedkernel.TopicDispatch(instID), func(ctx context.Context, msg queue.Message) error {
		var cmd sharedkernel.DispatchCommand
		if err := json.Unmarshal(msg.Payload, &cmd); err != nil {
			return err
		}
		return worker.HandleDispatch(ctx, cmd)
	})
	_ = bus.Subscribe(ctx, sharedkernel.TopicTaskStatus, func(ctx context.Context, msg queue.Message) error {
		var ev sharedkernel.TaskStatusEvent
		if err := json.Unmarshal(msg.Payload, &ev); err != nil {
			return err
		}
		return orch.OnStatus(ctx, ev)
	})

	addr := envOr("HTTP_ADDR", ":8080")
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.RealIP, middleware.Recoverer)
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	srv := &http.Server{Addr: addr, Handler: r, ReadHeaderTimeout: 5 * time.Second}
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

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
