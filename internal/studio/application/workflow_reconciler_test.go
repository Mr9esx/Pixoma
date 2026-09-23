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
	require.Len(t, nodes, 2)
	require.Len(t, edges, 1)
	require.Equal(t, "operation-a", edges[0].SourceNodeID)
	require.Equal(t, assets[0].ID, nodes[1].AssetID)
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
	_, edges, err := repo.GetFlow(ctx, "account-a", "session-a")
	require.NoError(t, err)
	require.Len(t, edges, 1)
}

func seedSucceededWorkflow(t *testing.T, ctx context.Context, repo domain.Repository, tasks runtimedomain.TaskRepository, now time.Time) {
	t.Helper()
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
