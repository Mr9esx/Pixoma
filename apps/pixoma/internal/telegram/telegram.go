package telegram

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/go-telegram/bot"
	"github.com/google/uuid"

	catalogdomain "github.com/Mr9esx/Pixoma/internal/cases/domain"
	"github.com/Mr9esx/Pixoma/internal/cases/infrastructure/validation"
	channelapp "github.com/Mr9esx/Pixoma/internal/channels/application"
	"github.com/Mr9esx/Pixoma/internal/channels/application/capability"
	channeldomain "github.com/Mr9esx/Pixoma/internal/channels/domain"
	templates "github.com/Mr9esx/Pixoma/internal/channels/domain/templates"
	"github.com/Mr9esx/Pixoma/internal/channels/tg"
	mcdomain "github.com/Mr9esx/Pixoma/internal/menus/domain"
	mencardpersist "github.com/Mr9esx/Pixoma/internal/menus/infrastructure/persistence"
	"github.com/Mr9esx/Pixoma/internal/packaging/botapp"
	"github.com/Mr9esx/Pixoma/internal/platform/blob"
	"github.com/Mr9esx/Pixoma/internal/platform/notify"
	"github.com/Mr9esx/Pixoma/internal/platform/queue"
	convdomain "github.com/Mr9esx/Pixoma/internal/sessions/domain"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
	runtimedomain "github.com/Mr9esx/Pixoma/internal/tasks/domain"
	identitydomain "github.com/Mr9esx/Pixoma/internal/users/domain"
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
	// Probe is shared by the 30s ticker and POST /api/v1/channels/probe.
	Probe *channelapp.ReachabilityProbe
	done  chan struct{}
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
	Texts templates.Renderer
}

// StartBotRuntime builds the facade, channel assembler, background
// reachability probe, and notify router.
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

	registry := &notifyRegistry{handlers: map[string]channelapp.NotifyHandler{}}
	router := &channelapp.NotifyRouter{
		HandlerByChannel: registry.lookup,
	}
	factory := &tgChannelFactory{
		facade:   facade,
		deps:     deps,
		registry: registry,
	}
	caps := newCapabilityRegistry(facade, deps.Texts, deps.Users)
	factory.caps = caps
	assembler := &channelapp.Assembler{
		Store:    &channelSnapshotStore{svc: deps.Channels},
		Factory:  factory,
		Interval: channelWatchInterval,
	}
	probe := &channelapp.ReachabilityProbe{Svc: deps.Channels}
	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		if err := assembler.Run(ctx); err != nil && ctx.Err() == nil {
			slog.Error("channel assembler stopped", "err", err)
		}
	}()
	go func() {
		defer wg.Done()
		if err := probe.Run(ctx); err != nil && ctx.Err() == nil {
			slog.Error("channel reachability probe stopped", "err", err)
		}
	}()
	go func() {
		wg.Wait()
		close(done)
	}()
	botRT := &BotRuntime{
		Facade:       facade,
		Notify:       router,
		Stop:         assembler.StopAll,
		Capabilities: caps,
		Probe:        probe,
		done:         done,
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

const (
	channelWatchInterval = 5 * time.Second
	telegramPollTimeout  = 15 * time.Second
)

// telegramHTTPClient is the Bot API client. Timeout stays 0: getUpdates
// long-polls for pollTimeout-1s, and a Client.Timeout at pollTimeout races
// that wait ("awaiting headers") even when the proxy already works.
func telegramHTTPClient() *http.Client {
	return &http.Client{}
}

// Wait blocks until the assembler and reachability probe have exited after
// ctx cancel. Used by pixoma reload so the next run does not share SQLite /
// getUpdates with the previous one.
func (rt *BotRuntime) Wait(ctx context.Context) error {
	if rt == nil || rt.done == nil {
		return nil
	}
	select {
	case <-rt.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

type channelSnapshotStore struct {
	svc *channelapp.Service
}

func (s *channelSnapshotStore) ListChannels(ctx context.Context) ([]channelapp.ChannelSnapshot, error) {
	chs, err := s.svc.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]channelapp.ChannelSnapshot, 0, len(chs))
	for _, ch := range chs {
		cred, err := channeldomain.DecryptCredential(s.svc.Key, ch.CredentialCiphertext)
		if err != nil {
			slog.Error("channel credential decrypt", "err", err, "channel", ch.ID)
			continue
		}
		sum := sha256.Sum256([]byte(cred.BotToken))
		out = append(out, channelapp.ChannelSnapshot{
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
	handlers map[string]channelapp.NotifyHandler
}

func (r *notifyRegistry) set(channelID string, h channelapp.NotifyHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[channelID] = h
}

func (r *notifyRegistry) unset(channelID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.handlers, channelID)
}

func (r *notifyRegistry) lookup(channelID string) (channelapp.NotifyHandler, bool) {
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

func newCapabilityRegistry(facade *botapp.Facade, texts templates.Renderer, users identitydomain.Repository) *capability.Registry {
	r := capability.NewRegistry()
	_ = r.Register(capability.OpenCase{App: facade, Texts: texts, Users: users})
	_ = r.Register(capability.ListTasks{Tasks: facade.Tasks, Cases: facade.Cases, Texts: texts})
	return r
}

func (f *tgChannelFactory) Create(snap channelapp.ChannelSnapshot) (channelapp.Adapter, error) {
	menuReader := channelMenuReader{cards: f.deps.MenuCards, channelID: snap.ID}
	// Skip getMe here: bot.New's default 5s probe would hold assembler
	// restart under Telegram RTT. Reachability belongs to ReachabilityProbe.
	botInst, err := newTelegramBot(snap.Credential, bot.WithSkipGetMe(), bot.WithHTTPClient(telegramPollTimeout, telegramHTTPClient()))
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
	done      chan struct{}
}

func (w *tgBotWrapper) Start(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	bctx, cancel := context.WithCancel(ctx)
	w.cancel = cancel
	w.done = make(chan struct{})
	go func() {
		defer close(w.done)
		w.bot.Start(bctx)
	}()
	w.registry.set(w.channelID, w.adapter)
	slog.Info("telegram bot started", "channel", w.channelID)
	return nil
}

func (w *tgBotWrapper) Stop(ctx context.Context) error {
	w.mu.Lock()
	cancel := w.cancel
	done := w.done
	w.cancel = nil
	w.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	w.registry.unset(w.channelID)
	if done == nil {
		return nil
	}
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

type channelMenuReader struct {
	cards     mencardpersist.CardRepository
	channelID string
}

func (r channelMenuReader) GetTree(ctx context.Context) (mcdomain.MenuTree, error) {
	return r.cards.GetTree(ctx, r.channelID)
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
