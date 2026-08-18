// Package ports defines the channel runtime contract implemented by platform adapters.
package ports

import (
	"context"
	"fmt"

	"github.com/mr9esx/comfyui_tgbot/internal/identity/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

// ActionType is a normalized user intent translated by adapters.
type ActionType string

const (
	ActionOpenMenu     ActionType = "open_menu"
	ActionOpenFolder   ActionType = "open_folder"
	ActionOpenCase     ActionType = "open_case"
	ActionStartCase    ActionType = "start_case"
	ActionSubmitText   ActionType = "submit_text"
	ActionSubmitMedia  ActionType = "submit_media"
	ActionConfirm      ActionType = "confirm"
	ActionSkip         ActionType = "skip"
	ActionExit         ActionType = "exit"
	ActionContinue     ActionType = "continue"
	ActionReplaceStart ActionType = "replace_start"
)

// Action is a normalized user intent with platform-specific ids resolved.
type Action struct {
	Type       ActionType    `json:"type"`
	MenuItemID string        `json:"menu_item_id,omitempty"`
	CaseID     string        `json:"case_id,omitempty"`
	BackRef    string        `json:"back_ref,omitempty"`
	Text       string        `json:"text,omitempty"`
	Media      *InboundMedia `json:"media,omitempty"`
}

// Validate checks required fields per action type.
func (a Action) Validate() error {
	switch a.Type {
	case ActionOpenMenu, ActionConfirm, ActionSkip, ActionExit, ActionContinue:
		return nil
	case ActionOpenFolder:
		if a.MenuItemID == "" {
			return fmt.Errorf("ports: open_folder requires menu_item_id")
		}
		return nil
	case ActionOpenCase, ActionStartCase, ActionReplaceStart:
		if a.CaseID == "" {
			return fmt.Errorf("ports: %s requires case_id", a.Type)
		}
		return nil
	case ActionSubmitText:
		if a.Text == "" {
			return fmt.Errorf("ports: submit_text requires text")
		}
		return nil
	case ActionSubmitMedia:
		if a.Media == nil || a.Media.ExternalFileID == "" {
			return fmt.Errorf("ports: submit_media requires media")
		}
		return nil
	default:
		return fmt.Errorf("ports: unknown action type %q", a.Type)
	}
}

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

// InboundEvent is a normalized platform event.
type InboundEvent struct {
	Addr   sharedkernel.ChannelAddr `json:"addr"`
	Kind   EventKind                `json:"kind"`
	Text   string                   `json:"text,omitempty"`
	Media  *InboundMedia            `json:"media,omitempty"`
	Action Action                   `json:"action,omitempty"`
}

// MenuEntry is a root menu entry for SendMenu.
type MenuEntry struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// Button is a clickable list item for SendList.
type Button struct {
	Text   string `json:"text"`
	Action Action `json:"action"`
}

// Outbound renders messages to a channel conversation.
type Outbound interface {
	SendText(ctx context.Context, addr sharedkernel.ChannelAddr, text string) error
	SendMenu(ctx context.Context, addr sharedkernel.ChannelAddr, title string, items []MenuEntry) error
	SendList(ctx context.Context, addr sharedkernel.ChannelAddr, title string, rows [][]Button) error
	SendMedia(ctx context.Context, addr sharedkernel.ChannelAddr, ref sharedkernel.BlobRef, caption string) error
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
