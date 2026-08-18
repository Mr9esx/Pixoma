package domain

import "context"

// Repository is the persistence port for a channel-scoped menu tree.
type Repository interface {
	GetTree(ctx context.Context, channelID string) (MenuTree, error)
	ReplaceTree(ctx context.Context, tree MenuTree) error
	ListPlacementsByCase(ctx context.Context, caseID string) ([]MenuPlacement, error)
	ListExtras(ctx context.Context, channelID string) (map[string][]Extra, error)
	SaveExtras(ctx context.Context, channelID string, extras map[string][]Extra) error
}
