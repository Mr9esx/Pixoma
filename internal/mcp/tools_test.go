package mcp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	domain "github.com/Mr9esx/Pixoma/internal/cases/domain"
	"github.com/Mr9esx/Pixoma/internal/cases/infrastructure/validation"
	pixmcp "github.com/Mr9esx/Pixoma/internal/mcp"
	"github.com/Mr9esx/Pixoma/internal/packaging/botapp"
	"github.com/Mr9esx/Pixoma/internal/platform/blob"
	"github.com/Mr9esx/Pixoma/internal/platform/blob/localfs"
	"github.com/Mr9esx/Pixoma/internal/platform/queue"
	convdomain "github.com/Mr9esx/Pixoma/internal/sessions/domain"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
	runtimedomain "github.com/Mr9esx/Pixoma/internal/tasks/domain"
	identitydomain "github.com/Mr9esx/Pixoma/internal/users/domain"
	"github.com/modelcontextprotocol/go-sdk/mcp"
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
func (r *memCases) List(_ context.Context, q domain.ListQuery) ([]*domain.Case, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*domain.Case
	for _, c := range r.m {
		if q.Enabled != nil && c.Enabled != *q.Enabled {
			continue
		}
		cp := *c
		out = append(out, &cp)
	}
	return out, nil
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

type memUsers struct {
	byID map[string]*identitydomain.User
}

func (r *memUsers) UpsertByChannelExternal(context.Context, identitydomain.UpsertFrom) (*identitydomain.User, error) {
	return nil, errors.New("unused")
}
func (r *memUsers) SetAccess(context.Context, string, identitydomain.UserAccess) (*identitydomain.User, error) {
	return nil, errors.New("unused")
}
func (r *memUsers) List(context.Context, identitydomain.ListQuery) ([]*identitydomain.User, error) {
	return nil, nil
}
func (r *memUsers) Delete(context.Context, string) error {
	return errors.New("unused")
}
func (r *memUsers) GetByID(_ context.Context, id string) (*identitydomain.User, error) {
	u, ok := r.byID[id]
	if !ok {
		return nil, identitydomain.ErrNotFound
	}
	cp := *u
	return &cp, nil
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

func mcpHarness(t *testing.T) (*botapp.Facade, *runtimedomain.MemoryTaskRepository, pixmcp.Identity) {
	t.Helper()
	ctx := context.Background()
	cases := &memCases{}
	_ = cases.Create(ctx, &domain.Case{Document: sampleDoc(), Enabled: true})
	store, err := localfs.New(filepath.Join(t.TempDir(), "blob"))
	if err != nil {
		t.Fatal(err)
	}
	user := &identitydomain.User{
		ID:             "user-mcp-1",
		ChannelID:      "ch-mcp",
		ExternalUserID: "ext-1",
		Access:         identitydomain.UserAccessAlwaysAllowed,
	}
	seq := 0
	tasks := runtimedomain.NewMemoryTaskRepository()
	facade := &botapp.Facade{
		Cases:        cases,
		Validator:    validation.New(),
		Users:        &memUsers{byID: map[string]*identitydomain.User{"user-mcp-1": user}},
		SessionStore: convdomain.NewMemoryRepository(),
		Tasks:        tasks,
		Blob:         store,
		Publisher:    &capturePub{},
		NewTaskID: func() sharedkernel.TaskID {
			seq++
			return sharedkernel.TaskID(fmt.Sprintf("task-mcp-%d", seq))
		},
		Now: func() time.Time { return time.Unix(20, 0).UTC() },
	}
	ident := pixmcp.Identity{UserID: "user-mcp-1", ChannelID: "ch-mcp", ExternalUserID: "ext-1"}
	return facade, tasks, ident
}

func connectMCP(t *testing.T, deps pixmcp.Deps) (*mcp.ClientSession, func()) {
	t.Helper()
	srv := httptest.NewServer(pixmcp.NewHandler(deps))
	ctx := context.Background()
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "v1"}, nil)
	cs, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: srv.URL + "/mcp"}, nil)
	if err != nil {
		srv.Close()
		t.Fatalf("connect: %v", err)
	}
	return cs, func() {
		_ = cs.Close()
		srv.Close()
	}
}

func TestMCP_RunWorkflow_ReturnsTaskID(t *testing.T) {
	facade, tasks, ident := mcpHarness(t)
	cs, cleanup := connectMCP(t, pixmcp.Deps{Facade: facade, Identity: ident})
	defer cleanup()

	ctx := context.Background()
	res, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name: "run_workflow",
		Arguments: map[string]any{
			"case_id": 1,
			"inputs":  map[string]any{"prompt": "a cat"},
		},
	})
	if err != nil {
		t.Fatalf("run_workflow: %v", err)
	}
	if res.IsError {
		t.Fatalf("run_workflow error: %+v", res.Content)
	}
	id := structuredString(t, res, "task_id")
	if id == "" {
		t.Fatal("empty task_id")
	}
	got, err := tasks.Get(ctx, sharedkernel.TaskID(id))
	if err != nil || got.Status != sharedkernel.TaskPending {
		t.Fatalf("task=%v err=%v", got, err)
	}
}

func TestMCP_RunWorkflow_MissingRequired(t *testing.T) {
	facade, tasks, ident := mcpHarness(t)
	cs, cleanup := connectMCP(t, pixmcp.Deps{Facade: facade, Identity: ident})
	defer cleanup()

	ctx := context.Background()
	res, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      "run_workflow",
		Arguments: map[string]any{"case_id": 1, "inputs": map[string]any{}},
	})
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected tool error for missing required")
	}
	list, err := tasks.ListByChat(ctx, ident.ChatID(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("created tasks=%d", len(list))
	}
}

func TestMCP_ListTasks_OnlyOwnUser(t *testing.T) {
	facade, _, ident := mcpHarness(t)
	other := pixmcp.Identity{UserID: "user-mcp-2", ChannelID: "ch-mcp", ExternalUserID: "ext-2"}
	otherUser := &identitydomain.User{
		ID:             "user-mcp-2",
		ChannelID:      "ch-mcp",
		ExternalUserID: "ext-2",
		Access:         identitydomain.UserAccessAlwaysAllowed,
	}
	users := facade.Users.(*memUsers)
	users.byID["user-mcp-2"] = otherUser

	owner, cleanupA := connectMCP(t, pixmcp.Deps{Facade: facade, Identity: ident})
	defer cleanupA()
	stranger, cleanupB := connectMCP(t, pixmcp.Deps{Facade: facade, Identity: other})
	defer cleanupB()

	ctx := context.Background()
	created, err := owner.CallTool(ctx, &mcp.CallToolParams{
		Name: "run_workflow",
		Arguments: map[string]any{
			"case_id": 1,
			"inputs":  map[string]any{"prompt": "a cat"},
		},
	})
	if err != nil || created.IsError {
		t.Fatalf("run: err=%v res=%+v", err, created)
	}
	taskID := structuredString(t, created, "task_id")

	listed, err := stranger.CallTool(ctx, &mcp.CallToolParams{Name: "list_tasks"})
	if err != nil || listed.IsError {
		t.Fatalf("list: err=%v res=%+v", err, listed)
	}
	if n := structuredLen(t, listed, "tasks"); n != 0 {
		t.Fatalf("stranger saw %d tasks", n)
	}
	got, err := stranger.CallTool(ctx, &mcp.CallToolParams{
		Name:      "get_task",
		Arguments: map[string]any{"task_id": taskID},
	})
	if err != nil {
		t.Fatalf("get_task: %v", err)
	}
	if !got.IsError {
		t.Fatal("stranger must not read another user's task")
	}
}

func TestMCP_GetTask_ReadsSucceededImage(t *testing.T) {
	facade, tasks, ident := mcpHarness(t)
	cs, cleanup := connectMCP(t, pixmcp.Deps{Facade: facade, Identity: ident})
	defer cleanup()

	ctx := context.Background()
	created, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name: "run_workflow",
		Arguments: map[string]any{
			"case_id": 1,
			"inputs":  map[string]any{"prompt": "a cat"},
		},
	})
	if err != nil || created.IsError {
		t.Fatalf("run: err=%v res=%+v", err, created)
	}
	taskID := structuredString(t, created, "task_id")
	task, err := tasks.Get(ctx, sharedkernel.TaskID(taskID))
	if err != nil {
		t.Fatal(err)
	}
	png := []byte("fake-png-bytes")
	ref, err := facade.Blob.Put(ctx, "outputs/"+taskID+"/0.png", bytes.NewReader(png), blob.PutOptions{MIME: "image/png"})
	if err != nil {
		t.Fatal(err)
	}
	if err := task.MarkQueued("", time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := task.MarkSucceeded([]runtimedomain.OutputRef{{Key: "image", Blob: ref}}, time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := tasks.Update(ctx, task); err != nil {
		t.Fatal(err)
	}

	got, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      "get_task",
		Arguments: map[string]any{"task_id": taskID},
	})
	if err != nil || got.IsError {
		t.Fatalf("get_task: err=%v res=%+v", err, got)
	}
	item := structuredMap(t, got)
	if item["status"] != string(sharedkernel.TaskSucceeded) {
		t.Fatalf("status=%v", item["status"])
	}
	outputs, _ := item["outputs"].([]any)
	if len(outputs) != 1 {
		t.Fatalf("outputs=%v", item["outputs"])
	}
	out, _ := outputs[0].(map[string]any)
	uri, _ := out["uri"].(string)
	wantURI := fmt.Sprintf("pixoma://task/%s/output/0", taskID)
	if uri != wantURI {
		t.Fatalf("uri=%q want %q", uri, wantURI)
	}

	rr, err := cs.ReadResource(ctx, &mcp.ReadResourceParams{URI: uri})
	if err != nil {
		t.Fatalf("read resource: %v", err)
	}
	if len(rr.Contents) != 1 {
		t.Fatalf("contents=%d", len(rr.Contents))
	}
	if !bytes.Equal(rr.Contents[0].Blob, png) {
		t.Fatalf("blob=%q", rr.Contents[0].Blob)
	}
}

func TestMCP_RunWorkflow_PaidUserDoesNotCreateTask(t *testing.T) {
	facade, tasks, ident := mcpHarness(t)
	users := facade.Users.(*memUsers)
	paid := users.byID[ident.UserID]
	paid.Access = identitydomain.UserAccessPaid

	cs, cleanup := connectMCP(t, pixmcp.Deps{Facade: facade, Identity: ident})
	defer cleanup()

	ctx := context.Background()
	res, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name: "run_workflow",
		Arguments: map[string]any{
			"case_id": 1,
			"inputs":  map[string]any{"prompt": "a cat"},
		},
	})
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	if !res.IsError {
		t.Fatal("paid user must not run workflow")
	}
	list, err := tasks.ListByChat(ctx, ident.ChatID(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("created tasks=%d", len(list))
	}
}

func toolErrorText(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	var b strings.Builder
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			b.WriteString(tc.Text)
			continue
		}
		b.WriteString(fmt.Sprint(c))
	}
	return b.String()
}

func TestMCP_RunWorkflow_PaidUserGuidesAdmin(t *testing.T) {
	facade, tasks, ident := mcpHarness(t)
	users := facade.Users.(*memUsers)
	paid := users.byID[ident.UserID]
	paid.Access = identitydomain.UserAccessPaid

	cs, cleanup := connectMCP(t, pixmcp.Deps{Facade: facade, Identity: ident})
	defer cleanup()

	ctx := context.Background()
	res, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name: "run_workflow",
		Arguments: map[string]any{
			"case_id": 1,
			"inputs":  map[string]any{"prompt": "a cat"},
		},
	})
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	if !res.IsError {
		t.Fatal("paid user must not run workflow")
	}
	got := toolErrorText(t, res)
	if !strings.Contains(got, "不能跑工作流") || !strings.Contains(got, "MCP 渠道用户") {
		t.Fatalf("guide=%q", got)
	}
	list, err := tasks.ListByChat(ctx, ident.ChatID(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("created tasks=%d", len(list))
	}
}

func TestMCP_RunWorkflow_DisabledGuidesList(t *testing.T) {
	facade, tasks, ident := mcpHarness(t)
	ctx := context.Background()
	c, err := facade.Cases.Get(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	c.Enabled = false
	if err := facade.Cases.Save(ctx, c); err != nil {
		t.Fatal(err)
	}
	cs, cleanup := connectMCP(t, pixmcp.Deps{Facade: facade, Identity: ident})
	defer cleanup()
	res, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name: "run_workflow",
		Arguments: map[string]any{
			"case_id": 1,
			"inputs":  map[string]any{"prompt": "a cat"},
		},
	})
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected error")
	}
	got := toolErrorText(t, res)
	if !strings.Contains(got, "list_workflows") || !strings.Contains(got, "停用") {
		t.Fatalf("guide=%q", got)
	}
	list, err := tasks.ListByChat(ctx, ident.ChatID(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("created tasks=%d", len(list))
	}
}

func TestMCP_GetWorkflow_MissingGuidesList(t *testing.T) {
	facade, _, ident := mcpHarness(t)
	cs, cleanup := connectMCP(t, pixmcp.Deps{Facade: facade, Identity: ident})
	defer cleanup()
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "get_workflow",
		Arguments: map[string]any{"case_id": 999},
	})
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected error")
	}
	got := toolErrorText(t, res)
	if !strings.Contains(got, "list_workflows") {
		t.Fatalf("guide=%q", got)
	}
}

func TestMCP_RunWorkflow_MissingRequiredGuidesGetWorkflow(t *testing.T) {
	facade, tasks, ident := mcpHarness(t)
	cs, cleanup := connectMCP(t, pixmcp.Deps{Facade: facade, Identity: ident})
	defer cleanup()
	ctx := context.Background()
	res, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      "run_workflow",
		Arguments: map[string]any{"case_id": 1, "inputs": map[string]any{}},
	})
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected tool error for missing required")
	}
	got := toolErrorText(t, res)
	if !strings.Contains(got, "prompt") || !strings.Contains(got, "get_workflow") || !strings.Contains(got, "inputs") {
		t.Fatalf("guide=%q", got)
	}
	list, err := tasks.ListByChat(ctx, ident.ChatID(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("created tasks=%d", len(list))
	}
}

func TestMCP_GetTask_ForeignOrEmptyGuidesListTasks(t *testing.T) {
	facade, _, ident := mcpHarness(t)
	other := pixmcp.Identity{UserID: "user-mcp-2", ChannelID: "ch-mcp", ExternalUserID: "ext-2"}
	users := facade.Users.(*memUsers)
	users.byID["user-mcp-2"] = &identitydomain.User{
		ID:             "user-mcp-2",
		ChannelID:      "ch-mcp",
		ExternalUserID: "ext-2",
		Access:         identitydomain.UserAccessAlwaysAllowed,
	}
	owner, cleanupA := connectMCP(t, pixmcp.Deps{Facade: facade, Identity: ident})
	defer cleanupA()
	stranger, cleanupB := connectMCP(t, pixmcp.Deps{Facade: facade, Identity: other})
	defer cleanupB()
	ctx := context.Background()
	created, err := owner.CallTool(ctx, &mcp.CallToolParams{
		Name: "run_workflow",
		Arguments: map[string]any{
			"case_id": 1,
			"inputs":  map[string]any{"prompt": "a cat"},
		},
	})
	if err != nil || created.IsError {
		t.Fatalf("run: err=%v res=%+v", err, created)
	}
	taskID := structuredString(t, created, "task_id")
	got, err := stranger.CallTool(ctx, &mcp.CallToolParams{
		Name:      "get_task",
		Arguments: map[string]any{"task_id": taskID},
	})
	if err != nil {
		t.Fatalf("get_task: %v", err)
	}
	if !got.IsError {
		t.Fatal("stranger must not read another user's task")
	}
	text := toolErrorText(t, got)
	if !strings.Contains(text, "连接器") && !strings.Contains(text, "list_tasks") {
		t.Fatalf("foreign guide=%q", text)
	}
	empty, err := owner.CallTool(ctx, &mcp.CallToolParams{
		Name:      "get_task",
		Arguments: map[string]any{"task_id": ""},
	})
	if err != nil {
		t.Fatalf("empty: %v", err)
	}
	if !empty.IsError {
		t.Fatal("empty task_id must error")
	}
	emptyText := toolErrorText(t, empty)
	if !strings.Contains(emptyText, "list_tasks") && !strings.Contains(emptyText, "task_id") {
		t.Fatalf("empty guide=%q", emptyText)
	}
}

func structuredString(t *testing.T, res *mcp.CallToolResult, key string) string {
	t.Helper()
	m := structuredMap(t, res)
	v, _ := m[key].(string)
	return v
}

func structuredLen(t *testing.T, res *mcp.CallToolResult, key string) int {
	t.Helper()
	m := structuredMap(t, res)
	switch v := m[key].(type) {
	case []any:
		return len(v)
	default:
		return 0
	}
}

func structuredMap(t *testing.T, res *mcp.CallToolResult) map[string]any {
	t.Helper()
	if res.StructuredContent == nil {
		t.Fatal("no structured content")
	}
	raw, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	return m
}
