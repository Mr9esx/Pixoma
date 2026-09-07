package routing_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/mr9esx/comfyui_tgbot/internal/httpapi/routing"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/domain/condition"
)

func TestAttributesCatalog(t *testing.T) {
	reg := condition.NewRegistry()

	h := &routing.Handler{Registry: reg}
	r := chi.NewRouter()
	r.Route("/api/v1/routing", h.Mount)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/routing/attributes", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		Attributes []map[string]any `json:"attributes"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Attributes) != 0 {
		t.Fatalf("attributes = %d, want 0", len(body.Attributes))
	}
}
