package edge

import (
	"context"
	"time"

	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

// MetricsRepository persists live system metric snapshots per Edge.
type MetricsRepository interface {
	Append(ctx context.Context, edgeID sharedkernel.EdgeID, m Metrics) error
	ListSince(ctx context.Context, edgeID sharedkernel.EdgeID, since time.Time, limit int) ([]Metrics, error)
	LatestAll(ctx context.Context, since time.Time) (map[sharedkernel.EdgeID]Metrics, error)
}
