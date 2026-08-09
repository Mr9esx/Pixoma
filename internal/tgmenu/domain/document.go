package domain

import "time"

const (
	DocumentIDDefault = "default"
	BotIDDefault      = "default"
	MaxTreeDepth      = 5
)

type MenuKind string

const (
	KindFolder         MenuKind = "folder"
	KindOpenCase       MenuKind = "open_case"
	KindPlaceholder    MenuKind = "placeholder"
	KindReplyMedia     MenuKind = "reply_media"
	KindListCasesByTag MenuKind = "list_cases_by_tag"
)

type ReplyPayload struct {
	Text   string   `json:"text,omitempty"`
	Images []string `json:"images,omitempty"`
}

type MenuItem struct {
	ID              string
	ParentID        string
	Label           string
	Row             int
	Col             int
	Enabled         bool
	Kind            MenuKind
	CaseIDs         []string
	Tag             string
	PlaceholderText string
	Reply           *ReplyPayload
}

type MenuNode struct {
	ID              string        `json:"id"`
	ParentID        string        `json:"parent_id,omitempty"`
	Label           string        `json:"label"`
	Row             int           `json:"row"`
	Col             int           `json:"col"`
	Enabled         bool          `json:"enabled"`
	Kind            MenuKind      `json:"kind"`
	CaseIDs         []string      `json:"case_ids,omitempty"`
	Tag             string        `json:"tag,omitempty"`
	PlaceholderText string        `json:"placeholder_text,omitempty"`
	Reply           *ReplyPayload `json:"reply,omitempty"`
	Children        []MenuNode    `json:"children,omitempty"`
}

type MenuTree struct {
	ID        string     `json:"id"`
	BotID     string     `json:"bot_id"`
	Items     []MenuNode `json:"items"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type PlacementStep struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type MenuPlacement struct {
	MenuID string          `json:"menu_id"`
	ItemID string          `json:"item_id"`
	Path   []PlacementStep `json:"path"`
}
