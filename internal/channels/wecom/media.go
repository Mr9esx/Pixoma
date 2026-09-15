package wecom

import (
	"context"
	"fmt"
	"net/http"
	"time"

	aibot "github.com/seastart/wecom-aibot-go"
)

const maxMediaBytes = 20 << 20

// MediaBridge validates plaintext returned by the protocol client's AES media
// download path. Decryption stays inside the protocol wrapper.
type MediaBridge struct {
	DownloadPlaintext func(ctx context.Context, url, aesKey string) ([]byte, error)
}

// NewMediaBridge uses the SDK's authenticated AES-decryption path with a
// bounded HTTP client; callers still receive the shared plaintext size guard.
func NewMediaBridge() MediaBridge {
	client := &http.Client{Timeout: 30 * time.Second}
	return MediaBridge{DownloadPlaintext: func(ctx context.Context, url, aesKey string) ([]byte, error) {
		file, err := aibot.DownloadFileWithClient(ctx, client, url, aesKey)
		if err != nil {
			return nil, err
		}
		return file.Buffer, nil
	}}
}

func (m MediaBridge) Download(ctx context.Context, url, aesKey string) ([]byte, error) {
	if m.DownloadPlaintext == nil {
		return nil, fmt.Errorf("wecom media bridge: downloader not configured")
	}
	data, err := m.DownloadPlaintext(ctx, url, aesKey)
	if err != nil {
		return nil, err
	}
	if len(data) > maxMediaBytes {
		return nil, fmt.Errorf("wecom media bridge: plaintext exceeds %d bytes", maxMediaBytes)
	}
	return data, nil
}
