package actuator_test

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	catalogdomain "github.com/mr9esx/comfyui_tgbot/internal/catalog/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/blob"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/blob/localfs"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/queue"
	runtimedomain "github.com/mr9esx/comfyui_tgbot/internal/runtime/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/actuator"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/comfyui"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

type statusCap struct {
	msgs []queue.Message
}

func (s *statusCap) Publish(_ context.Context, msg queue.Message) error {
	s.msgs = append(s.msgs, msg)
	return nil
}

func writeJob(t *testing.T, ctx context.Context, store blob.Store, job actuator.JobPackage) sharedkernel.BlobRef {
	t.Helper()
	raw, err := json.Marshal(job)
	if err != nil {
		t.Fatal(err)
	}
	ref, err := store.Put(ctx, "jobs/"+string(job.TaskID)+"/job.json", bytes.NewReader(raw), blob.PutOptions{MIME: "application/json"})
	if err != nil {
		t.Fatal(err)
	}
	return ref
}

func TestWorkerExtractsOutputsByBinding(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	store, err := localfs.New(filepath.Join(dir, "blob"))
	if err != nil {
		t.Fatal(err)
	}
	cap := &statusCap{}
	mock := &comfyui.Mock{
		WaitFn: func(_ context.Context, _ string) (*comfyui.Result, error) {
			return &comfyui.Result{
				PromptID: "p1",
				Outputs: comfyui.HistoryResult{
					"9": {
						Images: []comfyui.NodeImage{
							{OutputFile: comfyui.OutputFile{Filename: "out.png", Mime: "image/png", Data: []byte("png")}},
						},
					},
				},
			}, nil
		},
	}
	w := &actuator.Worker{
		EdgeID:    "local",
		Comfy:     mock,
		Blob:      store,
		Status:    cap,
		Workflows: actuator.StaticWorkflows{},
		Now:       func() time.Time { return time.Unix(1, 0).UTC() },
	}
	jobRef := writeJob(t, ctx, store, actuator.JobPackage{
		TaskID:   "t1",
		EdgeID:   "local",
		Workflow: comfyui.Graph{"1": map[string]any{"class_type": "SaveImage", "inputs": map[string]any{}}},
		Outputs:  []catalogdomain.OutputBinding{{Key: "image", NodeID: "9", Index: 0}},
	})
	if err := w.HandleDispatch(ctx, sharedkernel.DispatchCommand{
		TaskID: "t1", EdgeID: "local", JobRef: jobRef,
	}); err != nil {
		t.Fatal(err)
	}
	if len(cap.msgs) != 2 {
		t.Fatalf("msgs=%d", len(cap.msgs))
	}
	var done sharedkernel.TaskStatusEvent
	_ = json.Unmarshal(cap.msgs[1].Payload, &done)
	if done.Status != sharedkernel.TaskSucceeded || len(done.Outputs) != 1 {
		t.Fatalf("done=%+v", done)
	}
	if !strings.HasPrefix(done.Outputs[0].Key, "outputs/t1/image_") {
		t.Fatalf("want keyed output, got %q", done.Outputs[0].Key)
	}
}

func TestWorkerFailsWhenBoundNodeMissing(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	store, err := localfs.New(filepath.Join(dir, "blob"))
	if err != nil {
		t.Fatal(err)
	}
	cap := &statusCap{}
	mock := &comfyui.Mock{
		WaitFn: func(_ context.Context, _ string) (*comfyui.Result, error) {
			return &comfyui.Result{PromptID: "p1", Outputs: comfyui.HistoryResult{}}, nil
		},
	}
	w := &actuator.Worker{
		EdgeID:    "local",
		Comfy:     mock,
		Blob:      store,
		Status:    cap,
		Workflows: actuator.StaticWorkflows{},
		Now:       func() time.Time { return time.Unix(1, 0).UTC() },
	}
	jobRef := writeJob(t, ctx, store, actuator.JobPackage{
		TaskID:   "t2",
		EdgeID:   "local",
		Workflow: comfyui.Graph{"1": map[string]any{"class_type": "SaveImage", "inputs": map[string]any{}}},
		Outputs:  []catalogdomain.OutputBinding{{Key: "image", NodeID: "9", Index: 0}},
	})
	if err := w.HandleDispatch(ctx, sharedkernel.DispatchCommand{
		TaskID: "t2", EdgeID: "local", JobRef: jobRef,
	}); err != nil {
		t.Fatal(err)
	}
	// HandleDispatch 对任务失败返回 nil，失败信息在 status 事件里
	if len(cap.msgs) != 2 {
		t.Fatalf("msgs=%d", len(cap.msgs))
	}
	var failed sharedkernel.TaskStatusEvent
	_ = json.Unmarshal(cap.msgs[1].Payload, &failed)
	if failed.Status != sharedkernel.TaskFailed || failed.ErrorCode != "output_extract" {
		t.Fatalf("failed=%+v", failed)
	}
}

func TestWorkerFallsBackToAllImagesWithoutBindings(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	store, err := localfs.New(filepath.Join(dir, "blob"))
	if err != nil {
		t.Fatal(err)
	}
	cap := &statusCap{}
	mock := &comfyui.Mock{
		WaitFn: func(_ context.Context, _ string) (*comfyui.Result, error) {
			return &comfyui.Result{
				PromptID: "p1",
				Outputs: comfyui.HistoryResult{
					"1": {Images: []comfyui.NodeImage{
						{OutputFile: comfyui.OutputFile{Filename: "a.png", Mime: "image/png", Data: []byte("a")}},
					}},
					"2": {Images: []comfyui.NodeImage{
						{OutputFile: comfyui.OutputFile{Filename: "b.png", Mime: "image/png", Data: []byte("b")}},
					}},
				},
			}, nil
		},
	}
	w := &actuator.Worker{
		EdgeID:    "local",
		Comfy:     mock,
		Blob:      store,
		Status:    cap,
		Workflows: actuator.StaticWorkflows{},
		Now:       func() time.Time { return time.Unix(1, 0).UTC() },
	}
	jobRef := writeJob(t, ctx, store, actuator.JobPackage{
		TaskID:   "t3",
		EdgeID:   "local",
		Workflow: comfyui.Graph{"1": map[string]any{"class_type": "SaveImage", "inputs": map[string]any{}}},
	})
	if err := w.HandleDispatch(ctx, sharedkernel.DispatchCommand{
		TaskID: "t3", EdgeID: "local", JobRef: jobRef,
	}); err != nil {
		t.Fatal(err)
	}
	var done sharedkernel.TaskStatusEvent
	_ = json.Unmarshal(cap.msgs[1].Payload, &done)
	if done.Status != sharedkernel.TaskSucceeded || len(done.Outputs) != 2 {
		t.Fatalf("done=%+v", done)
	}
	if done.Outputs[0].Key != "outputs/t3/0_a.png" || done.Outputs[1].Key != "outputs/t3/1_b.png" {
		t.Fatalf("keys=%q %q", done.Outputs[0].Key, done.Outputs[1].Key)
	}
}

func TestHandleDispatchPublishesRunningAndSucceeded(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	store, err := localfs.New(filepath.Join(dir, "blob"))
	if err != nil {
		t.Fatal(err)
	}
	cap := &statusCap{}
	w := &actuator.Worker{
		EdgeID:    "local",
		Comfy:     &comfyui.Mock{},
		Blob:      store,
		Status:    cap,
		Workflows: actuator.StaticWorkflows{},
		Now:       func() time.Time { return time.Unix(1, 0).UTC() },
	}
	if err := w.HandleDispatch(ctx, sharedkernel.DispatchCommand{TaskID: "t1", EdgeID: "local"}); err != nil {
		t.Fatal(err)
	}
	if len(cap.msgs) != 2 {
		t.Fatalf("status msgs=%d", len(cap.msgs))
	}
	var running, done sharedkernel.TaskStatusEvent
	_ = json.Unmarshal(cap.msgs[0].Payload, &running)
	_ = json.Unmarshal(cap.msgs[1].Payload, &done)
	if running.Status != sharedkernel.TaskRunning || done.Status != sharedkernel.TaskSucceeded {
		t.Fatalf("running=%s done=%s", running.Status, done.Status)
	}
	if len(done.Outputs) == 0 {
		t.Fatal("expected outputs")
	}
}

func TestWorker_UsesDispatchInstanceClient(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	store, err := localfs.New(filepath.Join(dir, "blob"))
	if err != nil {
		t.Fatal(err)
	}

	var submitted []sharedkernel.EdgeID
	mockA := &comfyui.Mock{
		SubmitFn: func(_ context.Context, _ comfyui.Graph) (string, error) {
			submitted = append(submitted, "gpu-a")
			return "pa", nil
		},
	}
	mockB := &comfyui.Mock{
		SubmitFn: func(_ context.Context, _ comfyui.Graph) (string, error) {
			submitted = append(submitted, "gpu-b")
			return "pb", nil
		},
	}

	cap := &statusCap{}
	w := &actuator.Worker{
		Blob:      store,
		Status:    cap,
		Workflows: actuator.StaticWorkflows{},
		Now:       func() time.Time { return time.Unix(1, 0).UTC() },
		Clients: map[sharedkernel.EdgeID]comfyui.Client{
			"gpu-a": mockA,
			"gpu-b": mockB,
		},
	}

	if err := w.HandleDispatch(ctx, sharedkernel.DispatchCommand{TaskID: "t1", EdgeID: "gpu-b"}); err != nil {
		t.Fatal(err)
	}
	if len(submitted) != 1 || submitted[0] != "gpu-b" {
		t.Fatalf("submitted=%v want [gpu-b]", submitted)
	}
	var running sharedkernel.TaskStatusEvent
	_ = json.Unmarshal(cap.msgs[0].Payload, &running)
	if running.EdgeID != "gpu-b" {
		t.Fatalf("status instance=%s want gpu-b", running.EdgeID)
	}
}

func TestWorker_BadJobRefPublishesFailed(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	store, err := localfs.New(filepath.Join(dir, "blob"))
	if err != nil {
		t.Fatal(err)
	}
	cap := &statusCap{}
	w := &actuator.Worker{
		Comfy:  &comfyui.Mock{},
		Blob:   store,
		Status: cap,
		Now:    func() time.Time { return time.Unix(1, 0).UTC() },
	}
	if err := w.HandleDispatch(ctx, sharedkernel.DispatchCommand{
		TaskID: "t-bad-job",
		EdgeID: "local",
		JobRef: sharedkernel.BlobRef{Key: "jobs/missing/job.json", MIME: "application/json"},
	}); err != nil {
		t.Fatal(err)
	}
	if len(cap.msgs) != 1 {
		t.Fatalf("status msgs=%d want 1", len(cap.msgs))
	}
	var ev sharedkernel.TaskStatusEvent
	if err := json.Unmarshal(cap.msgs[0].Payload, &ev); err != nil {
		t.Fatal(err)
	}
	if ev.Status != sharedkernel.TaskFailed || ev.ErrorCode != "workflow" {
		t.Fatalf("ev=%+v", ev)
	}
}

func TestWorker_JobRefHappyPath(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	store, err := localfs.New(filepath.Join(dir, "blob"))
	if err != nil {
		t.Fatal(err)
	}
	job := actuator.JobPackage{
		TaskID:       "t-job",
		EdgeID:       "local",
		Workflow:     map[string]any{"1": map[string]any{"class_type": "Noop", "inputs": map[string]any{}}},
		OutputPrefix: "outputs/t-job",
	}
	raw, err := json.Marshal(job)
	if err != nil {
		t.Fatal(err)
	}
	ref, err := store.Put(ctx, "jobs/t-job/job.json", bytes.NewReader(raw), blob.PutOptions{MIME: "application/json"})
	if err != nil {
		t.Fatal(err)
	}
	cap := &statusCap{}
	w := &actuator.Worker{
		Comfy:  &comfyui.Mock{},
		Blob:   store,
		Status: cap,
		Now:    func() time.Time { return time.Unix(1, 0).UTC() },
	}
	if err := w.HandleDispatch(ctx, sharedkernel.DispatchCommand{
		TaskID: "t-job", EdgeID: "local", JobRef: ref,
	}); err != nil {
		t.Fatal(err)
	}
	if len(cap.msgs) != 2 {
		t.Fatalf("status msgs=%d", len(cap.msgs))
	}
	var done sharedkernel.TaskStatusEvent
	_ = json.Unmarshal(cap.msgs[1].Payload, &done)
	if done.Status != sharedkernel.TaskSucceeded || len(done.Outputs) == 0 {
		t.Fatalf("done=%+v", done)
	}
}

func TestWorker_UploadUsesDispatchInstanceClient(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	store, err := localfs.New(filepath.Join(dir, "blob"))
	if err != nil {
		t.Fatal(err)
	}

	prefix := "inputs/t-img"
	imgKey := prefix + "/reference.png"
	ref, err := store.Put(ctx, imgKey, bytes.NewReader([]byte("img-bytes")), blob.PutOptions{MIME: "image/png"})
	if err != nil {
		t.Fatal(err)
	}
	meta, _ := json.Marshal(ref)
	if _, err := store.Put(ctx, prefix+"/reference.blob.json", bytes.NewReader(meta), blob.PutOptions{MIME: "application/json"}); err != nil {
		t.Fatal(err)
	}

	tasks := runtimedomain.NewMemoryTaskRepository()
	now := time.Unix(1, 0).UTC()
	if err := tasks.Create(ctx, runtimedomain.NewPending("t-img", "s1", "image-inject", prefix, now)); err != nil {
		t.Fatal(err)
	}
	cases := &memCases{}
	if err := cases.Create(ctx, &catalogdomain.Case{Document: imageWorkflowCase(), Enabled: true}); err != nil {
		t.Fatal(err)
	}

	var uploadedTo, submittedTo sharedkernel.EdgeID
	defaultUploader := &comfyui.Mock{
		UploadImageFn: func(_ context.Context, _, _ string, _ []byte) (string, error) {
			uploadedTo = "default"
			return "default-remote.png", nil
		},
		SubmitFn: func(_ context.Context, _ comfyui.Graph) (string, error) {
			submittedTo = "default"
			return "pd", nil
		},
	}
	mockB := &comfyui.Mock{
		UploadImageFn: func(_ context.Context, _, _ string, _ []byte) (string, error) {
			uploadedTo = "gpu-b"
			return "b-remote.png", nil
		},
		SubmitFn: func(_ context.Context, _ comfyui.Graph) (string, error) {
			submittedTo = "gpu-b"
			return "pb", nil
		},
	}

	snap := &actuator.CaseSnapshot{
		Tasks: tasks, Cases: cases, Blob: store,
		Uploader: defaultUploader,
	}
	cap := &statusCap{}
	w := &actuator.Worker{
		Blob:      store,
		Status:    cap,
		Workflows: snap,
		Now:       func() time.Time { return time.Unix(1, 0).UTC() },
		Clients: map[sharedkernel.EdgeID]comfyui.Client{
			"gpu-a": defaultUploader,
			"gpu-b": mockB,
		},
	}

	if err := w.HandleDispatch(ctx, sharedkernel.DispatchCommand{
		TaskID: "t-img", EdgeID: "gpu-b", InputPrefix: prefix,
	}); err != nil {
		t.Fatal(err)
	}
	if uploadedTo != "gpu-b" {
		t.Fatalf("upload instance=%q want gpu-b", uploadedTo)
	}
	if submittedTo != "gpu-b" {
		t.Fatalf("submit instance=%q want gpu-b", submittedTo)
	}
}
