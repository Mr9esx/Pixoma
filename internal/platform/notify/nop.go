package notify

import (
	"context"

	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

// Nop is a no-op Publisher for callers that must not deliver user notifications
// (e.g. admin cancel wiring).
type Nop struct{}

func (Nop) Publish(context.Context, sharedkernel.UserNotify) error { return nil }
