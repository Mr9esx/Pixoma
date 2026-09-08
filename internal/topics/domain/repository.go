package domain

import "context"

// Repository persists Topic records.
type Repository interface {
	List(ctx context.Context, enabled *bool) ([]Topic, error)
	Get(ctx context.Context, key string) (*Topic, error)
	Create(ctx context.Context, t Topic) error
	Update(ctx context.Context, t Topic) error
	Delete(ctx context.Context, key string) error
}
