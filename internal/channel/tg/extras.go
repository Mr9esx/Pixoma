package tg

import (
	"context"
	"encoding/json"

	"github.com/mr9esx/comfyui_tgbot/internal/menu/domain"
)

// ExtrasReader provides platform extras for the bound channel's menu.
type ExtrasReader interface {
	GetExtras(ctx context.Context) (map[string][]domain.Extra, error)
}

type tgRootLayout struct {
	Columns int `json:"columns"`
}

// RootColumns returns the configured ReplyKeyboard column count for a root item
// (from tg_root_layout extras), defaulting to 2 when absent or invalid.
func RootColumns(extras map[string][]domain.Extra, itemID string) int {
	for _, extra := range extras[itemID] {
		if extra.ExtraType != "tg_root_layout" {
			continue
		}
		var layout tgRootLayout
		if err := json.Unmarshal([]byte(extra.ExtraJSON), &layout); err != nil {
			continue
		}
		if layout.Columns >= 1 && layout.Columns <= 8 {
			return layout.Columns
		}
	}
	return 2
}
