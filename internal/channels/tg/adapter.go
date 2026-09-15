package tg

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/Mr9esx/Pixoma/internal/channels/application/capability"
	"github.com/Mr9esx/Pixoma/internal/channels/conversation"
	texttpl "github.com/Mr9esx/Pixoma/internal/channels/domain/templates"
	protocol "github.com/Mr9esx/Pixoma/internal/channels/protocol"
	mcdomain "github.com/Mr9esx/Pixoma/internal/menus/domain"
	"github.com/Mr9esx/Pixoma/internal/platform/blob"
	convdomain "github.com/Mr9esx/Pixoma/internal/sessions/domain"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
	identitydomain "github.com/Mr9esx/Pixoma/internal/users/domain"
)

// Adapter translates Telegram events into capability invokes and renders results.
type Adapter struct {
	Out       protocol.Outbound
	Media     protocol.MediaBridge
	Users     protocol.IdentityResolver
	Menu      MenuReader
	Registry  *capability.Registry
	Blob      blob.Store
	ChannelID string
	// Texts resolves configurable copy templates; nil falls back to built-ins.
	Texts         texttpl.Renderer
	back          *conversation.BackStack
	notifications *conversation.NotificationStore
	store         *conversation.ActionStore
}

func New(out protocol.Outbound) *Adapter {
	return &Adapter{
		Out:           out,
		Texts:         texttpl.StaticRenderer{},
		notifications: conversation.NewNotificationStore(),
		store:         conversation.NewActionStore(),
		back:          conversation.NewBackStack(),
	}
}

func addrOf(chatID sharedkernel.ChatID) (sharedkernel.ChannelAddr, error) {
	return sharedkernel.ParseChatID(string(chatID))
}

func (a *Adapter) account(ctx context.Context, chatID sharedkernel.ChatID) (protocol.AccountCtx, error) {
	addr, err := addrOf(chatID)
	if err != nil {
		return protocol.AccountCtx{}, err
	}
	acct := protocol.AccountCtx{ChannelID: a.ChannelID, ExternalUserID: addr.ExternalChatID}
	if a.Users != nil {
		id, err := a.Users.Resolve(ctx, addr, identitydomain.UpsertFrom{
			ChannelID: a.ChannelID, ExternalUserID: addr.ExternalChatID,
		})
		if err != nil {
			slog.Error("tg resolve account", "err", err, "chat_id", chatID)
		} else {
			acct.InternalUserID = id
		}
	}
	return acct, nil
}

func (a *Adapter) baseInvoke(ctx context.Context, chatID sharedkernel.ChatID) (protocol.CapabilityInvoke, error) {
	acct, err := a.account(ctx, chatID)
	if err != nil {
		return protocol.CapabilityInvoke{}, err
	}
	return protocol.CapabilityInvoke{Account: acct, ChatID: string(chatID), Nav: protocol.Nav{Back: "root"}}, nil
}

// HandleUserMedia downloads a photo and submits it to the open_case capability.
func (a *Adapter) HandleUserMedia(ctx context.Context, chatID sharedkernel.ChatID, fileID, mime string) error {
	addr0, err := addrOf(chatID)
	if err != nil {
		return err
	}
	if active, _ := a.appSessionExists(ctx, chatID); !active {
		return a.sendMainMenu(ctx, addr0)
	}
	if a.Media == nil || a.Blob == nil {
		return a.Out.SendText(ctx, mustAddr(chatID), "无法处理图片：未配置媒体桥")
	}
	data, err := a.Media.Download(ctx, fileID, mime)
	if err != nil {
		slog.Error("tg download user media", "err", err, "chat_id", chatID)
		return a.Out.SendText(ctx, mustAddr(chatID), "下载图片失败")
	}
	if mime == "" {
		mime = "image/jpeg"
	}
	addr, _ := addrOf(chatID)
	key := fmt.Sprintf("tg/%s/%d%s", addr.ExternalChatID, time.Now().UnixNano(), extForMIME(mime))
	ref, err := a.Blob.Put(ctx, key, bytes.NewReader(data), blob.PutOptions{MIME: mime})
	if err != nil {
		return a.Out.SendText(ctx, addr, "保存图片失败: "+err.Error())
	}
	return a.controller().HandleMedia(ctx, conversation.Inbound{Addr: addr, ExternalUserID: addr.ExternalChatID}, ref)
}

func (a *Adapter) HandleText(ctx context.Context, chatID sharedkernel.ChatID, text, _ string) error {
	addr, err := addrOf(chatID)
	if err != nil {
		return err
	}
	text = strings.TrimSpace(text)

	if !a.isMenuCommand(ctx, text) {
		if active, err := a.appSessionExists(ctx, chatID); err == nil && active {
			return a.controller().HandleText(ctx, conversation.Inbound{Addr: addr, ExternalUserID: addr.ExternalChatID}, text)
		}
	}

	switch text {
	case "/start", "/menu":
		return a.sendMainMenu(ctx, addr)
	case "/help":
		return a.Out.SendMenu(ctx, addr, a.renderText(ctx, texttpl.KeyHelp, nil), nil)
	case "/skip":
		return a.openCaseStep(ctx, chatID, "skip", nil)
	case "/exit":
		return a.openCaseStep(ctx, chatID, "exit", nil)
	case "/confirm":
		return a.openCaseStep(ctx, chatID, "confirm", nil)
	case "🎬 视频脱衣", "🔥 热门模版", "🤝 邀请赚钱", "👤 我的", "🔞 图片", "🔞 视频":
		return a.Out.SendMenu(ctx, addr, a.renderText(ctx, texttpl.KeyMenuUpdated, nil), nil)
	default:
		doc := a.loadMenu(ctx)
		if item, ok := FindEnabledItemByLabel(doc, text); ok {
			return a.actionDispatch(ctx, chatID, addr, item, "root")
		}
		return a.sendMainMenu(ctx, addr)
	}
}

func (a *Adapter) controller() *conversation.Controller {
	ctl := conversation.NewWithActionStore(a.ChannelID, tgInvokeDispatcher{adapter: a}, tgTextRenderer{out: a.Out}, a.store)
	ctl.Users = a.Users
	ctl.Outbound = a.Out
	ctl.Callbacks = tgCallbackCodec{}
	ctl.Blob = a.Blob
	ctl.Texts = a.Texts
	ctl.Notifications = a.notifications
	ctl.Menu = a.Menu
	ctl.Back = a.back
	return ctl
}

type tgInvokeDispatcher struct {
	adapter *Adapter
}

func (d tgInvokeDispatcher) Invoke(ctx context.Context, inv protocol.CapabilityInvoke) (protocol.Result, error) {
	if d.adapter == nil {
		return protocol.Result{}, fmt.Errorf("tg: adapter not configured")
	}
	return protocol.Result{}, d.adapter.dispatchInvoke(ctx, sharedkernel.ChatID(inv.ChatID), inv)
}

type tgTextRenderer struct {
	out protocol.Outbound
}

func (r tgTextRenderer) SendText(ctx context.Context, addr sharedkernel.ChannelAddr, text string) error {
	if r.out == nil {
		return nil
	}
	return r.out.SendText(ctx, addr, text)
}

type tgCallbackCodec struct{}

func (tgCallbackCodec) Invoke(token string) string { return CBInvoke + token }
func (tgCallbackCodec) MainMenu() string           { return CBMenu }
func (tgCallbackCodec) Back(target string) string  { return CBMenuBack + target }

func (a *Adapter) HandleCallback(ctx context.Context, chatID sharedkernel.ChatID, messageID int, data string, _ string) error {
	addr, err := addrOf(chatID)
	if err != nil {
		return err
	}
	if messageID != 0 {
		if err := a.Out.EditReplyMarkup(ctx, addr, messageID, nil); err != nil {
			slog.Error("tg clear callback markup", "err", err, "chat_id", chatID, "message_id", messageID)
		}
	}
	if strings.HasPrefix(data, CBInvoke) {
		token := strings.TrimPrefix(data, CBInvoke)
		inv, ok := a.store.Get(token)
		if !ok {
			return a.Out.SendText(ctx, addr, "操作已过期，重新选择。")
		}
		if inv.CapabilityID == "" {
			if id, ok := inv.Params["button_id"].(string); ok {
				if btn, found := mcdomain.FindButtonByID(a.loadMenu(ctx), id); found {
					a.store.Consume(token)
					return a.actionDispatch(ctx, chatID, addr, btn, inv.Nav.Back)
				}
			}
			return a.Out.SendText(ctx, addr, "未知操作")
		}
		return a.controller().HandleAction(ctx, conversation.Inbound{Addr: addr, ExternalUserID: addr.ExternalChatID}, token)
	}
	nav, err := TranslateMenuCallback(data)
	if err != nil {
		return a.Out.SendText(ctx, addr, "未知操作")
	}
	switch nav.kind {
	case "main":
		return a.sendMainMenu(ctx, addr)
	case "back":
		return a.handleBack(ctx, chatID, addr, nav.id)
	default:
		return a.Out.SendText(ctx, addr, "未知操作")
	}
}

// renderText resolves a configurable copy template (with defaults) and
// interpolates the supplied variables.
func (a *Adapter) renderText(ctx context.Context, key string, vars map[string]string) string {
	if a.Texts == nil {
		return texttpl.Render(texttpl.Default(key), vars)
	}
	return a.Texts.Render(ctx, a.ChannelID, key, vars)
}

func (a *Adapter) HandleUserNotify(ctx context.Context, n sharedkernel.UserNotify) error {
	return a.controller().HandleNotify(ctx, n)
}

// HandleNotify satisfies channel runtime NotifyHandler.
func (a *Adapter) HandleNotify(ctx context.Context, n sharedkernel.UserNotify) error {
	return a.HandleUserNotify(ctx, n)
}

func (a *Adapter) openCaseStep(ctx context.Context, chatID sharedkernel.ChatID, step string, extra map[string]any) error {
	inv, err := a.baseInvoke(ctx, chatID)
	if err != nil {
		return err
	}
	inv.CapabilityID = "open_case"
	inv.Params = map[string]any{"step": step}
	for k, v := range extra {
		inv.Params[k] = v
	}
	return a.dispatchInvoke(ctx, chatID, inv)
}

func (a *Adapter) actionDispatch(ctx context.Context, chatID sharedkernel.ChatID, addr sharedkernel.ChannelAddr, btn mcdomain.TreeButton, backCtx string) error {
	return a.controller().HandleMenuAction(ctx, conversation.Inbound{Addr: addr, ExternalUserID: addr.ExternalChatID}, btn, backCtx)
}

func (a *Adapter) sendCard(ctx context.Context, addr sharedkernel.ChannelAddr, card mcdomain.TreeCard, openerID, backCtx string) error {
	return a.controller().SendCard(ctx, conversation.Inbound{Addr: addr, ExternalUserID: addr.ExternalChatID}, card, openerID, backCtx)
}

func (a *Adapter) handleBack(ctx context.Context, chatID sharedkernel.ChatID, addr sharedkernel.ChannelAddr, target string) error {
	return a.controller().HandleBack(ctx, conversation.Inbound{Addr: addr, ExternalUserID: addr.ExternalChatID}, target)
}

func (a *Adapter) dispatchInvoke(ctx context.Context, chatID sharedkernel.ChatID, inv protocol.CapabilityInvoke) error {
	addr, err := addrOf(chatID)
	if err != nil {
		return err
	}
	if a.Registry == nil {
		return a.Out.SendText(ctx, addr, "功能暂不可用")
	}
	res, err := a.Registry.Invoke(ctx, inv)
	if err != nil {
		if errors.Is(err, convdomain.ErrNoActiveSession) || errors.Is(err, convdomain.ErrInvalidState) || errors.Is(err, convdomain.ErrInputOutOfOrder) {
			return a.sendMainMenu(ctx, addr)
		}
		slog.Error("capability invoke", "err", err, "capability", inv.CapabilityID)
		return a.Out.SendText(ctx, addr, "操作失败，稍后重试。")
	}
	return a.renderResult(ctx, addr, chatID, inv, res)
}

func (a *Adapter) renderResult(ctx context.Context, addr sharedkernel.ChannelAddr, chatID sharedkernel.ChatID, base protocol.CapabilityInvoke, res protocol.Result) error {
	return a.controller().RenderResult(ctx, conversation.Inbound{Addr: addr, ExternalUserID: addr.ExternalChatID}, base, res)
}

func (a *Adapter) sendMainMenu(ctx context.Context, addr sharedkernel.ChannelAddr) error {
	return a.controller().SendMainMenu(ctx, conversation.Inbound{Addr: addr, ExternalUserID: addr.ExternalChatID})
}

func (a *Adapter) appSessionExists(ctx context.Context, chatID sharedkernel.ChatID) (bool, error) {
	inv, err := a.baseInvoke(ctx, chatID)
	if err != nil {
		return false, err
	}
	inv.CapabilityID = "open_case"
	inv.Params = map[string]any{"step": "check"}
	res, err := a.Registry.Invoke(ctx, inv)
	if err != nil {
		return false, err
	}
	return res.Text == "active", nil
}

func extForMIME(mime string) string {
	switch strings.ToLower(mime) {
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	default:
		return ".jpg"
	}
}

func (a *Adapter) isMenuCommand(ctx context.Context, text string) bool {
	switch text {
	case "/start", "/menu", "/help", "/skip", "/exit", "/confirm":
		return true
	}
	doc := a.loadMenu(ctx)
	if _, ok := FindEnabledItemByLabel(doc, text); ok {
		return true
	}
	return false
}

func mustAddr(chatID sharedkernel.ChatID) sharedkernel.ChannelAddr {
	addr, err := addrOf(chatID)
	if err != nil {
		return sharedkernel.ChannelAddr{}
	}
	return addr
}

// mediaKindToMIME maps a menucard domain.Media.Kind to a Telegram-friendly mime
// hint. Unknown kinds fall back to "" so the URL extension-based guesser takes
// over.
func mediaKindToMIME(kind string) string {
	switch kind {
	case "image":
		return "image/jpeg"
	case "animation":
		return "image/gif"
	case "video":
		return "video/mp4"
	}
	return ""
}
