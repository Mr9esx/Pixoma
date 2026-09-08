package application

import (
	"context"
	"fmt"
	"time"

	"github.com/Mr9esx/Pixoma/internal/topics/domain"
)

// DefaultName is the display name for the reserved default topic.
const DefaultName = "默认"

// EnsureDefaultTopic idempotently creates the reserved default topic.
func EnsureDefaultTopic(ctx context.Context, repo domain.Repository) error {
	if repo == nil {
		return fmt.Errorf("topic: nil repository")
	}
	got, err := repo.Get(ctx, domain.DefaultKey)
	if err == nil {
		if got.Name != DefaultName {
			got.Name = DefaultName
			return repo.Update(ctx, *got)
		}
		return nil
	}
	if err != domain.ErrTopicNotFound {
		return err
	}
	now := time.Now().UTC()
	return repo.Create(ctx, domain.Topic{
		Key:       domain.DefaultKey,
		Name:      DefaultName,
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	})
}
