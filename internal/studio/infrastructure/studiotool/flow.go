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
	assets studioapp.SessionAssetLister
}

type flowEditCommand struct {
	Type           string `json:"type"`
	Title          string `json:"title"`
	Body           string `json:"body,omitempty"`
	SourceNodeID   string `json:"source_node_id,omitempty"`
	TargetNodeID   string `json:"target_node_id,omitempty"`
	Label          string `json:"label,omitempty"`
	AssetID        string `json:"asset_id,omitempty"`
	AssetVersionID string `json:"asset_version_id,omitempty"`
	StageNodeID    string `json:"stage_node_id,omitempty"`
}

func newEditSessionFlowTool(access ToolAccess, lister studioapp.FlowNodeLister) (einotool.BaseTool, error) {
	assetLister, _ := access.Sink.(studioapp.SessionAssetLister)
	description := "按语义命令新增阶段、计划或未执行的操作节点，连接已有节点，或将当前 Session 资产的指定版本挂到已有阶段；不会移动已有节点。add_operation 只创建计划，不执行工作流。"
	if access.Locale == "en" {
		description = "Create stages, plans, or planned operations; connect existing nodes and attach a specific asset version to an existing stage. add_operation plans an operation without executing a workflow."
	}
	var rawSchema einojsonschema.Schema
	if err := json.Unmarshal([]byte(`{"type":"object","additionalProperties":false,"required":["operations"],"properties":{"operations":{"type":"array","minItems":1,"maxItems":12,"items":{"type":"object","additionalProperties":false,"required":["type"],"properties":{"type":{"type":"string","enum":["create_stage","create_plan","connect_nodes","attach_asset","add_operation"]},"title":{"type":"string"},"body":{"type":"string"},"source_node_id":{"type":"string"},"target_node_id":{"type":"string"},"label":{"type":"string"},"asset_id":{"type":"string"},"asset_version_id":{"type":"string"},"stage_node_id":{"type":"string"}}}}}}`), &rawSchema); err != nil {
		return nil, err
	}
	return &editSessionFlowTool{
		info: &schema.ToolInfo{
			Name:        "edit_session_flow",
			Desc:        description,
			ParamsOneOf: schema.NewParamsOneOfByJSONSchema(&rawSchema),
		},
		access: access,
		lister: lister,
		assets: assetLister,
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
		command.AssetID = strings.TrimSpace(command.AssetID)
		command.AssetVersionID = strings.TrimSpace(command.AssetVersionID)
		command.StageNodeID = strings.TrimSpace(command.StageNodeID)
		if (command.Type == "connect_nodes" && (command.SourceNodeID == "" || command.TargetNodeID == "" || command.SourceNodeID == command.TargetNodeID)) ||
			(command.Type == "attach_asset" && (command.AssetID == "" || command.AssetVersionID == "" || command.StageNodeID == "")) ||
			(command.Type == "add_operation" && (command.Title == "" || command.StageNodeID == "")) ||
			(command.Type != "connect_nodes" && command.Type != "attach_asset" && command.Type != "add_operation" && (command.Title == "" || (command.Type != "create_stage" && command.Type != "create_plan"))) {
			return "", fmt.Errorf("studio: unsupported or incomplete edit_session_flow command at index %d", i)
		}
	}
	nodes, err := t.lister.ListFlowNodes(ctx)
	if err != nil {
		return "", err
	}
	known := make(map[string]struct{}, len(nodes))
	stages := make(map[string]struct{})
	for _, node := range nodes {
		if node != nil {
			known[node.ID] = struct{}{}
			if node.Type == domain.FlowNodeStage {
				stages[node.ID] = struct{}{}
			}
		}
	}
	var sessionAssets []*domain.Asset
	for _, command := range input.Operations {
		if command.Type != "attach_asset" {
			continue
		}
		if t.assets == nil {
			return "", fmt.Errorf("studio: session assets are not available")
		}
		sessionAssets, err = t.assets.ListSessionAssets(ctx, 10000)
		if err != nil {
			return "", err
		}
		break
	}
	for i, command := range input.Operations {
		if command.Type == "add_operation" {
			if _, ok := stages[command.StageNodeID]; !ok {
				return "", fmt.Errorf("studio: stage for command %d is not in this session", i)
			}
		}
		if command.Type == "attach_asset" {
			if _, ok := stages[command.StageNodeID]; !ok {
				return "", fmt.Errorf("studio: stage for command %d is not in this session", i)
			}
			valid := false
			for _, asset := range sessionAssets {
				if asset == nil || asset.ID != command.AssetID {
					continue
				}
				for _, version := range asset.Versions {
					if version.ID == command.AssetVersionID {
						valid = true
						break
					}
				}
			}
			if !valid {
				return "", fmt.Errorf("studio: asset version for command %d is not in this session", i)
			}
		}
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
		if command.Type == "add_operation" {
			node, createErr := t.access.Sink.CreateFlowNode(ctx, studioapp.FlowNodeInput{ActionID: fmt.Sprintf("%s.%d", action, i), Type: domain.FlowNodeOperation, Title: command.Title, Body: command.Body, SortOrder: sortOrder})
			if createErr != nil {
				return "", t.finish(ctx, action, createErr)
			}
			if node == nil {
				return "", t.finish(ctx, action, fmt.Errorf("studio: edit_session_flow returned no operation node"))
			}
			if _, createErr = t.access.Sink.CreateFlowEdge(ctx, command.StageNodeID, node.ID, "计划步骤"); createErr != nil {
				return "", t.finish(ctx, action, createErr)
			}
			created = append(created, fmt.Sprintf("%s（%s）", command.Title, node.ID))
			sortOrder += 10
			continue
		}
		if command.Type == "attach_asset" {
			assetName := command.AssetID
			for _, asset := range sessionAssets {
				if asset != nil && asset.ID == command.AssetID {
					assetName = asset.Name
					break
				}
			}
			node, createErr := t.access.Sink.CreateFlowNode(ctx, studioapp.FlowNodeInput{ActionID: fmt.Sprintf("%s.%d", action, i), Type: domain.FlowNodeAsset, Title: assetName, AssetID: command.AssetID, AssetVersionID: command.AssetVersionID, SortOrder: sortOrder})
			if createErr != nil {
				return "", t.finish(ctx, action, createErr)
			}
			if node == nil {
				return "", t.finish(ctx, action, fmt.Errorf("studio: edit_session_flow returned no asset node"))
			}
			if _, createErr = t.access.Sink.CreateFlowEdge(ctx, node.ID, command.StageNodeID, "关联资产"); createErr != nil {
				return "", t.finish(ctx, action, createErr)
			}
			created = append(created, fmt.Sprintf("%s（%s）", assetName, node.ID))
			sortOrder += 10
			continue
		}
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
