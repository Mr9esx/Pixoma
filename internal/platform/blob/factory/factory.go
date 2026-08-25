package factory

import (
	"fmt"
	"os"
	"strings"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/blob"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/blob/localfs"
	blobs3 "github.com/mr9esx/comfyui_tgbot/internal/platform/blob/s3"
	blobtos "github.com/mr9esx/comfyui_tgbot/internal/platform/blob/tos"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/botconfig"
)

// New builds a blob.Store for the given driver (credentials from environment).
// Prefer NewFromConfig when YAML BlobConfig (e.g. tos endpoint/region/bucket) is available.
func New(driver, localRoot string) (blob.Store, error) {
	return NewFromConfig(botconfig.Config{
		Blob: botconfig.BlobConfig{Driver: driver},
	}, localRoot)
}

// NewFromConfig builds a blob.Store from botconfig.
// localRoot is used only for localfs. TOS non-secret fields prefer cfg.Blob.TOS, then TOS_* env.
// S3/TOS access keys always come from environment.
func NewFromConfig(cfg botconfig.Config, localRoot string) (blob.Store, error) {
	switch strings.TrimSpace(cfg.Blob.Driver) {
	case "", botconfig.BlobDriverLocalFS, botconfig.BlobDriverSharedFS:
		return localfs.New(localRoot)
	case botconfig.BlobDriverS3:
		return blobs3.New(S3Options(cfg))
	case botconfig.BlobDriverTOS:
		return blobtos.New(blobtos.Options{
			Endpoint:        firstNonEmpty(cfg.Blob.TOS.Endpoint, os.Getenv("TOS_ENDPOINT")),
			Region:          firstNonEmpty(cfg.Blob.TOS.Region, os.Getenv("TOS_REGION")),
			Bucket:          firstNonEmpty(cfg.Blob.TOS.Bucket, os.Getenv("TOS_BUCKET")),
			AccessKeyID:     os.Getenv("TOS_ACCESS_KEY"),
			SecretAccessKey: os.Getenv("TOS_SECRET_KEY"),
		})
	default:
		return nil, fmt.Errorf("blob/factory: unknown driver %q", cfg.Blob.Driver)
	}
}

// S3Options resolves wizard/YAML fields first, then environment, then defaults.
func S3Options(cfg botconfig.Config) blobs3.Options {
	return blobs3.Options{
		Endpoint:        firstNonEmpty(cfg.Blob.S3.Endpoint, os.Getenv("S3_ENDPOINT")),
		Region:          firstNonEmpty(cfg.Blob.S3.Region, envOr("S3_REGION", "us-east-1")),
		Bucket:          firstNonEmpty(cfg.Blob.S3.Bucket, envOr("S3_BUCKET", "pixoma")),
		AccessKeyID:     os.Getenv("S3_ACCESS_KEY"),
		SecretAccessKey: os.Getenv("S3_SECRET_KEY"),
		UsePathStyle:    envBool("S3_PATH_STYLE", true),
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}

func envOr(k, def string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return def
}

func envBool(k string, def bool) bool {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		return def
	}
	switch strings.ToLower(v) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return def
	}
}
