package server_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mr9esx/comfyui_tgbot/apps/bot/internal/server"
)

func TestBotServer_HealthzOnly(t *testing.T) {
	h := server.NewHandler()

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("healthz status=%d", rec.Code)
	}
	if body := strings.TrimSpace(rec.Body.String()); body != "ok" {
		t.Fatalf("healthz body=%q", body)
	}

	for _, path := range []string{
		"/api/v1/cases", "/api/v1/users", "/api/v1/sessions", "/api/v1/tasks", "/api/v1/edges",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("%s status=%d, want 404", path, rec.Code)
		}
	}
}
