package tg

import (
	"fmt"
	"strings"
)

// Callback data prefixes for menu navigation (capability buttons use inv:<token>).
const (
	CBInvoke     = "inv:" // inv:<token> → capability invoke
	CBMenu       = "mn"
	CBMenuFolder = "mf:" // mf:<menu_item_id>
	CBMenuBack   = "mb:" // mb:root | mb:<parent_item_id>
)

// navTarget is a menu-level navigation intent handled by the adapter.
type navTarget struct {
	kind string // main | group | back
	id   string
}

// TranslateMenuCallback converts menu navigation callback data.
func TranslateMenuCallback(data string) (navTarget, error) {
	switch {
	case data == CBMenu:
		return navTarget{kind: "main"}, nil
	case strings.HasPrefix(data, CBMenuFolder):
		id := strings.TrimPrefix(data, CBMenuFolder)
		if id == "" {
			return navTarget{}, fmt.Errorf("tg callback: empty folder")
		}
		return navTarget{kind: "group", id: id}, nil
	case strings.HasPrefix(data, CBMenuBack):
		target := strings.TrimPrefix(data, CBMenuBack)
		if target == "root" {
			return navTarget{kind: "main"}, nil
		}
		if target == "" {
			return navTarget{}, fmt.Errorf("tg callback: empty back target")
		}
		return navTarget{kind: "group", id: target}, nil
	default:
		return navTarget{}, fmt.Errorf("tg callback: unknown nav %q", data)
	}
}
