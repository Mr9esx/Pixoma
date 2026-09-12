package application

import (
	"context"
	"log/slog"

	"github.com/Mr9esx/Pixoma/internal/channels/domain"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
)

// NotifyHandler delivers a user notification through a channel adapter.
type NotifyHandler interface {
	HandleNotify(ctx context.Context, n sharedkernel.UserNotify) error
}

// NotifyRouter routes user notifications to the owning channel's handler.
type NotifyRouter struct {
	HandlerByChannel func(channelID string) (NotifyHandler, bool)
	PlatformOf       func(ctx context.Context, channelID string) (string, bool)
}

// Publish routes n to the channel handler; unknown channels are logged and skipped.
func (r *NotifyRouter) Publish(ctx context.Context, n sharedkernel.UserNotify) error {
	addr, err := sharedkernel.ParseChatID(string(n.ChatID))
	if err != nil {
		return err
	}
	if r != nil && r.PlatformOf != nil {
		if platform, ok := r.PlatformOf(ctx, addr.ChannelID); ok && platform == string(domain.PlatformMCP) {
			return nil
		}
	}
	if r == nil || r.HandlerByChannel == nil {
		slog.Warn("notify router: no handler lookup configured", "chat_id", n.ChatID)
		return nil
	}
	h, ok := r.HandlerByChannel(addr.ChannelID)
	if !ok {
		slog.Warn("notify router: unknown channel", "channel_id", addr.ChannelID, "task", n.TaskID)
		return nil
	}
	return h.HandleNotify(ctx, n)
}
