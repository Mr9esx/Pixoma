package app_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mr9esx/comfyui_tgbot/apps/pixoma/internal/app"
	"github.com/mr9esx/comfyui_tgbot/internal/httpapi/adminhost"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/bootstrap"
)

func TestAgentTokenFile_RoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent.token")
	if err := app.WriteAgentTokenFile(path, "plain-token"); err != nil {
		t.Fatal(err)
	}
	got, err := app.ReadAgentTokenFile(path)
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

func TestBootstrapBannerContract_FirstOpen(t *testing.T) {
	dir := t.TempDir()
	st, creds, err := bootstrap.Open(filepath.Join(dir, "bootstrap.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	tok, minted, err := st.EnsureAgentToken()
	if err != nil || !minted || tok == "" {
		t.Fatalf("token minted=%v err=%v", minted, err)
	}
	if err := app.WriteAgentTokenFile(filepath.Join(dir, "agent.token"), tok); err != nil {
		t.Fatal(err)
	}
	msg := app.StartupBanner(app.BannerInput{
		ListenURL: "http://127.0.0.1:8080",
		Username:  creds.Username,
		Password:  creds.Password,
	})
	for _, want := range []string{"http://127.0.0.1:8080", creds.Username, creds.Password} {
		if !strings.Contains(msg, want) {
			t.Fatalf("banner missing %q:\n%s", want, msg)
		}
	}
	h := adminhost.NewHandler(adminhost.Options{})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != "ok" {
		t.Fatalf("healthz=%d %q", rec.Code, rec.Body.String())
	}
}
