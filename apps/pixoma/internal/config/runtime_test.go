package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Mr9esx/Pixoma/apps/pixoma/internal/config"
	"github.com/Mr9esx/Pixoma/internal/platform/bootstrap"
	"github.com/Mr9esx/Pixoma/internal/platform/botconfig"
	settingsdomain "github.com/Mr9esx/Pixoma/internal/settings/domain"
)

func TestWriteReadAgentTokenFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent.token")
	if err := config.WriteAgentTokenFile(path, "plain-token"); err != nil {
		t.Fatal(err)
	}
	got, err := config.ReadAgentTokenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != "plain-token" {
		t.Fatalf("got %q", got)
	}
	st, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm()&0o077 != 0 {
		t.Fatalf("token file too open: %v", st.Mode())
	}
}

func TestApplyBlobEnvSetsS3Connection(t *testing.T) {
	for _, k := range []string{"S3_ENDPOINT", "S3_REGION", "S3_BUCKET", "S3_ACCESS_KEY", "S3_SECRET_KEY"} {
		t.Setenv(k, "")
		_ = os.Unsetenv(k)
	}
	config.ApplyBlobEnv(settingsdomain.Settings{
		BlobDriver:    botconfig.BlobDriverS3,
		BlobEndpoint:  "http://minio:9000",
		BlobRegion:    "us-east-1",
		BlobBucket:    "from-wizard",
		BlobAccessKey: "ak",
		BlobSecretKey: "sk",
	})
	if os.Getenv("S3_ENDPOINT") != "http://minio:9000" {
		t.Fatalf("S3_ENDPOINT=%q", os.Getenv("S3_ENDPOINT"))
	}
	if os.Getenv("S3_BUCKET") != "from-wizard" {
		t.Fatalf("S3_BUCKET=%q", os.Getenv("S3_BUCKET"))
	}
	if os.Getenv("S3_ACCESS_KEY") != "ak" || os.Getenv("S3_SECRET_KEY") != "sk" {
		t.Fatal("s3 secrets not applied")
	}
}

func TestApplyHTTPProxyFromSettingsWhenEnvEmpty(t *testing.T) {
	for _, k := range []string{
		"HTTP_PROXY", "HTTPS_PROXY", "http_proxy", "https_proxy", "NO_PROXY", "no_proxy", "ALL_PROXY", "all_proxy",
	} {
		t.Setenv(k, "")
		_ = os.Unsetenv(k)
	}
	config.ApplyHTTPProxy(settingsdomain.Settings{
		ProxyKind: settingsdomain.ProxyHTTP,
		ProxyHost: "127.0.0.1",
		ProxyPort: 7897,
	})
	if os.Getenv("HTTPS_PROXY") != "http://127.0.0.1:7897" {
		t.Fatalf("HTTPS_PROXY=%q", os.Getenv("HTTPS_PROXY"))
	}
	if os.Getenv("HTTP_PROXY") != "http://127.0.0.1:7897" {
		t.Fatalf("HTTP_PROXY=%q", os.Getenv("HTTP_PROXY"))
	}
	if !strings.Contains(os.Getenv("NO_PROXY"), "127.0.0.1") {
		t.Fatalf("NO_PROXY=%q", os.Getenv("NO_PROXY"))
	}
}

func TestApplyHTTPProxySocks5(t *testing.T) {
	for _, k := range []string{
		"HTTP_PROXY", "HTTPS_PROXY", "http_proxy", "https_proxy", "NO_PROXY", "no_proxy", "ALL_PROXY", "all_proxy",
	} {
		t.Setenv(k, "")
		_ = os.Unsetenv(k)
	}
	config.ApplyHTTPProxy(settingsdomain.Settings{
		ProxyKind: settingsdomain.ProxySOCKS,
		ProxyHost: "127.0.0.1",
		ProxyPort: 7897,
	})
	if os.Getenv("ALL_PROXY") != "socks5://127.0.0.1:7897" {
		t.Fatalf("ALL_PROXY=%q", os.Getenv("ALL_PROXY"))
	}
	if os.Getenv("HTTPS_PROXY") != "socks5://127.0.0.1:7897" {
		t.Fatalf("HTTPS_PROXY=%q", os.Getenv("HTTPS_PROXY"))
	}
}

func TestApplyHTTPProxyKeepsExistingEnv(t *testing.T) {
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:8888")
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:8888")
	t.Setenv("NO_PROXY", "")
	_ = os.Unsetenv("NO_PROXY")
	_ = os.Unsetenv("no_proxy")
	config.ApplyHTTPProxy(settingsdomain.Settings{
		ProxyKind: settingsdomain.ProxyHTTP,
		ProxyHost: "127.0.0.1",
		ProxyPort: 7897,
	})
	if os.Getenv("HTTPS_PROXY") != "http://127.0.0.1:8888" {
		t.Fatalf("HTTPS_PROXY=%q", os.Getenv("HTTPS_PROXY"))
	}
	if !strings.Contains(os.Getenv("NO_PROXY"), "192.168.0.0/16") {
		t.Fatalf("NO_PROXY=%q", os.Getenv("NO_PROXY"))
	}
}

func TestLoadVerifiedAgentTokenRejectsMismatch(t *testing.T) {
	dir := t.TempDir()
	st, _, err := bootstrap.Open(filepath.Join(dir, "bootstrap.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	tok, minted, err := st.EnsureAgentToken()
	if err != nil || !minted {
		t.Fatalf("minted=%v err=%v", minted, err)
	}
	path := filepath.Join(dir, "agent.token")
	if err := config.WriteAgentTokenFile(path, "not-the-token"); err != nil {
		t.Fatal(err)
	}
	if _, err := config.LoadVerifiedAgentToken(st, path); err == nil {
		t.Fatal("expected mismatch error")
	}
	if err := config.WriteAgentTokenFile(path, tok); err != nil {
		t.Fatal(err)
	}
	got, err := config.LoadVerifiedAgentToken(st, path)
	if err != nil {
		t.Fatal(err)
	}
	if got != tok {
		t.Fatalf("got %q", got)
	}
}
