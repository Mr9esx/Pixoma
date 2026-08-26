package edge

import (
	"time"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/topic"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

// Record is the persisted ComfyUI instance metadata.
type Record struct {
	ID                       sharedkernel.EdgeID
	Name                     string
	Description              string
	Enabled                  bool
	Capabilities             []string
	SubscribeTopics          []string
	AgentTokenEnc            string
	Hardware                 Hardware
	HardwareRefreshRequested bool
	StartedAt                *time.Time
	ComfyVersion             string
	CreatedAt                time.Time
	UpdatedAt                time.Time
}

// EffectiveTopics returns the normalized subscription set; an empty result
// means the edge has no topic binding and consumes no tasks.
func (r *Record) EffectiveTopics() []string {
	return topic.NormalizeTopics(r.SubscribeTopics)
}
