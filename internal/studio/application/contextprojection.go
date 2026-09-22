package application

import (
	"fmt"
	"sort"
	"strings"

	"github.com/cloudwego/eino/schema"

	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

// ModelHistory is the model-visible projection of a persisted session. Each
// BoundaryMessageID matches Messages at the same index and identifies the
// durable message boundary that can safely be summarized before that item.
// Tool events have no message row of their own, so they inherit their run's
// trigger message as the conservative boundary.
type ModelHistory struct {
	Messages           []*schema.Message
	BoundaryMessageIDs []string
}

// ProjectModelHistory reconstructs only the messages that are safe to send to
// a model. It deliberately differs from the chat transcript: reasoning and
// diagnostic events stay out, while a completed tool call/result pair becomes
// an atomic assistant/tool group.
func ProjectModelHistory(messages []*domain.Message, runs []*domain.Run, events []*domain.Event, summaryThroughMessageID, beforeMessageID string) ([]*schema.Message, error) {
	projection, err := ProjectModelHistoryWithBoundaries(messages, runs, events, summaryThroughMessageID, beforeMessageID)
	if err != nil {
		return nil, err
	}
	return projection.Messages, nil
}

// ProjectModelHistoryWithBoundaries is the executor-facing form of
// ProjectModelHistory. Keeping the mapping alongside the projection prevents
// a later compaction from persisting a boundary in the middle of a tool group.
func ProjectModelHistoryWithBoundaries(messages []*domain.Message, runs []*domain.Run, events []*domain.Event, summaryThroughMessageID, beforeMessageID string) (ModelHistory, error) {
	orderedMessages := append([]*domain.Message(nil), messages...)
	sort.SliceStable(orderedMessages, func(i, j int) bool {
		if orderedMessages[i].CreatedAt.Equal(orderedMessages[j].CreatedAt) {
			return orderedMessages[i].ID < orderedMessages[j].ID
		}
		return orderedMessages[i].CreatedAt.Before(orderedMessages[j].CreatedAt)
	})

	allowed, byID := modelHistoryMessages(orderedMessages, summaryThroughMessageID, beforeMessageID)
	byRun := make(map[string][]*domain.Message)
	for _, message := range allowed {
		if message.RunID != "" {
			byRun[message.RunID] = append(byRun[message.RunID], message)
		}
	}

	eventsByRun := make(map[string][]*domain.Event)
	for _, event := range events {
		if event != nil {
			eventsByRun[event.RunID] = append(eventsByRun[event.RunID], event)
		}
	}
	for _, runEvents := range eventsByRun {
		sort.SliceStable(runEvents, func(i, j int) bool { return runEvents[i].Sequence < runEvents[j].Sequence })
	}

	orderedRuns := append([]*domain.Run(nil), runs...)
	sort.SliceStable(orderedRuns, func(i, j int) bool {
		if orderedRuns[i].CreatedAt.Equal(orderedRuns[j].CreatedAt) {
			return orderedRuns[i].ID < orderedRuns[j].ID
		}
		return orderedRuns[i].CreatedAt.Before(orderedRuns[j].CreatedAt)
	})

	used := make(map[string]bool, len(allowed))
	history := ModelHistory{
		Messages:           make([]*schema.Message, 0, len(allowed)),
		BoundaryMessageIDs: make([]string, 0, len(allowed)),
	}
	for _, run := range orderedRuns {
		if run == nil {
			continue
		}
		trigger := byID[run.TriggerMessageID]
		if trigger == nil {
			continue
		}
		if message, err := schemaMessageFromDomain(trigger); err != nil {
			return ModelHistory{}, err
		} else if message != nil {
			history.append(message, trigger.ID)
		}
		used[trigger.ID] = true

		assistantByID := make(map[string]*domain.Message, len(byRun[run.ID]))
		storedAssistants := make([]*domain.Message, 0, len(byRun[run.ID]))
		for _, message := range byRun[run.ID] {
			assistantByID[message.ID] = message
			if message.Role == domain.MessageRoleAssistant {
				storedAssistants = append(storedAssistants, message)
			}
		}
		projectRunModelHistory(&history, eventsByRun[run.ID], assistantByID, used, trigger.ID)
		appendUnreferencedAssistants(&history, storedAssistants, used)
	}

	for _, message := range allowed {
		if message == nil || used[message.ID] {
			continue
		}
		converted, err := schemaMessageFromDomain(message)
		if err != nil {
			return ModelHistory{}, err
		}
		if converted != nil {
			history.append(converted, message.ID)
		}
	}
	if err := ValidateModelHistory(history.Messages); err != nil {
		return ModelHistory{}, err
	}
	return history, nil
}

func appendUnreferencedAssistants(history *ModelHistory, assistants []*domain.Message, used map[string]bool) {
	sort.SliceStable(assistants, func(i, j int) bool {
		if assistants[i].CreatedAt.Equal(assistants[j].CreatedAt) {
			return assistants[i].ID < assistants[j].ID
		}
		return assistants[i].CreatedAt.Before(assistants[j].CreatedAt)
	})
	for _, message := range assistants {
		if message == nil || used[message.ID] {
			continue
		}
		converted, err := schemaMessageFromDomain(message)
		if err != nil || converted == nil {
			continue
		}
		history.append(converted, message.ID)
		used[message.ID] = true
	}
}

func (h *ModelHistory) append(message *schema.Message, boundaryMessageID string) {
	if message == nil {
		return
	}
	h.Messages = append(h.Messages, message)
	h.BoundaryMessageIDs = append(h.BoundaryMessageIDs, boundaryMessageID)
}

func modelHistoryMessages(messages []*domain.Message, summaryThroughMessageID, beforeMessageID string) ([]*domain.Message, map[string]*domain.Message) {
	allowed := make([]*domain.Message, 0, len(messages))
	byID := make(map[string]*domain.Message, len(messages))
	passedSummary := strings.TrimSpace(summaryThroughMessageID) == ""
	for _, message := range messages {
		if message == nil {
			continue
		}
		if !passedSummary {
			if message.ID == summaryThroughMessageID {
				passedSummary = true
			}
			continue
		}
		if beforeMessageID != "" && message.ID == beforeMessageID {
			break
		}
		allowed = append(allowed, message)
		byID[message.ID] = message
	}
	return allowed, byID
}

type modelToolCall struct {
	id        string
	name      string
	arguments strings.Builder
}

func projectRunModelHistory(history *ModelHistory, events []*domain.Event, assistantByID map[string]*domain.Message, used map[string]bool, runBoundaryMessageID string) {
	pending := make(map[string]*modelToolCall)
	for _, event := range events {
		if event == nil {
			continue
		}
		payload := decodeTranscriptPayload(event.Payload)
		callID := payload.stringValue("tool_call_id")
		switch event.Type {
		case EventToolCallStart:
			if callID == "" {
				continue
			}
			pending[callID] = &modelToolCall{id: callID, name: payload.stringValue("tool_name")}
		case EventToolCallArgs:
			if call := pending[callID]; call != nil {
				call.arguments.WriteString(payload.stringValue("delta"))
			}
		case EventToolCallResult:
			call := pending[callID]
			if call == nil || strings.TrimSpace(call.name) == "" {
				continue
			}
			history.append(
				&schema.Message{Role: schema.Assistant, ToolCalls: []schema.ToolCall{{
					ID: call.id, Type: "function", Function: schema.FunctionCall{Name: call.name, Arguments: call.arguments.String()},
				}}},
				runBoundaryMessageID,
			)
			history.append(
				&schema.Message{Role: schema.Tool, ToolCallID: call.id, Name: call.name, Content: payload.stringValue("content")},
				runBoundaryMessageID,
			)
			delete(pending, callID)
		case EventTextMessageEnd:
			messageID := payload.stringValue("message_id")
			message := assistantByID[messageID]
			if message == nil || used[messageID] {
				continue
			}
			converted, err := schemaMessageFromDomain(message)
			if err != nil {
				return
			}
			if converted != nil {
				history.append(converted, message.ID)
				used[messageID] = true
			}
		}
	}
}

// ValidateModelHistory exists to keep invalid protocol groups from reaching
// provider adapters when future projections add new event types.
func ValidateModelHistory(history []*schema.Message) error {
	pending := make(map[string]bool)
	for _, message := range history {
		if message == nil {
			continue
		}
		for _, call := range message.ToolCalls {
			if call.ID == "" || call.Function.Name == "" {
				return fmt.Errorf("studio: invalid historical tool call")
			}
			pending[call.ID] = true
		}
		if message.Role == schema.Tool {
			if message.ToolCallID == "" || !pending[message.ToolCallID] {
				return fmt.Errorf("studio: historical tool result has no matching call")
			}
			delete(pending, message.ToolCallID)
		}
	}
	if len(pending) != 0 {
		return fmt.Errorf("studio: historical tool call has no result")
	}
	return nil
}
