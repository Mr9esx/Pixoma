package domain

import "context"

// Repository is the persistence port for the single TG menu document.
type Repository interface {
	Get(ctx context.Context, id string) (MenuDocument, error)
	Replace(ctx context.Context, doc MenuDocument) error
}
