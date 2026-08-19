package domain

import "time"

const (
	MaxTreeDepth      = 2 // root + one level of grouping
	MaxRootEntries    = 6 // explicit main-keyboard entries
)

type ReplyPayload struct {
	Text   string   `json:"text,omitempty"`
	Images []string `json:"images,omitempty"`
}

type MenuItem struct {
	ID              string
	ParentID        string
	Label           string
	Order           int
	Enabled         bool
	CapabilityID    string
	Params          map[string]any
	RenderOverride  map[string]any
	PlaceholderText string
	IntroText       string
	Reply           *ReplyPayload
}

type MenuNode struct {
	ID              string         `json:"id"`
	ParentID        string         `json:"parent_id,omitempty"`
	Label           string         `json:"label"`
	Order           int            `json:"order"`
	Enabled         bool           `json:"enabled"`
	CapabilityID    string         `json:"capability_id,omitempty"`
	Params          map[string]any `json:"params,omitempty"`
	RenderOverride  map[string]any `json:"render_override,omitempty"`
	PlaceholderText string         `json:"placeholder_text,omitempty"`
	IntroText       string         `json:"intro_text,omitempty"`
	Reply           *ReplyPayload  `json:"reply,omitempty"`
	Children        []MenuNode     `json:"children,omitempty"`
}

type MenuTree struct {
	ChannelID string     `json:"channel_id"`
	Items     []MenuNode `json:"items"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type PlacementStep struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type MenuPlacement struct {
	ChannelID string          `json:"channel_id"`
	ItemID    string          `json:"item_id"`
	Path      []PlacementStep `json:"path"`
}

// Extra is a legacy platform-specific data blob (phased out; see interaction
// framework design D8 — platform differences move to capability render decls).
type Extra struct {
	ChannelID  string    `json:"channel_id"`
	MenuItemID string    `json:"menu_item_id"`
	ExtraType  string    `json:"extra_type"`
	ExtraJSON  string    `json:"extra_json"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// CaseIDsOf returns the case_ids from an open_case capability's params.
func CaseIDsOf(node MenuNode) []string {
	if node.CapabilityID != "open_case" {
		return nil
	}
	raw, ok := node.Params["case_ids"].([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		if s, ok := v.(string); ok && s != "" {
			out = append(out, s)
		}
	}
	return out
}
