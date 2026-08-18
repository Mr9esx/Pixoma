package instance

import (
	"time"

	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

// Record is the persisted ComfyUI instance metadata.
type Record struct {
	ID           sharedkernel.InstanceID
	BaseURL      string
	Enabled      bool
	Capabilities []string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
