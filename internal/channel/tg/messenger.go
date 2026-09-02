package tg

import (
	"context"
	"io"
	"log/slog"
	"strconv"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"github.com/mr9esx/comfyui_tgbot/internal/channel/ports"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/blob"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

// BotMessenger renders ports.Outbound through the Telegram Bot API.
type BotMessenger struct {
	Bot  *bot.Bot
	Blob blob.Store
	Menu MenuReader
}

func (m *BotMessenger) SendText(ctx context.Context, addr sharedkernel.ChannelAddr, text string) error {
	chatID, err := externalChatID(addr)
	if err != nil {
		return err
	}
	_, err = m.Bot.SendMessage(ctx, &bot.SendMessageParams{ChatID: chatID, Text: truncateTGText(text, maxTGTextRunes)})
	return err
}

func (m *BotMessenger) SendMenu(ctx context.Context, addr sharedkernel.ChannelAddr, title string, _ []ports.MenuEntry) error {
	chatID, err := externalChatID(addr)
	if err != nil {
		return err
	}
	kb := m.replyKeyboard(ctx)
	_, err = m.Bot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        truncateTGText(title, maxTGTextRunes),
		ReplyMarkup: kb,
	})
	return err
}

func (m *BotMessenger) SendList(ctx context.Context, addr sharedkernel.ChannelAddr, title string, rows [][]ports.Button) error {
	chatID, err := externalChatID(addr)
	if err != nil {
		return err
	}
	_, err = m.Bot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        truncateTGText(title, maxTGTextRunes),
		ReplyMarkup: toInlineMarkup(rows),
	})
	return err
}

func (m *BotMessenger) SendMedia(ctx context.Context, addr sharedkernel.ChannelAddr, ref sharedkernel.BlobRef, caption string, buttons [][]ports.Button) error {
	chatID, err := externalChatID(addr)
	if err != nil {
		return err
	}
	if m.Blob == nil {
		_, err := m.Bot.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:      chatID,
			Text:        truncateTGText(caption+"\n(blob: "+ref.Key+")", maxTGTextRunes),
			ReplyMarkup: optionalInlineMarkup(buttons),
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
	name := photoUploadName(ref.Key)
	_, err = m.Bot.SendPhoto(ctx, &bot.SendPhotoParams{
		ChatID:      chatID,
		Caption:     truncateTGText(caption, maxTGCaptionRunes),
		Photo:       &models.InputFileUpload{Filename: name, Data: bytesReader(data)},
		ReplyMarkup: optionalInlineMarkup(buttons),
	})
	return err
}

func (m *BotMessenger) SendMediaURL(ctx context.Context, addr sharedkernel.ChannelAddr, imageURL, caption string, buttons [][]ports.Button) error {
	chatID, err := externalChatID(addr)
	if err != nil {
		return err
	}
	_, err = m.Bot.SendPhoto(ctx, &bot.SendPhotoParams{
		ChatID:      chatID,
		Caption:     truncateTGText(caption, maxTGCaptionRunes),
		Photo:       &models.InputFileString{Data: imageURL},
		ReplyMarkup: optionalInlineMarkup(buttons),
	})
	return err
}

func (m *BotMessenger) EditReplyMarkup(ctx context.Context, addr sharedkernel.ChannelAddr, messageID int, rows [][]ports.Button) error {
	chatID, err := externalChatID(addr)
	if err != nil {
		return err
	}
	markup := optionalInlineMarkup(rows)
	if markup == nil {
		markup = &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{}}
	}
	_, err = m.Bot.EditMessageReplyMarkup(ctx, &bot.EditMessageReplyMarkupParams{
		ChatID:      chatID,
		MessageID:   messageID,
		ReplyMarkup: markup,
	})
	return err
}

func (m *BotMessenger) replyKeyboard(ctx context.Context) *models.ReplyKeyboardMarkup {
	if m != nil && m.Menu != nil {
		menu, err := m.Menu.GetTree(ctx)
		if err == nil {
			return BuildReplyKeyboard(menu)
		}
		slog.Error("tg menu load for keyboard failed; using default", "err", err)
	}
	return BuildReplyKeyboard(DefaultMainMenu())
}

func optionalInlineMarkup(rows [][]ports.Button) models.ReplyMarkup {
	if len(rows) == 0 {
		return nil
	}
	return toInlineMarkup(rows)
}

func toInlineMarkup(rows [][]ports.Button) *models.InlineKeyboardMarkup {
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

func externalChatID(addr sharedkernel.ChannelAddr) (int64, error) {
	return strconv.ParseInt(addr.ExternalChatID, 10, 64)
}

const (
	maxTGTextRunes    = 4096
	maxTGCaptionRunes = 1024
)

func truncateTGText(s string, maxRunes int) string {
	r := []rune(s)
	if len(r) <= maxRunes {
		return s
	}
	if maxRunes <= 1 {
		return "…"
	}
	return string(r[:maxRunes-1]) + "…"
}
