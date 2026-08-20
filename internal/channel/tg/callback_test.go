package tg

import (
	"testing"

	"github.com/mr9esx/comfyui_tgbot/internal/channel/protocol"
)

func TestTranslateMenuCallback(t *testing.T) {
	cases := []struct {
		data string
		want navTarget
	}{
		{CBMenu, navTarget{kind: "main"}},
		{"mb:root", navTarget{kind: "main"}},
		{"mb:card-1", navTarget{kind: "back", id: "card-1"}},
	}
	for _, tc := range cases {
		got, err := TranslateMenuCallback(tc.data)
		if err != nil {
			t.Fatalf("%q: %v", tc.data, err)
		}
		if got != tc.want {
			t.Fatalf("%q: got %+v want %+v", tc.data, got, tc.want)
		}
	}
	if _, err := TranslateMenuCallback("bogus"); err == nil {
		t.Fatal("unknown nav must error")
	}
}

func TestInvokeStorePutGet(t *testing.T) {
	s := newInvokeStore()
	inv := protocol.CapabilityInvoke{CapabilityID: "open_case", Params: map[string]any{"step": "preview", "case_id": "c1"}}
	token := s.put(inv)
	if token == "" {
		t.Fatal("empty token")
	}
	got, ok := s.get(token)
	if !ok || got.CapabilityID != "open_case" {
		t.Fatalf("get: %+v %v", got, ok)
	}
	// 一次性消费
	if _, ok := s.get(token); ok {
		t.Fatal("token must be single-use")
	}
}
