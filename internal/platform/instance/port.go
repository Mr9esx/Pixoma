package instance

import (
	"context"

	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

type Instance struct {
	ID            sharedkernel.InstanceID
	DispatchTopic string
	Capabilities  []string
}

type CapabilityFilter struct {
	AnyOf []string // empty = no filter
}

type Registry interface {
	ListHealthy(ctx context.Context, filter CapabilityFilter) ([]Instance, error)
	Get(ctx context.Context, id sharedkernel.InstanceID) (*Instance, error)
}
