package application

import (
	"context"
	"testing"

	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
)

type fakeNotifyHandler struct {
	calls []sharedkernel.UserNotify
}

func (f *fakeNotifyHandler) HandleNotify(_ context.Context, n sharedkernel.UserNotify) error {
	f.calls = append(f.calls, n)
	return nil
}

func TestNotifyRouter_RoutesByChannel(t *testing.T) {
	tgHandler := &fakeNotifyHandler{}
	router := &NotifyRouter{
		HandlerByChannel: func(channelID string) (NotifyHandler, bool) {
			if channelID == "tg" {
				return tgHandler, true
			}
			return nil, false
		},
	}

	n := sharedkernel.UserNotify{
		ChatID:  "tg:123",
		TaskID:  "t1",
		Kind:    "task_succeeded",
		Outputs: []sharedkernel.BlobRef{{Key: "out.png"}},
	}
	if err := router.Publish(context.Background(), n); err != nil {
		t.Fatal(err)
	}
	if len(tgHandler.calls) != 1 || tgHandler.calls[0].TaskID != "t1" {
		t.Fatalf("calls=%+v", tgHandler.calls)
	}

	// 未知消息平台不崩溃
	n.ChatID = "feishu-1:999"
	if err := router.Publish(context.Background(), n); err != nil {
		t.Fatal(err)
	}
	if len(tgHandler.calls) != 1 {
		t.Fatalf("unknown channel must not dispatch: %+v", tgHandler.calls)
	}

	// 非法地址报错
	n.ChatID = "bad"
	if err := router.Publish(context.Background(), n); err == nil {
		t.Fatal("invalid chat id must error")
	}
}
