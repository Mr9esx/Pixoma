package actuator

import (
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/comfyui"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

// JobPackage is the scheme-A payload stored at jobs/<task_id>/job.json.
type JobPackage struct {
	TaskID       sharedkernel.TaskID     `json:"task_id"`
	InstanceID   sharedkernel.InstanceID `json:"instance_id"`
	Workflow     comfyui.Graph           `json:"workflow"`
	Images       []JobImage              `json:"images,omitempty"`
	OutputPrefix string                  `json:"output_prefix"`
}

// JobImage describes an image that the executor must upload locally before Submit.
type JobImage struct {
	NodeID    string               `json:"node_id"`
	FieldPath string               `json:"field_path"`
	Blob      sharedkernel.BlobRef `json:"blob"`
}
