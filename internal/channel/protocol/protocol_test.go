package protocol

import (
	"encoding/json"
	"testing"
)

func TestCapabilityInvokeJSONRoundTrip(t *testing.T) {
	inv := CapabilityInvoke{
		CapabilityID: "open_case",
		Params:       map[string]any{"case_ids": []string{"c1", "c2"}},
		Account: AccountCtx{
			ChannelID:      "tg-default",
			ExternalUserID: "1001",
			InternalUserID: "u-1",
		},
		Nav: Nav{Back: "root"},
	}
	raw, err := json.Marshal(inv)
	if err != nil {
		t.Fatal(err)
	}
	var got CapabilityInvoke
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got.CapabilityID != inv.CapabilityID || got.Account != inv.Account || got.Nav != inv.Nav {
		t.Fatalf("roundtrip mismatch: %+v != %+v", got, inv)
	}
	ids, ok := got.Params["case_ids"].([]any)
	if !ok || len(ids) != 2 || ids[0] != "c1" {
		t.Fatalf("params case_ids: %+v", got.Params)
	}
}

func TestResultOptionsAndMedia(t *testing.T) {
	opt := Option{Label: "图片 A · ¥10", Value: map[string]any{"case_id": "c1", "back": "video"}}
	res := Result{
		Text:    "请选择模板",
		Options: []Option{opt},
		Media:   []MediaRef{{Key: "out.png", MIME: "image/png"}},
	}
	raw, err := json.Marshal(res)
	if err != nil {
		t.Fatal(err)
	}
	var got Result
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Options) != 1 || got.Options[0].Label != "图片 A · ¥10" {
		t.Fatalf("options=%+v", got.Options)
	}
	if len(got.Media) != 1 || got.Media[0].Key != "out.png" {
		t.Fatalf("media=%+v", got.Media)
	}
}
