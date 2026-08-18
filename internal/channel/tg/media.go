package tg

import (
	"context"
	"fmt"
)

// tgMediaBridge implements ports.MediaBridge with Telegram file download.
type tgMediaBridge struct {
	dl FileDownloader
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
