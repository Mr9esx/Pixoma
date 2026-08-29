package domain

import (
	"errors"
	"time"
)

var (
	ErrNotFound         = errors.New("channel not found")
	ErrDeleteRestricted = errors.New("channel delete restricted")
)

// Platform identifies a messaging platform.
type Platform string

const (
	PlatformTelegram Platform = "telegram"
	PlatformFeishu   Platform = "feishu"
	PlatformWeCom    Platform = "wecom"
	PlatformDingTalk Platform = "dingtalk"
)

// ValidPlatform reports whether p is a supported platform.
func ValidPlatform(p Platform) bool {
	switch p {
	case PlatformTelegram, PlatformFeishu, PlatformWeCom, PlatformDingTalk:
		return true
	default:
		return false
	}
}

// Channel is a channel instance bound to a messaging platform.
type Channel struct {
	ID       string
	Platform string
	Name     string
	// ExtraInfo holds platform-specific identity metadata as raw JSON
	// (e.g. the Telegram getMe User object). Fields differ per platform,
	// so it is kept opaque here.
	ExtraInfo            string
	CredentialCiphertext string
	Enabled              bool
	CreatedAt            time.Time
	UpdatedAt            time.Time
}
