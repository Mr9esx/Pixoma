package adminconfig_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/adminconfig"
)

func TestDefault_HTTPAddr(t *testing.T) {
	cfg := adminconfig.Default()
	if cfg.HTTPAddr != "127.0.0.1:8081" {
		t.Fatalf("HTTPAddr=%q, want 127.0.0.1:8081", cfg.HTTPAddr)
	}
}

func TestLoad_YAMLAndEnvOverride(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "admin-api.yaml")
	content := "http_addr: \":9000\"\ncomfy_mock: true\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HTTP_ADDR", ":18081")
	t.Setenv("COMFY_MOCK", "0")

	cfg, err := adminconfig.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTPAddr != ":18081" {
		t.Fatalf("HTTPAddr=%q, want :18081 from env", cfg.HTTPAddr)
	}
	if cfg.ComfyMock {
		t.Fatal("COMFY_MOCK=0 should disable mock")
	}
}

func TestLoad_DefaultCORSOrigins(t *testing.T) {
	t.Setenv("ADMIN_CONFIG", "")
	t.Chdir(t.TempDir())

	cfg, err := adminconfig.Load("")
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.CORSOrigins) == 0 {
		t.Fatal("default CORSOrigins should allow local frontend sources")
	}
	found := false
	for _, o := range cfg.CORSOrigins {
		if o == "http://localhost:5173" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("CORSOrigins=%v, want http://localhost:5173", cfg.CORSOrigins)
	}
}
