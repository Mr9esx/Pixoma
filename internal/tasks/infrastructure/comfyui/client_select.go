package comfyui

import "fmt"

// Options configures the real HTTP ComfyUI client.
type Options struct {
	BaseURL string
}

// NewClient returns the real HTTP client; a reachable ComfyUI root is required.
func NewClient(opts Options) (Client, error) {
	if opts.BaseURL == "" {
		return nil, fmt.Errorf("comfyui: base URL required")
	}
	return NewHTTP(opts.BaseURL), nil
}
