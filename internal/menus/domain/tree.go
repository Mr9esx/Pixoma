package domain

import "encoding/json"

const (
	maxRootButtons = 6
	minColumns     = 1
	maxColumns     = 6
	maxCardDepth   = 8
)

// MenuTree is the nested menu document for a channel.
type MenuTree struct {
	ID      string       `json:"id"`
	Columns int          `json:"columns"`
	Items   []TreeButton `json:"items"`
}

// TreeButton is a keyboard or card button in the nested tree.
type TreeButton struct {
	ID     string     `json:"id"`
	Label  string     `json:"label"`
	Action TreeAction `json:"action"`
}

// TreeAction is what a tree button does. Open-card actions embed TreeCard.
type TreeAction struct {
	Type       string    `json:"type"`
	WorkflowID string    `json:"workflow_id,omitempty"`
	Text       string    `json:"text,omitempty"`
	Media      []Media   `json:"media,omitempty"`
	URL        string    `json:"url,omitempty"`
	Card       *TreeCard `json:"card,omitempty"`
}

// TreeCard is an inline message nested under an open_card button.
type TreeCard struct {
	Text    string       `json:"text"`
	Media   []Media      `json:"media,omitempty"`
	Buttons []TreeButton `json:"buttons,omitempty"`
}

// DefaultMenuTree is used when a channel has no saved document.
func DefaultMenuTree(channelID string) MenuTree {
	return MenuTree{
		ID:      channelID,
		Columns: 2,
		Items: []TreeButton{
			{ID: "mi-image", Label: "图片生成", Action: TreeAction{Type: "open_workflow", WorkflowID: "default-image"}},
			{ID: "mi-help", Label: "帮助", Action: TreeAction{Type: "send_text", Text: "使用主菜单中的功能。"}},
		},
	}
}

// ValidateTree checks a nested menu document.
func ValidateTree(t MenuTree, workflowExists func(string) bool) error {
	if t.ID == "" {
		return ErrValidation
	}
	if t.Columns < minColumns || t.Columns > maxColumns {
		return ErrValidation
	}
	n := len(t.Items)
	if n < 1 || n > maxRootButtons {
		return ErrValidation
	}
	seen := map[string]struct{}{}
	for i := range t.Items {
		if err := validateButton(&t.Items[i], 0, seen, workflowExists); err != nil {
			return err
		}
	}
	return nil
}

func validateButton(b *TreeButton, depth int, seen map[string]struct{}, workflowExists func(string) bool) error {
	if b.ID == "" || b.Label == "" {
		return ErrValidation
	}
	if _, ok := seen[b.ID]; ok {
		return ErrValidation
	}
	seen[b.ID] = struct{}{}
	return validateTreeAction(&b.Action, depth, seen, workflowExists)
}

func validateTreeAction(a *TreeAction, depth int, seen map[string]struct{}, workflowExists func(string) bool) error {
	switch a.Type {
	case "open_card":
		if a.Card == nil {
			return ErrValidation
		}
		next := depth + 1
		if next > maxCardDepth {
			return ErrValidation
		}
		return validateTreeCard(a.Card, next, seen, workflowExists)
	case "open_workflow":
		if a.WorkflowID == "" {
			return ErrValidation
		}
		if workflowExists != nil && !workflowExists(a.WorkflowID) {
			return ErrValidation
		}
		return nil
	case "list_tasks":
		return nil
	case "open_url":
		if !hasHTTPPrefix(a.URL) {
			return ErrValidation
		}
		return nil
	case "send_media":
		if len(a.Media) == 0 {
			return ErrValidation
		}
		for _, m := range a.Media {
			if !hasHTTPPrefix(m.URL) {
				return ErrValidation
			}
		}
		return nil
	case "send_text", "copy_text":
		return nil
	default:
		return ErrValidation
	}
}

func validateTreeCard(c *TreeCard, depth int, seen map[string]struct{}, workflowExists func(string) bool) error {
	if c.Text == "" && len(c.Media) == 0 {
		return ErrValidation
	}
	for i := range c.Media {
		if !hasHTTPPrefix(c.Media[i].URL) {
			return ErrValidation
		}
	}
	for i := range c.Buttons {
		if err := validateButton(&c.Buttons[i], depth, seen, workflowExists); err != nil {
			return err
		}
	}
	return nil
}

// ParseStoredMenuTree decodes a saved document. Old Menu+card_id JSON is not usable.
func ParseStoredMenuTree(raw []byte, channelID string) (MenuTree, bool) {
	var t MenuTree
	if err := json.Unmarshal(raw, &t); err != nil {
		return MenuTree{}, false
	}
	if channelID != "" {
		t.ID = channelID
	}
	if err := ValidateTree(t, func(string) bool { return true }); err != nil {
		return MenuTree{}, false
	}
	return t, true
}

// FindButtonByID walks the tree for a button id.
func FindButtonByID(t MenuTree, id string) (TreeButton, bool) {
	var found TreeButton
	ok := false
	var walk func([]TreeButton)
	walk = func(items []TreeButton) {
		if ok {
			return
		}
		for i := range items {
			if items[i].ID == id {
				found = items[i]
				ok = true
				return
			}
			if items[i].Action.Card != nil {
				walk(items[i].Action.Card.Buttons)
			}
		}
	}
	walk(t.Items)
	return found, ok
}
