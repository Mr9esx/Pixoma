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

	catalogdomain "github.com/mr9esx/comfyui_tgbot/internal/catalog/domain"
	convdomain "github.com/mr9esx/comfyui_tgbot/internal/conversation/domain"
	identitydomain "github.com/mr9esx/comfyui_tgbot/internal/identity/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/packaging/botapp"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/blob"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
	tgmenudomain "github.com/mr9esx/comfyui_tgbot/internal/tgmenu/domain"
)

// FileDownloader fetches Telegram file bytes by file_id (injectable for tests).
type FileDownloader func(ctx context.Context, fileID string) ([]byte, error)

type Adapter struct {
	App      *botapp.Facade
	Out      Messenger
	Download FileDownloader
	// Users is optional; when set, message/callback From is upserted before handling.
	Users identitydomain.Repository
	// Menu is optional; when nil or load fails, DefaultSeed is used.
	Menu     MenuReader
	mu       sync.Mutex
	notified map[string]struct{}
}

// UpsertFromTG persists Telegram From into the users table when Users is configured.
// Returns the internal user id when upsert succeeds; empty string otherwise.
func (a *Adapter) UpsertFromTG(ctx context.Context, in identitydomain.UpsertFrom) (string, error) {
	if a == nil || a.Users == nil {
		return "", nil
	}
	if in.TgUserID == 0 {
		return "", nil
	}
	u, err := a.Users.UpsertByTgUserID(ctx, in)
	if err != nil {
		return "", err
	}
	return u.ID, nil
}

func New(app *botapp.Facade, out Messenger) *Adapter {
	return &Adapter{App: app, Out: out, notified: map[string]struct{}{}}
}

// HandleUserMedia accepts a Photo/image Document when the current field is image.
func (a *Adapter) HandleUserMedia(ctx context.Context, chatID int64, fileID, mime string) error {
	ft, err := a.currentFieldType(ctx, chatID)
	if err != nil {
		return a.Out.SendText(ctx, chatID, "当前没有进行中的 Case，请先开始。")
	}
	if ft != "image" {
		return a.Out.SendText(ctx, chatID, "当前不需要图片，请按提示输入。")
	}
	if a.Download == nil {
		return a.Out.SendText(ctx, chatID, "无法下载图片：未配置下载器")
	}
	if a.App == nil || a.App.Blob == nil {
		return a.Out.SendText(ctx, chatID, "无法保存图片：未配置存储")
	}
	data, err := a.Download(ctx, fileID)
	if err != nil {
		slog.Error("tg download user media", "err", err, "chat_id", chatID, "file_id", fileID)
		return a.Out.SendText(ctx, chatID, "下载图片失败")
	}
	if mime == "" {
		mime = "image/jpeg"
	}
	key := fmt.Sprintf("tg/%d/%d%s", chatID, time.Now().UnixNano(), extForMIME(mime))
	ref, err := a.App.Blob.Put(ctx, key, bytes.NewReader(data), blob.PutOptions{MIME: mime})
	if err != nil {
		return a.Out.SendText(ctx, chatID, "保存图片失败: "+err.Error())
	}
	view, err := a.App.SubmitInput(ctx, sharedkernel.ChatID(chatID), convdomain.DraftValue{Blob: &ref})
	if err != nil {
		return a.Out.SendText(ctx, chatID, "提交失败: "+err.Error())
	}
	return a.renderSession(ctx, chatID, view)
}

func (a *Adapter) HandleText(ctx context.Context, chatID int64, text, userID string) error {
	text = strings.TrimSpace(text)

	// Active session: treat free text as input (unless menu command).
	if !a.isMenuCommand(ctx, text) {
		if _, err := a.App.GetSession(ctx, sharedkernel.ChatID(chatID)); err == nil {
			return a.submitText(ctx, chatID, text)
		}
	}

	switch text {
	case "/start", "/menu", CBMenu:
		return a.sendMainMenu(ctx, chatID)
	case "/cases":
		return a.showCasesByTag(ctx, chatID, "image")
	case "/help":
		return a.Out.SendMenu(ctx, chatID, "帮助：点「图片」选 Case → 预览 → 开始 → 输入 prompt → 确认 → 等待出图。\n\n菜单更新后，点任意底部按钮或发 /menu 即可刷新。")
	case "/skip":
		return a.handleSkip(ctx, chatID)
	case "/exit":
		return a.handleExit(ctx, chatID)
	case "/confirm":
		return a.handleConfirm(ctx, chatID)
	case "🎬 视频脱衣", "🔥 热门模版", "🤝 邀请赚钱", "👤 我的", "🔞 图片", "🔞 视频":
		// Legacy keyboard labels from older builds — refresh to current menu.
		return a.Out.SendMenu(ctx, chatID, "菜单已更新，请使用下方新按钮。")
	default:
		if strings.HasPrefix(text, "/start_case ") {
			id := strings.TrimSpace(strings.TrimPrefix(text, "/start_case "))
			return a.startCase(ctx, chatID, sharedkernel.CaseID(id), userID)
		}
		doc := a.loadMenu(ctx)
		if item, ok := FindEnabledRootByLabel(doc, text); ok {
			return a.dispatchMenuItem(ctx, chatID, userID, item)
		}
		return a.sendMainMenu(ctx, chatID)
	}
}

func (a *Adapter) HandleCallback(ctx context.Context, chatID int64, callbackID, data, userID string) error {
	_ = a.Out.AnswerCallback(ctx, callbackID, "")
	switch {
	case data == CBMenu || data == CBImgList:
		if data == CBMenu {
			return a.sendMainMenu(ctx, chatID)
		}
		return a.showCasesByTag(ctx, chatID, "image")
	case data == CBConfirm:
		return a.handleConfirm(ctx, chatID)
	case data == CBExit:
		return a.handleExit(ctx, chatID)
	case data == CBSkip:
		return a.handleSkip(ctx, chatID)
	case data == CBContinue:
		return a.handleContinue(ctx, chatID)
	case strings.HasPrefix(data, CBReplaceStart):
		id := sharedkernel.CaseID(strings.TrimPrefix(data, CBReplaceStart))
		return a.replaceAndStart(ctx, chatID, id, userID)
	case strings.HasPrefix(data, CBCasePreviewFolder):
		rest := strings.TrimPrefix(data, CBCasePreviewFolder)
		folderID, caseID, ok := strings.Cut(rest, ":")
		if !ok || folderID == "" || caseID == "" {
			return a.Out.SendText(ctx, chatID, "未知操作")
		}
		return a.showCasePreview(ctx, chatID, sharedkernel.CaseID(caseID), CBMenuBack+folderID)
	case strings.HasPrefix(data, CBCasePreview):
		id := sharedkernel.CaseID(strings.TrimPrefix(data, CBCasePreview))
		return a.showCasePreview(ctx, chatID, id, CBImgList)
	case strings.HasPrefix(data, CBCaseStart):
		id := sharedkernel.CaseID(strings.TrimPrefix(data, CBCaseStart))
		return a.startCase(ctx, chatID, id, userID)
	case strings.HasPrefix(data, CBMenuFolder):
		id := strings.TrimPrefix(data, CBMenuFolder)
		return a.showMenuFolder(ctx, chatID, id)
	case strings.HasPrefix(data, CBMenuBack):
		target := strings.TrimPrefix(data, CBMenuBack)
		if target == "root" {
			return a.sendMainMenu(ctx, chatID)
		}
		return a.showMenuFolder(ctx, chatID, target)
	default:
		return a.Out.SendText(ctx, chatID, "未知操作")
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

	chat := int64(n.ChatID)
	if n.Kind == "task_succeeded" && len(n.Outputs) > 0 {
		caption := fmt.Sprintf("✅ Case 完成\ntask=%s", n.TaskID)
		if err := a.Out.SendPhoto(ctx, chat, n.Outputs[0], caption); err != nil {
			return err
		}
		return a.Out.SendMenu(ctx, chat, "还要继续？点菜单「"+BtnImage+"」再选一个 Case。")
	}
	msg := fmt.Sprintf("任务 %s: %s", n.TaskID, n.Kind)
	if n.ErrorMsg != "" {
		msg += " — " + n.ErrorMsg
	}
	return a.Out.SendText(ctx, chat, msg)
}

func (a *Adapter) sendMainMenu(ctx context.Context, chatID int64) error {
	return a.Out.SendMenu(ctx, chatID, "欢迎使用 ComfyUI Bot（mock）\n请选择功能：")
}

func (a *Adapter) showImageCases(ctx context.Context, chatID int64) error {
	return a.showCasesByTag(ctx, chatID, "image")
}

func (a *Adapter) showCasesByTag(ctx context.Context, chatID int64, tag string) error {
	// ReplyKeyboard 只能随消息下发；进分区前先刷一次主菜单，避免用户仍停在旧键盘。
	if err := a.Out.SendMenu(ctx, chatID, "已进入分区："+tag); err != nil {
		return err
	}
	enabled := true
	cases, err := a.App.ListCases(ctx, catalogdomain.ListQuery{Tag: tag, Enabled: &enabled})
	if err != nil {
		return a.Out.SendText(ctx, chatID, "列出 Case 失败: "+err.Error())
	}
	if len(cases) == 0 {
		return a.Out.SendText(ctx, chatID, "暂无 Case（tag="+tag+"），请检查种子配置。")
	}
	var rows [][]InlineButton
	for _, c := range cases {
		rows = append(rows, []InlineButton{{
			Text: fmt.Sprintf("%s · ¥%.0f", c.Document.Name, c.Document.Price),
			Data: CBCasePreview + string(c.Document.ID),
		}})
	}
	rows = append(rows, []InlineButton{{Text: "« 返回菜单", Data: CBMenu}})
	title := tag + " Case（mock）\n点选查看预览："
	if tag == "image" {
		title = BtnImage + " Case（mock）\n点选查看预览："
	}
	return a.Out.SendInline(ctx, chatID, title, rows)
}

func (a *Adapter) dispatchMenuItem(ctx context.Context, chatID int64, userID string, item tgmenudomain.MenuNode) error {
	switch item.Kind {
	case tgmenudomain.KindFolder:
		return a.showMenuFolder(ctx, chatID, item.ID)
	case tgmenudomain.KindListCasesByTag:
		return a.showCasesByTag(ctx, chatID, item.Tag)
	case tgmenudomain.KindOpenCase:
		if len(item.CaseIDs) == 0 {
			return a.Out.SendMenu(ctx, chatID, "菜单配置无效：缺少 case")
		}
		return a.showCasePreview(ctx, chatID, sharedkernel.CaseID(item.CaseIDs[0]), CBImgList)
	case tgmenudomain.KindPlaceholder:
		msg := strings.TrimSpace(item.PlaceholderText)
		if msg == "" {
			msg = item.Label + "：暂未开放，请先体验「" + BtnImage + "」。"
		}
		return a.Out.SendMenu(ctx, chatID, msg)
	case tgmenudomain.KindReplyMedia:
		return a.sendReplyMedia(ctx, chatID, item)
	default:
		return a.Out.SendMenu(ctx, chatID, "未知菜单动作")
	}
}

func (a *Adapter) showMenuFolder(ctx context.Context, chatID int64, itemID string) error {
	tree := a.loadMenu(ctx)
	node, ok := findNodeByID(tree.Items, itemID)
	if !ok {
		return a.Out.SendText(ctx, chatID, "菜单项不存在")
	}

	text := strings.TrimSpace(node.IntroText)
	if text == "" {
		text = node.Label
	}

	var rows [][]InlineButton
	for _, caseID := range node.CaseIDs {
		c, err := a.App.GetCase(ctx, sharedkernel.CaseID(caseID))
		if err != nil {
			continue
		}
		doc := c.Document
		rows = append(rows, []InlineButton{{
			Text: fmt.Sprintf("%s · ¥%.0f", doc.Name, doc.Price),
			Data: CBCasePreviewFolder + itemID + ":" + string(doc.ID),
		}})
	}
	for _, child := range node.Children {
		if !child.Enabled {
			continue
		}
		if child.Kind == tgmenudomain.KindFolder {
			rows = append(rows, []InlineButton{{
				Text: "📁 " + child.Label,
				Data: CBMenuFolder + child.ID,
			}})
		}
	}

	backData := CBMenuBack + "root"
	if node.ParentID != "" {
		backData = CBMenuBack + node.ParentID
	}
	rows = append(rows, []InlineButton{{Text: "⬅️ 返回", Data: backData}})

	return a.Out.SendInline(ctx, chatID, text, rows)
}

func (a *Adapter) sendReplyMedia(ctx context.Context, chatID int64, item tgmenudomain.MenuNode) error {
	if item.Reply == nil {
		return a.Out.SendText(ctx, chatID, "菜单配置无效：缺少 reply")
	}
	text := strings.TrimSpace(item.Reply.Text)
	if text != "" {
		if err := a.Out.SendText(ctx, chatID, text); err != nil {
			return err
		}
	}
	okCount := 0
	for _, u := range item.Reply.Images {
		if err := a.Out.SendPhotoURL(ctx, chatID, u, ""); err != nil {
			slog.Error("tg reply_media photo failed", "err", err, "url", u, "chat_id", chatID)
			continue
		}
		okCount++
	}
	if text == "" && okCount == 0 && len(item.Reply.Images) > 0 {
		return a.Out.SendText(ctx, chatID, "图片发送失败，请稍后重试")
	}
	return nil
}

func (a *Adapter) showCasePreview(ctx context.Context, chatID int64, id sharedkernel.CaseID, backData string) error {
	c, err := a.App.GetCase(ctx, id)
	if err != nil {
		return a.Out.SendText(ctx, chatID, "Case 不存在: "+err.Error())
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

	if backData == "" {
		backData = CBImgList
	}
	rows := [][]InlineButton{
		{{Text: "▶ 开始 Case", Data: CBCaseStart + string(doc.ID)}},
		{{Text: "« 返回列表", Data: backData}},
	}
	return a.Out.SendInline(ctx, chatID, b.String(), rows)
}

func (a *Adapter) startCase(ctx context.Context, chatID int64, id sharedkernel.CaseID, userID string) error {
	view, err := a.App.StartCase(ctx, botapp.StartCaseCmd{
		ChatID: sharedkernel.ChatID(chatID),
		UserID: userID,
		CaseID: id,
	})
	if errors.Is(err, convdomain.ErrSessionLocked) {
		return a.showSessionConflict(ctx, chatID, id)
	}
	if err != nil {
		return a.Out.SendText(ctx, chatID, "无法开始: "+err.Error())
	}
	return a.renderSession(ctx, chatID, view)
}

func (a *Adapter) showSessionConflict(ctx context.Context, chatID int64, want sharedkernel.CaseID) error {
	cur, err := a.App.GetSession(ctx, sharedkernel.ChatID(chatID))
	if err != nil {
		return a.Out.SendText(ctx, chatID, "已有进行中的填表，但读取会话失败，请稍后再试。")
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
	var rows [][]InlineButton
	if same {
		msg = fmt.Sprintf(
			"你正在填写「%s」\n进度：%s\n\n要接着填，还是退出后重来？",
			curName, step,
		)
		rows = [][]InlineButton{
			{{Text: "继续当前 Case", Data: CBContinue}},
			{{Text: "退出当前 Case", Data: CBExit}},
		}
	} else {
		msg = fmt.Sprintf(
			"检测到未完成的 Case\n\n正在进行：%s\n进度：%s\n你刚想开始：%s\n\n请选择：继续刚才的，或废弃它并开始新选的。",
			curName, step, wantName,
		)
		rows = [][]InlineButton{
			{{Text: "继续当前 Case", Data: CBContinue}},
			{{Text: "退出当前 Case", Data: CBExit}},
			{{Text: "开始当前", Data: CBReplaceStart + string(want)}},
		}
	}
	return a.Out.SendInline(ctx, chatID, msg, rows)
}

func (a *Adapter) handleContinue(ctx context.Context, chatID int64) error {
	view, err := a.App.GetSession(ctx, sharedkernel.ChatID(chatID))
	if err != nil {
		return a.Out.SendMenu(ctx, chatID, "当前没有进行中的 Case，已回到菜单。")
	}
	return a.renderSession(ctx, chatID, view)
}

func (a *Adapter) replaceAndStart(ctx context.Context, chatID int64, id sharedkernel.CaseID, userID string) error {
	_ = a.App.ExitSession(ctx, sharedkernel.ChatID(chatID))
	return a.startCase(ctx, chatID, id, userID)
}

func (a *Adapter) submitText(ctx context.Context, chatID int64, text string) error {
	ft, err := a.currentFieldType(ctx, chatID)
	if err != nil {
		return a.Out.SendText(ctx, chatID, "提交失败: "+err.Error())
	}
	var draft convdomain.DraftValue
	switch ft {
	case "image":
		return a.Out.SendText(ctx, chatID, "当前需要一张图片，请发送 Photo 或图片文件。")
	case "number":
		n, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return a.Out.SendText(ctx, chatID, "请输入合法数字，例如 42")
		}
		draft = convdomain.DraftValue{Number: &n}
	case "boolean":
		b, err := strconv.ParseBool(text)
		if err != nil {
			return a.Out.SendText(ctx, chatID, "请输入 true 或 false")
		}
		draft = convdomain.DraftValue{Bool: &b}
	default:
		draft = convdomain.DraftValue{Text: &text}
	}
	view, err := a.App.SubmitInput(ctx, sharedkernel.ChatID(chatID), draft)
	if err != nil {
		return a.Out.SendText(ctx, chatID, "提交失败: "+err.Error())
	}
	return a.renderSession(ctx, chatID, view)
}

func (a *Adapter) currentFieldType(ctx context.Context, chatID int64) (string, error) {
	view, err := a.App.GetSession(ctx, sharedkernel.ChatID(chatID))
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

func (a *Adapter) handleSkip(ctx context.Context, chatID int64) error {
	view, err := a.App.SkipInput(ctx, sharedkernel.ChatID(chatID))
	if err != nil {
		return a.Out.SendText(ctx, chatID, "跳过失败: "+err.Error())
	}
	return a.renderSession(ctx, chatID, view)
}

func (a *Adapter) handleExit(ctx context.Context, chatID int64) error {
	if err := a.App.ExitSession(ctx, sharedkernel.ChatID(chatID)); err != nil {
		return a.Out.SendText(ctx, chatID, "退出失败: "+err.Error())
	}
	return a.Out.SendMenu(ctx, chatID, "已退出当前 Case，可以重新选择。")
}

func (a *Adapter) handleConfirm(ctx context.Context, chatID int64) error {
	// Memory queue is synchronous: ConfirmRun may finish the whole pipeline
	// (including notify/photo) before returning. Acknowledge first so order is natural.
	if err := a.Out.SendText(ctx, chatID, "⏳ 已提交，正在生成…"); err != nil {
		return err
	}
	res, err := a.App.ConfirmRun(ctx, botapp.ConfirmRunCmd{ChatID: sharedkernel.ChatID(chatID)})
	if err != nil {
		return a.Out.SendText(ctx, chatID, "确认失败: "+err.Error())
	}
	tasks, err := a.App.ListMyTasks(ctx, sharedkernel.ChatID(chatID), 20)
	if err == nil {
		for _, t := range tasks {
			if t.ID != res.TaskID {
				continue
			}
			switch t.Status {
			case sharedkernel.TaskSucceeded, sharedkernel.TaskFailed, sharedkernel.TaskCancelled:
				// Result notify already sent on the sync path; avoid a late "queued" message.
				return nil
			}
			break
		}
	}
	return a.Out.SendText(ctx, chatID, fmt.Sprintf("已排队\ntask=%s\n完成后会把图片发回来。", res.TaskID))
}

func (a *Adapter) renderSession(ctx context.Context, chatID int64, view *botapp.SessionView) error {
	if view.Status == convdomain.StatusConfirming {
		return a.Out.SendInline(ctx, chatID, "输入完成，确认执行？", [][]InlineButton{
			{{Text: "✅ 确认生成", Data: CBConfirm}, {Text: "✕ 退出", Data: CBExit}},
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
				rows := [][]InlineButton{{{Text: "✕ 退出", Data: CBExit}}}
				if !in.Required || in.SkipAllowed {
					rows = [][]InlineButton{
						{{Text: "跳过", Data: CBSkip}, {Text: "✕ 退出", Data: CBExit}},
					}
				}
				return a.Out.SendInline(ctx, chatID, fmt.Sprintf("请输入「%s」\n%s", key, hint), rows)
			}
		}
	}
	return a.Out.SendInline(ctx, chatID, "请输入「"+key+"」", [][]InlineButton{
		{{Text: "✕ 退出", Data: CBExit}},
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
	case "/start", "/menu", "/help", "/cases", "/skip", "/exit", "/confirm":
		return true
	}
	if strings.HasPrefix(text, "/start_case ") {
		return true
	}
	doc := a.loadMenu(ctx)
	if _, ok := FindEnabledRootByLabel(doc, text); ok {
		return true
	}
	// Fallback legacy constants if menu load somehow omitted them.
	switch text {
	case BtnImage, BtnVideo, BtnRecharge, BtnCheckIn, BtnProfile, BtnHelp:
		return true
	}
	return false
}
