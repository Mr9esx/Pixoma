package tg

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/mr9esx/comfyui_tgbot/internal/channel/capability"
	"github.com/mr9esx/comfyui_tgbot/internal/channel/ports"
	"github.com/mr9esx/comfyui_tgbot/internal/channel/protocol"
	identitydomain "github.com/mr9esx/comfyui_tgbot/internal/identity/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/menu/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/blob"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

// Adapter translates Telegram events into capability invokes and renders results.
type Adapter struct {
	Out       ports.Outbound
	Media     ports.MediaBridge
	Users     ports.IdentityResolver
	Menu      MenuReader
	Extras    ExtrasReader
	Registry  *capability.Registry
	Blob      blob.Store
	ChannelID string
	mu        sync.Mutex
	notified  map[string]struct{}
	store     *invokeStore
}

func New(out ports.Outbound) *Adapter {
	return &Adapter{Out: out, notified: map[string]struct{}{}, store: newInvokeStore()}
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
	inv, err := a.baseInvoke(ctx, chatID)
	if err != nil {
		return err
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
	inv.CapabilityID = "open_case"
	inv.Params = map[string]any{
		"step": "media",
		"blob": map[string]any{"key": ref.Key, "mime": mime},
	}
	return a.dispatchInvoke(ctx, chatID, inv)
}

func (a *Adapter) HandleText(ctx context.Context, chatID sharedkernel.ChatID, text, _ string) error {
	addr, err := addrOf(chatID)
	if err != nil {
		return err
	}
	text = strings.TrimSpace(text)

	if !a.isMenuCommand(ctx, text) {
		if active, err := a.appSessionExists(ctx, chatID); err == nil && active {
			inv, ierr := a.baseInvoke(ctx, chatID)
			if ierr == nil {
				inv.CapabilityID = "open_case"
				inv.Params = map[string]any{"step": "text", "text": text}
				return a.dispatchInvoke(ctx, chatID, inv)
			}
		}
	}

	switch text {
	case "/start", "/menu":
		return a.sendMainMenu(ctx, addr)
	case "/help":
		return a.Out.SendMenu(ctx, addr, "帮助：点下方按钮选功能。菜单更新后发 /menu 刷新。", nil)
	case "/skip":
		return a.openCaseStep(ctx, chatID, "skip", nil)
	case "/exit":
		return a.openCaseStep(ctx, chatID, "exit", nil)
	case "/confirm":
		return a.openCaseStep(ctx, chatID, "confirm", nil)
	case "🎬 视频脱衣", "🔥 热门模版", "🤝 邀请赚钱", "👤 我的", "🔞 图片", "🔞 视频":
		return a.Out.SendMenu(ctx, addr, "菜单已更新，请使用下方新按钮。", nil)
	default:
		doc := a.loadMenu(ctx)
		if item, ok := FindEnabledRootByLabel(doc, text); ok {
			return a.menuItemDispatch(ctx, chatID, addr, item)
		}
		return a.sendMainMenu(ctx, addr)
	}
}

func (a *Adapter) HandleCallback(ctx context.Context, chatID sharedkernel.ChatID, _ string, data string, _ string) error {
	addr, err := addrOf(chatID)
	if err != nil {
		return err
	}
	if strings.HasPrefix(data, CBInvoke) {
		token := strings.TrimPrefix(data, CBInvoke)
		inv, ok := a.store.get(token)
		if !ok {
			return a.Out.SendText(ctx, addr, "操作已过期，请重新选择。")
		}
		return a.dispatchInvoke(ctx, chatID, inv)
	}
	nav, err := TranslateMenuCallback(data)
	if err != nil {
		return a.Out.SendText(ctx, addr, "未知操作")
	}
	switch nav.kind {
	case "main":
		return a.sendMainMenu(ctx, addr)
	case "group":
		return a.showGroup(ctx, addr, nav.id)
	default:
		return a.Out.SendText(ctx, addr, "未知操作")
	}
}

func (a *Adapter) HandleUserNotify(ctx context.Context, n sharedkernel.UserNotify) error {
	key := string(n.TaskID) + ":" + n.Kind
	a.mu.Lock()
	if _, ok := a.notified[key]; ok {
		a.mu.Unlock()
		return nil
	}
	a.notified[key] = struct{}{}
	a.mu.Unlock()

	addr, err := addrOf(n.ChatID)
	if err != nil {
		return err
	}
	if n.Kind == "task_succeeded" && len(n.Outputs) > 0 {
		caption := fmt.Sprintf("✅ Case 完成\ntask=%s", n.TaskID)
		if err := a.Out.SendMedia(ctx, addr, n.Outputs[0], caption); err != nil {
			return err
		}
		return a.Out.SendMenu(ctx, addr, "还要继续？点菜单再选一个 Case。", nil)
	}
	msg := fmt.Sprintf("任务 %s: %s", n.TaskID, n.Kind)
	if n.ErrorMsg != "" {
		msg += " — " + n.ErrorMsg
	}
	return a.Out.SendText(ctx, addr, msg)
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

func (a *Adapter) menuItemDispatch(ctx context.Context, chatID sharedkernel.ChatID, addr sharedkernel.ChannelAddr, item domain.MenuNode) error {
	if item.CapabilityID != "" {
		inv, err := a.baseInvoke(ctx, chatID)
		if err != nil {
			return err
		}
		inv.CapabilityID = item.CapabilityID
		inv.Params = item.Params
		inv.Nav = protocol.Nav{Back: "root"}
		return a.dispatchInvoke(ctx, chatID, inv)
	}
	if len(item.Children) > 0 {
		return a.showGroup(ctx, addr, item.ID)
	}
	if item.Reply != nil {
		return a.sendReplyMedia(ctx, addr, item)
	}
	msg := strings.TrimSpace(item.PlaceholderText)
	if msg == "" {
		msg = item.Label + "：暂未开放。"
	}
	return a.Out.SendMenu(ctx, addr, msg, nil)
}

func (a *Adapter) showGroup(ctx context.Context, addr sharedkernel.ChannelAddr, itemID string) error {
	tree := a.loadMenu(ctx)
	node, ok := findNodeByID(tree.Items, itemID)
	if !ok {
		return a.Out.SendText(ctx, addr, "菜单项不存在")
	}
	text := strings.TrimSpace(node.IntroText)
	if text == "" {
		text = node.Label
	}
	base, err := a.baseInvoke(ctx, sharedkernel.ChatID(sharedkernel.FormatChatID(addr)))
	if err != nil {
		return err
	}
	var rows [][]ports.Button
	for _, child := range node.Children {
		if !child.Enabled {
			continue
		}
		if child.CapabilityID != "" {
			inv := base
			inv.CapabilityID = child.CapabilityID
			inv.Params = child.Params
			inv.Nav = protocol.Nav{Back: node.ID}
			rows = append(rows, []ports.Button{{Text: child.Label, Data: CBInvoke + a.store.put(inv)}})
		} else {
			rows = append(rows, []ports.Button{{Text: child.Label, Data: CBMenuFolder + child.ID}})
		}
	}
	if node.ParentID == "" {
		rows = append(rows, []ports.Button{{Text: "⬅️ 返回", Data: CBMenu}})
	} else {
		rows = append(rows, []ports.Button{{Text: "⬅️ 返回", Data: CBMenuBack + node.ParentID}})
	}
	return a.Out.SendList(ctx, addr, text, rows)
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
		slog.Error("capability invoke", "err", err, "capability", inv.CapabilityID)
		return a.Out.SendText(ctx, addr, "操作失败，请稍后重试。")
	}
	return a.renderResult(ctx, addr, chatID, inv, res)
}

func (a *Adapter) renderResult(ctx context.Context, addr sharedkernel.ChannelAddr, chatID sharedkernel.ChatID, base protocol.CapabilityInvoke, res protocol.Result) error {
	for _, m := range res.Media {
		if err := a.Out.SendMedia(ctx, addr, sharedkernel.BlobRef{Key: m.Key, MIME: m.MIME}, res.Text); err != nil {
			return err
		}
	}
	if len(res.Options) > 0 {
		var rows [][]ports.Button
		for _, opt := range res.Options {
			next := base
			next.Params = opt.Value
			rows = append(rows, []ports.Button{{Text: opt.Label, Data: CBInvoke + a.store.put(next)}})
		}
		rows = append(rows, a.backButton(base.Nav))
		return a.Out.SendList(ctx, addr, res.Text, rows)
	}
	if res.Text != "" {
		return a.Out.SendText(ctx, addr, res.Text)
	}
	return nil
}

func (a *Adapter) backButton(nav protocol.Nav) []ports.Button {
	switch nav.Back {
	case "", "root":
		return []ports.Button{{Text: "⬅️ 返回主菜单", Data: CBMenu}}
	default:
		return []ports.Button{{Text: "⬅️ 返回", Data: CBMenuBack + nav.Back}}
	}
}

func (a *Adapter) sendMainMenu(ctx context.Context, addr sharedkernel.ChannelAddr) error {
	tree := a.loadMenu(ctx)
	items := make([]ports.MenuEntry, 0, len(tree.Items))
	for _, it := range tree.Items {
		if it.Enabled {
			items = append(items, ports.MenuEntry{ID: it.ID, Label: it.Label})
		}
	}
	return a.Out.SendMenu(ctx, addr, "欢迎使用 ComfyUI Bot\n请选择功能：", items)
}

func (a *Adapter) sendReplyMedia(ctx context.Context, addr sharedkernel.ChannelAddr, item domain.MenuNode) error {
	if item.Reply == nil {
		return a.Out.SendText(ctx, addr, "配置无效")
	}
	if text := strings.TrimSpace(item.Reply.Text); text != "" {
		if err := a.Out.SendText(ctx, addr, text); err != nil {
			return err
		}
	}
	for _, u := range item.Reply.Images {
		if err := a.Out.SendMediaURL(ctx, addr, u, ""); err != nil {
			slog.Error("tg reply_media photo failed", "err", err, "url", u)
		}
	}
	return nil
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
	if _, ok := FindEnabledRootByLabel(doc, text); ok {
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
