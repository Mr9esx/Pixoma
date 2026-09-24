package einoagent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	einojsonschema "github.com/eino-contrib/jsonschema"

	studioapp "github.com/Mr9esx/Pixoma/internal/studio/application"
	"github.com/Mr9esx/Pixoma/internal/studio/application/contextcompaction"
	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

type loadSkillTool struct {
	info      *schema.ToolInfo
	skills    map[string]domain.RunSkill
	sink      studioapp.AgentSink
	maxTokens int
}

func newLoadSkillTool(skills []domain.RunSkill, sink studioapp.AgentSink) (*loadSkillTool, error) {
	var parameters einojsonschema.Schema
	if err := json.Unmarshal([]byte(`{"type":"object","additionalProperties":false,"required":["skill_id"],"properties":{"skill_id":{"type":"string","description":"当前 Run 的 Skill 目录中的 ID"}}}`), &parameters); err != nil {
		return nil, err
	}
	byID := make(map[string]domain.RunSkill, len(skills))
	for _, skill := range skills {
		byID[skill.ID] = skill
	}
	return &loadSkillTool{
		info: &schema.ToolInfo{
			Name: "load_skill", Desc: "按 Skill ID 读取当前 Run 已启用 Skill 的完整操作说明。仅在任务需要该 Skill 时调用。",
			ParamsOneOf: schema.NewParamsOneOfByJSONSchema(&parameters),
		},
		skills: byID,
		sink:   sink,
	}, nil
}

func (t *loadSkillTool) Info(context.Context) (*schema.ToolInfo, error) { return t.info, nil }

func (t *loadSkillTool) InvokableRun(ctx context.Context, arguments string, _ ...einotool.Option) (string, error) {
	var input struct {
		SkillID string `json:"skill_id"`
	}
	if err := json.Unmarshal([]byte(arguments), &input); err != nil {
		return "", fmt.Errorf("studio: invalid load_skill arguments: %w", err)
	}
	skill, ok := t.skills[strings.TrimSpace(input.SkillID)]
	if !ok {
		return "", fmt.Errorf("studio: Skill is unavailable in this Run")
	}
	output := fmt.Sprintf("Skill ID: %s\nSkill: %s\n\n%s", skill.ID, skill.Name, skill.Prompt)
	if t.maxTokens <= 0 || contextcompaction.EstimateTokens([]*schema.Message{{Role: schema.Tool, Content: output}}) > t.maxTokens {
		return "", fmt.Errorf("studio: Skill content exceeds the available model context")
	}
	toolCallID := "skill.load." + skill.ID
	if err := t.sink.Emit(ctx, studioapp.EventToolCallStart, map[string]any{"tool_call_id": toolCallID, "tool_name": t.info.Name}); err != nil {
		return "", err
	}
	if err := t.sink.Emit(ctx, studioapp.EventToolCallArgs, map[string]any{"tool_call_id": toolCallID, "delta": arguments}); err != nil {
		return "", err
	}
	if err := t.sink.Emit(ctx, studioapp.EventToolCallResult, map[string]any{"tool_call_id": toolCallID, "content": fmt.Sprintf("已读取 Skill「%s」；后续 Run 可重新调用 load_skill。", skill.Name), "is_error": false}); err != nil {
		return "", err
	}
	if err := t.sink.Emit(ctx, studioapp.EventToolCallEnd, map[string]any{"tool_call_id": toolCallID, "tool_name": t.info.Name, "is_error": false}); err != nil {
		return "", err
	}
	return output, nil
}

var _ einotool.InvokableTool = (*loadSkillTool)(nil)
