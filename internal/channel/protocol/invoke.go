// Package protocol defines the channel interaction contract: UI events become
// capability invokes, results are channel-agnostic, and navigation is handled
// at the protocol layer.
package protocol

// AccountCtx identifies the channel-scoped account executing a capability.
type AccountCtx struct {
	ChannelID      string `json:"channel_id"`
	ExternalUserID string `json:"external_user_id"`
	InternalUserID string `json:"internal_user_id"`
}

// Nav carries protocol-level navigation context (back anchor), separate from
// capability parameters.
type Nav struct {
	Back string `json:"back"` // "root" | group id | flow-step anchor
	Step string `json:"step,omitempty"`
}

// CapabilityInvoke is the normalized protocol-level invocation.
type CapabilityInvoke struct {
	CapabilityID string         `json:"capability_id"`
	Params       map[string]any `json:"params"`
	Account      AccountCtx     `json:"account"`
	Nav          Nav            `json:"nav"`
}

// Option is a selectable choice rendered by adapters as buttons/cards.
type Option struct {
	Label string         `json:"label"`
	Value map[string]any `json:"value"`
}

// MediaRef references result media by blob key.
type MediaRef struct {
	Key  string `json:"key"`
	MIME string `json:"mime,omitempty"`
}

// Result is the channel-agnostic outcome of a capability invocation.
type Result struct {
	Text    string     `json:"text"`
	Options []Option   `json:"options"`
	Media   []MediaRef `json:"media"`
	Error   *string    `json:"error,omitempty"`
}

// RenderDecl is the per-channel rendering declaration for a capability entry.
type RenderDecl struct {
	Entry  string         `json:"entry"` // root | message_button | group
	Config map[string]any `json:"config"`
}
