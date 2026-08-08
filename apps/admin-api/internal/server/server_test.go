package server_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mr9esx/comfyui_tgbot/apps/admin-api/internal/server"
)

func TestNewHandler_Healthz(t *testing.T) {
	h := server.NewHandler(server.Options{
		CORSOrigins: []string{"http://localhost:5173"},
	})
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", rec.Code)
	}
	if body := strings.TrimSpace(rec.Body.String()); body != "ok" {
		t.Fatalf("body=%q, want ok", body)
	}
}

func TestNewHandler_CORSPreflight(t *testing.T) {
	origin := "http://localhost:5173"
	h := server.NewHandler(server.Options{
		CORSOrigins: []string{origin},
	})
	req := httptest.NewRequest(http.MethodOptions, "/healthz", nil)
	req.Header.Set("Origin", origin)
	req.Header.Set("Access-Control-Request-Method", "GET")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != origin {
		t.Fatalf("Access-Control-Allow-Origin=%q, want %q", got, origin)
	}
	if rec.Code != http.StatusOK && rec.Code != http.StatusNoContent {
		t.Fatalf("preflight status=%d", rec.Code)
	}
}

func TestNewHandler_EmptyComfyInstancesRouteGroup(t *testing.T) {
	h := server.NewHandler(server.Options{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/comfy-instances", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	// Empty mount group: chi returns 404 until Task 3 mounts handlers.
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d, want 404 for empty route group", rec.Code)
	}
}
