package tg

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/mr9esx/comfyui_tgbot/internal/channel/capability"
	"github.com/mr9esx/comfyui_tgbot/internal/channel/ports"
	"github.com/mr9esx/comfyui_tgbot/internal/channel/protocol"
	texttpl "github.com/mr9esx/comfyui_tgbot/internal/channel/text"
	identitydomain "github.com/mr9esx/comfyui_tgbot/internal/identity/domain"
	mcdomain "github.com/mr9esx/comfyui_tgbot/internal/menucard/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/blob"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

// Adapter translates Telegram events into capability invokes and renders results.
type Adapter struct {
	Out       ports.Outbound
	Media     ports.MediaBridge
	Users     ports.IdentityResolver
	Menu      MenuReader
	Registry  *capability.Registry
	Blob      blob.Store
	ChannelID string
	// Texts resolves configurable copy templates; nil falls back to built-ins.
	Texts    texttpl.Renderer
	back     *backStack
	mu       sync.Mutex
	notified map[string]struct{}
	store    *invokeStore
}

func New(out ports.Outbound) *Adapter {
	return &Adapter{
		Out:      out,
		Texts:    texttpl.StaticRenderer{},
		notified: map[string]struct{}{},
		store:    newInvokeStore(),
		back:     newBackStack(),
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
		inv, ok := a.store.get(token)
		if !ok {
			return a.Out.SendText(ctx, addr, "操作已过期，请重新选择。")
		}
		if inv.CapabilityID == "" {
			if id, ok := inv.Params["button_id"].(string); ok {
				if btn, found := mcdomain.FindButtonByID(a.loadMenu(ctx), id); found {
					a.store.consume(token)
					return a.actionDispatch(ctx, chatID, addr, btn, inv.Nav.Back)
				}
			}
			return a.Out.SendText(ctx, addr, "未知操作")
		}
		err := a.dispatchInvoke(ctx, chatID, inv)
		if err == nil {
			a.store.consume(token)
		}
		return err
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
	if n.Kind == "task_succeeded" {
		if len(n.Outputs) > 0 {
			for i, ref := range n.Outputs {
				caption := ""
				if i == 0 {
					caption = a.renderText(ctx, texttpl.KeyWorkflowDone, map[string]string{"task_id": string(n.TaskID)})
				}
				if strings.HasPrefix(ref.MIME, "text/") {
					if a.Blob == nil {
						continue
					}
					rc, err := a.Blob.Get(ctx, ref)
					if err != nil {
						return err
					}
					raw, readErr := io.ReadAll(rc)
					rc.Close()
					if readErr != nil {
						return readErr
					}
					if err := a.Out.SendText(ctx, addr, string(raw)); err != nil {
						return err
					}
					continue
				}
				if err := a.Out.SendMedia(ctx, addr, ref, caption, nil); err != nil {
					return err
				}
			}
		} else {
			if err := a.Out.SendText(ctx, addr, a.renderText(ctx, texttpl.KeyWorkflowDone, map[string]string{"task_id": string(n.TaskID)})); err != nil {
				return err
			}
		}
		return a.Out.SendMenu(ctx, addr, a.renderText(ctx, texttpl.KeyWorkflowDoneFollowp, nil), nil)
	}
	if n.Kind == "session_terminated" {
		msg := a.renderText(ctx, texttpl.KeySessionTerminated, map[string]string{"error_msg": n.ErrorMsg})
		return a.Out.SendText(ctx, addr, msg)
	}
	if n.Kind == "task_failed" || n.Kind == "task_cancelled" {
		key := texttpl.KeyTaskFailed
		if n.Kind == "task_cancelled" {
			key = texttpl.KeyTaskCancelled
		}
		vars := map[string]string{
			"task_id":   string(n.TaskID),
			"status":    strings.TrimPrefix(n.Kind, "task_"),
			"error_msg": n.ErrorMsg,
		}
		return a.Out.SendText(ctx, addr, a.renderText(ctx, key, vars))
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

func (a *Adapter) actionDispatch(ctx context.Context, chatID sharedkernel.ChatID, addr sharedkernel.ChannelAddr, btn mcdomain.TreeButton, backCtx string) error {
	action := btn.Action
	switch action.Type {
	case "open_card":
		card := action.Card
		if card == nil {
			compiled := mcdomain.Compile(a.loadMenu(ctx))
			if c, ok := compiled.CardByOpenerID[btn.ID]; ok {
				card = &c
			}
		}
		if card == nil {
			return a.Out.SendText(ctx, addr, "卡片不存在或已删除")
		}
		a.back.push(string(chatID), backCtx)
		return a.sendCard(ctx, addr, *card, btn.ID, backCtx)
	case "open_workflow":
		inv, err := a.baseInvoke(ctx, chatID)
		if err != nil {
			return err
		}
		inv.CapabilityID = "open_case"
		inv.Params = map[string]any{"step": "preview", "case_id": action.WorkflowID}
		inv.Nav = protocol.Nav{Back: backCtx}
		return a.dispatchInvoke(ctx, chatID, inv)
	case "list_tasks":
		inv, err := a.baseInvoke(ctx, chatID)
		if err != nil {
			return err
		}
		inv.CapabilityID = "list_tasks"
		inv.Params = map[string]any{}
		inv.Nav = protocol.Nav{Back: backCtx}
		return a.dispatchInvoke(ctx, chatID, inv)
	case "send_text", "copy_text":
		return a.Out.SendText(ctx, addr, action.Text)
	case "send_media":
		for _, m := range action.Media {
			if err := a.Out.SendMediaURL(ctx, addr, m.URL, mediaKindToMIME(m.Kind), action.Text, nil); err != nil {
				return err
			}
		}
		if action.Text != "" && len(action.Media) == 0 {
			return a.Out.SendText(ctx, addr, action.Text)
		}
		return nil
	case "open_url":
		return a.Out.SendText(ctx, addr, action.URL)
	default:
		return a.Out.SendText(ctx, addr, "菜单配置无效")
	}
}

func (a *Adapter) sendCard(ctx context.Context, addr sharedkernel.ChannelAddr, card mcdomain.TreeCard, openerID, backCtx string) error {
	for _, m := range card.Media {
		if err := a.Out.SendMediaURL(ctx, addr, m.URL, mediaKindToMIME(m.Kind), card.Text, nil); err != nil {
			return err
		}
	}
	if len(card.Buttons) == 0 {
		if card.Text != "" {
			return a.Out.SendText(ctx, addr, card.Text)
		}
		return nil
	}
	base, err := a.baseInvoke(ctx, sharedkernel.ChatID(sharedkernel.FormatChatID(addr)))
	if err != nil {
		return err
	}
	rows := make([][]ports.Button, 0, len(card.Buttons)+1)
	for _, b := range card.Buttons {
		inv := base
		inv.CapabilityID = ""
		inv.Params = map[string]any{"button_id": b.ID}
		inv.Nav = protocol.Nav{Back: openerID}
		rows = append(rows, []ports.Button{{Text: b.Label, Data: CBInvoke + a.store.put(inv)}})
	}
	backData := CBMenuBack + backCtx
	rows = append(rows, []ports.Button{{Text: "‹ 返回", Data: backData}})
	return a.Out.SendList(ctx, addr, card.Text, rows)
}

func (a *Adapter) handleBack(ctx context.Context, chatID sharedkernel.ChatID, addr sharedkernel.ChannelAddr, target string) error {
	if target == "root" {
		a.back.clear(string(chatID))
		return a.sendMainMenu(ctx, addr)
	}
	_, ok := a.back.pop(string(chatID))
	if !ok {
		return a.sendMainMenu(ctx, addr)
	}
	compiled := mcdomain.Compile(a.loadMenu(ctx))
	card, found := compiled.CardByOpenerID[target]
	if !found {
		return a.sendMainMenu(ctx, addr)
	}
	source := "root"
	if top, ok := a.back.top(string(chatID)); ok {
		source = top
	}
	return a.sendCard(ctx, addr, card, target, source)
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
	var rows [][]ports.Button
	if len(res.Options) > 0 {
		for _, opt := range res.Options {
			next := base
			next.Params = opt.Value
			rows = append(rows, []ports.Button{{Text: opt.Label, Data: CBInvoke + a.store.put(next)}})
		}
		rows = append(rows, a.backButton(base.Nav))
	}
	hasMedia := len(res.Media) > 0 || len(res.MediaURLs) > 0
	first := true
	for _, m := range res.Media {
		caption, buttons := mediaExtras(res.Text, rows, first)
		first = false
		if err := a.Out.SendMedia(ctx, addr, sharedkernel.BlobRef{Key: m.Key, MIME: m.MIME}, caption, buttons); err != nil {
			return err
		}
	}
	for _, u := range res.MediaURLs {
		caption, buttons := mediaExtras(res.Text, rows, first)
		first = false
		if err := a.Out.SendMediaURL(ctx, addr, u, "", caption, buttons); err != nil {
			return err
		}
	}
	if hasMedia {
		return nil
	}
	if len(rows) > 0 {
		return a.Out.SendList(ctx, addr, res.Text, rows)
	}
	if res.Text != "" {
		return a.Out.SendText(ctx, addr, res.Text)
	}
	return nil
}

func mediaExtras(text string, rows [][]ports.Button, first bool) (caption string, buttons [][]ports.Button) {
	if !first {
		return "", nil
	}
	return text, rows
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
	menu := a.loadMenu(ctx)
	items := make([]ports.MenuEntry, 0, len(menu.Items))
	for _, it := range menu.Items {
		items = append(items, ports.MenuEntry{ID: it.ID, Label: it.Label})
	}
	return a.Out.SendMenu(ctx, addr, a.renderText(ctx, texttpl.KeyWelcome, nil), items)
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
