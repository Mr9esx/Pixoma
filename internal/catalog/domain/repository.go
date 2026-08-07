package domain

import (
	"context"
	"errors"

	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

var (
	ErrNotFound      = errors.New("case not found")
	ErrAlreadyExists = errors.New("case already exists")
	ErrDisabled      = errors.New("case disabled")
)

// Case is the catalog aggregate: protocol document + availability.
type Case struct {
	Document CaseDocument
	Enabled  bool
}

type ListQuery struct {
	Tag      string
	MenuKey  string
	Category string
	Enabled  *bool // nil = all
	Limit    int
	Offset   int
}

type Repository interface {
	Save(ctx context.Context, c *Case) error
	// Create fails with ErrAlreadyExists if id is taken.
	Create(ctx context.Context, c *Case) error
	Get(ctx context.Context, id sharedkernel.CaseID) (*Case, error)
	List(ctx context.Context, q ListQuery) ([]*Case, error)
	Disable(ctx context.Context, id sharedkernel.CaseID) error
}
