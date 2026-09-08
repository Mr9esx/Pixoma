package blob

import (
	"context"
	"errors"
	"io"

	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
)

type PutOptions struct {
	MIME string
}

type Store interface {
	Put(ctx context.Context, key string, r io.Reader, opts PutOptions) (sharedkernel.BlobRef, error)
	Get(ctx context.Context, ref sharedkernel.BlobRef) (io.ReadCloser, error)
	Check(ctx context.Context) error
}

// ErrBucketNotFound is returned by connectivity checks when the target bucket
// does not exist (S3/TOS). Callers may offer to create it via EnsureBucket.
var ErrBucketNotFound = errors.New("blob: bucket not found")
