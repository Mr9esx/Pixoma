package botconfig_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/botconfig"
)

func TestLoad_DefaultMockOn(t *testing.T) {
	t.Setenv("BOT_CONFIG", "")
	t.Chdir(t.TempDir())
	cfg, err := botconfig.Load("")
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.ComfyMock {
		t.Fatal("default comfy_mock should be true")
	}
}

func TestLoad_YAMLAndEnvOverride(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bot.yaml")
	if err := os.WriteFile(path, []byte("comfy_mock: true\ncomfyui_base_url: http://x:9\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("COMFY_MOCK", "0")
	cfg, err := botconfig.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ComfyMock {
		t.Fatal("COMFY_MOCK=0 should disable mock")
	}
	if cfg.ComfyUIBaseURL != "http://x:9" {
		t.Fatalf("base url=%q", cfg.ComfyUIBaseURL)
	}
}
