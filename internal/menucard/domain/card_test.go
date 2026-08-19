package domain_test

import (
	"encoding/json"
	"testing"

	"github.com/mr9esx/comfyui_tgbot/internal/menucard/domain"
)

func TestCardJSONRoundTrip(t *testing.T) {
	c := domain.Card{
		ID: "card-1", Name: "开始生成", Text: "选一种风格：",
		Media: []domain.Media{{Kind: "image", URL: "https://a/img.png"}},
		Buttons: []domain.CardButton{
			{ID: "cb-1", Label: "写实风格", Action: domain.Action{Type: "open_workflow", WorkflowIDs: []string{"w1"}}},
		},
	}
	raw, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	var got domain.Card
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got.Media[0].URL != "https://a/img.png" || got.Buttons[0].Action.WorkflowIDs[0] != "w1" {
		t.Fatalf("got=%+v", got)
	}
}

func TestValidateCardRequiresContent(t *testing.T) {
	if err := domain.ValidateCard(domain.Card{ID: "c", Name: "x"}); err == nil {
		t.Fatal("empty card must fail")
	}
	ok := domain.Card{ID: "c", Name: "x", Text: "hi", Buttons: []domain.CardButton{
		{ID: "b", Label: "B", Action: domain.Action{Type: "placeholder"}},
	}}
	if err := domain.ValidateCard(ok); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	noButtons := domain.Card{ID: "c2", Name: "终局", Media: []domain.Media{{Kind: "image", URL: "https://a/x.png"}}}
	if err := domain.ValidateCard(noButtons); err != nil {
		t.Fatalf("card without buttons must pass (auto back still applies): %v", err)
	}
}
