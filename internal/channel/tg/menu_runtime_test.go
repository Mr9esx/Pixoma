package tg

import (
	"context"
	"testing"

	"github.com/mr9esx/comfyui_tgbot/internal/channel/ports"
	"github.com/mr9esx/comfyui_tgbot/internal/channel/protocol"
	"github.com/mr9esx/comfyui_tgbot/internal/menu/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

func TestBuildReplyKeyboardColumns(t *testing.T) {
	tree := domain.MenuTree{
		Items: []domain.MenuNode{
			{ID: "a", Label: "A", Order: 0, Enabled: true, RenderOverride: map[string]any{"columns": 3}},
			{ID: "b", Label: "B", Order: 1, Enabled: true},
			{ID: "c", Label: "C", Order: 2, Enabled: true},
		},
	}
	kb := BuildReplyKeyboard(tree)
	if len(kb.Keyboard) != 1 || len(kb.Keyboard[0]) != 3 {
		t.Fatalf("columns=3 expected one row of 3, got %+v", kb.Keyboard)
	}

	tree.Items[0].RenderOverride = nil
	kb2 := BuildReplyKeyboard(tree)
	if len(kb2.Keyboard) != 2 || len(kb2.Keyboard[0]) != 2 || len(kb2.Keyboard[1]) != 1 {
		t.Fatalf("default 2 columns expected 2+1, got %+v", kb2.Keyboard)
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
