package tg

import (
	"context"
	"log/slog"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

// BotMessenger sends via go-telegram/bot.
type BotMessenger struct {
	Bot *bot.Bot
}

func (m *BotMessenger) SendText(ctx context.Context, chatID int64, text string) error {
	_, err := m.Bot.SendMessage(ctx, &bot.SendMessageParams{ChatID: chatID, Text: text})
	return err
}

func (m *BotMessenger) SendPhoto(ctx context.Context, chatID int64, ref sharedkernel.BlobRef, caption string) error {
	// Phase1: send caption + blob key as text placeholder when photo bytes not loaded here.
	_, err := m.Bot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   caption + "\n(blob: " + ref.Key + ")",
	})
	return err
}

// RegisterHandlers wires default message handler to Adapter.HandleText.
func RegisterHandlers(b *bot.Bot, ad *Adapter) {
	b.RegisterHandlerMatchFunc(func(update *models.Update) bool {
		return update.Message != nil && update.Message.Text != ""
	}, func(ctx context.Context, _ *bot.Bot, update *models.Update) {
		chatID := update.Message.Chat.ID
		if err := ad.HandleText(ctx, chatID, update.Message.Text); err != nil {
			slog.Error("tg handle text", "err", err, "chat_id", chatID)
		}
	})
}
