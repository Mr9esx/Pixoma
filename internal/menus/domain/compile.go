package domain

// CompiledButton is one cell on the reply keyboard.
type CompiledButton struct {
	ID     string
	Label  string
	Action TreeAction
}

// CompiledMenu is the runtime view of a MenuTree.
type CompiledMenu struct {
	Rows           [][]CompiledButton
	CardByOpenerID map[string]TreeCard
}

// WorkflowPath is one open_workflow occurrence in the tree.
type WorkflowPath struct {
	WorkflowID string
	ButtonID   string
	Kind       string // keyboard | card_button
	Labels     []string
}

// Compile folds the root keyboard and indexes nested cards by opener button id.
func Compile(t MenuTree) CompiledMenu {
	out := CompiledMenu{CardByOpenerID: map[string]TreeCard{}}
	cols := t.Columns
	if cols < 1 {
		cols = 1
	}
	var row []CompiledButton
	for i := range t.Items {
		b := t.Items[i]
		row = append(row, CompiledButton{ID: b.ID, Label: b.Label, Action: b.Action})
		if len(row) == cols {
			out.Rows = append(out.Rows, row)
			row = nil
		}
		indexCards(b, out.CardByOpenerID)
	}
	if len(row) > 0 {
		out.Rows = append(out.Rows, row)
	}
	return out
}

func indexCards(b TreeButton, cards map[string]TreeCard) {
	if b.Action.Type != "open_card" || b.Action.Card == nil {
		return
	}
	cards[b.ID] = *b.Action.Card
	for i := range b.Action.Card.Buttons {
		indexCards(b.Action.Card.Buttons[i], cards)
	}
}

// WalkWorkflowPlacements lists every open_workflow node with its label path from the root.
func WalkWorkflowPlacements(t MenuTree) []WorkflowPath {
	var out []WorkflowPath
	for i := range t.Items {
		walkButton(&t.Items[i], nil, "keyboard", &out)
	}
	return out
}

func walkButton(b *TreeButton, prefix []string, kind string, out *[]WorkflowPath) {
	path := append(append([]string{}, prefix...), b.Label)
	if b.Action.Type == "open_workflow" {
		*out = append(*out, WorkflowPath{
			WorkflowID: b.Action.WorkflowID,
			ButtonID:   b.ID,
			Kind:       kind,
			Labels:     path,
		})
	}
	if b.Action.Type == "open_card" && b.Action.Card != nil {
		for i := range b.Action.Card.Buttons {
			walkButton(&b.Action.Card.Buttons[i], path, "card_button", out)
		}
	}
}
