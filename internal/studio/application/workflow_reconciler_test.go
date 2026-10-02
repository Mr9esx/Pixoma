package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
	studioapp "github.com/Mr9esx/Pixoma/internal/studio/application"
	"github.com/Mr9esx/Pixoma/internal/studio/domain"
	runtimedomain "github.com/Mr9esx/Pixoma/internal/tasks/domain"
)

func TestReconcileSucceededWorkflowAdoptsOutputsWithoutCopyingBlob(t *testing.T) {
	ctx := context.Background()
	repo := openRepository(t)
	tasks := runtimedomain.NewMemoryTaskRepository()
	now := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	seedSucceededWorkflow(t, ctx, repo, tasks, now)

	reconciler := &studioapp.WorkflowReconciler{Repo: repo, Tasks: tasks, IDs: (&idSequence{}).Next, Now: func() time.Time { return now }}
	require.NoError(t, reconciler.ReconcileOnce(ctx, 10))

	assets, err := repo.ListSessionAssets(ctx, "account-a", "session-a", 10)
	require.NoError(t, err)
	require.Len(t, assets, 1)
	require.Equal(t, domain.AssetOriginWorkflow, assets[0].Origin)
	require.Equal(t, domain.AssetImage, assets[0].Kind)
	require.Len(t, assets[0].Versions, 1)
	require.Equal(t, "outputs/task-a/storyboard.png", assets[0].Versions[0].BlobKey)

	execution, err := repo.GetWorkflowExecutionByTask(ctx, "account-a", "task-a")
	require.NoError(t, err)
	require.Equal(t, domain.WorkflowExecutionSucceeded, execution.Status)
	nodes, edges, err := repo.GetFlow(ctx, "account-a", "session-a")
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	require.Empty(t, edges)
	require.Equal(t, "operation-a", nodes[0].ID)
	require.Len(t, nodes[0].Outputs, 1)
	require.Equal(t, assets[0].ID, nodes[0].Outputs[0].AssetID)
	require.Equal(t, "storyboard", nodes[0].Outputs[0].Key)
}

func TestReconcileTerminalWorkflowIsIdempotent(t *testing.T) {
	ctx := context.Background()
	repo := openRepository(t)
	tasks := runtimedomain.NewMemoryTaskRepository()
	now := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	seedSucceededWorkflow(t, ctx, repo, tasks, now)

	reconciler := &studioapp.WorkflowReconciler{Repo: repo, Tasks: tasks, IDs: (&idSequence{}).Next, Now: func() time.Time { return now }}
	require.NoError(t, reconciler.ReconcileOnce(ctx, 10))
	require.NoError(t, reconciler.ReconcileOnce(ctx, 10))
	assets, err := repo.ListSessionAssets(ctx, "account-a", "session-a", 10)
	require.NoError(t, err)
	require.Len(t, assets, 1)
	nodes, edges, err := repo.GetFlow(ctx, "account-a", "session-a")
	require.NoError(t, err)
	require.Empty(t, edges)
	require.Len(t, nodes, 1)
	require.Len(t, nodes[0].Outputs, 1)
}

func TestWorkflowFieldRelationshipsUseRecordedAssets(t *testing.T) {
	ctx := context.Background()
	repo := openRepository(t)
	tasks := runtimedomain.NewMemoryTaskRepository()
	now := time.Now().UTC()
	seedSucceededWorkflow(t, ctx, repo, tasks, now)
	reconciler := &studioapp.WorkflowReconciler{Repo: repo, Tasks: tasks, IDs: (&idSequence{}).Next}
	require.NoError(t, reconciler.ReconcileOnce(ctx, 10))
	nodes, _, err := repo.GetFlow(ctx, "account-a", "session-a")
	require.NoError(t, err)
	output := nodes[0].Outputs[0]
	target, err := domain.NewFlowNode("target", "session-a", "account-a", domain.FlowNodeOperation, "图像合成", 2000, now)
	require.NoError(t, err)
	require.NoError(t, repo.SaveFlowNode(ctx, target))
	execution, err := domain.NewWorkflowExecution("target-execution", "account-a", "session-a", "run-a", "target-tool", "target-task", "13", target.ID, now)
	require.NoError(t, err)
	for _, key := range []string{"reference", "background"} {
		execution.InputFields = append(execution.InputFields, domain.FlowPort{Key: key, Type: "image"})
		execution.Inputs = append(execution.Inputs, domain.FlowInput{Key: key, AssetID: output.AssetID, AssetVersionID: output.AssetVersionID})
	}
	require.NoError(t, repo.CreateWorkflowExecution(ctx, execution))
	service := &studioapp.Service{Repo: repo}
	for _, key := range []string{"reference", "background"} {
		edge, err := service.CreateFlowEdge(ctx, "account-a", "session-a", studioapp.CreateFlowEdgeInput{Source: "operation-a", Target: target.ID, SourceOutputKey: output.Key, TargetInputKey: key})
		require.NoError(t, err)
		require.ErrorIs(t, service.DeleteFlowEdge(ctx, "account-a", "session-a", edge.ID), domain.ErrInvalid)
	}
	_, err = service.CreateFlowEdge(ctx, "account-a", "session-a", studioapp.CreateFlowEdgeInput{Source: "operation-a", Target: target.ID, SourceOutputKey: output.Key, TargetInputKey: "reference"})
	require.ErrorIs(t, err, domain.ErrInvalid)
	_, err = service.CreateFlowEdge(ctx, "account-a", "session-a", studioapp.CreateFlowEdgeInput{Source: "operation-a", Target: target.ID, SourceOutputKey: "missing", TargetInputKey: "reference"})
	require.ErrorIs(t, err, domain.ErrInvalid)
	require.ErrorIs(t, service.DeleteFlowNode(ctx, "account-a", "session-a", target.ID), domain.ErrInvalid)
	require.NoError(t, studioapp.RecordWorkflowInputEdges(ctx, repo, execution, now))
	_, edges, err := repo.GetFlow(ctx, "account-a", "session-a")
	require.NoError(t, err)
	require.Len(t, edges, 2)
	counts, err := repo.ListActiveWorkflowCounts(ctx, "account-a", []string{"session-a"})
	require.NoError(t, err)
	require.Equal(t, 1, counts["session-a"])
	foreignCounts, err := repo.ListActiveWorkflowCounts(ctx, "account-b", []string{"session-a"})
	require.NoError(t, err)
	require.Empty(t, foreignCounts)
}

func TestLegacyWorkflowOutputsPreserveConnections(t *testing.T) {
	ctx := context.Background()
	repo := openRepository(t)
	tasks := runtimedomain.NewMemoryTaskRepository()
	now := time.Now().UTC()
	seedSucceededWorkflow(t, ctx, repo, tasks, now)
	require.NoError(t, (&studioapp.WorkflowReconciler{Repo: repo, Tasks: tasks, IDs: (&idSequence{}).Next}).ReconcileOnce(ctx, 10))
	nodes, _, err := repo.GetFlow(ctx, "account-a", "session-a")
	require.NoError(t, err)
	output := nodes[0].Outputs[0]
	nodes[0].Outputs = nil
	require.NoError(t, repo.SaveFlowNode(ctx, nodes[0]))
	legacy, err := domain.NewFlowNode("workflow-output-node-task-a-0", "session-a", "account-a", domain.FlowNodeAsset, output.Name, 1100, now)
	require.NoError(t, err)
	legacy.AssetID, legacy.AssetVersionID = output.AssetID, output.AssetVersionID
	require.NoError(t, repo.SaveFlowNode(ctx, legacy))
	target, err := domain.NewFlowNode("target", "session-a", "account-a", domain.FlowNodeStage, "后续制作", 2000, now)
	require.NoError(t, err)
	require.NoError(t, repo.SaveFlowNode(ctx, target))
	edge, err := domain.NewFlowEdge("legacy-edge", "session-a", "account-a", legacy.ID, target.ID, now)
	require.NoError(t, err)
	require.NoError(t, repo.SaveFlowEdge(ctx, edge))
	require.NoError(t, studioapp.MigrateLegacyWorkflowOutputs(ctx, repo, "account-a", "session-a"))
	require.NoError(t, studioapp.MigrateLegacyWorkflowOutputs(ctx, repo, "account-a", "session-a"))
	nodes, edges, err := repo.GetFlow(ctx, "account-a", "session-a")
	require.NoError(t, err)
	require.Len(t, nodes, 2)
	require.Len(t, nodes[0].Outputs, 1)
	require.Equal(t, output.AssetVersionID, nodes[0].Outputs[0].AssetVersionID)
	require.Len(t, edges, 1)
	require.Equal(t, "operation-a", edges[0].SourceNodeID)
	require.Equal(t, "out-0", edges[0].SourceOutputKey)
	require.Equal(t, target.ID, edges[0].TargetNodeID)
}

func seedSucceededWorkflow(t *testing.T, ctx context.Context, repo domain.Repository, tasks runtimedomain.TaskRepository, now time.Time) {
	t.Helper()
	session, err := domain.NewSession("session-a", "account-a", now)
	require.NoError(t, err)
	require.NoError(t, repo.CreateSession(ctx, session))
	run, err := domain.NewRun("run-a", "session-a", "account-a", "message-a", now)
	require.NoError(t, err)
	require.NoError(t, repo.CreateRun(ctx, run))
	operation, err := domain.NewFlowNode("operation-a", "session-a", "account-a", domain.FlowNodeOperation, "分镜工作流", 1000, now)
	require.NoError(t, err)
	require.NoError(t, repo.SaveFlowNode(ctx, operation))
	task := runtimedomain.NewPending("task-a", "studio-session-session-a", 12, "studio-workflow-inputs/task-a", now)
	require.NoError(t, task.MarkQueued("edge-a", now))
	require.NoError(t, task.MarkRunning("prompt-a", now))
	require.NoError(t, task.MarkSucceeded([]runtimedomain.OutputRef{{Key: "storyboard", Blob: sharedkernel.BlobRef{Key: "outputs/task-a/storyboard.png", MIME: "image/png", Size: 42}}}, now))
	require.NoError(t, tasks.Create(ctx, task))
	execution, err := domain.NewWorkflowExecution("execution-a", "account-a", "session-a", "run-a", "tool-a", "task-a", "12", "operation-a", now)
	require.NoError(t, err)
	require.NoError(t, repo.CreateWorkflowExecution(ctx, execution))
}
