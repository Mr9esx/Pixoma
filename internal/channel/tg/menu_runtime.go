package tg

import (
	"context"
	"log/slog"
	"sort"

	"github.com/go-telegram/bot/models"

	tgmenudomain "github.com/mr9esx/comfyui_tgbot/internal/tgmenu/domain"
)

// MenuReader is the read port for main ReplyKeyboard config.
type MenuReader interface {
	GetMenu(ctx context.Context) (tgmenudomain.MenuTree, error)
}

// BuildReplyKeyboard builds a persistent ReplyKeyboard from enabled root items ordered by (row, col).
func BuildReplyKeyboard(tree tgmenudomain.MenuTree) *models.ReplyKeyboardMarkup {
	items := make([]tgmenudomain.MenuNode, 0, len(tree.Items))
	for _, it := range tree.Items {
		if it.Enabled {
			items = append(items, it)
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Row != items[j].Row {
			return items[i].Row < items[j].Row
		}
		return items[i].Col < items[j].Col
	})
	var kb [][]models.KeyboardButton
	var curRow = -1
	for _, it := range items {
		if it.Row != curRow {
			kb = append(kb, nil)
			curRow = it.Row
		}
		kb[len(kb)-1] = append(kb[len(kb)-1], models.KeyboardButton{Text: it.Label})
	}
	return &models.ReplyKeyboardMarkup{
		Keyboard:       kb,
		ResizeKeyboard: true,
		IsPersistent:   true,
	}
}

// FindEnabledRootByLabel returns the first enabled root item whose label matches.
func FindEnabledRootByLabel(tree tgmenudomain.MenuTree, label string) (tgmenudomain.MenuNode, bool) {
	for _, it := range tree.Items {
		if it.Enabled && it.Label == label {
			return it, true
		}
	}
	return tgmenudomain.MenuNode{}, false
}

func findNodeByID(nodes []tgmenudomain.MenuNode, id string) (tgmenudomain.MenuNode, bool) {
	for _, n := range nodes {
		if n.ID == id {
			return n, true
		}
		if child, ok := findNodeByID(n.Children, id); ok {
			return child, true
		}
	}
	return tgmenudomain.MenuNode{}, false
}

func (a *Adapter) loadMenu(ctx context.Context) tgmenudomain.MenuTree {
	if a != nil && a.Menu != nil {
		tree, err := a.Menu.GetMenu(ctx)
		if err == nil {
			return tree
		}
		slog.Error("tg menu load failed; using default seed", "err", err)
	}
	return tgmenudomain.DefaultSeedTree()
}
