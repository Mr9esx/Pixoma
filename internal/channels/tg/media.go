package tg

import (
	"context"
	"fmt"

	"github.com/go-telegram/bot"

	protocol "github.com/Mr9esx/Pixoma/internal/channels/protocol"
)

// tgMediaBridge implements protocol.MediaBridge with Telegram file download.
type tgMediaBridge struct {
	dl FileDownloader
}

// NewMediaBridge returns a MediaBridge backed by Telegram file download.
func NewMediaBridge(b *bot.Bot) protocol.MediaBridge {
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
