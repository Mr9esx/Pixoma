package domain

import (
	"encoding/json"
	"testing"
)

func TestCaseDocumentRoutingRoundTrip(t *testing.T) {
	doc := CaseDocument{
		ID:         1,
		Name:       "case-a",
		Routing: &RoutingConfig{
			Rules: []RoutingRule{
				{When: json.RawMessage(`{"field":"user.level","op":"eq","value":"image"}`), Topic: "fast-gpu"},
			},
		},
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var back CaseDocument
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if back.Routing == nil || len(back.Routing.Rules) != 1 || back.Routing.Rules[0].Topic != "fast-gpu" {
		t.Fatalf("routing roundtrip failed: %+v", back.Routing)
	}
}

func TestCaseDocumentRoutingOptional(t *testing.T) {
	raw := []byte(`{"id":2,"name":"no-routing","bindings":{"workflow":{}},"input_schema":{}}`)
	var doc CaseDocument
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if doc.Routing != nil {
		t.Fatalf("expected nil routing, got %+v", doc.Routing)
	}
}
