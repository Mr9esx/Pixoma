package feishu

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
	"time"

	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"

	channelapp "github.com/Mr9esx/Pixoma/internal/channels/application"
	"github.com/Mr9esx/Pixoma/internal/channels/application/capability"
	"github.com/Mr9esx/Pixoma/internal/channels/protocol"
	"github.com/Mr9esx/Pixoma/internal/platform/blob"
	convdomain "github.com/Mr9esx/Pixoma/internal/sessions/domain"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
	identitydomain "github.com/Mr9esx/Pixoma/internal/users/domain"
)

// LongConn is the long-lived Feishu WebSocket connection. It blocks in Start
// and returns once the connection is closed. The assembler guarantees the
// previous connection is stopped before a new one is started (single instance).
type LongConn interface {
	Start(ctx context.Context) error
}

// FeishuAdapter translates Feishu events into capability invokes and renders
// results back to the originating chat. Identity uses the sender's user id
// while delivery uses the chat id, keeping group and private chats separate.
type FeishuAdapter struct {
	ChannelID string
	Registry  *capability.Registry
	Users     protocol.IdentityResolver
	Blob      blob.Store
	IM        IMClient
	// NewLongConn builds the WebSocket connection and attaches the inbound
	// event callback. Testable by overriding.
	NewLongConn func(appID, appSecret string, onMsg func(context.Context, *larkim.P2MessageReceiveV1)) (LongConn, error)

	appID, appSecret string
	mu               sync.Mutex
	started          bool
	cancel           context.CancelFunc
	done             chan struct{}
}

// NewAdapter wires a Feishu adapter for a channel snapshot.
func NewAdapter(snap channelapp.ChannelSnapshot, im IMClient, deps AdapterDeps) *FeishuAdapter {
	return &FeishuAdapter{
		ChannelID:   snap.ID,
		Registry:    deps.Capabilities,
		Users:       deps.Users,
		Blob:        deps.Blob,
		IM:          im,
		appID:       snap.AppID,
		appSecret:   snap.AppSecret,
		NewLongConn: defaultLongConn,
	}
}

// AdapterDeps carries the shared runtime pieces a Feishu adapter needs.
type AdapterDeps struct {
	Capabilities *capability.Registry
	Users        protocol.IdentityResolver
	Blob         blob.Store
}

func (a *FeishuAdapter) imOrNil() (IMClient, bool) {
	return a.IM, a.IM != nil
}

func (a *FeishuAdapter) Start(ctx context.Context) error {
	a.mu.Lock()
	if a.started {
		a.mu.Unlock()
		return nil
	}
	if a.appID == "" || a.appSecret == "" {
		a.mu.Unlock()
		return fmt.Errorf("feishu: app_id/app_secret required")
	}
	newConn := a.NewLongConn
	if newConn == nil {
		newConn = defaultLongConn
	}
	conn, err := newConn(a.appID, a.appSecret, a.handleReceive)
	if err != nil {
		a.mu.Unlock()
		return err
	}
	rctx, cancel := context.WithCancel(ctx)
	a.cancel = cancel
	a.done = make(chan struct{})
	a.started = true
	a.mu.Unlock()

	go func() {
		defer close(a.done)
		if err := conn.Start(rctx); err != nil && rctx.Err() == nil {
			slog.Error("feishu long conn stopped", "channel", a.ChannelID, "err", err)
		}
	}()
	slog.Info("feishu channel started", "channel", a.ChannelID)
	return nil
}

func (a *FeishuAdapter) Stop(ctx context.Context) error {
	a.mu.Lock()
	cancel := a.cancel
	done := a.done
	a.cancel = nil
	a.done = nil
	a.started = false
	a.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if done == nil {
		return nil
	}
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// handleReceive acknowledges the event immediately and processes it on a
// background goroutine so Comfy/task work never blocks the platform ack.
func (a *FeishuAdapter) handleReceive(ctx context.Context, e *larkim.P2MessageReceiveV1) {
	in := parseInbound(e)
	if in == nil {
		return
	}
	go a.dispatchInbound(context.WithoutCancel(ctx), in)
}

func (a *FeishuAdapter) dispatchInbound(ctx context.Context, in *inbound) {
	inv := a.baseInvoke(ctx, in)
	switch {
	case in.ImageKey != "":
		data, err := a.IM.DownloadImage(ctx, in.MessageID, in.ImageKey)
		if err != nil {
			slog.Error("feishu download image", "channel", a.ChannelID, "err", err)
			_ = a.sendText(ctx, in.ChatID, "下载图片失败")
			return
		}
		key, putErr := a.putBlob(ctx, in, data)
		if putErr != nil {
			_ = a.sendText(ctx, in.ChatID, "保存图片失败: "+putErr.Error())
			return
		}
		inv.CapabilityID = "open_case"
		inv.Params = map[string]any{"step": "media", "blob": map[string]any{"key": key, "mime": in.MIME}}
	case in.Text != "":
		inv.CapabilityID = "open_case"
		inv.Params = map[string]any{"step": "text", "text": in.Text}
	default:
		return
	}
	a.dispatch(ctx, in.ChatID, inv)
}

func (a *FeishuAdapter) baseInvoke(ctx context.Context, in *inbound) protocol.CapabilityInvoke {
	chatAddr := sharedkernel.ChannelAddr{ChannelID: a.ChannelID, ExternalChatID: in.ChatID}
	inv := protocol.CapabilityInvoke{
		Account: protocol.AccountCtx{ChannelID: a.ChannelID, ExternalUserID: in.SenderUserID},
		ChatID:  sharedkernel.FormatChatID(chatAddr),
		Nav:     protocol.Nav{Back: "root"},
	}
	if a.Users != nil {
		id, err := a.Users.Resolve(ctx, sharedkernel.ChannelAddr{ChannelID: a.ChannelID, ExternalChatID: in.SenderUserID}, identitydomain.UpsertFrom{
			ChannelID: a.ChannelID, ExternalUserID: in.SenderUserID,
		})
		if err != nil {
			slog.Warn("feishu resolve account", "channel", a.ChannelID, "err", err)
		} else {
			inv.Account.InternalUserID = id
		}
	}
	return inv
}

func (a *FeishuAdapter) dispatch(ctx context.Context, chatID string, inv protocol.CapabilityInvoke) {
	if a.Registry == nil {
		_ = a.sendText(ctx, chatID, "功能暂不可用")
		return
	}
	res, err := a.Registry.Invoke(ctx, inv)
	if err != nil {
		if errors.Is(err, convdomain.ErrNoActiveSession) ||
			errors.Is(err, convdomain.ErrInvalidState) ||
			errors.Is(err, convdomain.ErrInputOutOfOrder) {
			_ = a.sendText(ctx, chatID, "暂无进行中的会话，请先发起一个会话。")
			return
		}
		slog.Error("feishu capability invoke", "channel", a.ChannelID, "err", err, "capability", inv.CapabilityID)
		_ = a.sendText(ctx, chatID, "操作失败，稍后重试。")
		return
	}
	a.renderResult(ctx, chatID, res)
}

func (a *FeishuAdapter) renderResult(ctx context.Context, chatID string, res protocol.Result) {
	if res.Error != nil && *res.Error != "" {
		_ = a.sendText(ctx, chatID, *res.Error)
		return
	}
	if len(res.Media) > 0 {
		if res.Text != "" {
			_ = a.sendText(ctx, chatID, res.Text)
		}
		for _, m := range res.Media {
			data, err := a.readBlob(ctx, m.Key)
			if err != nil {
				slog.Warn("feishu result blob", "key", m.Key, "err", err)
				continue
			}
			if err := a.sendImage(ctx, chatID, data, m.MIME); err != nil {
				_ = a.sendText(ctx, chatID, "发送图片失败")
			}
		}
		return
	}
	for _, u := range res.MediaURLs {
		if err := a.sendText(ctx, chatID, u); err != nil {
			return
		}
	}
	if len(res.Options) > 0 {
		var b strings.Builder
		if res.Text != "" {
			b.WriteString(res.Text)
			b.WriteString("\n")
		}
		for i, opt := range res.Options {
			b.WriteString(fmt.Sprintf("%d. %s\n", i+1, opt.Label))
		}
		_ = a.sendText(ctx, chatID, strings.TrimRight(b.String(), "\n"))
		return
	}
	if res.Text != "" {
		_ = a.sendText(ctx, chatID, res.Text)
	}
}

// HandleNotify satisfies the channel runtime NotifyHandler: task completion
// and lifecycle notifications are rendered back to the originating chat.
func (a *FeishuAdapter) HandleNotify(ctx context.Context, n sharedkernel.UserNotify) error {
	addr, err := sharedkernel.ParseChatID(string(n.ChatID))
	if err != nil {
		return err
	}
	chatID := addr.ExternalChatID
	switch n.Kind {
	case "task_succeeded":
		for _, ref := range n.Outputs {
			if strings.HasPrefix(ref.MIME, "text/") {
				raw, rerr := a.readBlob(ctx, ref.Key)
				if rerr != nil {
					continue
				}
				_ = a.sendText(ctx, chatID, string(raw))
				continue
			}
			data, derr := a.readBlob(ctx, ref.Key)
			if derr != nil {
				continue
			}
			if err := a.sendImage(ctx, chatID, data, ref.MIME); err != nil {
				slog.Warn("feishu notify image send", "err", err)
			}
		}
	case "task_failed", "task_cancelled":
		msg := "任务 " + string(n.TaskID) + " " + n.Kind
		if n.ErrorMsg != "" {
			msg += " — " + n.ErrorMsg
		}
		_ = a.sendText(ctx, chatID, msg)
	case "session_terminated":
		msg := "该消息平台已被管理员删除，当前会话已结束。"
		if n.ErrorMsg != "" {
			msg = n.ErrorMsg
		}
		_ = a.sendText(ctx, chatID, msg)
	default:
		_ = a.sendText(ctx, chatID, "任务 "+string(n.TaskID)+": "+n.Kind)
	}
	return nil
}

func (a *FeishuAdapter) sendText(ctx context.Context, chatID, text string) error {
	im, ok := a.imOrNil()
	if !ok {
		return errors.New("feishu: im client not configured")
	}
	return im.SendText(ctx, chatID, text)
}

func (a *FeishuAdapter) sendImage(ctx context.Context, chatID string, data []byte, mime string) error {
	im, ok := a.imOrNil()
	if !ok {
		return errors.New("feishu: im client not configured")
	}
	return im.SendImage(ctx, chatID, Image{Data: data, MIME: mime})
}

func (a *FeishuAdapter) putBlob(ctx context.Context, in *inbound, data []byte) (string, error) {
	if a.Blob == nil {
		return "", errors.New("feishu: blob store not configured")
	}
	key := fmt.Sprintf("feishu/%s/%d%s", in.ChatID, time.Now().UnixNano(), extForMIME(in.MIME))
	ref, err := a.Blob.Put(ctx, key, bytes.NewReader(data), blob.PutOptions{MIME: in.MIME})
	if err != nil {
		return "", err
	}
	return ref.Key, nil
}

func (a *FeishuAdapter) readBlob(ctx context.Context, key string) ([]byte, error) {
	if a.Blob == nil {
		return nil, errors.New("feishu: blob store not configured")
	}
	rc, err := a.Blob.Get(ctx, sharedkernel.BlobRef{Key: key})
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

func extForMIME(mime string) string {
	switch strings.ToLower(mime) {
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	default:
		return ".jpg"
	}
}
