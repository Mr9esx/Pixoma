package protocol

import (
	"sort"

	"github.com/mr9esx/comfyui_tgbot/internal/menu/domain"
)

// RootColumns returns the configured ReplyKeyboard column count for a root
// item, read from its render override ("columns"), defaulting to 2.
func RootColumns(item domain.MenuNode) int {
	switch v := item.RenderOverride["columns"].(type) {
	case float64:
		if v >= 1 && v <= 8 {
			return int(v)
		}
	case int:
		if v >= 1 && v <= 8 {
			return v
		}
	}
	return 2
}

// BuildKeyboardLayout returns the root-entry layout (rows of labels) using the
// same column rule as the real TG renderer, keeping preview and render in sync.
func BuildKeyboardLayout(tree domain.MenuTree) [][]string {
	items := make([]domain.MenuNode, 0, len(tree.Items))
	for _, it := range tree.Items {
		if it.Enabled {
			items = append(items, it)
		}
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].Order < items[j].Order })

	var rows [][]string
	var cur []string
	cols := 2
	for i, it := range items {
		if i == 0 {
			cols = RootColumns(it)
			if cols < 1 {
				cols = 2
			}
		}
		cur = append(cur, it.Label)
		if len(cur) >= cols || i == len(items)-1 {
			rows = append(rows, cur)
			cur = nil
		}
	}
	if len(cur) > 0 {
		rows = append(rows, cur)
	}
	return rows
}

// PreviewButton is a message-button preview entry.
type PreviewButton struct {
	Label string `json:"label"`
	Kind  string `json:"kind"` // capability | group
}

// GroupPreview is the message-button preview of a one-level group.
type GroupPreview struct {
	Title   string          `json:"title"`
	Buttons []PreviewButton `json:"buttons"`
}

// PreviewDTO is the channel-agnostic render preview (main keyboard + groups).
type PreviewDTO struct {
	MainKeyboard [][]string             `json:"main_keyboard"`
	Groups       map[string]GroupPreview `json:"groups"`
}

// BuildPreview builds a channel-agnostic preview from the same layout logic the
// real renderer uses, so the admin preview matches what users see.
func BuildPreview(tree domain.MenuTree) PreviewDTO {
	dto := PreviewDTO{
		MainKeyboard: BuildKeyboardLayout(tree),
		Groups:       map[string]GroupPreview{},
	}
	var walk func(nodes []domain.MenuNode)
	walk = func(nodes []domain.MenuNode) {
		for _, n := range nodes {
			if len(n.Children) > 0 {
				g := GroupPreview{Title: n.Label}
				for _, child := range n.Children {
					if !child.Enabled {
						continue
					}
					kind := "capability"
					if child.CapabilityID == "" {
						kind = "group"
					}
					g.Buttons = append(g.Buttons, PreviewButton{Label: child.Label, Kind: kind})
				}
				dto.Groups[n.ID] = g
			}
			walk(n.Children)
		}
	}
	walk(tree.Items)
	return dto
}
