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
	reg.Register(&condition.UserProvider{Lookup: nil})
	reg.Register(&condition.CaseProvider{Lookup: nil})

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
	if len(body.Attributes) != 3 {
		t.Fatalf("attributes = %d, want 3", len(body.Attributes))
	}
	keys := map[string]bool{}
	for _, a := range body.Attributes {
		key, _ := a["key"].(string)
		keys[key] = true
		if a["schema"] == nil || a["label"] == nil || a["context"] == nil {
			t.Fatalf("attribute %s missing fields: %+v", key, a)
		}
	}
	for _, want := range []string{"user.is_premium", "case.category", "case.tags"} {
		if !keys[want] {
			t.Fatalf("missing attribute %s: %v", want, keys)
		}
	}
}
