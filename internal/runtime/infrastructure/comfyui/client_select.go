package comfyui

import "fmt"

// Options selects Mock vs real HTTP ComfyUI client.
type Options struct {
	Mock    bool
	BaseURL string
}

// NewClient returns Mock when Mock is true; otherwise a real HTTP client.
// Mock defaults to the development path and MUST stay capable of end-to-end success
// whenever product features change (see project rule comfy-mock-parity).
func NewClient(opts Options) (Client, error) {
	if opts.Mock {
		return &Mock{}, nil
	}
	if opts.BaseURL == "" {
		return nil, fmt.Errorf("comfyui: base URL required when mock is disabled")
	}
	return NewHTTP(opts.BaseURL), nil
}
