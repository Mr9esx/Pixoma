package edge

import (
	"time"

	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

// Record is the persisted ComfyUI instance metadata.
type Record struct {
	ID                       sharedkernel.EdgeID
	Name                     string
	Description              string
	Enabled                  bool
	Capabilities             []string
	AgentTokenEnc            string
	Hardware                 Hardware
	HardwareRefreshRequested bool
	StartedAt                *time.Time
	ComfyVersion             string
	CreatedAt                time.Time
	UpdatedAt                time.Time
}
