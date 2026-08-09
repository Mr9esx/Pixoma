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
	GetMenu(ctx context.Context) (tgmenudomain.MenuDocument, error)
}

// BuildReplyKeyboard builds a persistent ReplyKeyboard from enabled items ordered by (row, col).
func BuildReplyKeyboard(doc tgmenudomain.MenuDocument) *models.ReplyKeyboardMarkup {
	items := make([]tgmenudomain.MenuItem, 0, len(doc.Items))
	for _, it := range doc.Items {
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

// FindEnabledByLabel returns the first enabled item whose label matches.
func FindEnabledByLabel(doc tgmenudomain.MenuDocument, label string) (tgmenudomain.MenuItem, bool) {
	for _, it := range doc.Items {
		if it.Enabled && it.Label == label {
			return it, true
		}
	}
	return tgmenudomain.MenuItem{}, false
}

func (a *Adapter) loadMenu(ctx context.Context) tgmenudomain.MenuDocument {
	if a != nil && a.Menu != nil {
		doc, err := a.Menu.GetMenu(ctx)
		if err == nil {
			return doc
		}
		slog.Error("tg menu load failed; using default seed", "err", err)
	}
	return tgmenudomain.DefaultSeed()
}
