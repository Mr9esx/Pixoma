package tg

import (
	"context"
	"log/slog"
	"sort"

	"github.com/go-telegram/bot/models"

	"github.com/mr9esx/comfyui_tgbot/internal/menu/domain"
)

// MenuReader is the read port for a channel's menu tree.
type MenuReader interface {
	GetMenu(ctx context.Context) (domain.MenuTree, error)
}

// BuildReplyKeyboard builds a persistent ReplyKeyboard from enabled root items.
// Root items are laid out by order with the configured column count (default 2).
func BuildReplyKeyboard(tree domain.MenuTree) *models.ReplyKeyboardMarkup {
	items := make([]domain.MenuNode, 0, len(tree.Items))
	for _, it := range tree.Items {
		if it.Enabled {
			items = append(items, it)
		}
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].Order < items[j].Order })

	var kb [][]models.KeyboardButton
	var cur []models.KeyboardButton
	cols := 2
	for i, it := range items {
		if i == 0 {
			cols = RootColumns(it)
			if cols < 1 {
				cols = 2
			}
		}
		cur = append(cur, models.KeyboardButton{Text: it.Label})
		if len(cur) >= cols || i == len(items)-1 {
			kb = append(kb, cur)
			cur = nil
		}
	}
	if len(cur) > 0 {
		kb = append(kb, cur)
	}
	return &models.ReplyKeyboardMarkup{
		Keyboard:       kb,
		ResizeKeyboard: true,
		IsPersistent:   true,
	}
}

// FindEnabledRootByLabel returns the first enabled root item whose label matches.
func FindEnabledRootByLabel(tree domain.MenuTree, label string) (domain.MenuNode, bool) {
	for _, it := range tree.Items {
		if it.Enabled && it.Label == label {
			return it, true
		}
	}
	return domain.MenuNode{}, false
}

func findNodeByID(nodes []domain.MenuNode, id string) (domain.MenuNode, bool) {
	for _, n := range nodes {
		if n.ID == id {
			return n, true
		}
		if child, ok := findNodeByID(n.Children, id); ok {
			return child, true
		}
	}
	return domain.MenuNode{}, false
}

func (a *Adapter) loadMenu(ctx context.Context) domain.MenuTree {
	if a != nil && a.Menu != nil {
		tree, err := a.Menu.GetMenu(ctx)
		if err == nil {
			return tree
		}
		slog.Error("menu load failed; using default seed", "err", err)
	}
	return domain.DefaultSeedTree(a.ChannelID)
}
