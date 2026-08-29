package domain

import "errors"

// ErrValidation marks invalid menu/card configurations.
var ErrValidation = errors.New("menucard validation failed")

// Menu is the main keyboard definition for a channel.
type Menu struct {
	ID      string     `json:"id"`
	Name    string     `json:"name"`
	Columns int        `json:"columns"`
	Items   []MenuItem `json:"items"`
}

// MenuItem is a button on the main keyboard.
type MenuItem struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Action Action `json:"action"`
}

// DefaultMenu is the fallback main menu when a channel has none configured.
func DefaultMenu() Menu {
	return Menu{
		ID: "default", Name: "主菜单", Columns: 2,
		Items: []MenuItem{
			{ID: "mi-image", Label: "🖼 图片生成", Action: Action{Type: "open_workflow", WorkflowID: "default-image"}},
			{ID: "mi-help", Label: "🆘 帮助", Action: Action{Type: "send_text", Text: "请使用主菜单中的功能。"}},
		},
	}
}

// Action describes what happens when a button is clicked.
//
// v2 schema:
//   - workflow_id: 绑定的 case id（单数，替代旧的 workflow_ids[] + mode + direct_id）
type Action struct {
	Type       string  `json:"type"` // open_card | open_workflow | send_text | send_media | open_url | copy_text
	CardID     string  `json:"card_id,omitempty"`
	WorkflowID string  `json:"workflow_id,omitempty"`
	Text       string  `json:"text,omitempty"`
	Media      []Media `json:"media,omitempty"`
	URL        string  `json:"url,omitempty"`
}

func ValidateMenu(m Menu) error {
	if m.ID == "" {
		return ErrValidation
	}
	for _, it := range m.Items {
		if it.Label == "" {
			return ErrValidation
		}
		if err := ValidateAction(it.Action); err != nil {
			return err
		}
	}
	return nil
}

func ValidateAction(a Action) error {
	switch a.Type {
	case "open_card":
		if a.CardID == "" {
			return ErrValidation
		}
	case "open_workflow":
		if a.WorkflowID == "" {
			return ErrValidation
		}
	case "open_url":
		if !hasHTTPPrefix(a.URL) {
			return ErrValidation
		}
	case "send_media":
		if len(a.Media) == 0 {
			return ErrValidation
		}
		for _, m := range a.Media {
			if m.URL == "" {
				return ErrValidation
			}
		}
	default:
		return ErrValidation
	}
	return nil
}

func hasHTTPPrefix(u string) bool {
	if len(u) >= 7 && u[:7] == "http://" {
		return true
	}
	return len(u) >= 8 && u[:8] == "https://"
}
