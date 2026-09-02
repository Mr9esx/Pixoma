package domain_test

import (
	"encoding/json"
	"testing"

	"github.com/mr9esx/comfyui_tgbot/internal/menucard/domain"
)

func TestMenuJSONRoundTrip(t *testing.T) {
	m := domain.Menu{
		ID: "menu-1", Name: "主菜单", Columns: 2,
		Items: []domain.MenuItem{
			{ID: "mi-1", Label: "图片生成", Action: domain.Action{Type: "open_card", CardID: "card-1"}},
			{ID: "mi-2", Label: "充值", Action: domain.Action{Type: "send_text", Text: "即将上线"}},
		},
	}
	raw, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	var got domain.Menu
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Items) != 2 || got.Items[0].Action.CardID != "card-1" {
		t.Fatalf("got=%+v", got)
	}
}

func TestValidateActionRules(t *testing.T) {
	if err := domain.ValidateAction(domain.Action{Type: "open_card"}); err == nil {
		t.Fatal("open_card without card_id must fail")
	}
	if err := domain.ValidateAction(domain.Action{Type: "open_workflow", WorkflowID: ""}); err == nil {
		t.Fatal("open_workflow without workflows must fail")
	}
	if err := domain.ValidateAction(domain.Action{Type: "open_url", URL: "ftp://x"}); err == nil {
		t.Fatal("non-http url must fail")
	}
	if err := domain.ValidateAction(domain.Action{Type: "send_text", Text: "hi"}); err != nil {
		t.Fatalf("placeholder must pass: %v", err)
	}
	if err := domain.ValidateAction(domain.Action{Type: "list_tasks"}); err != nil {
		t.Fatalf("list_tasks must pass: %v", err)
	}
}
