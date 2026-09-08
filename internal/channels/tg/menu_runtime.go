package tg

import (
	"context"
	"log/slog"
	"sync"

	"github.com/go-telegram/bot/models"

	mcdomain "github.com/Mr9esx/Pixoma/internal/menus/domain"
)

// MenuReader is the read port for a channel's menu tree.
type MenuReader interface {
	GetTree(ctx context.Context) (mcdomain.MenuTree, error)
}

// DefaultMainMenu is the fallback menu when none is configured.
func DefaultMainMenu() mcdomain.MenuTree {
	return mcdomain.DefaultMenuTree("default")
}

// BuildReplyKeyboard builds a ReplyKeyboard from the compiled tree.
func BuildReplyKeyboard(menu mcdomain.MenuTree) *models.ReplyKeyboardMarkup {
	compiled := mcdomain.Compile(menu)
	layout := make([][]models.KeyboardButton, 0, len(compiled.Rows))
	for _, row := range compiled.Rows {
		cells := make([]models.KeyboardButton, 0, len(row))
		for _, it := range row {
			cells = append(cells, models.KeyboardButton{Text: it.Label})
		}
		layout = append(layout, cells)
	}
	return &models.ReplyKeyboardMarkup{
		Keyboard:       layout,
		ResizeKeyboard: true,
		IsPersistent:   true,
	}
}

// FindEnabledItemByLabel returns the first root button whose label matches.
func FindEnabledItemByLabel(menu mcdomain.MenuTree, label string) (mcdomain.TreeButton, bool) {
	for _, it := range menu.Items {
		if it.Label == label {
			return it, true
		}
	}
	return mcdomain.TreeButton{}, false
}

func (a *Adapter) loadMenu(ctx context.Context) mcdomain.MenuTree {
	if a != nil && a.Menu != nil {
		menu, err := a.Menu.GetTree(ctx)
		if err == nil {
			return menu
		}
		slog.Error("menu load failed; using default", "err", err)
	}
	return DefaultMainMenu()
}

// backStack records the source chain per chat so auto "back" buttons can
// return through nested cards to the main menu.
type backStack struct {
	mu     sync.Mutex
	stacks map[string][]string
}

func newBackStack() *backStack {
	return &backStack{stacks: map[string][]string{}}
}

func (b *backStack) push(chat string, id string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.stacks[chat] = append(b.stacks[chat], id)
}

func (b *backStack) pop(chat string) (string, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	s := b.stacks[chat]
	if len(s) == 0 {
		return "", false
	}
	last := s[len(s)-1]
	b.stacks[chat] = s[:len(s)-1]
	return last, true
}

func (b *backStack) top(chat string) (string, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	s := b.stacks[chat]
	if len(s) == 0 {
		return "", false
	}
	return s[len(s)-1], true
}

func (b *backStack) clear(chat string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.stacks, chat)
}
