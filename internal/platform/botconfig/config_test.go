package botconfig_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Mr9esx/Pixoma/internal/platform/botconfig"
)

func TestLoad_YAMLAndEnvOverride(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bot.yaml")
	if err := os.WriteFile(path, []byte("comfyui_base_url: http://x:9\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := botconfig.Load(path)
	if err != nil {
		t.Fatal(err)
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

func TestValidateRuntimeDrivers_RemoteTOSOK(t *testing.T) {
	cfg := botconfig.Default()
	cfg.Placement = botconfig.PlacementRemote
	cfg.Blob.Driver = botconfig.BlobDriverTOS
	if err := cfg.ValidateRuntimeDrivers(); err != nil {
		t.Fatal(err)
	}
}

func TestValidateRuntimeDrivers_RemoteRejectsLocalFS(t *testing.T) {
	cfg := botconfig.Default()
	cfg.Placement = botconfig.PlacementRemote
	cfg.Blob.Driver = botconfig.BlobDriverLocalFS
	err := cfg.ValidateRuntimeDrivers()
	if err == nil {
		t.Fatal("expected error for remote+localfs")
	}
	if !strings.Contains(err.Error(), "blob.driver") {
		t.Fatalf("error should mention blob.driver, got %v", err)
	}
}

func TestValidateRuntimeDrivers_RemoteS3StillOK(t *testing.T) {
	cfg := botconfig.Default()
	cfg.Placement = botconfig.PlacementRemote
	cfg.Blob.Driver = botconfig.BlobDriverS3
	if err := cfg.ValidateRuntimeDrivers(); err != nil {
		t.Fatal(err)
	}
}

func TestLoad_PlacementFromYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bot.yaml")
	content := "placement: remote\nqueue:\n  driver: redis\nblob:\n  driver: s3\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := botconfig.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Placement != botconfig.PlacementRemote {
		t.Fatalf("placement=%q", cfg.Placement)
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
