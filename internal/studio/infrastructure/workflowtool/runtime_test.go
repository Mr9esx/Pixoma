package workflowtool_test

import (
	"context"
	"testing"

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
	events []string
	nodes  []studioapp.FlowNodeInput
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
func (*workflowSink) RequestApproval(context.Context, string, string) (*domain.Approval, error) {
	return nil, nil
}

func TestWorkflowToolCreatesOperationAndStartsTask(t *testing.T) {
	t.Parallel()
	sink := &workflowSink{}
	var started studioapp.WorkflowStartInput
	tools, err := workflowtool.NewRuntimeTools([]studioapp.ResolvedWorkflow{{
		ID: "12", ToolName: "studio_workflow_12", Name: "角色三视图", Description: "生成角色设定图",
		InputSchema: []byte(`{"type":"object","properties":{"prompt":{"type":"string"}},"required":["prompt"]}`),
	}}, workflowtool.ToolAccess{
		AccountID: "account-1", SessionID: "session-1", RunID: "run-1", Sink: sink,
		Starter: workflowStarter(func(_ context.Context, input studioapp.WorkflowStartInput) (*studioapp.WorkflowStartResult, error) {
			started = input
			return &studioapp.WorkflowStartResult{TaskID: "task-1", WorkflowID: "12"}, nil
		}),
	})
	require.NoError(t, err)
	require.Len(t, tools, 1)
	info, err := tools[0].Info(context.Background())
	require.NoError(t, err)
	require.Equal(t, "studio_workflow_12", info.Name)
	require.Equal(t, "生成角色设定图", info.Desc)

	invokable, ok := tools[0].(einotool.InvokableTool)
	require.True(t, ok)
	output, err := invokable.InvokableRun(context.Background(), `{"prompt":"雨夜侦探"}`)
	require.NoError(t, err)
	require.Contains(t, output, "task-1")
	require.Equal(t, "account-1", started.AccountID)
	require.Equal(t, "node-1", started.OperationNodeID)
	require.Equal(t, map[string]any{"prompt": "雨夜侦探"}, started.Inputs)
	require.Len(t, sink.nodes, 1)
	require.Equal(t, domain.FlowNodeOperation, sink.nodes[0].Type)
	require.Equal(t, "角色三视图", sink.nodes[0].Title)
	require.Equal(t, []string{studioapp.EventToolCallStart, studioapp.EventToolCallEnd}, sink.events)
}
