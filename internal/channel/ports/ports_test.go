package ports

import (
	"encoding/json"
	"testing"

	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

func TestActionJSONRoundTrip(t *testing.T) {
	a := Action{Type: ActionOpenFolder, MenuItemID: "btn-image"}
	raw, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	var got Action
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got != a {
		t.Fatalf("roundtrip: %+v != %+v", got, a)
	}
}

func TestActionValidate(t *testing.T) {
	if err := (Action{Type: ActionOpenFolder, MenuItemID: "f1"}).Validate(); err != nil {
		t.Fatalf("open_folder ok: %v", err)
	}
	if err := (Action{Type: ActionOpenFolder}).Validate(); err == nil {
		t.Fatal("open_folder without item must fail")
	}
	if err := (Action{Type: ActionOpenCase, CaseID: "c1"}).Validate(); err != nil {
		t.Fatalf("open_case ok: %v", err)
	}
	if err := (Action{Type: ActionStartCase}).Validate(); err == nil {
		t.Fatal("start_case without case must fail")
	}
	if err := (Action{Type: "bogus"}).Validate(); err == nil {
		t.Fatal("unknown type must fail")
	}
}

func TestInboundEventAddr(t *testing.T) {
	ev := InboundEvent{
		Addr: sharedkernel.ChannelAddr{ChannelID: "tg-default", ExternalChatID: "123"},
		Kind: EventCallback,
		Action: Action{Type: ActionOpenMenu},
	}
	if ev.Addr.ChannelID != "tg-default" || ev.Addr.ExternalChatID != "123" {
		t.Fatalf("addr: %+v", ev.Addr)
	}
}
