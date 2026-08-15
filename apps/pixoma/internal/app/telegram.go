package app

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/google/uuid"

	catalogdomain "github.com/mr9esx/comfyui_tgbot/internal/catalog/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/catalog/infrastructure/validation"
	"github.com/mr9esx/comfyui_tgbot/internal/channel/tg"
	"github.com/mr9esx/comfyui_tgbot/internal/channel/tg/notifybridge"
	convdomain "github.com/mr9esx/comfyui_tgbot/internal/conversation/domain"
	identitydomain "github.com/mr9esx/comfyui_tgbot/internal/identity/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/packaging/botapp"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/blob"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/notify"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/queue"
	runtimedomain "github.com/mr9esx/comfyui_tgbot/internal/runtime/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
	tgmenuapp "github.com/mr9esx/comfyui_tgbot/internal/tgmenu/application"
	tgmenudomain "github.com/mr9esx/comfyui_tgbot/internal/tgmenu/domain"
)

// BotRuntime is the in-process Telegram + notify wiring for pixoma.
type BotRuntime struct {
	Facade *botapp.Facade
	Notify notify.Publisher
}

// BotDeps is everything needed to accept Telegram work and notify users.
type BotDeps struct {
	Token        string
	Cases        catalogdomain.Repository
	Sessions     *convdomain.Service
	SessionStore convdomain.Repository
	Tasks        runtimedomain.TaskRepository
	Users        identitydomain.Repository
	Menu         *tgmenuapp.Service
	Blob         blob.Store
	Bus          queue.Publisher
}

// StartBotRuntime builds the facade, notify publisher, and optional Telegram polling.
func StartBotRuntime(ctx context.Context, deps BotDeps) (*BotRuntime, error) {
	facade := &botapp.Facade{
		Cases:        deps.Cases,
		Validator:    validation.New(),
		Sessions:     deps.Sessions,
		SessionStore: deps.SessionStore,
		Tasks:        deps.Tasks,
		Blob:         deps.Blob,
		Publisher:    deps.Bus,
		NewTaskID: func() sharedkernel.TaskID {
			return sharedkernel.TaskID(uuid.NewString())
		},
	}
	adapter := tg.New(nil, nil)
	menu := menuReader{svc: deps.Menu}
	var messenger tg.Messenger = logMessenger{}
	token := strings.TrimSpace(deps.Token)
	if token != "" {
		b, err := bot.New(token)
		if err != nil {
			return nil, err
		}
		messenger = &tg.BotMessenger{Bot: b, Blob: deps.Blob, Menu: menu}
		*adapter = *tg.New(facade, messenger)
		adapter.Users = deps.Users
		adapter.Menu = menu
		tg.RegisterHandlers(b, adapter)
		go b.Start(ctx)
		slog.Info("telegram bot started")
	} else {
		*adapter = *tg.New(facade, messenger)
		adapter.Users = deps.Users
		adapter.Menu = menu
		slog.Info("telegram bot token empty; polling disabled")
	}
	return &BotRuntime{
		Facade: facade,
		Notify: &notifybridge.Publisher{Adapter: adapter},
	}, nil
}

type menuReader struct {
	svc *tgmenuapp.Service
}

func (m menuReader) GetMenu(ctx context.Context) (tgmenudomain.MenuTree, error) {
	if m.svc == nil {
		return tgmenudomain.MenuTree{}, nil
	}
	return m.svc.Get(ctx)
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

// SeedCasesDir loads JSON case files if the directory exists.
func SeedCasesDir(ctx context.Context, repo catalogdomain.Repository, dir string) (int, error) {
	if strings.TrimSpace(dir) == "" {
		return 0, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
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
