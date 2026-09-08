package webembed_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Mr9esx/Pixoma/apps/pixoma/internal/webembed"
)

func TestHandler_ServesIndex(t *testing.T) {
	h := webembed.Handler()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Pixoma") {
		t.Fatalf("body=%s", rec.Body.String())
	}
}

func TestHandler_ApiPathReturnsJSONNotFound(t *testing.T) {
	h := webembed.Handler()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/stats/tasks/daily", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Fatalf("content-type=%q", ct)
	}
	if !strings.Contains(rec.Body.String(), `"error"`) {
		t.Fatalf("body=%s", rec.Body.String())
	}
}
