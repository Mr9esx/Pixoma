package actuator_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"path/filepath"
	"sync"
	"testing"
	"time"

	catalogdomain "github.com/mr9esx/comfyui_tgbot/internal/catalog/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/blob"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/blob/localfs"
	runtimedomain "github.com/mr9esx/comfyui_tgbot/internal/runtime/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/actuator"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

type recordingUploader struct {
	calls int
	last  struct {
		filename string
		mime     string
		data     []byte
	}
	remoteName string
	err        error
}

func (u *recordingUploader) UploadImage(_ context.Context, filename, mime string, data []byte) (string, error) {
	u.calls++
	u.last.filename = filename
	u.last.mime = mime
	u.last.data = append([]byte(nil), data...)
	if u.err != nil {
		return "", u.err
	}
	if u.remoteName != "" {
		return u.remoteName, nil
	}
	return "remote-upload.png", nil
}

var errBlobGetSentinel = errors.New("blob get failed: permission denied")

type errGetBlobStore struct{}

func (errGetBlobStore) Put(context.Context, string, io.Reader, blob.PutOptions) (sharedkernel.BlobRef, error) {
	return sharedkernel.BlobRef{}, errors.New("put not supported")
}

func (errGetBlobStore) Get(context.Context, sharedkernel.BlobRef) (io.ReadCloser, error) {
	return nil, errBlobGetSentinel
}

type memCases struct {
	mu sync.Mutex
	m  map[sharedkernel.CaseID]*catalogdomain.Case
}

func (r *memCases) Save(ctx context.Context, c *catalogdomain.Case) error {
	return r.Create(ctx, c)
}

func (r *memCases) Create(_ context.Context, c *catalogdomain.Case) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.m == nil {
		r.m = map[sharedkernel.CaseID]*catalogdomain.Case{}
	}
	cp := *c
	r.m[c.Document.ID] = &cp
	return nil
}

func (r *memCases) Get(_ context.Context, id sharedkernel.CaseID) (*catalogdomain.Case, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.m[id]
	if !ok {
		return nil, catalogdomain.ErrNotFound
	}
	cp := *c
	return &cp, nil
}

func (r *memCases) List(context.Context, catalogdomain.ListQuery) ([]*catalogdomain.Case, error) {
	return nil, nil
}

func (r *memCases) Disable(context.Context, sharedkernel.CaseID) error { return nil }

func textWorkflowCase() catalogdomain.CaseDocument {
	return catalogdomain.CaseDocument{
		ID:   "text-inject",
		Name: "Text inject",
		Inputs: []catalogdomain.InputField{
			{Key: "prompt", Type: "string", Required: true},
		},
		Outputs: []catalogdomain.OutputField{{Key: "image", Type: "image"}},
		Bindings: catalogdomain.ComfyBindings{
			WorkflowJSON: map[string]any{
				"20": map[string]any{
					"class_type": "CLIPTextEncode",
					"inputs": map[string]any{
						"text": "placeholder",
						"clip": []any{"19", 0},
					},
				},
			},
			Inputs: []catalogdomain.InputBinding{
				{Key: "prompt", NodeID: "20", FieldPath: "text"},
			},
		},
		InputSchema: map[string]any{"type": "object"},
	}
}

func TestCaseSnapshotInjectsStagedTextIntoNodeInputs(t *testing.T) {
	ctx := context.Background()
	store, err := localfs.New(filepath.Join(t.TempDir(), "blob"))
	if err != nil {
		t.Fatal(err)
	}
	prefix := "inputs/task-1"
	if _, err := store.Put(ctx, prefix+"/prompt.txt", bytes.NewReader([]byte("a cat sitting")), blob.PutOptions{MIME: "text/plain"}); err != nil {
		t.Fatal(err)
	}

	tasks := runtimedomain.NewMemoryTaskRepository()
	now := time.Unix(1, 0).UTC()
	if err := tasks.Create(ctx, runtimedomain.NewPending("task-1", "s1", "text-inject", prefix, now)); err != nil {
		t.Fatal(err)
	}

	cases := &memCases{}
	if err := cases.Create(ctx, &catalogdomain.Case{Document: textWorkflowCase(), Enabled: true}); err != nil {
		t.Fatal(err)
	}

	snap := &actuator.CaseSnapshot{Tasks: tasks, Cases: cases, Blob: store}
	graph, err := snap.WorkflowForTask(ctx, "task-1", nil)
	if err != nil {
		t.Fatalf("WorkflowForTask: %v", err)
	}

	node, ok := graph["20"].(map[string]any)
	if !ok {
		t.Fatalf("node 20 missing or wrong type: %#v", graph["20"])
	}
	inputs, ok := node["inputs"].(map[string]any)
	if !ok {
		t.Fatalf("node inputs missing: %#v", node)
	}
	got, _ := inputs["text"].(string)
	if got != "a cat sitting" {
		t.Fatalf("inputs.text=%q want %q", got, "a cat sitting")
	}
}

func TestCaseSnapshotFailsWhenBindingMissing(t *testing.T) {
	ctx := context.Background()
	store, err := localfs.New(filepath.Join(t.TempDir(), "blob"))
	if err != nil {
		t.Fatal(err)
	}
	prefix := "inputs/task-2"
	if _, err := store.Put(ctx, prefix+"/prompt.txt", bytes.NewReader([]byte("orphan text")), blob.PutOptions{MIME: "text/plain"}); err != nil {
		t.Fatal(err)
	}

	tasks := runtimedomain.NewMemoryTaskRepository()
	now := time.Unix(1, 0).UTC()
	if err := tasks.Create(ctx, runtimedomain.NewPending("task-2", "s1", "no-bind", prefix, now)); err != nil {
		t.Fatal(err)
	}

	doc := textWorkflowCase()
	doc.ID = "no-bind"
	doc.Bindings.Inputs = nil // staged prompt has no binding

	cases := &memCases{}
	if err := cases.Create(ctx, &catalogdomain.Case{Document: doc, Enabled: true}); err != nil {
		t.Fatal(err)
	}

	snap := &actuator.CaseSnapshot{Tasks: tasks, Cases: cases, Blob: store}
	_, err = snap.WorkflowForTask(ctx, "task-2", nil)
	if err == nil {
		t.Fatal("expected error when binding missing")
	}
}

func TestCaseSnapshotPropagatesNonMissingBlobGetError(t *testing.T) {
	ctx := context.Background()
	prefix := "inputs/task-blob-err"

	tasks := runtimedomain.NewMemoryTaskRepository()
	now := time.Unix(1, 0).UTC()
	if err := tasks.Create(ctx, runtimedomain.NewPending("task-blob-err", "s1", "text-inject", prefix, now)); err != nil {
		t.Fatal(err)
	}

	cases := &memCases{}
	if err := cases.Create(ctx, &catalogdomain.Case{Document: textWorkflowCase(), Enabled: true}); err != nil {
		t.Fatal(err)
	}

	snap := &actuator.CaseSnapshot{Tasks: tasks, Cases: cases, Blob: errGetBlobStore{}}
	_, err := snap.WorkflowForTask(ctx, "task-blob-err", nil)
	if err == nil {
		t.Fatal("expected blob Get error to propagate")
	}
	if !errors.Is(err, errBlobGetSentinel) {
		t.Fatalf("got %v, want wrapped %v", err, errBlobGetSentinel)
	}
}

func TestCaseSnapshotFailsWhenWorkflowEmpty(t *testing.T) {
	ctx := context.Background()
	store, err := localfs.New(filepath.Join(t.TempDir(), "blob"))
	if err != nil {
		t.Fatal(err)
	}
	prefix := "inputs/task-3"
	if _, err := store.Put(ctx, prefix+"/prompt.txt", bytes.NewReader([]byte("x")), blob.PutOptions{MIME: "text/plain"}); err != nil {
		t.Fatal(err)
	}

	tasks := runtimedomain.NewMemoryTaskRepository()
	now := time.Unix(1, 0).UTC()
	if err := tasks.Create(ctx, runtimedomain.NewPending("task-3", "s1", "empty-wf", prefix, now)); err != nil {
		t.Fatal(err)
	}

	doc := textWorkflowCase()
	doc.ID = "empty-wf"
	doc.Bindings.WorkflowJSON = map[string]any{}

	cases := &memCases{}
	if err := cases.Create(ctx, &catalogdomain.Case{Document: doc, Enabled: true}); err != nil {
		t.Fatal(err)
	}

	snap := &actuator.CaseSnapshot{Tasks: tasks, Cases: cases, Blob: store}
	_, err = snap.WorkflowForTask(ctx, "task-3", nil)
	if err == nil {
		t.Fatal("expected error when workflow empty")
	}
}

func imageWorkflowCase() catalogdomain.CaseDocument {
	return catalogdomain.CaseDocument{
		ID:   "image-inject",
		Name: "Image inject",
		Inputs: []catalogdomain.InputField{
			{Key: "reference", Type: "image", Required: true},
		},
		Outputs: []catalogdomain.OutputField{{Key: "image", Type: "image"}},
		Bindings: catalogdomain.ComfyBindings{
			WorkflowJSON: map[string]any{
				"10": map[string]any{
					"class_type": "LoadImage",
					"inputs": map[string]any{
						"image": "placeholder.png",
					},
				},
			},
			Inputs: []catalogdomain.InputBinding{
				{Key: "reference", NodeID: "10", FieldPath: "image"},
			},
		},
		InputSchema: map[string]any{"type": "object"},
	}
}

func TestCaseSnapshotUploadsImageAndWritesRemoteFilename(t *testing.T) {
	ctx := context.Background()
	store, err := localfs.New(filepath.Join(t.TempDir(), "blob"))
	if err != nil {
		t.Fatal(err)
	}
	prefix := "inputs/task-img"
	imgKey := prefix + "/reference.png"
	payload := []byte("user-image-bytes")
	ref, err := store.Put(ctx, imgKey, bytes.NewReader(payload), blob.PutOptions{MIME: "image/png"})
	if err != nil {
		t.Fatal(err)
	}
	meta, err := json.Marshal(ref)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Put(ctx, prefix+"/reference.blob.json", bytes.NewReader(meta), blob.PutOptions{MIME: "application/json"}); err != nil {
		t.Fatal(err)
	}

	tasks := runtimedomain.NewMemoryTaskRepository()
	now := time.Unix(1, 0).UTC()
	if err := tasks.Create(ctx, runtimedomain.NewPending("task-img", "s1", "image-inject", prefix, now)); err != nil {
		t.Fatal(err)
	}

	cases := &memCases{}
	if err := cases.Create(ctx, &catalogdomain.Case{Document: imageWorkflowCase(), Enabled: true}); err != nil {
		t.Fatal(err)
	}

	up := &recordingUploader{remoteName: "comfy-remote.png"}
	snap := &actuator.CaseSnapshot{Tasks: tasks, Cases: cases, Blob: store, Uploader: up}
	graph, err := snap.WorkflowForTask(ctx, "task-img", nil)
	if err != nil {
		t.Fatalf("WorkflowForTask: %v", err)
	}

	if up.calls != 1 {
		t.Fatalf("UploadImage calls=%d want 1", up.calls)
	}
	if up.last.mime != "image/png" {
		t.Fatalf("upload mime=%q want image/png", up.last.mime)
	}
	if !bytes.Equal(up.last.data, payload) {
		t.Fatalf("upload data=%q want %q", up.last.data, payload)
	}
	if up.last.filename != "reference.png" {
		t.Fatalf("upload filename=%q want reference.png", up.last.filename)
	}

	node, ok := graph["10"].(map[string]any)
	if !ok {
		t.Fatalf("node 10 missing or wrong type: %#v", graph["10"])
	}
	inputs, ok := node["inputs"].(map[string]any)
	if !ok {
		t.Fatalf("node inputs missing: %#v", node)
	}
	got, _ := inputs["image"].(string)
	if got != "comfy-remote.png" {
		t.Fatalf("inputs.image=%q want comfy-remote.png", got)
	}
}

func TestCaseSnapshotFailsWhenUploaderNilForImage(t *testing.T) {
	ctx := context.Background()
	store, err := localfs.New(filepath.Join(t.TempDir(), "blob"))
	if err != nil {
		t.Fatal(err)
	}
	prefix := "inputs/task-img-nil"
	imgKey := prefix + "/reference.png"
	ref, err := store.Put(ctx, imgKey, bytes.NewReader([]byte("x")), blob.PutOptions{MIME: "image/png"})
	if err != nil {
		t.Fatal(err)
	}
	meta, _ := json.Marshal(ref)
	if _, err := store.Put(ctx, prefix+"/reference.blob.json", bytes.NewReader(meta), blob.PutOptions{MIME: "application/json"}); err != nil {
		t.Fatal(err)
	}

	tasks := runtimedomain.NewMemoryTaskRepository()
	now := time.Unix(1, 0).UTC()
	if err := tasks.Create(ctx, runtimedomain.NewPending("task-img-nil", "s1", "image-inject", prefix, now)); err != nil {
		t.Fatal(err)
	}
	cases := &memCases{}
	if err := cases.Create(ctx, &catalogdomain.Case{Document: imageWorkflowCase(), Enabled: true}); err != nil {
		t.Fatal(err)
	}

	snap := &actuator.CaseSnapshot{Tasks: tasks, Cases: cases, Blob: store}
	_, err = snap.WorkflowForTask(ctx, "task-img-nil", nil)
	if err == nil {
		t.Fatal("expected error when Uploader is nil")
	}
}

func TestCaseSnapshot_PrefersPassedUploader(t *testing.T) {
	ctx := context.Background()
	store, err := localfs.New(filepath.Join(t.TempDir(), "blob"))
	if err != nil {
		t.Fatal(err)
	}
	prefix := "inputs/task-img-pass"
	imgKey := prefix + "/reference.png"
	ref, err := store.Put(ctx, imgKey, bytes.NewReader([]byte("x")), blob.PutOptions{MIME: "image/png"})
	if err != nil {
		t.Fatal(err)
	}
	meta, _ := json.Marshal(ref)
	if _, err := store.Put(ctx, prefix+"/reference.blob.json", bytes.NewReader(meta), blob.PutOptions{MIME: "application/json"}); err != nil {
		t.Fatal(err)
	}

	tasks := runtimedomain.NewMemoryTaskRepository()
	now := time.Unix(1, 0).UTC()
	if err := tasks.Create(ctx, runtimedomain.NewPending("task-img-pass", "s1", "image-inject", prefix, now)); err != nil {
		t.Fatal(err)
	}
	cases := &memCases{}
	if err := cases.Create(ctx, &catalogdomain.Case{Document: imageWorkflowCase(), Enabled: true}); err != nil {
		t.Fatal(err)
	}

	fieldUp := &recordingUploader{remoteName: "field.png"}
	passUp := &recordingUploader{remoteName: "passed.png"}
	snap := &actuator.CaseSnapshot{Tasks: tasks, Cases: cases, Blob: store, Uploader: fieldUp}
	graph, err := snap.WorkflowForTask(ctx, "task-img-pass", passUp)
	if err != nil {
		t.Fatalf("WorkflowForTask: %v", err)
	}
	if fieldUp.calls != 0 {
		t.Fatalf("field uploader calls=%d want 0", fieldUp.calls)
	}
	if passUp.calls != 1 {
		t.Fatalf("passed uploader calls=%d want 1", passUp.calls)
	}
	node := graph["10"].(map[string]any)
	inputs := node["inputs"].(map[string]any)
	if got, _ := inputs["image"].(string); got != "passed.png" {
		t.Fatalf("image=%q want passed.png", got)
	}
}
