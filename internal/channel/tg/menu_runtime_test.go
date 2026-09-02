package tg

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mr9esx/comfyui_tgbot/internal/channel/capability"
	"github.com/mr9esx/comfyui_tgbot/internal/channel/ports"
	"github.com/mr9esx/comfyui_tgbot/internal/channel/protocol"
	mcdomain "github.com/mr9esx/comfyui_tgbot/internal/menucard/domain"
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

type captureOutbound struct {
	lists         [][][]ports.Button
	mediaCaptions []string
	mediaButtons  [][][]ports.Button
	edited        []int
}

func (c *captureOutbound) SendText(context.Context, sharedkernel.ChannelAddr, string) error {
	return nil
}
func (c *captureOutbound) SendMenu(context.Context, sharedkernel.ChannelAddr, string, []ports.MenuEntry) error {
	return nil
}
func (c *captureOutbound) SendList(_ context.Context, _ sharedkernel.ChannelAddr, _ string, rows [][]ports.Button) error {
	c.lists = append(c.lists, rows)
	return nil
}
func (c *captureOutbound) SendMedia(_ context.Context, _ sharedkernel.ChannelAddr, _ sharedkernel.BlobRef, caption string, buttons [][]ports.Button) error {
	c.mediaCaptions = append(c.mediaCaptions, caption)
	c.mediaButtons = append(c.mediaButtons, buttons)
	return nil
}
func (c *captureOutbound) SendMediaURL(context.Context, sharedkernel.ChannelAddr, string, string, [][]ports.Button) error {
	return nil
}
func (c *captureOutbound) EditReplyMarkup(_ context.Context, _ sharedkernel.ChannelAddr, messageID int, _ [][]ports.Button) error {
	c.edited = append(c.edited, messageID)
	return nil
}

func TestRenderResultBackButton(t *testing.T) {
	out := &captureOutbound{}
	ad := New(out)
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
	if len(out.lists) != 1 {
		t.Fatalf("lists=%d", len(out.lists))
	}
	rows := out.lists[0]
	if len(rows) != 2 || rows[0][0].Data == "" || !isInvokeData(rows[0][0].Data) {
		t.Fatalf("row0=%+v", rows[0])
	}
	if rows[1][0].Text != "⬅️ 返回" || rows[1][0].Data != "mb:group-1" {
		t.Fatalf("back=%+v", rows[1])
	}

	// root nav → 返回主菜单
	base.Nav.Back = "root"
	out.lists = nil
	if err := ad.renderResult(context.Background(), addr, "tg-default:1", base, res); err != nil {
		t.Fatal(err)
	}
	if out.lists[0][1][0].Data != CBMenu {
		t.Fatalf("root back=%+v", out.lists[0][1])
	}
}

func TestRenderResultMediaWithOptionsDoesNotRepeatText(t *testing.T) {
	out := &captureOutbound{}
	ad := New(out)
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
	if len(out.lists) != 0 {
		t.Fatalf("list must not repeat caption: lists=%d", len(out.lists))
	}
	if len(out.mediaCaptions) != 1 || out.mediaCaptions[0] != "📎 Flux\n好工作流" {
		t.Fatalf("captions=%q", out.mediaCaptions)
	}
	if len(out.mediaButtons) != 1 || len(out.mediaButtons[0]) != 2 {
		t.Fatalf("buttons=%+v", out.mediaButtons)
	}
	if out.mediaButtons[0][0][0].Text != "▶ 开始 Case" {
		t.Fatalf("start=%+v", out.mediaButtons[0][0])
	}
}

func isInvokeData(d string) bool { return len(d) > len(CBInvoke) && d[:len(CBInvoke)] == CBInvoke }

type mediaURLOutbound struct {
	mediaURLs []string
}

func (m *mediaURLOutbound) SendText(context.Context, sharedkernel.ChannelAddr, string) error {
	return nil
}
func (m *mediaURLOutbound) SendMenu(context.Context, sharedkernel.ChannelAddr, string, []ports.MenuEntry) error {
	return nil
}
func (m *mediaURLOutbound) SendList(context.Context, sharedkernel.ChannelAddr, string, [][]ports.Button) error {
	return nil
}
func (m *mediaURLOutbound) SendMedia(context.Context, sharedkernel.ChannelAddr, sharedkernel.BlobRef, string, [][]ports.Button) error {
	return nil
}
func (m *mediaURLOutbound) SendMediaURL(_ context.Context, _ sharedkernel.ChannelAddr, imageURL, _ string, _ [][]ports.Button) error {
	m.mediaURLs = append(m.mediaURLs, imageURL)
	return nil
}
func (m *mediaURLOutbound) EditReplyMarkup(context.Context, sharedkernel.ChannelAddr, int, [][]ports.Button) error {
	return nil
}

func TestRenderResultSendsMediaURLs(t *testing.T) {
	out := &mediaURLOutbound{}
	ad := New(out)
	addr := sharedkernel.ChannelAddr{ChannelID: "tg-default", ExternalChatID: "1"}
	res := protocol.Result{
		Text:      "联系方式",
		MediaURLs: []string{"https://a/qr.png", "https://a/b.png"},
	}
	if err := ad.renderResult(context.Background(), addr, "tg-default:1", protocol.CapabilityInvoke{}, res); err != nil {
		t.Fatal(err)
	}
	if len(out.mediaURLs) != 2 || out.mediaURLs[0] != "https://a/qr.png" {
		t.Fatalf("mediaURLs=%q", out.mediaURLs)
	}
}

type textCaptureOutbound struct {
	texts []string
}

func (c *textCaptureOutbound) SendText(_ context.Context, _ sharedkernel.ChannelAddr, text string) error {
	c.texts = append(c.texts, text)
	return nil
}
func (c *textCaptureOutbound) SendMenu(context.Context, sharedkernel.ChannelAddr, string, []ports.MenuEntry) error {
	return nil
}
func (c *textCaptureOutbound) SendList(context.Context, sharedkernel.ChannelAddr, string, [][]ports.Button) error {
	return nil
}
func (c *textCaptureOutbound) SendMedia(context.Context, sharedkernel.ChannelAddr, sharedkernel.BlobRef, string, [][]ports.Button) error {
	return nil
}
func (c *textCaptureOutbound) SendMediaURL(context.Context, sharedkernel.ChannelAddr, string, string, [][]ports.Button) error {
	return nil
}
func (c *textCaptureOutbound) EditReplyMarkup(context.Context, sharedkernel.ChannelAddr, int, [][]ports.Button) error {
	return nil
}

func TestActionDispatchOpenCardSendsCard(t *testing.T) {
	out := &captureOutbound{}
	ad := New(out)
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
	if len(out.lists) != 1 {
		t.Fatalf("lists=%d", len(out.lists))
	}
	rows := out.lists[0]
	if rows[0][0].Text != "写实" {
		t.Fatalf("row0=%+v", rows[0])
	}
	if rows[len(rows)-1][0].Text != "‹ 返回" || rows[len(rows)-1][0].Data != CBMenuBack+"root" {
		t.Fatalf("back=%+v", rows[len(rows)-1])
	}
}

func TestActionDispatchSendText(t *testing.T) {
	out := &textCaptureOutbound{}
	ad := New(out)
	addr := sharedkernel.ChannelAddr{ChannelID: "tg-default", ExternalChatID: "1"}
	btn := mcdomain.TreeButton{ID: "t", Label: "x", Action: mcdomain.TreeAction{Type: "send_text", Text: "即将上线"}}
	if err := ad.actionDispatch(context.Background(), sharedkernel.ChatID("tg-default:1"), addr, btn, "root"); err != nil {
		t.Fatal(err)
	}
	if len(out.texts) != 1 || out.texts[0] != "即将上线" {
		t.Fatalf("texts=%q", out.texts)
	}
}

func TestBackChainTracksSources(t *testing.T) {
	ad := New(&captureOutbound{})
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
	out := &captureOutbound{}
	cap := &recordOpenCase{}
	reg := capability.NewRegistry()
	if err := reg.Register(cap); err != nil {
		t.Fatal(err)
	}
	ad := New(out)
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
	if len(out.lists) != 1 {
		t.Fatalf("lists=%d", len(out.lists))
	}
}
