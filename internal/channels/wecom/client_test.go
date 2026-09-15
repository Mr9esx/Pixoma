package wecom

import "testing"

func TestMapSDKMessagePreservesGroupAddressAndRequestID(t *testing.T) {
	got := mapSDKMessage(sdkMessage{
		ReqID: "req-1", ChatType: sdkGroupChat, ChatID: "room-1", FromUserID: "user-1", Text: "生成图片",
	})
	if got.ReqID != "req-1" || got.ChatType != Group || got.ChatID != "room-1" || got.UserID != "user-1" {
		t.Fatalf("message = %#v", got)
	}
}
