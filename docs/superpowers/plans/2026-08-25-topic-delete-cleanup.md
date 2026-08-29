# 任务队列（Topic）清理式删除 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Topic 删除改为「确认制清理式删除」：弹窗实时展示工作流规则/节点订阅/排队任务影响，确认后事务内整条删除指向该 Topic 的路由规则、移除节点订阅、queued 任务终态失败（`topic_deleted`）并通知、删除 Topic 行；错误改走 toast + code 本地化。

**Architecture:** 新增 `internal/topicadmin`（仿 edgeadmin，gdb 事务 + notify）执行清理；`topics` handler 注入 `DeleteWithCleanup` 并返回摘要；tasks 管理 API 增加 `dispatch_topic` 过滤供弹窗统计排队任务；前端 `topic-detail-panel` 删除弹窗 toast 化 + 影响清单 + 勾选确认。

**Tech Stack:** Go 1.x + GORM（SQLite 测试）、chi、React 18 + TanStack Query + react-i18next + sonner、Vitest（node 合同测试）、pnpm。

## Global Constraints

- 不新增第三方依赖。
- 后端 API 错误 `error` 字段保持英文兜底；用户可见文案走前端 i18n。
- 新增 i18n key 必须 zh/en 成对，并登记进 `web/admin/src/lib/i18n/locale.test.ts`。
- 规则处理采用用户已确认的 **A：整条删除**，不重定向到 default。
- queued 任务终态失败且不重试；running/pending 不处理（pending 规则删除后自然回退 default）。
- default Topic 始终不可删除。
- 前端 node 测试文件必须已在 `vitest.config.ts` include（本计划只修改现有测试文件）。
- 每个任务独立可测交付 + 单独 commit；`apps/pixoma/cmd/pixoma/main.go` 用 `git add -p` 只暂存本任务 hunk。

---

### Task 1: sharedkernel 常量 + tasks API dispatch_topic 过滤

**Files:**
- Modify: `internal/sharedkernel/ids.go`
- Modify: `internal/runtime/domain/task_repository.go`（AdminListQuery + Memory List）
- Modify: `internal/runtime/infrastructure/persistence/gorm_task.go`
- Modify: `internal/httpapi/tasks/handler.go`
- Modify: `internal/httpapi/tasks/handler_test.go`

**Interfaces:**
- Produces: `sharedkernel.TaskErrorTopicDeleted`、`TopicDeletedMessage`、`GET /api/v1/tasks?dispatch_topic=X` —— Task 2/4/6 依赖

- [ ] **Step 1: 常量 + 失败测试**

`sharedkernel/ids.go` 追加：

```go
// Terminal-failure metadata used when a dispatch topic is deleted.
const (
	TaskErrorTopicDeleted = "topic_deleted"
	TopicDeletedMessage   = "任务队列已删除"
)
```

`AdminListQuery` 加 `DispatchTopic string`；`gorm_task.go` 的 `List` 内（`if q.ChannelID != ""` 分支之后、普通过滤区）加：

```go
	if q.DispatchTopic != "" {
		tx = tx.Where(col("dispatch_topic")+" = ?", q.DispatchTopic)
	}
```

`MemoryTaskRepository.List` 内加：

```go
		if q.DispatchTopic != "" && t.DispatchTopic != q.DispatchTopic {
			continue
		}
```

`handler.go` 的 `parseAdminListQuery` 内（`chat_id` 之后）加：

```go
	if v := r.URL.Query().Get("dispatch_topic"); v != "" {
		q.DispatchTopic = v
	}
```

`handler_test.go` 的 `TestTasksHandler_ListGetCancel` 内（现有断言之后）追加：

```go
	queued := runtimedomain.NewPending("t-q", "sess-1", 1, "pfx", now)
	queued.Status = sharedkernel.TaskQueued
	queued.DispatchTopic = "fast-gpu"
	if err := tasks.Create(ctx, queued); err != nil {
		t.Fatal(err)
	}

	resQ, err := http.Get(srv.URL + "/api/v1/tasks?dispatch_topic=fast-gpu&status=queued")
	if err != nil {
		t.Fatal(err)
	}
	defer resQ.Body.Close()
	if resQ.StatusCode != http.StatusOK {
		t.Fatalf("dispatch filter status=%d", resQ.StatusCode)
	}
	var qList []map[string]any
	if err := json.NewDecoder(resQ.Body).Decode(&qList); err != nil {
		t.Fatal(err)
	}
	if len(qList) != 1 || qList[0]["id"] != "t-q" {
		t.Fatalf("dispatch filter: %+v", qList)
	}
```

（`TestTasksHandler_ListGetCancel` 用的是 MemoryTaskRepository，Status 直接赋值即可；`queued.Status`/`DispatchTopic` 字段存在。）

- [ ] **Step 2: 运行确认失败**

Run: `go test ./internal/httpapi/tasks/ -run TestTasksHandler_ListGetCancel -v`
Expected: FAIL（过滤未生效，返回 2 条或 0 条）。

- [ ] **Step 3: 重跑测试**

Run: `go test ./internal/httpapi/tasks/... ./internal/runtime/... ./internal/sharedkernel/`
Expected: PASS。

- [ ] **Step 4: Commit**

```bash
git add internal/sharedkernel/ids.go internal/runtime/domain/task_repository.go internal/runtime/infrastructure/persistence/gorm_task.go internal/httpapi/tasks/handler.go internal/httpapi/tasks/handler_test.go
git commit -m "feat(tasks): filter admin list by dispatch_topic; topic-deleted constants"
```

---

### Task 2: topicadmin 清理服务

**Files:**
- Create: `internal/topicadmin/topic_delete.go`
- Create: `internal/topicadmin/topic_delete_test.go`

**Interfaces:**
- Consumes: `topicpersist.TopicRepository`、`casepersist.GormRepository`、`instpersist.EdgeRepository`、`taskpersist.TaskRepository`、`sesspersist.SessionRepository`、`sharedkernel.TaskErrorTopicDeleted/TopicDeletedMessage`（Task 1）、`notify.Publisher`
- Produces: `topicadmin.ErrNeedsAck`、`ErrDefaultProtected`、`DeleteSummary`、`Service.DeleteTopic` —— Task 3 依赖

- [ ] **Step 1: 失败测试**

`topic_delete_test.go`（模式仿 `edgeadmin_test`；AutoMigrate topics + cases + edges + tasks + sessions）：

```go
package topicadmin_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"gorm.io/gorm"

	catalogdomain "github.com/mr9esx/comfyui_tgbot/internal/catalog/domain"
	casepersist "github.com/mr9esx/comfyui_tgbot/internal/catalog/infrastructure/persistence"
	convdomain "github.com/mr9esx/comfyui_tgbot/internal/conversation/domain"
	sesspersist "github.com/mr9esx/comfyui_tgbot/internal/conversation/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/db"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/edge"
	instpersist "github.com/mr9esx/comfyui_tgbot/internal/platform/edge/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/topic"
	topicpersist "github.com/mr9esx/comfyui_tgbot/internal/platform/topic/persistence"
	runtimedomain "github.com/mr9esx/comfyui_tgbot/internal/runtime/domain"
	taskpersist "github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
	"github.com/mr9esx/comfyui_tgbot/internal/topicadmin"
)

type captureNotify struct {
	items []sharedkernel.UserNotify
}

func (c *captureNotify) Publish(_ context.Context, n sharedkernel.UserNotify) error {
	c.items = append(c.items, n)
	return nil
}

func newTopicTest(t *testing.T) (*gorm.DB, *topicadmin.Service, *captureNotify, context.Context, time.Time) {
	t.Helper()
	gdb, err := db.Open(db.Options{DSN: "file:topic_delete_" + t.Name() + "?mode=memory&cache=shared"})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(gdb,
		&topicpersist.TopicRow{},
		&casepersist.CaseRow{},
		&instpersist.EdgeRow{},
		&taskpersist.TaskRow{},
		&sesspersist.SessionRow{}); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	now := time.Now().UTC()
	n := &captureNotify{}
	return gdb, topicadmin.NewService(gdb, n), n, ctx, now
}

func seedTopicData(t *testing.T, gdb *gorm.DB, now time.Time) {
	t.Helper()
	ctx := context.Background()
	topicRepo := topicpersist.NewTopicRepository(gdb)
	_ = topicRepo.Create(ctx, topic.Topic{Key: "default", Name: "默认", Enabled: true, CreatedAt: now, UpdatedAt: now})
	_ = topicRepo.Create(ctx, topic.Topic{Key: "fast-gpu", Name: "F", Enabled: true, CreatedAt: now, UpdatedAt: now})

	caseRepo := casepersist.NewGormRepository(gdb)
	_ = caseRepo.Save(ctx, &catalogdomain.Case{Document: catalogdomain.CaseDocument{
		ID:   1,
		Name: "c1",
		Routing: &catalogdomain.RoutingConfig{Rules: []catalogdomain.RoutingRule{
			{When: nil, Topic: "fast-gpu"},
			{When: nil, Topic: "default"},
		}},
	}, Enabled: true})

	edgeRepo := instpersist.NewEdgeRepository(gdb)
	_ = edgeRepo.Upsert(ctx, &edge.Record{
		ID: "gpu-1", Name: "gpu-1", Enabled: true,
		Capabilities:   []string{"comfy"},
		SubscribeTopics: []string{"fast-gpu", "default"},
		CreatedAt:      now, UpdatedAt: now,
	})

	taskRepo := taskpersist.NewTaskRepository(gdb)
	sessions := sesspersist.NewSessionRepository(gdb)
	_ = sessions.Save(ctx, convdomain.NewCollecting("s1", "tg:9", 1, []string{"a"}, now))
	queued := runtimedomain.NewPending("t1", "s1", 1, "inputs/t1", now)
	queued.Status = sharedkernel.TaskQueued
	queued.DispatchTopic = "fast-gpu"
	queued.JobRef = sharedkernel.BlobRef{Key: "jobs/t1/job.json"}
	_ = taskRepo.Create(ctx, queued)
}

func TestDeleteTopic_AckRequired(t *testing.T) {
	gdb, svc, _, ctx, now := newTopicTest(t)
	seedTopicData(t, gdb, now)

	_, err := svc.DeleteTopic(ctx, "fast-gpu", false)
	if !errors.Is(err, topicadmin.ErrNeedsAck) {
		t.Fatalf("err=%v want ErrNeedsAck", err)
	}
	topicRepo := topicpersist.NewTopicRepository(gdb)
	if _, err := topicRepo.Get(ctx, "fast-gpu"); err != nil {
		t.Fatalf("topic must survive rollback: %v", err)
	}
}

func TestDeleteTopic_CleanupAndDelete(t *testing.T) {
	gdb, svc, n, ctx, now := newTopicTest(t)
	seedTopicData(t, gdb, now)

	summary, err := svc.DeleteTopic(ctx, "fast-gpu", true)
	if err != nil {
		t.Fatal(err)
	}
	if summary.RemovedCaseRules != 1 || summary.RemovedEdgeSubs != 1 || summary.FailedTasks != 1 {
		t.Fatalf("summary=%+v", summary)
	}
	topicRepo := topicpersist.NewTopicRepository(gdb)
	if _, err := topicRepo.Get(ctx, "fast-gpu"); !errors.Is(err, topic.ErrTopicNotFound) {
		t.Fatalf("topic delete err=%v", err)
	}
	caseRepo := casepersist.NewGormRepository(gdb)
	c, err := caseRepo.Get(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Document.Routing.Rules) != 1 || c.Document.Routing.Rules[0].Topic != "default" {
		t.Fatalf("case rules=%+v", c.Document.Routing.Rules)
	}
	edgeRepo := instpersist.NewEdgeRepository(gdb)
	rec, err := edgeRepo.Get(ctx, "gpu-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(rec.SubscribeTopics) != 1 || rec.SubscribeTopics[0] != "default" {
		t.Fatalf("edge topics=%+v", rec.SubscribeTopics)
	}
	taskRepo := taskpersist.NewTaskRepository(gdb)
	t1, err := taskRepo.Get(ctx, "t1")
	if err != nil || t1.Status != sharedkernel.TaskFailed || t1.ErrorCode != sharedkernel.TaskErrorTopicDeleted {
		t.Fatalf("queued task=%+v err=%v", t1, err)
	}
	if len(n.items) != 1 || n.items[0].Kind != "task_failed" || n.items[0].ChatID != "tg:9" {
		t.Fatalf("notifies=%+v", n.items)
	}
}

func TestDeleteTopic_DefaultProtected(t *testing.T) {
	gdb, svc, _, ctx, now := newTopicTest(t)
	seedTopicData(t, gdb, now)
	_, err := svc.DeleteTopic(ctx, "default", true)
	if !errors.Is(err, topicadmin.ErrDefaultProtected) {
		t.Fatalf("err=%v want ErrDefaultProtected", err)
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `go test ./internal/topicadmin/ -v`
Expected: FAIL（包不存在）。

- [ ] **Step 3: 实现 topic_delete.go**

```go
// Package topicadmin implements admin dispatch-topic lifecycle operations
// that span topic/case/edge/task repositories in a single transaction.
package topicadmin

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"gorm.io/gorm"

	catalogdomain "github.com/mr9esx/comfyui_tgbot/internal/catalog/domain"
	casepersist "github.com/mr9esx/comfyui_tgbot/internal/catalog/infrastructure/persistence"
	sesspersist "github.com/mr9esx/comfyui_tgbot/internal/conversation/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/edge"
	instpersist "github.com/mr9esx/comfyui_tgbot/internal/platform/edge/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/notify"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/topic"
	topicpersist "github.com/mr9esx/comfyui_tgbot/internal/platform/topic/persistence"
	runtimedomain "github.com/mr9esx/comfyui_tgbot/internal/runtime/domain"
	taskpersist "github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

// ErrNeedsAck is returned when the topic is referenced by cases or edges and
// the caller did not confirm cleanup.
var ErrNeedsAck = errors.New("topic delete needs ack for references")

// ErrDefaultProtected is returned when deleting the system default topic.
var ErrDefaultProtected = errors.New("default topic cannot be deleted")

// DeleteSummary reports what the cleanup did.
type DeleteSummary struct {
	RemovedCaseRules int `json:"removed_case_rules"`
	RemovedEdgeSubs  int `json:"removed_edge_subscriptions"`
	FailedTasks      int `json:"failed_tasks"`
}

// Service deletes a dispatch topic with cleanup in one DB transaction.
type Service struct {
	db     *gorm.DB
	notify notify.Publisher
	now    func() time.Time
}

// NewService constructs a delete service over the shared database handle.
func NewService(db *gorm.DB, n notify.Publisher) *Service {
	return &Service{db: db, notify: n, now: func() time.Time { return time.Now().UTC() }}
}

// DeleteTopic removes case routing rules targeting the topic, unsubscribes
// edges, fails queued tasks (topic_deleted) and deletes the topic row
// atomically. Notifications are sent best-effort after commit.
func (s *Service) DeleteTopic(ctx context.Context, key string, ack bool) (DeleteSummary, error) {
	var summary DeleteSummary
	var taskNotifies []sharedkernel.UserNotify

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		topicRepo := topicpersist.NewTopicRepository(tx)
		caseRepo := casepersist.NewGormRepository(tx)
		edgeRepo := instpersist.NewEdgeRepository(tx)
		taskRepo := taskpersist.NewTaskRepository(tx)
		sessRepo := sesspersist.NewSessionRepository(tx)

		if _, err := topicRepo.Get(ctx, key); err != nil {
			return err
		}
		if key == topic.DefaultKey {
			return ErrDefaultProtected
		}

		cases, err := caseRepo.List(ctx, catalogdomain.ListQuery{})
		if err != nil {
			return err
		}
		edges, err := edgeRepo.List(ctx)
		if err != nil {
			return err
		}

		var caseRefs, edgeRefs int
		for _, c := range cases {
			for _, rule := range c.Document.RoutingRules() {
				if rule.Topic == key {
					caseRefs++
				}
			}
		}
		for _, e := range edges {
			if containsTopic(e.SubscribeTopics, key) {
				edgeRefs++
			}
		}
		if caseRefs+edgeRefs > 0 && !ack {
			return ErrNeedsAck
		}

		now := s.now()
		for _, c := range cases {
			rules := c.Document.Routing.Rules
			kept := rules[:0]
			for _, rule := range rules {
				if rule.Topic != key {
					kept = append(kept, rule)
				}
			}
			if len(kept) == len(rules) {
				continue
			}
			summary.RemovedCaseRules += len(rules) - len(kept)
			c.Document.Routing.Rules = kept
			if err := caseRepo.Save(ctx, c); err != nil {
				return err
			}
		}
		for _, e := range edges {
			if !containsTopic(e.SubscribeTopics, key) {
				continue
			}
			topics := removeTopic(e.SubscribeTopics, key)
			if err := edgeRepo.UpdateSubscribeTopics(ctx, e.ID, topics); err != nil {
				return err
			}
			summary.RemovedEdgeSubs++
		}

		queued, err := taskRepo.ListByTopic(ctx, key, runtimedomain.ListByTopicQuery{Status: sharedkernel.TaskQueued})
		if err != nil {
			return err
		}
		for _, t := range queued {
			if err := t.MarkFailed(sharedkernel.TaskErrorTopicDeleted, sharedkernel.TopicDeletedMessage, now); err != nil {
				continue
			}
			if err := taskRepo.Update(ctx, t); err != nil {
				return err
			}
			summary.FailedTasks++
			chatID := t.ChatID
			if chatID == "" {
				if sess, err := sessRepo.GetByID(ctx, t.SessionID); err == nil && sess != nil {
					chatID = sess.ChatID
				}
			}
			taskNotifies = append(taskNotifies, sharedkernel.UserNotify{
				ChatID:   chatID,
				TaskID:   t.ID,
				Kind:     "task_failed",
				ErrorMsg: t.ErrorMessage,
			})
		}
		return topicRepo.Delete(ctx, key)
	})
	if err != nil {
		return DeleteSummary{}, err
	}

	for _, n := range taskNotifies {
		if n.ChatID == "" {
			continue
		}
		if err := s.notify.Publish(ctx, n); err != nil {
			slog.Warn("topic delete: task notify failed", "task", n.TaskID, "err", err)
		}
	}
	return summary, nil
}

func containsTopic(topics []string, key string) bool {
	for _, t := range topics {
		if t == key {
			return true
		}
	}
	return false
}

func removeTopic(topics []string, key string) []string {
	var out []string
	for _, t := range topics {
		if t != key {
			out = append(out, t)
		}
	}
	return out
}
```

> 注意：`c.Document.RoutingRules()` 是计划中假设的辅助方法；实际实现直接读 `c.Document.Routing.Rules`（若 `Routing == nil` 则跳过该 case，防御式判断）。

- [ ] **Step 4: 修正实现细节后重跑**

把 Step 3 中引用计数的循环改为直接处理 `c.Document.Routing`（nil 检查），删除 `RoutingRules()` 假设：

```go
		for _, c := range cases {
			if c.Document.Routing == nil {
				continue
			}
			for _, rule := range c.Document.Routing.Rules {
				if rule.Topic == key {
					caseRefs++
				}
			}
		}
```

清理段同样加 nil 检查。

Run: `go test ./internal/topicadmin/ -v`
Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add internal/topicadmin/topic_delete.go internal/topicadmin/topic_delete_test.go
git commit -m "feat(topicadmin): delete topic with rule/subscription/task cleanup"
```

---

### Task 3: topics handler + main.go 接线

**Files:**
- Modify: `internal/httpapi/topics/handler.go`
- Modify: `internal/httpapi/topics/handler_test.go`
- Modify: `apps/pixoma/cmd/pixoma/main.go`

**Interfaces:**
- Consumes: `topicadmin.Service.DeleteTopic`、`ErrNeedsAck`、`ErrDefaultProtected`、`DeleteSummary`（Task 2）
- Produces: `DELETE /api/v1/topics/{key}` 新契约

- [ ] **Step 1: 失败测试**

`handler_test.go` 替换 `TestTopics_DeleteProtections` 与 `newTestRouter`：

```go
func newTestRouter(repo *fakeRepo, deleteFn func(ctx context.Context, key string, ack bool) (topicadmin.DeleteSummary, error)) http.Handler {
	h := &topics.Handler{
		Repo:              repo,
		DeleteWithCleanup: deleteFn,
	}
	r := chi.NewRouter()
	r.Route("/api/v1/topics", func(r chi.Router) { h.Mount(r) })
	return r
}

func TestTopics_DeleteCleanup(t *testing.T) {
	repo := &fakeRepo{topics: map[string]topic.Topic{}}
	seedRepo(repo)
	now := time.Now().UTC()
	repo.topics["fast-gpu"] = topic.Topic{Key: "fast-gpu", Name: "F", Enabled: true, CreatedAt: now, UpdatedAt: now}

	deleteFn := func(ctx context.Context, key string, ack bool) (topicadmin.DeleteSummary, error) {
		if key == "default" {
			return topicadmin.DeleteSummary{}, topicadmin.ErrDefaultProtected
		}
		if key == "fast-gpu" && !ack {
			return topicadmin.DeleteSummary{}, topicadmin.ErrNeedsAck
		}
		if key == "missing" {
			return topicadmin.DeleteSummary{}, topic.ErrTopicNotFound
		}
		return topicadmin.DeleteSummary{RemovedCaseRules: 1, RemovedEdgeSubs: 2, FailedTasks: 3}, nil
	}
	r := newTestRouter(repo, deleteFn)

	rec := do(t, r, http.MethodDelete, "/default", "")
	if rec.Code != http.StatusConflict {
		t.Fatalf("delete default status = %d", rec.Code)
	}
	rec = do(t, r, http.MethodDelete, "/fast-gpu", "")
	if rec.Code != http.StatusConflict {
		t.Fatalf("delete referenced status = %d", rec.Code)
	}
	rec = do(t, r, http.MethodDelete, "/fast-gpu", `{"ack_references":true}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete ack status = %d", rec.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["removed_case_rules"] != float64(1) || body["failed_tasks"] != float64(3) {
		t.Fatalf("body=%+v", body)
	}
	rec = do(t, r, http.MethodDelete, "/missing", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("delete missing status = %d", rec.Code)
	}
}
```

（现有 `do(t, r, method, path, body)` 已支持 body 参数；`handler_test.go` 补 `topicadmin` import。）

- [ ] **Step 2: 运行确认失败**

Run: `go test ./internal/httpapi/topics/ -run TestTopics_DeleteCleanup -v`
Expected: FAIL（`DeleteWithCleanup` 字段不存在）。

- [ ] **Step 3: 实现 handler.go**

- 删除 `CountCaseRefs` / `CountEdgeRefs` 字段与 `strings` import（若不再使用）。
- 加字段与 import：

```go
	// DeleteWithCleanup performs the cleanup delete (see internal/topicadmin).
	DeleteWithCleanup func(ctx context.Context, key string, ack bool) (topicadmin.DeleteSummary, error)
```

```go
	"github.com/mr9esx/comfyui_tgbot/internal/topicadmin"
```

- `delete` 替换为：

```go
func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	var body struct {
		AckReferences bool `json:"ack_references"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if h.DeleteWithCleanup == nil {
		writeErr(w, http.StatusInternalServerError, "delete cleanup not configured")
		return
	}
	summary, err := h.DeleteWithCleanup(r.Context(), key, body.AckReferences)
	if errors.Is(err, topicadmin.ErrDefaultProtected) {
		writeErrCode(w, http.StatusConflict, "topic_default_protected", "default topic cannot be deleted")
		return
	}
	if errors.Is(err, topicadmin.ErrNeedsAck) {
		writeErrCode(w, http.StatusConflict, "topic_delete_needs_ack",
			"topic is referenced by cases or edges; confirm with ack_references to remove references")
		return
	}
	if errors.Is(err, topic.ErrTopicNotFound) {
		writeErr(w, http.StatusNotFound, "topic not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"deleted":                 true,
		"removed_case_rules":      summary.RemovedCaseRules,
		"removed_edge_subscriptions": summary.RemovedEdgeSubs,
		"failed_tasks":            summary.FailedTasks,
	})
}
```

- `writeErr` 旁加 `writeErrCode`（同其他 handler）。

`main.go`：`topicDeleteSvc := topicadmin.NewService(gdb, botRT.Notify)`（放 `edgeDeleteSvc` 旁）；`Topics` handler 改为 `DeleteWithCleanup: topicDeleteSvc.DeleteTopic`，删除 `CountCaseRefs`/`CountEdgeRefs` 闭包；加 import `"github.com/mr9esx/comfyui_tgbot/internal/topicadmin"`。

- [ ] **Step 4: 重跑测试**

Run: `go test ./internal/httpapi/topics/... ./internal/topicadmin/...`
Expected: PASS。

Run: `go build ./...`
Expected: PASS。

- [ ] **Step 5: Commit（main.go 用 git add -p）**

```bash
git add internal/httpapi/topics/handler.go internal/httpapi/topics/handler_test.go
git add -p apps/pixoma/cmd/pixoma/main.go
git commit -m "feat(api): cleanup-based topic delete with ack_references"
```

---

### Task 4: 前端 API（deleteTopic ack、listTasks dispatch_topic）

**Files:**
- Modify: `web/admin/src/lib/api/topics.ts`
- Modify: `web/admin/src/lib/api/tasks.ts`
- Modify: `web/admin/src/lib/api/tasks.test.ts`

**Interfaces:**
- Produces: `deleteTopic(key, ack?)`、`listTasks({dispatch_topic})` —— Task 6 依赖

- [ ] **Step 1: 失败测试**

`tasks.test.ts` 追加：

```ts
  it('listTasks filters by dispatch_topic', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify([{ id: 't1', status: 'queued' }]), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
    vi.stubGlobal('fetch', fetchMock)

    const data = await listTasks({ dispatch_topic: 'fast-gpu', status: 'queued' })

    expect(data[0].id).toBe('t1')
    expect(fetchMock).toHaveBeenCalledWith(
      'http://127.0.0.1:8081/api/v1/tasks?dispatch_topic=fast-gpu&status=queued',
      expect.anything(),
    )
  })
```

- [ ] **Step 2: 运行确认失败**

Run: `pnpm vitest run src/lib/api/tasks.test.ts`
Expected: FAIL（URL 不匹配）。

- [ ] **Step 3: 实现**

`topics.ts`：

```ts
export type DeleteTopicResult = {
  deleted: boolean
  removed_case_rules?: number
  removed_edge_subscriptions?: number
  failed_tasks?: number
}

export function deleteTopic(key: string, ack?: boolean) {
  return apiFetch<DeleteTopicResult>(`/api/v1/topics/${encodeURIComponent(key)}`, {
    method: 'DELETE',
    ...(ack ? { body: JSON.stringify({ ack_references: true }) } : {}),
  })
}
```

`tasks.ts` 的 `listTasks` params 加 `dispatch_topic?: string`。

- [ ] **Step 4: 重跑测试**

Run: `pnpm vitest run src/lib/api/tasks.test.ts`
Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add web/admin/src/lib/api/topics.ts web/admin/src/lib/api/tasks.ts web/admin/src/lib/api/tasks.test.ts
git commit -m "feat(admin): delete topic with ack; filter tasks by dispatch_topic"
```

---

### Task 5: i18n 文案 + locale 成对

**Files:**
- Modify: `web/admin/src/lib/i18n/locales/zh.json`
- Modify: `web/admin/src/lib/i18n/locales/en.json`
- Modify: `web/admin/src/lib/i18n/locale.test.ts`

**Interfaces:**
- Produces: Topic 删除 keys —— Task 6 依赖

- [ ] **Step 1: locale.test.ts（先失败）**

新增成对列表与用例：

```ts
const TOPIC_DELETE_I18N_KEYS = [
  'topics.deleteWillRemoveRules',
  'topics.deleteWillUnbindNodes',
  'topics.deleteWillFailQueued',
  'topics.deleteAckImpact',
  'topics.deleteNeedsAck',
  'topics.deleteDefaultProtected',
  'topics.deleteDone',
  'topics.deletedFailed',
] as const

describe('topic delete i18n', () => {
  it('defines topic-delete keys in zh and en locales', () => {
    for (const key of TOPIC_DELETE_I18N_KEYS) {
      expect(lookup(zh, key), `zh missing ${key}`).toEqual(expect.any(String))
      expect(lookup(en, key), `en missing ${key}`).toEqual(expect.any(String))
    }
  })
})
```

- [ ] **Step 2: 运行确认失败**

Run: `pnpm vitest run src/lib/i18n/locale.test.ts`
Expected: FAIL。

- [ ] **Step 3: i18n keys**

`zh.json` 的 `topics` 段新增：

```json
    "deleteWillRemoveRules": "{{count}} 个工作流路由规则将被删除",
    "deleteWillUnbindNodes": "{{count}} 个计算节点的订阅将被移除",
    "deleteWillFailQueued": "{{count}} 个排队任务将标记为失败并通知用户",
    "deleteAckImpact": "我已知悉上述影响",
    "deleteNeedsAck": "该任务队列仍被工作流或计算节点引用，确认后将自动清理",
    "deleteDefaultProtected": "默认任务队列不允许删除",
    "deleteDone": "任务队列已删除（移除 {{rules}} 条规则、解除 {{nodes}} 个订阅、失败 {{tasks}} 个任务）",
    "deletedFailed": "删除失败"
```

`en.json` 对应：

```json
    "deleteWillRemoveRules": "{{count}} workflow routing rule(s) will be removed",
    "deleteWillUnbindNodes": "{{count}} compute node subscription(s) will be removed",
    "deleteWillFailQueued": "{{count}} queued task(s) will be marked failed and the users notified",
    "deleteAckImpact": "I understand the impact above",
    "deleteNeedsAck": "This task queue is still referenced by workflows or compute nodes; confirm to clean up automatically",
    "deleteDefaultProtected": "The default task queue cannot be deleted",
    "deleteDone": "Task queue deleted ({{rules}} rules removed, {{nodes}} subscriptions removed, {{tasks}} tasks failed)",
    "deletedFailed": "Delete failed"
```

（若该文件有用户未提交 hunk，用 `git add -p` 只暂存本任务 hunk。）

- [ ] **Step 4: 重跑测试**

Run: `pnpm vitest run src/lib/i18n/locale.test.ts`
Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add web/admin/src/lib/i18n/locales/zh.json web/admin/src/lib/i18n/locales/en.json web/admin/src/lib/i18n/locale.test.ts
git commit -m "feat(admin): topic delete impact i18n"
```

---

### Task 6: 前端 Topic 删除弹窗（toast 化）

**Files:**
- Modify: `web/admin/src/lib/api/localized-errors.ts`
- Modify: `web/admin/src/lib/api/localized-errors.test.ts`
- Modify: `web/admin/src/features/topics/topic-detail-panel.tsx`
- Modify: `web/admin/src/features/topics/topic-layout.contract.test.ts`

**Interfaces:**
- Consumes: `deleteTopic(key, ack)`、`listTasks({dispatch_topic})`（Task 4）、i18n keys（Task 5）
- Produces: toast 化删除流程

- [ ] **Step 1: localized-errors 补 topic 映射（先失败）**

`localized-errors.ts` 追加：

```ts
const TOPIC_DELETE_ERROR_CODES: Record<string, string> = {
  topic_delete_needs_ack: 'topics.deleteNeedsAck',
  topic_default_protected: 'topics.deleteDefaultProtected',
}

export function topicDeleteErrorMessage(
  err: unknown,
  t: (key: string) => string
): string | undefined {
  if (!(err instanceof ApiError)) return undefined
  const key = err.code ? TOPIC_DELETE_ERROR_CODES[err.code] : undefined
  return key ? t(key) : undefined
}
```

`localized-errors.test.ts` 追加：

```ts
  it('maps topic_delete_needs_ack to i18n key', () => {
    const err = new ApiError(409, 'referenced', 'topic_delete_needs_ack')
    expect(topicDeleteErrorMessage(err, t)).toBe('[topics.deleteNeedsAck]')
  })
```

- [ ] **Step 2: 运行确认失败**

Run: `pnpm vitest run src/lib/api/localized-errors.test.ts`
Expected: FAIL。

- [ ] **Step 3: 实现 topic-detail-panel.tsx**

- import 补 `listTasks`、`topicDeleteErrorMessage`；`deleteTopic` 已 import。
- 加 state 与查询：

```tsx
  const [ackImpact, setAckImpact] = useState(false)
  const queuedTasksQuery = useQuery({
    queryKey: ['topics', topicKey, 'queued-tasks'] as const,
    queryFn: () => listTasks({ dispatch_topic: topicKey, status: 'queued' }),
  })
  const caseRefCount = topicRefs.cases.length
  const edgeRefCount = topicRefs.edges.length
  const queuedCount = queuedTasksQuery.data?.length ?? 0
  const hasImpact = caseRefCount > 0 || edgeRefCount > 0 || queuedCount > 0
```

- `deleteMutation` 改为：

```tsx
  const deleteMutation = useMutation({
    mutationFn: () => deleteTopic(topicKey, ackImpact),
    onSuccess: async (summary) => {
      invalidate()
      toast.success(
        summary?.removed_case_rules !== undefined
          ? t('topics.deleteDone', {
              rules: summary.removed_case_rules ?? 0,
              nodes: summary.removed_edge_subscriptions ?? 0,
              tasks: summary.failed_tasks ?? 0,
            })
          : t('topics.deleted')
      )
    },
    onError: (err) => {
      toast.error(
        topicDeleteErrorMessage(err, t) ?? errorMessage(err) ?? t('topics.deletedFailed')
      )
    },
  })
```

- 弹窗 `AlertDialogDescription` 内追加影响区与勾选：

```tsx
                  <AlertDialogDescription>
                    {t('topics.deleteConfirmBody', { name: topic.name })}
                    {caseRefCount > 0 ? (
                      <p className='mt-3'>
                        {t('topics.deleteWillRemoveRules', { count: caseRefCount })}
                      </p>
                    ) : null}
                    {edgeRefCount > 0 ? (
                      <p className='mt-1'>
                        {t('topics.deleteWillUnbindNodes', { count: edgeRefCount })}
                      </p>
                    ) : null}
                    {queuedCount > 0 ? (
                      <p className='mt-1'>
                        {t('topics.deleteWillFailQueued', { count: queuedCount })}
                      </p>
                    ) : null}
                    {hasImpact ? (
                      <label className='mt-4 flex items-start gap-2'>
                        <input
                          type='checkbox'
                          checked={ackImpact}
                          onChange={(e) => setAckImpact(e.target.checked)}
                          data-testid='topic-delete-ack'
                        />
                        <span>{t('topics.deleteAckImpact')}</span>
                      </label>
                    ) : null}
                  </AlertDialogDescription>
```

- `AlertDialogAction` 的 `disabled` 改为 `hasImpact ? !ackImpact : false`。
- 删除内联 `{deleteMutation.isError ? <ErrorBanner .../> : null}`（保留 update/enable 的 ErrorBanner）。

`topic-layout.contract.test.ts` 的删除用例更新断言：

```ts
    expect(DETAIL_PANEL).toContain('topics.deleteWillRemoveRules')
    expect(DETAIL_PANEL).toContain('topics.deleteWillUnbindNodes')
    expect(DETAIL_PANEL).toContain('topics.deleteWillFailQueued')
    expect(DETAIL_PANEL).toContain('topicDeleteErrorMessage')
    expect(DETAIL_PANEL).toContain('deleteTopic(topicKey, ackImpact)')
```

- [ ] **Step 4: 重跑测试**

Run: `pnpm vitest run src/lib/api/localized-errors.test.ts src/features/topics/topic-layout.contract.test.ts`
Expected: PASS。

Run: `pnpm exec tsc -b`
Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add web/admin/src/lib/api/localized-errors.ts web/admin/src/lib/api/localized-errors.test.ts web/admin/src/features/topics/topic-detail-panel.tsx web/admin/src/features/topics/topic-layout.contract.test.ts web/admin/src/lib/i18n/locales/zh.json web/admin/src/lib/i18n/locales/en.json
git commit -m "feat(admin): topic delete dialog with impact and toast errors"
```

---

### Task 7: 全量验证

**Files:** 无代码改动

- [ ] **Step 1: 后端全量测试 + vet**

Run: `go test ./... && go vet ./...`
Expected: PASS。

- [ ] **Step 2: 前端全量测试 + 类型 + lint**

Run: `pnpm vitest run`
Expected: PASS。

Run: `pnpm exec tsc -b`
Expected: PASS。

Run: `pnpm exec eslint src/features/topics src/lib/api`
Expected: PASS。

- [ ] **Step 3: 手工冒烟（可选）**

删除一个被工作流规则引用 + 节点订阅 + 有 queued 任务的 Topic：弹窗展示三行影响，勾选后删除成功；case 规则消失、节点订阅移除、queued 任务变 failed（`topic_deleted`），用户收到失败通知；错误场景走 toast 中文提示。

- [ ] **Step 4: 收尾**

`git status --porcelain` 检查无遗留；`main.go` / i18n 中用户并发改动保持未提交。
