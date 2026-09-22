package studio

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	studioapp "github.com/Mr9esx/Pixoma/internal/studio/application"
	"github.com/Mr9esx/Pixoma/internal/studio/domain"
	"github.com/gorilla/websocket"
)

type aguiMessage struct {
	ID      string          `json:"id"`
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

type aguiRunInput struct {
	ThreadID        string          `json:"threadId"`
	RunID           string          `json:"runId"`
	ProtocolVersion string          `json:"protocolVersion"`
	Messages        []aguiMessage   `json:"messages"`
	ForwardedProps  json.RawMessage `json:"forwardedProps"`
	Tools           json.RawMessage `json:"tools"`
	Context         json.RawMessage `json:"context"`
	State           json.RawMessage `json:"state"`
}

type aguiRunConfig struct {
	ModelConfigID    string                `json:"modelConfigId"`
	PermissionMode   domain.PermissionMode `json:"permissionMode"`
	SelectedSkillIDs []string              `json:"selectedSkillIds"`
	SelectedAssetIDs []string              `json:"selectedAssetIds"`
	SelectedAssets   []aguiAssetReference  `json:"selectedAssets"`
}

type aguiAssetReference struct {
	AssetID        string `json:"assetId"`
	AssetVersionID string `json:"assetVersionId"`
}

var aguiWebSocketUpgrader = websocket.Upgrader{
	ReadBufferSize:  16 << 10,
	WriteBufferSize: 16 << 10,
	CheckOrigin: func(r *http.Request) bool {
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if origin == "" {
			return true
		}
		parsed, err := url.Parse(origin)
		if err != nil || parsed.Host == "" {
			return false
		}
		if strings.EqualFold(parsed.Host, r.Host) {
			return true
		}
		// The development UI is served by Vite and proxies this socket to the
		// API process, so its localhost origin naturally has a different port.
		return strings.HasPrefix(strings.ToLower(parsed.Hostname()), "localhost") || parsed.Hostname() == "127.0.0.1"
	},
}

func (h *Handler) streamAGUI(w http.ResponseWriter, r *http.Request) {
	accountID, ok := accountID(w, r)
	if !ok {
		return
	}
	var input aguiRunInput
	if err := decodeAGUIJSON(r, &input); err != nil {
		slog.Warn("studio agui request decode failed", "path", r.URL.Path, "err", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求内容格式不正确"})
		return
	}
	input.ThreadID = strings.TrimSpace(input.ThreadID)
	input.RunID = strings.TrimSpace(input.RunID)
	text := lastAGUIUserText(input.Messages)
	if input.ThreadID == "" || input.RunID == "" || text == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "会话、运行和用户消息不能为空"})
		return
	}
	config := decodeAGUIRunConfig(input.ForwardedProps)
	if !config.PermissionMode.Valid() {
		config.PermissionMode = domain.PermissionRequestApproval
	}
	result, err := h.Service.SendMessage(r.Context(), studioapp.SendMessageInput{
		AccountID: accountID, SessionID: input.ThreadID, Text: text,
		ModelConfigID: config.ModelConfigID, PermissionMode: config.PermissionMode, SkillIDs: config.SelectedSkillIDs, SelectedAssetIDs: config.SelectedAssetIDs, SelectedAssets: toAssetReferences(config.SelectedAssets),
	})
	if err != nil {
		writeError(w, err)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "当前连接不支持流式响应"})
		return
	}
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	writeAGUIEvent(w, flusher, 0, map[string]any{
		"type": "RUN_STARTED", "threadId": input.ThreadID, "runId": input.RunID,
		"protocolVersion": "1.0", "metadata": map[string]any{"studioRunId": result.Run.ID},
	})

	var after uint64
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	h.pumpAGUI(r.Context(), accountID, result.Run.ID, input, after, ticker.C, func(sequence uint64, event map[string]any) error {
		writeAGUIEvent(w, flusher, sequence, event)
		return nil
	})
}

func (h *Handler) streamAGUIWebSocket(w http.ResponseWriter, r *http.Request) {
	accountID, ok := accountID(w, r)
	if !ok {
		return
	}
	conn, err := aguiWebSocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Warn("studio agui websocket upgrade failed", "path", r.URL.Path, "err", err)
		return
	}
	defer conn.Close()
	var input aguiRunInput
	if err := conn.ReadJSON(&input); err != nil {
		_ = conn.WriteJSON(aguiRunError(input, "请求内容格式不正确"))
		return
	}
	input.ThreadID = strings.TrimSpace(input.ThreadID)
	input.RunID = strings.TrimSpace(input.RunID)
	text := lastAGUIUserText(input.Messages)
	if input.ThreadID == "" || input.RunID == "" || text == "" {
		_ = conn.WriteJSON(aguiRunError(input, "会话、运行和用户消息不能为空"))
		return
	}
	config := decodeAGUIRunConfig(input.ForwardedProps)
	if !config.PermissionMode.Valid() {
		config.PermissionMode = domain.PermissionRequestApproval
	}
	result, err := h.Service.SendMessage(r.Context(), studioapp.SendMessageInput{
		AccountID: accountID, SessionID: input.ThreadID, Text: text,
		ModelConfigID: config.ModelConfigID, PermissionMode: config.PermissionMode, SkillIDs: config.SelectedSkillIDs, SelectedAssetIDs: config.SelectedAssetIDs, SelectedAssets: toAssetReferences(config.SelectedAssets),
	})
	if err != nil {
		_ = conn.WriteJSON(aguiRunError(input, err.Error()))
		return
	}
	if err := conn.WriteJSON(map[string]any{
		"type": "RUN_STARTED", "threadId": input.ThreadID, "runId": input.RunID,
		"protocolVersion": "1.0", "metadata": map[string]any{"studioRunId": result.Run.ID},
	}); err != nil {
		return
	}
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	_ = h.pumpAGUI(r.Context(), accountID, result.Run.ID, input, 0, ticker.C, func(_ uint64, event map[string]any) error {
		return conn.WriteJSON(event)
	})
}

func (h *Handler) pumpAGUI(ctx context.Context, accountID, studioRunID string, input aguiRunInput, after uint64, ticks <-chan time.Time, emit func(uint64, map[string]any) error) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		events, err := h.Repo.ListEventsAfter(ctx, accountID, studioRunID, after, 200)
		if err != nil {
			return emit(after+1, aguiRunError(input, "读取运行事件失败"))
		}
		for _, event := range events {
			after = event.Sequence
			mapped := mapStudioEventToAGUI(event.Type, event.Payload, input.ThreadID, input.RunID)
			if mapped == nil {
				continue
			}
			if err := emit(event.Sequence, mapped); err != nil {
				return err
			}
			if event.Type == studioapp.EventRunFinished {
				return nil
			}
		}

		run, err := h.Repo.GetRun(ctx, accountID, studioRunID)
		if err != nil {
			return emit(after+1, aguiRunError(input, "读取运行状态失败"))
		}
		switch run.Status {
		case domain.RunWaitingApproval:
			approvals, _ := h.Repo.ListApprovals(ctx, accountID, run.ID)
			interrupts := make([]map[string]any, 0, len(approvals))
			for _, approval := range approvals {
				if approval.Status != domain.ApprovalPending {
					continue
				}
				interrupts = append(interrupts, map[string]any{
					"id": approval.ID, "reason": "tool_approval", "message": "需要批准后继续执行",
					"toolCallId":     approval.ToolCallID,
					"responseSchema": map[string]any{"type": "boolean"},
				})
			}
			if len(interrupts) > 0 {
				if err := emit(after+1, map[string]any{
					"type": "RUN_FINISHED", "threadId": input.ThreadID, "runId": input.RunID,
					"outcome": map[string]any{"type": "interrupt", "interrupts": interrupts},
				}); err != nil {
					return err
				}
				return nil
			}
		case domain.RunFailed:
			return emit(after+1, aguiRunError(input, run.ErrorMessage))
		case domain.RunCancelled:
			if err := emit(after+1, map[string]any{
				"type": "RUN_FINISHED", "threadId": input.ThreadID, "runId": input.RunID,
				"outcome": map[string]any{"type": "cancelled"},
			}); err != nil {
				return err
			}
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticks:
		}
	}
}

// AG-UI is an external protocol and may add fields that Pixoma does not use
// yet. Keep JSON syntax validation, but allow those protocol extensions.
func decodeAGUIJSON(r *http.Request, out any) error {
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	return decoder.Decode(out)
}

func toAssetReferences(values []aguiAssetReference) []domain.AssetReference {
	if len(values) == 0 {
		return nil
	}
	out := make([]domain.AssetReference, 0, len(values))
	for _, value := range values {
		out = append(out, domain.AssetReference{AssetID: value.AssetID, AssetVersionID: value.AssetVersionID})
	}
	return out
}

func lastAGUIUserText(messages []aguiMessage) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role != "user" {
			continue
		}
		var text string
		if json.Unmarshal(messages[i].Content, &text) == nil {
			return strings.TrimSpace(text)
		}
		var parts []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}
		if json.Unmarshal(messages[i].Content, &parts) == nil {
			var builder strings.Builder
			for _, part := range parts {
				if part.Type == "text" {
					builder.WriteString(part.Text)
				}
			}
			return strings.TrimSpace(builder.String())
		}
	}
	return ""
}

func decodeAGUIRunConfig(raw json.RawMessage) aguiRunConfig {
	var forwarded struct {
		RunConfig aguiRunConfig `json:"runConfig"`
	}
	_ = json.Unmarshal(raw, &forwarded)
	return forwarded.RunConfig
}

func mapStudioEventToAGUI(eventType string, payload json.RawMessage, threadID, runID string) map[string]any {
	var value map[string]any
	if json.Unmarshal(payload, &value) != nil {
		value = map[string]any{}
	}
	stringValue := func(key string) string {
		result, _ := value[key].(string)
		return result
	}
	switch eventType {
	case studioapp.EventRunStarted:
		return nil
	case studioapp.EventTextMessageStart:
		return map[string]any{"type": eventType, "messageId": stringValue("message_id"), "role": "assistant"}
	case studioapp.EventTextMessageContent:
		return map[string]any{"type": eventType, "messageId": stringValue("message_id"), "delta": stringValue("delta")}
	case studioapp.EventTextMessageEnd:
		return map[string]any{"type": eventType, "messageId": stringValue("message_id")}
	case studioapp.EventReasoningStart, studioapp.EventReasoningEnd:
		return map[string]any{"type": eventType}
	case studioapp.EventReasoningMessageStart:
		return map[string]any{"type": eventType, "messageId": stringValue("message_id")}
	case studioapp.EventReasoningMessageContent:
		return map[string]any{"type": eventType, "messageId": stringValue("message_id"), "delta": stringValue("delta")}
	case studioapp.EventReasoningMessageEnd:
		return map[string]any{"type": eventType, "messageId": stringValue("message_id")}
	case studioapp.EventToolCallStart:
		return map[string]any{
			"type": eventType, "toolCallId": stringValue("tool_call_id"),
			"toolCallName": stringValue("tool_name"),
		}
	case studioapp.EventToolCallArgs:
		return map[string]any{
			"type": eventType, "toolCallId": stringValue("tool_call_id"),
			"delta": stringValue("delta"),
		}
	case studioapp.EventToolCallResult:
		return map[string]any{
			"type": eventType, "toolCallId": stringValue("tool_call_id"),
			"content": stringValue("content"), "role": "tool",
			"isError": boolValue(value, "is_error"),
		}
	case studioapp.EventToolCallEnd:
		return map[string]any{"type": eventType, "toolCallId": stringValue("tool_call_id")}
	case studioapp.EventRunFinished:
		return map[string]any{
			"type": eventType, "threadId": threadID, "runId": runID,
			"outcome": map[string]any{"type": "success"},
		}
	default:
		return map[string]any{"type": "CUSTOM", "name": "pixoma." + strings.ToLower(eventType), "value": value}
	}
}

func boolValue(value map[string]any, key string) bool {
	result, _ := value[key].(bool)
	return result
}

func writeAGUIEvent(w http.ResponseWriter, flusher http.Flusher, sequence uint64, event map[string]any) {
	raw, err := json.Marshal(event)
	if err != nil {
		return
	}
	_, _ = fmt.Fprintf(w, "id: %d\ndata: %s\n\n", sequence, raw)
	flusher.Flush()
}

func aguiRunError(input aguiRunInput, message string) map[string]any {
	if strings.TrimSpace(message) == "" {
		message = "运行失败"
	}
	return map[string]any{
		"type": "RUN_ERROR", "threadId": input.ThreadID, "runId": input.RunID,
		"message": message,
	}
}
