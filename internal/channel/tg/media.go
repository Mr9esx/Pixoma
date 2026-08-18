package tg

import (
	"context"
	"fmt"

	"github.com/go-telegram/bot"

	"github.com/mr9esx/comfyui_tgbot/internal/channel/ports"
)

// tgMediaBridge implements ports.MediaBridge with Telegram file download.
type tgMediaBridge struct {
	dl FileDownloader
}

// NewMediaBridge returns a MediaBridge backed by Telegram file download.
func NewMediaBridge(b *bot.Bot) ports.MediaBridge {
	return &tgMediaBridge{dl: telegramDownloader(b)}
}

func (m *tgMediaBridge) Download(ctx context.Context, externalFileID, mime string) ([]byte, error) {
	if m == nil || m.dl == nil {
		return nil, fmt.Errorf("tg media bridge: downloader not configured")
	}
	return m.dl(ctx, externalFileID)
}

func (m *tgMediaBridge) Upload(context.Context, []byte, string) (string, error) {
	return "", fmt.Errorf("tg media bridge: upload not implemented")
}
