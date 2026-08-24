# Case 清理式删除 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把 Case 删除从「四重阻止 + 英文报错」改为「确认制清理式删除」：同一事务内失败 pending 任务、终止活跃会话、自动解除菜单/卡片引用并删除 case，running/queued 任务自然跑完，会话用户收到终止提示。

**Architecture:** 新增 `internal/caseadmin` 应用服务，持有 `*gorm.DB` 和 `notify.Publisher`，在单个 `gorm.Transaction` 内用 tx 作用域构造四个仓库完成清理；`cases` HTTP handler 只保留一个注入的 `DeleteWithCleanup` 函数并转发 `ack_references` 确认位。前端删除弹窗展示引用清单与影响摘要，勾选「我已知悉」后携带 ack 调用 DELETE，错误按后端 `code` 映射为本地化文案。

**Tech Stack:** Go 1.x + GORM（SQLite 测试）、chi、React 18 + TanStack Query + react-i18next + sonner、Vitest（node 合同测试）、pnpm。

## Global Constraints

- 不新增第三方依赖。
- 后端 API 错误 `error` 字段保持英文作为兜底；用户可见文案必须走前端 i18n 或 TG adapter 内固定中文（与现有 `"卡片不存在或已删除"` 等一致）。
- 新增 i18n key 必须 zh/en 成对，并在 `web/admin/src/lib/i18n/locale.test.ts` 的成对列表中登记。
- 删除不得触碰 running/queued 任务；不得物理删除任务行。
- pending 任务失败必须绕过 `RequeueAfterFailure`（直接 `MarkFailed`，不发布 `TaskFailed` 状态事件）。
- 前端 node 测试文件必须已存在于 `vitest.config.ts` 的 include 列表（本计划不新增前端测试文件，只修改现有文件）。
- 每个任务以独立可测试交付物结束并单独 commit。

---

### Task 1: 会话仓库支持按 case 查询活跃会话

**Files:**
- Modify: `internal/conversation/domain/service.go`（Repository 接口 + MemoryRepository）
- Modify: `internal/conversation/infrastructure/persistence/gorm_session.go`
- Test: `internal/conversation/infrastructure/persistence/gorm_session_test.go`

**Interfaces:**
- Consumes: `domain.Status.IsActive()`、`domain.Session`、`sharedkernel.CaseID`（均已存在）
- Produces: `conversation.Repository.ListActiveByCase(ctx, caseID) ([]*Session, error)` —— 后续 Task 3 依赖

- [ ] **Step 1: 接口与两个实现加方法，先写 GORM 单测**

`internal/conversation/domain/service.go` 的 `Repository` 接口内追加：

```go
	// ListActiveByCase returns collecting/confirming sessions for a case.
	ListActiveByCase(ctx context.Context, caseID sharedkernel.CaseID) ([]*Session, error)
```

同文件 `MemoryRepository` 追加：

```go
func (r *MemoryRepository) ListActiveByCase(_ context.Context, caseID sharedkernel.CaseID) ([]*Session, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*Session
	for _, s := range r.byID {
		if s.CaseID != caseID || !s.Status.IsActive() {
			continue
		}
		out = append(out, cloneSession(s))
	}
	return out, nil
}
```

`internal/conversation/infrastructure/persistence/gorm_session.go` 追加：

```go
func (r *SessionRepository) ListActiveByCase(ctx context.Context, caseID sharedkernel.CaseID) ([]*domain.Session, error) {
	var rows []SessionRow
	if err := r.db.WithContext(ctx).
		Where("case_id = ? AND status IN ?", uint64(caseID), []string{
			string(domain.StatusCollecting),
			string(domain.StatusConfirming),
		}).
		Order("updated_at desc").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*domain.Session, 0, len(rows))
	for _, row := range rows {
		s, err := fromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}
```

在 `gorm_session_test.go` 追加单测（复用文件顶部 `db.Open` + `AutoMigrate` 模式）：

```go
func TestSessionListActiveByCase(t *testing.T) {
	gdb, err := db.Open(db.Options{DSN: "file:sess_active_" + t.Name() + "?mode=memory&cache=shared"})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(gdb, &persistence.SessionRow{}); err != nil {
		t.Fatal(err)
	}
	repo := persistence.NewSessionRepository(gdb)
	ctx := context.Background()
	now := time.Now().UTC()
	if err := repo.Save(ctx, domain.NewCollecting("s1", "tg:1", 10, []string{"a"}, now)); err != nil {
		t.Fatal(err)
	}
	s2 := domain.NewCollecting("s2", "tg:2", 10, []string{"a"}, now)
	s2.Status = domain.StatusSubmitted
	if err := repo.Save(ctx, s2); err != nil {
		t.Fatal(err)
	}
	if err := repo.Save(ctx, domain.NewCollecting("s3", "tg:3", 20, []string{"a"}, now)); err != nil {
		t.Fatal(err)
	}
	got, err := repo.ListActiveByCase(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "s1" {
		t.Fatalf("ListActiveByCase(10)=%+v", got)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/conversation/infrastructure/persistence/ -run TestSessionListActiveByCase -v`
Expected: FAIL，`ListActiveByCase` 未定义（编译失败即视为失败）。

- [ ] **Step 3: 应用上面的接口与实现，重跑测试**

Run: `go test ./internal/conversation/...`
Expected: PASS。

- [ ] **Step 4: Commit**

```bash
git add internal/conversation/domain/service.go internal/conversation/infrastructure/persistence/gorm_session.go internal/conversation/infrastructure/persistence/gorm_session_test.go
git commit -m "feat(conversation): list active sessions by case"
```

---

### Task 2: 菜单/卡片仓库支持解除工作流引用

**Files:**
- Modify: `internal/menucard/infrastructure/persistence/card_repository.go`
- Modify: `internal/httpapi/menucards/handler_test.go`（memCardRepo fake）
- Test: `internal/menucard/infrastructure/persistence/card_repository_test.go`

**Interfaces:**
- Consumes: `WorkflowPlacement`、`mcdomain.Menu/Card/Action`、`fillChannelNames`（均已存在）
- Produces: `CardRepository.RemoveWorkflowReferences(ctx, workflowID) ([]WorkflowPlacement, error)` —— 后续 Task 3 依赖

- [ ] **Step 1: 接口加方法，先写失败测试**

`internal/menucard/infrastructure/persistence/card_repository.go` 的 `CardRepository` 接口内追加：

```go
	// RemoveWorkflowReferences removes workflowID from open_workflow actions in
	// all menus and cards; drops items/buttons whose workflow list becomes empty.
	// Returns the placements that were removed.
	RemoveWorkflowReferences(ctx context.Context, workflowID string) ([]WorkflowPlacement, error)
```

`card_repository_test.go` 追加：

```go
func TestRemoveWorkflowReferences(t *testing.T) {
	gdb, err := db.Open(db.Options{DSN: "file:unlink_" + t.Name() + "?mode=memory&cache=shared"})
	if err != nil {
		t.Fatal(err)
	}
	if err := gdb.AutoMigrate(&persistence.MainMenuRow{}, &persistence.CardRow{}); err != nil {
		t.Fatal(err)
	}
	repo := persistence.NewGormCardRepository(gdb)
	ctx := context.Background()

	menu := mcdomain.Menu{ID: "m", Name: "主", Columns: 2, Items: []mcdomain.MenuItem{
		{ID: "mi-keep", Label: "保留", Action: mcdomain.Action{Type: "open_workflow", WorkflowIDs: []string{"1", "2"}}},
		{ID: "mi-drop", Label: "删除", Action: mcdomain.Action{Type: "open_workflow", WorkflowIDs: []string{"1"}}},
		{ID: "mi-other", Label: "无关", Action: mcdomain.Action{Type: "placeholder"}},
	}}
	if err := repo.PutMenu(ctx, "ch1", menu); err != nil {
		t.Fatal(err)
	}
	card := mcdomain.Card{ID: "c1", Name: "卡", Text: "hi", Buttons: []mcdomain.CardButton{
		{ID: "b1", Label: "B", Action: mcdomain.Action{Type: "open_workflow", WorkflowIDs: []string{"1", "3"}}},
		{ID: "b2", Label: "B2", Action: mcdomain.Action{Type: "open_workflow", WorkflowIDs: []string{"2"}}},
	}}
	if err := repo.CreateCard(ctx, "ch1", card); err != nil {
		t.Fatal(err)
	}

	removed, err := repo.RemoveWorkflowReferences(ctx, "1")
	if err != nil {
		t.Fatal(err)
	}
	if len(removed) != 2 {
		t.Fatalf("removed=%+v", removed)
	}

	gotMenu, err := repo.GetMenu(ctx, "ch1")
	if err != nil {
		t.Fatal(err)
	}
	if len(gotMenu.Items) != 2 {
		t.Fatalf("menu items=%+v", gotMenu.Items)
	}
	if len(gotMenu.Items[0].Action.WorkflowIDs) != 1 || gotMenu.Items[0].Action.WorkflowIDs[0] != "2" {
		t.Fatalf("mi-keep workflow_ids=%v", gotMenu.Items[0].Action.WorkflowIDs)
	}
	if gotMenu.Items[1].ID != "mi-other" {
		t.Fatalf("second item=%+v", gotMenu.Items[1])
	}

	cards, err := repo.ListCards(ctx, "ch1")
	if err != nil {
		t.Fatal(err)
	}
	if len(cards) != 1 || len(cards[0].Buttons) != 1 || cards[0].Buttons[0].ID != "b2" {
		t.Fatalf("cards=%+v", cards)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/menucard/infrastructure/persistence/ -run TestRemoveWorkflowReferences -v`
Expected: FAIL（接口缺方法，编译失败）。

- [ ] **Step 3: GORM 实现 + helper**

`card_repository.go` 追加 helper 与实现（放在 `WorkflowPlacements` 之后）：

```go
func removeWorkflowID(ids []string, workflowID string) []string {
	var out []string
	for _, id := range ids {
		if id != workflowID {
			out = append(out, id)
		}
	}
	return out
}

func (r *GormCardRepository) RemoveWorkflowReferences(ctx context.Context, workflowID string) ([]WorkflowPlacement, error) {
	var removed []WorkflowPlacement
	var menus []MainMenuRow
	if err := r.db.WithContext(ctx).Find(&menus).Error; err != nil {
		return nil, err
	}
	for _, row := range menus {
		var menu mcdomain.Menu
		if err := json.Unmarshal([]byte(row.DocJSON), &menu); err != nil {
			return nil, fmt.Errorf("decode menu: %w", err)
		}
		items := menu.Items[:0]
		changed := false
		for _, it := range menu.Items {
			if it.Action.Type != "open_workflow" {
				items = append(items, it)
				continue
			}
			kept := removeWorkflowID(it.Action.WorkflowIDs, workflowID)
			if len(kept) == len(it.Action.WorkflowIDs) {
				items = append(items, it)
				continue
			}
			changed = true
			removed = append(removed, WorkflowPlacement{ChannelID: row.ChannelID, ItemID: it.ID, Label: it.Label, Kind: "menu_item"})
			if len(kept) > 0 {
				it.Action.WorkflowIDs = kept
				items = append(items, it)
			}
		}
		if !changed {
			continue
		}
		menu.Items = items
		raw, err := json.Marshal(menu)
		if err != nil {
			return nil, err
		}
		if err := r.db.WithContext(ctx).Model(&MainMenuRow{}).Where("channel_id = ?", row.ChannelID).Update("doc_json", string(raw)).Error; err != nil {
			return nil, err
		}
	}

	var cards []CardRow
	if err := r.db.WithContext(ctx).Find(&cards).Error; err != nil {
		return nil, err
	}
	for _, row := range cards {
		var card mcdomain.Card
		if err := json.Unmarshal([]byte(row.DocJSON), &card); err != nil {
			return nil, fmt.Errorf("decode card: %w", err)
		}
		buttons := card.Buttons[:0]
		changed := false
		for _, b := range card.Buttons {
			if b.Action.Type != "open_workflow" {
				buttons = append(buttons, b)
				continue
			}
			kept := removeWorkflowID(b.Action.WorkflowIDs, workflowID)
			if len(kept) == len(b.Action.WorkflowIDs) {
				buttons = append(buttons, b)
				continue
			}
			changed = true
			removed = append(removed, WorkflowPlacement{ChannelID: row.ChannelID, ItemID: b.ID, Label: b.Label, Kind: "card_button"})
			if len(kept) > 0 {
				b.Action.WorkflowIDs = kept
				buttons = append(buttons, b)
			}
		}
		if !changed {
			continue
		}
		card.Buttons = buttons
		raw, err := json.Marshal(card)
		if err != nil {
			return nil, err
		}
		if err := r.db.WithContext(ctx).Model(&CardRow{}).Where("id = ?", row.ID).Update("doc_json", string(raw)).Error; err != nil {
			return nil, err
		}
	}
	if len(removed) > 0 {
		if err := r.fillChannelNames(ctx, removed); err != nil {
			return nil, err
		}
	}
	return removed, nil
}
```

- [ ] **Step 4: memCardRepo fake 补方法**

`internal/httpapi/menucards/handler_test.go` 的 `memCardRepo` 追加（处理 `m.cards == nil`）：

```go
func (m *memCardRepo) RemoveWorkflowReferences(_ context.Context, workflowID string) ([]persistence.WorkflowPlacement, error) {
	var removed []persistence.WorkflowPlacement
	items := m.menu.Items[:0]
	for _, it := range m.menu.Items {
		if it.Action.Type != "open_workflow" {
			items = append(items, it)
			continue
		}
		var kept []string
		for _, id := range it.Action.WorkflowIDs {
			if id != workflowID {
				kept = append(kept, id)
			}
		}
		if len(kept) == len(it.Action.WorkflowIDs) {
			items = append(items, it)
			continue
		}
		removed = append(removed, persistence.WorkflowPlacement{ChannelID: "ch1", ItemID: it.ID, Label: it.Label, Kind: "menu_item"})
		if len(kept) > 0 {
			it.Action.WorkflowIDs = kept
			items = append(items, it)
		}
	}
	m.menu.Items = items
	for id, card := range m.cards {
		buttons := card.Buttons[:0]
		for _, b := range card.Buttons {
			if b.Action.Type != "open_workflow" {
				buttons = append(buttons, b)
				continue
			}
			var kept []string
			for _, wid := range b.Action.WorkflowIDs {
				if wid != workflowID {
					kept = append(kept, wid)
				}
			}
			if len(kept) == len(b.Action.WorkflowIDs) {
				buttons = append(buttons, b)
				continue
			}
			removed = append(removed, persistence.WorkflowPlacement{ChannelID: "ch1", ItemID: b.ID, Label: b.Label, Kind: "card_button"})
			if len(kept) > 0 {
				b.Action.WorkflowIDs = kept
				buttons = append(buttons, b)
			}
		}
		card.Buttons = buttons
		m.cards[id] = card
	}
	return removed, nil
}
```

- [ ] **Step 5: 重跑测试**

Run: `go test ./internal/menucard/... ./internal/httpapi/menucards/...`
Expected: PASS。

- [ ] **Step 6: Commit**

```bash
git add internal/menucard/infrastructure/persistence/card_repository.go internal/menucard/infrastructure/persistence/card_repository_test.go internal/httpapi/menucards/handler_test.go
git commit -m "feat(menucard): remove workflow references from menus and cards"
```

---

### Task 3: 删除清理服务（caseadmin）

**Files:**
- Create: `internal/caseadmin/case_delete.go`
- Create: `internal/caseadmin/case_delete_test.go`
- Modify: `internal/sharedkernel/ids.go`（错误码常量）

**Interfaces:**
- Consumes: `conversation.Repository.ListActiveByCase`（Task 1）、`CardRepository.RemoveWorkflowReferences`（Task 2）、`casepersist.GormRepository`、`taskpersist.TaskRepository`、`sesspersist.SessionRepository`、`mencardpersist.NewGormCardRepository`、`notify.Publisher`（`Publish(ctx, sharedkernel.UserNotify) error`）
- Produces: `caseadmin.Service`（`NewService(db *gorm.DB, n notify.Publisher) *Service`、`DeleteCase(ctx, id sharedkernel.CaseID, ack bool) (DeleteSummary, error)`）、`ErrNeedsAck`、`RemovedPlacement`、`DeleteSummary` —— Task 4 依赖

- [ ] **Step 1: 常量 + 失败测试**

`internal/sharedkernel/ids.go` 的任务状态常量旁追加：

```go
// Terminal-failure metadata used when a workflow is deleted.
const (
	TaskErrorCaseDeleted = "case_deleted"
	CaseDeletedMessage   = "工作流已删除"
)
```

`internal/caseadmin/case_delete_test.go`：

```go
package caseadmin_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/mr9esx/comfyui_tgbot/internal/caseadmin"
	catalogdomain "github.com/mr9esx/comfyui_tgbot/internal/catalog/domain"
	casepersist "github.com/mr9esx/comfyui_tgbot/internal/catalog/infrastructure/persistence"
	convdomain "github.com/mr9esx/comfyui_tgbot/internal/conversation/domain"
	sesspersist "github.com/mr9esx/comfyui_tgbot/internal/conversation/infrastructure/persistence"
	mcdomain "github.com/mr9esx/comfyui_tgbot/internal/menucard/domain"
	mencardpersist "github.com/mr9esx/comfyui_tgbot/internal/menucard/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/db"
	runtimedomain "github.com/mr9esx/comfyui_tgbot/internal/runtime/domain"
	taskpersist "github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

type captureNotify struct {
	items []sharedkernel.UserNotify
}

func (c *captureNotify) Publish(_ context.Context, n sharedkernel.UserNotify) error {
	c.items = append(c.items, n)
	return nil
}

func newDeleteTest(t *testing.T) (*gorm.DB, *caseadmin.Service, *captureNotify, context.Context, time.Time) {
	t.Helper()
	gdb, err := db.Open(db.Options{DSN: "file:case_delete_" + t.Name() + "?mode=memory&cache=shared"})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(gdb,
		&casepersist.CaseRow{},
		&taskpersist.TaskRow{},
		&sesspersist.SessionRow{},
		&mencardpersist.MainMenuRow{},
		&mencardpersist.CardRow{}); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	now := time.Now().UTC()
	n := &captureNotify{}
	return gdb, caseadmin.NewService(gdb, n), n, ctx, now
}
```

测试主体：

```go
func TestDeleteCase_AckRequired(t *testing.T) {
	gdb, svc, _, ctx, now := newDeleteTest(t)
	caseRepo := casepersist.NewGormRepository(gdb)
	taskRepo := taskpersist.NewTaskRepository(gdb)
	sessRepo := sesspersist.NewSessionRepository(gdb)
	menuRepo := mencardpersist.NewGormCardRepository(gdb)

	_ = caseRepo.Save(ctx, &catalogdomain.Case{Document: catalogdomain.CaseDocument{ID: 10, Name: "c"}, Enabled: true})
	_ = taskRepo.Create(ctx, runtimedomain.NewPending("t1", "s1", 10, "inputs/t1", now))
	_ = sessRepo.Save(ctx, convdomain.NewCollecting("s1", "tg:9", 10, []string{"a"}, now))
	_ = menuRepo.PutMenu(ctx, "ch1", mcdomain.Menu{ID: "m", Name: "主", Columns: 2, Items: []mcdomain.MenuItem{
		{ID: "mi", Label: "L", Action: mcdomain.Action{Type: "open_workflow", WorkflowIDs: []string{"10"}}},
	}})

	_, err := svc.DeleteCase(ctx, 10, false)
	if !errors.Is(err, caseadmin.ErrNeedsAck) {
		t.Fatalf("err=%v want ErrNeedsAck", err)
	}
	if _, err := caseRepo.Get(ctx, 10); err != nil {
		t.Fatalf("case must survive rollback: %v", err)
	}
	task, _ := taskRepo.Get(ctx, "t1")
	if task.Status != sharedkernel.TaskPending {
		t.Fatalf("task must survive rollback: %+v", task)
	}
	sess, err := sessRepo.ListActiveByCase(ctx, 10)
	if err != nil || len(sess) != 1 {
		t.Fatalf("session must survive rollback: %v %+v", err, sess)
	}
}

func TestDeleteCase_CleanupAndDelete(t *testing.T) {
	gdb, svc, n, ctx, now := newDeleteTest(t)
	caseRepo := casepersist.NewGormRepository(gdb)
	taskRepo := taskpersist.NewTaskRepository(gdb)
	sessRepo := sesspersist.NewSessionRepository(gdb)
	menuRepo := mencardpersist.NewGormCardRepository(gdb)

	_ = caseRepo.Save(ctx, &catalogdomain.Case{Document: catalogdomain.CaseDocument{ID: 10, Name: "c"}, Enabled: true})
	pending := runtimedomain.NewPending("t1", "s1", 10, "inputs/t1", now)
	pending.ChatID = "tg:9"
	_ = taskRepo.Create(ctx, pending)
	running := runtimedomain.NewPending("t2", "s2", 10, "inputs/t2", now)
	_ = running.MarkQueued("local", now)
	_ = running.MarkRunning("p", now)
	_ = taskRepo.Create(ctx, running)
	_ = sessRepo.Save(ctx, convdomain.NewCollecting("s1", "tg:9", 10, []string{"a"}, now))
	_ = menuRepo.PutMenu(ctx, "ch1", mcdomain.Menu{ID: "m", Name: "主", Columns: 2, Items: []mcdomain.MenuItem{
		{ID: "mi", Label: "L", Action: mcdomain.Action{Type: "open_workflow", WorkflowIDs: []string{"10"}}},
	}})

	summary, err := svc.DeleteCase(ctx, 10, true)
	if err != nil {
		t.Fatal(err)
	}
	if summary.FailedTasks != 1 || summary.TerminatedSessions != 1 || len(summary.RemovedPlacements) != 1 {
		t.Fatalf("summary=%+v", summary)
	}
	if _, err := caseRepo.Get(ctx, 10); !errors.Is(err, catalogdomain.ErrNotFound) {
		t.Fatalf("case delete err=%v", err)
	}
	t1, _ := taskRepo.Get(ctx, "t1")
	if t1.Status != sharedkernel.TaskFailed || t1.ErrorCode != sharedkernel.TaskErrorCaseDeleted {
		t.Fatalf("pending task=%+v", t1)
	}
	t2, _ := taskRepo.Get(ctx, "t2")
	if t2.Status != sharedkernel.TaskRunning {
		t.Fatalf("running task must be untouched: %+v", t2)
	}
	sess, _ := sessRepo.ListActiveByCase(ctx, 10)
	if len(sess) != 0 {
		t.Fatalf("sessions not terminated: %+v", sess)
	}
	menu, _ := menuRepo.GetMenu(ctx, "ch1")
	if len(menu.Items) != 0 {
		t.Fatalf("menu items=%+v", menu.Items)
	}
	var kinds []string
	for _, item := range n.items {
		kinds = append(kinds, item.Kind)
	}
	if len(n.items) != 2 || !containsKind(kinds, "task_failed") || !containsKind(kinds, "session_terminated") {
		t.Fatalf("notifies=%+v", n.items)
	}
}

func containsKind(kinds []string, want string) bool {
	for _, k := range kinds {
		if k == want {
			return true
		}
	}
	return false
}
```

测试文件 import 需包含 `"gorm.io/gorm"` 以及
`casepersist`/`taskpersist`/`sesspersist`/`mencardpersist`/`catalogdomain`/`convdomain`/`mcdomain`/`runtimedomain`/`sharedkernel`/`db`/`errors`/`time`/`testing`/`context`。

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/caseadmin/ -v`
Expected: FAIL（`caseadmin` 包不存在 / 方法未定义）。

- [ ] **Step 3: 实现 case_delete.go**

`internal/caseadmin/case_delete.go`：

```go
// Package caseadmin implements admin case lifecycle operations that span
// multiple repositories in a single transaction.
package caseadmin

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm"

	catalogdomain "github.com/mr9esx/comfyui_tgbot/internal/catalog/domain"
	casepersist "github.com/mr9esx/comfyui_tgbot/internal/catalog/infrastructure/persistence"
	convdomain "github.com/mr9esx/comfyui_tgbot/internal/conversation/domain"
	sesspersist "github.com/mr9esx/comfyui_tgbot/internal/conversation/infrastructure/persistence"
	mcpersist "github.com/mr9esx/comfyui_tgbot/internal/menucard/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/notify"
	runtimedomain "github.com/mr9esx/comfyui_tgbot/internal/runtime/domain"
	taskpersist "github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

// ErrNeedsAck is returned when the case is referenced by menu/card entries
// and the caller did not confirm removal with ack_references.
var ErrNeedsAck = errors.New("case delete needs ack for references")

const sessionTerminatedMessage = "该工作流已被管理员删除，当前会话已结束。"

// RemovedPlacement is a menu/card entry removed because it referenced the case.
type RemovedPlacement struct {
	ChannelID   string `json:"channel_id"`
	ChannelName string `json:"channel_name,omitempty"`
	ItemID      string `json:"item_id"`
	Label       string `json:"label"`
	Kind        string `json:"kind"`
}

// DeleteSummary reports what the cleanup did.
type DeleteSummary struct {
	RemovedPlacements  []RemovedPlacement `json:"removed_placements"`
	FailedTasks        int                `json:"failed_tasks"`
	TerminatedSessions int                `json:"terminated_sessions"`
}

// Service deletes a case with cleanup in one DB transaction.
type Service struct {
	db     *gorm.DB
	notify notify.Publisher
	now    func() time.Time
}

// NewService constructs a delete service over the shared database handle.
func NewService(db *gorm.DB, n notify.Publisher) *Service {
	return &Service{db: db, notify: n, now: func() time.Time { return time.Now().UTC() }}
}

// DeleteCase fails pending tasks, terminates active sessions, unlinks
// menu/card references and deletes the case row atomically. Notifications are
// sent best-effort after commit.
func (s *Service) DeleteCase(ctx context.Context, id sharedkernel.CaseID, ack bool) (DeleteSummary, error) {
	var summary DeleteSummary
	var taskNotifies []sharedkernel.UserNotify
	var terminatedChats []sharedkernel.ChatID

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		caseRepo := casepersist.NewGormRepository(tx)
		taskRepo := taskpersist.NewTaskRepository(tx)
		sessRepo := sesspersist.NewSessionRepository(tx)
		menuRepo := mcpersist.NewGormCardRepository(tx)

		if _, err := caseRepo.Get(ctx, id); err != nil {
			return err
		}
		idStr := fmt.Sprintf("%d", uint64(id))
		existing, err := menuRepo.WorkflowPlacements(ctx, idStr)
		if err != nil {
			return err
		}
		if len(existing) > 0 && !ack {
			return ErrNeedsAck
		}

		now := s.now()
		pending, err := taskRepo.List(ctx, runtimedomain.AdminListQuery{CaseID: id, Status: sharedkernel.TaskPending})
		if err != nil {
			return err
		}
		for _, t := range pending {
			if err := t.MarkFailed(sharedkernel.TaskErrorCaseDeleted, sharedkernel.CaseDeletedMessage, now); err != nil {
				return err
			}
			if err := taskRepo.Update(ctx, t); err != nil {
				return err
			}
			summary.FailedTasks++
			taskNotifies = append(taskNotifies, sharedkernel.UserNotify{
				ChatID:   t.ChatID,
				TaskID:   t.ID,
				Kind:     "task_failed",
				ErrorMsg: t.ErrorMessage,
			})
		}

		sessions, err := sessRepo.ListActiveByCase(ctx, id)
		if err != nil {
			return err
		}
		for _, sess := range sessions {
			if err := sess.Exit(now); err != nil {
				return err
			}
			if err := sessRepo.Save(ctx, sess); err != nil {
				return err
			}
			summary.TerminatedSessions++
			terminatedChats = append(terminatedChats, sess.ChatID)
		}

		placements, err := menuRepo.RemoveWorkflowReferences(ctx, idStr)
		if err != nil {
			return err
		}
		for _, p := range placements {
			summary.RemovedPlacements = append(summary.RemovedPlacements, RemovedPlacement{
				ChannelID: p.ChannelID, ChannelName: p.ChannelName,
				ItemID: p.ItemID, Label: p.Label, Kind: p.Kind,
			})
		}
		return caseRepo.Delete(ctx, id)
	})
	if err != nil {
		return DeleteSummary{}, err
	}

	for _, n := range taskNotifies {
		if n.ChatID == "" {
			continue
		}
		if err := s.notify.Publish(ctx, n); err != nil {
			slog.Warn("case delete: task notify failed", "task", n.TaskID, "err", err)
		}
	}
	for _, chatID := range terminatedChats {
		n := sharedkernel.UserNotify{ChatID: chatID, Kind: "session_terminated", ErrorMsg: sessionTerminatedMessage}
		if err := s.notify.Publish(ctx, n); err != nil {
			slog.Warn("case delete: session notify failed", "chat", chatID, "err", err)
		}
	}
	return summary, nil
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/caseadmin/ -v`
Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add internal/sharedkernel/ids.go internal/caseadmin/case_delete.go internal/caseadmin/case_delete_test.go
git commit -m "feat(caseadmin): atomic case delete with cleanup and notifications"
```

---

### Task 4: cases HTTP handler 改为清理式删除 + main.go 接线

**Files:**
- Modify: `internal/httpapi/cases/handler.go`
- Modify: `internal/httpapi/cases/handler_test.go`（重写删除相关测试）
- Modify: `apps/pixoma/cmd/pixoma/main.go`

**Interfaces:**
- Consumes: `caseadmin.Service.DeleteCase`、`caseadmin.ErrNeedsAck`、`caseadmin.DeleteSummary`（Task 3）、`botRT.Notify`（main.go 已有）
- Produces: `DELETE /api/v1/cases/{id}` 新契约；handler 不再需要 Count* 注入字段

- [ ] **Step 1: 先改 handler_test 为失败测试**

`handler_test.go` 删除 `TestCasesHandler_DeleteProtections`，替换为：

```go
func TestCasesHandler_DeleteCleanup(t *testing.T) {
	ctx := context.Background()
	h := &Handler{
		Repo: &memRepo{cases: map[uint64]*domain.Case{}},
		DeleteWithCleanup: func(ctx context.Context, id sharedkernel.CaseID, ack bool) (caseadmin.DeleteSummary, error) {
			if id == 10 && !ack {
				return caseadmin.DeleteSummary{}, caseadmin.ErrNeedsAck
			}
			if id == 999999 {
				return caseadmin.DeleteSummary{}, domain.ErrNotFound
			}
			return caseadmin.DeleteSummary{FailedTasks: 1, TerminatedSessions: 2}, nil
		},
	}
	r := chi.NewRouter()
	r.Route("/api/v1/cases", func(r chi.Router) { h.Mount(r) })
	srv := httptest.NewServer(r)
	defer srv.Close()

	do := func(id string, body string) *http.Response {
		var rdr io.Reader
		if body != "" {
			rdr = strings.NewReader(body)
		}
		req, err := http.NewRequest(http.MethodDelete, srv.URL+"/api/v1/cases/"+id, rdr)
		if err != nil {
			t.Fatal(err)
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		return res
	}

	res := do("999999", "")
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("missing status=%d want 404", res.StatusCode)
	}
	res.Body.Close()

	res = do("10", "")
	if res.StatusCode != http.StatusConflict || decodeErr(t, res)["code"] != "case_delete_needs_ack" {
		t.Fatalf("no-ack status=%d", res.StatusCode)
	}
	res.Body.Close()

	res = do("10", `{"ack_references":true}`)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("ack status=%d", res.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if body["deleted"] != true || body["failed_tasks"] != float64(1) || body["terminated_sessions"] != float64(2) {
		t.Fatalf("body=%+v", body)
	}
}
```

（handler_test 顶部补 `caseadmin`、`io`、`strings` import；`decodeErr` 改为返回 `map[string]string`，见 Step 3 说明。）

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/httpapi/cases/ -run TestCasesHandler_DeleteCleanup -v`
Expected: FAIL（`DeleteWithCleanup` 字段不存在）。

- [ ] **Step 3: 改 handler**

`internal/httpapi/cases/handler.go`：

- 删除 `CountMenuRefs`、`CountActiveSessions`、`CountActiveTasks` 三个字段与 `strconv` import；`context` 仍保留（`DeleteWithCleanup` 签名用）。
- 加字段与 import：

```go
	// DeleteWithCleanup performs the cleanup delete (see internal/caseadmin).
	DeleteWithCleanup func(ctx context.Context, id sharedkernel.CaseID, ack bool) (caseadmin.DeleteSummary, error)
```

```go
	"github.com/mr9esx/comfyui_tgbot/internal/caseadmin"
```

- 替换 `delete` 方法为：

```go
func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseCaseID(chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid case id")
		return
	}
	var body struct {
		AckReferences bool `json:"ack_references"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if h.DeleteWithCleanup == nil {
		writeErr(w, http.StatusInternalServerError, "delete cleanup not configured")
		return
	}
	summary, err := h.DeleteWithCleanup(r.Context(), id, body.AckReferences)
	if errors.Is(err, caseadmin.ErrNeedsAck) {
		writeErrCode(w, http.StatusConflict, "case_delete_needs_ack",
			"case is referenced by menu or card entries; confirm with ack_references to remove references")
		return
	}
	if errors.Is(err, domain.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "case not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"deleted":             true,
		"removed_placements":  summary.RemovedPlacements,
		"failed_tasks":        summary.FailedTasks,
		"terminated_sessions": summary.TerminatedSessions,
	})
}
```

- 新增 `writeErrCode`（放在 `writeErr` 旁）：

```go
func writeErrCode(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, map[string]string{"error": msg, "code": code})
}
```

- 若 handler_test 里现有 `decodeErr` 只读 `error` 字段，改为：

```go
func decodeErr(t *testing.T, res *http.Response) map[string]string {
	t.Helper()
	var body map[string]string
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	return body
}
```

并同步修正测试里所有 `decodeErr(t, res) != "..."` 的比较为 `decodeErr(t, res)["error"] != "..."`。

- [ ] **Step 4: main.go 接线**

`apps/pixoma/cmd/pixoma/main.go`：

```go
caseDeleteSvc := caseadmin.NewService(gdb, botRT.Notify)
```

放在 `botRT` 之后、`adminH` 之前；然后 `adminH` 的 `Cases` 改为：

```go
		Cases: &casesapi.Handler{
			Repo: caseRepo,
			Validate: func(doc catalogdomain.CaseDocument) error {
				if err := validator.ValidateDocument(doc); err != nil {
					return err
				}
				return validation.ValidateRouting(context.Background(), doc.Routing, topicRepo, conditionReg)
			},
			DeleteWithCleanup: caseDeleteSvc.DeleteCase,
		},
```

删除原来的 `CountMenuRefs`、`CountActiveSessions`、`CountActiveTasks` 三个闭包；加 import `"github.com/mr9esx/comfyui_tgbot/internal/caseadmin"`。

- [ ] **Step 5: 全量相关测试**

Run: `go test ./internal/httpapi/cases/... ./internal/caseadmin/...`
Expected: PASS。

Run: `go build ./...`
Expected: PASS。

- [ ] **Step 6: Commit**

```bash
git add internal/httpapi/cases/handler.go internal/httpapi/cases/handler_test.go apps/pixoma/cmd/pixoma/main.go
git commit -m "feat(api): cleanup-based case delete with ack_references"
```

---

### Task 5: 调度防御——case 不存在时 pending 终态失败

**Files:**
- Modify: `internal/runtime/application/orchestrator/service.go`
- Test: `internal/runtime/application/orchestrator/service_test.go`

**Interfaces:**
- Consumes: `catalogdomain.ErrNotFound`、`sharedkernel.TaskErrorCaseDeleted`/`CaseDeletedMessage`（Task 3）、`s.publishNotify`
- Produces: 无新接口；行为修复

- [ ] **Step 1: 失败测试**

`service_test.go` 追加：

```go
type missingCaseReader struct{}

func (missingCaseReader) GetCase(context.Context, sharedkernel.CaseID) (*catalogdomain.CaseDocument, error) {
	return nil, catalogdomain.ErrNotFound
}

func TestDispatchPendingFailsTerminalWhenCaseDeleted(t *testing.T) {
	ctx := context.Background()
	tasks := runtimedomain.NewMemoryTaskRepository()
	now := time.Unix(60, 0).UTC()
	task := runtimedomain.NewPending("t1", "s1", sharedkernel.CaseID(1), "inputs/t1", now)
	task.ChatID = "tg:9"
	_ = tasks.Create(ctx, task)
	n := &memNotify{}
	svc := orchestrator.New(tasks, static.New(edge.Instance{ID: "local", DispatchTopic: "dispatch.local"}), &captureBus{}, n)
	svc.Now = func() time.Time { return now }
	svc.Cases = missingCaseReader{}

	if err := svc.SchedulePending(ctx, 1); err != nil {
		t.Fatal(err)
	}
	got, _ := tasks.Get(ctx, "t1")
	if got.Status != sharedkernel.TaskFailed || got.ErrorCode != sharedkernel.TaskErrorCaseDeleted {
		t.Fatalf("task=%+v", got)
	}
	if len(n.items) != 1 || n.items[0].Kind != "task_failed" {
		t.Fatalf("notifies=%+v", n.items)
	}
}
```

（测试文件补 `catalogdomain` import。）

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/runtime/application/orchestrator/ -run TestDispatchPendingFailsTerminalWhenCaseDeleted -v`
Expected: FAIL（任务仍是 pending）。

- [ ] **Step 3: 实现**

`service.go` 的 `dispatchTask` 内，`resolveTopic` 出错分支改为：

```go
	topicKey, err := s.resolveTopic(ctx, t)
	if err != nil {
		if errors.Is(err, catalogdomain.ErrNotFound) {
			now := s.Now()
			if merr := t.MarkFailed(sharedkernel.TaskErrorCaseDeleted, sharedkernel.CaseDeletedMessage, now); merr == nil {
				_ = s.Tasks.Update(ctx, t)
				_ = s.publishNotify(ctx, t)
			}
			return nil
		}
		// Evaluation failure: keep pending with a recorded reason; the next
		// SchedulePending cycle retries (transient provider errors self-heal).
		t.ErrorMessage = "routing: " + err.Error()
		t.UpdatedAt = s.Now()
		_ = s.Tasks.Update(ctx, t)
		return nil
	}
```

（确认 `service.go` 已 import `catalogdomain` 与 `errors`；没有则补。）

- [ ] **Step 4: 重跑测试**

Run: `go test ./internal/runtime/application/orchestrator/`
Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add internal/runtime/application/orchestrator/service.go internal/runtime/application/orchestrator/service_test.go
git commit -m "fix(orchestrator): terminal-fail pending tasks whose case is gone"
```

---

### Task 6: sessions 管理 API 支持 case_id 过滤

**Files:**
- Modify: `internal/conversation/domain/service.go`（ListQuery 字段 + Memory List）
- Modify: `internal/conversation/infrastructure/persistence/gorm_session.go`（List 过滤）
- Modify: `internal/httpapi/sessions/handler.go`（parseListQuery）
- Modify: `internal/httpapi/sessions/handler_test.go`

**Interfaces:**
- Consumes: `domain.ListQuery`、`sharedkernel.CaseID`
- Produces: `GET /api/v1/sessions?case_id=N` —— Task 8/9 前端依赖

- [ ] **Step 1: 加字段与过滤 + 失败测试**

`internal/conversation/domain/service.go` 的 `ListQuery` 加：

```go
	CaseID      sharedkernel.CaseID
```

Memory `List` 内（`if q.UserID != ""` 附近）加：

```go
		if q.CaseID != 0 && s.CaseID != q.CaseID {
			continue
		}
```

`gorm_session.go` 的 `List` 内（`if q.Status != ""` 之后）加：

```go
	if q.CaseID != 0 {
		tx = tx.Where("case_id = ?", uint64(q.CaseID))
	}
```

`handler.go` 的 `parseListQuery` 内加：

```go
	if v := r.URL.Query().Get("case_id"); v != "" {
		n, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			return domain.ListQuery{}, fmt.Errorf("invalid case_id")
		}
		q.CaseID = sharedkernel.CaseID(n)
	}
```

（`handler.go` 补 `strconv`、`sharedkernel` import；`gorm_session.go` 无需新 import。）

`handler_test.go` 的 `TestSessionsHandler_ListGetReadOnly` 内（现有 `?user_id=user-a` 断言之后）追加：

```go
	res3, err := http.Get(srv.URL + "/api/v1/sessions?case_id=1")
	if err != nil {
		t.Fatal(err)
	}
	defer res3.Body.Close()
	if res3.StatusCode != http.StatusOK {
		t.Fatalf("list case_id status=%d", res3.StatusCode)
	}
	var byCase []map[string]any
	if err := json.NewDecoder(res3.Body).Decode(&byCase); err != nil {
		t.Fatal(err)
	}
	if len(byCase) != 1 || byCase[0]["id"] != "sess-a" {
		t.Fatalf("filter case_id: %+v", byCase)
	}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/httpapi/sessions/... -run TestSessionsHandler_ListGetReadOnly -v`
Expected: FAIL（case_id=1 过滤结果为空）。

- [ ] **Step 3: 重跑测试**

Run: `go test ./internal/httpapi/sessions/... ./internal/conversation/...`
Expected: PASS。

- [ ] **Step 4: Commit**

```bash
git add internal/conversation/domain/service.go internal/conversation/infrastructure/persistence/gorm_session.go internal/httpapi/sessions/handler.go internal/httpapi/sessions/handler_test.go
git commit -m "feat(sessions): filter admin list by case_id"
```

---

### Task 7: TG adapter 支持 session_terminated 通知文案

**Files:**
- Modify: `internal/channel/tg/adapter.go`
- Test: `internal/channel/tg/notify_test.go`

**Interfaces:**
- Consumes: `sharedkernel.UserNotify.Kind == "session_terminated"`（Task 3 发送）
- Produces: 无新接口

- [ ] **Step 1: 失败测试**

`notify_test.go` 追加：

```go
func TestHandleUserNotifySessionTerminated(t *testing.T) {
	out := &recordingOutbound{}
	a := tg.New(out)
	a.ChannelID = "tg"
	err := a.HandleUserNotify(context.Background(), sharedkernel.UserNotify{
		ChatID:   sharedkernel.ChatID("tg:123"),
		Kind:     "session_terminated",
		ErrorMsg: "该工作流已被管理员删除，当前会话已结束。",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.texts) != 1 || out.texts[0] != "该工作流已被管理员删除，当前会话已结束。" {
		t.Fatalf("texts=%+v", out.texts)
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `go test ./internal/channel/tg/ -run TestHandleUserNotifySessionTerminated -v`
Expected: FAIL（默认渲染成「任务 : session_terminated — …」）。

- [ ] **Step 3: 实现**

`adapter.go` 的 `HandleUserNotify` 中，`task_succeeded` 分支之后、`msg := fmt.Sprintf(...)` 之前插入：

```go
	if n.Kind == "session_terminated" {
		msg := n.ErrorMsg
		if msg == "" {
			msg = "该工作流已被管理员删除，当前会话已结束。"
		}
		return a.Out.SendText(ctx, addr, msg)
	}
```

- [ ] **Step 4: 重跑测试**

Run: `go test ./internal/channel/tg/`
Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add internal/channel/tg/adapter.go internal/channel/tg/notify_test.go
git commit -m "feat(tg): render session_terminated user notify"
```

---

### Task 8: 前端 ApiError 携带 code + client 解析

**Files:**
- Modify: `web/admin/src/lib/api/client.ts`
- Test: `web/admin/src/lib/api/client.test.ts`

**Interfaces:**
- Consumes: 后端 `{"error": ..., "code": ...}`（Task 4）
- Produces: `ApiError.code?: string` —— Task 10 依赖

- [ ] **Step 1: 失败测试**

`client.test.ts` 现有错误测试旁追加（沿用文件内 `vi.stubGlobal` 模式）：

```ts
  it('throws ApiError with backend error code', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            error: 'case is referenced',
            code: 'case_delete_needs_ack',
          }),
          {
            status: 409,
            headers: { 'Content-Type': 'application/json' },
          }
        )
      )
    )
    await expect(
      apiFetch('/api/v1/cases/1', { method: 'DELETE' })
    ).rejects.toMatchObject({
      status: 409,
      code: 'case_delete_needs_ack',
    } satisfies Partial<ApiError>)
  })
```

- [ ] **Step 2: 运行确认失败**

Run: `pnpm vitest run src/lib/api/client.test.ts`
Expected: FAIL（`err.code` 为 undefined）。

- [ ] **Step 3: 实现**

`client.ts`：

```ts
export class ApiError extends Error {
  status: number
  code?: string
  constructor(status: number, message: string, code?: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
  }
}
```

`apiFetch` 错误分支：

```ts
  if (!res.ok) {
    const msg =
      typeof body === 'object' &&
      body !== null &&
      'error' in body &&
      typeof (body as { error: unknown }).error === 'string'
        ? (body as { error: string }).error
        : `Request failed (${res.status})`
    const code =
      typeof body === 'object' &&
      body !== null &&
      'code' in body &&
      typeof (body as { code: unknown }).code === 'string'
        ? (body as { code: string }).code
        : undefined
    throw new ApiError(res.status, msg, code)
  }
```

- [ ] **Step 4: 重跑测试**

Run: `pnpm vitest run src/lib/api/client.test.ts`
Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add web/admin/src/lib/api/client.ts web/admin/src/lib/api/client.test.ts
git commit -m "feat(admin): surface backend error code on ApiError"
```

---

### Task 9: 前端 API 更新（deleteCase body、tasks/sessions case_id）

**Files:**
- Modify: `web/admin/src/lib/api/cases.ts`
- Modify: `web/admin/src/lib/api/tasks.ts`
- Modify: `web/admin/src/lib/api/sessions.ts`
- Test: `web/admin/src/lib/api/cases.test.ts`、`tasks.test.ts`、`sessions.test.ts`

**Interfaces:**
- Consumes: `apiFetch`（Task 8）
- Produces: `deleteCase(id, { ackReferences })`、`listTasks({ case_id })`、`listSessions({ case_id })` —— Task 10/11 依赖

- [ ] **Step 1: 失败测试**

`cases.test.ts` 现有 deleteCase 用例更新为：

```ts
  it('deleteCase sends ack_references body when ack is true', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ deleted: true }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      })
    )
    vi.stubGlobal('fetch', fetchMock)

    await deleteCase(42, true)

    expect(fetchMock).toHaveBeenCalledWith(
      'http://127.0.0.1:8081/api/v1/cases/42',
      expect.objectContaining({
        method: 'DELETE',
        body: JSON.stringify({ ack_references: true }),
      })
    )
  })
```

`tasks.test.ts` 补（沿用文件内模式）：

```ts
  it('listTasks filters by case_id', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify([{ id: 't1', status: 'pending' }]), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
    vi.stubGlobal('fetch', fetchMock)

    const data = await listTasks({ case_id: 10, status: 'pending' })

    expect(data[0].id).toBe('t1')
    expect(fetchMock).toHaveBeenCalledWith(
      'http://127.0.0.1:8081/api/v1/tasks?case_id=10&status=pending',
      expect.anything(),
    )
  })
```

`sessions.test.ts` 补：

```ts
  it('listSessions filters by case_id', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify([{ id: 's1' }]), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
    vi.stubGlobal('fetch', fetchMock)

    const data = await listSessions({ case_id: 10, status: 'collecting' })

    expect(data[0].id).toBe('s1')
    expect(fetchMock).toHaveBeenCalledWith(
      'http://127.0.0.1:8081/api/v1/sessions?case_id=10&status=collecting',
      expect.anything(),
    )
  })
```

- [ ] **Step 2: 运行确认失败**

Run: `pnpm vitest run src/lib/api/cases.test.ts src/lib/api/tasks.test.ts src/lib/api/sessions.test.ts`
Expected: FAIL（URL/body 不匹配）。

- [ ] **Step 3: 实现**

`cases.ts`：

```ts
export type DeleteCaseResult = {
  deleted: boolean
  removed_placements?: {
    channel_id: string
    channel_name?: string
    item_id: string
    label: string
    kind: string
  }[]
  failed_tasks?: number
  terminated_sessions?: number
}

export function deleteCase(id: number, ack?: boolean) {
  return apiFetch<DeleteCaseResult>(`/api/v1/cases/${id}`, {
    method: 'DELETE',
    ...(ack
      ? { body: JSON.stringify({ ack_references: true }) }
      : {}),
  })
}
```

`tasks.ts` 的 `listTasks` 参数加 `case_id?: number`；`sessions.ts` 的 `listSessions` 参数加 `case_id?: number`（`toQuery` 自动拼接）。

- [ ] **Step 4: 重跑测试**

Run: `pnpm vitest run src/lib/api/cases.test.ts src/lib/api/tasks.test.ts src/lib/api/sessions.test.ts`
Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add web/admin/src/lib/api/cases.ts web/admin/src/lib/api/tasks.ts web/admin/src/lib/api/sessions.ts web/admin/src/lib/api/cases.test.ts web/admin/src/lib/api/tasks.test.ts web/admin/src/lib/api/sessions.test.ts
git commit -m "feat(admin): delete case with ack; filter task/session lists by case"
```

---

### Task 10: i18n 文案 + localized-errors 改按 code 映射

**Files:**
- Modify: `web/admin/src/lib/i18n/locales/zh.json`
- Modify: `web/admin/src/lib/i18n/locales/en.json`
- Modify: `web/admin/src/lib/i18n/locale.test.ts`
- Modify: `web/admin/src/lib/api/localized-errors.ts`
- Modify: `web/admin/src/lib/api/localized-errors.test.ts`

**Interfaces:**
- Consumes: `ApiError.code`（Task 8）
- Produces: i18n keys + `caseDeleteErrorMessage(err, t)` 新映射 —— Task 11 依赖

- [ ] **Step 1: 先改 localized-errors 与测试**

`localized-errors.ts` 整体替换为：

```ts
import { ApiError } from './client'

/** Backend delete-conflict error codes → i18n keys. */
const CASE_DELETE_ERROR_CODES: Record<string, string> = {
  case_delete_needs_ack: 'cases.deleteNeedsAck',
}

/** Translate a case-delete error into localized guidance, if recognised. */
export function caseDeleteErrorMessage(
  err: unknown,
  t: (key: string) => string
): string | undefined {
  if (!(err instanceof ApiError)) return undefined
  const key = err.code ? CASE_DELETE_ERROR_CODES[err.code] : undefined
  return key ? t(key) : undefined
}
```

`localized-errors.test.ts` 替换为按 code 断言：

```ts
import { describe, expect, it } from 'vitest'
import { ApiError } from './client'
import { caseDeleteErrorMessage } from './localized-errors'

const t = (key: string) => `[${key}]`

describe('caseDeleteErrorMessage', () => {
  it('maps case_delete_needs_ack to i18n key', () => {
    const err = new ApiError(409, 'case is referenced', 'case_delete_needs_ack')
    expect(caseDeleteErrorMessage(err, t)).toBe('[cases.deleteNeedsAck]')
  })

  it('falls back to undefined for unrecognised or non-ApiError input', () => {
    expect(caseDeleteErrorMessage(new ApiError(409, 'boom'), t)).toBeUndefined()
    expect(caseDeleteErrorMessage(new Error('boom'), t)).toBeUndefined()
  })
})
```

- [ ] **Step 2: 运行确认失败**

Run: `pnpm vitest run src/lib/api/localized-errors.test.ts`
Expected: 替换后先跑一遍确认新断言通过（映射不存在时 FAIL，实现后 PASS）。

- [ ] **Step 3: i18n keys**

`zh.json` 的 `cases` 段：删除 `deleteConflictEnabled/Referenced/ActiveSessions/ActiveTasks` 四个 key，新增：

```json
    "deleteNeedsAck": "该工作流仍被菜单或卡片引用，请确认删除后将自动移除这些引用。",
    "deleteWillRemoveRefs": "删除后将自动移除以下 {{count}} 处引用：",
    "deleteWillFailTasks": "{{count}} 个排队任务将标记为失败（原因：工作流已删除）",
    "deleteWillEndSessions": "{{count}} 个进行中会话将被终止",
    "deleteAckRefs": "我已知悉引用将被自动移除",
    "deleteSuccessSummary": "工作流已删除（移除 {{refs}} 处引用、失败 {{tasks}} 个任务、终止 {{sessions}} 个会话）"
```

`en.json` 同位置：

```json
    "deleteNeedsAck": "This workflow is still referenced by menus or cards. Confirm deletion to remove the references automatically.",
    "deleteWillRemoveRefs": "The following {{count}} references will be removed automatically:",
    "deleteWillFailTasks": "{{count}} queued task(s) will be marked failed (reason: workflow deleted)",
    "deleteWillEndSessions": "{{count}} active session(s) will be terminated",
    "deleteAckRefs": "I understand the references will be removed automatically",
    "deleteSuccessSummary": "Workflow deleted ({{refs}} references removed, {{tasks}} tasks failed, {{sessions}} sessions terminated)"
```

`locale.test.ts` 新增成对检查列表（仿 `COMMAND_MENU_I18N_KEYS`）：

```ts
const CASE_DELETE_I18N_KEYS = [
  'cases.deleteNeedsAck',
  'cases.deleteWillRemoveRefs',
  'cases.deleteWillFailTasks',
  'cases.deleteWillEndSessions',
  'cases.deleteAckRefs',
  'cases.deleteSuccessSummary',
] as const

it('defines case-delete keys in zh and en locales', () => {
  for (const key of CASE_DELETE_I18N_KEYS) {
    expect(lookup(zh, key), `zh missing ${key}`).toEqual(expect.any(String))
    expect(lookup(en, key), `en missing ${key}`).toEqual(expect.any(String))
  }
})
```

- [ ] **Step 4: 运行测试**

Run: `pnpm vitest run src/lib/api/localized-errors.test.ts src/lib/i18n/locale.test.ts`
Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add web/admin/src/lib/api/localized-errors.ts web/admin/src/lib/api/localized-errors.test.ts web/admin/src/lib/i18n/locales/zh.json web/admin/src/lib/i18n/locales/en.json web/admin/src/lib/i18n/locale.test.ts
git commit -m "feat(admin): localized case-delete guidance by error code"
```

---

### Task 11: 前端删除弹窗改造

**Files:**
- Modify: `web/admin/src/features/cases/detail-panel.tsx`
- Test: `web/admin/src/features/cases/cases-detail.contract.test.ts`

**Interfaces:**
- Consumes: `getCaseMenuPlacements`、`queryKeys.cases.menuPlacements`、`listTasks({ case_id, status })`、`listSessions({ case_id, status })`、`deleteCase(id, { ackReferences })`（Task 9）、`caseDeleteErrorMessage`（Task 10）
- Produces: 新弹窗 UX

- [ ] **Step 1: 改 contract 测试（先失败）**

`cases-detail.contract.test.ts` 的 `'detail panel offers delete workflow with confirmation'` 用例更新断言：

```ts
    expect(source).toContain('cases.deleteWillRemoveRefs')
    expect(source).toContain('cases.deleteWillFailTasks')
    expect(source).toContain('cases.deleteWillEndSessions')
    expect(source).toContain('cases.deleteAckRefs')
    expect(source).toContain('deleteCase(record.id, ackRefs)')
    expect(source).toContain('getCaseMenuPlacements')
    expect(source).not.toContain('disableCase(record.id)')
```

- [ ] **Step 2: 运行确认失败**

Run: `pnpm vitest run src/features/cases/cases-detail.contract.test.ts`
Expected: FAIL。

- [ ] **Step 3: 实现 detail-panel.tsx**

import 区：

```tsx
import { getCaseMenuPlacements } from '@/lib/api/channel-menu'
import { listSessions } from '@/lib/api/sessions'
import { listTasks } from '@/lib/api/tasks'
```

组件内新增 state 与 queries（`deleteMutation` 之前）：

```tsx
  const [ackRefs, setAckRefs] = useState(false)

  const placementsQuery = useQuery({
    queryKey: queryKeys.cases.menuPlacements(id),
    queryFn: () => getCaseMenuPlacements(id),
  })
  const pendingTasksQuery = useQuery({
    queryKey: ['cases', id, 'pending-tasks'] as const,
    queryFn: () => listTasks({ case_id: id, status: 'pending' }),
  })
  const activeSessionsQuery = useQuery({
    queryKey: ['cases', id, 'active-sessions'] as const,
    queryFn: async () => {
      const [collecting, confirming] = await Promise.all([
        listSessions({ case_id: id, status: 'collecting' }),
        listSessions({ case_id: id, status: 'confirming' }),
      ])
      return collecting.length + confirming.length
    },
  })
```

`deleteMutation` 改为：

```tsx
  const deleteMutation = useMutation({
    mutationFn: async () => {
      if (!record) throw new Error('case missing')
      return deleteCase(record.id, ackRefs)
    },
    onSuccess: async (summary) => {
      const refs = summary.removed_placements?.length ?? 0
      if (refs > 0 || summary.failed_tasks > 0 || summary.terminated_sessions > 0) {
        toast.success(
          t('cases.deleteSuccessSummary', {
            refs,
            tasks: summary.failed_tasks ?? 0,
            sessions: summary.terminated_sessions ?? 0,
          })
        )
      } else {
        toast.success(t('cases.deleteSuccess'))
      }
      await queryClient.invalidateQueries({ queryKey: queryKeys.cases.all })
      queryClient.removeQueries({ queryKey: queryKeys.cases.detail(id) })
      void navigate({ to: '/cases', state: { backToList: true } } as never)
    },
    onError: (err) => {
      const detail = caseDeleteErrorMessage(err, t) ?? errorMessage(err)
      toast.error(
        detail
          ? `${t('cases.deleteFailed')}：${detail}`
          : t('cases.deleteFailed')
      )
    },
  })
```

删除 `record.enabled ? await disableCase(record.id)` 分支；若 `disableCase` 不再被引用则移除其 import。

从 `zh.json` / `en.json` 删除 `deleteWorkflowNeedDisable`（弹窗不再展示「删除前会自动先停用」段落）。

`AlertDialogContent` 内、`AlertDialogDescription` 末尾（`deleteWorkflowNeedDisable` 段落之后）追加：

```tsx
                      {placementsQuery.data && placementsQuery.data.length > 0 ? (
                        <div className='mt-3 space-y-2'>
                          <p className='font-medium'>
                            {t('cases.deleteWillRemoveRefs', {
                              count: placementsQuery.data.length,
                            })}
                          </p>
                          <ul className='max-h-32 overflow-auto rounded-md border bg-muted/20 p-3 text-xs'>
                            {placementsQuery.data.map((p) => (
                              <li key={`${p.channel_id}:${p.item_id}`}>
                                {p.channel_name || p.channel_id} ·{' '}
                                {p.path.map((s) => s.label).join(' / ')}
                              </li>
                            ))}
                          </ul>
                        </div>
                      ) : null}
                      {(pendingTasksQuery.data?.length ?? 0) > 0 ||
                      (activeSessionsQuery.data ?? 0) > 0 ? (
                        <p className='mt-3 text-xs text-muted-foreground'>
                          {(pendingTasksQuery.data?.length ?? 0) > 0
                            ? t('cases.deleteWillFailTasks', {
                                count: pendingTasksQuery.data?.length ?? 0,
                              })
                            : null}
                          {(activeSessionsQuery.data ?? 0) > 0
                            ? t('cases.deleteWillEndSessions', {
                                count: activeSessionsQuery.data ?? 0,
                              })
                            : null}
                        </p>
                      ) : null}
                      <label className='mt-4 flex items-start gap-2 text-sm'>
                        <input
                          type='checkbox'
                          checked={ackRefs}
                          onChange={(e) => setAckRefs(e.target.checked)}
                          data-testid='case-delete-ack'
                        />
                        <span>{t('cases.deleteAckRefs')}</span>
                      </label>
```

确认按钮加 `disabled={deleteMutation.isPending || !ackRefs}`，并移除 `deleteWorkflowNeedDisable` 段落（该文案不再需要）与对应的 i18n key（Task 10 未删则本任务从 zh/en 一并删除 `deleteWorkflowNeedDisable`）。

- [ ] **Step 4: 重跑 contract 测试 + 全量前端测试**

Run: `pnpm vitest run src/features/cases/cases-detail.contract.test.ts`
Expected: PASS。

Run: `pnpm vitest run`
Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add web/admin/src/features/cases/detail-panel.tsx web/admin/src/features/cases/cases-detail.contract.test.ts web/admin/src/lib/i18n/locales/zh.json web/admin/src/lib/i18n/locales/en.json
git commit -m "feat(admin): case delete dialog with reference impact and ack"
```

---

### Task 12: 全量验证

**Files:** 无代码改动

- [ ] **Step 1: 后端全量测试 + 构建**

Run: `go test ./...`
Expected: PASS。

Run: `go vet ./...`
Expected: PASS。

- [ ] **Step 2: 前端全量测试 + 类型检查 + lint**

Run: `pnpm vitest run`
Expected: PASS。

Run: `pnpm exec tsc -b`
Expected: PASS。

Run: `pnpm exec eslint src/lib/api src/features/cases src/features/cases/sections`
Expected: PASS。

- [ ] **Step 3: 手工冒烟（可选）**

启动后端 + 前端，删除一个「被菜单引用 + 有 pending 任务」的 Case：弹窗应展示引用与影响数，勾选后删除成功并返回摘要；被终止会话的用户在 TG 收到「该工作流已被管理员删除，当前会话已结束。」。

- [ ] **Step 4: 收尾确认**

`git status --porcelain` 应只有本计划相关改动；如有遗留，按任务归属 commit。
