package edge

import (
	"context"

	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

type Instance struct {
	ID            sharedkernel.EdgeID
	DispatchTopic string
	Capabilities  []string
}

type CapabilityFilter struct {
	AnyOf []string // empty = no filter
}

type Registry interface {
	ListHealthy(ctx context.Context, filter CapabilityFilter) ([]Instance, error)
	// ListEnabled returns enabled instances matching the filter, ignoring cloud health probes.
	// Used in split mode where Edge heartbeat (Online) is the presence signal.
	ListEnabled(ctx context.Context, filter CapabilityFilter) ([]Instance, error)
	Get(ctx context.Context, id sharedkernel.EdgeID) (*Instance, error)
}
