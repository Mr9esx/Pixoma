package webembed_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mr9esx/comfyui_tgbot/apps/pixoma/internal/webembed"
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
