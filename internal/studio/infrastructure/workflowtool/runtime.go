package workflowtool

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"

	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	einojsonschema "github.com/eino-contrib/jsonschema"
	"github.com/google/uuid"

	studioapp "github.com/Mr9esx/Pixoma/internal/studio/application"
	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

// Starter keeps the Eino adapter dependent on the application port, not the
// task, blob, or Studio persistence implementations.
type Starter interface {
	Start(context.Context, studioapp.WorkflowStartInput) (*studioapp.WorkflowStartResult, error)
}

// ToolAccess scopes dynamic workflow tools to one Studio Agent run.
type ToolAccess struct {
	AccountID       string
	SessionID       string
	RunID           string
	PermissionMode  domain.PermissionMode
	IsApproved      func(action string) bool
	RequestApproval func(ctx context.Context, toolCallID, action string) error
	Sink            studioapp.AgentSink
	Starter         Starter
}

func NewRuntimeTools(workflows []studioapp.ResolvedWorkflow, access ToolAccess) ([]einotool.BaseTool, error) {
	if access.Starter == nil || access.Sink == nil {
		return nil, fmt.Errorf("studio: workflow tool access is not configured")
	}
	tools := make([]einotool.BaseTool, 0, len(workflows))
	for _, workflow := range workflows {
		if strings.TrimSpace(workflow.ID) == "" || strings.TrimSpace(workflow.ToolName) == "" || !json.Valid(workflow.InputSchema) {
			return nil, fmt.Errorf("studio: invalid Agent workflow tool")
		}
		var rawSchema einojsonschema.Schema
		if err := json.Unmarshal(workflow.InputSchema, &rawSchema); err != nil {
			return nil, fmt.Errorf("studio: decode workflow %s input schema: %w", workflow.ID, err)
		}
		params := schema.NewParamsOneOfByJSONSchema(&rawSchema)
		tools = append(tools, &runtimeTool{
			info:     &schema.ToolInfo{Name: workflow.ToolName, Desc: workflow.Description, ParamsOneOf: params},
			workflow: workflow,
			access:   access,
		})
	}
	return tools, nil
}

type runtimeTool struct {
	info     *schema.ToolInfo
	workflow studioapp.ResolvedWorkflow
	access   ToolAccess
}

func (t *runtimeTool) Info(context.Context) (*schema.ToolInfo, error) { return t.info, nil }

func (t *runtimeTool) InvokableRun(ctx context.Context, arguments string, _ ...einotool.Option) (string, error) {
	inputs, err := decodeArguments(arguments)
	if err != nil {
		return "", fmt.Errorf("studio: invalid workflow tool arguments for %s: %w", t.info.Name, err)
	}
	action, err := workflowAction(t.workflow.ID, inputs)
	if err != nil {
		return "", err
	}
	if t.requiresApproval() && (t.access.IsApproved == nil || !t.access.IsApproved(action)) {
		if t.access.RequestApproval == nil {
			return "", fmt.Errorf("studio: workflow approval handler is not configured")
		}
		if err := t.access.RequestApproval(ctx, action+"."+uuid.NewString(), action); err != nil {
			return "", err
		}
		return "", studioapp.ErrApprovalRequired
	}
	toolCallID := action
	if err := t.emit(ctx, studioapp.EventToolCallStart, map[string]any{
		"tool_call_id": toolCallID, "tool_name": t.info.Name, "workflow_id": t.workflow.ID,
		"argument_bytes": len(arguments),
	}); err != nil {
		return "", err
	}
	if err := t.emit(ctx, studioapp.EventToolCallArgs, map[string]any{
		"tool_call_id": toolCallID, "delta": arguments,
	}); err != nil {
		return "", err
	}
	node, err := t.access.Sink.CreateFlowNode(ctx, studioapp.FlowNodeInput{
		Type: domain.FlowNodeOperation, Title: t.workflow.Name,
		Body: "已提交，正在后台执行工作流。", SortOrder: 1000,
	})
	if err != nil {
		return "", t.finish(ctx, toolCallID, err)
	}
	if node == nil || strings.TrimSpace(node.ID) == "" {
		return "", t.finish(ctx, toolCallID, fmt.Errorf("studio: workflow operation node was not created"))
	}
	result, err := t.access.Starter.Start(ctx, studioapp.WorkflowStartInput{
		AccountID: t.access.AccountID, SessionID: t.access.SessionID, RunID: t.access.RunID,
		ToolCallID: toolCallID, WorkflowID: t.workflow.ID, OperationNodeID: node.ID, Inputs: inputs,
	})
	if err != nil {
		return "", t.finish(ctx, toolCallID, err)
	}
	var output string
	if result.Reused {
		output = fmt.Sprintf("工作流「%s」已在后台运行，任务编号：%s。", t.workflow.Name, result.TaskID)
	} else {
		output = fmt.Sprintf("已提交工作流「%s」，任务编号：%s，正在后台运行。", t.workflow.Name, result.TaskID)
	}
	if err := t.emit(ctx, studioapp.EventToolCallResult, map[string]any{
		"tool_call_id": toolCallID, "content": output, "is_error": false,
	}); err != nil {
		return "", err
	}
	if err := t.emit(ctx, studioapp.EventToolCallEnd, map[string]any{
		"tool_call_id": toolCallID, "tool_name": t.info.Name, "workflow_id": t.workflow.ID,
		"task_id": result.TaskID, "is_error": false,
	}); err != nil {
		return "", err
	}
	return output, nil
}

func (t *runtimeTool) requiresApproval() bool {
	return t.access.PermissionMode == domain.PermissionRequestApproval
}

func (t *runtimeTool) finish(ctx context.Context, toolCallID string, callErr error) error {
	if err := t.emit(ctx, studioapp.EventToolCallResult, map[string]any{
		"tool_call_id": toolCallID, "content": callErr.Error(), "is_error": true,
	}); err != nil {
		return err
	}
	if emitErr := t.emit(ctx, studioapp.EventToolCallEnd, map[string]any{
		"tool_call_id": toolCallID, "tool_name": t.info.Name, "workflow_id": t.workflow.ID, "is_error": true,
	}); emitErr != nil {
		return emitErr
	}
	return callErr
}

func (t *runtimeTool) emit(ctx context.Context, eventType string, payload any) error {
	return t.access.Sink.Emit(ctx, eventType, payload)
}

func decodeArguments(arguments string) (map[string]any, error) {
	if strings.TrimSpace(arguments) == "" {
		return map[string]any{}, nil
	}
	var inputs map[string]any
	if err := json.Unmarshal([]byte(arguments), &inputs); err != nil {
		return nil, err
	}
	if inputs == nil {
		return nil, fmt.Errorf("arguments must be a JSON object")
	}
	return inputs, nil
}

func workflowAction(workflowID string, inputs map[string]any) (string, error) {
	canonical, err := json.Marshal(inputs)
	if err != nil {
		return "", fmt.Errorf("studio: encode workflow approval arguments: %w", err)
	}
	digest := sha256.Sum256(canonical)
	return fmt.Sprintf("workflow.%s.%x", workflowID, digest), nil
}

var _ einotool.InvokableTool = (*runtimeTool)(nil)
