package conversation

import (
	"context"
	"testing"

	"github.com/Mr9esx/Pixoma/internal/channels/protocol"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
)

type recordingInvoker struct {
	last protocol.CapabilityInvoke
}

func (r *recordingInvoker) Invoke(_ context.Context, inv protocol.CapabilityInvoke) (protocol.Result, error) {
	r.last = inv
	return protocol.Result{}, nil
}

type recordingRenderer struct {
	text string
}

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
