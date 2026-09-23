package persistence

import (
	"encoding/json"
)

// The list endpoint never needs provider bodies or full tool/message output.
// Keep this projection alongside the append-only source event so reading
// session-wide trajectory rows does not transfer raw prompts and responses.
func trajectorySummaryPayload(eventType string, source json.RawMessage) []byte {
	var payload map[string]json.RawMessage
	if json.Unmarshal(source, &payload) != nil {
		return []byte("{}")
	}
	keys := []string{}
	switch eventType {
	case "MODEL_REQUEST_STARTED":
		keys = []string{"attempt_id", "model", "purpose", "protocol"}
	case "MODEL_FIRST_TOKEN":
		keys = []string{"attempt_id", "elapsed_ms"}
	case "MODEL_REQUEST_FINISHED", "MODEL_REQUEST_FAILED":
		keys = []string{"attempt_id", "elapsed_ms", "status_code", "provider_request_id", "input_tokens", "output_tokens", "error"}
	case "TOOL_CALL_START":
		keys = []string{"tool_call_id", "tool_name"}
	case "TOOL_CALL_ARGS":
		keys = []string{"tool_call_id", "delta"}
	case "TOOL_CALL_RESULT":
		keys = []string{"tool_call_id", "content", "is_error"}
	case "TOOL_CALL_END":
		keys = []string{"tool_call_id", "is_error"}
	case "TEXT_MESSAGE_START", "REASONING_MESSAGE_START":
		keys = []string{"message_id"}
	case "TEXT_MESSAGE_CONTENT", "REASONING_MESSAGE_CONTENT":
		keys = []string{"message_id", "delta"}
	case "TEXT_MESSAGE_END", "REASONING_MESSAGE_END":
		keys = []string{"message_id", "content"}
	case "CONTEXT_COMPACTED":
		keys = []string{"summary", "before_tokens_estimated", "after_tokens_estimated", "retained_from", "auto_compact"}
	}
	selected := make(map[string]json.RawMessage, len(keys))
	for _, key := range keys {
		raw, ok := payload[key]
		if !ok {
			continue
		}
		var value string
		if json.Unmarshal(raw, &value) == nil {
			limit := 256
			if key == "delta" || key == "content" || key == "summary" || key == "error" {
				limit = 120
			}
			chars := []rune(value)
			if len(chars) > limit {
				raw, _ = json.Marshal(string(chars[:limit]) + "…")
			}
		}
		selected[key] = raw
	}
	encoded, err := json.Marshal(selected)
	if err != nil {
		return []byte("{}")
	}
	return encoded
}
