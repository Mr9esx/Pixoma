package comfyui

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
)

type Graph map[string]any

type Result struct {
	PromptID string
	Outputs  []OutputFile
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
	Mock      bool           `json:"mock,omitempty"`
	Reachable bool           `json:"reachable"`
	Error     string         `json:"error,omitempty"`
	Raw       map[string]any `json:"raw,omitempty"`
}

// QueueView is a read-only view of Comfy GET /queue.
type QueueView struct {
	Mock      bool   `json:"mock,omitempty"`
	Reachable bool   `json:"reachable"`
	Error     string `json:"error,omitempty"`
	Running   []any  `json:"running"`
	Pending   []any  `json:"pending"`
}

// Mock is an in-memory ComfyUI client that returns a real PNG.
type Mock struct {
	SubmitFn      func(ctx context.Context, graph Graph) (string, error)
	WaitFn        func(ctx context.Context, promptID string) (*Result, error)
	UploadImageFn func(ctx context.Context, filename, mime string, data []byte) (string, error)
	Color         color.RGBA // optional tint
}

func (m *Mock) Submit(ctx context.Context, graph Graph) (string, error) {
	if m.SubmitFn != nil {
		return m.SubmitFn(ctx, graph)
	}
	return "prompt-mock", nil
}

func (m *Mock) Wait(ctx context.Context, promptID string) (*Result, error) {
	if m.WaitFn != nil {
		return m.WaitFn(ctx, promptID)
	}
	c := m.Color
	if c.A == 0 {
		c = color.RGBA{R: 70, G: 130, B: 180, A: 255}
	}
	return &Result{
		PromptID: promptID,
		Outputs: []OutputFile{{
			Filename: "out.png",
			Mime:     "image/png",
			Data:     GenerateMockPNG(320, 240, c),
		}},
	}, nil
}

func (m *Mock) UploadImage(ctx context.Context, filename, mime string, data []byte) (string, error) {
	if m.UploadImageFn != nil {
		return m.UploadImageFn(ctx, filename, mime, data)
	}
	return "mock-upload.png", nil
}

func (m *Mock) SystemStats(_ context.Context) (*SystemStats, error) {
	return &SystemStats{
		Mock:      true,
		Reachable: true,
		Raw: map[string]any{
			"system": map[string]any{"comfyui_version": "mock"},
		},
	}, nil
}

func (m *Mock) Queue(_ context.Context) (*QueueView, error) {
	return &QueueView{
		Mock:      true,
		Reachable: true,
		Running:   []any{},
		Pending:   []any{},
	}, nil
}

// GenerateMockPNG creates a simple gradient PNG for Telegram delivery tests.
func GenerateMockPNG(w, h int, base color.RGBA) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8(int(base.R) * x / max(w, 1)),
				G: uint8(int(base.G) * y / max(h, 1)),
				B: base.B,
				A: 255,
			})
		}
	}
	// center block
	for y := h/3; y < 2*h/3; y++ {
		for x := w/3; x < 2*w/3; x++ {
			img.Set(x, y, color.RGBA{R: 255, G: 255, B: 255, A: 255})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
