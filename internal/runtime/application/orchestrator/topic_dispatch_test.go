package orchestrator_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/mr9esx/comfyui_tgbot/internal/catalog/domain"
	convdomain "github.com/mr9esx/comfyui_tgbot/internal/conversation/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/edge"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/edge/static"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/application/orchestrator"
	runtimedomain "github.com/mr9esx/comfyui_tgbot/internal/runtime/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/domain/condition"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

type caseReaderStub struct {
	docs map[sharedkernel.CaseID]*domain.CaseDocument
}

func (c *caseReaderStub) GetCase(_ context.Context, id sharedkernel.CaseID) (*domain.CaseDocument, error) {
	if c.docs == nil {
		return nil, nil
	}
	return c.docs[id], nil
}

func buildTopicSvc(t *testing.T, reg *static.Registry, cases *caseReaderStub, cond *condition.Registry) (*orchestrator.Service, *runtimedomain.MemoryTaskRepository) {
	t.Helper()
	tasks := runtimedomain.NewMemoryTaskRepository()
	bus := &captureBus{}
	n := &memNotify{}
	svc := orchestrator.New(tasks, reg, bus, n)
	svc.Now = func() time.Time { return time.Unix(50, 0).UTC() }
	svc.Prep = stubPrep{}
	svc.Cases = cases
	svc.Condition = cond
	sessions := convdomain.NewMemoryRepository()
	_ = sessions.Save(context.Background(), &convdomain.Session{
		ID:     "s1",
		UserID: "u1",
		ChatID: "tg:1",
	})
	svc.Sessions = sessions
	return svc, tasks
}

type topicStubProvider struct {
	val func(ctx context.Context, field string) (any, error)
}

func (s *topicStubProvider) Namespace() string { return "user" }
func (s *topicStubProvider) ListAttributes() []condition.AttributeDescriptor {
	return []condition.AttributeDescriptor{{
		Key:     "user.level",
		Context: "user",
		Label:   "Level",
		Schema:  map[string]any{"type": "string", "enum": []any{"image", "video"}},
	}}
}
func (s *topicStubProvider) Value(ctx context.Context, field string) (any, error) {
	if s.val == nil {
		return nil, condition.ErrAttributeMissing
	}
	return s.val(ctx, field)
}

func TestDispatch_RoutesToTopicWithoutEdgeBinding(t *testing.T) {
	ctx := context.Background()
	reg := static.New(edge.Instance{ID: "gpu-1", SubscribeTopics: []string{"fast-gpu"}})
	cases := &caseReaderStub{docs: map[sharedkernel.CaseID]*domain.CaseDocument{
		1: {
			ID: 1,
			Routing: &domain.RoutingConfig{Rules: []domain.RoutingRule{
				{When: json.RawMessage(`{"field":"user.level","op":"eq","value":"image"}`), Topic: "fast-gpu"},
			}},
		},
	}}
	cond := condition.NewRegistry()
	cond.Register(&topicStubProvider{val: func(context.Context, string) (any, error) {
		return "image", nil
	}})
	svc, tasks := buildTopicSvc(t, reg, cases, cond)

	now := time.Unix(50, 0).UTC()
	_ = tasks.Create(ctx, runtimedomain.NewPending("t1", "s1", sharedkernel.CaseID(1), "inputs/t1", now))
	if err := svc.OnTaskCreated(ctx, sharedkernel.TaskCreated{TaskID: "t1"}); err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	got, _ := tasks.Get(ctx, "t1")
	if got.Status != sharedkernel.TaskQueued {
		t.Fatalf("status = %s", got.Status)
	}
	if got.DispatchTopic != "fast-gpu" {
		t.Fatalf("dispatch_topic = %q", got.DispatchTopic)
	}
	if got.EdgeID != "" {
		t.Fatalf("edge_id must be empty, got %q", got.EdgeID)
	}
	if got.JobRef.Key == "" {
		t.Fatal("job_ref missing")
	}
}

func TestDispatch_EvalErrorKeepsPending(t *testing.T) {
	ctx := context.Background()
	reg := static.New(edge.Instance{ID: "gpu-1", SubscribeTopics: []string{"fast-gpu"}})
	cases := &caseReaderStub{docs: map[sharedkernel.CaseID]*domain.CaseDocument{
		1: {
			ID: 1,
			Routing: &domain.RoutingConfig{Rules: []domain.RoutingRule{
				{When: json.RawMessage(`{"field":"user.level","op":"eq","value":"image"}`), Topic: "fast-gpu"},
			}},
		},
	}}
	cond := condition.NewRegistry()
	cond.Register(&topicStubProvider{val: func(context.Context, string) (any, error) {
		return nil, errors.New("provider boom")
	}})
	svc, tasks := buildTopicSvc(t, reg, cases, cond)

	now := time.Unix(50, 0).UTC()
	_ = tasks.Create(ctx, runtimedomain.NewPending("t2", "s1", sharedkernel.CaseID(1), "inputs/t2", now))
	if err := svc.OnTaskCreated(ctx, sharedkernel.TaskCreated{TaskID: "t2"}); err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	got, _ := tasks.Get(ctx, "t2")
	if got.Status != sharedkernel.TaskPending {
		t.Fatalf("status = %s, want pending", got.Status)
	}
	if !strings.Contains(got.ErrorMessage, "routing:") {
		t.Fatalf("error_message = %q", got.ErrorMessage)
	}
}

func TestDispatch_EmptyRulesFailsWithoutFallback(t *testing.T) {
	ctx := context.Background()
	reg := static.New(edge.Instance{ID: "gpu-1", SubscribeTopics: []string{"default"}})
	cases := &caseReaderStub{docs: map[sharedkernel.CaseID]*domain.CaseDocument{
		1: {ID: 1, Routing: &domain.RoutingConfig{}},
	}}
	svc, tasks := buildTopicSvc(t, reg, cases, condition.NewRegistry())

	now := time.Unix(50, 0).UTC()
	_ = tasks.Create(ctx, runtimedomain.NewPending("t-empty", "s1", sharedkernel.CaseID(1), "inputs/t-empty", now))
	if err := svc.OnTaskCreated(ctx, sharedkernel.TaskCreated{TaskID: "t-empty"}); err != nil {
		t.Fatalf("dispatch: %v", err)
	}

	got, _ := tasks.Get(ctx, "t-empty")
	if got.Status != sharedkernel.TaskFailed {
		t.Fatalf("status = %s, want failed", got.Status)
	}
	if got.ErrorCode != sharedkernel.TaskErrorRoutingNoMatch {
		t.Fatalf("error_code = %q, want %q", got.ErrorCode, sharedkernel.TaskErrorRoutingNoMatch)
	}
	if got.ErrorMessage != sharedkernel.TaskErrorMessageRoutingNoMatch {
		t.Fatalf("error_message = %q", got.ErrorMessage)
	}
}

func TestDispatch_UnmatchedRulesFailsWithoutFallback(t *testing.T) {
	ctx := context.Background()
	reg := static.New(edge.Instance{ID: "gpu-1", SubscribeTopics: []string{"default"}})
	cases := &caseReaderStub{docs: map[sharedkernel.CaseID]*domain.CaseDocument{
		1: {
			ID: 1,
			Routing: &domain.RoutingConfig{Rules: []domain.RoutingRule{
				{When: json.RawMessage(`{"field":"user.level","op":"eq","value":"image"}`), Topic: "fast-gpu"},
			}},
		},
	}}
	cond := condition.NewRegistry()
	cond.Register(&topicStubProvider{val: func(context.Context, string) (any, error) {
		return "", nil
	}})
	svc, tasks := buildTopicSvc(t, reg, cases, cond)

	now := time.Unix(50, 0).UTC()
	_ = tasks.Create(ctx, runtimedomain.NewPending("t-unmatched", "s1", sharedkernel.CaseID(1), "inputs/t-unmatched", now))
	if err := svc.OnTaskCreated(ctx, sharedkernel.TaskCreated{TaskID: "t-unmatched"}); err != nil {
		t.Fatalf("dispatch: %v", err)
	}

	got, _ := tasks.Get(ctx, "t-unmatched")
	if got.Status != sharedkernel.TaskFailed {
		t.Fatalf("status = %s, want failed", got.Status)
	}
	if got.ErrorCode != sharedkernel.TaskErrorRoutingNoMatch {
		t.Fatalf("error_code = %q, want %q", got.ErrorCode, sharedkernel.TaskErrorRoutingNoMatch)
	}
	if got.ErrorMessage != sharedkernel.TaskErrorMessageRoutingNoMatch {
		t.Fatalf("error_message = %q", got.ErrorMessage)
	}
}

func TestDispatch_MissingRoutingDependenciesDoesNotFallback(t *testing.T) {
	ctx := context.Background()
	reg := static.New(edge.Instance{ID: "gpu-1", SubscribeTopics: []string{"default"}})
	svc, tasks := buildTopicSvc(t, reg, &caseReaderStub{}, condition.NewRegistry())
	svc.Cases = nil
	svc.Condition = nil

	now := time.Unix(50, 0).UTC()
	_ = tasks.Create(ctx, runtimedomain.NewPending("t-no-router", "s1", sharedkernel.CaseID(1), "inputs/t-no-router", now))
	if err := svc.OnTaskCreated(ctx, sharedkernel.TaskCreated{TaskID: "t-no-router"}); err != nil {
		t.Fatalf("dispatch: %v", err)
	}

	got, _ := tasks.Get(ctx, "t-no-router")
	if got.Status != sharedkernel.TaskFailed {
		t.Fatalf("status = %s, want failed", got.Status)
	}
	if got.ErrorCode != sharedkernel.TaskErrorRoutingNoMatch {
		t.Fatalf("error_code = %q, want %q", got.ErrorCode, sharedkernel.TaskErrorRoutingNoMatch)
	}
}

func TestDispatch_NoOnlineConsumerKeepsPending(t *testing.T) {
	ctx := context.Background()
	// Node exists but is offline (Online returns false).
	reg := static.New(edge.Instance{ID: "gpu-1", SubscribeTopics: []string{"fast-gpu"}})
	cases := &caseReaderStub{docs: map[sharedkernel.CaseID]*domain.CaseDocument{
		1: {
			ID: 1,
			Routing: &domain.RoutingConfig{Rules: []domain.RoutingRule{
				{When: json.RawMessage(`{"field":"user.level","op":"eq","value":"image"}`), Topic: "fast-gpu"},
			}},
		},
	}}
	cond := condition.NewRegistry()
	cond.Register(&topicStubProvider{val: func(context.Context, string) (any, error) {
		return "image", nil
	}})
	svc, tasks := buildTopicSvc(t, reg, cases, cond)
	svc.Online = func(context.Context, sharedkernel.EdgeID) bool { return false }

	now := time.Unix(50, 0).UTC()
	_ = tasks.Create(ctx, runtimedomain.NewPending("t3", "s1", sharedkernel.CaseID(1), "inputs/t3", now))
	if err := svc.OnTaskCreated(ctx, sharedkernel.TaskCreated{TaskID: "t3"}); err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	got, _ := tasks.Get(ctx, "t3")
	if got.Status != sharedkernel.TaskPending {
		t.Fatalf("status = %s, want pending", got.Status)
	}
	if !strings.Contains(got.ErrorMessage, "no online consumer") {
		t.Fatalf("error_message = %q", got.ErrorMessage)
	}
}
