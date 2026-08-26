package edge

import (
	"context"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/topic"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

type Instance struct {
	ID              sharedkernel.EdgeID
	DispatchTopic   string
	Capabilities    []string
	SubscribeTopics []string
}

// EffectiveTopics returns the normalized subscription set; an empty result
// means the instance has no topic binding and consumes no tasks.
func (i *Instance) EffectiveTopics() []string {
	return topic.NormalizeTopics(i.SubscribeTopics)
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
