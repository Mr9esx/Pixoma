package tg_test

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"

	"github.com/mr9esx/comfyui_tgbot/internal/channel/ports"
	texttpl "github.com/mr9esx/comfyui_tgbot/internal/channel/text"
	"github.com/mr9esx/comfyui_tgbot/internal/channel/tg"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/blob"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/blob/localfs"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

type recordingOutbound struct {
	media []sharedkernel.BlobRef
	texts []string
}

type sessionTextRenderer struct{}

func (sessionTextRenderer) Render(_ context.Context, _ string, key string, vars map[string]string) string {
	return texttpl.Render(key+"|custom", vars)
}

func (r *recordingOutbound) SendText(_ context.Context, _ sharedkernel.ChannelAddr, text string) error {
	r.texts = append(r.texts, text)
	return nil
}

func (r *recordingOutbound) SendMenu(_ context.Context, _ sharedkernel.ChannelAddr, _ string, _ []ports.MenuEntry) error {
	return nil
}

func (r *recordingOutbound) SendList(_ context.Context, _ sharedkernel.ChannelAddr, _ string, _ [][]ports.Button) error {
	return nil
}

func (r *recordingOutbound) SendMedia(_ context.Context, _ sharedkernel.ChannelAddr, ref sharedkernel.BlobRef, _ string, _ [][]ports.Button) error {
	r.media = append(r.media, ref)
	return nil
}

func (r *recordingOutbound) SendMediaURL(_ context.Context, _ sharedkernel.ChannelAddr, _, _ string, _ [][]ports.Button) error {
	return nil
}

func (r *recordingOutbound) EditReplyMarkup(context.Context, sharedkernel.ChannelAddr, int, [][]ports.Button) error {
	return nil
}

func TestHandleUserNotifySendsAllOutputs(t *testing.T) {
	out := &recordingOutbound{}
	a := tg.New(out)
	a.ChannelID = "tg"
	err := a.HandleUserNotify(context.Background(), sharedkernel.UserNotify{
		ChatID: sharedkernel.ChatID("tg:123"),
		TaskID: "t1",
		Kind:   "task_succeeded",
		Outputs: []sharedkernel.BlobRef{
			{Key: "outputs/t1/image_0_a.png", MIME: "image/png"},
			{Key: "outputs/t1/image_1_b.png", MIME: "image/png"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.media) != 2 {
		t.Fatalf("media=%d", len(out.media))
	}
	if out.media[0].Key != "outputs/t1/image_0_a.png" || out.media[1].Key != "outputs/t1/image_1_b.png" {
		t.Fatalf("media keys=%q %q", out.media[0].Key, out.media[1].Key)
	}
}

func TestHandleUserNotifySendsTextOutput(t *testing.T) {
	ctx := context.Background()
	store, err := localfs.New(filepath.Join(t.TempDir(), "blob"))
	if err != nil {
		t.Fatal(err)
	}
	ref, err := store.Put(ctx, "outputs/t1/note.txt", bytes.NewReader([]byte("hello from workflow")), blob.PutOptions{MIME: "text/plain"})
	if err != nil {
		t.Fatal(err)
	}
	out := &recordingOutbound{}
	a := tg.New(out)
	a.ChannelID = "tg"
	a.Blob = store
	if err := a.HandleUserNotify(ctx, sharedkernel.UserNotify{
		ChatID:  sharedkernel.ChatID("tg:123"),
		TaskID:  "t1",
		Kind:    "task_succeeded",
		Outputs: []sharedkernel.BlobRef{ref},
	}); err != nil {
		t.Fatal(err)
	}
	if len(out.texts) != 1 || out.texts[0] != "hello from workflow" {
		t.Fatalf("texts=%q", out.texts)
	}
}

func TestHandleUserNotifySessionTerminated(t *testing.T) {
	out := &recordingOutbound{}
	a := tg.New(out)
	a.ChannelID = "tg"
	err := a.HandleUserNotify(context.Background(), sharedkernel.UserNotify{
		ChatID:   sharedkernel.ChatID("tg:123"),
		Kind:     "session_terminated",
		ErrorMsg: "该工作流已被管理员删除，当前会话已结束。",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.texts) != 1 || out.texts[0] != "该工作流已被管理员删除，当前会话已结束。" {
		t.Fatalf("texts=%+v", out.texts)
	}
}

func TestHandleUserNotifySessionTerminatedRendersTemplate(t *testing.T) {
	out := &recordingOutbound{}
	a := tg.New(out)
	a.ChannelID = "tg-custom"
	a.Texts = sessionTextRenderer{}
	if err := a.HandleUserNotify(context.Background(), sharedkernel.UserNotify{
		ChatID:   sharedkernel.ChatID("tg-custom:123"),
		Kind:     "session_terminated",
		ErrorMsg: "case removed",
	}); err != nil {
		t.Fatal(err)
	}
	if len(out.texts) != 1 || out.texts[0] != "session_terminated|custom" {
		t.Fatalf("texts=%+v", out.texts)
	}
}

func TestHandleUserNotifyFailedRendersTemplate(t *testing.T) {
	out := &recordingOutbound{}
	a := tg.New(out)
	a.ChannelID = "tg"
	if err := a.HandleUserNotify(context.Background(), sharedkernel.UserNotify{
		ChatID:   sharedkernel.ChatID("tg:123"),
		TaskID:   "T42",
		Kind:     "task_failed",
		ErrorMsg: "comfy timeout",
	}); err != nil {
		t.Fatal(err)
	}
	if len(out.texts) != 1 {
		t.Fatalf("texts=%+v", out.texts)
	}
	want := "❌ 任务执行失败\ntask=T42\n状态：failed\ncomfy timeout"
	if out.texts[0] != want {
		t.Fatalf("got %q want %q", out.texts[0], want)
	}
}

func TestHandleUserNotifySuccessWithoutOutputsSendsDone(t *testing.T) {
	out := &recordingOutbound{}
	a := tg.New(out)
	a.ChannelID = "tg"
	if err := a.HandleUserNotify(context.Background(), sharedkernel.UserNotify{
		ChatID: sharedkernel.ChatID("tg:123"),
		TaskID: "T7",
		Kind:   "task_succeeded",
	}); err != nil {
		t.Fatal(err)
	}
	if len(out.texts) != 1 {
		t.Fatalf("texts=%+v", out.texts)
	}
	want := "✅ 工作流完成\ntask=T7"
	if out.texts[0] != want {
		t.Fatalf("got %q want %q", out.texts[0], want)
	}
}
