package domain

import "context"

// Repository is the persistence port for the single TG menu tree.
type Repository interface {
	GetTree(ctx context.Context, id string) (MenuTree, error)
	ReplaceTree(ctx context.Context, tree MenuTree) error
	ListPlacementsByCase(ctx context.Context, caseID string) ([]MenuPlacement, error)
}
