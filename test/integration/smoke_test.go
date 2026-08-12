package smoke_test

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/mr9esx/comfyui_tgbot/internal/catalog/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/catalog/infrastructure/validation"
	convdomain "github.com/mr9esx/comfyui_tgbot/internal/conversation/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/packaging/botapp"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/blob"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/blob/localfs"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/instance"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/instance/static"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/queue"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/queue/memory"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/application/orchestrator"
	runtimedomain "github.com/mr9esx/comfyui_tgbot/internal/runtime/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/actuator"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/comfyui"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

type memCases struct{ c *domain.Case }

func (m *memCases) Save(ctx context.Context, c *domain.Case) error { return m.Create(ctx, c) }
func (m *memCases) Create(_ context.Context, c *domain.Case) error  { m.c = c; return nil }
func (m *memCases) Get(_ context.Context, id sharedkernel.CaseID) (*domain.Case, error) {
	if m.c == nil || m.c.Document.ID != id {
		return nil, domain.ErrNotFound
	}
	cp := *m.c
	return &cp, nil
}
func (m *memCases) List(context.Context, domain.ListQuery) ([]*domain.Case, error) { return nil, nil }
func (m *memCases) Disable(context.Context, sharedkernel.CaseID) error             { return nil }
func (m *memCases) Enable(context.Context, sharedkernel.CaseID) error              { return nil }

type memNotify struct{ last *sharedkernel.UserNotify }

func (m *memNotify) Publish(_ context.Context, n sharedkernel.UserNotify) error {
	cp := n
	m.last = &cp
	return nil
}

func TestMemoryAllInOneText2Img(t *testing.T) {
	ctx := context.Background()
	now := time.Unix(1000, 0).UTC()
	bus := memory.New()
	tasks := runtimedomain.NewMemoryTaskRepository()
	n := &memNotify{}
	reg := static.New(instance.Instance{ID: "local", DispatchTopic: "dispatch.local"})
	orch := orchestrator.New(tasks, reg, bus, n)
	orch.Now = func() time.Time { return now }

	store, err := localfs.New(filepath.Join(t.TempDir(), "blob"))
	if err != nil {
		t.Fatal(err)
	}
	var submitted comfyui.Graph
	mock := &comfyui.Mock{
		SubmitFn: func(_ context.Context, graph comfyui.Graph) (string, error) {
			submitted = graph
			return "prompt-mock", nil
		},
	}
	cases := &memCases{}
	snap := &actuator.CaseSnapshot{Tasks: tasks, Cases: cases, Blob: store, Uploader: mock}
	orch.Prep = snap
	worker := &actuator.Worker{
		InstanceID: "local",
		Comfy:      mock,
		Blob:       store,
		Status:     bus,
		Workflows:  snap,
		Now:        func() time.Time { return now },
	}

	_ = bus.Subscribe(ctx, sharedkernel.TopicTaskCreated, func(ctx context.Context, msg queue.Message) error {
		var ev sharedkernel.TaskCreated
		if err := json.Unmarshal(msg.Payload, &ev); err != nil {
			return err
		}
		if err := orch.OnTaskCreated(ctx, ev); err != nil {
			return err
		}
		claimed, err := tasks.ClaimNextWithLease(ctx, "local", 90*time.Second, now)
		if err != nil {
			return err
		}
		if claimed == nil {
			return nil
		}
		return worker.HandleDispatch(ctx, sharedkernel.DispatchCommand{
			TaskID: claimed.ID, InstanceID: claimed.InstanceID, JobRef: claimed.JobRef,
		})
	})
	_ = bus.Subscribe(ctx, sharedkernel.TopicTaskStatus, func(ctx context.Context, msg queue.Message) error {
		var ev sharedkernel.TaskStatusEvent
		if err := json.Unmarshal(msg.Payload, &ev); err != nil {
			return err
		}
		return orch.OnStatus(ctx, ev)
	})

	doc := domain.CaseDocument{
		ID: "text2img-demo", Name: "Demo",
		Inputs:  []domain.InputField{{Key: "prompt", Type: "string", Required: true}},
		Outputs: []domain.OutputField{{Key: "image", Type: "image"}},
		Bindings: domain.ComfyBindings{
			WorkflowJSON: map[string]any{"1": map[string]any{}},
			Inputs:       []domain.InputBinding{{Key: "prompt", NodeID: "1", FieldPath: "text"}},
		},
		InputSchema: map[string]any{
			"type": "object", "required": []any{"prompt"},
			"properties": map[string]any{"prompt": map[string]any{"type": "string", "minLength": 1}},
		},
	}
	_ = cases.Create(ctx, &domain.Case{Document: doc, Enabled: true})

	sessRepo := convdomain.NewMemoryRepository()
	sessSvc := convdomain.NewService(sessRepo, func() sharedkernel.SessionID { return "s1" }, func() time.Time { return now })
	_, _ = sessSvc.StartCase(ctx, 42, "user-smoke", "text2img-demo", []string{"prompt"})
	p := "cat"
	_, _ = sessSvc.SubmitInput(ctx, 42, convdomain.DraftValue{Text: &p})

	facade := &botapp.Facade{
		Cases: cases, Validator: validation.New(), Sessions: sessSvc, SessionStore: sessRepo,
		Tasks: tasks, Blob: store, Publisher: bus,
		NewTaskID: func() sharedkernel.TaskID { return "task-smoke" },
		Now:       func() time.Time { return now },
	}
	res, err := facade.ConfirmRun(ctx, botapp.ConfirmRunCmd{ChatID: 42})
	if err != nil {
		t.Fatal(err)
	}
	got, err := tasks.Get(ctx, res.TaskID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != sharedkernel.TaskSucceeded {
		t.Fatalf("want succeeded, got %s", got.Status)
	}
	if n.last == nil || n.last.Kind != "task_succeeded" {
		t.Fatalf("notify=%+v", n.last)
	}
	node, _ := submitted["1"].(map[string]any)
	inputs, _ := node["inputs"].(map[string]any)
	if gotText, _ := inputs["text"].(string); gotText != "cat" {
		t.Fatalf("submitted workflow missing Case injection: inputs.text=%q want %q (graph=%#v)", gotText, "cat", submitted)
	}
}

// TestMemoryAllInOneImageAndPrompt covers ConfirmRun → CaseSnapshot (upload+inject) → Mock Comfy → notify
// with a staged user image (reference) and prompt text.
func TestMemoryAllInOneImageAndPrompt(t *testing.T) {
	ctx := context.Background()
	now := time.Unix(2000, 0).UTC()
	bus := memory.New()
	tasks := runtimedomain.NewMemoryTaskRepository()
	n := &memNotify{}
	reg := static.New(instance.Instance{ID: "local", DispatchTopic: "dispatch.local"})
	orch := orchestrator.New(tasks, reg, bus, n)
	orch.Now = func() time.Time { return now }

	store, err := localfs.New(filepath.Join(t.TempDir(), "blob"))
	if err != nil {
		t.Fatal(err)
	}

	var submitted comfyui.Graph
	var uploadedName string
	mock := &comfyui.Mock{
		SubmitFn: func(_ context.Context, graph comfyui.Graph) (string, error) {
			submitted = graph
			return "prompt-img-mock", nil
		},
		UploadImageFn: func(_ context.Context, filename, mime string, data []byte) (string, error) {
			if filename != "user-ref.png" {
				t.Errorf("UploadImage filename=%q want user-ref.png", filename)
			}
			if mime != "image/png" {
				t.Errorf("UploadImage mime=%q want image/png", mime)
			}
			if !bytes.Equal(data, []byte("user-image-bytes")) {
				t.Errorf("UploadImage data=%q want user-image-bytes", data)
			}
			uploadedName = "mock-upload-ref.png"
			return uploadedName, nil
		},
	}
	cases := &memCases{}
	snap := &actuator.CaseSnapshot{Tasks: tasks, Cases: cases, Blob: store, Uploader: mock}
	orch.Prep = snap
	worker := &actuator.Worker{
		InstanceID: "local",
		Comfy:      mock,
		Blob:       store,
		Status:     bus,
		Workflows:  snap,
		Now:        func() time.Time { return now },
	}

	_ = bus.Subscribe(ctx, sharedkernel.TopicTaskCreated, func(ctx context.Context, msg queue.Message) error {
		var ev sharedkernel.TaskCreated
		if err := json.Unmarshal(msg.Payload, &ev); err != nil {
			return err
		}
		if err := orch.OnTaskCreated(ctx, ev); err != nil {
			return err
		}
		claimed, err := tasks.ClaimNextWithLease(ctx, "local", 90*time.Second, now)
		if err != nil {
			return err
		}
		if claimed == nil {
			return nil
		}
		return worker.HandleDispatch(ctx, sharedkernel.DispatchCommand{
			TaskID: claimed.ID, InstanceID: claimed.InstanceID, JobRef: claimed.JobRef,
		})
	})
	_ = bus.Subscribe(ctx, sharedkernel.TopicTaskStatus, func(ctx context.Context, msg queue.Message) error {
		var ev sharedkernel.TaskStatusEvent
		if err := json.Unmarshal(msg.Payload, &ev); err != nil {
			return err
		}
		return orch.OnStatus(ctx, ev)
	})

	doc := domain.CaseDocument{
		ID: "img-edit-smoke", Name: "Edit smoke",
		Inputs: []domain.InputField{
			{Key: "reference", Type: "image", Required: true},
			{Key: "prompt", Type: "string", Required: true},
		},
		Outputs: []domain.OutputField{{Key: "image", Type: "image"}},
		Bindings: domain.ComfyBindings{
			WorkflowJSON: map[string]any{
				"10": map[string]any{
					"class_type": "LoadImage",
					"inputs":     map[string]any{"image": "placeholder.png"},
				},
				"20": map[string]any{
					"class_type": "CLIPTextEncode",
					"inputs":     map[string]any{"text": "placeholder"},
				},
			},
			Inputs: []domain.InputBinding{
				{Key: "reference", NodeID: "10", FieldPath: "image"},
				{Key: "prompt", NodeID: "20", FieldPath: "text"},
			},
		},
		InputSchema: map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"required":             []any{"reference", "prompt"},
			"properties": map[string]any{
				"reference": map[string]any{
					"type":     "object",
					"required": []any{"key"},
					"properties": map[string]any{
						"key":  map[string]any{"type": "string", "minLength": 1},
						"mime": map[string]any{"type": "string"},
						"size": map[string]any{"type": "number"},
					},
				},
				"prompt": map[string]any{"type": "string", "minLength": 1},
			},
		},
	}
	_ = cases.Create(ctx, &domain.Case{Document: doc, Enabled: true})

	ref, err := store.Put(ctx, "uploads/user-ref.png", bytes.NewReader([]byte("user-image-bytes")), blob.PutOptions{MIME: "image/png"})
	if err != nil {
		t.Fatal(err)
	}

	sessRepo := convdomain.NewMemoryRepository()
	sessSvc := convdomain.NewService(sessRepo, func() sharedkernel.SessionID { return "s-img" }, func() time.Time { return now })
	_, _ = sessSvc.StartCase(ctx, 77, "user-img", "img-edit-smoke", []string{"reference", "prompt"})
	_, _ = sessSvc.SubmitInput(ctx, 77, convdomain.DraftValue{Blob: &ref})
	prompt := "make it anime"
	_, _ = sessSvc.SubmitInput(ctx, 77, convdomain.DraftValue{Text: &prompt})

	facade := &botapp.Facade{
		Cases: cases, Validator: validation.New(), Sessions: sessSvc, SessionStore: sessRepo,
		Tasks: tasks, Blob: store, Publisher: bus,
		NewTaskID: func() sharedkernel.TaskID { return "task-img-smoke" },
		Now:       func() time.Time { return now },
	}
	res, err := facade.ConfirmRun(ctx, botapp.ConfirmRunCmd{ChatID: 77})
	if err != nil {
		t.Fatal(err)
	}
	got, err := tasks.Get(ctx, res.TaskID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != sharedkernel.TaskSucceeded {
		t.Fatalf("want succeeded, got %s", got.Status)
	}
	if n.last == nil || n.last.Kind != "task_succeeded" {
		t.Fatalf("notify=%+v", n.last)
	}
	if uploadedName == "" {
		t.Fatal("expected UploadImage to be called")
	}

	imgNode, _ := submitted["10"].(map[string]any)
	imgInputs, _ := imgNode["inputs"].(map[string]any)
	if gotImg, _ := imgInputs["image"].(string); gotImg != uploadedName {
		t.Fatalf("node 10 image=%q want %q (graph=%#v)", gotImg, uploadedName, submitted)
	}
	txtNode, _ := submitted["20"].(map[string]any)
	txtInputs, _ := txtNode["inputs"].(map[string]any)
	if gotText, _ := txtInputs["text"].(string); gotText != "make it anime" {
		t.Fatalf("node 20 text=%q want %q (graph=%#v)", gotText, "make it anime", submitted)
	}
}
