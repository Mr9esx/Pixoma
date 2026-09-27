package einoagent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	einojsonschema "github.com/eino-contrib/jsonschema"
	"github.com/google/uuid"

	studioapp "github.com/Mr9esx/Pixoma/internal/studio/application"
)

type installSkillEventSink interface {
	Emit(context.Context, string, any) error
}

type installSkillTool struct {
	info      *schema.ToolInfo
	accountID string
	creator   studioapp.SkillCreator
	sink      installSkillEventSink
}

func newInstallSkillTool(accountID string, creator studioapp.SkillCreator, sink installSkillEventSink) (*installSkillTool, error) {
	if strings.TrimSpace(accountID) == "" || creator == nil || sink == nil {
		return nil, fmt.Errorf("studio: Skill installer dependencies are required")
	}
	info, err := installSkillToolInfo()
	if err != nil {
		return nil, err
	}
	return &installSkillTool{info: info, accountID: accountID, creator: creator, sink: sink}, nil
}

func installSkillToolInfo() (*schema.ToolInfo, error) {
	var parameters einojsonschema.Schema
	if err := json.Unmarshal([]byte(`{"type":"object","additionalProperties":false,"required":["github_url"],"properties":{"github_url":{"type":"string","description":"公开 GitHub Skill 的 blob 或 tree 链接"}}}`), &parameters); err != nil {
		return nil, err
	}
	return &schema.ToolInfo{
		Name:        "install_skill",
		Desc:        "仅在用户明确要求时，从公开 GitHub 链接安装 Skill 到当前账户，供后续 Run 选择。",
		ParamsOneOf: schema.NewParamsOneOfByJSONSchema(&parameters),
	}, nil
}

func (t *installSkillTool) Info(context.Context) (*schema.ToolInfo, error) { return t.info, nil }

func (t *installSkillTool) InvokableRun(ctx context.Context, arguments string, _ ...einotool.Option) (string, error) {
	var input struct {
		GitHubURL string `json:"github_url"`
	}
	if err := json.Unmarshal([]byte(arguments), &input); err != nil {
		return "", fmt.Errorf("studio: invalid install_skill arguments: %w", err)
	}
	input.GitHubURL = strings.TrimSpace(input.GitHubURL)
	if input.GitHubURL == "" {
		return "", fmt.Errorf("studio: GitHub Skill URL is required")
	}
	toolCallID := "skill.install." + uuid.NewString()
	if err := t.sink.Emit(ctx, studioapp.EventToolCallStart, map[string]any{"tool_call_id": toolCallID, "tool_name": t.info.Name}); err != nil {
		return "", err
	}
	if err := t.sink.Emit(ctx, studioapp.EventToolCallArgs, map[string]any{"tool_call_id": toolCallID, "delta": arguments}); err != nil {
		return "", err
	}
	finishWithError := func(cause error) (string, error) {
		if err := t.sink.Emit(ctx, studioapp.EventToolCallResult, map[string]any{"tool_call_id": toolCallID, "content": cause.Error(), "is_error": true}); err != nil {
			return "", err
		}
		if err := t.sink.Emit(ctx, studioapp.EventToolCallEnd, map[string]any{"tool_call_id": toolCallID, "tool_name": t.info.Name, "is_error": true}); err != nil {
			return "", err
		}
		return "", cause
	}
	source, err := parseGitHubSkillURL(input.GitHubURL)
	if err != nil {
		return finishWithError(err)
	}
	installed, err := fetchGitHubSkill(ctx, source)
	if err != nil {
		return finishWithError(err)
	}
	skill, err := t.creator.CreateSkill(ctx, studioapp.CreateSkillInput{
		AccountID: t.accountID, Name: installed.Name, Description: installed.Description, Prompt: installed.Prompt, Enabled: true,
	})
	if err != nil {
		return finishWithError(err)
	}
	result := fmt.Sprintf("已安装 Skill「%s」，后续 Run 可在聊天栏选择。", skill.Name)
	if err := t.sink.Emit(ctx, studioapp.EventToolCallResult, map[string]any{"tool_call_id": toolCallID, "content": result, "is_error": false}); err != nil {
		return "", err
	}
	if err := t.sink.Emit(ctx, studioapp.EventToolCallEnd, map[string]any{"tool_call_id": toolCallID, "tool_name": t.info.Name, "is_error": false}); err != nil {
		return "", err
	}
	return result, nil
}

var _ einotool.InvokableTool = (*installSkillTool)(nil)
