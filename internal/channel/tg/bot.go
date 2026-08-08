package tg

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path"
	"strings"
	"time"

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
	name := photoUploadName(ref.Key)
	_, err = m.Bot.SendPhoto(ctx, &bot.SendPhotoParams{
		ChatID:  chatID,
		Caption: caption,
		Photo:   &models.InputFileUpload{Filename: name, Data: bytesReader(data)},
	})
	return err
}

// photoUploadName returns a Telegram-safe upload basename (no path separators).
func photoUploadName(key string) string {
	key = strings.ReplaceAll(key, "\\", "/")
	name := path.Base(key)
	if name == "" || name == "." || name == "/" {
		return "result.png"
	}
	return name
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

func telegramDownloader(b *bot.Bot) FileDownloader {
	return func(ctx context.Context, fileID string) ([]byte, error) {
		f, err := b.GetFile(ctx, &bot.GetFileParams{FileID: fileID})
		if err != nil {
			return nil, err
		}
		link := b.FileDownloadLink(f)
		return fetchTelegramFileBytes(ctx, telegramFileHTTPClient, link)
	}
}

const maxTelegramFileBytes = 20 << 20 // 20 MiB

var telegramFileHTTPClient = &http.Client{Timeout: 30 * time.Second}

func fetchTelegramFileBytes(ctx context.Context, client *http.Client, url string) ([]byte, error) {
	if client == nil {
		client = telegramFileHTTPClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("telegram file download: status %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxTelegramFileBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxTelegramFileBytes {
		return nil, fmt.Errorf("telegram file download: body exceeds %d bytes", maxTelegramFileBytes)
	}
	return data, nil
}

func isImageMIME(mime string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(mime)), "image/")
}

func isImageDocument(d *models.Document) bool {
	if d == nil {
		return false
	}
	if isImageMIME(d.MimeType) {
		return true
	}
	name := strings.ToLower(d.FileName)
	for _, ext := range []string{".png", ".jpg", ".jpeg", ".webp", ".gif"} {
		if strings.HasSuffix(name, ext) {
			return true
		}
	}
	return false
}

func documentMIME(d *models.Document) string {
	if d == nil {
		return "image/jpeg"
	}
	if d.MimeType != "" {
		return d.MimeType
	}
	name := strings.ToLower(d.FileName)
	switch {
	case strings.HasSuffix(name, ".png"):
		return "image/png"
	case strings.HasSuffix(name, ".webp"):
		return "image/webp"
	case strings.HasSuffix(name, ".gif"):
		return "image/gif"
	default:
		return "image/jpeg"
	}
}

// RegisterHandlers wires message + callback handlers.
func RegisterHandlers(b *bot.Bot, ad *Adapter) {
	if ad.Download == nil {
		ad.Download = telegramDownloader(b)
	}
	b.RegisterHandlerMatchFunc(func(update *models.Update) bool {
		return update.Message != nil && update.Message.Text != ""
	}, func(ctx context.Context, _ *bot.Bot, update *models.Update) {
		chatID := update.Message.Chat.ID
		if err := ad.HandleText(ctx, chatID, update.Message.Text); err != nil {
			slog.Error("tg handle text", "err", err, "chat_id", chatID)
		}
	})
	b.RegisterHandlerMatchFunc(func(update *models.Update) bool {
		return update.Message != nil && len(update.Message.Photo) > 0
	}, func(ctx context.Context, _ *bot.Bot, update *models.Update) {
		chatID := update.Message.Chat.ID
		photos := update.Message.Photo
		best := photos[len(photos)-1]
		if err := ad.HandleUserMedia(ctx, chatID, best.FileID, "image/jpeg"); err != nil {
			slog.Error("tg handle photo", "err", err, "chat_id", chatID)
		}
	})
	b.RegisterHandlerMatchFunc(func(update *models.Update) bool {
		return update.Message != nil && isImageDocument(update.Message.Document)
	}, func(ctx context.Context, _ *bot.Bot, update *models.Update) {
		chatID := update.Message.Chat.ID
		doc := update.Message.Document
		if err := ad.HandleUserMedia(ctx, chatID, doc.FileID, documentMIME(doc)); err != nil {
			slog.Error("tg handle document", "err", err, "chat_id", chatID)
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
