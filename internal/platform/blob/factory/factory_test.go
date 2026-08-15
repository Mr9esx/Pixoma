package factory_test

import (
	"os"
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

func TestNewFromConfig_TOSUsesYAMLWhenEnvEmpty(t *testing.T) {
	for _, k := range []string{"TOS_ENDPOINT", "TOS_REGION", "TOS_BUCKET", "TOS_ACCESS_KEY", "TOS_SECRET_KEY"} {
		t.Setenv(k, "")
		_ = os.Unsetenv(k)
	}
	cfg := botconfig.Config{
		Blob: botconfig.BlobConfig{
			Driver: botconfig.BlobDriverTOS,
			TOS: botconfig.BlobTOSConfig{
				Endpoint: "https://tos.example.local",
				Region:   "cn-beijing",
				Bucket:   "from-yaml",
			},
		},
	}
	// Credentials still required from env for a live client; empty keys should still construct
	// after we pass YAML endpoint/region/bucket (SDK allows empty AK for New; Put would fail).
	t.Setenv("TOS_ACCESS_KEY", "ak")
	t.Setenv("TOS_SECRET_KEY", "sk")

	store, err := factory.NewFromConfig(cfg, t.TempDir())
	if err != nil {
		t.Fatalf("expected TOS store from YAML config, got %v", err)
	}
	if store == nil {
		t.Fatal("nil store")
	}
}

func TestNewFromConfig_S3UsesConfigWhenEnvEmpty(t *testing.T) {
	for _, k := range []string{"S3_ENDPOINT", "S3_REGION", "S3_BUCKET", "S3_ACCESS_KEY", "S3_SECRET_KEY"} {
		_ = os.Unsetenv(k)
	}
	cfg := botconfig.Config{
		Blob: botconfig.BlobConfig{
			Driver: botconfig.BlobDriverS3,
			S3: botconfig.BlobTOSConfig{
				Endpoint: "http://minio.example.local:9000",
				Region:   "us-east-1",
				Bucket:   "from-wizard",
			},
		},
	}
	t.Setenv("S3_ACCESS_KEY", "ak")
	t.Setenv("S3_SECRET_KEY", "sk")

	opts := factory.S3Options(cfg)
	if opts.Bucket != "from-wizard" {
		t.Fatalf("bucket=%q, want from-wizard (wizard settings, not default pixoma)", opts.Bucket)
	}
	if opts.Endpoint != "http://minio.example.local:9000" {
		t.Fatalf("endpoint=%q", opts.Endpoint)
	}
}

func TestNew_TOSDoesNotFallBackToLocalFS(t *testing.T) {
	for _, k := range []string{"TOS_ENDPOINT", "TOS_REGION", "TOS_BUCKET", "TOS_ACCESS_KEY", "TOS_SECRET_KEY"} {
		_ = os.Unsetenv(k)
	}
	_, err := factory.New(botconfig.BlobDriverTOS, t.TempDir())
	if err == nil {
		t.Fatal("expected error when TOS env missing, not silent localfs")
	}
}
