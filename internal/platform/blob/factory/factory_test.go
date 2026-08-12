package factory_test

import (
	"strings"
	"testing"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/blob/factory"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/botconfig"
)

func TestNew_UnknownDriver(t *testing.T) {
	_, err := factory.New("nope", t.TempDir())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "driver") {
		t.Fatalf("got %v", err)
	}
}

func TestNew_LocalFS(t *testing.T) {
	store, err := factory.New(botconfig.BlobDriverLocalFS, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if store == nil {
		t.Fatal("nil store")
	}
}
