package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Mr9esx/Pixoma/internal/platform/blob"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

type BlobIntentWorker struct {
	Repo domain.Repository
	Blob blob.Store
	Now  func() time.Time
}

func (w *BlobIntentWorker) ProcessOnce(ctx context.Context, limit int) error {
	if w == nil || w.Repo == nil || w.Blob == nil {
		return fmt.Errorf("studio: blob intent worker is not configured")
	}
	now := time.Now().UTC()
	if w.Now != nil {
		now = w.Now().UTC()
	}
	intents, err := w.Repo.ClaimExpiredBlobWriteIntents(ctx, limit, now)
	if err != nil {
		return err
	}
	var failures []error
	for _, intent := range intents {
		if err := ctx.Err(); err != nil {
			return err
		}
		referenced, err := w.Repo.IsBlobReferenced(ctx, intent.AccountID, intent.BlobKey)
		if err != nil {
			failures = append(failures, err)
			continue
		}
		if !referenced {
			if err := w.Blob.Delete(ctx, sharedkernel.BlobRef{Key: intent.BlobKey}); err != nil {
				failures = append(failures, err)
				continue
			}
		}
		if err := w.Repo.DeleteBlobWriteIntent(ctx, intent.AccountID, intent.BlobKey); err != nil {
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}
