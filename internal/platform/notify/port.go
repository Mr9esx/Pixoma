package notify

import (
	"context"

	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

type Publisher interface {
	Publish(ctx context.Context, n sharedkernel.UserNotify) error
}
