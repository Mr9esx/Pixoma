package actuator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/blob"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/queue"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/comfyui"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

// CaseSnapshotProvider supplies workflow graph for a task (Phase1: injected map).
type CaseSnapshotProvider interface {
	WorkflowForTask(ctx context.Context, taskID sharedkernel.TaskID) (comfyui.Graph, error)
}

type StaticWorkflows map[sharedkernel.TaskID]comfyui.Graph

func (s StaticWorkflows) WorkflowForTask(_ context.Context, taskID sharedkernel.TaskID) (comfyui.Graph, error) {
	g, ok := s[taskID]
	if !ok {
		return comfyui.Graph{"1": map[string]any{}}, nil
	}
	return g, nil
}

type Worker struct {
	InstanceID sharedkernel.InstanceID
	Comfy      comfyui.Client
	Blob       blob.Store
	Status     queue.Publisher
	Workflows  CaseSnapshotProvider
	Now        func() time.Time
}

func (w *Worker) HandleDispatch(ctx context.Context, ev sharedkernel.DispatchCommand) error {
	now := w.now()

	graph, err := w.Workflows.WorkflowForTask(ctx, ev.TaskID)
	if err != nil {
		return w.fail(ctx, ev, "workflow", err.Error(), now)
	}

	promptID, err := w.Comfy.Submit(ctx, graph)
	if err != nil {
		return w.fail(ctx, ev, "comfy_submit", err.Error(), now)
	}
	if err := w.publishStatus(ctx, sharedkernel.TaskStatusEvent{
		TaskID: ev.TaskID, InstanceID: w.InstanceID, Status: sharedkernel.TaskRunning,
		PromptID: promptID, At: w.now(),
	}); err != nil {
		return err
	}

	res, err := w.Comfy.Wait(ctx, promptID)
	if err != nil {
		return w.fail(ctx, ev, "comfy_wait", err.Error(), w.now())
	}

	var outs []sharedkernel.BlobRef
	for i, f := range res.Outputs {
		key := fmt.Sprintf("outputs/%s/%d_%s", ev.TaskID, i, f.Filename)
		ref, err := w.Blob.Put(ctx, key, bytes.NewReader(f.Data), blob.PutOptions{MIME: f.Mime})
		if err != nil {
			return w.fail(ctx, ev, "blob_put", err.Error(), w.now())
		}
		outs = append(outs, ref)
	}
	return w.publishStatus(ctx, sharedkernel.TaskStatusEvent{
		TaskID: ev.TaskID, InstanceID: w.InstanceID, Status: sharedkernel.TaskSucceeded,
		PromptID: promptID, Outputs: outs, At: w.now(),
	})
}

func (w *Worker) fail(ctx context.Context, ev sharedkernel.DispatchCommand, code, msg string, now time.Time) error {
	_ = w.publishStatus(ctx, sharedkernel.TaskStatusEvent{
		TaskID: ev.TaskID, InstanceID: w.InstanceID, Status: sharedkernel.TaskFailed,
		ErrorCode: code, ErrorMsg: msg, At: now,
	})
	return nil
}

func (w *Worker) publishStatus(ctx context.Context, ev sharedkernel.TaskStatusEvent) error {
	payload, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	return w.Status.Publish(ctx, queue.Message{
		Topic:   sharedkernel.TopicTaskStatus,
		Key:     string(ev.TaskID),
		Payload: payload,
	})
}

func (w *Worker) now() time.Time {
	if w.Now != nil {
		return w.Now()
	}
	return time.Now().UTC()
}
