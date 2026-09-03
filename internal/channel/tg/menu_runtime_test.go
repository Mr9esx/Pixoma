package tg

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/mr9esx/comfyui_tgbot/internal/channel/capability"
	"github.com/mr9esx/comfyui_tgbot/internal/channel/protocol"
	"github.com/mr9esx/comfyui_tgbot/internal/channel/tg/tginternal"
	mcdomain "github.com/mr9esx/comfyui_tgbot/internal/menucard/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/blob"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/blob/localfs"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

func TestBuildReplyKeyboardFreeColumns(t *testing.T) {
	menu := mcdomain.MenuTree{
		ID: "m", Columns: 3,
		Items: []mcdomain.TreeButton{
			{ID: "a", Label: "A", Action: mcdomain.TreeAction{Type: "send_text", Text: "-"}},
			{ID: "b", Label: "B", Action: mcdomain.TreeAction{Type: "send_text", Text: "-"}},
			{ID: "c", Label: "C", Action: mcdomain.TreeAction{Type: "send_text", Text: "-"}},
			{ID: "d", Label: "D", Action: mcdomain.TreeAction{Type: "send_text", Text: "-"}},
			{ID: "e", Label: "E", Action: mcdomain.TreeAction{Type: "send_text", Text: "-"}},
		},
	}
	kb := BuildReplyKeyboard(menu)
	if len(kb.Keyboard) != 2 {
		t.Fatalf("rows=%d kb=%+v", len(kb.Keyboard), kb.Keyboard)
	}
	if len(kb.Keyboard[0]) != 3 || len(kb.Keyboard[1]) != 2 {
		t.Fatalf("layout=%+v", kb.Keyboard)
	}
}

func TestFindEnabledItemByLabel(t *testing.T) {
	menu := mcdomain.MenuTree{Items: []mcdomain.TreeButton{
		{ID: "a", Label: "图片", Action: mcdomain.TreeAction{Type: "open_card", Card: &mcdomain.TreeCard{Text: "c"}}},
	}}
	it, ok := FindEnabledItemByLabel(menu, "图片")
	if !ok || it.Action.Card == nil {
		t.Fatalf("item=%+v ok=%v", it, ok)
	}
}

func TestRenderResultBackButton(t *testing.T) {
	b, srv := tginternal.NewBot(t)
	ad := New(&BotMessenger{Bot: b})
	addr := sharedkernel.ChannelAddr{ChannelID: "tg-default", ExternalChatID: "1"}

	base := protocol.CapabilityInvoke{
		CapabilityID: "open_case",
		Account:      protocol.AccountCtx{ChannelID: "tg-default", ExternalUserID: "1"},
		Nav:          protocol.Nav{Back: "group-1"},
		ChatID:       "tg-default:1",
	}
	res := protocol.Result{
		Text:    "请选择",
		Options: []protocol.Option{{Label: "图片 A", Value: map[string]any{"step": "preview"}}},
	}
	if err := ad.renderResult(context.Background(), addr, "tg-default:1", base, res); err != nil {
		t.Fatal(err)
	}
	texts, data := srv.LastInlineKeyboardTexts(), srv.LastInlineKeyboardData()
	if len(texts) != 2 {
		t.Fatalf("rows=%d", len(texts))
	}
	if len(texts[0]) == 0 || !isInvokeData(data[0][0]) {
		t.Fatalf("row0=%+v", texts[0])
	}
	if texts[1][0] != "⬅️ 返回" || data[1][0] != "mb:group-1" {
		t.Fatalf("back=%+v data=%+v", texts[1], data[1])
	}

	// root nav → 返回主菜单
	base.Nav.Back = "root"
	if err := ad.renderResult(context.Background(), addr, "tg-default:1", base, res); err != nil {
		t.Fatal(err)
	}
	texts, data = srv.LastInlineKeyboardTexts(), srv.LastInlineKeyboardData()
	if data[len(data)-1][0] != CBMenu {
		t.Fatalf("root back=%+v", data[len(data)-1])
	}
}

func TestRenderResultMediaWithOptionsDoesNotRepeatText(t *testing.T) {
	ctx := context.Background()
	store, err := localfs.New(filepath.Join(t.TempDir(), "blob"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Put(ctx, "previews/a.png", bytes.NewReader([]byte("png-bytes")), blob.PutOptions{MIME: "image/png"}); err != nil {
		t.Fatal(err)
	}
	b, srv := tginternal.NewBot(t)
	messenger := &BotMessenger{Bot: b, Blob: store}
	ad := New(messenger)
	addr := sharedkernel.ChannelAddr{ChannelID: "tg-default", ExternalChatID: "1"}
	base := protocol.CapabilityInvoke{
		CapabilityID: "open_case",
		Nav:          protocol.Nav{Back: "root"},
		ChatID:       "tg-default:1",
	}
	res := protocol.Result{
		Text:    "📎 Flux\n好工作流",
		Media:   []protocol.MediaRef{{Key: "previews/a.png", MIME: "image/png"}},
		Options: []protocol.Option{{Label: "▶ 开始 Case", Value: map[string]any{"step": "start"}}},
	}
	if err := ad.renderResult(context.Background(), addr, "tg-default:1", base, res); err != nil {
		t.Fatal(err)
	}
	// Media path: 1 sendPhoto, no sendMessage follow-up list.
	if got := srv.SendPhotoCalls(); got != 1 {
		t.Fatalf("sendPhoto calls = %d, want 1", got)
	}
	if got := srv.LastCaption(); got != "📎 Flux\n好工作流" {
		t.Fatalf("caption = %q", got)
	}
	texts, _ := srv.LastInlineKeyboardTexts(), srv.LastInlineKeyboardData()
	if len(texts) < 2 {
		t.Fatalf("inline keyboard rows = %+v (want at least 2)", texts)
	}
	// First row is the option button; last row is the back button.
	if texts[0][0] != "▶ 开始 Case" {
		t.Fatalf("start button = %+v", texts[0])
	}
}

func isInvokeData(d string) bool { return len(d) > len(CBInvoke) && d[:len(CBInvoke)] == CBInvoke }

func TestRenderResultSendsMediaURLs(t *testing.T) {
	b, srv := tginternal.NewBot(t)
	ad := New(&BotMessenger{Bot: b})
	addr := sharedkernel.ChannelAddr{ChannelID: "tg-default", ExternalChatID: "1"}
	res := protocol.Result{
		Text:      "联系方式",
		MediaURLs: []string{"https://a/qr.png", "https://a/b.png"},
	}
	if err := ad.renderResult(context.Background(), addr, "tg-default:1", protocol.CapabilityInvoke{}, res); err != nil {
		t.Fatal(err)
	}
	if got := srv.SendPhotoCalls(); got != 2 {
		t.Fatalf("sendPhoto calls = %d, want 2", got)
	}
}

func TestActionDispatchOpenCardSendsCard(t *testing.T) {
	b, srv := tginternal.NewBot(t)
	ad := New(&BotMessenger{Bot: b})
	ad.ChannelID = "ch1"
	addr := sharedkernel.ChannelAddr{ChannelID: "tg-default", ExternalChatID: "1"}
	btn := mcdomain.TreeButton{
		ID:    "open",
		Label: "卡",
		Action: mcdomain.TreeAction{
			Type: "open_card",
			Card: &mcdomain.TreeCard{
				Text: "选一种风格：",
				Buttons: []mcdomain.TreeButton{
					{ID: "b", Label: "写实", Action: mcdomain.TreeAction{Type: "open_workflow", WorkflowID: "w1"}},
				},
			},
		},
	}
	if err := ad.actionDispatch(context.Background(), sharedkernel.ChatID("tg-default:1"), addr, btn, "root"); err != nil {
		t.Fatal(err)
	}
	texts, data := srv.LastInlineKeyboardTexts(), srv.LastInlineKeyboardData()
	if len(texts) == 0 {
		t.Fatalf("no inline keyboard rows")
	}
	if texts[0][0] != "写实" {
		t.Fatalf("row0=%+v", texts[0])
	}
	last := len(texts) - 1
	if texts[last][0] != "‹ 返回" || data[last][0] != CBMenuBack+"root" {
		t.Fatalf("back=%+v", texts[last])
	}
}

func TestActionDispatchSendText(t *testing.T) {
	b, srv := tginternal.NewBot(t)
	ad := New(&BotMessenger{Bot: b})
	addr := sharedkernel.ChannelAddr{ChannelID: "tg-default", ExternalChatID: "1"}
	btn := mcdomain.TreeButton{ID: "t", Label: "x", Action: mcdomain.TreeAction{Type: "send_text", Text: "即将上线"}}
	if err := ad.actionDispatch(context.Background(), sharedkernel.ChatID("tg-default:1"), addr, btn, "root"); err != nil {
		t.Fatal(err)
	}
	texts := srv.SendMessageTexts()
	if len(texts) < 1 || texts[0] != "即将上线" {
		t.Fatalf("sendMessage texts = %v, want first %q", texts, "即将上线")
	}
}

func TestBackChainTracksSources(t *testing.T) {
	b, _ := tginternal.NewBot(t)
	ad := New(&BotMessenger{Bot: b})
	ad.ChannelID = "ch1"
	addr := sharedkernel.ChannelAddr{ChannelID: "tg-default", ExternalChatID: "1"}
	chat := "tg-default:1"
	inner := &mcdomain.TreeCard{Text: "第二张"}
	outer := mcdomain.TreeButton{
		ID: "c1", Label: "一",
		Action: mcdomain.TreeAction{Type: "open_card", Card: &mcdomain.TreeCard{
			Text: "第一张",
			Buttons: []mcdomain.TreeButton{{
				ID: "c2", Label: "二",
				Action: mcdomain.TreeAction{Type: "open_card", Card: inner},
			}},
		}},
	}
	innerBtn := mcdomain.TreeButton{
		ID: "c2", Label: "二",
		Action: mcdomain.TreeAction{Type: "open_card", Card: inner},
	}
	_ = ad.actionDispatch(context.Background(), sharedkernel.ChatID(chat), addr, outer, "root")
	_ = ad.actionDispatch(context.Background(), sharedkernel.ChatID(chat), addr, innerBtn, "c1")
	if top, ok := ad.back.top(chat); !ok || top != "c1" {
		t.Fatalf("top=%q ok=%v", top, ok)
	}
}

type recordOpenCase struct {
	params map[string]any
}

func (recordOpenCase) ID() string          { return "open_case" }
func (recordOpenCase) DisplayName() string { return "打开工作流" }
func (recordOpenCase) ParamsSchema() json.RawMessage {
	return json.RawMessage(`{"type":"object"}`)
}
func (recordOpenCase) Render(string, map[string]any) (protocol.RenderDecl, error) {
	return protocol.RenderDecl{}, nil
}
func (r *recordOpenCase) Invoke(_ context.Context, _ protocol.AccountCtx, _ protocol.Nav, _ sharedkernel.ChatID, params map[string]any) (protocol.Result, error) {
	r.params = params
	return protocol.Result{
		Text:    "📎 图片 B",
		Options: []protocol.Option{{Label: "▶ 开始 Case", Value: map[string]any{"step": "start", "case_id": "10"}}},
	}, nil
}

func TestActionDispatchOpenWorkflowPreviews(t *testing.T) {
	b, srv := tginternal.NewBot(t)
	cap := &recordOpenCase{}
	reg := capability.NewRegistry()
	if err := reg.Register(cap); err != nil {
		t.Fatal(err)
	}
	ad := New(&BotMessenger{Bot: b})
	ad.ChannelID = "tg-default"
	ad.Registry = reg
	addr := sharedkernel.ChannelAddr{ChannelID: "tg-default", ExternalChatID: "1"}
	err := ad.actionDispatch(context.Background(), "tg-default:1", addr, mcdomain.TreeButton{
		ID: "w", Label: "流", Action: mcdomain.TreeAction{Type: "open_workflow", WorkflowID: "10"},
	}, "root")
	if err != nil {
		t.Fatal(err)
	}
	if cap.params["step"] != "preview" {
		t.Fatalf("step=%v want preview", cap.params["step"])
	}
	if cap.params["case_id"] != "10" {
		t.Fatalf("case_id=%v", cap.params["case_id"])
	}
	if got := srv.SendMessageCalls(); got < 1 {
		t.Fatalf("expected at least one sendMessage, got %d", got)
	}
}

func TestSendCardPassesVideoMIMEToSendMediaURL(t *testing.T) {
	b, srv := tginternal.NewBot(t)
	ad := New(&BotMessenger{Bot: b})
	addr := sharedkernel.ChannelAddr{ChannelID: "tg-default", ExternalChatID: "1"}
	card := mcdomain.TreeCard{
		Text:  "卡片正文",
		Media: []mcdomain.Media{{Kind: "video", URL: "https://cdn/x.mp4"}},
	}
	if err := ad.sendCard(context.Background(), addr, card, "", ""); err != nil {
		t.Fatal(err)
	}
	if got := srv.SendVideoCalls(); got != 1 {
		t.Fatalf("sendVideo calls = %d, want 1 (got sendPhoto=%d sendAnimation=%d sendDocument=%d)",
			got, srv.SendPhotoCalls(), srv.SendAnimationCalls(), srv.SendDocumentCalls())
	}
}

func TestSendCardPassesAnimationMIMEToSendMediaURL(t *testing.T) {
	b, srv := tginternal.NewBot(t)
	ad := New(&BotMessenger{Bot: b})
	addr := sharedkernel.ChannelAddr{ChannelID: "tg-default", ExternalChatID: "1"}
	card := mcdomain.TreeCard{
		Media: []mcdomain.Media{{Kind: "animation", URL: "https://cdn/x.gif"}},
	}
	if err := ad.sendCard(context.Background(), addr, card, "", ""); err != nil {
		t.Fatal(err)
	}
	if got := srv.SendAnimationCalls(); got != 1 {
		t.Fatalf("sendAnimation calls = %d, want 1 (got sendPhoto=%d sendVideo=%d sendDocument=%d)",
			got, srv.SendPhotoCalls(), srv.SendVideoCalls(), srv.SendDocumentCalls())
	}
}
