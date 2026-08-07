package comfyui

import "context"

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
}

// Mock is an in-memory ComfyUI client for tests.
type Mock struct {
	SubmitFn func(ctx context.Context, graph Graph) (string, error)
	WaitFn   func(ctx context.Context, promptID string) (*Result, error)
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
	return &Result{
		PromptID: promptID,
		Outputs:  []OutputFile{{Filename: "out.png", Mime: "image/png", Data: []byte("png")}},
	}, nil
}
