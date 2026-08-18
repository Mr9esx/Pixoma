package tg

import (
	"testing"

	"github.com/mr9esx/comfyui_tgbot/internal/channel/ports"
	"github.com/mr9esx/comfyui_tgbot/internal/menu/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

func TestTranslateCallback(t *testing.T) {
	cases := []struct {
		data string
		want ports.Action
	}{
		{CBMenu, ports.Action{Type: ports.ActionOpenMenu}},
		{CBConfirm, ports.Action{Type: ports.ActionConfirm}},
		{CBExit, ports.Action{Type: ports.ActionExit}},
		{CBSkip, ports.Action{Type: ports.ActionSkip}},
		{CBContinue, ports.Action{Type: ports.ActionContinue}},
		{"mf:btn-image", ports.Action{Type: ports.ActionOpenFolder, MenuItemID: "btn-image"}},
		{"mb:root", ports.Action{Type: ports.ActionOpenMenu}},
		{"mb:parent-1", ports.Action{Type: ports.ActionOpenFolder, MenuItemID: "parent-1"}},
		{"cp:case-1", ports.Action{Type: ports.ActionOpenCase, CaseID: "case-1"}},
		{"cpf:folder-1:case-2", ports.Action{Type: ports.ActionOpenCase, CaseID: "case-2", BackRef: "folder-1"}},
		{"cs:case-3", ports.Action{Type: ports.ActionStartCase, CaseID: "case-3"}},
		{"rs:case-4", ports.Action{Type: ports.ActionReplaceStart, CaseID: "case-4"}},
	}
	for _, tc := range cases {
		got, err := TranslateCallback(tc.data)
		if err != nil {
			t.Fatalf("%q: %v", tc.data, err)
		}
		if got != tc.want {
			t.Fatalf("%q: got %+v want %+v", tc.data, got, tc.want)
		}
	}
	if _, err := TranslateCallback("bogus"); err == nil {
		t.Fatal("unknown callback must error")
	}
	if _, err := TranslateCallback("mf:"); err == nil {
		t.Fatal("empty folder must error")
	}
}

func TestEncodeActionRoundTrip(t *testing.T) {
	actions := []ports.Action{
		{Type: ports.ActionOpenFolder, MenuItemID: "f1"},
		{Type: ports.ActionOpenCase, CaseID: "c1", BackRef: "f1"},
		{Type: ports.ActionOpenCase, CaseID: "c1", BackRef: ""}, // "" 与 "root" 编码相同，解码后为空
		{Type: ports.ActionStartCase, CaseID: "c1"},
		{Type: ports.ActionReplaceStart, CaseID: "c1"},
		{Type: ports.ActionOpenMenu},
		{Type: ports.ActionConfirm},
	}
	for _, a := range actions {
		got, err := TranslateCallback(encodeAction(a))
		if err != nil {
			t.Fatalf("%+v: %v", a, err)
		}
		if got != a {
			t.Fatalf("roundtrip %+v != %+v", got, a)
		}
	}
}

func TestRootColumnsFromExtras(t *testing.T) {
	extras := map[string][]domain.Extra{
		"btn-image": {{ExtraType: "tg_root_layout", ExtraJSON: `{"columns":3}`}},
		"btn-video": {{ExtraType: "tg_root_layout", ExtraJSON: `{"columns":99}`}},
	}
	if got := RootColumns(extras, "btn-image"); got != 3 {
		t.Fatalf("columns=%d", got)
	}
	if got := RootColumns(extras, "btn-video"); got != 2 {
		t.Fatalf("invalid columns should fall back: %d", got)
	}
	if got := RootColumns(extras, "missing"); got != 2 {
		t.Fatalf("missing extras: %d", got)
	}
}

func TestChatAddressMapping(t *testing.T) {
	ad := &Adapter{ChannelID: "tg-default"}
	chatID := formatChatID(ad, 123456789)
	addr, err := sharedkernel.ParseChatID(string(chatID))
	if err != nil {
		t.Fatal(err)
	}
	if addr.ChannelID != "tg-default" || addr.ExternalChatID != "123456789" {
		t.Fatalf("addr=%+v", addr)
	}
}
