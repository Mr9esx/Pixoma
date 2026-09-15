package wecom

import (
	"context"
	"strings"
	"testing"
)

func TestMediaBridgeDownloadRejectsOversizedPlaintext(t *testing.T) {
	bridge := MediaBridge{DownloadPlaintext: func(context.Context, string, string) ([]byte, error) {
		return []byte(strings.Repeat("a", maxMediaBytes+1)), nil
	}}
	if _, err := bridge.Download(context.Background(), "https://example.invalid/file", "aes-key"); err == nil {
		t.Fatal("want oversized plaintext error")
	}
}
