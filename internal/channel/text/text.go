// Package text defines the platform-level Telegram copy templates that can be
// customized by admins. Each template supports {{variable}} interpolation so
// dynamic values (e.g. task ids) can be injected at render time.
package text

import (
	"context"
	"regexp"
	"strings"
)

// Template keys. These are the stable identifiers stored in the database and
// exposed through the admin API.
const (
	KeyWelcome               = "welcome"
	KeyHelp                  = "help"
	KeySelectTemplate        = "select_template"
	KeyWorkflowDone          = "workflow_done"
	KeyWorkflowDoneFollowp   = "workflow_done_followup"
	KeySubmitStarted         = "submit_started"
	KeyConfirmRun            = "confirm_run"
	KeyInputPrompt           = "input_prompt"
	KeyExitDone              = "exit_done"
	KeyUnfinishedSession     = "unfinished_session"
	KeyMenuActionPlaceholder = "menu_action_placeholder"
	KeyMenuUpdated           = "menu_updated"
	KeyTaskFailed            = "task_failed"
	KeyTaskCancelled         = "task_cancelled"
	KeyPreviewHintLabel      = "preview_hint_label"
	KeyPreviewMockHint       = "preview_mock_hint"
	KeyButtonStartCase       = "button_start_case"
	KeyInputInvalidNumber    = "input_invalid_number"
	KeyInputInvalidBoolean   = "input_invalid_boolean"
	KeyButtonSkip            = "button_skip"
	KeyButtonConfirmRun      = "button_confirm_run"
	KeyButtonExit            = "button_exit"
	KeySessionTerminated     = "session_terminated"
	KeyAccessDenied          = "access_denied"
)

// Scenario groups used by the admin configuration page. They describe where a
// template is edited; storage and rendering always use the template key.
const (
	GroupWorkflow      = "workflow"
	GroupNotifications = "notifications"
	GroupPlatform      = "platform"
	GroupCommands      = "commands"
)

// Spec is the admin-facing description of a single configurable copy template.
type Spec struct {
	Key         string
	Group       string
	Description string
	Default     string
	Variables   []string
}

// Default returns the built-in template for a key.
func Default(key string) string {
	return defaultOf(key)
}

var defaults = map[string]string{
	KeyWelcome:               "欢迎使用 ComfyUI Bot\n请选择功能：",
	KeyHelp:                  "帮助：点下方按钮选功能。菜单更新后发 /menu 刷新。",
	KeySelectTemplate:        "请选择模板：",
	KeyWorkflowDone:          "✅ 工作流完成\ntask={{ task_id }}",
	KeyWorkflowDoneFollowp:   "还要继续？点菜单再选一个工作流。",
	KeySubmitStarted:         "已提交，正在生成… task={{ task_id }}",
	KeyConfirmRun:            "输入完成，确认执行？",
	KeyInputPrompt:           "填写参数 {{ progress }}：请输入「{{ key }}」",
	KeyExitDone:              "已退出当前 Case，可以重新选择。",
	KeyUnfinishedSession:     "你有一个未完成的工作流，先退出再重新开始。",
	KeyMenuActionPlaceholder: "暂未开放",
	KeyMenuUpdated:           "菜单已更新，请使用下方新按钮。",
	KeyTaskFailed:            "❌ 任务执行失败\ntask={{ task_id }}\n状态：{{ status }}\n{{ error_msg }}",
	KeyTaskCancelled:         "任务已取消\ntask={{ task_id }}",
	KeyPreviewHintLabel:      "预览说明：",
	KeyPreviewMockHint:       "（mock）确认后将返回一张示例图",
	KeyButtonStartCase:       "▶ 开始 Case",
	KeyInputInvalidNumber:    "请输入合法数字，例如 42",
	KeyInputInvalidBoolean:   "请输入 true 或 false",
	KeyButtonSkip:            "跳过",
	KeyButtonConfirmRun:      "✅ 确认生成",
	KeyButtonExit:            "✕ 退出",
	KeySessionTerminated:     "该工作流已被管理员删除，当前会话已结束。",
	KeyAccessDenied:          "当前账号没有使用权限，请联系管理员。",
}

var variableOf = map[string][]string{
	KeyWorkflowDone:      {"task_id"},
	KeySubmitStarted:     {"task_id"},
	KeyInputPrompt:       {"key", "case_name", "progress"},
	KeyTaskFailed:        {"task_id", "status", "error_msg"},
	KeyTaskCancelled:     {"task_id", "status"},
	KeySessionTerminated: {"error_msg"},
	KeyPreviewMockHint:   {"case_name"},
}

// defaultOf looks up the built-in default template. Unknown keys fall back to
// the empty string so rendering never panics on a misconfigured key.
func defaultOf(key string) string {
	return defaults[key]
}

// Specs returns the ordered list of all configurable copy templates.
func Specs() []Spec {
	order := []string{
		KeyPreviewHintLabel, KeyPreviewMockHint, KeyButtonStartCase,
		KeyInputPrompt, KeyInputInvalidNumber, KeyInputInvalidBoolean,
		KeyButtonSkip, KeyButtonExit, KeyConfirmRun, KeyButtonConfirmRun,
		KeySubmitStarted, KeyWorkflowDone, KeyWorkflowDoneFollowp,
		KeyTaskFailed, KeyTaskCancelled, KeySessionTerminated,
		KeyWelcome, KeySelectTemplate, KeyExitDone, KeyUnfinishedSession,
		KeyMenuActionPlaceholder, KeyMenuUpdated, KeyHelp,
		KeyAccessDenied,
	}
	out := make([]Spec, 0, len(order))
	for _, k := range order {
		out = append(out, Spec{
			Key:         k,
			Group:       groupOf(k),
			Default:     defaults[k],
			Variables:   variableOf[k],
			Description: descriptionOf[k],
		})
	}
	return out
}

var descriptionOf = map[string]string{
	KeyWelcome:               "进入机器人时显示的主菜单标题",
	KeyHelp:                  "/help 帮助文案",
	KeySelectTemplate:        "选择工作流模板时的提示",
	KeyWorkflowDone:          "工作流执行成功后的摘要，支持 {{ task_id }}",
	KeyWorkflowDoneFollowp:   "工作流执行成功后的引导文案",
	KeySubmitStarted:         "提交任务后的确认文案，支持 {{ task_id }}",
	KeyConfirmRun:            "确认执行前的确认文案",
	KeyInputPrompt:           "等待输入自定义字段时的提示，支持 {{ key }}、{{ case_name }}、{{ progress }}",
	KeyExitDone:              "退出当前会话后的提示",
	KeyUnfinishedSession:     "存在未完成会话时的提示",
	KeyMenuActionPlaceholder: "菜单里尚未开放的功能提示",
	KeyMenuUpdated:           "检测到旧按钮时的提示",
	KeyTaskFailed:            "任务执行失败时的通知，支持 {{ task_id }}、{{ status }}、{{ error_msg }}",
	KeyTaskCancelled:         "任务被取消时的通知，支持 {{ task_id }}、{{ status }}",
	KeyPreviewHintLabel:      "工作流预览说明标题",
	KeyPreviewMockHint:       "Case 没有预览媒体时的提示",
	KeyButtonStartCase:       "工作流预览的开始按钮",
	KeyInputInvalidNumber:    "数字输入格式错误提示",
	KeyInputInvalidBoolean:   "布尔输入格式错误提示",
	KeyButtonSkip:            "工作流输入阶段的跳过按钮",
	KeyButtonConfirmRun:      "工作流确认阶段的提交按钮",
	KeyButtonExit:            "工作流输入和确认阶段的退出按钮",
	KeySessionTerminated:     "工作流删除导致会话终止时的通知",
}

func groupOf(key string) string {
	switch key {
	case KeyPreviewHintLabel, KeyPreviewMockHint, KeyButtonStartCase,
		KeyInputPrompt, KeyInputInvalidNumber, KeyInputInvalidBoolean,
		KeyButtonSkip, KeyButtonExit, KeyConfirmRun, KeyButtonConfirmRun,
		KeySubmitStarted:
		return GroupWorkflow
	case KeyWorkflowDone, KeyWorkflowDoneFollowp, KeyTaskFailed,
		KeyTaskCancelled, KeySessionTerminated:
		return GroupNotifications
	case KeyHelp:
		return GroupCommands
	default:
		return GroupPlatform
	}
}

var varPattern = regexp.MustCompile(`\{\{\s*([A-Za-z_][A-Za-z0-9_]*)\s*\}\}`)

// Render interpolates {{var}} placeholders in a template with the provided
// values. Unknown variables are left untouched so a misprint never silently
// drops content.
func Render(template string, vars map[string]string) string {
	if template == "" || len(vars) == 0 {
		return template
	}
	return varPattern.ReplaceAllStringFunc(template, func(m string) string {
		g := varPattern.FindStringSubmatch(m)
		if len(g) == 2 {
			if v, ok := vars[strings.TrimSpace(g[1])]; ok {
				return v
			}
		}
		return m
	})
}

// Renderer resolves a template by key for a channel, applying the platform
// default and finally the built-in default when no override has been
// configured for the channel.
type Renderer interface {
	Render(ctx context.Context, channelID, key string, vars map[string]string) string
}

// StaticRenderer serves fixed defaults. It is the safe fallback used when no
// store is wired (e.g. unit tests).
type StaticRenderer struct{}

// Render returns the default template for key, interpolating vars.
func (StaticRenderer) Render(_ context.Context, _ string, key string, vars map[string]string) string {
	return Render(defaultOf(key), vars)
}
