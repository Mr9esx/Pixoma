package botconfig_test

import (
	"os"
	"path/filepath"
	"strings"
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

func TestDefault_LocalLocalfs(t *testing.T) {
	cfg := botconfig.Default()
	if cfg.Placement != botconfig.PlacementLocal {
		t.Fatalf("placement=%q", cfg.Placement)
	}
	if cfg.Blob.Driver != botconfig.BlobDriverLocalFS {
		t.Fatalf("blob=%q", cfg.Blob.Driver)
	}
	if err := cfg.ValidateRuntimeDrivers(); err != nil {
		t.Fatal(err)
	}
}

func TestValidateRuntimeDrivers_RemoteIgnoresQueue(t *testing.T) {
	cfg := botconfig.Default()
	cfg.Placement = botconfig.PlacementRemote
	cfg.RuntimeMode = botconfig.RuntimeModeSplit
	cfg.Queue.Driver = botconfig.QueueDriverMemory
	cfg.Blob.Driver = botconfig.BlobDriverS3
	if err := cfg.ValidateRuntimeDrivers(); err != nil {
		t.Fatal(err)
	}
}

func TestValidateRuntimeDrivers_RemoteOK(t *testing.T) {
	cfg := botconfig.Default()
	cfg.Placement = botconfig.PlacementRemote
	cfg.Blob.Driver = botconfig.BlobDriverS3
	if err := cfg.ValidateRuntimeDrivers(); err != nil {
		t.Fatal(err)
	}
}

func TestValidateRuntimeDrivers_SplitTOSOK(t *testing.T) {
	cfg := botconfig.Default()
	cfg.RuntimeMode = botconfig.RuntimeModeSplit
	cfg.Blob.Driver = botconfig.BlobDriverTOS
	if err := cfg.ValidateRuntimeDrivers(); err != nil {
		t.Fatal(err)
	}
}

func TestValidateRuntimeDrivers_SplitRejectsLocalFS(t *testing.T) {
	cfg := botconfig.Default()
	cfg.RuntimeMode = botconfig.RuntimeModeSplit
	cfg.Blob.Driver = botconfig.BlobDriverLocalFS
	err := cfg.ValidateRuntimeDrivers()
	if err == nil {
		t.Fatal("expected error for remote+localfs")
	}
	if !strings.Contains(err.Error(), "blob.driver") {
		t.Fatalf("error should mention blob.driver, got %v", err)
	}
}

func TestValidateRuntimeDrivers_SplitS3StillOK(t *testing.T) {
	cfg := botconfig.Default()
	cfg.RuntimeMode = botconfig.RuntimeModeSplit
	cfg.Blob.Driver = botconfig.BlobDriverS3
	if err := cfg.ValidateRuntimeDrivers(); err != nil {
		t.Fatal(err)
	}
}

func TestLoad_RuntimeModeFromYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bot.yaml")
	content := "runtime_mode: split\nqueue:\n  driver: redis\nblob:\n  driver: s3\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := botconfig.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.RuntimeMode != botconfig.RuntimeModeSplit {
		t.Fatalf("mode=%q", cfg.RuntimeMode)
	}
	if cfg.Queue.Driver != botconfig.QueueDriverRedis {
		t.Fatalf("queue=%q", cfg.Queue.Driver)
	}
	if cfg.Blob.Driver != botconfig.BlobDriverS3 {
		t.Fatalf("blob=%q", cfg.Blob.Driver)
	}
	if err := cfg.ValidateRuntimeDrivers(); err != nil {
		t.Fatal(err)
	}
}
