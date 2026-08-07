package actuator_test

import (
	"bytes"
	"context"
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
	if err := tasks.Create(ctx, runtimedomain.NewPending("task-1", 1, "text-inject", prefix, now)); err != nil {
		t.Fatal(err)
	}

	cases := &memCases{}
	if err := cases.Create(ctx, &catalogdomain.Case{Document: textWorkflowCase(), Enabled: true}); err != nil {
		t.Fatal(err)
	}

	snap := &actuator.CaseSnapshot{Tasks: tasks, Cases: cases, Blob: store}
	graph, err := snap.WorkflowForTask(ctx, "task-1")
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
	if err := tasks.Create(ctx, runtimedomain.NewPending("task-2", 1, "no-bind", prefix, now)); err != nil {
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
	_, err = snap.WorkflowForTask(ctx, "task-2")
	if err == nil {
		t.Fatal("expected error when binding missing")
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
	if err := tasks.Create(ctx, runtimedomain.NewPending("task-3", 1, "empty-wf", prefix, now)); err != nil {
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
	_, err = snap.WorkflowForTask(ctx, "task-3")
	if err == nil {
		t.Fatal("expected error when workflow empty")
	}
}
