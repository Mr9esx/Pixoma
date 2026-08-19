package tg

import (
	"context"

	"github.com/mr9esx/comfyui_tgbot/internal/menu/domain"
)

// ExtrasReader is deprecated (platform differences move to capability render
// declarations); kept for API compatibility during migration.
type ExtrasReader interface {
	GetExtras(ctx context.Context) (map[string][]domain.Extra, error)
}

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
