package application

import (
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

// TrajectoryRecord is a bounded row in the session trajectory. Full request
// and result bodies are available only from the record detail endpoint.
type TrajectoryRecord struct {
	ID        string     `json:"id"`
	RunID     string     `json:"run_id"`
	Kind      string     `json:"kind"`
	Step      int        `json:"step,omitempty"`
	Attempt   int        `json:"attempt,omitempty"`
	ParentID  string     `json:"parent_id,omitempty"`
	Title     string     `json:"title"`
	Preview   string     `json:"preview,omitempty"`
	Status    string     `json:"status"`
	StartedAt time.Time  `json:"started_at"`
	EndedAt   *time.Time `json:"ended_at,omitempty"`
}

type TrajectoryDetail struct {
	Record   TrajectoryRecord `json:"record"`
	Overview map[string]any   `json:"overview"`
	Input    any              `json:"input,omitempty"`
	Output   any              `json:"output,omitempty"`
	Raw      any              `json:"raw,omitempty"`
	Schema   any              `json:"schema,omitempty"`
	Usage    map[string]any   `json:"usage,omitempty"`
	Timing   map[string]any   `json:"timing,omitempty"`
}

type TrajectoryRun struct {
	Run     *domain.Run        `json:"run"`
	Records []TrajectoryRecord `json:"records"`
}

type trajectoryFold struct {
	rows    []TrajectoryRecord
	details map[string]TrajectoryDetail
	index   map[string]int
}

func newTrajectoryFold() *trajectoryFold {
	return &trajectoryFold{details: make(map[string]TrajectoryDetail), index: make(map[string]int)}
}

func (f *trajectoryFold) start(row TrajectoryRecord, detail TrajectoryDetail) {
	if _, ok := f.index[row.ID]; ok {
		return
	}
	f.index[row.ID] = len(f.rows)
	f.rows = append(f.rows, row)
	detail.Record = row
	f.details[row.ID] = detail
}

func (f *trajectoryFold) update(id string, apply func(*TrajectoryRecord, *TrajectoryDetail)) {
	index, ok := f.index[id]
	if !ok {
		return
	}
	row := &f.rows[index]
	detail := f.details[id]
	apply(row, &detail)
	detail.Record = *row
	f.details[id] = detail
}

// ProjectRunTrajectory folds the durable events of one run. Missing terminal
// events leave a record running, preserving crash and cancellation evidence.
func ProjectRunTrajectory(run *domain.Run, trigger *domain.Message, events []*domain.Event) ([]TrajectoryRecord, map[string]TrajectoryDetail) {
	fold := newTrajectoryFold()
	if run == nil {
		return fold.rows, fold.details
	}
	if trigger != nil {
		text, _ := messageText(trigger.ContentJSON)
		fold.start(TrajectoryRecord{
			ID: trigger.ID, RunID: run.ID, Kind: "user", Title: "用户输入",
			Preview: tracePreview(text), Status: "done", StartedAt: trigger.CreatedAt,
		}, TrajectoryDetail{Input: text, Raw: json.RawMessage(trigger.ContentJSON), Overview: map[string]any{"role": "user"}})
	}
	ordered := append([]*domain.Event(nil), events...)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].Sequence < ordered[j].Sequence })
	ids := make(map[string]string)
	toolSchemas := make(map[string]any)
	currentStep, nextStep := 0, 0
	activeModelID, lastModelID := "", ""
	for _, event := range ordered {
		if event == nil {
			continue
		}
		payload := decodeTranscriptPayload(event.Payload)
		switch event.Type {
		case EventModelRequestStarted:
			if activeModelID == "" {
				nextStep++
				currentStep = nextStep
			}
			attemptID := payload.stringValue("attempt_id")
			if attemptID == "" {
				attemptID = event.ID
			}
			id := event.ID
			ids["model:"+attemptID] = id
			activeModelID = id
			lastModelID = id
			title := payload.stringValue("model")
			if title == "" {
				title = "模型请求"
			}
			var raw any
			if json.Unmarshal(payload["request_body"], &raw) != nil {
				raw = nil
			}
			collectTraceToolSchemas(raw, toolSchemas)
			attempt := 1
			for _, row := range fold.rows {
				if row.Kind == "model" && row.Step == currentStep {
					attempt++
				}
			}
			fold.start(TrajectoryRecord{ID: id, RunID: run.ID, Kind: "model", Step: currentStep, Attempt: attempt, Title: title, Status: "running", StartedAt: event.CreatedAt}, TrajectoryDetail{
				Overview: map[string]any{"purpose": payload.stringValue("purpose"), "protocol": payload.stringValue("protocol"), "model": title},
				Input:    raw, Raw: raw,
				Timing: map[string]any{"started_at": event.CreatedAt},
			})
		case EventModelFirstToken, EventModelRequestFinished, EventModelRequestFailed:
			id := ids["model:"+payload.stringValue("attempt_id")]
			fold.update(id, func(row *TrajectoryRecord, detail *TrajectoryDetail) {
				if detail.Timing == nil {
					detail.Timing = make(map[string]any)
				}
				if event.Type == EventModelFirstToken {
					detail.Timing["first_token_at"] = event.CreatedAt
					detail.Timing["ttft_ms"] = payload.numberValue("elapsed_ms")
					return
				}
				end := event.CreatedAt
				row.EndedAt = &end
				detail.Timing["ended_at"] = end
				detail.Timing["duration_ms"] = payload.numberValue("elapsed_ms")
				detail.Overview["status_code"] = payload.numberValue("status_code")
				detail.Overview["provider_request_id"] = payload.stringValue("provider_request_id")
				if event.Type == EventModelRequestFailed {
					row.Status = "failed"
					detail.Overview["error"] = payload.stringValue("error")
					row.Preview = tracePreview(payload.stringValue("error"))
					if raw, ok := payload["response_body"]; ok {
						var value any
						if json.Unmarshal(raw, &value) == nil {
							detail.Output = value
							detail.Raw = map[string]any{"request": detail.Input, "response": value}
						}
					}
					return
				}
				row.Status = "done"
				if len(payload["input_tokens"]) > 0 || len(payload["output_tokens"]) > 0 {
					detail.Usage = map[string]any{"input_tokens": payload.optionalNumberValue("input_tokens"), "output_tokens": payload.optionalNumberValue("output_tokens"), "source": "provider"}
				}
				if output, ok := payload["response_body"]; ok {
					var value any
					if json.Unmarshal(output, &value) == nil {
						detail.Output = value
						detail.Raw = map[string]any{"request": detail.Input, "response": value}
					}
				}
			})
			if event.Type == EventModelRequestFinished {
				activeModelID = ""
			}
			// A failed request may be retried in the same step.
		case EventToolCallStart:
			key := payload.stringValue("tool_call_id")
			if key == "" {
				key = event.ID
			}
			id := event.ID
			ids["tool:"+key] = id
			name := payload.stringValue("tool_name")
			if name == "" {
				name = "工具调用"
			}
			fold.start(TrajectoryRecord{ID: id, RunID: run.ID, Kind: "tool", Step: currentStep, ParentID: lastModelID, Title: name, Status: "running", StartedAt: event.CreatedAt}, TrajectoryDetail{
				Overview: map[string]any{"tool_call_id": key, "name": name},
				Schema:   toolSchemas[name],
				Timing:   map[string]any{"started_at": event.CreatedAt},
			})
		case EventToolCallArgs:
			fold.update(ids["tool:"+payload.stringValue("tool_call_id")], func(_ *TrajectoryRecord, detail *TrajectoryDetail) {
				previous, _ := detail.Input.(string)
				detail.Input = previous + payload.stringValue("delta")
			})
		case EventToolCallResult:
			fold.update(ids["tool:"+payload.stringValue("tool_call_id")], func(row *TrajectoryRecord, detail *TrajectoryDetail) {
				output := payload.stringValue("content")
				detail.Output = output
				row.Preview = tracePreview(output)
				if payload.boolValue("is_error") {
					row.Status = "failed"
					detail.Overview["is_error"] = true
				}
			})
		case EventToolCallEnd:
			fold.update(ids["tool:"+payload.stringValue("tool_call_id")], func(row *TrajectoryRecord, detail *TrajectoryDetail) {
				end := event.CreatedAt
				row.EndedAt = &end
				if row.Status != "failed" {
					row.Status = "done"
				}
				detail.Timing["ended_at"] = end
				detail.Timing["duration_ms"] = end.Sub(row.StartedAt).Milliseconds()
				detail.Raw = map[string]any{"input": detail.Input, "output": detail.Output}
			})
		case EventTextMessageStart, EventReasoningMessageStart:
			key := payload.stringValue("message_id")
			if key == "" {
				key = event.ID
			}
			kind := "assistant"
			title := "助手输出"
			if event.Type == EventReasoningMessageStart {
				kind, title = "reasoning", "思考"
			}
			ids[kind+":"+key] = event.ID
			fold.start(TrajectoryRecord{ID: event.ID, RunID: run.ID, Kind: kind, Step: currentStep, Title: title, ParentID: lastModelID, Status: "running", StartedAt: event.CreatedAt}, TrajectoryDetail{
				Overview: map[string]any{"message_id": key}, Timing: map[string]any{"started_at": event.CreatedAt},
			})
		case EventTextMessageContent, EventReasoningMessageContent:
			kind := "assistant"
			if event.Type == EventReasoningMessageContent {
				kind = "reasoning"
			}
			fold.update(ids[kind+":"+payload.stringValue("message_id")], func(row *TrajectoryRecord, detail *TrajectoryDetail) {
				source, _ := detail.Output.(string)
				source += payload.stringValue("delta")
				detail.Output = source
				detail.Raw = source
				row.Preview = tracePreview(source)
			})
		case EventTextMessageEnd, EventReasoningMessageEnd:
			kind := "assistant"
			if event.Type == EventReasoningMessageEnd {
				kind = "reasoning"
			}
			fold.update(ids[kind+":"+payload.stringValue("message_id")], func(row *TrajectoryRecord, detail *TrajectoryDetail) {
				end := event.CreatedAt
				row.EndedAt = &end
				row.Status = "done"
				detail.Timing["ended_at"] = end
				detail.Timing["duration_ms"] = end.Sub(row.StartedAt).Milliseconds()
				if content := payload.stringValue("content"); content != "" {
					detail.Output, detail.Raw = content, content
					row.Preview = tracePreview(content)
				}
			})
		case EventContextCompacted:
			fold.start(TrajectoryRecord{ID: event.ID, RunID: run.ID, Kind: "compaction", Step: currentStep, Title: "上下文压缩", Preview: tracePreview(payload.stringValue("summary")), Status: "done", StartedAt: event.CreatedAt}, TrajectoryDetail{
				Overview: map[string]any{"retained_from": payload.numberValue("retained_from"), "auto_compact": payload.boolValue("auto_compact")},
				Output:   payload.stringValue("summary"),
				Usage:    map[string]any{"before_tokens_estimated": payload.numberValue("before_tokens_estimated"), "after_tokens_estimated": payload.numberValue("after_tokens_estimated"), "source": "estimate"},
				Raw:      json.RawMessage(event.Payload),
			})
		case EventRunStarted, EventRunFinished, EventReasoningStart, EventReasoningEnd:
			continue
		default:
			id := event.ID
			title := strings.ReplaceAll(strings.ToLower(event.Type), "_", " ")
			fold.start(TrajectoryRecord{ID: id, RunID: run.ID, Kind: "event", Step: currentStep, Title: title, Status: "done", StartedAt: event.CreatedAt}, TrajectoryDetail{
				Overview: map[string]any{"type": event.Type}, Raw: json.RawMessage(event.Payload),
			})
		}
	}
	return fold.rows, fold.details
}

func tracePreview(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	runes := []rune(value)
	if len(runes) > 120 {
		return string(runes[:120]) + "…"
	}
	return value
}

func collectTraceToolSchemas(body any, schemas map[string]any) {
	request, ok := body.(map[string]any)
	if !ok {
		return
	}
	tools, ok := request["tools"].([]any)
	if !ok {
		return
	}
	for _, item := range tools {
		tool, ok := item.(map[string]any)
		if !ok {
			continue
		}
		definition := tool
		if function, ok := tool["function"].(map[string]any); ok {
			definition = function
		}
		if name, ok := definition["name"].(string); ok && name != "" {
			schemas[name] = definition
		}
	}
}

func (p transcriptPayload) numberValue(key string) int64 {
	var value int64
	_ = json.Unmarshal(p[key], &value)
	return value
}

func (p transcriptPayload) optionalNumberValue(key string) any {
	if len(p[key]) == 0 {
		return nil
	}
	return p.numberValue(key)
}
