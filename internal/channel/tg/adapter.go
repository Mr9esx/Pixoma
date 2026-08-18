package tg

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mr9esx/comfyui_tgbot/internal/channel/ports"
	convdomain "github.com/mr9esx/comfyui_tgbot/internal/conversation/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/menu/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/packaging/botapp"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/blob"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

// Adapter translates Telegram events into normalized actions over ports and the app facade.
type Adapter struct {
	App       *botapp.Facade
	Out       ports.Outbound
	Media     ports.MediaBridge
	Users     ports.IdentityResolver
	Menu      MenuReader
	Extras    ExtrasReader
	ChannelID string
	mu        sync.Mutex
	notified  map[string]struct{}
}

func New(app *botapp.Facade, out ports.Outbound) *Adapter {
	return &Adapter{App: app, Out: out, notified: map[string]struct{}{}}
}

func addrOf(chatID sharedkernel.ChatID) (sharedkernel.ChannelAddr, error) {
	return sharedkernel.ParseChatID(string(chatID))
}

// HandleUserMedia accepts a Photo/image Document when the current field is image.
func (a *Adapter) HandleUserMedia(ctx context.Context, chatID sharedkernel.ChatID, fileID, mime string) error {
	addr, err := addrOf(chatID)
	if err != nil {
		return err
	}
	ft, err := a.currentFieldType(ctx, chatID)
	if err != nil {
		return a.Out.SendText(ctx, addr, "当前没有进行中的 Case，请先开始。")
	}
	if ft != "image" {
		return a.Out.SendText(ctx, addr, "当前不需要图片，请按提示输入。")
	}
	if a.Media == nil {
		return a.Out.SendText(ctx, addr, "无法下载图片：未配置下载器")
	}
	if a.App == nil || a.App.Blob == nil {
		return a.Out.SendText(ctx, addr, "无法保存图片：未配置存储")
	}
	data, err := a.Media.Download(ctx, fileID, mime)
	if err != nil {
		slog.Error("tg download user media", "err", err, "chat_id", chatID, "file_id", fileID)
		return a.Out.SendText(ctx, addr, "下载图片失败")
	}
	if mime == "" {
		mime = "image/jpeg"
	}
	key := fmt.Sprintf("tg/%s/%d%s", addr.ExternalChatID, time.Now().UnixNano(), extForMIME(mime))
	ref, err := a.App.Blob.Put(ctx, key, bytes.NewReader(data), blob.PutOptions{MIME: mime})
	if err != nil {
		return a.Out.SendText(ctx, addr, "保存图片失败: "+err.Error())
	}
	view, err := a.App.SubmitInput(ctx, chatID, convdomain.DraftValue{Blob: &ref})
	if err != nil {
		return a.Out.SendText(ctx, addr, "提交失败: "+err.Error())
	}
	return a.renderSession(ctx, addr, view)
}

func (a *Adapter) HandleText(ctx context.Context, chatID sharedkernel.ChatID, text, userID string) error {
	addr, err := addrOf(chatID)
	if err != nil {
		return err
	}
	text = strings.TrimSpace(text)

	if !a.isMenuCommand(ctx, text) {
		if _, err := a.App.GetSession(ctx, chatID); err == nil {
			return a.submitText(ctx, addr, chatID, text)
		}
	}

	switch text {
	case "/start", "/menu":
		return a.sendMainMenu(ctx, addr)
	case "/help":
		return a.Out.SendMenu(ctx, addr, "帮助：点「图片」选 Case → 预览 → 开始 → 输入 prompt → 确认 → 等待出图。\n\n菜单更新后，点任意底部按钮或发 /menu 即可刷新。", nil)
	case "/skip":
		return a.handleSkip(ctx, addr, chatID)
	case "/exit":
		return a.handleExit(ctx, addr, chatID)
	case "/confirm":
		return a.handleConfirm(ctx, addr, chatID)
	case "🎬 视频脱衣", "🔥 热门模版", "🤝 邀请赚钱", "👤 我的", "🔞 图片", "🔞 视频":
		return a.Out.SendMenu(ctx, addr, "菜单已更新，请使用下方新按钮。", nil)
	default:
		if strings.HasPrefix(text, "/start_case ") {
			id := strings.TrimSpace(strings.TrimPrefix(text, "/start_case "))
			return a.startCase(ctx, addr, chatID, sharedkernel.CaseID(id), userID)
		}
		doc := a.loadMenu(ctx)
		if item, ok := FindEnabledRootByLabel(doc, text); ok {
			return a.dispatchMenuItem(ctx, addr, chatID, userID, item)
		}
		return a.sendMainMenu(ctx, addr)
	}
}

func (a *Adapter) HandleCallback(ctx context.Context, chatID sharedkernel.ChatID, callbackID, data, userID string) error {
	addr, err := addrOf(chatID)
	if err != nil {
		return err
	}
	action, err := TranslateCallback(data)
	if err != nil {
		return a.Out.SendText(ctx, addr, "未知操作")
	}
	return a.dispatchAction(ctx, addr, chatID, userID, action)
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
		return a.Out.SendMenu(ctx, addr, "还要继续？点菜单「"+BtnImage+"」再选一个 Case。", nil)
	}
	msg := fmt.Sprintf("任务 %s: %s", n.TaskID, n.Kind)
	if n.ErrorMsg != "" {
		msg += " — " + n.ErrorMsg
	}
	return a.Out.SendText(ctx, addr, msg)
}

func (a *Adapter) dispatchAction(ctx context.Context, addr sharedkernel.ChannelAddr, chatID sharedkernel.ChatID, userID string, action ports.Action) error {
	switch action.Type {
	case ports.ActionOpenMenu:
		return a.sendMainMenu(ctx, addr)
	case ports.ActionOpenFolder:
		return a.showMenuFolder(ctx, addr, action.MenuItemID)
	case ports.ActionOpenCase:
		return a.showCasePreview(ctx, addr, sharedkernel.CaseID(action.CaseID), action.BackRef)
	case ports.ActionStartCase:
		return a.startCase(ctx, addr, chatID, sharedkernel.CaseID(action.CaseID), userID)
	case ports.ActionSubmitText:
		return a.submitText(ctx, addr, chatID, action.Text)
	case ports.ActionConfirm:
		return a.handleConfirm(ctx, addr, chatID)
	case ports.ActionSkip:
		return a.handleSkip(ctx, addr, chatID)
	case ports.ActionExit:
		return a.handleExit(ctx, addr, chatID)
	case ports.ActionContinue:
		return a.handleContinue(ctx, addr, chatID)
	case ports.ActionReplaceStart:
		return a.replaceAndStart(ctx, addr, chatID, sharedkernel.CaseID(action.CaseID), userID)
	default:
		return a.Out.SendText(ctx, addr, "未知操作")
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

func (a *Adapter) dispatchMenuItem(ctx context.Context, addr sharedkernel.ChannelAddr, chatID sharedkernel.ChatID, userID string, item domain.MenuNode) error {
	switch item.Kind {
	case domain.KindFolder:
		return a.showMenuFolder(ctx, addr, item.ID)
	case domain.KindOpenCase:
		if len(item.CaseIDs) == 0 {
			return a.Out.SendMenu(ctx, addr, "菜单配置无效：缺少 case", nil)
		}
		return a.showCasePreview(ctx, addr, sharedkernel.CaseID(item.CaseIDs[0]), "root")
	case domain.KindPlaceholder:
		msg := strings.TrimSpace(item.PlaceholderText)
		if msg == "" {
			msg = item.Label + "：暂未开放，请先体验「" + BtnImage + "」。"
		}
		return a.Out.SendMenu(ctx, addr, msg, nil)
	case domain.KindReplyMedia:
		return a.sendReplyMedia(ctx, addr, item)
	default:
		return a.Out.SendMenu(ctx, addr, "未知菜单动作", nil)
	}
}

func (a *Adapter) showMenuFolder(ctx context.Context, addr sharedkernel.ChannelAddr, itemID string) error {
	tree := a.loadMenu(ctx)
	node, ok := findNodeByID(tree.Items, itemID)
	if !ok {
		return a.Out.SendText(ctx, addr, "菜单项不存在")
	}

	text := strings.TrimSpace(node.IntroText)
	if text == "" {
		text = node.Label
	}

	backRef := "root"
	if node.ParentID != "" {
		backRef = node.ParentID
	}
	var rows [][]ports.Button
	for _, caseID := range node.CaseIDs {
		c, err := a.App.GetCase(ctx, sharedkernel.CaseID(caseID))
		if err != nil {
			continue
		}
		doc := c.Document
		rows = append(rows, []ports.Button{{
			Text:   fmt.Sprintf("%s · ¥%.0f", doc.Name, doc.Price),
			Action: ports.Action{Type: ports.ActionOpenCase, CaseID: caseID, BackRef: backRef},
		}})
	}
	for _, child := range node.Children {
		if !child.Enabled {
			continue
		}
		if child.Kind == domain.KindFolder {
			rows = append(rows, []ports.Button{{
				Text:   "📁 " + child.Label,
				Action: ports.Action{Type: ports.ActionOpenFolder, MenuItemID: child.ID},
			}})
		}
	}
	if node.ParentID == "" {
		rows = append(rows, []ports.Button{{Text: "⬅️ 返回", Action: ports.Action{Type: ports.ActionOpenMenu}}})
	} else {
		rows = append(rows, []ports.Button{{Text: "⬅️ 返回", Action: ports.Action{Type: ports.ActionOpenFolder, MenuItemID: node.ParentID}}})
	}
	return a.Out.SendList(ctx, addr, text, rows)
}

func (a *Adapter) sendReplyMedia(ctx context.Context, addr sharedkernel.ChannelAddr, item domain.MenuNode) error {
	if item.Reply == nil {
		return a.Out.SendText(ctx, addr, "菜单配置无效：缺少 reply")
	}
	text := strings.TrimSpace(item.Reply.Text)
	if text != "" {
		if err := a.Out.SendText(ctx, addr, text); err != nil {
			return err
		}
	}
	okCount := 0
	for _, u := range item.Reply.Images {
		if err := a.Out.SendMediaURL(ctx, addr, u, ""); err != nil {
			slog.Error("tg reply_media photo failed", "err", err, "url", u, "chat_id", addr.ExternalChatID)
			continue
		}
		okCount++
	}
	if text == "" && okCount == 0 && len(item.Reply.Images) > 0 {
		return a.Out.SendText(ctx, addr, "图片发送失败，请稍后重试")
	}
	return nil
}

func (a *Adapter) showCasePreview(ctx context.Context, addr sharedkernel.ChannelAddr, id sharedkernel.CaseID, backRef string) error {
	c, err := a.App.GetCase(ctx, id)
	if err != nil {
		return a.Out.SendText(ctx, addr, "Case 不存在: "+err.Error())
	}
	doc := c.Document
	var b strings.Builder
	fmt.Fprintf(&b, "📎 %s\n", doc.Name)
	if doc.Description != "" {
		fmt.Fprintf(&b, "%s\n", doc.Description)
	}
	fmt.Fprintf(&b, "价格：%.0f\n", doc.Price)
	fmt.Fprintf(&b, "输入：")
	for i, in := range doc.Inputs {
		if i > 0 {
			b.WriteString(" → ")
		}
		req := ""
		if in.Required {
			req = "*"
		}
		fmt.Fprintf(&b, "%s%s", in.Key, req)
	}
	b.WriteString("\n\n预览说明：")
	if doc.Preview != "" {
		b.WriteString(doc.Preview)
	} else {
		b.WriteString("（mock）确认后将返回一张示例图")
	}

	back := ports.Action{Type: ports.ActionOpenMenu}
	if backRef != "" && backRef != "root" {
		back = ports.Action{Type: ports.ActionOpenFolder, MenuItemID: backRef}
	}
	rows := [][]ports.Button{
		{{Text: "▶ 开始 Case", Action: ports.Action{Type: ports.ActionStartCase, CaseID: string(id)}}},
		{{Text: "« 返回列表", Action: back}},
	}
	return a.Out.SendList(ctx, addr, b.String(), rows)
}

func (a *Adapter) startCase(ctx context.Context, addr sharedkernel.ChannelAddr, chatID sharedkernel.ChatID, id sharedkernel.CaseID, userID string) error {
	view, err := a.App.StartCase(ctx, botapp.StartCaseCmd{
		ChatID: chatID,
		UserID: userID,
		CaseID: id,
	})
	if errors.Is(err, convdomain.ErrSessionLocked) {
		return a.showSessionConflict(ctx, addr, chatID, id)
	}
	if err != nil {
		return a.Out.SendText(ctx, addr, "无法开始: "+err.Error())
	}
	return a.renderSession(ctx, addr, view)
}

func (a *Adapter) showSessionConflict(ctx context.Context, addr sharedkernel.ChannelAddr, chatID sharedkernel.ChatID, want sharedkernel.CaseID) error {
	cur, err := a.App.GetSession(ctx, chatID)
	if err != nil {
		return a.Out.SendText(ctx, addr, "已有进行中的填表，但读取会话失败，请稍后再试。")
	}

	curName := string(cur.CaseID)
	if c, err := a.App.GetCase(ctx, cur.CaseID); err == nil {
		curName = c.Document.Name
	}
	wantName := string(want)
	if c, err := a.App.GetCase(ctx, want); err == nil {
		wantName = c.Document.Name
	}

	step := "填写中"
	if cur.Status == convdomain.StatusConfirming {
		step = "待确认执行"
	} else if key := currentKey(cur); key != "(done)" {
		step = "填写中（当前项：" + key + "）"
	}

	same := cur.CaseID == want
	var msg string
	var rows [][]ports.Button
	if same {
		msg = fmt.Sprintf("你正在填写「%s」\n进度：%s\n\n要接着填，还是退出后重来？", curName, step)
		rows = [][]ports.Button{
			{{Text: "继续当前 Case", Action: ports.Action{Type: ports.ActionContinue}}},
			{{Text: "退出当前 Case", Action: ports.Action{Type: ports.ActionExit}}},
		}
	} else {
		msg = fmt.Sprintf("检测到未完成的 Case\n\n正在进行：%s\n进度：%s\n你刚想开始：%s\n\n请选择：继续刚才的，或废弃它并开始新选的。", curName, step, wantName)
		rows = [][]ports.Button{
			{{Text: "继续当前 Case", Action: ports.Action{Type: ports.ActionContinue}}},
			{{Text: "退出当前 Case", Action: ports.Action{Type: ports.ActionExit}}},
			{{Text: "开始当前", Action: ports.Action{Type: ports.ActionReplaceStart, CaseID: string(want)}}},
		}
	}
	return a.Out.SendList(ctx, addr, msg, rows)
}

func (a *Adapter) handleContinue(ctx context.Context, addr sharedkernel.ChannelAddr, chatID sharedkernel.ChatID) error {
	view, err := a.App.GetSession(ctx, chatID)
	if err != nil {
		return a.Out.SendMenu(ctx, addr, "当前没有进行中的 Case，已回到菜单。", nil)
	}
	return a.renderSession(ctx, addr, view)
}

func (a *Adapter) replaceAndStart(ctx context.Context, addr sharedkernel.ChannelAddr, chatID sharedkernel.ChatID, id sharedkernel.CaseID, userID string) error {
	_ = a.App.ExitSession(ctx, chatID)
	return a.startCase(ctx, addr, chatID, id, userID)
}

func (a *Adapter) submitText(ctx context.Context, addr sharedkernel.ChannelAddr, chatID sharedkernel.ChatID, text string) error {
	ft, err := a.currentFieldType(ctx, chatID)
	if err != nil {
		return a.Out.SendText(ctx, addr, "提交失败: "+err.Error())
	}
	var draft convdomain.DraftValue
	switch ft {
	case "image":
		return a.Out.SendText(ctx, addr, "当前需要一张图片，请发送 Photo 或图片文件。")
	case "number":
		n, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return a.Out.SendText(ctx, addr, "请输入合法数字，例如 42")
		}
		draft = convdomain.DraftValue{Number: &n}
	case "boolean":
		b, err := strconv.ParseBool(text)
		if err != nil {
			return a.Out.SendText(ctx, addr, "请输入 true 或 false")
		}
		draft = convdomain.DraftValue{Bool: &b}
	default:
		draft = convdomain.DraftValue{Text: &text}
	}
	view, err := a.App.SubmitInput(ctx, chatID, draft)
	if err != nil {
		return a.Out.SendText(ctx, addr, "提交失败: "+err.Error())
	}
	return a.renderSession(ctx, addr, view)
}

func (a *Adapter) currentFieldType(ctx context.Context, chatID sharedkernel.ChatID) (string, error) {
	view, err := a.App.GetSession(ctx, chatID)
	if err != nil {
		return "", err
	}
	key := currentKey(view)
	if key == "(done)" {
		return "", errors.New("no current input field")
	}
	c, err := a.App.GetCase(ctx, view.CaseID)
	if err != nil {
		return "", err
	}
	for _, in := range c.Document.Inputs {
		if in.Key == key {
			return in.Type, nil
		}
	}
	return "", fmt.Errorf("unknown field %q", key)
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

func (a *Adapter) handleSkip(ctx context.Context, addr sharedkernel.ChannelAddr, chatID sharedkernel.ChatID) error {
	view, err := a.App.SkipInput(ctx, chatID)
	if err != nil {
		return a.Out.SendText(ctx, addr, "跳过失败: "+err.Error())
	}
	return a.renderSession(ctx, addr, view)
}

func (a *Adapter) handleExit(ctx context.Context, addr sharedkernel.ChannelAddr, chatID sharedkernel.ChatID) error {
	if err := a.App.ExitSession(ctx, chatID); err != nil {
		return a.Out.SendText(ctx, addr, "退出失败: "+err.Error())
	}
	return a.Out.SendMenu(ctx, addr, "已退出当前 Case，可以重新选择。", nil)
}

func (a *Adapter) handleConfirm(ctx context.Context, addr sharedkernel.ChannelAddr, chatID sharedkernel.ChatID) error {
	if err := a.Out.SendText(ctx, addr, "⏳ 已提交，正在生成…"); err != nil {
		return err
	}
	res, err := a.App.ConfirmRun(ctx, botapp.ConfirmRunCmd{ChatID: chatID})
	if err != nil {
		return a.Out.SendText(ctx, addr, "确认失败: "+err.Error())
	}
	tasks, err := a.App.ListMyTasks(ctx, chatID, 20)
	if err == nil {
		for _, t := range tasks {
			if t.ID != res.TaskID {
				continue
			}
			switch t.Status {
			case sharedkernel.TaskSucceeded, sharedkernel.TaskFailed, sharedkernel.TaskCancelled:
				return nil
			}
			break
		}
	}
	return a.Out.SendText(ctx, addr, fmt.Sprintf("已排队\ntask=%s\n完成后会把图片发回来。", res.TaskID))
}

func (a *Adapter) renderSession(ctx context.Context, addr sharedkernel.ChannelAddr, view *botapp.SessionView) error {
	if view.Status == convdomain.StatusConfirming {
		return a.Out.SendList(ctx, addr, "输入完成，确认执行？", [][]ports.Button{
			{{Text: "✅ 确认生成", Action: ports.Action{Type: ports.ActionConfirm}}, {Text: "✕ 退出", Action: ports.Action{Type: ports.ActionExit}}},
		})
	}
	key := currentKey(view)
	c, err := a.App.GetCase(ctx, view.CaseID)
	hint := key
	if err == nil {
		for _, in := range c.Document.Inputs {
			if in.Key == key {
				if in.Description != "" {
					hint = in.Description
				}
				rows := [][]ports.Button{{{Text: "✕ 退出", Action: ports.Action{Type: ports.ActionExit}}}}
				if !in.Required || in.SkipAllowed {
					rows = [][]ports.Button{
						{{Text: "跳过", Action: ports.Action{Type: ports.ActionSkip}}, {Text: "✕ 退出", Action: ports.Action{Type: ports.ActionExit}}},
					}
				}
				return a.Out.SendList(ctx, addr, fmt.Sprintf("请输入「%s」\n%s", key, hint), rows)
			}
		}
	}
	return a.Out.SendList(ctx, addr, "请输入「"+key+"」", [][]ports.Button{
		{{Text: "✕ 退出", Action: ports.Action{Type: ports.ActionExit}}},
	})
}

func currentKey(view *botapp.SessionView) string {
	if view.Index >= 0 && view.Index < len(view.Keys) {
		return view.Keys[view.Index]
	}
	return "(done)"
}

func (a *Adapter) isMenuCommand(ctx context.Context, text string) bool {
	switch text {
	case "/start", "/menu", "/help", "/skip", "/exit", "/confirm":
		return true
	}
	if strings.HasPrefix(text, "/start_case ") {
		return true
	}
	doc := a.loadMenu(ctx)
	if _, ok := FindEnabledRootByLabel(doc, text); ok {
		return true
	}
	switch text {
	case BtnImage, BtnVideo, BtnRecharge, BtnCheckIn, BtnProfile, BtnHelp:
		return true
	}
	return false
}
