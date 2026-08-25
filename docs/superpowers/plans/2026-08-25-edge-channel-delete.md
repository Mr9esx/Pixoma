# 计算节点与渠道清理式删除 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 计算节点与渠道删除改为「确认制清理式删除」：节点删除时 running 任务终态失败（`edge_deleted`）并通知用户、清理 presence；渠道删除不再要求先停用，事务内删渠道行 + 终止活跃会话（exited）并通知会话用户 + 清理菜单/卡片。两者确认弹窗都先实时展示任务/会话状态，有影响时要求二次确认。

**Architecture:** 新增 `internal/edgeadmin`（仿 caseadmin，gdb 事务 + notify）负责节点清理；新增 `internal/channeladmin` 提供渠道删除清理闭包（删渠道行 + 会话 exited + 菜单/卡片清理），`channelapp.Service` 去掉 Enabled/HasActiveRefs 改为注入 `DeleteWithCleanup` + `Notify`。前端在节点/渠道详情页增加带影响面与勾选确认的删除弹窗。

**Tech Stack:** Go 1.x + GORM（SQLite 测试）、chi、React 18 + TanStack Query + react-i18next + sonner、Vitest（node 合同测试）、pnpm。

## Global Constraints

- 不新增第三方依赖。
- 后端 API 错误 `error` 字段保持英文兜底；用户可见文案走前端 i18n 或 TG adapter 固定中文。
- 新增 i18n key 必须 zh/en 成对，并在 `web/admin/src/lib/i18n/locale.test.ts` 的成对列表登记。
- 节点删除只处理 running 任务；queued/pending 不处理；metrics 历史保留。
- 渠道删除不中断执行中/排队任务。
- 不停止远程 agent 进程（只提示）。
- 前端 node 测试文件必须已在 `vitest.config.ts` include（本计划只修改现有测试文件，不新增）。
- 每个任务以独立可测试交付物结束并单独 commit；涉及 `apps/pixoma/cmd/pixoma/main.go` 时用 `git add -p` 只暂存本任务 hunk（该文件可能有并发未提交改动）。

---

### Task 1: presence.Remove + sharedkernel 节点删除常量

**Files:**
- Modify: `internal/platform/presence/store.go`
- Test: `internal/platform/presence/store_test.go`
- Modify: `internal/sharedkernel/ids.go`

**Interfaces:**
- Consumes: `presence.Store`（已有）
- Produces: `Store.Remove(id sharedkernel.EdgeID)`、`sharedkernel.TaskErrorEdgeDeleted`、`sharedkernel.EdgeDeletedMessage` —— Task 6 依赖

- [ ] **Step 1: 失败测试**

`store_test.go` 追加（复用文件内 `NewStore` + `Report` 模式）：

```go
func TestStore_RemoveClearsRecord(t *testing.T) {
	s := NewStore()
	s.Report("gpu-1", true)
	if got := s.Snapshot("gpu-1"); !got.EdgeOnline {
		t.Fatalf("snapshot before remove: %+v", got)
	}
	s.Remove("gpu-1")
	if got := s.Snapshot("gpu-1"); got.EdgeOnline || got.ComfyRunning {
		t.Fatalf("snapshot after remove: %+v", got)
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `go test ./internal/platform/presence/ -run TestStore_RemoveClearsRecord -v`
Expected: FAIL（`Remove` 未定义）。

- [ ] **Step 3: 实现**

`store.go` 的 `Touch` 之后追加：

```go
// Remove drops the in-memory record for an edge (admin delete cleanup).
func (s *Store) Remove(id sharedkernel.EdgeID) {
	if s == nil || id == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.records, id)
}
```

`sharedkernel/ids.go` 的任务常量旁追加：

```go
// Terminal-failure metadata used when a compute node is deleted.
const (
	TaskErrorEdgeDeleted = "edge_deleted"
	EdgeDeletedMessage   = "节点已删除"
)
```

- [ ] **Step 4: 重跑测试**

Run: `go test ./internal/platform/presence/ ./internal/sharedkernel/`
Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add internal/platform/presence/store.go internal/platform/presence/store_test.go internal/sharedkernel/ids.go
git commit -m "feat(presence): remove edge records; edge-deleted task constants"
```

---

### Task 2: sessions 管理 API 支持 channel_id 过滤

**Files:**
- Modify: `internal/conversation/domain/service.go`（ListQuery + Memory List）
- Modify: `internal/conversation/infrastructure/persistence/gorm_session.go`
- Modify: `internal/httpapi/sessions/handler.go`
- Modify: `internal/httpapi/sessions/handler_test.go`

**Interfaces:**
- Consumes: `domain.ListQuery`、`sharedkernel.ParseChatID`
- Produces: `GET /api/v1/sessions?channel_id=X` —— Task 11 前端依赖

- [ ] **Step 1: 加字段/过滤 + 失败测试**

`domain/service.go` 的 `ListQuery` 加：

```go
	ChannelID   string
```

Memory `List` 内（`if q.CaseID != 0` 之后）加：

```go
		if q.ChannelID != "" {
			addr, err := sharedkernel.ParseChatID(string(s.ChatID))
			if err != nil || addr.ChannelID != q.ChannelID {
				continue
			}
		}
```

`gorm_session.go` 的 `List` 内（`if q.CaseID != 0` 之后）加：

```go
	if q.ChannelID != "" {
		tx = tx.Where("channel_id = ?", q.ChannelID)
	}
```

`handler.go` 的 `parseListQuery` 内（`case_id` 之后）加：

```go
	if v := r.URL.Query().Get("channel_id"); v != "" {
		q.ChannelID = v
	}
```

`handler_test.go` 的 `TestSessionsHandler_ListGetReadOnly` 内（`case_id=1` 断言之后）追加：

```go
	resChan, err := http.Get(srv.URL + "/api/v1/sessions?channel_id=tg")
	if err != nil {
		t.Fatal(err)
	}
	defer resChan.Body.Close()
	if resChan.StatusCode != http.StatusOK {
		t.Fatalf("list channel_id status=%d", resChan.StatusCode)
	}
	var byChannel []map[string]any
	if err := json.NewDecoder(resChan.Body).Decode(&byChannel); err != nil {
		t.Fatal(err)
	}
	if len(byChannel) != 1 || byChannel[0]["id"] != "sess-a" {
		t.Fatalf("filter channel_id: %+v", byChannel)
	}
```

（s1 的 ChatID 是 `tg:101` → channel_id=`tg`，s2 是 `tg:202`。）

- [ ] **Step 2: 运行确认失败**

Run: `go test ./internal/httpapi/sessions/ -run TestSessionsHandler_ListGetReadOnly -v`
Expected: FAIL（channel_id 过滤未生效，返回 0 或 2 条）。

- [ ] **Step 3: 重跑测试**

Run: `go test ./internal/httpapi/sessions/... ./internal/conversation/...`
Expected: PASS。

- [ ] **Step 4: Commit**

```bash
git add internal/conversation/domain/service.go internal/conversation/infrastructure/persistence/gorm_session.go internal/httpapi/sessions/handler.go internal/httpapi/sessions/handler_test.go
git commit -m "feat(sessions): filter admin list by channel_id"
```

---

### Task 3: tasks 管理 API 支持 channel_id 过滤（join sessions）

**Files:**
- Modify: `internal/runtime/domain/task_repository.go`（AdminListQuery）
- Modify: `internal/runtime/infrastructure/persistence/gorm_task.go`
- Modify: `internal/httpapi/tasks/handler.go`
- Modify: `internal/httpapi/tasks/handler_test.go`

**Interfaces:**
- Consumes: `AdminListQuery`、`sessions` 表（channel_id 关联）
- Produces: `GET /api/v1/tasks?channel_id=X` —— Task 11 前端依赖

- [ ] **Step 1: 加字段/过滤 + 失败测试**

`task_repository.go` 的 `AdminListQuery` 加：

```go
	// ChannelID filters tasks whose session belongs to a channel (join sessions).
	ChannelID string
```

`gorm_task.go` 的 `List` 改为（把 `joinChat` 扩为 `join`）：

```go
	join := q.ChatID != "" || q.ChannelID != ""
	col := func(name string) string {
		if join {
			return "tasks." + name
		}
		return name
	}

	var tx *gorm.DB
	switch {
	case q.ChatID != "":
		addr, err := sharedkernel.ParseChatID(string(q.ChatID))
		if err != nil {
			return nil, err
		}
		tx = r.db.WithContext(ctx).Table("tasks").
			Joins("JOIN sessions ON tasks.session_id = sessions.id").
			Where("sessions.channel_id = ? AND sessions.chat_external_id = ?", addr.ChannelID, addr.ExternalChatID)
	case q.ChannelID != "":
		tx = r.db.WithContext(ctx).Table("tasks").
			Joins("JOIN sessions ON tasks.session_id = sessions.id").
			Where("sessions.channel_id = ?", q.ChannelID)
	default:
		tx = r.db.WithContext(ctx).Model(&TaskRow{})
	}
```

`handler.go` 的 list 解析内（`case_id` 之后）加：

```go
	if v := r.URL.Query().Get("channel_id"); v != "" {
		q.ChannelID = v
	}
```

`handler_test.go` 追加（复用现有 sqlite + AutoMigrate 模式；确认文件内 `openTasksHandler` 存在并迁移了 tasks + sessions 表，若只迁移 tasks 则补 `&sesspersist.SessionRow{}`）：

```go
func TestTasksHandler_FilterByChannel(t *testing.T) {
	gdb, err := db.Open(db.Options{DSN: "file:tasks_chan_" + t.Name() + "?mode=memory&cache=shared"})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(gdb, &taskpersist.TaskRow{}, &sesspersist.SessionRow{}); err != nil {
		t.Fatal(err)
	}
	tasks := taskpersist.NewTaskRepository(gdb)
	sessions := sesspersist.NewSessionRepository(gdb)
	ctx := context.Background()
	now := time.Now().UTC()
	if err := sessions.Save(ctx, convdomain.NewCollecting("s1", "tg:9", 1, []string{"a"}, now)); err != nil {
		t.Fatal(err)
	}
	if err := sessions.Save(ctx, convdomain.NewCollecting("s2", "ig:9", 1, []string{"a"}, now)); err != nil {
		t.Fatal(err)
	}
	_ = tasks.Create(ctx, runtimedomain.NewPending("t1", "s1", 1, "inputs/t1", now))
	_ = tasks.Create(ctx, runtimedomain.NewPending("t2", "s2", 1, "inputs/t2", now))

	h := &Handler{Tasks: tasks}
	r := chi.NewRouter()
	r.Route("/api/v1/tasks", func(r chi.Router) { h.Mount(r) })
	srv := httptest.NewServer(r)
	defer srv.Close()

	res, err := http.Get(srv.URL + "/api/v1/tasks?channel_id=tg")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", res.StatusCode)
	}
	var list []map[string]any
	if err := json.NewDecoder(res.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0]["id"] != "t1" {
		t.Fatalf("channel filter: %+v", list)
	}
}
```

（`handler_test.go` 顶部 import 需补：`context`、`github.com/mr9esx/comfyui_tgbot/internal/conversation/domain`（convdomain）、`.../conversation/infrastructure/persistence`（sesspersist）、`.../platform/db`、`.../runtime/domain`（runtimedomain）、`.../runtime/infrastructure/persistence`（taskpersist）、`gorm.io/gorm`。）

`MemoryTaskRepository.List` 不实现 ChannelID 过滤（文档注明：仅 GORM 支持）。

- [ ] **Step 2: 运行确认失败**

Run: `go test ./internal/httpapi/tasks/ -run TestTasksHandler_FilterByChannel -v`
Expected: FAIL（未过滤，返回 2 条）。

- [ ] **Step 3: 重跑测试**

Run: `go test ./internal/httpapi/tasks/... ./internal/runtime/...`
Expected: PASS。

- [ ] **Step 4: Commit**

```bash
git add internal/runtime/domain/task_repository.go internal/runtime/infrastructure/persistence/gorm_task.go internal/httpapi/tasks/handler.go internal/httpapi/tasks/handler_test.go
git commit -m "feat(tasks): filter admin list by channel via session join"
```

---

### Task 4: channeladmin 渠道删除清理闭包

**Files:**
- Create: `internal/channeladmin/delete.go`
- Create: `internal/channeladmin/delete_test.go`

**Interfaces:**
- Consumes: `channelpersist.ChannelRow`、`sesspersist.SessionRow`、`mencardpersist.MainMenuRow/CardRow`、`convdomain.Status*`、`sharedkernel.FormatChatID`
- Produces: `channeladmin.DeleteWithCleanup(db *gorm.DB) func(ctx, channelID string) ([]sharedkernel.ChatID, error)` —— Task 5 依赖

- [ ] **Step 1: 失败测试**

`delete_test.go`：

```go
package channeladmin_test

import (
	"context"
	"testing"
	"time"

	channelpersist "github.com/mr9esx/comfyui_tgbot/internal/channel/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/channeladmin"
	convdomain "github.com/mr9esx/comfyui_tgbot/internal/conversation/domain"
	sesspersist "github.com/mr9esx/comfyui_tgbot/internal/conversation/infrastructure/persistence"
	mcdomain "github.com/mr9esx/comfyui_tgbot/internal/menucard/domain"
	mencardpersist "github.com/mr9esx/comfyui_tgbot/internal/menucard/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/db"
)

func TestDeleteWithCleanup(t *testing.T) {
	gdb, err := db.Open(db.Options{DSN: "file:ch_delete_" + t.Name() + "?mode=memory&cache=shared"})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(gdb,
		&channelpersist.ChannelRow{},
		&sesspersist.SessionRow{},
		&mencardpersist.MainMenuRow{},
		&mencardpersist.CardRow{}); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	now := time.Now().UTC()

	chRepo := channelpersist.NewGormRepository(gdb)
	if err := gdb.Create(&channelpersist.ChannelRow{
		ID:                   "tg",
		Platform:             "telegram",
		Name:                 "主",
		CredentialCiphertext: "enc",
		Enabled:              true,
		CreatedAt:            now,
		UpdatedAt:            now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	sessRepo := sesspersist.NewSessionRepository(gdb)
	_ = sessRepo.Save(ctx, convdomain.NewCollecting("s1", "tg:9", 1, []string{"a"}, now))
	submitted := convdomain.NewCollecting("s2", "tg:8", 1, []string{"a"}, now)
	submitted.Status = convdomain.StatusSubmitted
	_ = sessRepo.Save(ctx, submitted)
	menuRepo := mencardpersist.NewGormCardRepository(gdb)
	_ = menuRepo.PutMenu(ctx, "tg", mcdomain.Menu{ID: "m", Name: "主", Columns: 2})
	_ = menuRepo.CreateCard(ctx, "tg", mcdomain.Card{ID: "c1", Name: "卡", Text: "hi"})

	cleanup := channeladmin.DeleteWithCleanup(gdb)
	chats, err := cleanup(ctx, "tg")
	if err != nil {
		t.Fatal(err)
	}
	if len(chats) != 1 || chats[0] != "tg:9" {
		t.Fatalf("chats=%+v", chats)
	}
	if _, err := chRepo.Get(ctx, "tg"); err == nil {
		t.Fatal("channel must be deleted")
	}
	active, _ := sessRepo.ListActiveByCase(ctx, 1)
	_ = active
	var exited []sesspersist.SessionRow
	if err := gdb.Where("channel_id = ?", "tg").Find(&exited).Error; err != nil {
		t.Fatal(err)
	}
	for _, row := range exited {
		if row.Status != string(convdomain.StatusExited) {
			t.Fatalf("session %s status=%s want exited", row.ID, row.Status)
		}
	}
	if _, err := menuRepo.GetMenu(ctx, "tg"); err == nil {
		t.Fatal("menu must be deleted")
	}
	cards, _ := menuRepo.ListCards(ctx, "tg")
	if len(cards) != 0 {
		t.Fatalf("cards=%+v", cards)
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `go test ./internal/channeladmin/ -v`
Expected: FAIL（包不存在）。

- [ ] **Step 3: 实现 delete.go**

```go
// Package channeladmin implements admin channel lifecycle cleanup.
package channeladmin

import (
	"context"
	"time"

	"gorm.io/gorm"

	channelpersist "github.com/mr9esx/comfyui_tgbot/internal/channel/infrastructure/persistence"
	convdomain "github.com/mr9esx/comfyui_tgbot/internal/conversation/domain"
	sesspersist "github.com/mr9esx/comfyui_tgbot/internal/conversation/infrastructure/persistence"
	mencardpersist "github.com/mr9esx/comfyui_tgbot/internal/menucard/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

// DeleteWithCleanup returns a function that deletes the channel row,
// terminates its active sessions (collecting/confirming → exited) and removes
// channel-scoped menu/card rows in one transaction. It returns the chats of
// terminated sessions so the caller can notify users after commit.
func DeleteWithCleanup(db *gorm.DB) func(ctx context.Context, channelID string) ([]sharedkernel.ChatID, error) {
	return func(ctx context.Context, channelID string) ([]sharedkernel.ChatID, error) {
		var chats []sharedkernel.ChatID
		err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.Where("id = ?", channelID).Delete(&channelpersist.ChannelRow{}).Error; err != nil {
				return err
			}
			var rows []sesspersist.SessionRow
			if err := tx.Where("channel_id = ? AND status IN ?", channelID, []string{
				string(convdomain.StatusCollecting),
				string(convdomain.StatusConfirming),
			}).Find(&rows).Error; err != nil {
				return err
			}
			for _, row := range rows {
				chats = append(chats, sharedkernel.ChatID(sharedkernel.FormatChatID(sharedkernel.ChannelAddr{
					ChannelID:      row.ChannelID,
					ExternalChatID: row.ChatExternalID,
				})))
			}
			if len(rows) > 0 {
				if err := tx.Model(&sesspersist.SessionRow{}).
					Where("channel_id = ? AND status IN ?", channelID, []string{
						string(convdomain.StatusCollecting),
						string(convdomain.StatusConfirming),
					}).
					Updates(map[string]any{
						"status":     string(convdomain.StatusExited),
						"updated_at": time.Now().UTC(),
					}).Error; err != nil {
					return err
				}
			}
			if err := tx.Where("channel_id = ?", channelID).Delete(&mencardpersist.MainMenuRow{}).Error; err != nil {
				return err
			}
			return tx.Where("channel_id = ?", channelID).Delete(&mencardpersist.CardRow{}).Error
		})
		return chats, err
	}
}
```

- [ ] **Step 4: 重跑测试**

Run: `go test ./internal/channeladmin/ -v`
Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add internal/channeladmin/delete.go internal/channeladmin/delete_test.go
git commit -m "feat(channeladmin): transactional channel delete cleanup"
```

---

### Task 5: 渠道直接删除（channelapp.Service + handler + main.go）

**Files:**
- Modify: `internal/channel/application/service.go`
- Modify: `internal/channel/application/service_test.go`
- Modify: `internal/httpapi/channels/handler.go`
- Modify: `internal/httpapi/channels/handler_test.go`
- Modify: `internal/httpapi/adminhost/server_test.go`
- Modify: `apps/pixoma/cmd/pixoma/main.go`

**Interfaces:**
- Consumes: `channeladmin.DeleteWithCleanup`（Task 4）、`notify.Publisher`
- Produces: `channelapp.Service.Delete` 新行为（不再 409）

- [ ] **Step 1: 先改测试（失败）**

`service_test.go` 的 `TestService_DeleteRestricted` 替换为：

```go
type memNotify struct {
	items []sharedkernel.UserNotify
}

func (m *memNotify) Publish(_ context.Context, n sharedkernel.UserNotify) error {
	m.items = append(m.items, n)
	return nil
}

func TestService_DeleteDirectWithCleanup(t *testing.T) {
	store := &memStore{rows: map[string]domain.Channel{}}
	key := make([]byte, 32)
	n := &memNotify{}
	var gotChats []sharedkernel.ChatID
	svc := &Service{
		Store:  store,
		Key:    key,
		Notify: n,
		DeleteWithCleanup: func(_ context.Context, id string) ([]sharedkernel.ChatID, error) {
			if err := store.Delete(context.Background(), id); err != nil {
				return nil, err
			}
			gotChats = []sharedkernel.ChatID{"tg:9"}
			return gotChats, nil
		},
	}
	if _, err := svc.Create(context.Background(), "tg-1", domain.PlatformTelegram, "a", "t"); err != nil {
		t.Fatal(err)
	}
	// 启用中直接删除成功，不再要求停用。
	if err := svc.Delete(context.Background(), "tg-1"); err != nil {
		t.Fatalf("enabled delete: %v", err)
	}
	if _, err := svc.Get(context.Background(), "tg-1"); err != domain.ErrNotFound {
		t.Fatalf("want not found, got %v", err)
	}
	if len(n.items) != 1 || n.items[0].Kind != "session_terminated" {
		t.Fatalf("notifies=%+v", n.items)
	}
}

func TestService_DeleteFallsBackToStore(t *testing.T) {
	store := &memStore{rows: map[string]domain.Channel{}}
	svc := &Service{Store: store, Key: make([]byte, 32)}
	if _, err := svc.Create(context.Background(), "tg-1", domain.PlatformTelegram, "a", "t"); err != nil {
		t.Fatal(err)
	}
	if err := svc.Delete(context.Background(), "tg-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Get(context.Background(), "tg-1"); err != domain.ErrNotFound {
		t.Fatalf("want not found, got %v", err)
	}
}
```

`handler_test.go`：`openChannelsServer` 里删掉 `HasActiveRefs` 行；把「启用中删除 409」段替换为「启用中直接删除成功」：

```go
	// 启用中直接删除成功（不再要求停用）。
	delReq, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/v1/channels/"+id, nil)
	delRes, err := http.DefaultClient.Do(delReq)
	if err != nil {
		t.Fatal(err)
	}
	delRes.Body.Close()
	if delRes.StatusCode != http.StatusOK {
		t.Fatalf("delete enabled status=%d want 200", delRes.StatusCode)
	}
```

删掉原「禁用后删除成功」段落（含 `disable` 调用）。

`adminhost/server_test.go` 的 `chSvc` 构造删掉 `HasActiveRefs` 行。

- [ ] **Step 2: 运行确认失败**

Run: `go test ./internal/channel/... ./internal/httpapi/channels/... ./internal/httpapi/adminhost/...`
Expected: FAIL（Service 结构体无 `Notify`/`DeleteWithCleanup` 字段）。

- [ ] **Step 3: 实现 service.go**

删除 `HasActiveRefs HasActiveRefsFunc` 字段与 `HasActiveRefsFunc` 类型；`Service` 加：

```go
	Notify notify.Publisher
	// DeleteWithCleanup deletes the channel row, terminates its active sessions
	// and removes channel-scoped menu/card rows in one transaction; returns
	// chats to notify after commit.
	DeleteWithCleanup func(ctx context.Context, channelID string) ([]sharedkernel.ChatID, error)
```

import 补 `log/slog`、`notify`、`sharedkernel`。

`Delete` 替换为：

```go
// Delete removes a channel directly. When DeleteWithCleanup is wired, active
// sessions are terminated and menu/card rows are removed in the same
// transaction; affected users are notified best-effort after commit.
func (s *Service) Delete(ctx context.Context, id string) error {
	if _, err := s.Store.Get(ctx, id); err != nil {
		return err
	}
	if s.DeleteWithCleanup == nil {
		return s.Store.Delete(ctx, id)
	}
	chats, err := s.DeleteWithCleanup(ctx, id)
	if err != nil {
		return err
	}
	for _, chat := range chats {
		n := sharedkernel.UserNotify{
			ChatID:   chat,
			Kind:     "session_terminated",
			ErrorMsg: "该渠道已被管理员删除，当前会话已结束。",
		}
		if s.Notify == nil {
			continue
		}
		if err := s.Notify.Publish(ctx, n); err != nil {
			slog.Warn("channel delete: session notify failed", "chat", chat, "err", err)
		}
	}
	return nil
}
```

`handler.go` 的 `Delete` 删掉 `ErrDeleteRestricted` 分支（保留 404/500/200）。

`main.go`：`chSvc` 构造改为：

```go
	chSvc := &channelapp.Service{
		Store:             channelStore,
		Key:               encKey,
		Notify:            botRT.Notify,
		DeleteWithCleanup: channeladmin.DeleteWithCleanup(gdb),
	}
```

删除原来的 `HasActiveRefs` 闭包；加 import `"github.com/mr9esx/comfyui_tgbot/internal/channeladmin"`。

- [ ] **Step 4: 重跑测试**

Run: `go test ./internal/channel/... ./internal/httpapi/channels/... ./internal/httpapi/adminhost/...`
Expected: PASS。

Run: `go build ./...`
Expected: PASS。

- [ ] **Step 5: Commit（main.go 用 git add -p 只暂存本任务 hunk）**

```bash
git add internal/channel/application/service.go internal/channel/application/service_test.go internal/httpapi/channels/handler.go internal/httpapi/channels/handler_test.go internal/httpapi/adminhost/server_test.go
git add -p apps/pixoma/cmd/pixoma/main.go
git commit -m "feat(channels): direct delete with session termination and cleanup"
```

---

### Task 6: edgeadmin 节点删除清理服务

**Files:**
- Create: `internal/edgeadmin/edge_delete.go`
- Create: `internal/edgeadmin/edge_delete_test.go`

**Interfaces:**
- Consumes: `instpersist.NewEdgeRepository`、`taskpersist.TaskRepository`、`sesspersist.SessionRepository`、`sharedkernel.TaskErrorEdgeDeleted/EdgeDeletedMessage`（Task 1）、`notify.Publisher`
- Produces: `edgeadmin.ErrNeedsAck`、`DeleteSummary`、`Service.DeleteEdge` —— Task 7 依赖

- [ ] **Step 1: 失败测试**

`edge_delete_test.go`（模式仿 `internal/caseadmin/case_delete_test.go`，AutoMigrate edges + tasks + sessions）：

```go
package edgeadmin_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"gorm.io/gorm"

	convdomain "github.com/mr9esx/comfyui_tgbot/internal/conversation/domain"
	sesspersist "github.com/mr9esx/comfyui_tgbot/internal/conversation/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/edgeadmin"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/db"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/edge"
	instpersist "github.com/mr9esx/comfyui_tgbot/internal/platform/edge/persistence"
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

func newEdgeTest(t *testing.T) (*gorm.DB, *edgeadmin.Service, *captureNotify, context.Context, time.Time) {
	t.Helper()
	gdb, err := db.Open(db.Options{DSN: "file:edge_delete_" + t.Name() + "?mode=memory&cache=shared"})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(gdb,
		&instpersist.EdgeRow{},
		&taskpersist.TaskRow{},
		&sesspersist.SessionRow{}); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	now := time.Now().UTC()
	n := &captureNotify{}
	return gdb, edgeadmin.NewService(gdb, n), n, ctx, now
}

func seedEdge(t *testing.T, gdb *gorm.DB, id sharedkernel.EdgeID, now time.Time) {
	t.Helper()
	repo := instpersist.NewEdgeRepository(gdb)
	if err := repo.Upsert(context.Background(), &edge.Record{
		ID:           id,
		Name:         string(id),
		Enabled:      true,
		Capabilities: []string{"comfy"},
		CreatedAt:    now,
		UpdatedAt:    now,
	}); err != nil {
		t.Fatal(err)
	}
}

func TestDeleteEdge_AckRequired(t *testing.T) {
	gdb, svc, _, ctx, now := newEdgeTest(t)
	seedEdge(t, gdb, "gpu-1", now)
	tasks := taskpersist.NewTaskRepository(gdb)
	running := runtimedomain.NewPending("t1", "s1", 1, "inputs/t1", now)
	_ = running.MarkQueued("gpu-1", now)
	_ = running.MarkRunning("p", now)
	_ = tasks.Create(ctx, running)

	_, err := svc.DeleteEdge(ctx, "gpu-1", false)
	if !errors.Is(err, edgeadmin.ErrNeedsAck) {
		t.Fatalf("err=%v want ErrNeedsAck", err)
	}
	repo := instpersist.NewEdgeRepository(gdb)
	if _, err := repo.Get(ctx, "gpu-1"); err != nil {
		t.Fatalf("edge must survive rollback: %v", err)
	}
}

func TestDeleteEdge_CleanupAndDelete(t *testing.T) {
	gdb, svc, n, ctx, now := newEdgeTest(t)
	seedEdge(t, gdb, "gpu-1", now)
	tasks := taskpersist.NewTaskRepository(gdb)
	sessions := sesspersist.NewSessionRepository(gdb)
	_ = sessions.Save(ctx, convdomain.NewCollecting("s1", "tg:9", 1, []string{"a"}, now))
	running := runtimedomain.NewPending("t1", "s1", 1, "inputs/t1", now)
	_ = running.MarkQueued("gpu-1", now)
	_ = running.MarkRunning("p", now)
	_ = tasks.Create(ctx, running)
	queued := runtimedomain.NewPending("t2", "s2", 1, "inputs/t2", now)
	_ = tasks.Create(ctx, queued)

	summary, err := svc.DeleteEdge(ctx, "gpu-1", true)
	if err != nil {
		t.Fatal(err)
	}
	if summary.FailedTasks != 1 {
		t.Fatalf("summary=%+v", summary)
	}
	repo := instpersist.NewEdgeRepository(gdb)
	if _, err := repo.Get(ctx, "gpu-1"); !errors.Is(err, edge.ErrNotFound) {
		t.Fatalf("edge delete err=%v", err)
	}
	t1, _ := tasks.Get(ctx, "t1")
	if t1.Status != sharedkernel.TaskFailed || t1.ErrorCode != sharedkernel.TaskErrorEdgeDeleted {
		t.Fatalf("running task=%+v", t1)
	}
	t2, _ := tasks.Get(ctx, "t2")
	if t2.Status != sharedkernel.TaskPending {
		t.Fatalf("pending task must be untouched: %+v", t2)
	}
	if len(n.items) != 1 || n.items[0].Kind != "task_failed" || n.items[0].ChatID != "tg:9" {
		t.Fatalf("notifies=%+v", n.items)
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `go test ./internal/edgeadmin/ -v`
Expected: FAIL（包不存在）。

- [ ] **Step 3: 实现 edge_delete.go**

```go
// Package edgeadmin implements admin compute-node lifecycle operations that
// span edge/task/session repositories in a single transaction.
package edgeadmin

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"gorm.io/gorm"

	sesspersist "github.com/mr9esx/comfyui_tgbot/internal/conversation/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/edge"
	instpersist "github.com/mr9esx/comfyui_tgbot/internal/platform/edge/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/notify"
	runtimedomain "github.com/mr9esx/comfyui_tgbot/internal/runtime/domain"
	taskpersist "github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

// ErrNeedsAck is returned when the edge has running tasks and the caller did
// not confirm marking them failed.
var ErrNeedsAck = errors.New("edge delete needs ack for running tasks")

// DeleteSummary reports what the cleanup did.
type DeleteSummary struct {
	FailedTasks int `json:"failed_tasks"`
}

// Service deletes a compute node with task cleanup in one DB transaction.
type Service struct {
	db     *gorm.DB
	notify notify.Publisher
	now    func() time.Time
}

// NewService constructs a delete service over the shared database handle.
func NewService(db *gorm.DB, n notify.Publisher) *Service {
	return &Service{db: db, notify: n, now: func() time.Time { return time.Now().UTC() }}
}

// DeleteEdge fails the edge's running tasks (edge_deleted), deletes the edge
// row, then notifies task owners best-effort after commit.
func (s *Service) DeleteEdge(ctx context.Context, id sharedkernel.EdgeID, ack bool) (DeleteSummary, error) {
	var summary DeleteSummary
	var taskNotifies []sharedkernel.UserNotify

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		edgeRepo := instpersist.NewEdgeRepository(tx)
		taskRepo := taskpersist.NewTaskRepository(tx)
		sessRepo := sesspersist.NewSessionRepository(tx)

		if _, err := edgeRepo.Get(ctx, id); err != nil {
			return err
		}
		running, err := taskRepo.List(ctx, runtimedomain.AdminListQuery{EdgeID: id, Status: sharedkernel.TaskRunning})
		if err != nil {
			return err
		}
		if len(running) > 0 && !ack {
			return ErrNeedsAck
		}

		now := s.now()
		for _, t := range running {
			if err := t.MarkFailed(sharedkernel.TaskErrorEdgeDeleted, sharedkernel.EdgeDeletedMessage, now); err != nil {
				continue // already terminal (e.g. success raced in before delete)
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
		return edgeRepo.Delete(ctx, id)
	})
	if err != nil {
		return DeleteSummary{}, err
	}

	for _, n := range taskNotifies {
		if n.ChatID == "" {
			continue
		}
		if err := s.notify.Publish(ctx, n); err != nil {
			slog.Warn("edge delete: task notify failed", "task", n.TaskID, "err", err)
		}
	}
	return summary, nil
}
```

- [ ] **Step 4: 重跑测试**

Run: `go test ./internal/edgeadmin/ -v`
Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add internal/edgeadmin/edge_delete.go internal/edgeadmin/edge_delete_test.go
git commit -m "feat(edgeadmin): delete edge with running-task failure and notify"
```

---

### Task 7: edges handler + main.go 接线

**Files:**
- Modify: `internal/httpapi/edges/handler.go`
- Modify: `internal/httpapi/edges/handler_test.go`
- Modify: `apps/pixoma/cmd/pixoma/main.go`

**Interfaces:**
- Consumes: `edgeadmin.Service.DeleteEdge`、`edgeadmin.ErrNeedsAck`、`DeleteSummary`（Task 6）
- Produces: `DELETE /api/v1/edges/{id}` 新契约

- [ ] **Step 1: 失败测试**

`handler_test.go` 追加（复用现有 repo 构造）：

```go
func TestHandler_DeleteCleanup(t *testing.T) {
	ctx := context.Background()
	h := &Handler{
		DeleteWithCleanup: func(ctx context.Context, id sharedkernel.EdgeID, ack bool) (edgeadmin.DeleteSummary, error) {
			if id == "gpu-x" && !ack {
				return edgeadmin.DeleteSummary{}, edgeadmin.ErrNeedsAck
			}
			if id == "missing" {
				return edgeadmin.DeleteSummary{}, edge.ErrNotFound
			}
			return edgeadmin.DeleteSummary{FailedTasks: 1}, nil
		},
	}
	r := chi.NewRouter()
	r.Route("/api/v1/edges", func(r chi.Router) { h.Mount(r) })
	srv := httptest.NewServer(r)
	defer srv.Close()

	do := func(id string, body string) *http.Response {
		t.Helper()
		var rdr io.Reader
		if body != "" {
			rdr = bytes.NewBufferString(body)
		}
		req, err := http.NewRequest(http.MethodDelete, srv.URL+"/api/v1/edges/"+id, rdr)
		if err != nil {
			t.Fatal(err)
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		return res
	}

	res := do("missing", "")
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("missing status=%d want 404", res.StatusCode)
	}
	res.Body.Close()

	res = do("gpu-x", "")
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("no-ack status=%d want 409", res.StatusCode)
	}
	res.Body.Close()

	res = do("gpu-x", `{"ack_references":true}`)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("ack status=%d", res.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if body["deleted"] != true || body["failed_tasks"] != float64(1) {
		t.Fatalf("body=%+v", body)
	}
}
```

（`handler_test.go` 补 `edgeadmin`、`io`、`bytes` import；确认已有 `edge`/`sharedkernel`/`chi`/`httptest`。）

- [ ] **Step 2: 运行确认失败**

Run: `go test ./internal/httpapi/edges/ -run TestHandler_DeleteCleanup -v`
Expected: FAIL（`DeleteWithCleanup` 字段不存在）。

- [ ] **Step 3: 实现 handler.go**

`Handler` 加字段：

```go
	// DeleteWithCleanup performs the cleanup delete (see internal/edgeadmin).
	DeleteWithCleanup func(ctx context.Context, id sharedkernel.EdgeID, ack bool) (edgeadmin.DeleteSummary, error)
```

import 加 `"github.com/mr9esx/comfyui_tgbot/internal/edgeadmin"`。

`delete` 方法替换为：

```go
func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id := sharedkernel.EdgeID(chi.URLParam(r, "id"))
	var body struct {
		AckReferences bool `json:"ack_references"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if h.DeleteWithCleanup == nil {
		writeErr(w, http.StatusInternalServerError, "delete cleanup not configured")
		return
	}
	summary, err := h.DeleteWithCleanup(r.Context(), id, body.AckReferences)
	if errors.Is(err, edgeadmin.ErrNeedsAck) {
		writeErrCode(w, http.StatusConflict, "edge_delete_needs_ack",
			"edge has running tasks; confirm with ack_references to mark them failed")
		return
	}
	if errors.Is(err, edge.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "instance not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if h.Presence != nil {
		h.Presence.Remove(id)
	}
	if err := h.refreshPool(r); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"deleted":      true,
		"failed_tasks": summary.FailedTasks,
	})
}
```

`writeErr` 旁加 `writeErrCode`（同 cases handler 的实现）。

`main.go`：`botRT` 之后加 `edgeDeleteSvc := edgeadmin.NewService(gdb, botRT.Notify)`；`Instances` handler 加 `DeleteWithCleanup: edgeDeleteSvc.DeleteEdge`；加 import `"github.com/mr9esx/comfyui_tgbot/internal/edgeadmin"`。

- [ ] **Step 4: 重跑测试**

Run: `go test ./internal/httpapi/edges/... ./internal/edgeadmin/...`
Expected: PASS。

Run: `go build ./...`
Expected: PASS。

- [ ] **Step 5: Commit（main.go 用 git add -p）**

```bash
git add internal/httpapi/edges/handler.go internal/httpapi/edges/handler_test.go
git add -p apps/pixoma/cmd/pixoma/main.go
git commit -m "feat(api): cleanup-based edge delete with ack_references"
```

---

### Task 8: 前端 API（deleteEdge ack、sessions/tasks channel_id）

**Files:**
- Modify: `web/admin/src/lib/api/edges.ts`
- Modify: `web/admin/src/lib/api/sessions.ts`
- Modify: `web/admin/src/lib/api/tasks.ts`
- Modify: `web/admin/src/lib/api/edges.test.ts`（若有）
- Modify: `web/admin/src/lib/api/sessions.test.ts`
- Modify: `web/admin/src/lib/api/tasks.test.ts`

**Interfaces:**
- Consumes: `apiFetch`
- Produces: `deleteEdge(id, ack?)`、`listSessions({channel_id})`、`listTasks({channel_id})` —— Task 10/11 依赖

- [ ] **Step 1: 失败测试**

`sessions.test.ts` 补：

```ts
  it('listSessions filters by channel_id', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify([{ id: 's1' }]), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
    vi.stubGlobal('fetch', fetchMock)

    const data = await listSessions({ channel_id: 'tg', status: 'collecting' })

    expect(data[0].id).toBe('s1')
    expect(fetchMock).toHaveBeenCalledWith(
      'http://127.0.0.1:8081/api/v1/sessions?channel_id=tg&status=collecting',
      expect.anything(),
    )
  })
```

`tasks.test.ts` 补：

```ts
  it('listTasks filters by channel_id', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify([{ id: 't1', status: 'running' }]), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
    vi.stubGlobal('fetch', fetchMock)

    const data = await listTasks({ channel_id: 'tg', status: 'running' })

    expect(data[0].id).toBe('t1')
    expect(fetchMock).toHaveBeenCalledWith(
      'http://127.0.0.1:8081/api/v1/tasks?channel_id=tg&status=running',
      expect.anything(),
    )
  })
```

`edges.test.ts` 追加：

```ts
  it('deleteEdge sends ack_references body when ack is true', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ deleted: true, failed_tasks: 1 }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
    vi.stubGlobal('fetch', fetchMock)

    const data = await deleteEdge('gpu-1', true)

    expect(data.failed_tasks).toBe(1)
    expect(fetchMock).toHaveBeenCalledWith(
      'http://127.0.0.1:8081/api/v1/edges/gpu-1',
      expect.objectContaining({
        method: 'DELETE',
        body: JSON.stringify({ ack_references: true }),
      }),
    )
  })
```

（`edges.test.ts` import 补 `deleteEdge`。）

- [ ] **Step 2: 运行确认失败**

Run: `pnpm vitest run src/lib/api/sessions.test.ts src/lib/api/tasks.test.ts`
Expected: FAIL（URL 不匹配）。

- [ ] **Step 3: 实现**

`edges.ts`：

```ts
export type DeleteEdgeResult = {
  deleted: boolean
  failed_tasks?: number
}

export function deleteEdge(id: string, ack?: boolean) {
  return apiFetch<DeleteEdgeResult>(`/api/v1/edges/${encodeURIComponent(id)}`, {
    method: 'DELETE',
    ...(ack ? { body: JSON.stringify({ ack_references: true }) } : {}),
  })
}
```

`sessions.ts` / `tasks.ts` 的 params 各加 `channel_id?: string`。

- [ ] **Step 4: 重跑测试**

Run: `pnpm vitest run src/lib/api/sessions.test.ts src/lib/api/tasks.test.ts src/lib/api/edges.test.ts`
Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add web/admin/src/lib/api/edges.ts web/admin/src/lib/api/sessions.ts web/admin/src/lib/api/tasks.ts web/admin/src/lib/api/sessions.test.ts web/admin/src/lib/api/tasks.test.ts
git commit -m "feat(admin): delete edge with ack; filter task/session lists by channel"
```

---

### Task 9: i18n 文案 + locale 成对

**Files:**
- Modify: `web/admin/src/lib/i18n/locales/zh.json`
- Modify: `web/admin/src/lib/i18n/locales/en.json`
- Modify: `web/admin/src/lib/i18n/locale.test.ts`

**Interfaces:**
- Produces: 删除弹窗文案 keys —— Task 10/11 依赖

- [ ] **Step 1: 改 locale.test.ts（先失败）**

新增成对列表与用例：

```ts
const EDGE_CHANNEL_DELETE_I18N_KEYS = [
  'edges.deleteWillFailRunning',
  'edges.deleteSubscribedTopics',
  'edges.deleteAckRunning',
  'edges.deleteNoRunning',
  'edges.deleteDone',
  'edges.deleteFailed',
  'channels.deleteWillEndSessions',
  'channels.deleteInFlightTasks',
  'channels.deleteAckImpact',
] as const

describe('edge/channel delete i18n', () => {
  it('defines delete keys in zh and en locales', () => {
    for (const key of EDGE_CHANNEL_DELETE_I18N_KEYS) {
      expect(lookup(zh, key), `zh missing ${key}`).toEqual(expect.any(String))
      expect(lookup(en, key), `en missing ${key}`).toEqual(expect.any(String))
    }
  })
})
```

- [ ] **Step 2: 运行确认失败**

Run: `pnpm vitest run src/lib/i18n/locale.test.ts`
Expected: FAIL（keys 缺失）。

- [ ] **Step 3: i18n keys**

`zh.json`：

- `edges` 段删除 `deleteConfirmDesc`（Task 10 移除 EdgeForm 删除后不再使用；`deleteConfirmTitle` 保留给新弹窗），新增：

```json
    "deleteWillFailRunning": "{{count}} 个执行中任务将标记为失败并通知用户",
    "deleteSubscribedTopics": "该节点订阅的 Topics",
    "deleteAckRunning": "我已知悉这些任务将标记为失败",
    "deleteNoRunning": "当前没有执行中的任务",
    "deleteDone": "节点已删除，{{count}} 个任务已标记失败",
    "deleteFailed": "删除失败"
```

- `channels` 段新增：

```json
    "deleteWillEndSessions": "{{count}} 个进行中会话将被终止并通知用户",
    "deleteInFlightTasks": "{{count}} 个执行中/排队任务将继续运行，但完成后通知无法送达",
    "deleteAckImpact": "我已知悉上述影响"
```

`channels.deleteConfirmBody` 改为：

```json
    "deleteConfirmBody": "确定删除渠道「{{name}}」？该操作不可恢复。进行中会话将被终止，菜单/卡片将一并移除。"
```

`en.json` 对应英文：

```json
    "deleteWillFailRunning": "{{count}} running task(s) will be marked failed and the users notified",
    "deleteSubscribedTopics": "Topics this node subscribes to",
    "deleteAckRunning": "I understand these tasks will be marked failed",
    "deleteWillEndSessions": "{{count}} active session(s) will be terminated and the users notified",
    "deleteInFlightTasks": "{{count}} running/queued task(s) will continue, but completion notifications cannot be delivered",
    "deleteAckImpact": "I understand the impact above",
    "deleteConfirmBody": "Delete channel “{{name}}”? This cannot be undone. Active sessions will be terminated and menus/cards removed."
```

`en.json` 对应补 `edges.deleteNoRunning/deleteDone/deleteFailed`：

```json
    "deleteNoRunning": "No running tasks",
    "deleteDone": "Node deleted; {{count}} task(s) marked failed",
    "deleteFailed": "Delete failed"
```

删除 en 的 `edges.deleteConfirmDesc`（保留 `deleteConfirmTitle`）。

- [ ] **Step 4: 重跑测试**

Run: `pnpm vitest run src/lib/i18n/locale.test.ts`
Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add web/admin/src/lib/i18n/locales/zh.json web/admin/src/lib/i18n/locales/en.json web/admin/src/lib/i18n/locale.test.ts
git commit -m "feat(admin): edge/channel delete impact i18n"
```

---

### Task 10: 节点详情页删除弹窗 + 移除 EdgeForm 删除

**Files:**
- Modify: `web/admin/src/features/edges/detail-panel.tsx`
- Modify: `web/admin/src/features/edges/edge-form.tsx`
- Modify: `web/admin/src/features/edges/edges.contract.test.ts`

**Interfaces:**
- Consumes: `listEdgeTasks`、`deleteEdge(id, ack)`（Task 8）、i18n keys（Task 9）
- Produces: 详情页删除入口 + 影响弹窗

- [ ] **Step 1: 先改 contract 测试（失败）**

`edges.contract.test.ts` 追加用例：

```ts
  it('detail panel offers delete with running-task impact', () => {
    const detail = read('detail-panel.tsx')
    const form = read('edge-form.tsx')
    expect(detail).toContain('edges.deleteWillFailRunning')
    expect(detail).toContain('edges.deleteSubscribedTopics')
    expect(detail).toContain('edges.deleteAckRunning')
    expect(detail).toContain('deleteEdge(id, ackRunning)')
    expect(form).not.toContain('deleteEdge(')
  })
```

- [ ] **Step 2: 运行确认失败**

Run: `pnpm vitest run src/features/edges/edges.contract.test.ts`
Expected: FAIL。

- [ ] **Step 3: 实现 detail-panel.tsx**

import 补 `DialogFooter`（`@/components/ui/dialog`）与 `listEdgeTasks`、`deleteEdge`。

组件内加 state 与查询：

```tsx
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [ackRunning, setAckRunning] = useState(false)
  const runningTasksQuery = useQuery({
    queryKey: ['edges', id, 'running-tasks'] as const,
    queryFn: () => listEdgeTasks(id, { status: 'running' }),
    enabled: deleteOpen,
  })
  const runningCount = runningTasksQuery.data?.length ?? 0

  const deleteMutation = useMutation({
    mutationFn: () => deleteEdge(id, ackRunning),
    onSuccess: async (summary) => {
      toast.success(
        summary.failed_tasks
          ? t('edges.deleteDone', { count: summary.failed_tasks })
          : t('edges.deleteSuccess')
      )
      await queryClient.invalidateQueries({ queryKey: queryKeys.edges.all })
      queryClient.removeQueries({ queryKey: queryKeys.edges.detail(id) })
      void navigate({ to: '/edges' })
    },
    onError: (err) => {
      toast.error(errorMessage(err) ?? t('edges.deleteFailed'))
    },
  })
```

（`edges.deleteDone`、`edges.deleteFailed` 若不存在，在 Task 9 的 zh/en 一并补上；`errorMessage` 若 detail-panel 已有则复用。）

详情页头部（编辑按钮旁）加删除按钮：

```tsx
            <Button
              type='button'
              variant='outline'
              className='h-8 gap-1.5 px-3 text-xs text-destructive hover:text-destructive'
              onClick={() => setDeleteOpen(true)}
            >
              <Trash2 className='size-3.5' />
              {t('common.delete')}
            </Button>
```

（`Trash2` 已 import 则复用。）

组件尾部（`deployOpen` Dialog 附近）加删除弹窗：

```tsx
      <Dialog open={deleteOpen} onOpenChange={setDeleteOpen}>
        <DialogContent className='sm:max-w-lg'>
          <DialogHeader>
            <DialogTitle>{t('edges.deleteConfirmTitle')}</DialogTitle>
          </DialogHeader>
          <div className='space-y-3 text-sm'>
            {runningCount > 0 ? (
              <p>{t('edges.deleteWillFailRunning', { count: runningCount })}</p>
            ) : (
              <p>{t('edges.deleteNoRunning')}</p>
            )}
            {(edge.subscribe_topics ?? []).length > 0 ? (
              <div className='space-y-1'>
                <p className='font-medium'>{t('edges.deleteSubscribedTopics')}</p>
                <ul className='max-h-32 overflow-auto rounded-md border bg-muted/20 p-3 text-xs'>
                  {edge.subscribe_topics.map((tp) => (
                    <li key={tp}>{tp}</li>
                  ))}
                </ul>
              </div>
            ) : null}
            {runningCount > 0 ? (
              <label className='flex items-start gap-2'>
                <input
                  type='checkbox'
                  checked={ackRunning}
                  onChange={(e) => setAckRunning(e.target.checked)}
                  data-testid='edge-delete-ack'
                />
                <span>{t('edges.deleteAckRunning')}</span>
              </label>
            ) : null}
          </div>
          <DialogFooter>
            <Button type='button' variant='outline' onClick={() => setDeleteOpen(false)}>
              {t('common.cancel')}
            </Button>
            <Button
              type='button'
              variant='destructive'
              disabled={deleteMutation.isPending || (runningCount > 0 && !ackRunning)}
              onClick={() => deleteMutation.mutate()}
            >
              {t('common.delete')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
```

`edge-form.tsx`：删除 `deleteMutation`、`ConfirmDialog`、`DialogFooter` 中的删除按钮、`deleteEdge` 与 `ConfirmDialog` import、`confirmOpen` state。

（`edges.deleteConfirmTitle` 在 Task 9 保留给新弹窗使用；`edges.deleteNoRunning`、`edges.deleteDone`、`edges.deleteFailed` 在 Task 9 补 zh/en。）

- [ ] **Step 4: 重跑测试**

Run: `pnpm vitest run src/features/edges/edges.contract.test.ts`
Expected: PASS。

Run: `pnpm exec tsc -b`
Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add web/admin/src/features/edges/detail-panel.tsx web/admin/src/features/edges/edge-form.tsx web/admin/src/features/edges/edges.contract.test.ts web/admin/src/lib/i18n/locales/zh.json web/admin/src/lib/i18n/locales/en.json
git commit -m "feat(admin): edge detail delete dialog with running-task impact"
```

---

### Task 11: 渠道详情页删除弹窗

**Files:**
- Modify: `web/admin/src/features/channels/channel-detail-panel.tsx`
- Modify: `web/admin/src/features/channels/channel-layout.contract.test.ts`

**Interfaces:**
- Consumes: `listSessions({channel_id})`、`listTasks({channel_id})`（Task 8）、i18n keys（Task 9）
- Produces: 渠道删除影响弹窗

- [ ] **Step 1: 先改 contract 测试（失败）**

`channel-layout.contract.test.ts` 追加用例：

```ts
  it('delete is available while enabled and shows impact', () => {
    const source = readFileSync(
      join(here, 'channel-detail-panel.tsx'),
      'utf8'
    )
    expect(source).toContain('disabled={deleteMutation.isPending')
    expect(source).not.toContain('ch.enabled || deleteMutation.isPending')
    expect(source).toContain('channels.deleteWillEndSessions')
    expect(source).toContain('channels.deleteInFlightTasks')
    expect(source).toContain('channels.deleteAckImpact')
  })
```

（按该文件现有 `here`/`readFileSync` 模式调整。）

- [ ] **Step 2: 运行确认失败**

Run: `pnpm vitest run src/features/channels/channel-layout.contract.test.ts`
Expected: FAIL。

- [ ] **Step 3: 实现 channel-detail-panel.tsx**

- 删除按钮 `disabled={ch.enabled || deleteMutation.isPending}` 改为 `disabled={deleteMutation.isPending}`。
- 加 state 与查询：

```tsx
  const [ackImpact, setAckImpact] = useState(false)
  const sessionsQuery = useQuery({
    queryKey: ['channels', id, 'active-sessions'] as const,
    queryFn: async () => {
      const [collecting, confirming] = await Promise.all([
        listSessions({ channel_id: id, status: 'collecting' }),
        listSessions({ channel_id: id, status: 'confirming' }),
      ])
      return collecting.length + confirming.length
    },
  })
  const inFlightTasksQuery = useQuery({
    queryKey: ['channels', id, 'in-flight-tasks'] as const,
    queryFn: async () => {
      const [pending, queued, running] = await Promise.all([
        listTasks({ channel_id: id, status: 'pending' }),
        listTasks({ channel_id: id, status: 'queued' }),
        listTasks({ channel_id: id, status: 'running' }),
      ])
      return pending.length + queued.length + running.length
    },
  })
```

- 弹窗 `AlertDialogDescription` 内追加影响区与勾选：

```tsx
                  <AlertDialogDescription>
                    {t('channels.deleteConfirmBody', { name: ch.name })}
                    {(sessionsQuery.data ?? 0) > 0 ? (
                      <p className='mt-3'>
                        {t('channels.deleteWillEndSessions', {
                          count: sessionsQuery.data ?? 0,
                        })}
                      </p>
                    ) : null}
                    {(inFlightTasksQuery.data ?? 0) > 0 ? (
                      <p className='mt-1'>
                        {t('channels.deleteInFlightTasks', {
                          count: inFlightTasksQuery.data ?? 0,
                        })}
                      </p>
                    ) : null}
                    {(sessionsQuery.data ?? 0) > 0 ||
                    (inFlightTasksQuery.data ?? 0) > 0 ? (
                      <label className='mt-4 flex items-start gap-2'>
                        <input
                          type='checkbox'
                          checked={ackImpact}
                          onChange={(e) => setAckImpact(e.target.checked)}
                          data-testid='channel-delete-ack'
                        />
                        <span>{t('channels.deleteAckImpact')}</span>
                      </label>
                    ) : null}
                  </AlertDialogDescription>
```

- `AlertDialogAction` 的 `disabled` 改为 `(sessionsQuery.data ?? 0) > 0 || (inFlightTasksQuery.data ?? 0) > 0 ? !ackImpact : false`。
- import 补 `listSessions`、`listTasks`；`channels.deleteConfirmTitle` 保留。

- [ ] **Step 4: 重跑测试**

Run: `pnpm vitest run src/features/channels/channel-layout.contract.test.ts`
Expected: PASS。

Run: `pnpm exec tsc -b`
Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add web/admin/src/features/channels/channel-detail-panel.tsx web/admin/src/features/channels/channel-layout.contract.test.ts
git commit -m "feat(admin): channel delete dialog shows sessions/tasks impact"
```

---

### Task 12: 全量验证

**Files:** 无代码改动

- [ ] **Step 1: 后端全量测试 + vet**

Run: `go test ./... && go vet ./...`
Expected: PASS。

- [ ] **Step 2: 前端全量测试 + 类型 + lint**

Run: `pnpm vitest run`
Expected: PASS。

Run: `pnpm exec tsc -b`
Expected: PASS。

Run: `pnpm exec eslint src/features/edges src/features/channels src/lib/api`
Expected: PASS。

- [ ] **Step 3: 手工冒烟（可选）**

- 删除一个在线且有 running 任务的节点：弹窗显示任务数，勾选后删除成功，任务变 failed（`edge_deleted`），用户收到失败通知。
- 启用中的渠道直接删除：弹窗显示会话/任务数，勾选后删除成功，会话用户收到「该渠道已被管理员删除」。

- [ ] **Step 4: 收尾**

`git status --porcelain` 检查无遗留；涉及 `main.go` 的并发改动保持未提交。
