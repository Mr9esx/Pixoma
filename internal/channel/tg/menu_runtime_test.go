package tg

import (
	"context"
	"testing"

	"github.com/mr9esx/comfyui_tgbot/internal/channel/ports"
	"github.com/mr9esx/comfyui_tgbot/internal/channel/protocol"
	mcdomain "github.com/mr9esx/comfyui_tgbot/internal/menucard/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

func TestBuildReplyKeyboardFreeColumns(t *testing.T) {
	menu := mcdomain.Menu{ID: "m", Name: "主", Columns: 3, Items: []mcdomain.MenuItem{
		{ID: "a", Label: "A", Action: mcdomain.Action{Type: "send_text", Text: "-"}},
		{ID: "b", Label: "B", Action: mcdomain.Action{Type: "send_text", Text: "-"}},
		{ID: "c", Label: "C", Action: mcdomain.Action{Type: "send_text", Text: "-"}},
		{ID: "d", Label: "D", Action: mcdomain.Action{Type: "send_text", Text: "-"}},
		{ID: "e", Label: "E", Action: mcdomain.Action{Type: "send_text", Text: "-"}},
		{ID: "f", Label: "F", Action: mcdomain.Action{Type: "send_text", Text: "-"}},
		{ID: "g", Label: "G", Action: mcdomain.Action{Type: "send_text", Text: "-"}},
	}}
	kb := BuildReplyKeyboard(menu)
	if len(kb.Keyboard) != 3 { // 3+3+1，7 个按钮不设上限
		t.Fatalf("rows=%d kb=%+v", len(kb.Keyboard), kb.Keyboard)
	}
	if len(kb.Keyboard[0]) != 3 || len(kb.Keyboard[2]) != 1 {
		t.Fatalf("layout=%+v", kb.Keyboard)
	}
}

func TestFindEnabledItemByLabel(t *testing.T) {
	menu := mcdomain.Menu{Items: []mcdomain.MenuItem{
		{ID: "a", Label: "图片", Action: mcdomain.Action{Type: "open_card", CardID: "c1"}},
	}}
	it, ok := FindEnabledItemByLabel(menu, "图片")
	if !ok || it.Action.CardID != "c1" {
		t.Fatalf("item=%+v ok=%v", it, ok)
	}
}

type captureOutbound struct {
	lists [][][]ports.Button
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
func (c *captureOutbound) SendMedia(context.Context, sharedkernel.ChannelAddr, sharedkernel.BlobRef, string) error {
	return nil
}
func (c *captureOutbound) SendMediaURL(context.Context, sharedkernel.ChannelAddr, string, string) error {
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
func (m *mediaURLOutbound) SendMedia(context.Context, sharedkernel.ChannelAddr, sharedkernel.BlobRef, string) error {
	return nil
}
func (m *mediaURLOutbound) SendMediaURL(_ context.Context, _ sharedkernel.ChannelAddr, imageURL, _ string) error {
	m.mediaURLs = append(m.mediaURLs, imageURL)
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
func (c *textCaptureOutbound) SendMedia(context.Context, sharedkernel.ChannelAddr, sharedkernel.BlobRef, string) error {
	return nil
}
func (c *textCaptureOutbound) SendMediaURL(context.Context, sharedkernel.ChannelAddr, string, string) error {
	return nil
}

type cardProviderStub struct {
	card mcdomain.Card
	err  error
}

func (p cardProviderStub) GetCard(_ context.Context, _, _ string) (mcdomain.Card, error) {
	return p.card, p.err
}

func TestActionDispatchOpenCardSendsCard(t *testing.T) {
	out := &captureOutbound{}
	ad := New(out)
	ad.ChannelID = "ch1"
	ad.Cards = cardProviderStub{card: mcdomain.Card{
		ID: "c1", Name: "x", Text: "选一种风格：",
		Buttons: []mcdomain.CardButton{
			{ID: "b", Label: "写实", Action: mcdomain.Action{Type: "open_workflow", WorkflowID: "w1"}},
		},
	}}
	addr := sharedkernel.ChannelAddr{ChannelID: "tg-default", ExternalChatID: "1"}
	if err := ad.actionDispatch(context.Background(), sharedkernel.ChatID("tg-default:1"), addr, mcdomain.Action{Type: "open_card", CardID: "c1"}, "root"); err != nil {
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
	if err := ad.actionDispatch(context.Background(), sharedkernel.ChatID("tg-default:1"), addr, mcdomain.Action{Type: "send_text", Text: "即将上线"}, "root"); err != nil {
		t.Fatal(err)
	}
	if len(out.texts) != 1 || out.texts[0] != "即将上线" {
		t.Fatalf("texts=%q", out.texts)
	}
}

func TestBackChainTracksSources(t *testing.T) {
	ad := New(&captureOutbound{})
	ad.ChannelID = "ch1"
	ad.Cards = cardProviderStub{card: mcdomain.Card{ID: "c2", Name: "x", Text: "第二张"}}
	addr := sharedkernel.ChannelAddr{ChannelID: "tg-default", ExternalChatID: "1"}
	chat := "tg-default:1"
	_ = ad.actionDispatch(context.Background(), sharedkernel.ChatID(chat), addr, mcdomain.Action{Type: "open_card", CardID: "c1"}, "root")
	_ = ad.actionDispatch(context.Background(), sharedkernel.ChatID(chat), addr, mcdomain.Action{Type: "open_card", CardID: "c2"}, "c1")
	if top, ok := ad.back.top(chat); !ok || top != "c1" {
		t.Fatalf("top=%q ok=%v", top, ok)
	}
}
