package actuator

import (
	catalogdomain "github.com/Mr9esx/Pixoma/internal/cases/domain"
	"github.com/Mr9esx/Pixoma/internal/tasks/infrastructure/comfyui"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
)

// JobPackage is the scheme-A payload stored at jobs/<task_id>/job.json.
type JobPackage struct {
	TaskID       sharedkernel.TaskID           `json:"task_id"`
	EdgeID       sharedkernel.EdgeID           `json:"edge_id"`
	Workflow     comfyui.Graph                 `json:"workflow"`
	Images       []JobImage                    `json:"images,omitempty"`
	OutputPrefix string                        `json:"output_prefix"`
	Outputs      []catalogdomain.OutputBinding `json:"outputs,omitempty"`
}

// JobImage describes an image that the executor must upload locally before Submit.
type JobImage struct {
	NodeID    string               `json:"node_id"`
	FieldPath string               `json:"field_path"`
	Blob      sharedkernel.BlobRef `json:"blob"`
}
