package domain

import (
	"os"
	"strings"

	"github.com/Mr9esx/Pixoma/internal/platform/botconfig"
)

// ApplyEnv overlays emergency env overrides onto saved settings.
func ApplyEnv(s *Settings) {
	if s == nil {
		return
	}
	if v := strings.TrimSpace(os.Getenv("BLOB_DRIVER")); v != "" {
		s.BlobDriver = v
	}
	if v := strings.TrimSpace(os.Getenv("COMFYUI_BASE_URL")); v != "" {
		s.ComfyUIBaseURL = v
	}
	if v := strings.TrimSpace(os.Getenv("S3_ACCESS_KEY")); v != "" {
		s.BlobAccessKey = v
	}
	if v := strings.TrimSpace(os.Getenv("S3_SECRET_KEY")); v != "" {
		s.BlobSecretKey = v
	}
	if v := strings.TrimSpace(os.Getenv("TOS_ACCESS_KEY")); v != "" && s.BlobDriver == botconfig.BlobDriverTOS {
		s.BlobAccessKey = v
	}
	if v := strings.TrimSpace(os.Getenv("TOS_SECRET_KEY")); v != "" && s.BlobDriver == botconfig.BlobDriverTOS {
		s.BlobSecretKey = v
	}
}
