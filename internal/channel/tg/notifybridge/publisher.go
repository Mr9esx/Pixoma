package notifybridge

import (
	"context"

	"github.com/mr9esx/comfyui_tgbot/internal/channel/tg"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

// Publisher adapts orchestrator notify.Publisher to the TG adapter.
type Publisher struct {
	Adapter *tg.Adapter
}

func (p *Publisher) Publish(ctx context.Context, n sharedkernel.UserNotify) error {
	return p.Adapter.HandleUserNotify(ctx, n)
}
