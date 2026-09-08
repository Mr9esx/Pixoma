package application

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/Mr9esx/Pixoma/internal/channels/domain"
	"github.com/Mr9esx/Pixoma/internal/platform/notify"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
)

// Repository is the channel persistence port.
type Repository interface {
	Create(ctx context.Context, ch domain.Channel) error
	Get(ctx context.Context, id string) (domain.Channel, error)
	List(ctx context.Context) ([]domain.Channel, error)
	Update(ctx context.Context, ch domain.Channel) error
	Delete(ctx context.Context, id string) error
}

// Service orchestrates channel lifecycle and credential encryption.
type Service struct {
	Store  Repository
	Key    []byte
	Notify notify.Publisher
	// DeleteWithCleanup deletes the channel row, terminates its active sessions
	// and removes channel-scoped menu/card rows in one transaction; returns
	// chats to notify after commit.
	DeleteWithCleanup func(ctx context.Context, channelID string) ([]sharedkernel.ChatID, error)
	// CheckTelegram overrides the default getMe probe (tests).
	CheckTelegram func(ctx context.Context, token string) (ReachabilityResult, error)
	// FetchTelegram overrides the default getMe identity fetch (tests).
	FetchTelegram func(ctx context.Context, token string) (json.RawMessage, error)
	// AdapterStatus reports the background adapter state for a channel
	// (state, last error, found); wired from the channel runtime in main.
	AdapterStatus func(ctx context.Context, id string) (state string, lastErr string, found bool)
	now           func() time.Time
}

func (s *Service) nowFn() func() time.Time {
	if s.now != nil {
		return s.now
	}
	return time.Now
}

func (s *Service) Create(ctx context.Context, id string, platform domain.Platform, name, token string, extraInfo string) (domain.Channel, error) {
	if !domain.ValidPlatform(platform) {
		return domain.Channel{}, fmt.Errorf("channel: unsupported platform %q", platform)
	}
	if id == "" || token == "" {
		return domain.Channel{}, fmt.Errorf("channel: id/token required")
	}
	if extraInfo == "" && platform == domain.PlatformTelegram {
		if info, err := s.FetchBotInfo(ctx, token); err == nil {
			extraInfo = string(info)
			if strings.TrimSpace(name) == "" {
				name = telegramBotDisplayName(info)
			}
		}
	} else if extraInfo != "" && !json.Valid([]byte(extraInfo)) {
		return domain.Channel{}, fmt.Errorf("channel: invalid extra_info json")
	}
	if strings.TrimSpace(name) == "" {
		return domain.Channel{}, fmt.Errorf("channel: name required")
	}
	ct, err := domain.EncryptCredential(s.Key, domain.Credential{BotToken: token})
	if err != nil {
		return domain.Channel{}, err
	}
	now := s.nowFn()().UTC()
	ch := domain.Channel{
		ID: id, Platform: string(platform), Name: name, ExtraInfo: extraInfo,
		CredentialCiphertext: ct, Enabled: true, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.Store.Create(ctx, ch); err != nil {
		return domain.Channel{}, err
	}
	return ch, nil
}

// FetchBotInfo probes the Telegram Bot API with a raw token and returns the
// full identity object (JSON) reported by getMe.
func (s *Service) FetchBotInfo(ctx context.Context, token string) (json.RawMessage, error) {
	if token == "" {
		return nil, fmt.Errorf("channel: token required")
	}
	fetch := s.FetchTelegram
	if fetch == nil {
		fetch = fetchTelegramBotInfo
	}
	return fetch(ctx, token)
}

func (s *Service) Get(ctx context.Context, id string) (domain.Channel, error) {
	return s.Store.Get(ctx, id)
}

// Masked returns the channel with its credential rendered as a masked token.
func (s *Service) Masked(ctx context.Context, id string) (string, error) {
	ch, err := s.Store.Get(ctx, id)
	if err != nil {
		return "", err
	}
	cred, err := domain.DecryptCredential(s.Key, ch.CredentialCiphertext)
	if err != nil {
		return "", err
	}
	return domain.MaskedToken(cred.BotToken), nil
}

// CheckReachability probes the Telegram Bot API once with the channel token
// and classifies the result (ok / network / auth / other).
func (s *Service) CheckReachability(ctx context.Context, id string) (ReachabilityResult, error) {
	ch, err := s.Store.Get(ctx, id)
	if err != nil {
		return ReachabilityResult{}, err
	}
	cred, err := domain.DecryptCredential(s.Key, ch.CredentialCiphertext)
	if err != nil {
		return ReachabilityResult{}, err
	}
	probe := s.CheckTelegram
	if probe == nil {
		probe = checkTelegramReachability
	}
	res, err := probe(ctx, cred.BotToken)
	if err != nil {
		return ReachabilityResult{}, err
	}
	now := s.nowFn()().UTC()
	ch.LastCheckKind = string(res.Kind)
	ch.LastCheckMessage = res.Message
	ch.LastCheckAt = &now
	ch.UpdatedAt = now
	if err := s.Store.Update(ctx, ch); err != nil {
		return ReachabilityResult{}, fmt.Errorf("channel: persist last check: %w", err)
	}
	return res, nil
}

func (s *Service) List(ctx context.Context) ([]domain.Channel, error) {
	return s.Store.List(ctx)
}

func (s *Service) Update(ctx context.Context, id, name string, token *string) (domain.Channel, error) {
	ch, err := s.Store.Get(ctx, id)
	if err != nil {
		return domain.Channel{}, err
	}
	if name != "" {
		ch.Name = name
	}
	if token != nil && *token != "" {
		ct, err := domain.EncryptCredential(s.Key, domain.Credential{BotToken: *token})
		if err != nil {
			return domain.Channel{}, err
		}
		ch.CredentialCiphertext = ct
		ch.LastCheckKind = ""
		ch.LastCheckMessage = ""
		ch.LastCheckAt = nil
	}
	ch.UpdatedAt = s.nowFn()().UTC()
	if err := s.Store.Update(ctx, ch); err != nil {
		return domain.Channel{}, err
	}
	return ch, nil
}

func (s *Service) Disable(ctx context.Context, id string) error {
	ch, err := s.Store.Get(ctx, id)
	if err != nil {
		return err
	}
	ch.Enabled = false
	ch.UpdatedAt = s.nowFn()().UTC()
	return s.Store.Update(ctx, ch)
}

func (s *Service) Enable(ctx context.Context, id string) error {
	ch, err := s.Store.Get(ctx, id)
	if err != nil {
		return err
	}
	ch.Enabled = true
	ch.UpdatedAt = s.nowFn()().UTC()
	return s.Store.Update(ctx, ch)
}

// Delete removes a channel directly. When DeleteWithCleanup is wired, active
// sessions are terminated and menu/card rows are removed in the same
// transaction; affected users are notified best-effort after commit.
func (s *Service) Delete(ctx context.Context, id string) error {
	if _, err := s.Store.Get(ctx, id); err != nil {
		return err
	}
	if s.DeleteWithCleanup == nil {
		return s.Store.Delete(ctx, id)
	}
	chats, err := s.DeleteWithCleanup(ctx, id)
	if err != nil {
		return err
	}
	for _, chat := range chats {
		n := sharedkernel.UserNotify{
			ChatID:   chat,
			Kind:     "session_terminated",
			ErrorMsg: "该消息平台已被管理员删除，当前会话已结束。",
		}
		if s.Notify == nil {
			continue
		}
		if err := s.Notify.Publish(ctx, n); err != nil {
			slog.Warn("channel delete: session notify failed", "chat", chat, "err", err)
		}
	}
	return nil
}
