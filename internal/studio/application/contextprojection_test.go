package application

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/Mr9esx/Pixoma/internal/studio/domain"
	"github.com/cloudwego/eino/schema"
)

func TestProjectModelHistoryReplaysCompletedToolPair(t *testing.T) {
	now := time.Date(2026, 9, 22, 8, 0, 0, 0, time.UTC)
	user := &domain.Message{ID: "user-1", SessionID: "session-1", AccountID: "account-1", Role: domain.MessageRoleUser, ContentJSON: json.RawMessage(`[{"type":"text","text":"查雨夜资料"}]`), CreatedAt: now}
	run := &domain.Run{ID: "run-1", SessionID: "session-1", AccountID: "account-1", TriggerMessageID: user.ID, CreatedAt: now}
	events := []*domain.Event{
		{ID: "e1", RunID: run.ID, SessionID: run.SessionID, AccountID: run.AccountID, Sequence: 1, Type: EventToolCallStart, Payload: json.RawMessage(`{"tool_call_id":"call-1","tool_name":"search"}`), CreatedAt: now.Add(time.Second)},
		{ID: "e2", RunID: run.ID, SessionID: run.SessionID, AccountID: run.AccountID, Sequence: 2, Type: EventToolCallArgs, Payload: json.RawMessage(`{"tool_call_id":"call-1","delta":"{\"query\":\"雨夜\"}"}`), CreatedAt: now.Add(2 * time.Second)},
		{ID: "e3", RunID: run.ID, SessionID: run.SessionID, AccountID: run.AccountID, Sequence: 3, Type: EventToolCallResult, Payload: json.RawMessage(`{"tool_call_id":"call-1","content":"找到三条资料","is_error":false}`), CreatedAt: now.Add(3 * time.Second)},
		{ID: "e4", RunID: run.ID, SessionID: run.SessionID, AccountID: run.AccountID, Sequence: 4, Type: EventToolCallEnd, Payload: json.RawMessage(`{"tool_call_id":"call-1","tool_name":"search","is_error":false}`), CreatedAt: now.Add(4 * time.Second)},
	}

	history, err := ProjectModelHistory([]*domain.Message{user}, []*domain.Run{run}, events, "", "")
	if err != nil {
		t.Fatalf("ProjectModelHistory() error = %v", err)
	}
	if len(history) != 3 {
		t.Fatalf("history length = %d, want 3: %#v", len(history), history)
	}
	if history[0].Role != schema.User || history[0].Content != "查雨夜资料" {
		t.Fatalf("user = %#v", history[0])
	}
	if history[1].Role != schema.Assistant || len(history[1].ToolCalls) != 1 {
		t.Fatalf("tool request = %#v", history[1])
	}
	call := history[1].ToolCalls[0]
	if call.ID != "call-1" || call.Function.Name != "search" || call.Function.Arguments != `{"query":"雨夜"}` {
		t.Fatalf("tool call = %#v", call)
	}
	if history[2].Role != schema.Tool || history[2].ToolCallID != "call-1" || history[2].Content != "找到三条资料" {
		t.Fatalf("tool result = %#v", history[2])
	}
}

func TestProjectModelHistoryDropsUnmatchedToolCall(t *testing.T) {
	now := time.Date(2026, 9, 22, 8, 0, 0, 0, time.UTC)
	user := &domain.Message{ID: "user-1", SessionID: "session-1", AccountID: "account-1", Role: domain.MessageRoleUser, ContentJSON: json.RawMessage(`[{"type":"text","text":"继续"}]`), CreatedAt: now}
	run := &domain.Run{ID: "run-1", SessionID: "session-1", AccountID: "account-1", TriggerMessageID: user.ID, CreatedAt: now}
	events := []*domain.Event{{ID: "e1", RunID: run.ID, SessionID: run.SessionID, AccountID: run.AccountID, Sequence: 1, Type: EventToolCallStart, Payload: json.RawMessage(`{"tool_call_id":"call-1","tool_name":"search"}`), CreatedAt: now.Add(time.Second)}}

	history, err := ProjectModelHistory([]*domain.Message{user}, []*domain.Run{run}, events, "", "")
	if err != nil {
		t.Fatalf("ProjectModelHistory() error = %v", err)
	}
	if len(history) != 1 || history[0].Role != schema.User {
		t.Fatalf("history = %#v, want only the user message", history)
	}
}

func TestProjectModelHistoryMapsToolGroupToRunBoundary(t *testing.T) {
	now := time.Date(2026, 9, 22, 8, 0, 0, 0, time.UTC)
	user := &domain.Message{ID: "user-1", SessionID: "session-1", AccountID: "account-1", Role: domain.MessageRoleUser, ContentJSON: json.RawMessage(`[{"type":"text","text":"查资料"}]`), CreatedAt: now}
	run := &domain.Run{ID: "run-1", SessionID: "session-1", AccountID: "account-1", TriggerMessageID: user.ID, CreatedAt: now}
	events := []*domain.Event{
		{ID: "e1", RunID: run.ID, SessionID: run.SessionID, AccountID: run.AccountID, Sequence: 1, Type: EventToolCallStart, Payload: json.RawMessage(`{"tool_call_id":"call-1","tool_name":"search"}`), CreatedAt: now.Add(time.Second)},
		{ID: "e2", RunID: run.ID, SessionID: run.SessionID, AccountID: run.AccountID, Sequence: 2, Type: EventToolCallResult, Payload: json.RawMessage(`{"tool_call_id":"call-1","content":"结果"}`), CreatedAt: now.Add(2 * time.Second)},
	}

	projection, err := ProjectModelHistoryWithBoundaries([]*domain.Message{user}, []*domain.Run{run}, events, "", "")
	if err != nil {
		t.Fatalf("ProjectModelHistoryWithBoundaries() error = %v", err)
	}
	if len(projection.Messages) != 3 || len(projection.BoundaryMessageIDs) != 3 {
		t.Fatalf("projection = %#v", projection)
	}
	for index, boundary := range projection.BoundaryMessageIDs {
		if boundary != user.ID {
			t.Fatalf("boundary[%d] = %q, want %q", index, boundary, user.ID)
		}
	}
}

func TestProjectModelHistoryKeepsStoredAssistantInItsOriginalTurnWhenEndEventIsMissing(t *testing.T) {
	now := time.Date(2026, 9, 22, 8, 0, 0, 0, time.UTC)
	firstUser := &domain.Message{ID: "user-1", SessionID: "session-1", AccountID: "account-1", RunID: "run-1", Role: domain.MessageRoleUser, ContentJSON: json.RawMessage(`[{"type":"text","text":"第一轮"}]`), CreatedAt: now}
	firstAssistant := &domain.Message{ID: "assistant-1", SessionID: "session-1", AccountID: "account-1", RunID: "run-1", Role: domain.MessageRoleAssistant, ContentJSON: json.RawMessage(`[{"type":"text","text":"第一轮结论"}]`), CreatedAt: now.Add(time.Second)}
	secondUser := &domain.Message{ID: "user-2", SessionID: "session-1", AccountID: "account-1", RunID: "run-2", Role: domain.MessageRoleUser, ContentJSON: json.RawMessage(`[{"type":"text","text":"第二轮"}]`), CreatedAt: now.Add(2 * time.Second)}
	runs := []*domain.Run{
		{ID: "run-1", SessionID: "session-1", AccountID: "account-1", TriggerMessageID: firstUser.ID, CreatedAt: now},
		{ID: "run-2", SessionID: "session-1", AccountID: "account-1", TriggerMessageID: secondUser.ID, CreatedAt: now.Add(2 * time.Second)},
	}

	history, err := ProjectModelHistory([]*domain.Message{firstUser, firstAssistant, secondUser}, runs, nil, "", "")
	if err != nil {
		t.Fatalf("ProjectModelHistory() error = %v", err)
	}
	if len(history) != 3 || history[0].Content != "第一轮" || history[1].Content != "第一轮结论" || history[2].Content != "第二轮" {
		t.Fatalf("history = %#v", history)
	}
}
