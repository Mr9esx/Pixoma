package botapp_test

import (
	"context"
	"encoding/json"
	"path/filepath"
	"sync"
	"testing"
	"time"

	domain "github.com/Mr9esx/Pixoma/internal/cases/domain"
	"github.com/Mr9esx/Pixoma/internal/cases/infrastructure/validation"
	convdomain "github.com/Mr9esx/Pixoma/internal/sessions/domain"
	"github.com/Mr9esx/Pixoma/internal/packaging/botapp"
	"github.com/Mr9esx/Pixoma/internal/platform/blob/localfs"
	"github.com/Mr9esx/Pixoma/internal/platform/queue"
	runtimedomain "github.com/Mr9esx/Pixoma/internal/tasks/domain"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
)

type memCases struct {
	mu sync.Mutex
	m  map[sharedkernel.CaseID]*domain.Case
}

func (r *memCases) Save(ctx context.Context, c *domain.Case) error { return r.Create(ctx, c) }
func (r *memCases) Create(_ context.Context, c *domain.Case) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.m == nil {
		r.m = map[sharedkernel.CaseID]*domain.Case{}
	}
	cp := *c
	r.m[c.Document.ID] = &cp
	return nil
}
func (r *memCases) Get(_ context.Context, id sharedkernel.CaseID) (*domain.Case, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.m[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *c
	return &cp, nil
}
func (r *memCases) List(context.Context, domain.ListQuery) ([]*domain.Case, error) {
	return nil, nil
}
func (r *memCases) Disable(context.Context, sharedkernel.CaseID) error { return nil }
func (r *memCases) Enable(context.Context, sharedkernel.CaseID) error  { return nil }
func (r *memCases) Delete(_ context.Context, id sharedkernel.CaseID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.m[id]; !ok {
		return domain.ErrNotFound
	}
	delete(r.m, id)
	return nil
}

type capturePub struct {
	msgs []queue.Message
}

func (p *capturePub) Publish(_ context.Context, msg queue.Message) error {
	p.msgs = append(p.msgs, msg)
	return nil
}

func sampleDoc() domain.CaseDocument {
	return domain.CaseDocument{
		ID:   1,
		Name: "Demo",
		Inputs: []domain.InputField{
			{Key: "prompt", Type: "string", Required: true},
		},
		Outputs: []domain.OutputField{{Key: "image", Type: "image"}},
		Bindings: domain.ComfyBindings{
			WorkflowJSON: map[string]any{
				"1": map[string]any{"class_type": "CLIPTextEncode", "inputs": map[string]any{"text": "x"}},
			},
			Inputs: []domain.InputBinding{{Key: "prompt", NodeID: "1", FieldPath: "text"}},
		},
		InputSchema: map[string]any{
			"type":     "object",
			"required": []any{"prompt"},
			"properties": map[string]any{
				"prompt": map[string]any{"type": "string", "minLength": 1},
			},
		},
	}
}

func TestConfirmRun_WritesSessionID(t *testing.T) {
	ctx := context.Background()
	cases := &memCases{}
	_ = cases.Create(ctx, &domain.Case{Document: sampleDoc(), Enabled: true})

	sessRepo := convdomain.NewMemoryRepository()
	sessSvc := convdomain.NewService(sessRepo, func() sharedkernel.SessionID { return "sess-write" }, func() time.Time {
		return time.Unix(10, 0).UTC()
	})
	_, err := sessSvc.StartCase(ctx, "tg:100", "user-test", 1, []string{"prompt"})
	if err != nil {
		t.Fatal(err)
	}
	prompt := "a cat"
	_, err = sessSvc.SubmitInput(ctx, "tg:100", convdomain.DraftValue{Text: &prompt})
	if err != nil {
		t.Fatal(err)
	}

	pub := &capturePub{}
	tasks := runtimedomain.NewMemoryTaskRepository()
	store, err := localfs.New(filepath.Join(t.TempDir(), "blob"))
	if err != nil {
		t.Fatal(err)
	}
	facade := &botapp.Facade{
		Cases:        cases,
		Validator:    validation.New(),
		Sessions:     sessSvc,
		SessionStore: sessRepo,
		Tasks:        tasks,
		Blob:         store,
		Publisher:    pub,
		NewTaskID:    func() sharedkernel.TaskID { return "task-sid" },
		Now:          func() time.Time { return time.Unix(20, 0).UTC() },
	}

	res, err := facade.ConfirmRun(ctx, botapp.ConfirmRunCmd{ChatID: "tg:100"})
	if err != nil {
		t.Fatal(err)
	}
	task, err := facade.Tasks.Get(ctx, res.TaskID)
	if err != nil {
		t.Fatal(err)
	}
	if task.SessionID == "" {
		t.Fatal("expected session_id")
	}
	sess, err := facade.SessionStore.GetByID(ctx, task.SessionID)
	if err != nil {
		t.Fatal(err)
	}
	if sess.ChatID != "tg:100" {
		t.Fatalf("chat via session: %s", sess.ChatID)
	}
}

func TestConfirmRunCreatesPendingAndPublishes(t *testing.T) {
	ctx := context.Background()
	cases := &memCases{}
	_ = cases.Create(ctx, &domain.Case{Document: sampleDoc(), Enabled: true})

	sessRepo := convdomain.NewMemoryRepository()
	sessSvc := convdomain.NewService(sessRepo, func() sharedkernel.SessionID { return "sess-1" }, func() time.Time {
		return time.Unix(10, 0).UTC()
	})
	_, err := sessSvc.StartCase(ctx, "tg:100", "user-test", 1, []string{"prompt"})
	if err != nil {
		t.Fatal(err)
	}
	prompt := "a cat"
	_, err = sessSvc.SubmitInput(ctx, "tg:100", convdomain.DraftValue{Text: &prompt})
	if err != nil {
		t.Fatal(err)
	}

	pub := &capturePub{}
	tasks := runtimedomain.NewMemoryTaskRepository()
	store, err := localfs.New(filepath.Join(t.TempDir(), "blob"))
	if err != nil {
		t.Fatal(err)
	}
	facade := &botapp.Facade{
		Cases:        cases,
		Validator:    validation.New(),
		Sessions:     sessSvc,
		SessionStore: sessRepo,
		Tasks:        tasks,
		Blob:         store,
		Publisher:    pub,
		NewTaskID:    func() sharedkernel.TaskID { return "task-1" },
		Now:          func() time.Time { return time.Unix(20, 0).UTC() },
	}

	res, err := facade.ConfirmRun(ctx, botapp.ConfirmRunCmd{ChatID: "tg:100"})
	if err != nil {
		t.Fatalf("ConfirmRun: %v", err)
	}
	if res.TaskID != "task-1" || res.Status != sharedkernel.TaskPending {
		t.Fatalf("result=%+v", res)
	}

	got, err := tasks.Get(ctx, "task-1")
	if err != nil || got.Status != sharedkernel.TaskPending {
		t.Fatalf("task=%v err=%v", got, err)
	}
	if got.SessionID != "sess-1" {
		t.Fatalf("session_id=%q", got.SessionID)
	}
	rc, err := store.Get(ctx, sharedkernel.BlobRef{Key: "inputs/task-1/prompt.txt"})
	if err != nil {
		t.Fatalf("staged blob: %v", err)
	}
	_ = rc.Close()
	if _, err := sessSvc.Get(ctx, "tg:100"); err == nil {
		t.Fatal("session should be cleared after submit")
	}
	if len(pub.msgs) != 1 || pub.msgs[0].Topic != sharedkernel.TopicTaskCreated {
		t.Fatalf("msgs=%+v", pub.msgs)
	}
	var ev sharedkernel.TaskCreated
	if err := json.Unmarshal(pub.msgs[0].Payload, &ev); err != nil {
		t.Fatal(err)
	}
	if ev.TaskID != "task-1" || ev.ChatID != "tg:100" {
		t.Fatalf("event=%+v", ev)
	}
}
