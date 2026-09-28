package modelprovider

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/schema"

	"github.com/Mr9esx/Pixoma/internal/studio/application/contextcompaction"
)

type ContextPart struct {
	ID              string `json:"id"`
	Category        string `json:"category"`
	Source          string `json:"source"`
	Label           string `json:"label"`
	Content         string `json:"content,omitempty"`
	EstimatedTokens int    `json:"estimated_tokens"`
}

func AnalyzeRequest(body json.RawMessage, runContext ...string) ([]ContextPart, error) {
	var request map[string]json.RawMessage
	if err := json.Unmarshal(body, &request); err != nil {
		return nil, fmt.Errorf("model provider: decode context request: %w", err)
	}
	parts := make([]ContextPart, 0)
	add := func(id, category, source, label, content string) {
		if strings.TrimSpace(content) == "" {
			return
		}
		parts = append(parts, ContextPart{ID: id, Category: category, Source: source, Label: label, Content: content,
			EstimatedTokens: EstimateContextText(content)})
	}
	addMessage := func(id, category, source, label, content string) {
		if category == "user_message" && strings.Contains(content, "<session_context_summary>") {
			add(id, "injection", "context_summary", "上下文摘要", content)
			return
		}
		if category == "user_message" && len(runContext) > 0 && runContext[0] != "" && strings.HasPrefix(content, runContext[0]+"\n\n") {
			add(id+"/injection", "injection", "run_context", "会话运行上下文", runContext[0])
			content = strings.TrimPrefix(content, runContext[0]+"\n\n")
		}
		add(id, category, source, label, content)
	}
	if raw := request["system"]; len(raw) > 0 {
		var system string
		if err := json.Unmarshal(raw, &system); err != nil {
			return nil, fmt.Errorf("model provider: decode system context: %w", err)
		}
		add("system", "system_prompt", "system", "系统提示词", system)
	}
	if raw := request["tools"]; len(raw) > 0 {
		var tools []json.RawMessage
		if err := json.Unmarshal(raw, &tools); err != nil {
			return nil, fmt.Errorf("model provider: decode tool context: %w", err)
		}
		for index, tool := range tools {
			var named struct {
				Name     string `json:"name"`
				Function struct {
					Name string `json:"name"`
				} `json:"function"`
			}
			if err := json.Unmarshal(tool, &named); err != nil {
				return nil, err
			}
			name := named.Name
			if name == "" {
				name = named.Function.Name
			}
			add(fmt.Sprintf("tools/%d", index), "tool_definition", "tools", name, string(tool))
		}
	}
	toolNames := map[string]string{}
	for _, field := range []string{"messages", "input"} {
		if len(request[field]) == 0 {
			continue
		}
		var messages []json.RawMessage
		if err := json.Unmarshal(request[field], &messages); err != nil {
			return nil, fmt.Errorf("model provider: decode %s context: %w", field, err)
		}
		for index, raw := range messages {
			var message map[string]json.RawMessage
			if err := json.Unmarshal(raw, &message); err != nil {
				return nil, err
			}
			var role, itemType, name, callID string
			_ = json.Unmarshal(message["role"], &role)
			_ = json.Unmarshal(message["type"], &itemType)
			_ = json.Unmarshal(message["name"], &name)
			_ = json.Unmarshal(message["call_id"], &callID)
			if itemType == "function_call" && callID != "" {
				toolNames[callID] = name
			}
			if len(message["tool_calls"]) > 0 {
				var calls []struct {
					ID       string `json:"id"`
					Function struct {
						Name string `json:"name"`
					} `json:"function"`
				}
				if err := json.Unmarshal(message["tool_calls"], &calls); err != nil {
					return nil, err
				}
				for _, call := range calls {
					toolNames[call.ID] = call.Function.Name
				}
			}
			if role == "tool" {
				_ = json.Unmarshal(message["tool_call_id"], &callID)
				name = toolNames[callID]
			}
			category := contextCategory(role, itemType)
			if name == "load_skill" {
				category = "skill_injection"
			}
			base := fmt.Sprintf("%s/%d", field, index)
			if len(message["content"]) > 0 {
				var blocks []json.RawMessage
				if json.Unmarshal(message["content"], &blocks) == nil {
					for blockIndex, block := range blocks {
						var typed struct {
							Type      string `json:"type"`
							Text      string `json:"text"`
							ID        string `json:"id"`
							Name      string `json:"name"`
							ToolUseID string `json:"tool_use_id"`
						}
						if err := json.Unmarshal(block, &typed); err != nil {
							return nil, err
						}
						if typed.Type == "image" || typed.Type == "image_url" || typed.Type == "input_image" {
							continue
						}
						partCategory := category
						if typed.Type == "tool_result" {
							partCategory = "tool_result"
							if toolNames[typed.ToolUseID] == "load_skill" {
								partCategory = "skill_injection"
							}
						} else if typed.Type == "tool_use" {
							partCategory = "assistant_message"
							toolNames[typed.ID] = typed.Name
						}
						content := typed.Text
						if content == "" {
							content = string(block)
						}
						addMessage(fmt.Sprintf("%s/content/%d", base, blockIndex), partCategory, role, typed.Type, content)
					}
				} else {
					var content string
					if json.Unmarshal(message["content"], &content) != nil {
						content = string(message["content"])
					}
					addMessage(base+"/content", category, role, name, content)
				}
			}
			if itemType == "function_call" || len(message["tool_calls"]) > 0 {
				add(base+"/tool_calls", "assistant_message", role, name, string(raw))
			}
			if itemType == "function_call_output" && len(message["output"]) > 0 {
				var output string
				if json.Unmarshal(message["output"], &output) != nil {
					output = string(message["output"])
				}
				if toolNames[callID] == "load_skill" {
					category = "skill_injection"
				}
				add(base+"/output", category, role, toolNames[callID], output)
			}
			if itemType == "reasoning" {
				add(base+"/reasoning", "assistant_message", role, "推理内容", string(raw))
			}
		}
	}
	return parts, nil
}

func EstimateContextText(content string) int {
	return contextcompaction.EstimateTokens([]*schema.Message{{Role: schema.System, Content: content}})
}

func contextCategory(role, itemType string) string {
	switch {
	case role == "system" || role == "developer":
		return "system_prompt"
	case role == "assistant" || itemType == "function_call":
		return "assistant_message"
	case role == "tool" || itemType == "function_call_output":
		return "tool_result"
	default:
		return "user_message"
	}
}
