package blob

import (
	"context"
	"io"

	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

type PutOptions struct {
	MIME string
}

type Store interface {
	Put(ctx context.Context, key string, r io.Reader, opts PutOptions) (sharedkernel.BlobRef, error)
	Get(ctx context.Context, ref sharedkernel.BlobRef) (io.ReadCloser, error)
}
