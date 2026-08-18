package domain

import (
	"context"
	"errors"
	"time"
)

var ErrNotFound = errors.New("user not found")

// ListQuery filters admin user listing (parameterized; no client keys in SQL).
type ListQuery struct {
	Q              string
	ChannelID      *string
	ExternalUserID *string
	CreatedFrom    *time.Time
	CreatedTo      *time.Time
	Limit          int
	Offset         int
}

// Repository persists users and channel-scoped external identities.
type Repository interface {
	UpsertByChannelExternal(ctx context.Context, in UpsertFrom) (*User, error)
	GetByID(ctx context.Context, id string) (*User, error)
	List(ctx context.Context, q ListQuery) ([]*User, error)
}
