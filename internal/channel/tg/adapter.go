package tg

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	catalogdomain "github.com/mr9esx/comfyui_tgbot/internal/catalog/domain"
	convdomain "github.com/mr9esx/comfyui_tgbot/internal/conversation/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/packaging/botapp"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

type Adapter struct {
	App      *botapp.Facade
	Out      Messenger
	mu       sync.Mutex
	notified map[string]struct{}
}

func New(app *botapp.Facade, out Messenger) *Adapter {
	return &Adapter{App: app, Out: out, notified: map[string]struct{}{}}
}

func (a *Adapter) HandleText(ctx context.Context, chatID int64, text string) error {
	text = strings.TrimSpace(text)

	// Active session: treat free text as input (unless menu command).
	if !isMenuCommand(text) {
		if _, err := a.App.GetSession(ctx, sharedkernel.ChatID(chatID)); err == nil {
			return a.submitText(ctx, chatID, text)
		}
	}

	switch text {
	case "/start", "/menu", CBMenu:
		return a.sendMainMenu(ctx, chatID)
	case BtnImage, "/cases":
		return a.showImageCases(ctx, chatID)
	case BtnHelp, "/help":
		return a.Out.SendMenu(ctx, chatID, "帮助：点「图片」选 Case → 预览 → 开始 → 输入 prompt → 确认 → 等待出图。")
	case BtnVideo, BtnVideoUndress, BtnHotTemplates, BtnRecharge, BtnCheckIn, BtnProfile, BtnInvite:
		return a.Out.SendMenu(ctx, chatID, text+"：本期 mock 未开放，请先体验「🔞 图片」。")
	case "/skip":
		return a.handleSkip(ctx, chatID)
	case "/exit":
		return a.handleExit(ctx, chatID)
	case "/confirm":
		return a.handleConfirm(ctx, chatID)
	default:
		if strings.HasPrefix(text, "/start_case ") {
			id := strings.TrimSpace(strings.TrimPrefix(text, "/start_case "))
			return a.startCase(ctx, chatID, sharedkernel.CaseID(id))
		}
		return a.sendMainMenu(ctx, chatID)
	}
}

func (a *Adapter) HandleCallback(ctx context.Context, chatID int64, callbackID, data string) error {
	_ = a.Out.AnswerCallback(ctx, callbackID, "")
	switch {
	case data == CBMenu || data == CBImgList:
		if data == CBMenu {
			return a.sendMainMenu(ctx, chatID)
		}
		return a.showImageCases(ctx, chatID)
	case data == CBConfirm:
		return a.handleConfirm(ctx, chatID)
	case data == CBExit:
		return a.handleExit(ctx, chatID)
	case data == CBSkip:
		return a.handleSkip(ctx, chatID)
	case strings.HasPrefix(data, CBCasePreview):
		id := sharedkernel.CaseID(strings.TrimPrefix(data, CBCasePreview))
		return a.showCasePreview(ctx, chatID, id)
	case strings.HasPrefix(data, CBCaseStart):
		id := sharedkernel.CaseID(strings.TrimPrefix(data, CBCaseStart))
		return a.startCase(ctx, chatID, id)
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
		return a.Out.SendMenu(ctx, chat, "还要继续？点菜单「🔞 图片」再选一个 Case。")
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
	enabled := true
	cases, err := a.App.ListCases(ctx, catalogdomain.ListQuery{Tag: "image", Enabled: &enabled})
	if err != nil {
		return a.Out.SendText(ctx, chatID, "列出 Case 失败: "+err.Error())
	}
	if len(cases) == 0 {
		return a.Out.SendText(ctx, chatID, "暂无图片 Case，请检查种子配置。")
	}
	var rows [][]InlineButton
	for _, c := range cases {
		rows = append(rows, []InlineButton{{
			Text: fmt.Sprintf("%s · ¥%.0f", c.Document.Name, c.Document.Price),
			Data: CBCasePreview + string(c.Document.ID),
		}})
	}
	rows = append(rows, []InlineButton{{Text: "« 返回菜单", Data: CBMenu}})
	return a.Out.SendInline(ctx, chatID, "🔞 图片 Case（mock）\n点选查看预览：", rows)
}

func (a *Adapter) showCasePreview(ctx context.Context, chatID int64, id sharedkernel.CaseID) error {
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

	rows := [][]InlineButton{
		{{Text: "▶ 开始 Case", Data: CBCaseStart + string(doc.ID)}},
		{{Text: "« 返回列表", Data: CBImgList}},
	}
	return a.Out.SendInline(ctx, chatID, b.String(), rows)
}

func (a *Adapter) startCase(ctx context.Context, chatID int64, id sharedkernel.CaseID) error {
	view, err := a.App.StartCase(ctx, botapp.StartCaseCmd{
		ChatID: sharedkernel.ChatID(chatID),
		CaseID: id,
	})
	if errors.Is(err, convdomain.ErrSessionLocked) {
		return a.Out.SendInline(ctx, chatID, "当前有进行中的填表会话。", [][]InlineButton{
			{{Text: "退出当前", Data: CBExit}},
		})
	}
	if err != nil {
		return a.Out.SendText(ctx, chatID, "无法开始: "+err.Error())
	}
	return a.renderSession(ctx, chatID, view)
}

func (a *Adapter) submitText(ctx context.Context, chatID int64, text string) error {
	view, err := a.App.SubmitInput(ctx, sharedkernel.ChatID(chatID), convdomain.DraftValue{Text: &text})
	if err != nil {
		return a.Out.SendText(ctx, chatID, "提交失败: "+err.Error())
	}
	return a.renderSession(ctx, chatID, view)
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
	return a.Out.SendMenu(ctx, chatID, "已退出填表。")
}

func (a *Adapter) handleConfirm(ctx context.Context, chatID int64) error {
	res, err := a.App.ConfirmRun(ctx, botapp.ConfirmRunCmd{ChatID: sharedkernel.ChatID(chatID)})
	if err != nil {
		return a.Out.SendText(ctx, chatID, "确认失败: "+err.Error())
	}
	return a.Out.SendText(ctx, chatID, fmt.Sprintf("⏳ 已排队生成\ntask=%s\n完成后会把图片发回来。", res.TaskID))
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

func isMenuCommand(text string) bool {
	switch text {
	case "/start", "/menu", "/help", "/cases", "/skip", "/exit", "/confirm",
		BtnImage, BtnVideo, BtnVideoUndress, BtnHotTemplates, BtnRecharge, BtnCheckIn, BtnProfile, BtnInvite, BtnHelp:
		return true
	}
	return strings.HasPrefix(text, "/start_case ")
}
