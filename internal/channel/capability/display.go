package capability

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mr9esx/comfyui_tgbot/internal/channel/protocol"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

// ReplyText replies with a fixed text (placeholder-style button).
type ReplyText struct{}

func (ReplyText) ID() string          { return "reply_text" }
func (ReplyText) DisplayName() string { return "提示文字" }
func (ReplyText) ParamsSchema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"text":{"type":"string","x-admin":{"widget":"text"}}},"required":["text"]}`)
}
func (ReplyText) Render(string, map[string]any) (protocol.RenderDecl, error) {
	return protocol.RenderDecl{Entry: "message_button"}, nil
}
func (ReplyText) Invoke(_ context.Context, _ protocol.AccountCtx, _ protocol.Nav, _ sharedkernel.ChatID, params map[string]any) (protocol.Result, error) {
	text, _ := params["text"].(string)
	return protocol.Result{Text: text}, nil
}

// ReplyMedia replies with text plus image URLs.
type ReplyMedia struct{}

func (ReplyMedia) ID() string          { return "reply_media" }
func (ReplyMedia) DisplayName() string { return "回复图文" }
func (ReplyMedia) ParamsSchema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"text":{"type":"string","x-admin":{"widget":"text"}},"images":{"type":"array","items":{"type":"string"}}},"required":["text"]}`)
}
func (ReplyMedia) Render(string, map[string]any) (protocol.RenderDecl, error) {
	return protocol.RenderDecl{Entry: "message_button"}, nil
}
func (ReplyMedia) Invoke(_ context.Context, _ protocol.AccountCtx, _ protocol.Nav, _ sharedkernel.ChatID, params map[string]any) (protocol.Result, error) {
	text, _ := params["text"].(string)
	raw, ok := params["images"].([]any)
	if !ok {
		return protocol.Result{}, fmt.Errorf("reply_media: images must be an array")
	}
	var urls []string
	for _, v := range raw {
		if s, ok := v.(string); ok && s != "" {
			urls = append(urls, s)
		}
	}
	return protocol.Result{Text: text, MediaURLs: urls}, nil
}
