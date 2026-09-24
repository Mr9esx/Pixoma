package studiotool

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"

	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	einojsonschema "github.com/eino-contrib/jsonschema"

	studioapp "github.com/Mr9esx/Pixoma/internal/studio/application"
	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

type editSessionFlowTool struct {
	info   *schema.ToolInfo
	access ToolAccess
	lister studioapp.FlowNodeLister
}

type flowEditCommand struct {
	Type         string `json:"type"`
	Title        string `json:"title"`
	Body         string `json:"body,omitempty"`
	SourceNodeID string `json:"source_node_id,omitempty"`
	TargetNodeID string `json:"target_node_id,omitempty"`
	Label        string `json:"label,omitempty"`
}

func newEditSessionFlowTool(access ToolAccess, lister studioapp.FlowNodeLister) (einotool.BaseTool, error) {
	var rawSchema einojsonschema.Schema
	if err := json.Unmarshal([]byte(`{"type":"object","additionalProperties":false,"required":["operations"],"properties":{"operations":{"type":"array","minItems":1,"maxItems":12,"items":{"type":"object","additionalProperties":false,"required":["type"],"properties":{"type":{"type":"string","enum":["create_stage","create_plan","connect_nodes"]},"title":{"type":"string"},"body":{"type":"string"},"source_node_id":{"type":"string"},"target_node_id":{"type":"string"},"label":{"type":"string"}}}}}}`), &rawSchema); err != nil {
		return nil, err
	}
	return &editSessionFlowTool{
		info: &schema.ToolInfo{
			Name:        "edit_session_flow",
			Desc:        "按语义命令新增阶段或计划节点，或连接当前 Session 已有节点；不会移动已有节点。",
			ParamsOneOf: schema.NewParamsOneOfByJSONSchema(&rawSchema),
		},
		access: access,
		lister: lister,
	}, nil
}

func (t *editSessionFlowTool) Info(context.Context) (*schema.ToolInfo, error) { return t.info, nil }

func (t *editSessionFlowTool) InvokableRun(ctx context.Context, arguments string, _ ...einotool.Option) (string, error) {
	var input struct {
		Operations []flowEditCommand `json:"operations"`
	}
	if err := json.Unmarshal([]byte(arguments), &input); err != nil {
		return "", fmt.Errorf("studio: invalid edit_session_flow arguments: %w", err)
	}
	if len(input.Operations) == 0 || len(input.Operations) > 12 {
		return "", fmt.Errorf("studio: edit_session_flow requires 1-12 operations")
	}
	for i := range input.Operations {
		command := &input.Operations[i]
		command.Title = strings.TrimSpace(command.Title)
		command.Body = strings.TrimSpace(command.Body)
		command.SourceNodeID = strings.TrimSpace(command.SourceNodeID)
		command.TargetNodeID = strings.TrimSpace(command.TargetNodeID)
		command.Label = strings.TrimSpace(command.Label)
		if (command.Type == "connect_nodes" && (command.SourceNodeID == "" || command.TargetNodeID == "" || command.SourceNodeID == command.TargetNodeID)) ||
			(command.Type != "connect_nodes" && (command.Title == "" || (command.Type != "create_stage" && command.Type != "create_plan"))) {
			return "", fmt.Errorf("studio: unsupported or incomplete edit_session_flow command at index %d", i)
		}
	}
	nodes, err := t.lister.ListFlowNodes(ctx)
	if err != nil {
		return "", err
	}
	known := make(map[string]struct{}, len(nodes))
	for _, node := range nodes {
		if node != nil {
			known[node.ID] = struct{}{}
		}
	}
	for i, command := range input.Operations {
		if command.Type != "connect_nodes" {
			continue
		}
		if _, ok := known[command.SourceNodeID]; !ok {
			return "", fmt.Errorf("studio: source node for command %d is not in this session", i)
		}
		if _, ok := known[command.TargetNodeID]; !ok {
			return "", fmt.Errorf("studio: target node for command %d is not in this session", i)
		}
	}
	sortOrder := 0
	for _, node := range nodes {
		if node != nil && node.SortOrder >= sortOrder {
			sortOrder = node.SortOrder + 10
		}
	}
	canonical, err := json.Marshal(input.Operations)
	if err != nil {
		return "", err
	}
	action := fmt.Sprintf("flow.edit.%x", sha256.Sum256(canonical))
	if err := t.access.Sink.Emit(ctx, studioapp.EventToolCallStart, map[string]any{"tool_call_id": action, "tool_name": t.info.Name}); err != nil {
		return "", err
	}
	if err := t.access.Sink.Emit(ctx, studioapp.EventToolCallArgs, map[string]any{"tool_call_id": action, "delta": arguments}); err != nil {
		return "", err
	}
	created := make([]string, 0, len(input.Operations))
	for i, command := range input.Operations {
		if command.Type == "connect_nodes" {
			edge, createErr := t.access.Sink.CreateFlowEdge(ctx, command.SourceNodeID, command.TargetNodeID, command.Label)
			if createErr != nil {
				return "", t.finish(ctx, action, createErr)
			}
			if edge == nil {
				return "", t.finish(ctx, action, fmt.Errorf("studio: edit_session_flow returned no edge"))
			}
			created = append(created, fmt.Sprintf("关系（%s）", edge.ID))
			continue
		}
		kind := domain.FlowNodeStage
		if command.Type == "create_plan" {
			kind = domain.FlowNodePlan
		}
		node, createErr := t.access.Sink.CreateFlowNode(ctx, studioapp.FlowNodeInput{
			ActionID: fmt.Sprintf("%s.%d", action, i), Type: kind,
			Title: command.Title, Body: command.Body, SortOrder: sortOrder,
		})
		if createErr != nil {
			return "", t.finish(ctx, action, createErr)
		}
		if node == nil {
			return "", t.finish(ctx, action, fmt.Errorf("studio: edit_session_flow returned no node"))
		}
		created = append(created, fmt.Sprintf("%s（%s）", command.Title, node.ID))
		sortOrder += 10
	}
	result := "已更新 Flow：" + strings.Join(created, "、")
	if err := t.access.Sink.Emit(ctx, studioapp.EventToolCallResult, map[string]any{"tool_call_id": action, "content": result, "is_error": false}); err != nil {
		return "", err
	}
	if err := t.access.Sink.Emit(ctx, studioapp.EventToolCallEnd, map[string]any{"tool_call_id": action, "tool_name": t.info.Name, "node_count": len(created), "is_error": false}); err != nil {
		return "", err
	}
	return result, nil
}

func (t *editSessionFlowTool) finish(ctx context.Context, action string, cause error) error {
	if err := t.access.Sink.Emit(ctx, studioapp.EventToolCallResult, map[string]any{"tool_call_id": action, "content": cause.Error(), "is_error": true}); err != nil {
		return err
	}
	if err := t.access.Sink.Emit(ctx, studioapp.EventToolCallEnd, map[string]any{"tool_call_id": action, "tool_name": t.info.Name, "is_error": true}); err != nil {
		return err
	}
	return cause
}
