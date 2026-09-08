package domain

import (
	"time"

	topicdomain "github.com/Mr9esx/Pixoma/internal/topics/domain"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
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
	return topicdomain.NormalizeTopics(r.SubscribeTopics)
}
