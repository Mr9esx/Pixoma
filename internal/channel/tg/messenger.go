package tg

import (
	"context"
	"io"
	"log/slog"
	"strconv"
	"strings"

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
	return sendUploadByMIME(ctx, m.Bot, chatID, ref.MIME, name, data, caption, buttons)
}

func (m *BotMessenger) SendMediaURL(ctx context.Context, addr sharedkernel.ChannelAddr, imageURL, mime, caption string, buttons [][]ports.Button) error {
	chatID, err := externalChatID(addr)
	if err != nil {
		return err
	}
	effective := mime
	if effective == "" {
		effective = mimeFromURLExt(imageURL)
	}
	return sendURLByMIME(ctx, m.Bot, chatID, effective, imageURL, caption, buttons)
}

// sendUploadByMIME 路由图片 / 视频 / 动图 / 其它 blob 上传到合适的 Telegram API。
func sendUploadByMIME(ctx context.Context, b *bot.Bot, chatID int64, mime, name string, data []byte, caption string, buttons [][]ports.Button) error {
	truncated := truncateTGText(caption, maxTGCaptionRunes)
	markup := optionalInlineMarkup(buttons)
	switch {
	case strings.HasPrefix(mime, "image/gif"):
		_, err := b.SendAnimation(ctx, &bot.SendAnimationParams{
			ChatID:      chatID,
			Caption:     truncated,
			Animation:   &models.InputFileUpload{Filename: name, Data: bytesReader(data)},
			ReplyMarkup: markup,
		})
		return err
	case strings.HasPrefix(mime, "image/"):
		_, err := b.SendPhoto(ctx, &bot.SendPhotoParams{
			ChatID:      chatID,
			Caption:     truncated,
			Photo:       &models.InputFileUpload{Filename: name, Data: bytesReader(data)},
			ReplyMarkup: markup,
		})
		return err
	case strings.HasPrefix(mime, "video/"):
		_, err := b.SendVideo(ctx, &bot.SendVideoParams{
			ChatID:      chatID,
			Caption:     truncated,
			Video:       &models.InputFileUpload{Filename: name, Data: bytesReader(data)},
			ReplyMarkup: markup,
		})
		return err
	default:
		_, err := b.SendDocument(ctx, &bot.SendDocumentParams{
			ChatID:      chatID,
			Caption:     truncated,
			Document:    &models.InputFileUpload{Filename: name, Data: bytesReader(data)},
			ReplyMarkup: markup,
		})
		return err
	}
}

// sendURLByMIME 路由外链 URL 到合适的 Telegram API；mime 为空时回退图片。
func sendURLByMIME(ctx context.Context, b *bot.Bot, chatID int64, mime, url, caption string, buttons [][]ports.Button) error {
	truncated := truncateTGText(caption, maxTGCaptionRunes)
	markup := optionalInlineMarkup(buttons)
	switch {
	case strings.HasPrefix(mime, "image/gif"):
		_, err := b.SendAnimation(ctx, &bot.SendAnimationParams{
			ChatID:      chatID,
			Caption:     truncated,
			Animation:   &models.InputFileString{Data: url},
			ReplyMarkup: markup,
		})
		return err
	case strings.HasPrefix(mime, "video/"):
		_, err := b.SendVideo(ctx, &bot.SendVideoParams{
			ChatID:      chatID,
			Caption:     truncated,
			Video:       &models.InputFileString{Data: url},
			ReplyMarkup: markup,
		})
		return err
	case strings.HasPrefix(mime, "image/") || mime == "":
		_, err := b.SendPhoto(ctx, &bot.SendPhotoParams{
			ChatID:      chatID,
			Caption:     truncated,
			Photo:       &models.InputFileString{Data: url},
			ReplyMarkup: markup,
		})
		return err
	default:
		_, err := b.SendDocument(ctx, &bot.SendDocumentParams{
			ChatID:      chatID,
			Caption:     truncated,
			Document:    &models.InputFileString{Data: url},
			ReplyMarkup: markup,
		})
		return err
	}
}

// mimeFromURLExt 从 URL 路径末段扩展名推断 mime（兜底用）。
// 已知后缀：图片（png/jpg/jpeg/webp/gif）、视频（mp4/webm/mov）。
func mimeFromURLExt(rawURL string) string {
	u := rawURL
	if i := strings.Index(u, "?"); i >= 0 {
		u = u[:i]
	}
	dot := strings.LastIndex(u, ".")
	if dot < 0 {
		return ""
	}
	ext := strings.ToLower(u[dot:])
	switch ext {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	case ".mp4":
		return "video/mp4"
	case ".webm":
		return "video/webm"
	case ".mov":
		return "video/quicktime"
	}
	return ""
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
