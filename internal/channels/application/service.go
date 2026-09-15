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
	// CheckFeishu overrides the default Feishu tenant-access probe (tests).
	CheckFeishu func(ctx context.Context, appID, appSecret string) (ReachabilityResult, error)
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
	return s.CreateWithCredential(ctx, id, platform, name, domain.Credential{BotToken: token}, extraInfo)
}

// CreateWithCredential creates a channel with a platform-specific credential.
// Telegram stores the bot token; Feishu stores App ID + App Secret; WeCom
// stores its intelligent-bot ID, Secret, and optional WSS URL.
func (s *Service) CreateWithCredential(ctx context.Context, id string, platform domain.Platform, name string, cred domain.Credential, extraInfo string) (domain.Channel, error) {
	if !domain.ValidPlatform(platform) {
		return domain.Channel{}, fmt.Errorf("channel: unsupported platform %q", platform)
	}
	if id == "" {
		return domain.Channel{}, fmt.Errorf("channel: id required")
	}
	if platform == domain.PlatformMCP && strings.TrimSpace(name) == "" {
		return domain.Channel{}, fmt.Errorf("channel: name required")
	}
	if err := validateCredential(platform, cred); err != nil {
		return domain.Channel{}, err
	}
	if extraInfo == "" && platform == domain.PlatformTelegram {
		if info, err := s.FetchBotInfo(ctx, cred.BotToken); err == nil {
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
	ct, err := domain.EncryptCredential(s.Key, cred)
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

// validateCredential enforces the per-platform credential shape.
func validateCredential(platform domain.Platform, cred domain.Credential) error {
	switch platform {
	case domain.PlatformMCP:
		return nil
	case domain.PlatformTelegram:
		if strings.TrimSpace(cred.BotToken) == "" {
			return fmt.Errorf("channel: id/credential required")
		}
		return nil
	case domain.PlatformFeishu:
		if strings.TrimSpace(cred.AppID) == "" || strings.TrimSpace(cred.AppSecret) == "" {
			return fmt.Errorf("channel: feishu app_id/app_secret required")
		}
		return nil
	case domain.PlatformWeCom:
		if strings.TrimSpace(cred.WeComBotID) == "" || strings.TrimSpace(cred.WeComSecret) == "" {
			return fmt.Errorf("channel: wecom bot_id/secret required")
		}
		return nil
	default:
		// wecom/dingtalk reserved; require a non-empty secret.
		if cred.BotToken == "" && cred.AppSecret == "" {
			return fmt.Errorf("channel: id/credential required")
		}
		return nil
	}
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
	return cred.Masked(), nil
}

// CheckReachability probes the platform identity endpoint once and classifies
// the result (ok / network / auth / other). Telegram uses getMe; Feishu probes
// tenant_access_token; WeCom reuses its running adapter state and never dials a
// second WSS connection; unprobeable platforms report ok without network calls.
func (s *Service) CheckReachability(ctx context.Context, id string) (ReachabilityResult, error) {
	ch, err := s.Store.Get(ctx, id)
	if err != nil {
		return ReachabilityResult{}, err
	}
	if ch.Platform == string(domain.PlatformMCP) {
		return ReachabilityResult{OK: true, Kind: ReachabilityOK}, nil
	}
	cred, err := domain.DecryptCredential(s.Key, ch.CredentialCiphertext)
	if err != nil {
		return ReachabilityResult{}, err
	}
	var res ReachabilityResult
	switch domain.Platform(ch.Platform) {
	case domain.PlatformWeCom:
		res = wecomAdapterReachability(ctx, s.AdapterStatus, ch.ID)
	case domain.PlatformFeishu:
		probe := s.CheckFeishu
		if probe == nil {
			probe = checkFeishuReachability
		}
		res, err = probe(ctx, cred.AppID, cred.AppSecret)
	case domain.PlatformTelegram:
		probe := s.CheckTelegram
		if probe == nil {
			probe = checkTelegramReachability
		}
		res, err = probe(ctx, cred.BotToken)
	default:
		res = ReachabilityResult{OK: true, Kind: ReachabilityOK, Message: "probe not implemented"}
	}
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
	var cred *domain.Credential
	if token != nil && *token != "" {
		cred = &domain.Credential{BotToken: *token}
	}
	return s.UpdateCredential(ctx, id, name, cred)
}

// UpdateCredential updates a channel's name and/or credential. The credential
// is replaced wholesale for the platform; every call bumps UpdatedAt and clears
// the last-check so the assembler notices the change.
func (s *Service) UpdateCredential(ctx context.Context, id, name string, cred *domain.Credential) (domain.Channel, error) {
	ch, err := s.Store.Get(ctx, id)
	if err != nil {
		return domain.Channel{}, err
	}
	if name != "" {
		ch.Name = name
	}
	if cred != nil {
		if err := validateCredential(domain.Platform(ch.Platform), *cred); err != nil {
			return domain.Channel{}, err
		}
		ct, err := domain.EncryptCredential(s.Key, *cred)
		if err != nil {
			return domain.Channel{}, err
		}
		ch.CredentialCiphertext = ct
	}
	ch.LastCheckKind = ""
	ch.LastCheckMessage = ""
	ch.LastCheckAt = nil
	ch.UpdatedAt = s.nowFn()().UTC()
	if err := s.Store.Update(ctx, ch); err != nil {
		return domain.Channel{}, err
	}
	return ch, nil
}

func wecomAdapterReachability(ctx context.Context, adapterStatus func(context.Context, string) (state string, lastErr string, found bool), id string) ReachabilityResult {
	if adapterStatus == nil {
		return ReachabilityResult{Kind: ReachabilityOther, Message: "wecom adapter status unavailable"}
	}
	state, lastErr, found := adapterStatus(ctx, id)
	if found && state == "running" {
		return ReachabilityResult{OK: true, Kind: ReachabilityOK}
	}
	if strings.TrimSpace(lastErr) != "" {
		return ReachabilityResult{Kind: ReachabilityOther, Message: lastErr}
	}
	if !found {
		return ReachabilityResult{Kind: ReachabilityOther, Message: "wecom adapter not found"}
	}
	return ReachabilityResult{Kind: ReachabilityOther, Message: "wecom adapter " + state}
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
