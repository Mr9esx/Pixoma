package actuator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"sort"
	"time"

	catalogdomain "github.com/mr9esx/comfyui_tgbot/internal/catalog/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/blob"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/queue"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/comfyui"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

// CaseSnapshotProvider supplies workflow graph for a task (Phase1: injected map).
// uploader is the per-dispatch Comfy client used for image uploads (may be nil).
type CaseSnapshotProvider interface {
	WorkflowForTask(ctx context.Context, taskID sharedkernel.TaskID, uploader ImageUploader) (comfyui.Graph, error)
}

type StaticWorkflows map[sharedkernel.TaskID]comfyui.Graph

func (s StaticWorkflows) WorkflowForTask(_ context.Context, taskID sharedkernel.TaskID, _ ImageUploader) (comfyui.Graph, error) {
	g, ok := s[taskID]
	if !ok {
		return comfyui.Graph{"1": map[string]any{}}, nil
	}
	return g, nil
}

type Worker struct {
	EdgeID sharedkernel.EdgeID
	Comfy  comfyui.Client
	// Clients maps EdgeID → Comfy client; preferred over Comfy when set.
	Clients map[sharedkernel.EdgeID]comfyui.Client
	// ResolveClient optionally resolves a client by EdgeID (e.g. pool.Client).
	ResolveClient func(sharedkernel.EdgeID) (comfyui.Client, error)
	Blob          blob.Store
	Status        queue.Publisher
	Workflows     CaseSnapshotProvider
	Now           func() time.Time
}

func (w *Worker) HandleDispatch(ctx context.Context, ev sharedkernel.DispatchCommand) error {
	now := w.now()
	cli, err := w.clientFor(ev.EdgeID)
	if err != nil {
		return w.fail(ctx, ev, "comfy_client", err.Error(), now)
	}

	graph, outputBindings, err := w.resolveJob(ctx, ev, cli)
	if err != nil {
		return w.fail(ctx, ev, "workflow", err.Error(), now)
	}

	graph = stripAnnotationNodes(graph)
	promptID, err := cli.Submit(ctx, graph)
	if err != nil {
		return w.fail(ctx, ev, "comfy_submit", err.Error(), now)
	}
	if err := w.publishStatus(ctx, sharedkernel.TaskStatusEvent{
		TaskID: ev.TaskID, EdgeID: ev.EdgeID, Status: sharedkernel.TaskRunning,
		PromptID: promptID, At: w.now(),
	}); err != nil {
		return err
	}

	res, err := cli.Wait(ctx, promptID)
	if err != nil {
		return w.fail(ctx, ev, "comfy_wait", err.Error(), w.now())
	}

	outs, err := w.storeOutputs(ctx, ev.TaskID, res, outputBindings)
	if err != nil {
		return w.fail(ctx, ev, "output_extract", err.Error(), w.now())
	}
	return w.publishStatus(ctx, sharedkernel.TaskStatusEvent{
		TaskID: ev.TaskID, EdgeID: ev.EdgeID, Status: sharedkernel.TaskSucceeded,
		PromptID: promptID, Outputs: outs, At: w.now(),
	})
}

// annotationClassTypes 是不参与执行的纯批注/备注节点，ComfyUI 通常未安装
// 对应自定义节点，提交前剔除，避免整条工作流被 400 拒掉。
var annotationClassTypes = map[string]bool{
	"MarkdownNote": true,
	"Note":         true,
	"note":         true,
}

// stripAnnotationNodes 删除未被其它节点引用的批注节点，保留被引用的情况以免破坏图。
func stripAnnotationNodes(graph comfyui.Graph) comfyui.Graph {
	referenced := map[string]bool{}
	for _, raw := range graph {
		node, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		inputs, _ := node["inputs"].(map[string]any)
		for _, v := range inputs {
			arr, ok := v.([]any)
			if ok && len(arr) > 0 {
				if s, ok := arr[0].(string); ok {
					referenced[s] = true
				}
			}
		}
	}
	for id, raw := range graph {
		node, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		ct, _ := node["class_type"].(string)
		if annotationClassTypes[ct] && !referenced[id] {
			delete(graph, id)
		}
	}
	return graph
}

func (w *Worker) resolveJob(ctx context.Context, ev sharedkernel.DispatchCommand, cli comfyui.Client) (comfyui.Graph, []catalogdomain.OutputBinding, error) {
	if ev.JobRef.Key != "" {
		job, err := w.jobFromBlob(ctx, ev.JobRef, cli)
		if err != nil {
			return nil, nil, err
		}
		return job.Workflow, job.Outputs, nil
	}
	if w.Workflows == nil {
		return nil, nil, fmt.Errorf("actuator: missing job_ref and workflows provider")
	}
	graph, err := w.Workflows.WorkflowForTask(ctx, ev.TaskID, cli)
	return graph, nil, err
}

func (w *Worker) jobFromBlob(ctx context.Context, ref sharedkernel.BlobRef, uploader ImageUploader) (*JobPackage, error) {
	if w.Blob == nil {
		return nil, fmt.Errorf("actuator: blob store not configured")
	}
	rc, err := w.Blob.Get(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("actuator: get job: %w", err)
	}
	defer rc.Close()
	raw, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("actuator: read job: %w", err)
	}
	var job JobPackage
	if err := json.Unmarshal(raw, &job); err != nil {
		return nil, fmt.Errorf("actuator: parse job: %w", err)
	}
	if len(job.Workflow) == 0 {
		return nil, fmt.Errorf("actuator: empty workflow in job")
	}
	for _, img := range job.Images {
		remote, err := w.uploadJobImage(ctx, uploader, img.Blob)
		if err != nil {
			return nil, err
		}
		if err := writeNodeInput(job.Workflow, img.NodeID, img.FieldPath, remote); err != nil {
			return nil, err
		}
	}
	return &job, nil
}

func (w *Worker) storeOutputs(ctx context.Context, taskID sharedkernel.TaskID, res *comfyui.Result, bindings []catalogdomain.OutputBinding) ([]sharedkernel.BlobRef, error) {
	if len(bindings) == 0 {
		var keys []string
		for nodeID := range res.Outputs {
			keys = append(keys, nodeID)
		}
		sort.Strings(keys)
		var outs []sharedkernel.BlobRef
		i := 0
		for _, nodeID := range keys {
			for _, img := range res.Outputs[nodeID].Images {
				key := fmt.Sprintf("outputs/%s/%d_%s", taskID, i, img.Filename)
				ref, err := w.Blob.Put(ctx, key, bytes.NewReader(img.Data), blob.PutOptions{MIME: img.Mime})
				if err != nil {
					return nil, err
				}
				outs = append(outs, ref)
				i++
			}
		}
		return outs, nil
	}

	var outs []sharedkernel.BlobRef
	for _, binding := range bindings {
		node, ok := res.Outputs[binding.NodeID]
		if !ok || len(node.Images) == 0 {
			return nil, fmt.Errorf("output binding %q: node %s produced no image outputs", binding.Key, binding.NodeID)
		}
		index := binding.Index
		if index < 0 {
			index = 0
		}
		if index >= len(node.Images) {
			return nil, fmt.Errorf("output binding %q: node %s index %d out of range", binding.Key, binding.NodeID, index)
		}
		img := node.Images[index]
		key := fmt.Sprintf("outputs/%s/%s_%d_%s", taskID, binding.Key, index, img.Filename)
		ref, err := w.Blob.Put(ctx, key, bytes.NewReader(img.Data), blob.PutOptions{MIME: img.Mime})
		if err != nil {
			return nil, err
		}
		outs = append(outs, ref)
	}
	return outs, nil
}

func (w *Worker) uploadJobImage(ctx context.Context, uploader ImageUploader, ref sharedkernel.BlobRef) (string, error) {
	if uploader == nil {
		return "", fmt.Errorf("actuator: image uploader not configured")
	}
	rc, err := w.Blob.Get(ctx, ref)
	if err != nil {
		return "", fmt.Errorf("actuator: get image blob %s: %w", ref.Key, err)
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil {
		return "", fmt.Errorf("actuator: read image blob %s: %w", ref.Key, err)
	}
	filename := path.Base(ref.Key)
	if filename == "" || filename == "." || filename == "/" {
		filename = "image.png"
	}
	mime := ref.MIME
	if mime == "" {
		mime = "application/octet-stream"
	}
	remote, err := uploader.UploadImage(ctx, filename, mime, data)
	if err != nil {
		return "", fmt.Errorf("actuator: upload image %s: %w", ref.Key, err)
	}
	if remote == "" {
		return "", fmt.Errorf("actuator: upload image %s returned empty filename", ref.Key)
	}
	return remote, nil
}

func (w *Worker) clientFor(id sharedkernel.EdgeID) (comfyui.Client, error) {
	if w.ResolveClient != nil {
		return w.ResolveClient(id)
	}
	if len(w.Clients) > 0 {
		cli, ok := w.Clients[id]
		if !ok || cli == nil {
			return nil, fmt.Errorf("actuator: no client for instance %s", id)
		}
		return cli, nil
	}
	if w.Comfy != nil {
		return w.Comfy, nil
	}
	return nil, fmt.Errorf("actuator: no client for instance %s", id)
}

func (w *Worker) fail(ctx context.Context, ev sharedkernel.DispatchCommand, code, msg string, now time.Time) error {
	if err := w.publishStatus(ctx, sharedkernel.TaskStatusEvent{
		TaskID: ev.TaskID, EdgeID: ev.EdgeID, Status: sharedkernel.TaskFailed,
		ErrorCode: code, ErrorMsg: msg, At: now,
	}); err != nil {
		return err
	}
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
