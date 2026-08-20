package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"sync"

	"github.com/go-telegram/bot"

	channelapp "github.com/mr9esx/comfyui_tgbot/internal/channel/application"
	"github.com/mr9esx/comfyui_tgbot/internal/channel/capability"
	channeldomain "github.com/mr9esx/comfyui_tgbot/internal/channel/domain"
	channelruntime "github.com/mr9esx/comfyui_tgbot/internal/channel/runtime"
	"github.com/mr9esx/comfyui_tgbot/internal/channel/tg"
	identitydomain "github.com/mr9esx/comfyui_tgbot/internal/identity/domain"
	mcdomain "github.com/mr9esx/comfyui_tgbot/internal/menucard/domain"
	mencardpersist "github.com/mr9esx/comfyui_tgbot/internal/menucard/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/packaging/botapp"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/blob"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

type botChannelSnapshotStore struct {
	svc *channelapp.Service
}

func (s *botChannelSnapshotStore) ListChannels(ctx context.Context) ([]channelruntime.ChannelSnapshot, error) {
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

type botNotifyRegistry struct {
	mu       sync.Mutex
	handlers map[string]channelruntime.NotifyHandler
}

func (r *botNotifyRegistry) set(channelID string, h channelruntime.NotifyHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[channelID] = h
}

func (r *botNotifyRegistry) lookup(channelID string) (channelruntime.NotifyHandler, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	h, ok := r.handlers[channelID]
	return h, ok
}

type botTGAdapterFactory struct {
	facade   *botapp.Facade
	cards    mencardpersist.CardRepository
	blob     blob.Store
	users    botIdentityResolver
	registry *botNotifyRegistry
	caps     *capability.Registry
}

func (f *botTGAdapterFactory) Create(snap channelruntime.ChannelSnapshot) (channelruntime.Adapter, error) {
	botInst, err := bot.New(snap.Credential)
	if err != nil {
		return nil, err
	}
	reader := botMenuReader{cards: f.cards, channelID: snap.ID}
	messenger := &tg.BotMessenger{Bot: botInst, Blob: f.blob, Menu: reader}
	adapter := tg.New(messenger)
	adapter.Registry = f.caps
	adapter.Blob = f.blob
	adapter.Media = tg.NewMediaBridge(botInst)
	adapter.Users = f.users
	adapter.Menu = reader
	adapter.ChannelID = snap.ID
	tg.RegisterHandlers(botInst, adapter)
	return &botTGWrapper{bot: botInst, adapter: adapter, registry: f.registry, channelID: snap.ID}, nil
}

type botMenuReader struct {
	cards     mencardpersist.CardRepository
	channelID string
}

func (r botMenuReader) GetMenu(ctx context.Context) (mcdomain.Menu, error) {
	return r.cards.GetMenu(ctx, r.channelID)
}

type botIdentityResolver struct {
	users identitydomain.Repository
}

func (r botIdentityResolver) Resolve(ctx context.Context, _ sharedkernel.ChannelAddr, profile identitydomain.UpsertFrom) (string, error) {
	u, err := r.users.UpsertByChannelExternal(ctx, profile)
	if err != nil {
		return "", err
	}
	return u.ID, nil
}

type botTGWrapper struct {
	bot       *bot.Bot
	adapter   *tg.Adapter
	registry  *botNotifyRegistry
	channelID string
	mu        sync.Mutex
	cancel    context.CancelFunc
}

func (w *botTGWrapper) Start(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	bctx, cancel := context.WithCancel(ctx)
	w.cancel = cancel
	go w.bot.Start(bctx)
	w.registry.set(w.channelID, w.adapter)
	slog.Info("telegram bot started", "channel", w.channelID)
	return nil
}

func (w *botTGWrapper) Stop(context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.cancel != nil {
		w.cancel()
		w.cancel = nil
	}
	return nil
}

var _ = sharedkernel.ChatID("")
