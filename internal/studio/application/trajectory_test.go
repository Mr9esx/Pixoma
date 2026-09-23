package application

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

func TestProjectRunTrajectoryRequestDetailsAndUnknownUsage(t *testing.T) {
	now := time.Date(2026, 9, 22, 10, 0, 0, 0, time.UTC)
	run := &domain.Run{ID: "run-1"}
	event := func(id string, seq int64, eventType string, payload string) *domain.Event {
		return &domain.Event{ID: id, RunID: run.ID, Sequence: uint64(seq), Type: eventType, Payload: json.RawMessage(payload), CreatedAt: now.Add(time.Duration(seq) * time.Second)}
	}
	events := []*domain.Event{
		event("start", 1, EventModelRequestStarted, `{"attempt_id":"a1","model":"test","request_body":{"messages":[{"role":"user","content":"hello"}],"tools":[{"type":"function","function":{"name":"search","parameters":{"type":"object"}}}]}}`),
		event("first", 2, EventModelFirstToken, `{"attempt_id":"a1","elapsed_ms":45}`),
		event("finish", 3, EventModelRequestFinished, `{"attempt_id":"a1","elapsed_ms":125,"response_body":{"choices":[{"message":{"content":"world"}}]}}`),
		event("tool", 4, EventToolCallStart, `{"tool_call_id":"t1","tool_name":"search"}`),
		event("args", 5, EventToolCallArgs, `{"tool_call_id":"t1","delta":"{\"q\":\"x\"}"}`),
		event("result", 6, EventToolCallResult, `{"tool_call_id":"t1","content":"found","is_error":false}`),
	}
	rows, details := ProjectRunTrajectory(run, nil, events)
	if len(rows) != 2 || rows[0].Kind != "model" || rows[1].ParentID != "start" {
		t.Fatalf("rows = %#v", rows)
	}
	model := details["start"]
	if model.Usage != nil || model.Timing["ttft_ms"] != int64(45) || model.Output == nil {
		t.Fatalf("model detail = %#v", model)
	}
	if details["tool"].Input != `{"q":"x"}` || details["tool"].Output != "found" || details["tool"].Schema == nil {
		t.Fatalf("tool detail = %#v", details["tool"])
	}
}

func TestProjectRunTrajectoryCompaction(t *testing.T) {
	run := &domain.Run{ID: "run-1"}
	rows, details := ProjectRunTrajectory(run, nil, []*domain.Event{{
		ID: "compact", Type: EventContextCompacted,
		Payload: json.RawMessage(`{"summary":"历史摘要","before_tokens_estimated":100,"after_tokens_estimated":30,"retained_from":2,"auto_compact":true}`),
	}})
	if len(rows) != 1 || rows[0].Kind != "compaction" || details["compact"].Usage["source"] != "estimate" {
		t.Fatalf("compaction = %#v %#v", rows, details)
	}
}

func TestProjectRunTrajectoryKeepsRetryInSameStep(t *testing.T) {
	run := &domain.Run{ID: "run-1"}
	events := []*domain.Event{
		{ID: "first", Sequence: 1, Type: EventModelRequestStarted, Payload: json.RawMessage(`{"attempt_id":"a1"}`)},
		{ID: "failure", Sequence: 2, Type: EventModelRequestFailed, Payload: json.RawMessage(`{"attempt_id":"a1","error":"rate limit"}`)},
		{ID: "second", Sequence: 3, Type: EventModelRequestStarted, Payload: json.RawMessage(`{"attempt_id":"a2"}`)},
		{ID: "success", Sequence: 4, Type: EventModelRequestFinished, Payload: json.RawMessage(`{"attempt_id":"a2","input_tokens":0,"output_tokens":0}`)},
	}
	rows, details := ProjectRunTrajectory(run, nil, events)
	if len(rows) != 2 || rows[0].Status != "failed" || rows[0].Step != 1 || rows[1].Step != 1 || rows[1].Attempt != 2 || details["second"].Usage["input_tokens"] != int64(0) {
		t.Fatalf("retry = %#v %#v", rows, details)
	}
}
