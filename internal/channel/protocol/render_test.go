package protocol

import (
	"testing"

	"github.com/mr9esx/comfyui_tgbot/internal/menu/domain"
)

func TestBuildKeyboardLayoutColumns(t *testing.T) {
	tree := domain.MenuTree{
		Items: []domain.MenuNode{
			{ID: "a", Label: "A", Order: 0, Enabled: true, RenderOverride: map[string]any{"columns": 3}},
			{ID: "b", Label: "B", Order: 1, Enabled: true},
			{ID: "c", Label: "C", Order: 2, Enabled: true},
		},
	}
	rows := BuildKeyboardLayout(tree)
	if len(rows) != 1 || len(rows[0]) != 3 {
		t.Fatalf("3-column layout: %+v", rows)
	}
}

func TestBuildPreviewGroups(t *testing.T) {
	tree := domain.MenuTree{
		Items: []domain.MenuNode{
			{ID: "root1", Label: "开 Case", Order: 0, Enabled: true, CapabilityID: "open_case"},
			{ID: "g", Label: "视频专区", Order: 1, Enabled: true,
				Children: []domain.MenuNode{
					{ID: "c1", Label: "图片 B", Order: 0, Enabled: true, CapabilityID: "open_case"},
					{ID: "sub", Label: "子组", Order: 1, Enabled: true},
				}},
		},
	}
	p := BuildPreview(tree)
	if len(p.MainKeyboard) != 1 || len(p.MainKeyboard[0]) != 2 {
		t.Fatalf("main=%+v", p.MainKeyboard)
	}
	g, ok := p.Groups["g"]
	if !ok || len(g.Buttons) != 2 {
		t.Fatalf("group=%+v", p.Groups)
	}
	if g.Buttons[0].Kind != "capability" || g.Buttons[1].Kind != "group" {
		t.Fatalf("buttons=%+v", g.Buttons)
	}
}
