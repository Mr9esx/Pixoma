package application

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

// TranscriptMessage is the protocol-neutral subset of an AG-UI message that
// the Studio history endpoint exposes. Keeping this shape in the application
// layer lets the HTTP and WebSocket paths share the same replay semantics.
type TranscriptMessage struct {
	ID         string               `json:"id"`
	Role       string               `json:"role"`
	Content    string               `json:"content"`
	ToolCalls  []TranscriptToolCall `json:"toolCalls,omitempty"`
	ToolCallID string               `json:"toolCallId,omitempty"`
	IsError    bool                 `json:"isError,omitempty"`
}

type TranscriptToolCall struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Function TranscriptFunctionCall `json:"function"`
}

type TranscriptFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type TranscriptEvent struct {
	ID        string          `json:"id"`
	RunID     string          `json:"runId"`
	Sequence  uint64          `json:"sequence"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	CreatedAt time.Time       `json:"createdAt"`
}

type SessionTranscript struct {
	Messages []TranscriptMessage `json:"messages"`
	Events   []TranscriptEvent   `json:"events"`
}

// ProjectSessionTranscript rebuilds the visible conversation from persisted
// user messages and the append-only run events. Events remain in the result so
// Trace and chat can use the same canonical source without losing details.
func ProjectSessionTranscript(messages []*domain.Message, runs []*domain.Run, eventsByRun map[string][]*domain.Event) SessionTranscript {
	orderedRuns := append([]*domain.Run(nil), runs...)
	sort.SliceStable(orderedRuns, func(i, j int) bool {
		if orderedRuns[i].CreatedAt.Equal(orderedRuns[j].CreatedAt) {
			return orderedRuns[i].ID < orderedRuns[j].ID
		}
		return orderedRuns[i].CreatedAt.Before(orderedRuns[j].CreatedAt)
	})

	byID := make(map[string]*domain.Message, len(messages))
	for _, message := range messages {
		if message != nil {
			byID[message.ID] = message
		}
	}
	usedMessages := make(map[string]bool, len(messages))
	transcript := SessionTranscript{Messages: make([]TranscriptMessage, 0), Events: make([]TranscriptEvent, 0)}
	for _, run := range orderedRuns {
		if run == nil {
			continue
		}
		if trigger := byID[run.TriggerMessageID]; trigger != nil {
			transcript.Messages = append(transcript.Messages, transcriptMessagesFromStored(trigger)...)
			usedMessages[trigger.ID] = true
		}
		runEvents := append([]*domain.Event(nil), eventsByRun[run.ID]...)
		sort.SliceStable(runEvents, func(i, j int) bool {
			return runEvents[i].Sequence < runEvents[j].Sequence
		})
		transcript.appendRunEvents(runEvents)
		for _, event := range runEvents {
			if event == nil {
				continue
			}
			payload := decodeTranscriptPayload(event.Payload)
			if messageID := payload.stringValue("message_id"); messageID != "" {
				usedMessages[messageID] = true
			}
			transcript.Events = append(transcript.Events, TranscriptEvent{
				ID: event.ID, RunID: event.RunID, Sequence: event.Sequence, Type: event.Type,
				Payload: append(json.RawMessage(nil), event.Payload...), CreatedAt: event.CreatedAt,
			})
		}
	}

	orderedMessages := append([]*domain.Message(nil), messages...)
	sort.SliceStable(orderedMessages, func(i, j int) bool {
		if orderedMessages[i].CreatedAt.Equal(orderedMessages[j].CreatedAt) {
			return orderedMessages[i].ID < orderedMessages[j].ID
		}
		return orderedMessages[i].CreatedAt.Before(orderedMessages[j].CreatedAt)
	})
	for _, message := range orderedMessages {
		if message == nil || usedMessages[message.ID] {
			continue
		}
		transcript.Messages = append(transcript.Messages, transcriptMessagesFromStored(message)...)
	}
	return transcript
}

func (t *SessionTranscript) appendRunEvents(events []*domain.Event) {
	assistantIndices := make(map[string]int)
	reasoningIndices := make(map[string]int)
	toolIndices := make(map[string]int)
	toolResultIndices := make(map[string]int)
	for _, event := range events {
		if event == nil {
			continue
		}
		payload := decodeTranscriptPayload(event.Payload)
		messageID := payload.stringValue("message_id")
		toolCallID := payload.stringValue("tool_call_id")
		switch event.Type {
		case EventReasoningMessageStart:
			if messageID == "" {
				messageID = event.RunID + ":reasoning"
			}
			if _, ok := reasoningIndices[messageID]; !ok {
				reasoningIndices[messageID] = len(t.Messages)
				t.Messages = append(t.Messages, TranscriptMessage{ID: messageID, Role: "reasoning"})
			}
		case EventReasoningMessageContent:
			if messageID == "" {
				messageID = event.RunID + ":reasoning"
			}
			index, ok := reasoningIndices[messageID]
			if !ok {
				index = len(t.Messages)
				reasoningIndices[messageID] = index
				t.Messages = append(t.Messages, TranscriptMessage{ID: messageID, Role: "reasoning"})
			}
			t.Messages[index].Content += payload.stringValue("delta")
		case EventTextMessageStart:
			if messageID == "" {
				messageID = event.RunID + ":assistant"
			}
			if _, ok := assistantIndices[messageID]; !ok {
				assistantIndices[messageID] = len(t.Messages)
				t.Messages = append(t.Messages, TranscriptMessage{ID: messageID, Role: "assistant"})
			}
		case EventTextMessageContent:
			if messageID == "" {
				messageID = event.RunID + ":assistant"
			}
			index, ok := assistantIndices[messageID]
			if !ok {
				index = len(t.Messages)
				assistantIndices[messageID] = index
				t.Messages = append(t.Messages, TranscriptMessage{ID: messageID, Role: "assistant"})
			}
			t.Messages[index].Content += payload.stringValue("delta")
		case EventToolCallStart:
			if toolCallID == "" {
				continue
			}
			index, ok := toolIndices[toolCallID]
			if !ok {
				index = len(t.Messages)
				toolIndices[toolCallID] = index
				t.Messages = append(t.Messages, TranscriptMessage{
					ID: event.RunID + ":tool:" + toolCallID, Role: "assistant",
					ToolCalls: []TranscriptToolCall{{ID: toolCallID, Type: "function", Function: TranscriptFunctionCall{Name: payload.stringValue("tool_name")}}},
				})
			}
		case EventToolCallArgs:
			if index, ok := toolIndices[toolCallID]; ok && len(t.Messages[index].ToolCalls) > 0 {
				t.Messages[index].ToolCalls[0].Function.Arguments += payload.stringValue("delta")
			}
		case EventToolCallResult, EventToolCallEnd:
			if event.Type == EventToolCallEnd && payload.stringValue("content") == "" {
				continue
			}
			if toolCallID == "" {
				continue
			}
			index, ok := toolResultIndices[toolCallID]
			if !ok {
				index = len(t.Messages)
				toolResultIndices[toolCallID] = index
				t.Messages = append(t.Messages, TranscriptMessage{ID: event.RunID + ":tool-result:" + toolCallID, Role: "tool", ToolCallID: toolCallID})
			}
			if content := payload.stringValue("content"); content != "" {
				t.Messages[index].Content = content
			}
			t.Messages[index].IsError = payload.boolValue("is_error")
		}
	}
}

type transcriptPayload map[string]json.RawMessage

func decodeTranscriptPayload(raw json.RawMessage) transcriptPayload {
	value := transcriptPayload{}
	_ = json.Unmarshal(raw, &value)
	return value
}

func (p transcriptPayload) stringValue(key string) string {
	var value string
	_ = json.Unmarshal(p[key], &value)
	return value
}

func (p transcriptPayload) boolValue(key string) bool {
	var value bool
	_ = json.Unmarshal(p[key], &value)
	return value
}

func transcriptMessagesFromStored(message *domain.Message) []TranscriptMessage {
	if message == nil {
		return nil
	}
	var parts []messagePart
	if err := json.Unmarshal(message.ContentJSON, &parts); err != nil {
		return nil
	}
	text := strings.Builder{}
	var out []TranscriptMessage
	for index, part := range parts {
		if part.Type == "reasoning" {
			out = append(out, TranscriptMessage{ID: message.ID + ":reasoning:" + strconv.Itoa(index), Role: "reasoning", Content: part.Text})
			continue
		}
		if part.Type == "text" {
			text.WriteString(part.Text)
		}
	}
	if text.Len() > 0 {
		out = append(out, TranscriptMessage{ID: message.ID, Role: string(message.Role), Content: text.String()})
	}
	return out
}
