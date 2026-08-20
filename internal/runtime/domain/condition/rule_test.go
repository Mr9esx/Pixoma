package condition

import (
	"encoding/json"
	"testing"
)

func TestParseRule_Leaf(t *testing.T) {
	raw := json.RawMessage(`{"field":"user.is_premium","op":"eq","value":true}`)
	r, err := ParseRule(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if r.Field != "user.is_premium" || r.Op != "eq" || r.Value != true {
		t.Fatalf("rule = %+v", r)
	}
}

func TestParseRule_Combinators(t *testing.T) {
	raw := json.RawMessage(`{"and":[{"field":"a","op":"eq","value":1},{"or":[{"field":"b","op":"in","value":["x"]}]}]}`)
	r, err := ParseRule(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(r.And) != 2 || len(r.And[1].Or) != 1 {
		t.Fatalf("rule = %+v", r)
	}
}

func TestParseRule_Errors(t *testing.T) {
	cases := []string{
		`{}`,
		`{"field":"x"}`,
		`{"op":"eq","value":1}`,
		`{"and":[]}`,
		`{"or":[]}`,
		`{"and":[{}],"field":"x"}`,
		`{"field":"x","op":"eq","value":1,"and":[{"field":"y","op":"eq","value":2}]}`,
		`"not-an-object"`,
	}
	for _, c := range cases {
		if _, err := ParseRule(json.RawMessage(c)); err == nil {
			t.Fatalf("expected error for %s", c)
		}
	}
}

func TestValidateRule(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&stubProvider{ns: "user", attrs: []AttributeDescriptor{
		{Key: "user.is_premium", Context: "user", Label: "Premium", Schema: map[string]any{"type": "boolean"}},
		{Key: "user.level", Context: "user", Label: "Level", Schema: map[string]any{"type": "number"}},
	}})
	reg.Register(&stubProvider{ns: "case", attrs: []AttributeDescriptor{
		{Key: "case.category", Context: "case", Label: "Category", Schema: map[string]any{"type": "string", "enum": []any{"image", "video"}}},
	}})

	ok := []string{
		`{"field":"user.is_premium","op":"eq","value":true}`,
		`{"and":[{"field":"user.level","op":"gt","value":1},{"field":"case.category","op":"in","value":["image"]}]}`,
		`{"field":"user.is_premium","op":"exists"}`,
	}
	for _, c := range ok {
		r, err := ParseRule(json.RawMessage(c))
		if err != nil {
			t.Fatalf("parse %s: %v", c, err)
		}
		if err := ValidateRule(r, reg); err != nil {
			t.Fatalf("validate %s: %v", c, err)
		}
	}

	bad := []string{
		`{"field":"user.unknown","op":"eq","value":true}`,
		`{"field":"user.is_premium","op":"like","value":true}`,
		`{"field":"user.is_premium","op":"eq","value":"yes"}`,
		`{"field":"case.category","op":"in","value":["video","poster"]}`,
	}
	for _, c := range bad {
		r, err := ParseRule(json.RawMessage(c))
		if err != nil {
			continue // parse-level rejection is also fine
		}
		if err := ValidateRule(r, reg); err == nil {
			t.Fatalf("expected validate error for %s", c)
		}
	}
}
