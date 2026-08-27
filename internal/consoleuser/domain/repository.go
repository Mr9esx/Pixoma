package domain

import "context"

// ListQuery filters console user listing.
type ListQuery struct {
	Q      string // search username/email/nickname
	Limit  int
	Offset int
}

// Repository persists console (login) accounts.
type Repository interface {
	Create(ctx context.Context, u *ConsoleUser) error
	GetByUsername(ctx context.Context, username string) (*ConsoleUser, error)
	GetByID(ctx context.Context, id string) (*ConsoleUser, error)
	List(ctx context.Context, q ListQuery) ([]*ConsoleUser, error)
	CountAdmins(ctx context.Context) (int64, error)
	Update(ctx context.Context, u *ConsoleUser) error
	Delete(ctx context.Context, id string) error
}
