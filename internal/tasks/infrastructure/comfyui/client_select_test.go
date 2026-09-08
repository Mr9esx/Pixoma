package comfyui_test

import (
	"testing"

	"github.com/Mr9esx/Pixoma/internal/tasks/infrastructure/comfyui"
)

func TestNewClient_HTTPRequiresBaseURL(t *testing.T) {
	_, err := comfyui.NewClient(comfyui.Options{})
	if err == nil {
		t.Fatal("expected base URL error")
	}
}

func TestNewClient_HTTP(t *testing.T) {
	c, err := comfyui.NewClient(comfyui.Options{BaseURL: "http://127.0.0.1:8188"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := c.(*comfyui.HTTP); !ok {
		t.Fatalf("got %T, want *comfyui.HTTP", c)
	}
}
