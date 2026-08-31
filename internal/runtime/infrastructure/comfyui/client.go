package comfyui

import "context"

type Graph map[string]any

type Result struct {
	PromptID string
	Outputs  HistoryResult
}

type OutputFile struct {
	Filename string
	Mime     string
	Data     []byte
}

type Client interface {
	Submit(ctx context.Context, graph Graph) (promptID string, err error)
	Wait(ctx context.Context, promptID string) (*Result, error)
	UploadImage(ctx context.Context, filename, mime string, data []byte) (remoteFilename string, err error)
	SystemStats(ctx context.Context) (*SystemStats, error)
	Queue(ctx context.Context) (*QueueView, error)
}

// SystemStats is a read-only view of Comfy GET /system_stats.
type SystemStats struct {
	Reachable      bool           `json:"reachable"`
	Error          string         `json:"error,omitempty"`
	ComfyUIVersion string         `json:"comfyui_version,omitempty"`
	Raw            map[string]any `json:"raw,omitempty"`
}

// QueueView is a read-only view of Comfy GET /queue.
type QueueView struct {
	Reachable bool   `json:"reachable"`
	Error     string `json:"error,omitempty"`
	Running   []any  `json:"running"`
	Pending   []any  `json:"pending"`
}
