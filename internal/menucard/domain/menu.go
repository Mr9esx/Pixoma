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

// Action describes what happens when a button is clicked.
type Action struct {
	Type        string   `json:"type"` // open_card | open_workflow | send_text | send_media | open_url | copy_text | placeholder
	CardID      string   `json:"card_id,omitempty"`
	WorkflowIDs []string `json:"workflow_ids,omitempty"`
	Mode        string   `json:"mode,omitempty"` // list | direct
	DirectID    string   `json:"direct_id,omitempty"`
	Text        string   `json:"text,omitempty"`
	Media       []Media  `json:"media,omitempty"`
	URL         string   `json:"url,omitempty"`
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
		if len(a.WorkflowIDs) == 0 {
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
	case "send_text", "copy_text", "placeholder":
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
