package actuator_test

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
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

func TestHandleDispatchPublishesRunningAndSucceeded(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	store, err := localfs.New(filepath.Join(dir, "blob"))
	if err != nil {
		t.Fatal(err)
	}
	cap := &statusCap{}
	w := &actuator.Worker{
		InstanceID: "local",
		Comfy:      &comfyui.Mock{},
		Blob:       store,
		Status:     cap,
		Workflows:  actuator.StaticWorkflows{},
		Now:        func() time.Time { return time.Unix(1, 0).UTC() },
	}
	if err := w.HandleDispatch(ctx, sharedkernel.DispatchCommand{TaskID: "t1", InstanceID: "local"}); err != nil {
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

	var submitted []sharedkernel.InstanceID
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
		Blob:   store,
		Status: cap,
		Workflows: actuator.StaticWorkflows{},
		Now:       func() time.Time { return time.Unix(1, 0).UTC() },
		Clients: map[sharedkernel.InstanceID]comfyui.Client{
			"gpu-a": mockA,
			"gpu-b": mockB,
		},
	}

	if err := w.HandleDispatch(ctx, sharedkernel.DispatchCommand{TaskID: "t1", InstanceID: "gpu-b"}); err != nil {
		t.Fatal(err)
	}
	if len(submitted) != 1 || submitted[0] != "gpu-b" {
		t.Fatalf("submitted=%v want [gpu-b]", submitted)
	}
	var running sharedkernel.TaskStatusEvent
	_ = json.Unmarshal(cap.msgs[0].Payload, &running)
	if running.InstanceID != "gpu-b" {
		t.Fatalf("status instance=%s want gpu-b", running.InstanceID)
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

	var uploadedTo, submittedTo sharedkernel.InstanceID
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
		Clients: map[sharedkernel.InstanceID]comfyui.Client{
			"gpu-a": defaultUploader,
			"gpu-b": mockB,
		},
	}

	if err := w.HandleDispatch(ctx, sharedkernel.DispatchCommand{
		TaskID: "t-img", InstanceID: "gpu-b", InputPrefix: prefix,
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
