package app_test

import (
	"strings"
	"testing"

	"github.com/mr9esx/comfyui_tgbot/apps/pixoma/internal/app"
)

func TestStartupBanner_IncludesURLAndCredentials(t *testing.T) {
	msg := app.StartupBanner(app.BannerInput{
		ListenURL: "http://127.0.0.1:8080",
		Username:  "admin",
		Password:  "secret-pass",
	})
	if !strings.Contains(msg, "http://127.0.0.1:8080") {
		t.Fatalf("missing URL: %s", msg)
	}
	if !strings.Contains(msg, "admin") || !strings.Contains(msg, "secret-pass") {
		t.Fatalf("missing credentials: %s", msg)
	}
}

func TestStartupBanner_OmitsPasswordWhenEmpty(t *testing.T) {
	msg := app.StartupBanner(app.BannerInput{
		ListenURL: "http://127.0.0.1:8080",
		Username:  "admin",
	})
	if strings.Contains(strings.ToLower(msg), "password:") && strings.Contains(msg, "admin") {
		// password line should not show empty secret
	}
	if strings.Contains(msg, "Password:") {
		t.Fatalf("must not print password line when empty: %s", msg)
	}
}
