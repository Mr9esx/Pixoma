package application

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

func TestProjectSessionTranscriptReplaysReasoningToolsAndTextInOrder(t *testing.T) {
	now := time.Date(2026, 9, 22, 7, 0, 0, 0, time.UTC)
	user := &domain.Message{
		ID: "user-1", SessionID: "session-1", AccountID: "account-1",
		Role: domain.MessageRoleUser, ContentJSON: json.RawMessage(`[{"type":"text","text":"请查资料"}]`), CreatedAt: now,
	}
	legacyAssistant := &domain.Message{
		ID: "assistant-1", SessionID: "session-1", AccountID: "account-1", RunID: "run-1",
		Role: domain.MessageRoleAssistant, ContentJSON: json.RawMessage(`[{"type":"text","text":"结论如下"}]`), CreatedAt: now.Add(10 * time.Second),
	}
	run := &domain.Run{ID: "run-1", SessionID: "session-1", AccountID: "account-1", TriggerMessageID: user.ID, CreatedAt: now.Add(time.Second)}
	events := map[string][]*domain.Event{
		run.ID: {
			{ID: "e1", RunID: run.ID, SessionID: run.SessionID, AccountID: run.AccountID, Sequence: 1, Type: EventReasoningMessageStart, Payload: json.RawMessage(`{"message_id":"reason-1"}`)},
			{ID: "e2", RunID: run.ID, SessionID: run.SessionID, AccountID: run.AccountID, Sequence: 2, Type: EventReasoningMessageContent, Payload: json.RawMessage(`{"message_id":"reason-1","delta":"先查资料。"}`)},
			{ID: "e3", RunID: run.ID, SessionID: run.SessionID, AccountID: run.AccountID, Sequence: 3, Type: EventReasoningMessageEnd, Payload: json.RawMessage(`{"message_id":"reason-1"}`)},
			{ID: "e4", RunID: run.ID, SessionID: run.SessionID, AccountID: run.AccountID, Sequence: 4, Type: EventToolCallStart, Payload: json.RawMessage(`{"tool_call_id":"tool-1","tool_name":"search"}`)},
			{ID: "e5", RunID: run.ID, SessionID: run.SessionID, AccountID: run.AccountID, Sequence: 5, Type: EventToolCallArgs, Payload: json.RawMessage(`{"tool_call_id":"tool-1","delta":"{\"q\":\"雨夜\"}"}`)},
			{ID: "e6", RunID: run.ID, SessionID: run.SessionID, AccountID: run.AccountID, Sequence: 6, Type: EventToolCallResult, Payload: json.RawMessage(`{"tool_call_id":"tool-1","content":"找到 3 条资料","is_error":false}`)},
			{ID: "e7", RunID: run.ID, SessionID: run.SessionID, AccountID: run.AccountID, Sequence: 7, Type: EventTextMessageStart, Payload: json.RawMessage(`{"message_id":"assistant-1"}`)},
			{ID: "e8", RunID: run.ID, SessionID: run.SessionID, AccountID: run.AccountID, Sequence: 8, Type: EventTextMessageContent, Payload: json.RawMessage(`{"message_id":"assistant-1","delta":"结论如下"}`)},
			{ID: "e9", RunID: run.ID, SessionID: run.SessionID, AccountID: run.AccountID, Sequence: 9, Type: EventTextMessageEnd, Payload: json.RawMessage(`{"message_id":"assistant-1"}`)},
		},
	}

	transcript := ProjectSessionTranscript([]*domain.Message{user, legacyAssistant}, []*domain.Run{run}, events)
	if len(transcript.Messages) != 5 {
		t.Fatalf("messages = %#v", transcript.Messages)
	}
	if transcript.Messages[0].Role != "user" || transcript.Messages[0].Content != "请查资料" {
		t.Fatalf("user message = %#v", transcript.Messages[0])
	}
	if transcript.Messages[1].Role != "reasoning" || transcript.Messages[1].Content != "先查资料。" {
		t.Fatalf("reasoning message = %#v", transcript.Messages[1])
	}
	if len(transcript.Messages[2].ToolCalls) != 1 || transcript.Messages[2].ToolCalls[0].Function.Arguments != "{\"q\":\"雨夜\"}" {
		t.Fatalf("tool call message = %#v", transcript.Messages[2])
	}
	if transcript.Messages[3].Role != "tool" || transcript.Messages[3].Content != "找到 3 条资料" {
		t.Fatalf("tool result message = %#v", transcript.Messages[3])
	}
	if transcript.Messages[4].Role != "assistant" || transcript.Messages[4].Content != "结论如下" {
		t.Fatalf("assistant message = %#v", transcript.Messages[4])
	}
	if len(transcript.Events) != 9 || transcript.Events[4].Sequence != 5 {
		t.Fatalf("events = %#v", transcript.Events)
	}
}
