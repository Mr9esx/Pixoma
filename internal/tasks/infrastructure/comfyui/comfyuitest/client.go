// Package comfyuitest provides fake clients for tests outside the comfyui package.
package comfyuitest

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"

	"github.com/Mr9esx/Pixoma/internal/tasks/infrastructure/comfyui"
)

type Fake struct {
	SubmitFn      func(ctx context.Context, graph comfyui.Graph) (string, error)
	WaitFn        func(ctx context.Context, promptID string) (*comfyui.Result, error)
	UploadImageFn func(ctx context.Context, filename, mime string, data []byte) (string, error)
	Color         color.RGBA
}

func (f *Fake) Submit(ctx context.Context, graph comfyui.Graph) (string, error) {
	if f.SubmitFn != nil {
		return f.SubmitFn(ctx, graph)
	}
	return "prompt-fake", nil
}

func (f *Fake) Wait(ctx context.Context, promptID string) (*comfyui.Result, error) {
	if f.WaitFn != nil {
		return f.WaitFn(ctx, promptID)
	}
	c := f.Color
	if c.A == 0 {
		c = color.RGBA{R: 70, G: 130, B: 180, A: 255}
	}
	return &comfyui.Result{
		PromptID: promptID,
		Outputs: comfyui.HistoryResult{
			"1": {
				Images: []comfyui.NodeImage{{
					OutputFile: comfyui.OutputFile{
						Filename: "out.png",
						Mime:     "image/png",
						Data:     GeneratePNG(320, 240, c),
					},
				}},
			},
		},
	}, nil
}

func (f *Fake) UploadImage(ctx context.Context, filename, mime string, data []byte) (string, error) {
	if f.UploadImageFn != nil {
		return f.UploadImageFn(ctx, filename, mime, data)
	}
	return "fake-upload.png", nil
}

func (f *Fake) SystemStats(_ context.Context) (*comfyui.SystemStats, error) {
	return &comfyui.SystemStats{
		Reachable:      true,
		ComfyUIVersion: "fake",
		Raw: map[string]any{
			"system": map[string]any{"comfyui_version": "fake"},
			"devices": []any{
				map[string]any{
					"name":       "Fake GPU",
					"vram_total": float64(8 << 30),
					"vram_free":  float64(8 << 30),
				},
			},
		},
	}, nil
}

func (f *Fake) Queue(_ context.Context) (*comfyui.QueueView, error) {
	return &comfyui.QueueView{
		Reachable: true,
		Running:   []any{},
		Pending:   []any{},
	}, nil
}

func GeneratePNG(w, h int, base color.RGBA) []byte {
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
	for y := h / 3; y < 2*h/3; y++ {
		for x := w / 3; x < 2*w/3; x++ {
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
