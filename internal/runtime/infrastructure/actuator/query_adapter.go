package actuator

import (
	"context"

	"github.com/mr9esx/comfyui_tgbot/internal/runtime/application/orchestrator"
	runtimedomain "github.com/mr9esx/comfyui_tgbot/internal/runtime/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

// QueryAdapter exposes Task repository to Orchestrator reconcile / execution queries.
type QueryAdapter struct {
	Tasks runtimedomain.TaskRepository
}

func (q *QueryAdapter) GetRun(ctx context.Context, taskID sharedkernel.TaskID) (*orchestrator.ExecutionView, error) {
	t, err := q.Tasks.Get(ctx, taskID)
	if err != nil {
		return &orchestrator.ExecutionView{TaskID: taskID, Phase: "unknown"}, nil
	}
	return executionViewFromTask(t), nil
}

func executionViewFromTask(t *runtimedomain.Task) *orchestrator.ExecutionView {
	view := &orchestrator.ExecutionView{
		TaskID:   t.ID,
		Phase:    phaseFromTaskStatus(t.Status),
		PromptID: t.PromptID,
		ErrorMsg: t.ErrorMessage,
	}
	for _, o := range t.Outputs {
		view.Outputs = append(view.Outputs, o.Blob)
	}
	return view
}

func phaseFromTaskStatus(st sharedkernel.TaskStatus) string {
	switch st {
	case sharedkernel.TaskQueued:
		return "accepted"
	case sharedkernel.TaskRunning:
		return "running"
	case sharedkernel.TaskSucceeded:
		return "succeeded"
	case sharedkernel.TaskFailed:
		return "failed"
	default:
		return "unknown"
	}
}

var _ orchestrator.ExecutionQuery = (*QueryAdapter)(nil)
