package einoagent

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	einojsonschema "github.com/eino-contrib/jsonschema"
	"github.com/google/uuid"

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

type skillArgs struct {
	SkillID string `json:"skill_id"`
	Path    string `json:"path,omitempty"`
	Offset  int    `json:"offset,omitempty"`
}

type skillReadPayload struct {
	SkillID       string     `json:"skill_id"`
	Name          string     `json:"name"`
	Path          string     `json:"path"`
	Offset        int        `json:"offset"`
	Total         int        `json:"total,omitempty"`
	Content       string     `json:"content"`
	FileCount     int        `json:"file_count,omitempty"`
	ReadableFiles []string   `json:"readable_files,omitempty"`
	Next          *skillArgs `json:"next,omitempty"`
	NextFiles     *skillArgs `json:"next_files,omitempty"`
}

func newLoadSkillTool(skills []domain.RunSkill, sink studioapp.AgentSink, locale string) (*loadSkillTool, error) {
	source := `{"type":"object","additionalProperties":false,"required":["skill_id"],"properties":{"skill_id":{"type":"string","description":"当前 Run 的 Skill ID"},"path":{"type":"string","description":"文件路径；省略时读取 SKILL.md，传入 . 时分页列出可读文本文件"},"offset":{"type":"integer","minimum":0,"description":"文件的 Unicode 字符位置；path 为 . 时表示文件目录中的序号"}}}`
	description := "按 Skill ID 分段读取 SKILL.md 与文本文件。返回 next 或 next_files 时，使用其中的参数继续读取。二进制资源与脚本不可执行。"
	if locale == "en" {
		source = `{"type":"object","additionalProperties":false,"required":["skill_id"],"properties":{"skill_id":{"type":"string","description":"Skill ID in this Run"},"path":{"type":"string","description":"Text file path; omit for SKILL.md or use . to list readable files in pages"},"offset":{"type":"integer","minimum":0,"description":"Unicode character offset, or file index when path is ."}}}`
		description = "Read SKILL.md and text files by Skill ID. Continue with returned next or next_files arguments. Scripts and binary resources cannot be executed."
	}
	var parameters einojsonschema.Schema
	if err := json.Unmarshal([]byte(source), &parameters); err != nil {
		return nil, err
	}
	byID := make(map[string]domain.RunSkill, len(skills))
	for _, skill := range skills {
		byID[skill.ID] = skill
	}
	return &loadSkillTool{
		info: &schema.ToolInfo{
			Name: "load_skill", Desc: description,
			ParamsOneOf: schema.NewParamsOneOfByJSONSchema(&parameters),
		},
		skills: byID,
		sink:   sink,
	}, nil
}

func (t *loadSkillTool) Info(context.Context) (*schema.ToolInfo, error) { return t.info, nil }

func (t *loadSkillTool) InvokableRun(ctx context.Context, arguments string, _ ...einotool.Option) (string, error) {
	var input skillArgs
	if err := json.Unmarshal([]byte(arguments), &input); err != nil {
		return "", fmt.Errorf("studio: invalid load_skill arguments: %w", err)
	}
	skill, ok := t.skills[strings.TrimSpace(input.SkillID)]
	if !ok {
		return "", fmt.Errorf("studio: Skill is unavailable in this Run")
	}
	output, err := renderSkillRead(skill, input, t.maxTokens)
	if err != nil {
		return "", err
	}
	toolCallID := "skill.load." + uuid.NewString()
	if err := t.sink.Emit(ctx, studioapp.EventToolCallStart, map[string]any{"tool_call_id": toolCallID, "tool_name": t.info.Name}); err != nil {
		return "", err
	}
	if err := t.sink.Emit(ctx, studioapp.EventToolCallArgs, map[string]any{"tool_call_id": toolCallID, "delta": arguments}); err != nil {
		return "", err
	}
	if err := t.sink.Emit(ctx, studioapp.EventToolCallResult, map[string]any{"tool_call_id": toolCallID, "content": fmt.Sprintf("已读取 Skill「%s」的 %s。", skill.Name, skillReadPath(input.Path)), "is_error": false}); err != nil {
		return "", err
	}
	if err := t.sink.Emit(ctx, studioapp.EventToolCallEnd, map[string]any{"tool_call_id": toolCallID, "tool_name": t.info.Name, "is_error": false}); err != nil {
		return "", err
	}
	if err := t.sink.Emit(ctx, studioapp.EventContextInjected, map[string]any{
		"source": "skill", "detail": skill.Name, "skill_id": skill.ID, "path": skillReadPath(input.Path),
		"content":                output,
		"delta_tokens_estimated": contextcompaction.EstimateTokens([]*schema.Message{{Role: schema.Tool, Content: output}}),
	}); err != nil {
		return "", err
	}
	return output, nil
}

func skillReadPath(path string) string {
	if path == "" {
		return "SKILL.md"
	}
	if path == "." {
		return "文件目录"
	}
	return path
}

func renderSkillRead(skill domain.RunSkill, input skillArgs, maxTokens int) (string, error) {
	if input.SkillID != skill.ID || input.Offset < 0 {
		return "", fmt.Errorf("studio: Skill 读取参数无效")
	}
	if maxTokens <= 0 {
		return "", fmt.Errorf("studio: Skill 读取超出当前模型的上下文")
	}
	limit := min(maxTokens, 2048)
	files := make(map[string]domain.SkillFile, len(skill.Files))
	paths := make([]string, 0, len(skill.Files))
	if len(skill.Files) == 0 {
		files["SKILL.md"] = domain.SkillFile{Path: "SKILL.md", Content: skill.Prompt}
		paths = append(paths, "SKILL.md")
	} else {
		for _, file := range skill.Files {
			files[file.Path] = file
			if !file.Binary && !file.Directory {
				paths = append(paths, file.Path)
			}
		}
		sort.Slice(paths, func(i, j int) bool {
			if paths[i] == "SKILL.md" {
				return true
			}
			if paths[j] == "SKILL.md" {
				return false
			}
			return paths[i] < paths[j]
		})
	}
	path := input.Path
	if path == "" {
		path = "SKILL.md"
	}
	response := skillReadPayload{SkillID: skill.ID, Name: skill.Name, Path: path, Offset: input.Offset, Content: ""}
	if input.Path == "." {
		if input.Offset > len(paths) {
			return "", fmt.Errorf("studio: Skill 文件目录位置无效")
		}
		response.FileCount = len(paths)
		for index := input.Offset; index < len(paths); index++ {
			candidate := response
			candidate.ReadableFiles = append(append([]string(nil), response.ReadableFiles...), paths[index])
			if index+1 < len(paths) {
				candidate.Next = &skillArgs{SkillID: skill.ID, Path: ".", Offset: index + 1}
			} else {
				candidate.Next = nil
			}
			if _, fits, err := encodeSkillRead(candidate, limit); err != nil {
				return "", err
			} else if !fits {
				break
			}
			response = candidate
		}
		if input.Offset < len(paths) && len(response.ReadableFiles) == 0 {
			return "", fmt.Errorf("studio: Skill 文件路径超出当前模型的上下文")
		}
		output, fits, err := encodeSkillRead(response, limit)
		if err != nil {
			return "", err
		}
		if !fits {
			return "", fmt.Errorf("studio: Skill 文件目录超出当前模型的上下文")
		}
		return output, nil
	}
	file, exists := files[path]
	if !exists {
		return "", fmt.Errorf("studio: Skill 文件 %q 不存在", path)
	}
	if file.Binary {
		return "", fmt.Errorf("studio: Skill 二进制文件 %q 不可读取", path)
	}
	if file.Directory {
		return "", fmt.Errorf("studio: Skill 路径 %q 是文件夹", path)
	}
	runes := []rune(file.Content)
	if input.Offset > len(runes) {
		return "", fmt.Errorf("studio: Skill 文件字符位置无效")
	}
	response.Total = len(runes)
	if input.Path == "" {
		response.FileCount = len(paths)
		response.NextFiles = &skillArgs{SkillID: skill.ID, Path: "."}
		listLimit := min(limit-20, max(120, limit/3))
		for index, candidatePath := range paths {
			candidate := response
			candidate.ReadableFiles = append(append([]string(nil), response.ReadableFiles...), candidatePath)
			if index+1 < len(paths) {
				candidate.NextFiles = &skillArgs{SkillID: skill.ID, Path: ".", Offset: index + 1}
			} else {
				candidate.NextFiles = nil
			}
			if _, fits, err := encodeSkillRead(candidate, listLimit); err != nil {
				return "", err
			} else if !fits {
				break
			}
			response = candidate
		}
	}
	start := input.Offset
	end := len(runes)
	for start < end {
		middle := (start + end + 1) / 2
		candidate := response
		candidate.Content = string(runes[input.Offset:middle])
		if middle < len(runes) {
			candidate.Next = &skillArgs{SkillID: skill.ID, Path: path, Offset: middle}
		}
		if _, fits, err := encodeSkillRead(candidate, limit); err != nil {
			return "", err
		} else if fits {
			start = middle
		} else {
			end = middle - 1
		}
	}
	if start > input.Offset {
		response.Content = string(runes[input.Offset:start])
		if start < len(runes) {
			response.Next = &skillArgs{SkillID: skill.ID, Path: path, Offset: start}
		}
	} else if input.Offset < len(runes) {
		return "", fmt.Errorf("studio: Skill 文件内容超出当前模型的上下文")
	}
	output, fits, err := encodeSkillRead(response, limit)
	if err != nil {
		return "", err
	}
	if !fits {
		return "", fmt.Errorf("studio: Skill 文件内容超出当前模型的上下文")
	}
	return output, nil
}

func encodeSkillRead(response skillReadPayload, maxTokens int) (string, bool, error) {
	encoded, err := json.Marshal(response)
	if err != nil {
		return "", false, err
	}
	output := string(encoded)
	return output, contextcompaction.EstimateTokens([]*schema.Message{{Role: schema.Tool, Content: output}}) <= maxTokens, nil
}

var _ einotool.InvokableTool = (*loadSkillTool)(nil)
