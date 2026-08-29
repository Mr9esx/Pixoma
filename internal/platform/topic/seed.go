package topic

import (
	"context"
	"fmt"
	"time"
)

// DefaultName is the display name for the reserved default topic.
const DefaultName = "默认"

// EnsureDefaultTopic idempotently creates the reserved default topic.
func EnsureDefaultTopic(ctx context.Context, repo Repository) error {
	if repo == nil {
		return fmt.Errorf("topic: nil repository")
	}
	got, err := repo.Get(ctx, DefaultKey)
	if err == nil {
		if got.Name != DefaultName {
			got.Name = DefaultName
			return repo.Update(ctx, *got)
		}
		return nil
	}
	if err != ErrTopicNotFound {
		return err
	}
	now := time.Now().UTC()
	return repo.Create(ctx, Topic{
		Key:       DefaultKey,
		Name:      DefaultName,
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	})
}
