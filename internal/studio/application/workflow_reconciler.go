package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
	"github.com/Mr9esx/Pixoma/internal/studio/domain"
	runtimedomain "github.com/Mr9esx/Pixoma/internal/tasks/domain"
)

const EventWorkflowTaskCompleted = "WORKFLOW_TASK_COMPLETED"

// WorkflowReconciler adopts asynchronous Pixoma task results into the Studio
// session that initiated them. It never copies output blobs; asset versions
// point at the task runtime's immutable BlobRef instead.
type WorkflowReconciler struct {
	Repo  domain.Repository
	Tasks runtimedomain.TaskRepository
	IDs   func() string
	Now   func() time.Time
}

func (r *WorkflowReconciler) ReconcileOnce(ctx context.Context, limit int) error {
	if r == nil || r.Repo == nil || r.Tasks == nil {
		return fmt.Errorf("studio: workflow reconciler is not configured")
	}
	if limit <= 0 {
		limit = 50
	}
	executions, err := r.Repo.ListPendingWorkflowExecutions(ctx, limit)
	if err != nil {
		return fmt.Errorf("studio: list pending workflow executions: %w", err)
	}
	for _, execution := range executions {
		if execution == nil {
			continue
		}
		task, err := r.Tasks.Get(ctx, sharedkernel.TaskID(execution.TaskID))
		if err != nil {
			return fmt.Errorf("studio: get workflow task %s: %w", execution.TaskID, err)
		}
		switch task.Status {
		case sharedkernel.TaskSucceeded:
			if err := r.adoptSucceeded(ctx, execution, task); err != nil {
				return err
			}
		case sharedkernel.TaskFailed, sharedkernel.TaskCancelled:
			if err := r.completeWithoutOutputs(ctx, execution, task); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *WorkflowReconciler) adoptSucceeded(ctx context.Context, execution *domain.WorkflowExecution, task *runtimedomain.Task) error {
	now := r.now()
	for index, output := range task.Outputs {
		asset, err := r.workflowAsset(execution, output, index, now)
		if err != nil {
			return err
		}
		if err := r.Repo.CreateAsset(ctx, asset); err != nil {
			if !errors.Is(err, domain.ErrAlreadyExists) {
				return fmt.Errorf("studio: create workflow output asset: %w", err)
			}
			asset, err = r.Repo.GetAsset(ctx, execution.AccountID, asset.ID)
			if err != nil {
				return fmt.Errorf("studio: read existing workflow output asset: %w", err)
			}
		}
		node, err := domain.NewFlowNode(workflowOutputNodeID(execution, index), execution.SessionID, execution.AccountID, domain.FlowNodeAsset, asset.Name, 1100+index, now)
		if err != nil {
			return err
		}
		node.AssetID = asset.ID
		if len(asset.Versions) == 0 {
			return fmt.Errorf("studio: workflow output asset has no version")
		}
		node.AssetVersionID = asset.Versions[len(asset.Versions)-1].ID
		node.AssetVersion = asset.Versions[len(asset.Versions)-1].Version
		node.RunID = execution.RunID
		if err := r.Repo.SaveFlowNode(ctx, node); err != nil {
			return fmt.Errorf("studio: save workflow output node: %w", err)
		}
		edge, err := domain.NewFlowEdge(workflowOutputEdgeID(execution, index), execution.SessionID, execution.AccountID, execution.OperationNodeID, node.ID, now)
		if err != nil {
			return err
		}
		if err := r.Repo.SaveFlowEdge(ctx, edge); err != nil {
			return fmt.Errorf("studio: connect workflow output node: %w", err)
		}
		if err := r.appendEvent(ctx, execution, EventAssetCreated, map[string]any{"asset_id": asset.ID, "kind": asset.Kind}); err != nil {
			return err
		}
		if err := r.appendEvent(ctx, execution, EventFlowUpdated, map[string]any{"node_id": node.ID, "edge_id": edge.ID, "action": "workflow_output_created"}); err != nil {
			return err
		}
	}
	if err := execution.Complete(domain.WorkflowExecutionSucceeded, "", now); err != nil {
		return err
	}
	if err := r.Repo.UpdateWorkflowExecution(ctx, execution); err != nil {
		return fmt.Errorf("studio: complete workflow execution: %w", err)
	}
	return r.appendEvent(ctx, execution, EventWorkflowTaskCompleted, map[string]any{"task_id": execution.TaskID, "status": "succeeded", "output_count": len(task.Outputs)})
}

func (r *WorkflowReconciler) completeWithoutOutputs(ctx context.Context, execution *domain.WorkflowExecution, task *runtimedomain.Task) error {
	status := domain.WorkflowExecutionFailed
	if task.Status == sharedkernel.TaskCancelled {
		status = domain.WorkflowExecutionCancelled
	}
	if err := execution.Complete(status, task.ErrorMessage, r.now()); err != nil {
		return err
	}
	if err := r.Repo.UpdateWorkflowExecution(ctx, execution); err != nil {
		return fmt.Errorf("studio: complete workflow execution: %w", err)
	}
	return r.appendEvent(ctx, execution, EventWorkflowTaskCompleted, map[string]any{"task_id": execution.TaskID, "status": status})
}

func (r *WorkflowReconciler) workflowAsset(execution *domain.WorkflowExecution, output runtimedomain.OutputRef, index int, now time.Time) (*domain.Asset, error) {
	name := strings.TrimSpace(filepath.Base(output.Blob.Key))
	if name == "" || name == "." {
		name = fmt.Sprintf("工作流输出-%d", index+1)
	}
	asset, err := domain.NewAsset(workflowOutputAssetID(execution, index), execution.SessionID, execution.AccountID, name, assetKindForMIME(output.Blob.MIME), domain.AssetOriginWorkflow, now)
	if err != nil {
		return nil, err
	}
	asset.SourceRunID = execution.RunID
	mimeType := strings.TrimSpace(output.Blob.MIME)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	if _, err := asset.AppendVersion(workflowOutputAssetID(execution, index)+"-v1", mimeType, output.Blob.Key, output.Blob.Size, now); err != nil {
		return nil, err
	}
	return asset, nil
}

func workflowOutputAssetID(execution *domain.WorkflowExecution, index int) string {
	return fmt.Sprintf("workflow-output-%s-%d", execution.TaskID, index)
}

func workflowOutputNodeID(execution *domain.WorkflowExecution, index int) string {
	return fmt.Sprintf("workflow-output-node-%s-%d", execution.TaskID, index)
}

func workflowOutputEdgeID(execution *domain.WorkflowExecution, index int) string {
	return fmt.Sprintf("workflow-output-edge-%s-%d", execution.TaskID, index)
}

func assetKindForMIME(mimeType string) domain.AssetKind {
	switch {
	case strings.HasPrefix(mimeType, "image/"):
		return domain.AssetImage
	case strings.HasPrefix(mimeType, "video/"):
		return domain.AssetVideo
	case strings.HasPrefix(mimeType, "audio/"):
		return domain.AssetAudio
	case strings.HasPrefix(mimeType, "text/"):
		return domain.AssetDocument
	default:
		return domain.AssetFile
	}
}

func (r *WorkflowReconciler) appendEvent(ctx context.Context, execution *domain.WorkflowExecution, eventType string, payload any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = r.Repo.AppendRunEvent(ctx, &domain.Event{
		ID: r.newID(), RunID: execution.RunID, SessionID: execution.SessionID, AccountID: execution.AccountID,
		Type: eventType, Payload: raw, CreatedAt: r.now(),
	})
	return err
}

func (r *WorkflowReconciler) now() time.Time {
	if r.Now != nil {
		return r.Now().UTC()
	}
	return time.Now().UTC()
}

func (r *WorkflowReconciler) newID() string {
	if r.IDs != nil {
		return r.IDs()
	}
	return uuid.NewString()
}
