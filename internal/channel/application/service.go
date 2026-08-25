package application

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/mr9esx/comfyui_tgbot/internal/channel/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/notify"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
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
	Store Repository
	Key   []byte
	Notify notify.Publisher
	// DeleteWithCleanup deletes the channel row, terminates its active sessions
	// and removes channel-scoped menu/card rows in one transaction; returns
	// chats to notify after commit.
	DeleteWithCleanup func(ctx context.Context, channelID string) ([]sharedkernel.ChatID, error)
	now               func() time.Time
}

func (s *Service) nowFn() func() time.Time {
	if s.now != nil {
		return s.now
	}
	return time.Now
}

func (s *Service) Create(ctx context.Context, id string, platform domain.Platform, name, token string) (domain.Channel, error) {
	if !domain.ValidPlatform(platform) {
		return domain.Channel{}, fmt.Errorf("channel: unsupported platform %q", platform)
	}
	if id == "" || name == "" || token == "" {
		return domain.Channel{}, fmt.Errorf("channel: id/name/token required")
	}
	ct, err := domain.EncryptCredential(s.Key, domain.Credential{BotToken: token})
	if err != nil {
		return domain.Channel{}, err
	}
	now := s.nowFn()().UTC()
	ch := domain.Channel{
		ID: id, Platform: string(platform), Name: name,
		CredentialCiphertext: ct, Enabled: true, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.Store.Create(ctx, ch); err != nil {
		return domain.Channel{}, err
	}
	return ch, nil
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
			ErrorMsg: "该渠道已被管理员删除，当前会话已结束。",
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
