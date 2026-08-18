package application

import (
	"context"
	"fmt"
	"time"

	"github.com/mr9esx/comfyui_tgbot/internal/channel/domain"
)

// Repository is the channel persistence port.
type Repository interface {
	Create(ctx context.Context, ch domain.Channel) error
	Get(ctx context.Context, id string) (domain.Channel, error)
	List(ctx context.Context) ([]domain.Channel, error)
	Update(ctx context.Context, ch domain.Channel) error
	Delete(ctx context.Context, id string) error
}

// HasActiveRefsFunc reports whether a channel still has active sessions/tasks.
type HasActiveRefsFunc func(ctx context.Context, channelID string) (bool, error)

// Service orchestrates channel lifecycle and credential encryption.
type Service struct {
	Store         Repository
	Key           []byte
	HasActiveRefs HasActiveRefsFunc
	now           func() time.Time
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

// Delete removes a channel only when disabled and free of active references.
func (s *Service) Delete(ctx context.Context, id string) error {
	ch, err := s.Store.Get(ctx, id)
	if err != nil {
		return err
	}
	if ch.Enabled {
		return domain.ErrDeleteRestricted
	}
	if s.HasActiveRefs != nil {
		active, err := s.HasActiveRefs(ctx, id)
		if err != nil {
			return err
		}
		if active {
			return domain.ErrDeleteRestricted
		}
	}
	return s.Store.Delete(ctx, id)
}
