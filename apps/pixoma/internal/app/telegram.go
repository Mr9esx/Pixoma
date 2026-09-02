package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"sync"

	"github.com/go-telegram/bot"
	"github.com/google/uuid"

	catalogdomain "github.com/mr9esx/comfyui_tgbot/internal/catalog/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/catalog/infrastructure/validation"
	channelapp "github.com/mr9esx/comfyui_tgbot/internal/channel/application"
	"github.com/mr9esx/comfyui_tgbot/internal/channel/capability"
	channeldomain "github.com/mr9esx/comfyui_tgbot/internal/channel/domain"
	channelruntime "github.com/mr9esx/comfyui_tgbot/internal/channel/runtime"
	"github.com/mr9esx/comfyui_tgbot/internal/channel/text"
	"github.com/mr9esx/comfyui_tgbot/internal/channel/tg"
	convdomain "github.com/mr9esx/comfyui_tgbot/internal/conversation/domain"
	identitydomain "github.com/mr9esx/comfyui_tgbot/internal/identity/domain"
	mcdomain "github.com/mr9esx/comfyui_tgbot/internal/menucard/domain"
	mencardpersist "github.com/mr9esx/comfyui_tgbot/internal/menucard/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/packaging/botapp"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/blob"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/notify"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/queue"
	runtimedomain "github.com/mr9esx/comfyui_tgbot/internal/runtime/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

type telegramBot = bot.Bot

var newTelegramBot = bot.New

// BotRuntime carries the channel runtime and notify publisher for pixoma.
type BotRuntime struct {
	Facade       *botapp.Facade
	Notify       notify.Publisher
	Stop         func(ctx context.Context) error
	Capabilities *capability.Registry
	// ChannelStatus reports the background adapter state for a channel ID
	// (state, last error, found).
	ChannelStatus func(channelID string) (state string, lastErr string, found bool)
}

// BotDeps is everything needed to run channel adapters and notify users.
type BotDeps struct {
	Channels     *channelapp.Service
	Cases        catalogdomain.Repository
	Sessions     *convdomain.Service
	SessionStore convdomain.Repository
	Tasks        runtimedomain.TaskRepository
	Users        identitydomain.Repository
	MenuCards    mencardpersist.CardRepository
	Blob         blob.Store
	Bus          queue.Publisher
	// Texts resolves configurable copy templates; nil falls back to built-ins.
	Texts text.Renderer
}

// StartBotRuntime builds the facade, channel assembler, and notify router.
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

	registry := &notifyRegistry{handlers: map[string]channelruntime.NotifyHandler{}}
	router := &channelruntime.NotifyRouter{
		HandlerByChannel: registry.lookup,
	}
	factory := &tgChannelFactory{
		facade:   facade,
		deps:     deps,
		registry: registry,
	}
	caps := newCapabilityRegistry(facade, deps.Texts, deps.Users)
	factory.caps = caps
	assembler := &channelruntime.Assembler{
		Store:    &channelSnapshotStore{svc: deps.Channels},
		Factory:  factory,
		Interval: channelWatchInterval,
	}
	go func() {
		if err := assembler.Run(ctx); err != nil && ctx.Err() == nil {
			slog.Error("channel assembler stopped", "err", err)
		}
	}()
	botRT := &BotRuntime{
		Facade:       facade,
		Notify:       router,
		Stop:         assembler.StopAll,
		Capabilities: caps,
	}
	botRT.ChannelStatus = func(channelID string) (string, string, bool) {
		st, ok := assembler.Status()[channelID]
		if !ok {
			return "", "", false
		}
		lastErr := ""
		if st.LastErr != nil {
			lastErr = st.LastErr.Error()
		}
		return string(st.State), lastErr, true
	}
	return botRT, nil
}

const channelWatchInterval = 5 * 1000_000_000 // 5s

type channelSnapshotStore struct {
	svc *channelapp.Service
}

func (s *channelSnapshotStore) ListChannels(ctx context.Context) ([]channelruntime.ChannelSnapshot, error) {
	chs, err := s.svc.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]channelruntime.ChannelSnapshot, 0, len(chs))
	for _, ch := range chs {
		cred, err := channeldomain.DecryptCredential(s.svc.Key, ch.CredentialCiphertext)
		if err != nil {
			slog.Error("channel credential decrypt", "err", err, "channel", ch.ID)
			continue
		}
		sum := sha256.Sum256([]byte(cred.BotToken))
		out = append(out, channelruntime.ChannelSnapshot{
			ID:             ch.ID,
			Platform:       ch.Platform,
			Credential:     cred.BotToken,
			CredentialHash: hex.EncodeToString(sum[:]),
			Enabled:        ch.Enabled,
			UpdatedAt:      ch.UpdatedAt,
		})
	}
	return out, nil
}

type notifyRegistry struct {
	mu       sync.Mutex
	handlers map[string]channelruntime.NotifyHandler
}

func (r *notifyRegistry) set(channelID string, h channelruntime.NotifyHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[channelID] = h
}

func (r *notifyRegistry) unset(channelID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.handlers, channelID)
}

func (r *notifyRegistry) lookup(channelID string) (channelruntime.NotifyHandler, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	h, ok := r.handlers[channelID]
	return h, ok
}

type tgChannelFactory struct {
	facade   *botapp.Facade
	deps     BotDeps
	registry *notifyRegistry
	caps     *capability.Registry
}

func newCapabilityRegistry(facade *botapp.Facade, texts text.Renderer, users identitydomain.Repository) *capability.Registry {
	r := capability.NewRegistry()
	_ = r.Register(capability.OpenCase{App: facade, Texts: texts, Users: users})
	_ = r.Register(capability.ListTasks{Tasks: facade.Tasks, Cases: facade.Cases})
	return r
}

func (f *tgChannelFactory) Create(snap channelruntime.ChannelSnapshot) (channelruntime.Adapter, error) {
	menuReader := channelMenuReader{cards: f.deps.MenuCards, channelID: snap.ID}
	botInst, err := newTelegramBot(snap.Credential)
	if err != nil {
		return nil, err
	}
	messenger := &tg.BotMessenger{
		Bot:  botInst,
		Blob: f.deps.Blob,
		Menu: menuReader,
	}
	adapter := tg.New(messenger)
	adapter.Registry = f.caps
	adapter.Texts = f.deps.Texts
	adapter.Blob = f.deps.Blob
	adapter.Media = tg.NewMediaBridge(botInst)
	adapter.Users = identityResolver{users: f.deps.Users}
	adapter.Menu = menuReader
	adapter.ChannelID = snap.ID
	tg.RegisterHandlers(botInst, adapter)
	return &tgBotWrapper{
		adapter:   adapter,
		registry:  f.registry,
		channelID: snap.ID,
		bot:       botInst,
	}, nil
}

type tgBotWrapper struct {
	adapter   *tg.Adapter
	registry  *notifyRegistry
	channelID string
	bot       *telegramBot
	mu        sync.Mutex
	cancel    context.CancelFunc
}

func (w *tgBotWrapper) Start(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	bctx, cancel := context.WithCancel(ctx)
	w.cancel = cancel
	go w.bot.Start(bctx)
	w.registry.set(w.channelID, w.adapter)
	slog.Info("telegram bot started", "channel", w.channelID)
	return nil
}

func (w *tgBotWrapper) Stop(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.cancel != nil {
		w.cancel()
		w.cancel = nil
	}
	w.registry.unset(w.channelID)
	return nil
}

type channelMenuReader struct {
	cards     mencardpersist.CardRepository
	channelID string
}

func (r channelMenuReader) GetMenu(ctx context.Context) (mcdomain.Menu, error) {
	return r.cards.GetMenu(ctx, r.channelID)
}

type identityResolver struct {
	users identitydomain.Repository
}

func (r identityResolver) Resolve(ctx context.Context, _ sharedkernel.ChannelAddr, profile identitydomain.UpsertFrom) (string, error) {
	u, err := r.users.UpsertByChannelExternal(ctx, profile)
	if err != nil {
		return "", err
	}
	return u.ID, nil
}
