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

// Messenger abstracts Telegram sends so unit tests need no Bot token.
type Messenger interface {
	SendText(ctx context.Context, chatID int64, text string) error
	SendPhoto(ctx context.Context, chatID int64, blob sharedkernel.BlobRef, caption string) error
}

type Adapter struct {
	App      *botapp.Facade
	Out      Messenger
	mu       sync.Mutex
	notified map[string]struct{}
}

func New(app *botapp.Facade, out Messenger) *Adapter {
	return &Adapter{App: app, Out: out, notified: map[string]struct{}{}}
}

// HandleText routes simple slash commands / plain input for Phase1.
func (a *Adapter) HandleText(ctx context.Context, chatID int64, text string) error {
	text = strings.TrimSpace(text)
	switch {
	case text == "/start" || text == "/menu":
		return a.Out.SendText(ctx, chatID, "菜单：发送 /cases 查看 Case；/start_case <id> 开始；填表中可 /skip /exit /confirm")
	case text == "/cases":
		enabled := true
		cases, err := a.App.ListCases(ctx, catalogdomain.ListQuery{Enabled: &enabled})
		if err != nil {
			return a.Out.SendText(ctx, chatID, "列出 Case 失败: "+err.Error())
		}
		if len(cases) == 0 {
			return a.Out.SendText(ctx, chatID, "暂无 Case")
		}
		var b strings.Builder
		b.WriteString("Cases:\n")
		for _, c := range cases {
			fmt.Fprintf(&b, "- %s (%s) price=%.2f\n", c.Document.ID, c.Document.Name, c.Document.Price)
		}
		return a.Out.SendText(ctx, chatID, b.String())
	case strings.HasPrefix(text, "/start_case "):
		id := strings.TrimSpace(strings.TrimPrefix(text, "/start_case "))
		view, err := a.App.StartCase(ctx, botapp.StartCaseCmd{
			ChatID: sharedkernel.ChatID(chatID),
			CaseID: sharedkernel.CaseID(id),
		})
		if errors.Is(err, convdomain.ErrSessionLocked) {
			return a.Out.SendText(ctx, chatID, "当前有进行中的填表会话。发送 /exit 退出，或继续输入。")
		}
		if err != nil {
			return a.Out.SendText(ctx, chatID, "无法开始: "+err.Error())
		}
		return a.Out.SendText(ctx, chatID, fmt.Sprintf("已开始 %s，请输入字段 %s", view.CaseID, currentKey(view)))
	case text == "/skip":
		view, err := a.App.SkipInput(ctx, sharedkernel.ChatID(chatID))
		if err != nil {
			return a.Out.SendText(ctx, chatID, "跳过失败: "+err.Error())
		}
		return a.renderSession(ctx, chatID, view)
	case text == "/exit":
		if err := a.App.ExitSession(ctx, sharedkernel.ChatID(chatID)); err != nil {
			return a.Out.SendText(ctx, chatID, "退出失败: "+err.Error())
		}
		return a.Out.SendText(ctx, chatID, "已退出填表。")
	case text == "/confirm":
		res, err := a.App.ConfirmRun(ctx, botapp.ConfirmRunCmd{ChatID: sharedkernel.ChatID(chatID)})
		if err != nil {
			return a.Out.SendText(ctx, chatID, "确认失败: "+err.Error())
		}
		return a.Out.SendText(ctx, chatID, fmt.Sprintf("已排队 task=%s status=%s", res.TaskID, res.Status))
	default:
		_, err := a.App.GetSession(ctx, sharedkernel.ChatID(chatID))
		if err != nil {
			return a.Out.SendText(ctx, chatID, "未知命令。试试 /menu")
		}
		val := text
		view, err := a.App.SubmitInput(ctx, sharedkernel.ChatID(chatID), convdomain.DraftValue{Text: &val})
		if err != nil {
			return a.Out.SendText(ctx, chatID, "提交失败: "+err.Error())
		}
		return a.renderSession(ctx, chatID, view)
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
		caption := fmt.Sprintf("任务 %s 完成", n.TaskID)
		return a.Out.SendPhoto(ctx, chat, n.Outputs[0], caption)
	}
	msg := fmt.Sprintf("任务 %s: %s", n.TaskID, n.Kind)
	if n.ErrorMsg != "" {
		msg += " — " + n.ErrorMsg
	}
	return a.Out.SendText(ctx, chat, msg)
}

func (a *Adapter) renderSession(ctx context.Context, chatID int64, view *botapp.SessionView) error {
	if view.Status == convdomain.StatusConfirming {
		return a.Out.SendText(ctx, chatID, "输入完成，发送 /confirm 执行，或 /exit 取消。")
	}
	return a.Out.SendText(ctx, chatID, "请输入字段: "+currentKey(view))
}

func currentKey(view *botapp.SessionView) string {
	if view.Index >= 0 && view.Index < len(view.Keys) {
		return view.Keys[view.Index]
	}
	return "(done)"
}
