package topic

import (
	"context"
	"fmt"
	"time"
)

// EnsureDefaultTopic idempotently creates the reserved default topic.
func EnsureDefaultTopic(ctx context.Context, repo Repository) error {
	if repo == nil {
		return fmt.Errorf("topic: nil repository")
	}
	_, err := repo.Get(ctx, DefaultKey)
	if err == nil {
		return nil
	}
	if err != ErrTopicNotFound {
		return err
	}
	now := time.Now().UTC()
	return repo.Create(ctx, Topic{
		Key:       DefaultKey,
		Name:      "Default",
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	})
}
