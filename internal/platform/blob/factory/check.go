package factory

import (
	"context"
	"fmt"
	"strings"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/blob"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/blob/localfs"
	blobs3 "github.com/mr9esx/comfyui_tgbot/internal/platform/blob/s3"
	blobtos "github.com/mr9esx/comfyui_tgbot/internal/platform/blob/tos"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/botconfig"
)

// CheckOptions carries explicit blob configuration for connectivity checks.
// Credentials come from the wizard form, not environment variables.
type CheckOptions struct {
	Driver    string
	LocalRoot string
	Endpoint  string
	Region    string
	Bucket    string
	AccessKey string
	SecretKey string
}

// Check verifies blob connectivity: localfs directory writability, S3/TOS
// bucket reachability (HeadBucket).
func Check(ctx context.Context, opts CheckOptions) error {
	store, err := storeForCheck(opts)
	if err != nil {
		return err
	}
	return store.Check(ctx)
}

// EnsureBucket creates the target bucket when missing (S3/TOS), then re-checks.
func EnsureBucket(ctx context.Context, opts CheckOptions) error {
	switch strings.TrimSpace(opts.Driver) {
	case botconfig.BlobDriverLocalFS:
		store, err := localfs.New(opts.LocalRoot)
		if err != nil {
			return err
		}
		return store.Check(ctx)
	case botconfig.BlobDriverS3:
		store, err := blobs3.New(blobs3.Options{
			Endpoint:        opts.Endpoint,
			Region:          opts.Region,
			Bucket:          opts.Bucket,
			AccessKeyID:     opts.AccessKey,
			SecretAccessKey: opts.SecretKey,
			UsePathStyle:    true,
		})
		if err != nil {
			return err
		}
		return store.EnsureBucket(ctx)
	case botconfig.BlobDriverTOS:
		store, err := blobtos.New(blobtos.Options{
			Endpoint:        opts.Endpoint,
			Region:          opts.Region,
			Bucket:          opts.Bucket,
			AccessKeyID:     opts.AccessKey,
			SecretAccessKey: opts.SecretKey,
		})
		if err != nil {
			return err
		}
		return store.EnsureBucket(ctx)
	default:
		return fmt.Errorf("blob/factory: unknown driver %q", opts.Driver)
	}
}

func storeForCheck(opts CheckOptions) (blob.Store, error) {
	switch strings.TrimSpace(opts.Driver) {
	case botconfig.BlobDriverLocalFS:
		return localfs.New(opts.LocalRoot)
	case botconfig.BlobDriverS3:
		return blobs3.New(blobs3.Options{
			Endpoint:        opts.Endpoint,
			Region:          opts.Region,
			Bucket:          opts.Bucket,
			AccessKeyID:     opts.AccessKey,
			SecretAccessKey: opts.SecretKey,
			UsePathStyle:    true,
		})
	case botconfig.BlobDriverTOS:
		return blobtos.New(blobtos.Options{
			Endpoint:        opts.Endpoint,
			Region:          opts.Region,
			Bucket:          opts.Bucket,
			AccessKeyID:     opts.AccessKey,
			SecretAccessKey: opts.SecretKey,
		})
	default:
		return nil, fmt.Errorf("blob/factory: unknown driver %q", opts.Driver)
	}
}
