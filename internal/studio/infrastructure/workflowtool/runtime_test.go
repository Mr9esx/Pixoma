package workflowtool_test

import (
	"context"
	"testing"
	"time"

	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/stretchr/testify/require"

	studioapp "github.com/Mr9esx/Pixoma/internal/studio/application"
	"github.com/Mr9esx/Pixoma/internal/studio/domain"
	"github.com/Mr9esx/Pixoma/internal/studio/infrastructure/workflowtool"
)

type workflowStarter func(context.Context, studioapp.WorkflowStartInput) (*studioapp.WorkflowStartResult, error)

func (f workflowStarter) Start(ctx context.Context, input studioapp.WorkflowStartInput) (*studioapp.WorkflowStartResult, error) {
	return f(ctx, input)
}

type workflowSink struct {
	events  []string
	nodes   []studioapp.FlowNodeInput
	request *domain.Clarification
}

func (s *workflowSink) Emit(_ context.Context, eventType string, _ any) error {
	s.events = append(s.events, eventType)
	return nil
}
func (*workflowSink) AssistantMessage(context.Context, string) (*domain.Message, error) {
	return nil, nil
}
func (*workflowSink) CreateAsset(context.Context, studioapp.GeneratedAsset) (*domain.Asset, error) {
	return nil, nil
}
func (s *workflowSink) CreateFlowNode(_ context.Context, input studioapp.FlowNodeInput) (*domain.FlowNode, error) {
	s.nodes = append(s.nodes, input)
	return &domain.FlowNode{ID: "node-1"}, nil
}
func (*workflowSink) CreateFlowEdge(context.Context, string, string, string) (*domain.FlowEdge, error) {
	return nil, nil
}
func (*workflowSink) RequestApproval(context.Context, string, string, string) (*domain.Approval, error) {
	return nil, nil
}

func (s *workflowSink) RequestWorkflowInput(_ context.Context, workflow domain.WorkflowRequest) (*domain.Clarification, error) {
	request, err := domain.NewWorkflowClarification("request-1", "run-1", "session-1", "account-1", workflow, time.Now())
	s.request = request
	return request, err
}

func TestWorkflowToolRequestsEditableInputsBeforeStartingTask(t *testing.T) {
	t.Parallel()
	sink := &workflowSink{}
	tools, err := workflowtool.NewRuntimeTools([]studioapp.ResolvedWorkflow{{
		ID: "12", ToolName: "studio_workflow_12", Name: "角色三视图", Description: "生成角色设定图",
		InputSchema: []byte(`{"type":"object","properties":{"prompt":{"type":"string"}},"required":["prompt"]}`),
	}}, workflowtool.ToolAccess{
		Sink: sink,
		Starter: workflowStarter(func(context.Context, studioapp.WorkflowStartInput) (*studioapp.WorkflowStartResult, error) {
			return &studioapp.WorkflowStartResult{TaskID: "task-1", WorkflowID: "12"}, nil
		}),
	})
	require.NoError(t, err)
	invokable, ok := tools[0].(einotool.InvokableTool)
	require.True(t, ok)
	_, err = invokable.InvokableRun(context.Background(), `{"prompt":"雨夜侦探"}`)
	require.Error(t, err)
	require.Equal(t, "角色三视图", sink.request.Workflow.Name)
	require.Equal(t, map[string]any{"prompt": "雨夜侦探"}, sink.request.Workflow.SuggestedInputs)
	require.Empty(t, sink.nodes)
}

func TestWorkflowToolRequestsInputsWhenAgentHasNoSuggestion(t *testing.T) {
	t.Parallel()
	sink := &workflowSink{}
	tools, err := workflowtool.NewRuntimeTools([]studioapp.ResolvedWorkflow{{
		ID: "12", ToolName: "studio_workflow_12", Name: "角色三视图",
		InputSchema: []byte(`{"type":"object","properties":{"prompt":{"type":"string"}},"required":["prompt"]}`),
	}}, workflowtool.ToolAccess{
		Sink: sink,
		Starter: workflowStarter(func(context.Context, studioapp.WorkflowStartInput) (*studioapp.WorkflowStartResult, error) {
			return &studioapp.WorkflowStartResult{TaskID: "task-1", WorkflowID: "12"}, nil
		}),
	})
	require.NoError(t, err)
	invokable, ok := tools[0].(einotool.InvokableTool)
	require.True(t, ok)
	_, err = invokable.InvokableRun(context.Background(), `{}`)
	require.Error(t, err)
	require.NotNil(t, sink.request)
	require.Empty(t, sink.request.Workflow.SuggestedInputs)
}

func TestWorkflowToolProvidesItsInputSchema(t *testing.T) {
	t.Parallel()
	sink := &workflowSink{}
	tools, err := workflowtool.NewRuntimeTools([]studioapp.ResolvedWorkflow{{
		ID: "12", ToolName: "studio_workflow_12", Name: "角色三视图", Description: "生成角色设定图",
		InputSchema: []byte(`{"type":"object","properties":{"prompt":{"type":"string"}},"required":["prompt"]}`),
	}}, workflowtool.ToolAccess{
		AccountID: "account-1", SessionID: "session-1", RunID: "run-1", Sink: sink,
		Starter: workflowStarter(func(_ context.Context, input studioapp.WorkflowStartInput) (*studioapp.WorkflowStartResult, error) {
			return &studioapp.WorkflowStartResult{TaskID: "task-1", WorkflowID: "12"}, nil
		}),
	})
	require.NoError(t, err)
	require.Len(t, tools, 1)
	info, err := tools[0].Info(context.Background())
	require.NoError(t, err)
	require.Equal(t, "studio_workflow_12", info.Name)
	require.Equal(t, "生成角色设定图", info.Desc)

	require.NotNil(t, info.ParamsOneOf)
}
