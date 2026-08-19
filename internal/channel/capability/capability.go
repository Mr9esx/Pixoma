// Package capability provides the capability registry: business modules
// register here so adapters and menus can invoke them without knowing internals.
package capability

import (
	"context"
	"encoding/json"

	"github.com/mr9esx/comfyui_tgbot/internal/channel/protocol"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

// Capability is a registered business module.
type Capability interface {
	ID() string
	DisplayName() string
	ParamsSchema() json.RawMessage // JSON Schema (draft-07 subset)
	Render(channelID string, override map[string]any) (protocol.RenderDecl, error)
	Invoke(ctx context.Context, acct protocol.AccountCtx, nav protocol.Nav, chatID sharedkernel.ChatID, params map[string]any) (protocol.Result, error)
}

// AdminSchemaProvider lets a capability expose a reduced schema for admin
// configuration, hiding runtime-only params from the editor.
type AdminSchemaProvider interface {
	AdminParamsSchema() json.RawMessage
}

// AdminSchema returns the admin-editable params schema for a capability.
func AdminSchema(c Capability) json.RawMessage {
	if p, ok := c.(AdminSchemaProvider); ok {
		return p.AdminParamsSchema()
	}
	return c.ParamsSchema()
}

// MergeRender applies a menu-item render override onto a capability's base
// render declaration, replacing only the keys present in the override.
func MergeRender(base protocol.RenderDecl, override map[string]any) protocol.RenderDecl {
	if len(override) == 0 {
		return base
	}
	merged := protocol.RenderDecl{Entry: base.Entry}
	if base.Config == nil {
		merged.Config = map[string]any{}
	} else {
		merged.Config = make(map[string]any, len(base.Config))
		for k, v := range base.Config {
			merged.Config[k] = v
		}
	}
	for k, v := range override {
		merged.Config[k] = v
	}
	return merged
}
