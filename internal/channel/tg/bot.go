package tg

import (
	"context"
	"io"
	"log/slog"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/blob"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

// BotMessenger sends via go-telegram/bot.
type BotMessenger struct {
	Bot  *bot.Bot
	Blob blob.Store
}

func (m *BotMessenger) SendText(ctx context.Context, chatID int64, text string) error {
	_, err := m.Bot.SendMessage(ctx, &bot.SendMessageParams{ChatID: chatID, Text: text})
	return err
}

func (m *BotMessenger) SendMenu(ctx context.Context, chatID int64, text string) error {
	kb := mainReplyKeyboard()
	_, err := m.Bot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        text,
		ReplyMarkup: kb,
	})
	return err
}

func (m *BotMessenger) SendInline(ctx context.Context, chatID int64, text string, rows [][]InlineButton) error {
	_, err := m.Bot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        text,
		ReplyMarkup: toInlineMarkup(rows),
	})
	return err
}

func (m *BotMessenger) SendPhoto(ctx context.Context, chatID int64, ref sharedkernel.BlobRef, caption string) error {
	if m.Blob == nil {
		_, err := m.Bot.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   caption + "\n(blob: " + ref.Key + ")",
		})
		return err
	}
	rc, err := m.Blob.Get(ctx, ref)
	if err != nil {
		return err
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil {
		return err
	}
	name := "result.png"
	if ref.Key != "" {
		name = ref.Key
	}
	_, err = m.Bot.SendPhoto(ctx, &bot.SendPhotoParams{
		ChatID:  chatID,
		Caption: caption,
		Photo:   &models.InputFileUpload{Filename: name, Data: bytesReader(data)},
	})
	return err
}

func (m *BotMessenger) AnswerCallback(ctx context.Context, callbackID, text string) error {
	if callbackID == "" {
		return nil
	}
	_, err := m.Bot.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: callbackID,
		Text:            text,
	})
	return err
}

func mainReplyKeyboard() *models.ReplyKeyboardMarkup {
	rows := MainMenuRows()
	kb := make([][]models.KeyboardButton, 0, len(rows))
	for _, row := range rows {
		btns := make([]models.KeyboardButton, 0, len(row))
		for _, t := range row {
			btns = append(btns, models.KeyboardButton{Text: t})
		}
		kb = append(kb, btns)
	}
	return &models.ReplyKeyboardMarkup{
		Keyboard:       kb,
		ResizeKeyboard: true,
		IsPersistent:   true,
	}
}

func toInlineMarkup(rows [][]InlineButton) *models.InlineKeyboardMarkup {
	out := make([][]models.InlineKeyboardButton, 0, len(rows))
	for _, row := range rows {
		btns := make([]models.InlineKeyboardButton, 0, len(row))
		for _, b := range row {
			btns = append(btns, models.InlineKeyboardButton{Text: b.Text, CallbackData: b.Data})
		}
		out = append(out, btns)
	}
	return &models.InlineKeyboardMarkup{InlineKeyboard: out}
}

type byteReader struct {
	b []byte
	i int
}

func bytesReader(b []byte) *byteReader { return &byteReader{b: b} }

func (r *byteReader) Read(p []byte) (int, error) {
	if r.i >= len(r.b) {
		return 0, io.EOF
	}
	n := copy(p, r.b[r.i:])
	r.i += n
	return n, nil
}

// RegisterHandlers wires message + callback handlers.
func RegisterHandlers(b *bot.Bot, ad *Adapter) {
	b.RegisterHandlerMatchFunc(func(update *models.Update) bool {
		return update.Message != nil && update.Message.Text != ""
	}, func(ctx context.Context, _ *bot.Bot, update *models.Update) {
		chatID := update.Message.Chat.ID
		if err := ad.HandleText(ctx, chatID, update.Message.Text); err != nil {
			slog.Error("tg handle text", "err", err, "chat_id", chatID)
		}
	})
	b.RegisterHandlerMatchFunc(func(update *models.Update) bool {
		return update.CallbackQuery != nil
	}, func(ctx context.Context, _ *bot.Bot, update *models.Update) {
		cq := update.CallbackQuery
		chatID := cq.From.ID
		if cq.Message.Message != nil {
			chatID = cq.Message.Message.Chat.ID
		}
		if err := ad.HandleCallback(ctx, chatID, cq.ID, cq.Data); err != nil {
			slog.Error("tg handle callback", "err", err, "chat_id", chatID)
		}
	})
}
