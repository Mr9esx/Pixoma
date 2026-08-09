package domain

import "time"

const DocumentIDDefault = "default"

type MenuAction string

const (
	ActionOpenCase        MenuAction = "open_case"
	ActionListCasesByTag  MenuAction = "list_cases_by_tag"
	ActionPlaceholder     MenuAction = "placeholder"
	ActionReplyMedia      MenuAction = "reply_media"
)

type ReplyPayload struct {
	Text   string   `json:"text,omitempty"`
	Images []string `json:"images,omitempty"`
}

type MenuItem struct {
	ID              string        `json:"id"`
	Label           string        `json:"label"`
	Row             int           `json:"row"`
	Col             int           `json:"col"`
	Enabled         bool          `json:"enabled"`
	Action          MenuAction    `json:"action"`
	CaseID          string        `json:"case_id,omitempty"`
	Tag             string        `json:"tag,omitempty"`
	PlaceholderText string        `json:"placeholder_text,omitempty"`
	Reply           *ReplyPayload `json:"reply,omitempty"`
}

type MenuDocument struct {
	ID        string     `json:"id"`
	Items     []MenuItem `json:"items"`
	UpdatedAt time.Time  `json:"updated_at"`
}
