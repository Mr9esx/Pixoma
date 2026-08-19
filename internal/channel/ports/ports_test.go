package ports

import (
	"encoding/json"
	"testing"
)

func TestButtonJSONRoundTrip(t *testing.T) {
	b := Button{Text: "开 Case", Data: "inv:abc123"}
	raw, err := json.Marshal(b)
	if err != nil {
		t.Fatal(err)
	}
	var got Button
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got.Text != "开 Case" || got.Data != "inv:abc123" {
		t.Fatalf("roundtrip: %+v", got)
	}
}

func TestMenuEntry(t *testing.T) {
	e := MenuEntry{ID: "btn-image", Label: "🖼 图片"}
	if e.ID != "btn-image" || e.Label == "" {
		t.Fatalf("entry=%+v", e)
	}
}
