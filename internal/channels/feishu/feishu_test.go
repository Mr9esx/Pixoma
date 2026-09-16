package feishu

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"runtime"
	"testing"

	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"

	"github.com/Mr9esx/Pixoma/internal/channels/protocol"
	"github.com/Mr9esx/Pixoma/internal/platform/blob"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
	identitydomain "github.com/Mr9esx/Pixoma/internal/users/domain"
)

func strp(s string) *string { return &s }

func textEvent(text, chatType string) *larkim.P2MessageReceiveV1 {
	return &larkim.P2MessageReceiveV1{
		Event: &larkim.P2MessageReceiveV1Data{
			Sender: &larkim.EventSender{SenderId: &larkim.UserId{UserId: strp("ou_user_1")}},
			Message: &larkim.EventMessage{
				MessageId:   strp("om_1"),
				ChatId:      strp("oc_chat_1"),
				ChatType:    strp(chatType),
				MessageType: strp("text"),
				Content:     strp(fmt.Sprintf(`{"text":%q}`, text)),
			},
		},
	}
}

func groupAtEvent(text string) *larkim.P2MessageReceiveV1 {
	e := textEvent(text, "group")
	e.Event.Message.Mentions = []*larkim.MentionEvent{{Key: strp("mention"), Name: strp("Pixoma")}}
	return e
}

func TestParseInboundPrivateText(t *testing.T) {
	in := parseInbound(textEvent("hello", "p2p"))
	if in == nil {
		t.Fatal("nil inbound")
	}
	if in.Text != "hello" || in.ChatID != "oc_chat_1" || in.SenderUserID != "ou_user_1" {
		t.Fatalf("unexpected inbound: %+v", in)
	}
	if !in.IsGroup {
		t.Log("p2p is not group")
	}
}

func TestParseInboundGroupRequiresMention(t *testing.T) {
	if parseInbound(textEvent("hello", "group")) != nil {
		t.Fatal("group message without mention must be ignored")
	}
	in := parseInbound(groupAtEvent("render"))
	if in == nil || !in.IsGroup || !in.IsAtBot || in.Text != "render" {
		t.Fatalf("unexpected: %+v", in)
	}
}

func TestParseInboundImage(t *testing.T) {
	e := textEvent("", "p2p")
	e.Event.Message.MessageType = strp("image")
	e.Event.Message.Content = strp(`{"image_key":"img_v2_abc"}`)
	in := parseInbound(e)
	if in == nil {
		t.Fatal("nil inbound")
	}
	if in.ImageKey != "img_v2_abc" || in.MIME != "image/jpeg" {
		t.Fatalf("unexpected image inbound: %+v", in)
	}
}

// identityientResolver stub
type stubIdentity struct{}

func (stubIdentity) Resolve(ctx context.Context, addr sharedkernel.ChannelAddr, p identitydomain.UpsertFrom) (string, error) {
	return "internal-" + p.ExternalUserID, nil
}

type fakeIM struct {
	sentText  []string
	sentImage [][]Image
	download  []byte
}

func (f *fakeIM) SendText(_ context.Context, _ string, text string) error {
	f.sentText = append(f.sentText, text)
	return nil
}
func (f *fakeIM) SendImage(_ context.Context, _ string, img Image) error {
	f.sentImage = append(f.sentImage, []Image{img})
	return nil
}
func (f *fakeIM) DownloadImage(_ context.Context, _, _ string) ([]byte, error) {
	if f.download == nil {
		return nil, errors.New("no download")
	}
	return f.download, nil
}

type stubBlob struct{ mem map[string][]byte }

func (s *stubBlob) Put(_ context.Context, key string, r io.Reader, opts blob.PutOptions) (sharedkernel.BlobRef, error) {
	data, _ := io.ReadAll(r)
	s.mem[key] = data
	return sharedkernel.BlobRef{Key: key, MIME: opts.MIME}, nil
}
func (s *stubBlob) Get(_ context.Context, ref sharedkernel.BlobRef) (io.ReadCloser, error) {
	data, ok := s.mem[ref.Key]
	if !ok {
		return nil, errors.New("not found")
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}
func (s *stubBlob) Check(context.Context) error { return nil }

func TestBaseInvokeSeparatesUserAndChat(t *testing.T) {
	a := &FeishuAdapter{ChannelID: "ch-fs", Users: stubIdentity{}, appID: "cli_1", appSecret: "s"}
	in := &inbound{ChatID: "oc_chat_9", SenderUserID: "ou_user_2"}
	inv := a.baseInvoke(context.Background(), in)
	if inv.Account.ExternalUserID != "ou_user_2" {
		t.Fatalf("external user = %q", inv.Account.ExternalUserID)
	}
	if inv.Account.InternalUserID != "internal-ou_user_2" {
		t.Fatalf("internal user = %q", inv.Account.InternalUserID)
	}
	if inv.ChatID != "ch-fs:oc_chat_9" {
		t.Fatalf("delivery chat = %q", inv.ChatID)
	}
}

func TestRenderResultSendsMedia(t *testing.T) {
	im := &fakeIM{}
	store := &stubBlob{mem: map[string][]byte{"fs/x.jpg": []byte("IMG")}}
	a := &FeishuAdapter{ChannelID: "ch-fs", IM: im, Blob: store}
	res := protocol.Result{Text: "完成", Media: []protocol.MediaRef{{Key: "fs/x.jpg", MIME: "image/jpeg"}}}
	a.renderResult(context.Background(), "oc_chat_1", res)
	if len(im.sentText) != 1 || im.sentText[0] != "完成" {
		t.Fatalf("caption not sent: %+v", im.sentText)
	}
	if len(im.sentImage) != 1 || !bytes.Equal(im.sentImage[0][0].Data, []byte("IMG")) {
		t.Fatalf("image not sent: %+v", im.sentImage)
	}
}

func TestStartRejectsMissingCredentials(t *testing.T) {
	a := &FeishuAdapter{ChannelID: "ch-fs"}
	if err := a.Start(context.Background()); err == nil {
		t.Fatal("expected error for missing credentials")
	}
}

func TestStartStopLifecycle(t *testing.T) {
	a := &FeishuAdapter{
		ChannelID: "ch-fs",
		appID:     "cli_1",
		appSecret: "s",
		NewLongConn: func(_, _ string, _ func(context.Context, *larkim.P2MessageReceiveV1)) (LongConn, error) {
			return stubLongConn{}, nil
		},
	}
	if err := a.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := a.Stop(context.Background()); err != nil {
		t.Fatalf("stop: %v", err)
	}
}

func TestStopClosesTheRunChannelCapturedAtStart(t *testing.T) {
	previous := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(previous)
	for i := 0; i < 100; i++ {
		a := &FeishuAdapter{
			ChannelID: "ch-fs",
			appID:     "cli_1",
			appSecret: "s",
			NewLongConn: func(_, _ string, _ func(context.Context, *larkim.P2MessageReceiveV1)) (LongConn, error) {
				return stubLongConn{}, nil
			},
		}
		if err := a.Start(context.Background()); err != nil {
			t.Fatalf("start: %v", err)
		}
		if err := a.Stop(context.Background()); err != nil {
			t.Fatalf("stop: %v", err)
		}
	}
}

type stubLongConn struct{}

func (stubLongConn) Start(ctx context.Context) error {
	<-ctx.Done()
	return nil
}
