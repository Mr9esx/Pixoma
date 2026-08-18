package domain

import "time"

const (
	MaxTreeDepth    = 5
	MaxIntroTextLen = 3500
)

type MenuKind string

const (
	KindFolder      MenuKind = "folder"
	KindOpenCase    MenuKind = "open_case"
	KindPlaceholder MenuKind = "placeholder"
	KindReplyMedia  MenuKind = "reply_media"
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
	Kind            MenuKind
	CaseIDs         []string
	PlaceholderText string
	IntroText       string
	Reply           *ReplyPayload
}

type MenuNode struct {
	ID              string        `json:"id"`
	ParentID        string        `json:"parent_id,omitempty"`
	Label           string        `json:"label"`
	Order           int           `json:"order"`
	Enabled         bool          `json:"enabled"`
	Kind            MenuKind      `json:"kind"`
	CaseIDs         []string      `json:"case_ids,omitempty"`
	PlaceholderText string        `json:"placeholder_text,omitempty"`
	IntroText       string        `json:"intro_text,omitempty"`
	Reply           *ReplyPayload `json:"reply,omitempty"`
	Children        []MenuNode    `json:"children,omitempty"`
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

// Extra is a platform-specific data blob attached to a channel menu item.
type Extra struct {
	ChannelID  string
	MenuItemID string
	ExtraType  string
	ExtraJSON  string
	UpdatedAt  time.Time
}
