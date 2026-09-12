package mcp

import (
	"context"

	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
)

type identityKey struct{}

// Identity is the MCP channel user bound to a Bearer (or a test fixture).
type Identity struct {
	UserID         string
	ChannelID      string
	ExternalUserID string
}

func (id Identity) ChatID() sharedkernel.ChatID {
	if id.ChannelID == "" || id.ExternalUserID == "" {
		return ""
	}
	return sharedkernel.ChatID(sharedkernel.FormatChatID(sharedkernel.ChannelAddr{
		ChannelID:      id.ChannelID,
		ExternalChatID: id.ExternalUserID,
	}))
}

func (id Identity) ok() bool {
	return id.UserID != "" && id.ChannelID != "" && id.ExternalUserID != ""
}

// WithIdentity stores the MCP user on the request context (set by Bearer middleware).
func WithIdentity(ctx context.Context, id Identity) context.Context {
	return context.WithValue(ctx, identityKey{}, id)
}

// IdentityFrom returns the MCP user on ctx, if any.
func IdentityFrom(ctx context.Context) (Identity, bool) {
	id, ok := ctx.Value(identityKey{}).(Identity)
	return id, ok && id.ok()
}
