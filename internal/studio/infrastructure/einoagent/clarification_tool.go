package einoagent

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"

	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	einojsonschema "github.com/eino-contrib/jsonschema"

	studioapp "github.com/Mr9esx/Pixoma/internal/studio/application"
	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

type askClarificationTool struct {
	info           *schema.ToolInfo
	sink           studioapp.AgentSink
	clarifications map[string]*domain.Clarification
	waiting        atomic.Bool
}

func newAskClarificationTool(sink studioapp.AgentSink, clarifications []*domain.Clarification, locale string) (*askClarificationTool, error) {
	source := `{"type":"object","additionalProperties":false,"required":["question","options"],"properties":{"question":{"type":"string","description":"需要用户回答的明确问题"},"options":{"type":"array","minItems":2,"maxItems":5,"items":{"type":"string"},"description":"互不重复的单选选项，不包含其他"}}}`
	description := "在关键条件不清楚且会影响创作结果时，提出一道单选问题并等待用户回答。"
	if locale == "en" {
		source = `{"type":"object","additionalProperties":false,"required":["question","options"],"properties":{"question":{"type":"string","description":"A specific question for the user"},"options":{"type":"array","minItems":2,"maxItems":5,"items":{"type":"string"},"description":"Distinct multiple-choice options, without Other"}}}`
		description = "Ask one multiple-choice question when a missing key detail would affect the result."
	}
	var parameters einojsonschema.Schema
	if err := json.Unmarshal([]byte(source), &parameters); err != nil {
		return nil, err
	}
	byID := make(map[string]*domain.Clarification, len(clarifications))
	for _, clarification := range clarifications {
		byID[clarification.ID] = clarification
	}
	return &askClarificationTool{
		info: &schema.ToolInfo{
			Name:        "ask_clarification",
			Desc:        description,
			ParamsOneOf: schema.NewParamsOneOfByJSONSchema(&parameters),
		},
		sink: sink, clarifications: byID,
	}, nil
}

func (t *askClarificationTool) Info(context.Context) (*schema.ToolInfo, error) { return t.info, nil }

func (t *askClarificationTool) InvokableRun(ctx context.Context, arguments string, _ ...einotool.Option) (string, error) {
	wasInterrupted, hasState, clarificationID := einotool.GetInterruptState[string](ctx)
	if wasInterrupted {
		if !hasState {
			return "", fmt.Errorf("studio: clarification interrupt has no state")
		}
		clarification, ok := t.clarifications[clarificationID]
		if !ok {
			return "", fmt.Errorf("studio: clarification %s is missing", clarificationID)
		}
		if clarification.Status == domain.ClarificationAnswered {
			return clarification.Answer, nil
		}
		if clarification.Status == domain.ClarificationSkipped {
			return "用户跳过了该问题。请使用已有信息继续。", nil
		}
		t.waiting.Store(true)
		return "", einotool.StatefulInterrupt(ctx, clarification.Question, clarificationID)
	}
	var input struct {
		Question string   `json:"question"`
		Options  []string `json:"options"`
	}
	if err := json.Unmarshal([]byte(arguments), &input); err != nil {
		return "", fmt.Errorf("studio: invalid ask_clarification arguments: %w", err)
	}
	sink, ok := t.sink.(studioapp.ClarificationSink)
	if !ok {
		return "", fmt.Errorf("studio: clarification sink is not configured")
	}
	clarification, err := sink.RequestClarification(ctx, input.Question, input.Options)
	if err != nil {
		return "", err
	}
	t.waiting.Store(true)
	return "", einotool.StatefulInterrupt(ctx, clarification.Question, clarification.ID)
}

var _ einotool.InvokableTool = (*askClarificationTool)(nil)
