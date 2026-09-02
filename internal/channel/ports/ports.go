// Package ports defines the channel runtime contract implemented by platform adapters.
package ports

import (
	"context"

	"github.com/mr9esx/comfyui_tgbot/internal/identity/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

// InboundMedia is a platform media reference awaiting materialization.
type InboundMedia struct {
	ExternalFileID string `json:"external_file_id"`
	MIME           string `json:"mime,omitempty"`
}

// EventKind classifies inbound events.
type EventKind string

const (
	EventText     EventKind = "text"
	EventMedia    EventKind = "media"
	EventCallback EventKind = "callback"
)

// MenuEntry is a root menu entry for SendMenu.
type MenuEntry struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// Button is a clickable list item for SendList.
type Button struct {
	Text string `json:"text"`
	Data string `json:"data"` // platform-encoded callback payload (adapter-owned)
}

// Outbound renders messages to a channel conversation.
type Outbound interface {
	SendText(ctx context.Context, addr sharedkernel.ChannelAddr, text string) error
	SendMenu(ctx context.Context, addr sharedkernel.ChannelAddr, title string, items []MenuEntry) error
	SendList(ctx context.Context, addr sharedkernel.ChannelAddr, title string, rows [][]Button) error
	SendMedia(ctx context.Context, addr sharedkernel.ChannelAddr, ref sharedkernel.BlobRef, caption string, buttons [][]Button) error
	SendMediaURL(ctx context.Context, addr sharedkernel.ChannelAddr, imageURL, caption string, buttons [][]Button) error
	EditReplyMarkup(ctx context.Context, addr sharedkernel.ChannelAddr, messageID int, rows [][]Button) error
}

// MediaBridge materializes platform media references to/from blobs.
type MediaBridge interface {
	Download(ctx context.Context, externalFileID, mime string) ([]byte, error)
	Upload(ctx context.Context, data []byte, mime string) (string, error)
}

// IdentityResolver maps a channel-scoped external user to an internal user.
type IdentityResolver interface {
	Resolve(ctx context.Context, addr sharedkernel.ChannelAddr, profile domain.UpsertFrom) (string, error)
}
