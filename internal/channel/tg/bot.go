package tg

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	identitydomain "github.com/mr9esx/comfyui_tgbot/internal/identity/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

// FileDownloader fetches Telegram file bytes by file_id (injectable for tests).
type FileDownloader func(ctx context.Context, fileID string) ([]byte, error)

// RegisterHandlers wires message + callback handlers to the bot instance.
func RegisterHandlers(b *bot.Bot, ad *Adapter) {
	if ad == nil || ad.Media == nil {
		return
	}
	if mb, ok := ad.Media.(*tgMediaBridge); ok && mb.dl == nil {
		mb.dl = telegramDownloader(b)
	}
	b.RegisterHandlerMatchFunc(func(update *models.Update) bool {
		return update.Message != nil && update.Message.Text != ""
	}, func(ctx context.Context, _ *bot.Bot, update *models.Update) {
		userID := resolveUser(ctx, ad, update.Message.From)
		chatID := formatChatID(ad, update.Message.Chat.ID)
		if err := ad.HandleText(ctx, chatID, update.Message.Text, userID); err != nil {
			slog.Error("tg handle text", "err", err, "chat_id", chatID)
		}
	})
	b.RegisterHandlerMatchFunc(func(update *models.Update) bool {
		return update.Message != nil && len(update.Message.Photo) > 0
	}, func(ctx context.Context, _ *bot.Bot, update *models.Update) {
		_ = resolveUser(ctx, ad, update.Message.From)
		chatID := formatChatID(ad, update.Message.Chat.ID)
		photos := update.Message.Photo
		best := photos[len(photos)-1]
		if err := ad.HandleUserMedia(ctx, chatID, best.FileID, "image/jpeg"); err != nil {
			slog.Error("tg handle photo", "err", err, "chat_id", chatID)
		}
	})
	b.RegisterHandlerMatchFunc(func(update *models.Update) bool {
		return update.Message != nil && isImageDocument(update.Message.Document)
	}, func(ctx context.Context, _ *bot.Bot, update *models.Update) {
		_ = resolveUser(ctx, ad, update.Message.From)
		chatID := formatChatID(ad, update.Message.Chat.ID)
		doc := update.Message.Document
		if err := ad.HandleUserMedia(ctx, chatID, doc.FileID, documentMIME(doc)); err != nil {
			slog.Error("tg handle document", "err", err, "chat_id", chatID)
		}
	})
	b.RegisterHandlerMatchFunc(func(update *models.Update) bool {
		return update.CallbackQuery != nil
	}, func(ctx context.Context, _ *bot.Bot, update *models.Update) {
		cq := update.CallbackQuery
		answer := func(ctx context.Context, callbackID string) error {
			_, err := b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: callbackID})
			return err
		}
		if err := handleCallbackUpdate(ctx, ad, cq, answer); err != nil {
			slog.Error("tg handle callback", "err", err)
		}
	})
}

func handleCallbackUpdate(ctx context.Context, ad *Adapter, cq *models.CallbackQuery, answerCallback func(context.Context, string) error) error {
	if answerCallback != nil {
		if err := answerCallback(ctx, cq.ID); err != nil {
			slog.Error("tg answer callback", "err", err, "callback_id", cq.ID)
		}
	}
	userID := resolveUser(ctx, ad, &cq.From)
	chatID := cq.From.ID
	if cq.Message.Message != nil {
		chatID = cq.Message.Message.Chat.ID
	}
	return ad.HandleCallback(ctx, formatChatID(ad, chatID), cq.ID, cq.Data, userID)
}

func formatChatID(ad *Adapter, chatID int64) sharedkernel.ChatID {
	return sharedkernel.ChatID(sharedkernel.FormatChatID(sharedkernel.ChannelAddr{
		ChannelID:      ad.ChannelID,
		ExternalChatID: strconv.FormatInt(chatID, 10),
	}))
}

func resolveUser(ctx context.Context, ad *Adapter, from *models.User) string {
	if ad == nil || ad.Users == nil || from == nil {
		return ""
	}
	isBot := from.IsBot
	isPremium := from.IsPremium
	profile, err := json.Marshal(map[string]any{
		"is_bot":     isBot,
		"is_premium": isPremium,
	})
	if err != nil {
		profile = nil
	}
	externalID := strconv.FormatInt(from.ID, 10)
	id, err := ad.Users.Resolve(ctx, sharedkernel.ChannelAddr{
		ChannelID:      ad.ChannelID,
		ExternalChatID: externalID,
	}, identitydomain.UpsertFrom{
		ChannelID:      ad.ChannelID,
		ExternalUserID: externalID,
		Username:       from.Username,
		FirstName:      from.FirstName,
		LastName:       from.LastName,
		LanguageCode:   from.LanguageCode,
		ProfileJSON:    string(profile),
	})
	if err != nil {
		slog.Error("tg resolve user", "err", err, "tg_user_id", from.ID)
		return ""
	}
	return id
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

// photoUploadName returns a Telegram-safe upload basename (no path separators).
func photoUploadName(key string) string {
	key = strings.ReplaceAll(key, "\\", "/")
	name := path.Base(key)
	if name == "" || name == "." || name == "/" {
		return "result.png"
	}
	return name
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
