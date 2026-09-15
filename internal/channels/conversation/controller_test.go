package conversation

import (
	"context"
	"testing"

	"github.com/Mr9esx/Pixoma/internal/channels/protocol"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
)

type recordingInvoker struct {
	last protocol.CapabilityInvoke
	err  error
}

func (r *recordingInvoker) Invoke(_ context.Context, inv protocol.CapabilityInvoke) (protocol.Result, error) {
	r.last = inv
	return protocol.Result{}, r.err
}

type recordingRenderer struct {
	text string
}

type recordingOutbound struct {
	listTitle string
	listRows  [][]protocol.Button
	texts     []string
	menus     int
}

func (r *recordingOutbound) SendText(_ context.Context, _ sharedkernel.ChannelAddr, text string) error {
	r.texts = append(r.texts, text)
	return nil
}
func (r *recordingOutbound) SendMenu(context.Context, sharedkernel.ChannelAddr, string, []protocol.MenuEntry) error {
	r.menus++
	return nil
}
func (r *recordingOutbound) SendList(_ context.Context, _ sharedkernel.ChannelAddr, title string, rows [][]protocol.Button) error {
	r.listTitle, r.listRows = title, rows
	return nil
}
func (r *recordingOutbound) SendMedia(context.Context, sharedkernel.ChannelAddr, sharedkernel.BlobRef, string, [][]protocol.Button) error {
	return nil
}
func (r *recordingOutbound) SendMediaURL(context.Context, sharedkernel.ChannelAddr, string, string, string, [][]protocol.Button) error {
	return nil
}
func (r *recordingOutbound) EditReplyMarkup(context.Context, sharedkernel.ChannelAddr, int, [][]protocol.Button) error {
	return nil
}

type testCallbacks struct{}

func (testCallbacks) Invoke(token string) string { return "invoke:" + token }
func (testCallbacks) MainMenu() string           { return "menu" }
func (testCallbacks) Back(target string) string  { return "back:" + target }

func (r *recordingRenderer) SendText(_ context.Context, _ sharedkernel.ChannelAddr, text string) error {
	r.text = text
	return nil
}

func TestController_SubmitsImageOnlyForActiveImageStep(t *testing.T) {
	invoker := &recordingInvoker{}
	ctl := New("tg-1", invoker, &recordingRenderer{})
	err := ctl.HandleMedia(context.Background(), Inbound{
		Addr:           sharedkernel.ChannelAddr{ChannelID: "tg-1", ExternalChatID: "123"},
		ExternalUserID: "123",
	}, sharedkernel.BlobRef{Key: "tg/123/a.png", MIME: "image/png"})
	if err != nil {
		t.Fatal(err)
	}
	if invoker.last.CapabilityID != "open_case" {
		t.Fatalf("capability = %q", invoker.last.CapabilityID)
	}
	if invoker.last.Params["step"] != "media" {
		t.Fatalf("step = %#v", invoker.last.Params["step"])
	}
	blob, ok := invoker.last.Params["blob"].(map[string]any)
	if !ok || blob["key"] != "tg/123/a.png" || blob["mime"] != "image/png" {
		t.Fatalf("blob = %#v", invoker.last.Params["blob"])
	}
}

func TestController_SubmitsTextForActiveWorkflow(t *testing.T) {
	invoker := &recordingInvoker{}
	ctl := New("tg-1", invoker, &recordingRenderer{})
	err := ctl.HandleText(context.Background(), Inbound{
		Addr:           sharedkernel.ChannelAddr{ChannelID: "tg-1", ExternalChatID: "123"},
		ExternalUserID: "123",
	}, "一只猫")
	if err != nil {
		t.Fatal(err)
	}
	if invoker.last.CapabilityID != "open_case" || invoker.last.Params["step"] != "text" || invoker.last.Params["text"] != "一只猫" {
		t.Fatalf("invoke = %#v", invoker.last)
	}
}

func TestController_ExpiredActionReturnsReselectEffect(t *testing.T) {
	renderer := &recordingRenderer{}
	ctl := New("tg-1", &recordingInvoker{}, renderer)
	err := ctl.HandleAction(context.Background(), Inbound{
		Addr:           sharedkernel.ChannelAddr{ChannelID: "tg-1", ExternalChatID: "123"},
		ExternalUserID: "123",
	}, "missing")
	if err != nil {
		t.Fatal(err)
	}
	if renderer.text != "操作已过期，重新选择。" {
		t.Fatalf("text = %q", renderer.text)
	}
}

func TestController_RetainsActionWhenDispatchFails(t *testing.T) {
	invoker := &recordingInvoker{err: context.DeadlineExceeded}
	renderer := &recordingRenderer{}
	ctl := New("tg-1", invoker, renderer)
	token := ctl.RememberAction(protocol.CapabilityInvoke{CapabilityID: "open_case"})
	in := Inbound{Addr: sharedkernel.ChannelAddr{ChannelID: "tg-1", ExternalChatID: "123"}, ExternalUserID: "123"}
	if err := ctl.HandleAction(context.Background(), in, token); err != context.DeadlineExceeded {
		t.Fatalf("first invoke error = %v", err)
	}
	invoker.err = nil
	if err := ctl.HandleAction(context.Background(), in, token); err != nil {
		t.Fatalf("retry error = %v", err)
	}
	if renderer.text != "" {
		t.Fatalf("unexpected expired-action text: %q", renderer.text)
	}
}

func TestController_RendersOptionsWithOneShotActionsAndBackNavigation(t *testing.T) {
	out := &recordingOutbound{}
	ctl := New("tg-1", &recordingInvoker{}, &recordingRenderer{})
	ctl.Outbound = out
	ctl.Callbacks = testCallbacks{}
	err := ctl.RenderResult(context.Background(), Inbound{
		Addr:           sharedkernel.ChannelAddr{ChannelID: "tg-1", ExternalChatID: "123"},
		ExternalUserID: "123",
	}, protocol.CapabilityInvoke{CapabilityID: "open_case", Nav: protocol.Nav{Back: "root"}}, protocol.Result{
		Text:    "请选择",
		Options: []protocol.Option{{Label: "开始", Value: map[string]any{"step": "start"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.listTitle != "请选择" || len(out.listRows) != 2 {
		t.Fatalf("list = %#v %#v", out.listTitle, out.listRows)
	}
	if out.listRows[0][0].Text != "开始" || out.listRows[0][0].Data != "invoke:1" {
		t.Fatalf("option = %#v", out.listRows[0])
	}
	if out.listRows[1][0].Text != "返回主菜单" || out.listRows[1][0].Data != "menu" {
		t.Fatalf("back = %#v", out.listRows[1])
	}
}

func TestController_NotifiesTaskSuccessOnce(t *testing.T) {
	out := &recordingOutbound{}
	ctl := New("tg-1", &recordingInvoker{}, &recordingRenderer{})
	ctl.Outbound = out
	if err := ctl.HandleNotify(context.Background(), sharedkernel.UserNotify{
		ChatID: sharedkernel.ChatID("tg-1:123"), TaskID: "task-1", Kind: "task_succeeded",
	}); err != nil {
		t.Fatal(err)
	}
	if err := ctl.HandleNotify(context.Background(), sharedkernel.UserNotify{
		ChatID: sharedkernel.ChatID("tg-1:123"), TaskID: "task-1", Kind: "task_succeeded",
	}); err != nil {
		t.Fatal(err)
	}
	if len(out.texts) != 1 || out.menus != 1 {
		t.Fatalf("texts=%v menus=%d", out.texts, out.menus)
	}
}
