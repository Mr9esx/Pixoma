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

// New builds a blob.Store for the given driver.
// localRoot is used only for localfs; s3/tos read credentials from the environment.
func New(driver, localRoot string) (blob.Store, error) {
	switch strings.TrimSpace(driver) {
	case "", botconfig.BlobDriverLocalFS:
		return localfs.New(localRoot)
	case botconfig.BlobDriverS3:
		return blobs3.New(blobs3.Options{
			Endpoint:        os.Getenv("S3_ENDPOINT"),
			Region:          envOr("S3_REGION", "us-east-1"),
			Bucket:          envOr("S3_BUCKET", "pixoma"),
			AccessKeyID:     os.Getenv("S3_ACCESS_KEY"),
			SecretAccessKey: os.Getenv("S3_SECRET_KEY"),
			UsePathStyle:    envBool("S3_PATH_STYLE", true),
		})
	case botconfig.BlobDriverTOS:
		return blobtos.New(blobtos.Options{
			Endpoint:        os.Getenv("TOS_ENDPOINT"),
			Region:          os.Getenv("TOS_REGION"),
			Bucket:          os.Getenv("TOS_BUCKET"),
			AccessKeyID:     os.Getenv("TOS_ACCESS_KEY"),
			SecretAccessKey: os.Getenv("TOS_SECRET_KEY"),
		})
	default:
		return nil, fmt.Errorf("blob/factory: unknown driver %q", driver)
	}
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
