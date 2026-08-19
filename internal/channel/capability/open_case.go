package capability

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	catalogdomain "github.com/mr9esx/comfyui_tgbot/internal/catalog/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/channel/protocol"
	convdomain "github.com/mr9esx/comfyui_tgbot/internal/conversation/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/packaging/botapp"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

// CaseService is the subset of botapp.Facade used by the open_case capability.
type CaseService interface {
	StartCase(ctx context.Context, cmd botapp.StartCaseCmd) (*botapp.SessionView, error)
	GetCase(ctx context.Context, id sharedkernel.CaseID) (*catalogdomain.Case, error)
	GetSession(ctx context.Context, chatID sharedkernel.ChatID) (*botapp.SessionView, error)
	SkipInput(ctx context.Context, chatID sharedkernel.ChatID) (*botapp.SessionView, error)
	SubmitInput(ctx context.Context, chatID sharedkernel.ChatID, v convdomain.DraftValue) (*botapp.SessionView, error)
	ExitSession(ctx context.Context, chatID sharedkernel.ChatID) error
	ConfirmRun(ctx context.Context, cmd botapp.ConfirmRunCmd) (*botapp.ConfirmRunResult, error)
}

// OpenCase is the built-in capability wrapping the existing Case workflow.
type OpenCase struct {
	App CaseService
}

func (OpenCase) ID() string          { return "open_case" }
func (OpenCase) DisplayName() string { return "打开工作流" }

// AdminParamsSchema exposes only the admin-configurable list entry params.
func (OpenCase) AdminParamsSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"case_ids": {
				"type": "array",
				"items": { "type": "string" },
				"x-admin": { "widget": "workflow_picker" }
			}
		}
	}`)
}

func (OpenCase) ParamsSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"case_ids": { "type": "array", "items": { "type": "string" } },
			"case_id":  { "type": "string" },
			"step":     { "type": "string", "enum": ["list", "preview", "start", "confirm", "exit", "skip", "text", "media", "check"] },
			"text":     { "type": "string" },
			"blob":     { "type": "object", "properties": { "key": { "type": "string" }, "mime": { "type": "string" } } }
		}
	}`)
}

func (OpenCase) Render(channelID string, override map[string]any) (protocol.RenderDecl, error) {
	return MergeRender(protocol.RenderDecl{
		Entry:  "root",
		Config: map[string]any{"columns": 2},
	}, override), nil
}

func (o OpenCase) Invoke(ctx context.Context, acct protocol.AccountCtx, nav protocol.Nav, chatID sharedkernel.ChatID, params map[string]any) (protocol.Result, error) {
	step, _ := params["step"].(string)
	if step == "" {
		step = "list"
	}
	switch step {
	case "check":
		if o.App == nil {
			return protocol.Result{Text: "none"}, nil
		}
		if _, err := o.App.GetSession(ctx, chatID); err != nil {
			return protocol.Result{Text: "none"}, nil
		}
		return protocol.Result{Text: "active"}, nil
	case "list":
		return o.list(ctx, params)
	case "preview":
		return o.preview(ctx, params)
	case "start":
		return o.start(ctx, acct, chatID, params)
	case "skip":
		return o.skip(ctx, chatID)
	case "text":
		return o.submitText(ctx, chatID, params)
	case "media":
		return o.submitMedia(ctx, chatID, params)
	case "confirm":
		return o.confirm(ctx, chatID)
	case "exit":
		if o.App == nil {
			return protocol.Result{}, fmt.Errorf("open_case: app not configured")
		}
		if err := o.App.ExitSession(ctx, chatID); err != nil {
			return protocol.Result{}, err
		}
		return protocol.Result{Text: "已退出当前 Case，可以重新选择。"}, nil
	default:
		return protocol.Result{}, fmt.Errorf("open_case: unknown step %q", step)
	}
}

func (o OpenCase) submitText(ctx context.Context, chatID sharedkernel.ChatID, params map[string]any) (protocol.Result, error) {
	if o.App == nil {
		return protocol.Result{}, fmt.Errorf("open_case: app not configured")
	}
	text, _ := params["text"].(string)
	ft, err := o.currentFieldType(ctx, chatID)
	if err != nil {
		return protocol.Result{}, err
	}
	var draft convdomain.DraftValue
	switch ft {
	case "image":
		return protocol.Result{Text: "当前需要一张图片，请发送图片。"}, nil
	case "number":
		n, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return protocol.Result{Text: "请输入合法数字，例如 42"}, nil
		}
		draft = convdomain.DraftValue{Number: &n}
	case "boolean":
		b, err := strconv.ParseBool(text)
		if err != nil {
			return protocol.Result{Text: "请输入 true 或 false"}, nil
		}
		draft = convdomain.DraftValue{Bool: &b}
	default:
		draft = convdomain.DraftValue{Text: &text}
	}
	view, err := o.App.SubmitInput(ctx, chatID, draft)
	if err != nil {
		return protocol.Result{}, err
	}
	return renderSession(o, view)
}

func (o OpenCase) submitMedia(ctx context.Context, chatID sharedkernel.ChatID, params map[string]any) (protocol.Result, error) {
	if o.App == nil {
		return protocol.Result{}, fmt.Errorf("open_case: app not configured")
	}
	ft, err := o.currentFieldType(ctx, chatID)
	if err != nil {
		return protocol.Result{}, err
	}
	if ft != "image" {
		return protocol.Result{Text: "当前不需要图片，请按提示输入。"}, nil
	}
	blobMap, ok := params["blob"].(map[string]any)
	if !ok {
		return protocol.Result{}, fmt.Errorf("open_case: blob required for media")
	}
	key, _ := blobMap["key"].(string)
	mime, _ := blobMap["mime"].(string)
	ref := sharedkernel.BlobRef{Key: key, MIME: mime}
	view, err := o.App.SubmitInput(ctx, chatID, convdomain.DraftValue{Blob: &ref})
	if err != nil {
		return protocol.Result{}, err
	}
	return renderSession(o, view)
}

func (o OpenCase) currentFieldType(ctx context.Context, chatID sharedkernel.ChatID) (string, error) {
	if o.App == nil {
		return "", fmt.Errorf("open_case: app not configured")
	}
	view, err := o.App.GetSession(ctx, chatID)
	if err != nil {
		return "", err
	}
	key := "(done)"
	if view.Index >= 0 && view.Index < len(view.Keys) {
		key = view.Keys[view.Index]
	}
	if key == "(done)" {
		return "", fmt.Errorf("open_case: no current input field")
	}
	c, err := o.App.GetCase(ctx, view.CaseID)
	if err != nil {
		return "", err
	}
	for _, in := range c.Document.Inputs {
		if in.Key == key {
			return in.Type, nil
		}
	}
	return "", fmt.Errorf("open_case: unknown field %q", key)
}

func (o OpenCase) list(ctx context.Context, params map[string]any) (protocol.Result, error) {
	if o.App == nil {
		return protocol.Result{}, fmt.Errorf("open_case: app not configured")
	}
	ids, _ := params["case_ids"].([]any)
	if len(ids) == 0 {
		return protocol.Result{}, fmt.Errorf("open_case: case_ids required")
	}
	var options []protocol.Option
	for _, raw := range ids {
		id, ok := raw.(string)
		if !ok || id == "" {
			continue
		}
		c, err := o.App.GetCase(ctx, sharedkernel.CaseID(id))
		if err != nil {
			continue
		}
		options = append(options, protocol.Option{
			Label: fmt.Sprintf("%s · ¥%.0f", c.Document.Name, c.Document.Price),
			Value: map[string]any{"step": "preview", "case_id": id},
		})
	}
	return protocol.Result{Text: "请选择模板：", Options: options}, nil
}

func (o OpenCase) preview(ctx context.Context, params map[string]any) (protocol.Result, error) {
	if o.App == nil {
		return protocol.Result{}, fmt.Errorf("open_case: app not configured")
	}
	caseID, _ := params["case_id"].(string)
	if caseID == "" {
		return protocol.Result{}, fmt.Errorf("open_case: case_id required for preview")
	}
	c, err := o.App.GetCase(ctx, sharedkernel.CaseID(caseID))
	if err != nil {
		return protocol.Result{}, err
	}
	doc := c.Document
	var b strings.Builder
	fmt.Fprintf(&b, "📎 %s\n", doc.Name)
	if doc.Description != "" {
		fmt.Fprintf(&b, "%s\n", doc.Description)
	}
	fmt.Fprintf(&b, "价格：%.0f\n", doc.Price)
	b.WriteString("\n预览说明：")
	if doc.Preview != "" {
		b.WriteString(doc.Preview)
	} else {
		b.WriteString("（mock）确认后将返回一张示例图")
	}
	return protocol.Result{
		Text: b.String(),
		Options: []protocol.Option{
			{Label: "▶ 开始 Case", Value: map[string]any{"step": "start", "case_id": caseID}},
			{Label: "« 返回列表", Value: map[string]any{"step": "list", "case_ids": params["case_ids"]}},
		},
	}, nil
}

func (o OpenCase) start(ctx context.Context, acct protocol.AccountCtx, chatID sharedkernel.ChatID, params map[string]any) (protocol.Result, error) {
	if o.App == nil {
		return protocol.Result{}, fmt.Errorf("open_case: app not configured")
	}
	caseID, _ := params["case_id"].(string)
	if caseID == "" {
		return protocol.Result{}, fmt.Errorf("open_case: case_id required for start")
	}
	view, err := o.App.StartCase(ctx, botapp.StartCaseCmd{
		ChatID: chatID, UserID: acct.InternalUserID, CaseID: sharedkernel.CaseID(caseID),
	})
	if err != nil {
		return protocol.Result{}, err
	}
	return renderSession(o, view)
}

func (o OpenCase) skip(ctx context.Context, chatID sharedkernel.ChatID) (protocol.Result, error) {
	if o.App == nil {
		return protocol.Result{}, fmt.Errorf("open_case: app not configured")
	}
	view, err := o.App.SkipInput(ctx, chatID)
	if err != nil {
		return protocol.Result{}, err
	}
	return renderSession(o, view)
}

func (o OpenCase) confirm(ctx context.Context, chatID sharedkernel.ChatID) (protocol.Result, error) {
	if o.App == nil {
		return protocol.Result{}, fmt.Errorf("open_case: app not configured")
	}
	res, err := o.App.ConfirmRun(ctx, botapp.ConfirmRunCmd{ChatID: chatID})
	if err != nil {
		return protocol.Result{}, err
	}
	return protocol.Result{Text: fmt.Sprintf("已提交，正在生成… task=%s", res.TaskID)}, nil
}

func renderSession(o OpenCase, view *botapp.SessionView) (protocol.Result, error) {
	if view.Status == convdomain.StatusConfirming {
		return protocol.Result{
			Text: "输入完成，确认执行？",
			Options: []protocol.Option{
				{Label: "✅ 确认生成", Value: map[string]any{"step": "confirm"}},
				{Label: "✕ 退出", Value: map[string]any{"step": "exit"}},
			},
		}, nil
	}
	key := "(done)"
	if view.Index >= 0 && view.Index < len(view.Keys) {
		key = view.Keys[view.Index]
	}
	options := []protocol.Option{{Label: "✕ 退出", Value: map[string]any{"step": "exit"}}}
	options = append([]protocol.Option{{Label: "跳过", Value: map[string]any{"step": "skip"}}}, options...)
	return protocol.Result{Text: fmt.Sprintf("请输入「%s」", key), Options: options}, nil
}
