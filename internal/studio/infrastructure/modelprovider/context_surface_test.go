package modelprovider

import (
	"encoding/json"
	"testing"
)

func TestTraceUsageNormalizesProviderCacheFields(t *testing.T) {
	cases := []struct {
		name, body                            string
		input, output, read, write, reasoning int
		readKnown, writeKnown, reasoningKnown bool
	}{
		{"openai_chat", `{"usage":{"prompt_tokens":100,"completion_tokens":20,"prompt_tokens_details":{"cached_tokens":70},"completion_tokens_details":{"reasoning_tokens":8}}}`, 100, 20, 70, 0, 8, true, false, true},
		{"deepseek", `{"usage":{"prompt_tokens":100,"completion_tokens":20,"prompt_cache_hit_tokens":60,"prompt_cache_miss_tokens":40}}`, 100, 20, 60, 0, 0, true, false, false},
		{"responses", `{"usage":{"input_tokens":90,"output_tokens":30,"input_tokens_details":{"cached_tokens":50},"output_tokens_details":{"reasoning_tokens":12}}}`, 90, 30, 50, 0, 12, true, false, true},
		{"anthropic", `{"usage":{"input_tokens":20,"output_tokens":10,"cache_read_input_tokens":65,"cache_creation_input_tokens":15}}`, 100, 10, 65, 15, 0, true, true, false},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			usage, err := traceUsage([]byte(test.body))
			if err != nil {
				t.Fatal(err)
			}
			if !usage.reported || usage.input != test.input || usage.output != test.output {
				t.Fatalf("usage = %#v", usage)
			}
			if (usage.cacheRead != nil) != test.readKnown || test.readKnown && *usage.cacheRead != test.read {
				t.Fatalf("cache read = %#v", usage.cacheRead)
			}
			if (usage.cacheWrite != nil) != test.writeKnown || test.writeKnown && *usage.cacheWrite != test.write {
				t.Fatalf("cache write = %#v", usage.cacheWrite)
			}
			if (usage.reasoning != nil) != test.reasoningKnown || test.reasoningKnown && *usage.reasoning != test.reasoning {
				t.Fatalf("reasoning = %#v", usage.reasoning)
			}
		})
	}
}

func TestAnalyzeRequestKeepsSkillSourceAndRunInjection(t *testing.T) {
	body := json.RawMessage(`{"model":"test","messages":[{"role":"system","content":"system"},{"role":"user","content":"<available_workflows>none</available_workflows>\n\n请继续"},{"role":"assistant","tool_calls":[{"id":"call-1","type":"function","function":{"name":"load_skill","arguments":"{}"}}]},{"role":"tool","tool_call_id":"call-1","content":"Skill instructions"}],"tools":[{"type":"function","function":{"name":"load_skill"}}]}`)
	parts, err := AnalyzeRequest(body, "<available_workflows>none</available_workflows>")
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, part := range parts {
		counts[part.Category]++
	}
	if counts["system_prompt"] != 1 || counts["tool_definition"] != 1 || counts["injection"] != 1 || counts["user_message"] != 1 || counts["skill_injection"] != 1 {
		t.Fatalf("categories = %#v, parts = %#v", counts, parts)
	}
}
