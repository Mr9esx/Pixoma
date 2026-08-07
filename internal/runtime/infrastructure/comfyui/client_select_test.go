package comfyui_test

import (
	"context"
	"testing"

	"github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/comfyui"
)

func TestNewClient_MockDefaultPath(t *testing.T) {
	c, err := comfyui.NewClient(comfyui.Options{Mock: true})
	if err != nil {
		t.Fatal(err)
	}
	id, err := c.Submit(context.Background(), comfyui.Graph{"1": map[string]any{}})
	if err != nil {
		t.Fatal(err)
	}
	res, err := c.Wait(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Outputs) == 0 || len(res.Outputs[0].Data) == 0 {
		t.Fatal("mock must return image bytes")
	}
}

func TestNewClient_HTTPRequiresBaseURL(t *testing.T) {
	_, err := comfyui.NewClient(comfyui.Options{Mock: false})
	if err == nil {
		t.Fatal("expected error when mock off without base URL")
	}
}

func TestNewClient_HTTP(t *testing.T) {
	c, err := comfyui.NewClient(comfyui.Options{Mock: false, BaseURL: "http://127.0.0.1:8188"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := c.(*comfyui.HTTP); !ok {
		t.Fatalf("got %T, want *comfyui.HTTP", c)
	}
}
