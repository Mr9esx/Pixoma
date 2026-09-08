package config

import (
	"os"
	"strings"

	"github.com/Mr9esx/Pixoma/internal/platform/botconfig"
	settingsdomain "github.com/Mr9esx/Pixoma/internal/settings/domain"
)

// ApplyBlobEnv copies wizard blob connection fields into process env so factory
// and spawned Edge children can read them.
func ApplyBlobEnv(cfg settingsdomain.Settings) {
	setIfEmpty := func(key, val string) {
		if strings.TrimSpace(os.Getenv(key)) != "" || strings.TrimSpace(val) == "" {
			return
		}
		_ = os.Setenv(key, val)
	}
	switch strings.TrimSpace(cfg.BlobDriver) {
	case botconfig.BlobDriverTOS:
		setIfEmpty("TOS_ENDPOINT", cfg.BlobEndpoint)
		setIfEmpty("TOS_REGION", cfg.BlobRegion)
		setIfEmpty("TOS_BUCKET", cfg.BlobBucket)
		setIfEmpty("TOS_ACCESS_KEY", cfg.BlobAccessKey)
		setIfEmpty("TOS_SECRET_KEY", cfg.BlobSecretKey)
	case botconfig.BlobDriverS3:
		setIfEmpty("S3_ENDPOINT", cfg.BlobEndpoint)
		setIfEmpty("S3_REGION", cfg.BlobRegion)
		setIfEmpty("S3_BUCKET", cfg.BlobBucket)
		setIfEmpty("S3_ACCESS_KEY", cfg.BlobAccessKey)
		setIfEmpty("S3_SECRET_KEY", cfg.BlobSecretKey)
	}
}
