package actuator

import (
	"context"

	"github.com/mr9esx/comfyui_tgbot/internal/runtime/application/orchestrator"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

// QueryAdapter exposes Worker ledger to Orchestrator reconcile.
type QueryAdapter struct {
	Worker *Worker
}

func (q *QueryAdapter) GetRun(ctx context.Context, taskID sharedkernel.TaskID) (*orchestrator.ExecutionView, error) {
	run, err := q.Worker.GetRun(ctx, taskID)
	if err != nil {
		return &orchestrator.ExecutionView{TaskID: taskID, Phase: "unknown"}, nil
	}
	return &orchestrator.ExecutionView{
		TaskID:   run.TaskID,
		Phase:    run.Phase,
		PromptID: run.PromptID,
		Outputs:  append([]sharedkernel.BlobRef(nil), run.Outputs...),
		ErrorMsg: run.ErrorMsg,
	}, nil
}

var _ orchestrator.ExecutionQuery = (*QueryAdapter)(nil)
