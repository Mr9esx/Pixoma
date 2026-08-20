package tg

import (
	"context"
	"log/slog"
	"sync"

	"github.com/go-telegram/bot/models"

	mcdomain "github.com/mr9esx/comfyui_tgbot/internal/menucard/domain"
)

// MenuReader is the read port for a channel's main menu.
type MenuReader interface {
	GetMenu(ctx context.Context) (mcdomain.Menu, error)
}

// CardProvider loads cards by id at runtime.
type CardProvider interface {
	GetCard(ctx context.Context, channelID, cardID string) (mcdomain.Card, error)
}

// DefaultMainMenu is the fallback menu when none is configured.
func DefaultMainMenu() mcdomain.Menu {
	return mcdomain.DefaultMenu()
}

// BuildReplyKeyboard builds a ReplyKeyboard from the main menu model.
// Columns are free-form (1-8); there is no button-count limit.
func BuildReplyKeyboard(menu mcdomain.Menu) *models.ReplyKeyboardMarkup {
	cols := menu.Columns
	if cols < 1 || cols > 8 {
		cols = 2
	}
	layout := make([][]models.KeyboardButton, 0)
	var row []models.KeyboardButton
	for _, it := range menu.Items {
		row = append(row, models.KeyboardButton{Text: it.Label})
		if len(row) >= cols {
			layout = append(layout, row)
			row = nil
		}
	}
	if len(row) > 0 {
		layout = append(layout, row)
	}
	return &models.ReplyKeyboardMarkup{
		Keyboard:       layout,
		ResizeKeyboard: true,
		IsPersistent:   true,
	}
}

// FindEnabledItemByLabel returns the first menu item whose label matches.
func FindEnabledItemByLabel(menu mcdomain.Menu, label string) (mcdomain.MenuItem, bool) {
	for _, it := range menu.Items {
		if it.Label == label {
			return it, true
		}
	}
	return mcdomain.MenuItem{}, false
}

func (a *Adapter) loadMenu(ctx context.Context) mcdomain.Menu {
	if a != nil && a.Menu != nil {
		menu, err := a.Menu.GetMenu(ctx)
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
