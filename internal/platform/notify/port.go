package notify

import (
	"context"

	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
)

type Publisher interface {
	Publish(ctx context.Context, n sharedkernel.UserNotify) error
}
