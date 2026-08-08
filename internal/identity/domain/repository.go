package domain

import (
	"context"
	"errors"
)

var ErrNotFound = errors.New("user not found")

// Repository persists users and supports Telegram upsert.
type Repository interface {
	UpsertByTgUserID(ctx context.Context, in UpsertFrom) (*User, error)
	GetByID(ctx context.Context, id string) (*User, error)
}
