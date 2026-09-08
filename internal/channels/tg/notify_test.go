package tg_test

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"

	texttpl "github.com/Mr9esx/Pixoma/internal/channels/domain/templates"
	"github.com/Mr9esx/Pixoma/internal/channels/tg"
	"github.com/Mr9esx/Pixoma/internal/channels/tg/tginternal"
	"github.com/Mr9esx/Pixoma/internal/platform/blob"
	"github.com/Mr9esx/Pixoma/internal/platform/blob/localfs"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
)

type sessionTextRenderer struct{}

func (sessionTextRenderer) Render(_ context.Context, _ string, key string, vars map[string]string) string {
	return texttpl.Render(key+"|custom", vars)
}

func TestHandleUserNotifySendsAllOutputs(t *testing.T) {
	ctx := context.Background()
	store, err := localfs.New(filepath.Join(t.TempDir(), "blob"))
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"image_0_a.png", "image_1_b.png"} {
		if _, err := store.Put(ctx, "outputs/t1/"+name, bytes.NewReader([]byte("png-bytes")), blob.PutOptions{MIME: "image/png"}); err != nil {
			t.Fatal(err)
		}
	}
	b, srv := tginternal.NewBot(t)
	messenger := &tg.BotMessenger{Bot: b, Blob: store}
	a := tg.New(messenger)
	a.ChannelID = "tg"
	a.Blob = store
	if err := a.HandleUserNotify(ctx, sharedkernel.UserNotify{
		ChatID: sharedkernel.ChatID("tg:123"),
		TaskID: "t1",
		Kind:   "task_succeeded",
		Outputs: []sharedkernel.BlobRef{
			{Key: "outputs/t1/image_0_a.png", MIME: "image/png"},
			{Key: "outputs/t1/image_1_b.png", MIME: "image/png"},
		},
	}); err != nil {
		t.Fatal(err)
	}
	if got := srv.SendPhotoCalls(); got != 2 {
		t.Fatalf("sendPhoto calls = %d, want 2", got)
	}
	names := srv.UploadNames()
	if len(names) != 2 || names[0] != "image_0_a.png" || names[1] != "image_1_b.png" {
		t.Fatalf("upload names = %v", names)
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
	b, srv := tginternal.NewBot(t)
	messenger := &tg.BotMessenger{Bot: b, Blob: store}
	a := tg.New(messenger)
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
	texts := srv.SendMessageTexts()
	if len(texts) < 1 || texts[0] != "hello from workflow" {
		t.Fatalf("sendMessage texts = %v, want first %q", texts, "hello from workflow")
	}
}

func TestHandleUserNotifySessionTerminated(t *testing.T) {
	b, srv := tginternal.NewBot(t)
	a := tg.New(&tg.BotMessenger{Bot: b})
	a.ChannelID = "tg"
	err := a.HandleUserNotify(context.Background(), sharedkernel.UserNotify{
		ChatID:   sharedkernel.ChatID("tg:123"),
		Kind:     "session_terminated",
		ErrorMsg: "该工作流已被管理员删除，当前会话已结束。",
	})
	if err != nil {
		t.Fatal(err)
	}
	texts := srv.SendMessageTexts()
	if len(texts) != 1 || texts[0] != "该工作流已被管理员删除，当前会话已结束。" {
		t.Fatalf("sendMessage texts = %v", texts)
	}
}

func TestHandleUserNotifySessionTerminatedRendersTemplate(t *testing.T) {
	b, srv := tginternal.NewBot(t)
	a := tg.New(&tg.BotMessenger{Bot: b})
	a.ChannelID = "tg-custom"
	a.Texts = sessionTextRenderer{}
	if err := a.HandleUserNotify(context.Background(), sharedkernel.UserNotify{
		ChatID:   sharedkernel.ChatID("tg-custom:123"),
		Kind:     "session_terminated",
		ErrorMsg: "case removed",
	}); err != nil {
		t.Fatal(err)
	}
	texts := srv.SendMessageTexts()
	if len(texts) != 1 || texts[0] != "session_terminated|custom" {
		t.Fatalf("sendMessage texts = %v", texts)
	}
}

func TestHandleUserNotifyFailedRendersTemplate(t *testing.T) {
	b, srv := tginternal.NewBot(t)
	a := tg.New(&tg.BotMessenger{Bot: b})
	a.ChannelID = "tg"
	if err := a.HandleUserNotify(context.Background(), sharedkernel.UserNotify{
		ChatID:   sharedkernel.ChatID("tg:123"),
		TaskID:   "T42",
		Kind:     "task_failed",
		ErrorMsg: "comfy timeout",
	}); err != nil {
		t.Fatal(err)
	}
	want := "❌ 任务执行失败\ntask=T42\n状态：failed\ncomfy timeout"
	texts := srv.SendMessageTexts()
	if len(texts) < 1 || texts[0] != want {
		t.Fatalf("sendMessage texts = %v, want first %q", texts, want)
	}
}

func TestHandleUserNotifySuccessWithoutOutputsSendsDone(t *testing.T) {
	b, srv := tginternal.NewBot(t)
	a := tg.New(&tg.BotMessenger{Bot: b})
	a.ChannelID = "tg"
	if err := a.HandleUserNotify(context.Background(), sharedkernel.UserNotify{
		ChatID: sharedkernel.ChatID("tg:123"),
		TaskID: "T7",
		Kind:   "task_succeeded",
	}); err != nil {
		t.Fatal(err)
	}
	want := "✅ 工作流完成\ntask=T7"
	texts := srv.SendMessageTexts()
	if len(texts) < 1 || texts[0] != want {
		t.Fatalf("sendMessage texts = %v, want first %q", texts, want)
	}
}
