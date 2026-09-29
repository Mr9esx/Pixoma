package application

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/Mr9esx/Pixoma/internal/platform/blob"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

func putOwnedBlob(ctx context.Context, repo domain.Repository, store blob.Store, accountID, key, mimeType string, content io.Reader, now time.Time) (sharedkernel.BlobRef, error) {
	intent := &domain.BlobWriteIntent{BlobKey: key, AccountID: accountID, CreatedAt: now.UTC(), ExpiresAt: now.UTC().Add(15 * time.Minute)}
	if err := repo.RegisterBlobWriteIntent(ctx, intent); err != nil {
		return sharedkernel.BlobRef{}, err
	}
	ref, err := store.Put(ctx, key, content, blob.PutOptions{MIME: mimeType})
	if err != nil {
		cleanupErr := cleanupOwnedBlob(ctx, repo, store, accountID, sharedkernel.BlobRef{Key: key})
		return sharedkernel.BlobRef{}, errors.Join(err, cleanupErr)
	}
	return ref, nil
}

func cleanupOwnedBlob(ctx context.Context, repo domain.Repository, store blob.Store, accountID string, ref sharedkernel.BlobRef) error {
	if err := store.Delete(context.WithoutCancel(ctx), ref); err != nil {
		return err
	}
	return repo.DeleteBlobWriteIntent(context.WithoutCancel(ctx), accountID, ref.Key)
}
