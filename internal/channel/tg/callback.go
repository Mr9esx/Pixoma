package tg

import (
	"fmt"
	"strings"
)

// Callback data prefixes for menu navigation (capability buttons use inv:<token>).
const (
	CBInvoke   = "inv:" // inv:<token> → capability invoke
	CBMenu     = "mn"
	CBMenuBack = "mb:" // mb:root | mb:<card_id>
)

// navTarget is a menu-level navigation intent handled by the adapter.
type navTarget struct {
	kind string // main | back
	id   string
}

// TranslateMenuCallback converts menu navigation callback data.
func TranslateMenuCallback(data string) (navTarget, error) {
	switch {
	case data == CBMenu:
		return navTarget{kind: "main"}, nil
	case strings.HasPrefix(data, CBMenuBack):
		target := strings.TrimPrefix(data, CBMenuBack)
		if target == "root" {
			return navTarget{kind: "main"}, nil
		}
		if target == "" {
			return navTarget{}, fmt.Errorf("tg callback: empty back target")
		}
		return navTarget{kind: "back", id: target}, nil
	default:
		return navTarget{}, fmt.Errorf("tg callback: unknown nav %q", data)
	}
}
