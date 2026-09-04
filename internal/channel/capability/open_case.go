package capability

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	catalogdomain "github.com/mr9esx/comfyui_tgbot/internal/catalog/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/channel/protocol"
	texttpl "github.com/mr9esx/comfyui_tgbot/internal/channel/text"
	convdomain "github.com/mr9esx/comfyui_tgbot/internal/conversation/domain"
	identitydomain "github.com/mr9esx/comfyui_tgbot/internal/identity/domain"
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
	// Texts resolves configurable copy templates; nil falls back to built-ins.
	Texts texttpl.Renderer
	// Users authorizes the channel account before task actions.
	Users identitydomain.Repository
}

// renderText resolves a configurable copy template for a channel (with
// defaults) and interpolates the supplied variables.
func (o OpenCase) renderText(ctx context.Context, channelID, key string, vars map[string]string) string {
	if o.Texts == nil {
		return texttpl.Render(texttpl.Default(key), vars)
	}
	return o.Texts.Render(ctx, channelID, key, vars)
}

func (OpenCase) ID() string          { return "open_case" }
func (OpenCase) DisplayName() string { return "打开工作流" }

func (OpenCase) ParamsSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"case_id":  { "type": "string" },
			"step":     { "type": "string", "enum": ["preview", "start", "confirm", "exit", "skip", "text", "media", "check"] },
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

// stringOf 把参数值统一成字符串：invoke 可能是内存直传（[]string / uint64），
// 也可能经过 JSON 反序列化（[]any / float64），两种形态都要兼容。
func stringOf(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case json.Number:
		return t.String()
	case uint64:
		return strconv.FormatUint(t, 10)
	case int64:
		return strconv.FormatInt(t, 10)
	case int:
		return strconv.Itoa(t)
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case nil:
		return ""
	default:
		return ""
	}
}

func (o OpenCase) Invoke(ctx context.Context, acct protocol.AccountCtx, nav protocol.Nav, chatID sharedkernel.ChatID, params map[string]any) (protocol.Result, error) {
	step, _ := params["step"].(string)
	if step == "" {
		step = "preview"
	}
	channelID := acct.ChannelID
	switch step {
	case "check":
		if o.App == nil {
			return protocol.Result{Text: "none"}, nil
		}
		if _, err := o.App.GetSession(ctx, chatID); err != nil {
			return protocol.Result{Text: "none"}, nil
		}
		return protocol.Result{Text: "active"}, nil
	}
	if denied, err := o.authorize(ctx, acct); err != nil {
		return protocol.Result{}, err
	} else if denied {
		return protocol.Result{
			Text: o.renderText(ctx, channelID, texttpl.KeyAccessDenied, nil),
		}, nil
	}
	switch step {
	case "preview":
		return o.preview(ctx, channelID, params)
	case "start":
		return o.start(ctx, channelID, acct, chatID, params)
	case "skip":
		return o.skip(ctx, channelID, chatID)
	case "text":
		return o.submitText(ctx, channelID, chatID, params)
	case "media":
		return o.submitMedia(ctx, channelID, chatID, params)
	case "confirm":
		return o.confirm(ctx, channelID, chatID)
	case "exit":
		if o.App == nil {
			return protocol.Result{}, fmt.Errorf("open_case: app not configured")
		}
		if err := o.App.ExitSession(ctx, chatID); err != nil {
			if !errors.Is(err, convdomain.ErrNoActiveSession) && !errors.Is(err, convdomain.ErrNotFound) {
				return protocol.Result{}, err
			}
		}
		return protocol.Result{Text: o.renderText(ctx, channelID, texttpl.KeyExitDone, nil)}, nil
	default:
		return protocol.Result{}, fmt.Errorf("open_case: unknown step %q", step)
	}
}

func (o OpenCase) authorize(ctx context.Context, acct protocol.AccountCtx) (bool, error) {
	if o.Users == nil {
		return false, fmt.Errorf("open_case: user access policy not configured")
	}
	user, err := o.Users.GetByID(ctx, acct.InternalUserID)
	if err != nil {
		return false, fmt.Errorf("open_case: resolve user access: %w", err)
	}
	if user == nil || user.Access != identitydomain.UserAccessAlwaysAllowed {
		return true, nil
	}
	return false, nil
}

func (o OpenCase) submitText(ctx context.Context, channelID string, chatID sharedkernel.ChatID, params map[string]any) (protocol.Result, error) {
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
			return protocol.Result{Text: o.renderText(ctx, channelID, texttpl.KeyInputInvalidNumber, nil)}, nil
		}
		draft = convdomain.DraftValue{Number: &n}
	case "boolean":
		b, err := strconv.ParseBool(text)
		if err != nil {
			return protocol.Result{Text: o.renderText(ctx, channelID, texttpl.KeyInputInvalidBoolean, nil)}, nil
		}
		draft = convdomain.DraftValue{Bool: &b}
	default:
		draft = convdomain.DraftValue{Text: &text}
	}
	view, err := o.App.SubmitInput(ctx, chatID, draft)
	if err != nil {
		return protocol.Result{}, err
	}
	return renderSession(ctx, channelID, o, view)
}

func (o OpenCase) submitMedia(ctx context.Context, channelID string, chatID sharedkernel.ChatID, params map[string]any) (protocol.Result, error) {
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
	return renderSession(ctx, channelID, o, view)
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

func (o OpenCase) preview(ctx context.Context, channelID string, params map[string]any) (protocol.Result, error) {
	if o.App == nil {
		return protocol.Result{}, fmt.Errorf("open_case: app not configured")
	}
	rawCaseID := stringOf(params["case_id"])
	if rawCaseID == "" {
		return protocol.Result{}, fmt.Errorf("open_case: case_id required for preview")
	}
	caseID, perr := sharedkernel.ParseCaseID(rawCaseID)
	if perr != nil {
		return protocol.Result{}, fmt.Errorf("open_case: invalid case_id")
	}
	c, err := o.App.GetCase(ctx, caseID)
	if err != nil {
		return protocol.Result{}, err
	}
	doc := c.Document
	var b strings.Builder
	fmt.Fprintf(&b, "📎 %s\n", doc.Name)
	if doc.Description != "" {
		fmt.Fprintf(&b, "%s\n", doc.Description)
	}
	text := strings.TrimSuffix(b.String(), "\n")
	// 预览效果图为媒体（blob key）时作为媒体随结果发送；否则回退为文本提示。
	var media []protocol.MediaRef
	if key, mime, ok := previewMediaRef(doc.Preview); ok {
		media = []protocol.MediaRef{{Key: key, MIME: mime}}
	} else if strings.TrimSpace(doc.Preview) != "" {
		text += "\n" + o.renderText(ctx, channelID, texttpl.KeyPreviewHintLabel, nil) + doc.Preview
	}
	return protocol.Result{
		Text:  text,
		Media: media,
		Options: []protocol.Option{
			{Label: o.renderText(ctx, channelID, texttpl.KeyButtonStartCase, nil), Value: map[string]any{"step": "start", "case_id": strconv.FormatUint(uint64(caseID), 10)}},
		},
	}, nil
}

// previewExtMIME maps preview media extensions to content types (admin allow-list).
var previewExtMIME = map[string]string{
	".png": "image/png", ".jpg": "image/jpeg", ".jpeg": "image/jpeg",
	".webp": "image/webp", ".gif": "image/gif", ".mp4": "video/mp4", ".webm": "video/webm",
}

// previewMediaRef 把 Case.preview 解析为可投递的 blob 媒体引用（仅 previews/ 前缀），
// 非媒体（旧文本）返回 ok=false，调用方回退文本提示。
func previewMediaRef(preview string) (key string, mime string, ok bool) {
	p := strings.TrimSpace(preview)
	if !strings.HasPrefix(p, "previews/") {
		return "", "", false
	}
	ext := ""
	if i := strings.LastIndex(p, "."); i >= 0 {
		ext = p[i:]
	}
	m := previewExtMIME[strings.ToLower(ext)]
	if m == "" {
		return "", "", false
	}
	return p, m, true
}

func (o OpenCase) start(ctx context.Context, channelID string, acct protocol.AccountCtx, chatID sharedkernel.ChatID, params map[string]any) (protocol.Result, error) {
	if o.App == nil {
		return protocol.Result{}, fmt.Errorf("open_case: app not configured")
	}
	rawCaseID := stringOf(params["case_id"])
	if rawCaseID == "" {
		return protocol.Result{}, fmt.Errorf("open_case: case_id required for start")
	}
	caseID, perr := sharedkernel.ParseCaseID(rawCaseID)
	if perr != nil {
		return protocol.Result{}, fmt.Errorf("open_case: invalid case_id")
	}
	view, err := o.App.StartCase(ctx, botapp.StartCaseCmd{
		ChatID: chatID, UserID: acct.InternalUserID, CaseID: caseID,
	})
	if err != nil {
		if errors.Is(err, convdomain.ErrSessionLocked) {
			return protocol.Result{
				Text: o.renderText(ctx, channelID, texttpl.KeyUnfinishedSession, nil),
				Options: []protocol.Option{
					{Label: o.renderText(ctx, channelID, texttpl.KeyButtonExit, nil), Value: map[string]any{"step": "exit"}},
				},
			}, nil
		}
		return protocol.Result{}, err
	}
	return renderSession(ctx, channelID, o, view)
}

func (o OpenCase) skip(ctx context.Context, channelID string, chatID sharedkernel.ChatID) (protocol.Result, error) {
	if o.App == nil {
		return protocol.Result{}, fmt.Errorf("open_case: app not configured")
	}
	view, err := o.App.SkipInput(ctx, chatID)
	if err != nil {
		return protocol.Result{}, err
	}
	return renderSession(ctx, channelID, o, view)
}

func (o OpenCase) confirm(ctx context.Context, channelID string, chatID sharedkernel.ChatID) (protocol.Result, error) {
	if o.App == nil {
		return protocol.Result{}, fmt.Errorf("open_case: app not configured")
	}
	res, err := o.App.ConfirmRun(ctx, botapp.ConfirmRunCmd{ChatID: chatID})
	if err != nil {
		return protocol.Result{}, err
	}
	return protocol.Result{Text: o.renderText(ctx, channelID, texttpl.KeySubmitStarted, map[string]string{"task_id": string(res.TaskID)})}, nil
}

func renderSession(ctx context.Context, channelID string, o OpenCase, view *botapp.SessionView) (protocol.Result, error) {
	if view.Status == convdomain.StatusConfirming {
		return protocol.Result{
			Text: o.renderText(ctx, channelID, texttpl.KeyConfirmRun, nil),
			Options: []protocol.Option{
				{Label: o.renderText(ctx, channelID, texttpl.KeyButtonConfirmRun, nil), Value: map[string]any{"step": "confirm"}},
				{Label: o.renderText(ctx, channelID, texttpl.KeyButtonExit, nil), Value: map[string]any{"step": "exit"}},
			},
		}, nil
	}
	key := "(done)"
	if view.Index >= 0 && view.Index < len(view.Keys) {
		key = view.Keys[view.Index]
	}
	vars := map[string]string{"key": key}
	if view.CaseID != 0 {
		if c, err := o.App.GetCase(ctx, view.CaseID); err == nil && c != nil {
			vars["case_name"] = c.Document.Name
		}
	}
	if view.Index >= 0 && len(view.Keys) > 0 {
		vars["progress"] = fmt.Sprintf("%d/%d", view.Index+1, len(view.Keys))
	}

	var currentField *catalogdomain.InputField
	if view.CaseID != 0 {
		if c, err := o.App.GetCase(ctx, view.CaseID); err == nil && c != nil {
			for i := range c.Document.Inputs {
				if c.Document.Inputs[i].Key == key {
					currentField = &c.Document.Inputs[i]
					break
				}
			}
		}
	}

	canSkip := currentField == nil || !currentField.Required || currentField.SkipAllowed
	if currentField != nil && currentField.Description != "" {
		vars["description"] = currentField.Description
	}

	options := []protocol.Option{{Label: o.renderText(ctx, channelID, texttpl.KeyButtonExit, nil), Value: map[string]any{"step": "exit"}}}
	if canSkip {
		options = append([]protocol.Option{{Label: o.renderText(ctx, channelID, texttpl.KeyButtonSkip, nil), Value: map[string]any{"step": "skip"}}}, options...)
	}
	prompt := o.renderText(ctx, channelID, texttpl.KeyInputPrompt, vars)
	if currentField != nil && currentField.Description != "" {
		prompt += "\n\n" + currentField.Description
	}
	return protocol.Result{Text: prompt, Options: options}, nil
}
