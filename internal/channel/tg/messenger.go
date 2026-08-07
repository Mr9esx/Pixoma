package tg

import (
	"context"

	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

type InlineButton struct {
	Text string
	Data string
}

// Messenger abstracts Telegram sends so unit tests need no Bot token.
type Messenger interface {
	SendText(ctx context.Context, chatID int64, text string) error
	SendMenu(ctx context.Context, chatID int64, text string) error
	SendInline(ctx context.Context, chatID int64, text string, rows [][]InlineButton) error
	SendPhoto(ctx context.Context, chatID int64, blob sharedkernel.BlobRef, caption string) error
	AnswerCallback(ctx context.Context, callbackID, text string) error
}
