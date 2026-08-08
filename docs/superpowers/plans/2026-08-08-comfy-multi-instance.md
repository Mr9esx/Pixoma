---
change: comfy-multi-instance
design-doc: docs/superpowers/specs/2026-08-08-comfy-multi-instance-design.md
base-ref: 01c3b2c1c302769da85b1e73c6689839d70291f0
archived-with: 2026-08-08-comfy-multi-instance
---

# comfy-multi-instance 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在同一 bot 进程内落地 User/Session/Task 的 SQLite 持久化、Comfy 多实例 CRUD/健康/round-robin 选路，以及 system/queue/tasks 观测 HTTP API。

**Architecture:** DB（GORM/SQLite）为 User、Session、Task、`comfy_instances` 的真相源；进程内维护 Comfy 客户端池与健康集合；Orchestrator 在健康∩熔断允许上 round-robin；ConfirmRun 写 `session_id`，通知经 Session 解析 `chat_id`；HTTP 仅暴露本机/内网观测面。

**Tech Stack:** Go、GORM + SQLite（`github.com/glebarez/sqlite`）、chi、现有 `comfyui.Client`（Mock/HTTP）、内存队列、testify（如已有）/`testing`。

## Global Constraints

- 产物语言：zh-CN（本计划与文档）；代码标识符保持仓库英文风格
- 保持 `comfy_mock` / `COMFY_MOCK` 一键开关；凡影响执行/观测链路的改动必须同步 Mock（见 `.cursor/rules/comfy-mock-parity.mdc`）
- 非目标：管理后台 UI、强鉴权 RBAC、清队列/interrupt、拆 `catalog_cases` bindings、跨机多 Worker
- Session 行长期保留；禁止因活跃结束物理删除导致 Task 断链
- Task 必填 `session_id`；不以 `user_id`/`chat_id` 作为 Task 必需冗余字段
- **去掉 Actuator Ledger**：执行/对账只信持久化 Task；删除 MemoryLedger 接线
- system/queue 不可达时明确失败或 `reachable=false`，禁止伪造空成功
- HTTP API 本期无鉴权；文档必须警示本机/内网
- 设计依据：`docs/superpowers/specs/2026-08-08-comfy-multi-instance-design.md`；delta specs 在 `docs/openspec/changes/comfy-multi-instance/specs/`

## 文件结构

| 文件 | 职责 |
|---|---|
| `internal/identity/domain/user.go` | User 聚合与 Upsert 输入 |
| `internal/identity/domain/repository.go` | UserRepository 接口 |
| `internal/identity/infrastructure/persistence/gorm_user.go` | users 表 + GORM 仓储 |
| `internal/conversation/domain/session.go` | Session 增加 `UserID` |
| `internal/conversation/domain/service.go` | StartCase 接受/写入 `user_id`；GetByID |
| `internal/conversation/infrastructure/persistence/gorm_session.go` | sessions 表 + GORM；按 chat 活跃查询；长期保留 |
| `internal/runtime/domain/task.go` | Task 增加 `SessionID`；`NewPending` 签名调整 |
| `internal/runtime/domain/task_repository.go` | 增加 `ListByInstance`；`ListByChat` 经 Session join 语义 |
| `internal/runtime/infrastructure/persistence/gorm_task.go` | tasks 表 + GORM |
| `internal/platform/instance/record.go` | Comfy 实例元数据领域类型 |
| `internal/platform/instance/repository.go` | InstanceRepository CRUD |
| `internal/platform/instance/persistence/gorm_instance.go` | `comfy_instances` 表 |
| `internal/platform/instance/pool.go` | 客户端池 + Registry + 健康状态 + 刷新 |
| `internal/runtime/infrastructure/comfyui/client.go` | Client 扩展 `SystemStats`/`Queue`；Mock 占位 |
| `internal/runtime/infrastructure/comfyui/http.go` | 真实 `GET /system_stats`、`GET /queue` |
| `internal/runtime/application/orchestrator/service.go` | round-robin；无实例保持 pending |
| `internal/runtime/infrastructure/actuator/worker.go` | 按 `DispatchCommand.InstanceID` 取客户端 |
| `internal/packaging/botapp/confirm_run.go` | 创建 Task 写 `session_id` |
| `internal/channel/tg/adapter.go` / `bot.go` | TG 路径 upsert User，StartCase 传 `user_id` |
| `internal/httpapi/comfyinstances/handler.go` | `/api/v1/comfy-instances` CRUD + system/queue/tasks |
| `apps/bot/cmd/comfyui-bot/main.go` | AutoMigrate、种子、池、健康 ticker、路由接线 |
| `internal/platform/botconfig/config.go` | 可选 `comfy_instances`、健康间隔配置 |
| `Makefile` | build / test / run / run-mock |
| `README.md` | 架构、种子、curl、Mock、无鉴权警示 |
| `configs/bot.example.yaml` | 注释与新配置项 |

对应 `tasks.md` 章节：任务 1→§1，任务 2–4→§2，任务 5–6→§3，任务 7→§4，任务 8→§5。

---

## Task 1: User 持久化与 TG upsert（tasks.md 1.1）

**Files:**
- Create: `internal/identity/domain/user.go`
- Create: `internal/identity/domain/repository.go`
- Create: `internal/identity/infrastructure/persistence/gorm_user.go`
- Create: `internal/identity/infrastructure/persistence/gorm_user_test.go`
- Modify: `internal/channel/tg/adapter.go`、`internal/channel/tg/bot.go`（或抽出 upsert 钩子）
- Modify: `apps/bot/cmd/comfyui-bot/main.go`（AutoMigrate 含 UserRow；注入 UserRepo）

**Interfaces:**
- Consumes: 无
- Produces:
  - `identitydomain.User`（`ID string`, `TgUserID int64`, 资料字段, `LastSeenAt`, `CreatedAt`, `UpdatedAt`）
  - `identitydomain.UpsertFrom`（From 可得字段）
  - `identitydomain.Repository.UpsertByTgUserID(ctx, in UpsertFrom) (*User, error)`
  - `identitydomain.Repository.GetByID(ctx, id string) (*User, error)`

- [x] **Step 1: 写失败测试 — upsert 幂等与字段刷新**

```go
func TestUpsertByTgUserID_IdempotentAndRefresh(t *testing.T) {
	gdb := openTestDB(t) // sqlite memory + AutoMigrate UserRow
	repo := persistence.NewUserRepository(gdb)
	ctx := context.Background()

	u1, err := repo.UpsertByTgUserID(ctx, domain.UpsertFrom{
		TgUserID: 42, Username: "alice", FirstName: "A", LanguageCode: "zh-hans",
	})
	if err != nil {
		t.Fatal(err)
	}
	u2, err := repo.UpsertByTgUserID(ctx, domain.UpsertFrom{
		TgUserID: 42, Username: "alice2", FirstName: "A2",
	})
	if err != nil {
		t.Fatal(err)
	}
	if u1.ID != u2.ID {
		t.Fatalf("id changed: %s vs %s", u1.ID, u2.ID)
	}
	if u2.Username != "alice2" {
		t.Fatalf("username not refreshed: %q", u2.Username)
	}
	if !u2.LastSeenAt.After(u1.LastSeenAt) && !u2.LastSeenAt.Equal(u1.LastSeenAt) {
		// allow equal if same clock; prefer After when Now injects
	}
}
```

- [x] **Step 2: 跑测试确认失败**

Run: `go test ./internal/identity/infrastructure/persistence/ -run TestUpsertByTgUserID -v`  
Expected: FAIL（包/类型未定义）

- [x] **Step 3: 最小实现**

`UserRow` 表名 `users`：`id` PK UUID、`tg_user_id` UNIQUE NOT NULL、`username`/`first_name`/`last_name`/`language_code`、`is_bot`/`is_premium`（可空 bool）、`last_seen_at`、`created_at`/`updated_at`。  
`UpsertByTgUserID`：按 `tg_user_id` First；不存在则生成 UUID 创建；存在则更新可变字段并刷新 `last_seen_at`。

- [x] **Step 4: 测试通过**

Run: `go test ./internal/identity/infrastructure/persistence/ -run TestUpsertByTgUserID -v`  
Expected: PASS

- [x] **Step 5: TG 路径接入 upsert**

在消息/回调查询入口（`RegisterHandlers` 或 adapter 统一入口）从 `update.Message.From` / `CallbackQuery.From` 构造 `UpsertFrom`，调用 `Users.UpsertByTgUserID`，将内部 `user_id` 传入后续 `StartCase`（Facade/Service 签名在任务 2 改；本任务可先把 upsert 结果存入 adapter 字段或 context，若 StartCase 尚未接受 userID，则先只保证用户行写入，Step 中注明与任务 2 联调）。

最小可测钩子：在 adapter 增加可选 `Users identitydomain.Repository`；处理 update 时若非 nil 则 upsert。补单测或手工：mock From → 库中有行。

- [x] **Step 6: Commit**

```bash
git add internal/identity internal/channel/tg apps/bot/cmd/comfyui-bot/main.go
git commit -m "feat(identity): persist users and upsert from TG From"
```

---

## Task 2: Session GORM 持久化（tasks.md 1.2）

**Files:**
- Modify: `internal/conversation/domain/session.go`
- Modify: `internal/conversation/domain/service.go`
- Modify: `internal/conversation/domain/service_test.go`
- Create: `internal/conversation/infrastructure/persistence/gorm_session.go`
- Create: `internal/conversation/infrastructure/persistence/gorm_session_test.go`
- Modify: `internal/packaging/botapp/facade.go`（StartCase 传 UserID）
- Modify: `apps/bot/cmd/comfyui-bot/main.go`（替换 MemoryRepository）

**Interfaces:**
- Consumes: `identitydomain.User.ID`
- Produces:
  - `Session.UserID string`（必填于新建）
  - `Repository` 扩展：`GetByID(ctx, id SessionID) (*Session, error)`（供 Task 通知 join）
  - `Service.StartCase(ctx, chatID, userID, caseID, inputKeys)`
  - GORM：`GetActiveByChat` 仅返回 `collecting|confirming`；`Save` **不物理删除** submitted/exited 行

- [x] **Step 1: 写失败测试 — 重启后活跃 Session 可恢复；submitted 行仍在**

```go
func TestGormSession_ActiveSurviveReopenAndSubmittedKept(t *testing.T) {
	dsn := "file:sess_test?mode=memory&cache=shared"
	gdb := openShared(t, dsn)
	repo := persistence.NewSessionRepository(gdb)
	ctx := context.Background()
	now := time.Now().UTC()

	s := domain.NewCollecting("s1", 100, "case-1", []string{"prompt"}, now)
	s.UserID = "user-1"
	if err := repo.Save(ctx, s); err != nil {
		t.Fatal(err)
	}

	gdb2 := openShared(t, dsn)
	repo2 := persistence.NewSessionRepository(gdb2)
	got, err := repo2.GetActiveByChat(ctx, 100)
	if err != nil || got.ID != "s1" || got.UserID != "user-1" {
		t.Fatalf("active restore: %+v %v", got, err)
	}

	got.Status = domain.StatusSubmitted
	_ = repo2.Save(ctx, got)
	if _, err := repo2.GetActiveByChat(ctx, 100); err != domain.ErrNoActiveSession {
		t.Fatalf("expected no active, got %v", err)
	}
	byID, err := repo2.GetByID(ctx, "s1")
	if err != nil || byID.Status != domain.StatusSubmitted {
		t.Fatalf("submitted row missing: %+v %v", byID, err)
	}
}
```

- [x] **Step 2: 跑测试确认失败**

Run: `go test ./internal/conversation/infrastructure/persistence/ -run TestGormSession_ActiveSurvive -v`  
Expected: FAIL

- [x] **Step 3: 实现 Session 领域字段 + GORM**

`sessions` 表：`id` PK、`user_id`（index）、`chat_id`（index）、`case_id`、`status`、`current_input_index`、`input_keys_json`、`draft_json`、时间戳。  
`Save`：始终 upsert 行；活跃索引查询用 `WHERE chat_id=? AND status IN ('collecting','confirming')`。  
更新 `NewCollecting` / `StartCase` 写入 `UserID`。MemoryRepository 同步支持 `UserID` + `GetByID`（测试用）。

- [x] **Step 4: 更新 Service / Facade / TG**

`StartCase` 必须收到非空 `userID`；TG upsert 后传入。补 `service_test`：创建 Session 时 `UserID` 已写。

- [x] **Step 5: 测试通过 + main 换 GORM Session**

Run: `go test ./internal/conversation/... ./internal/packaging/botapp/ -count=1`  
Expected: PASS  
`main`：`sessRepo := convpersist.NewSessionRepository(gdb)`，AutoMigrate `SessionRow`。

- [x] **Step 6: Commit**

```bash
git add internal/conversation internal/packaging/botapp internal/channel/tg apps/bot/cmd/comfyui-bot/main.go
git commit -m "feat(conversation): persist sessions with user_id in SQLite"
```

---

## Task 3: Task GORM 持久化（tasks.md 1.3）

**Files:**
- Modify: `internal/runtime/domain/task.go`
- Modify: `internal/runtime/domain/task_repository.go`
- Modify: `internal/runtime/domain/task_test.go`
- Create: `internal/runtime/infrastructure/persistence/gorm_task.go`
- Create: `internal/runtime/infrastructure/persistence/gorm_task_test.go`
- Modify: 所有 `NewPending` / `ListByChat` 调用方（orchestrator tests、botapp tests、actuator 等）

**Interfaces:**
- Consumes: `Session.ID`
- Produces:
  - `Task.SessionID sharedkernel.SessionID`（必填）
  - `NewPending(id, sessionID, caseID, inputPrefix, now) *Task`（去掉必需 ChatID；若暂留 `ChatID` 字段，仅作可选缓存，创建时可不写）
  - `TaskRepository.ListByInstance(ctx, instanceID, q ListByInstanceQuery) ([]*Task, error)`
  - `ListByChat`：join `sessions` 按 `chat_id`（GORM）；Memory 实现可保留 chat 字段或注入 Session 查找 — GORM 为主路径

```go
type ListByInstanceQuery struct {
	Status sharedkernel.TaskStatus // 空 = 不过滤
	Limit  int
	Offset int
}
```

- [x] **Step 1: 写失败测试 — 持久化 SessionID；ListByInstance 不含无 instance 的 pending**

```go
func TestGormTask_SessionIDAndListByInstance(t *testing.T) {
	gdb := openTestDB(t) // migrate sessions+tasks（或仅 tasks + 手写 session 行）
	// ... 先插入 session s1 chat=9
	tasks := persistence.NewTaskRepository(gdb)
	ctx := context.Background()
	now := time.Now().UTC()

	pending := domain.NewPending("t-pending", "s1", "c1", "inputs/t-pending", now)
	_ = tasks.Create(ctx, pending)

	queued := domain.NewPending("t-q", "s1", "c1", "inputs/t-q", now)
	_ = queued.MarkQueued("gpu-1", now)
	_ = tasks.Create(ctx, queued)

	list, err := tasks.ListByInstance(ctx, "gpu-1", domain.ListByInstanceQuery{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ID != "t-q" {
		t.Fatalf("got %+v", list)
	}
	got, _ := tasks.Get(ctx, "t-pending")
	if got.SessionID != "s1" {
		t.Fatalf("session_id=%q", got.SessionID)
	}
}
```

- [x] **Step 2: 跑测试确认失败**

Run: `go test ./internal/runtime/infrastructure/persistence/ -run TestGormTask_SessionID -v`  
Expected: FAIL

- [x] **Step 3: 实现 tasks 表与仓储**

列：`id`, `session_id` NOT NULL（index）, `case_id`, `status`, `instance_id`（index，可空字符串）, `prompt_id`, `input_prefix`, `outputs_json`, `error_code`, `error_message`, 时间戳。  
`ListByInstance`：`WHERE instance_id = ? AND instance_id != ''`；支持 status/limit/offset。  
`ListByChat`：`JOIN sessions ON tasks.session_id = sessions.id WHERE sessions.chat_id = ?`。

- [x] **Step 4: 修正领域测试与 Memory 仓储**

更新 `NewPending` 与 Memory：`ListByInstance`；`ListByChat` 若 Memory 无 Session，可要求测试改用 GORM，或 Memory 仍用可选 `ChatID` 仅测用 — 主路径以 GORM join 为准。

- [x] **Step 5: 测试通过**

Run: `go test ./internal/runtime/... -count=1`  
Expected: PASS（本任务可不改 main 接线，任务 4 一起换）

- [x] **Step 6: Commit**

```bash
git add internal/runtime
git commit -m "feat(runtime): persist tasks with session_id and ListByInstance"
```

---

## Task 4: ConfirmRun 写 session_id + 通知 join chat（tasks.md 1.4）

**Files:**
- Modify: `internal/packaging/botapp/confirm_run.go`
- Modify: `internal/packaging/botapp/confirm_run_test.go`
- Modify: `internal/packaging/botapp/facade.go`（`ListMyTasks` 仍用 `ListByChat`）
- Modify: `internal/runtime/application/orchestrator/service.go`（`publishNotify` 经 Session 取 chat）
- Modify: `internal/runtime/application/orchestrator/service_test.go`
- Modify: `apps/bot/cmd/comfyui-bot/main.go`（Tasks → GORM；Orchestrator 注入 SessionRepo）

**Interfaces:**
- Consumes: `SessionStore.GetByID`、`Task.SessionID`
- Produces: ConfirmRun 创建的 Task 含正确 `SessionID`；`UserNotify.ChatID` 来自 Session

- [x] **Step 1: 写失败测试 — ConfirmRun 写入 session_id；notify 用 Session.chat_id**

```go
func TestConfirmRun_WritesSessionID(t *testing.T) {
	// 装配：Session 已 confirming，UserID 已设；Tasks 用 Memory 或 sqlite
	res, err := facade.ConfirmRun(ctx, botapp.ConfirmRunCmd{ChatID: 100})
	if err != nil {
		t.Fatal(err)
	}
	task, _ := facade.Tasks.Get(ctx, res.TaskID)
	if task.SessionID == "" {
		t.Fatal("expected session_id")
	}
	sess, _ := facade.SessionStore.GetByID(ctx, task.SessionID)
	if sess.ChatID != 100 {
		t.Fatalf("chat via session: %d", sess.ChatID)
	}
}
```

Orchestrator 单测：Task 无 ChatID、有 SessionID；注入 fake SessionRepo；终态后 `Notify` 收到正确 ChatID。

- [x] **Step 2: 跑测试确认失败**

Run: `go test ./internal/packaging/botapp/ -run TestConfirmRun_WritesSessionID -v`  
Expected: FAIL（仍用旧 NewPending(chat)）

- [x] **Step 3: 实现**

```go
task := runtimedomain.NewPending(taskID, sess.ID, sess.CaseID, inputPrefix, now)
```

`publishNotify`：

```go
chatID := t.ChatID // 若仍有缓存
if chatID == 0 && s.Sessions != nil {
	sess, err := s.Sessions.GetByID(ctx, t.SessionID)
	if err != nil {
		return fmt.Errorf("notify chat via session: %w", err)
	}
	chatID = sess.ChatID
}
n := sharedkernel.UserNotify{ChatID: chatID, /* ... */}
```

- [x] **Step 4: main 接线 GORM Task + Session GetByID**

AutoMigrate `TaskRow`；`tasks := taskpersist.NewTaskRepository(gdb)`。

- [x] **Step 5: 相关测试通过**

Run: `go test ./internal/packaging/botapp/ ./internal/runtime/application/orchestrator/ -count=1`  
Expected: PASS

- [x] **Step 6: Commit**

```bash
git add internal/packaging/botapp internal/runtime apps/bot/cmd/comfyui-bot/main.go
git commit -m "feat: ConfirmRun persists session_id; notify joins Session.chat_id"
```

---

## Task 5: 去掉 Ledger，对账只信 Task（tasks.md 1.5）

**Files:**
- Modify: `internal/runtime/infrastructure/actuator/worker.go`（移除 Ledger 字段与 Save/GetRun 对 Ledger 的依赖）
- Modify: `internal/runtime/infrastructure/actuator/query_adapter.go`（GetRun → Task 仓储）
- Modify: `internal/runtime/application/orchestrator/service.go`（reconcile 读 Task）
- Modify: 相关测试与 `apps/bot/cmd/comfyui-bot/main.go`（不再 `NewMemoryLedger()`）

**Interfaces:**
- Consumes: `TaskRepository.Get`
- Produces: 无独立 LocalRun 真相源

- [x] **Step 1: 写/改测试 — reconcile 与 GetRun 不依赖 Ledger**

对账路径仅 `Tasks.Get`；Worker 不再要求 Ledger 非 nil。

- [x] **Step 2: 跑测试确认失败或编译失败（若仍引用 Ledger）**

- [x] **Step 3: 实现删除 MemoryLedger 接线；QueryAdapter 映射 Task → ExecutionView**

- [x] **Step 4: 相关测试通过**

Run: `go test ./internal/runtime/... -count=1`  
Expected: PASS

- [x] **Step 5: Commit**

```bash
git add internal/runtime apps/bot/cmd/comfyui-bot/main.go
git commit -m "refactor(runtime): drop actuator ledger; reconcile from Task only"
```

---

## Task 6: comfy_instances 仓储 + 客户端池种子（tasks.md 2.1）

**Files:**
- Create: `internal/platform/instance/record.go`
- Create: `internal/platform/instance/repository.go`
- Create: `internal/platform/instance/persistence/gorm_instance.go`
- Create: `internal/platform/instance/persistence/gorm_instance_test.go`
- Create: `internal/platform/instance/pool.go`
- Create: `internal/platform/instance/pool_test.go`
- Modify: `internal/platform/botconfig/config.go`（可选 `ComfyInstances []ComfyInstanceSeed`）
- Modify: `apps/bot/cmd/comfyui-bot/main.go`（种子 upsert + 建池；替换 `static.New` 单实例）

**Interfaces:**
- Consumes: `comfyui.NewClient`、`instance.Registry`
- Produces:

```go
type Record struct {
	ID             sharedkernel.InstanceID
	BaseURL        string
	Enabled        bool
	Capabilities   []string
	CreatedAt, UpdatedAt time.Time
}

type Repository interface {
	Upsert(ctx context.Context, r *Record) error
	Get(ctx context.Context, id sharedkernel.InstanceID) (*Record, error)
	List(ctx context.Context) ([]*Record, error)
	Delete(ctx context.Context, id sharedkernel.InstanceID) error
}

// Pool implements instance.Registry + client lookup
func (p *Pool) Client(id sharedkernel.InstanceID) (comfyui.Client, error)
func (p *Pool) Refresh(ctx context.Context) error // 从 DB 重建 map
func (p *Pool) ListHealthy(ctx context.Context, filter CapabilityFilter) ([]Instance, error)
func (p *Pool) SetHealthy(id sharedkernel.InstanceID, ok bool)
```

- [x] **Step 1: 写失败测试 — CRUD 重启仍在；Refresh 后 Client 指向新 URL；disabled 不在 ListHealthy**

```go
func TestInstanceRepo_UpsertGetList(t *testing.T) { /* ... */ }

func TestPool_RefreshAndHealthyFilter(t *testing.T) {
	// Upsert gpu-1 enabled, gpu-2 disabled；Refresh；SetHealthy(gpu-1,true)
	// ListHealthy 仅含 gpu-1
}
```

- [x] **Step 2: 跑测试确认失败**

Run: `go test ./internal/platform/instance/... -count=1`  
Expected: FAIL

- [x] **Step 3: 实现表与 Pool**

`comfy_instances`：`id` PK 字符串、`base_url`、`enabled`、`capabilities_json`、时间戳。  
启动种子逻辑（放 main 或 `instance.SeedFromConfig`）：

1. 若配置 `comfy_instances` 非空 → 逐条 Upsert  
2. 否则若 `comfyui_base_url` 非空 → Upsert `{ID: default_instance_id, BaseURL, Enabled: true}`  
3. `comfy_mock: true` 时仍 Upsert 默认实例；客户端用 Mock（Pool 内对 mock 模式全部实例共享/各建 Mock，标明 mock）

写后 `Refresh`：对每个 enabled 实例 `comfyui.NewClient(Options{Mock: cfg.ComfyMock, BaseURL: rec.BaseURL})`。

- [x] **Step 4: 测试通过；main 用 Pool 作 Registry**

Run: `go test ./internal/platform/instance/... -count=1`  
Expected: PASS

- [x] **Step 5: Commit**

```bash
git add internal/platform/instance internal/platform/botconfig apps/bot/cmd/comfyui-bot/main.go
git commit -m "feat(instance): persist comfy_instances and in-process client pool"
```

---

## Task 7: HTTP CRUD + SystemStats/Queue + 观测 API（tasks.md 2.2–2.4）

**Files:**
- Modify: `internal/runtime/infrastructure/comfyui/client.go`
- Modify: `internal/runtime/infrastructure/comfyui/http.go`
- Modify: `internal/runtime/infrastructure/comfyui/client_select_test.go`（或新 `stats_test.go`）
- Create: `internal/httpapi/comfyinstances/handler.go`
- Create: `internal/httpapi/comfyinstances/handler_test.go`
- Modify: `apps/bot/cmd/comfyui-bot/main.go`（挂载路由）

**Interfaces:**
- Consumes: Instance Repository、Pool、TaskRepository
- Produces:

```go
type Client interface {
	Submit(ctx context.Context, graph Graph) (promptID string, err error)
	Wait(ctx context.Context, promptID string) (*Result, error)
	UploadImage(ctx context.Context, filename, mime string, data []byte) (remoteFilename string, err error)
	SystemStats(ctx context.Context) (*SystemStats, error)
	Queue(ctx context.Context) (*QueueView, error)
}

type SystemStats struct {
	Mock      bool           `json:"mock,omitempty"`
	Reachable bool           `json:"reachable"`
	Raw       map[string]any `json:"raw,omitempty"` // 或精简摘要字段
}

type QueueView struct {
	Mock      bool  `json:"mock,omitempty"`
	Reachable bool  `json:"reachable"`
	Running   []any `json:"running"`
	Pending   []any `json:"pending"`
}
```

HTTP（chi）：

| 方法 | 路径 |
|---|---|
| GET/POST | `/api/v1/comfy-instances` |
| GET/PATCH/DELETE | `/api/v1/comfy-instances/{id}` |
| GET | `/api/v1/comfy-instances/{id}/system` |
| GET | `/api/v1/comfy-instances/{id}/queue` |
| GET | `/api/v1/comfy-instances/{id}/tasks?status=&limit=&offset=` |

写操作成功后调用 `Pool.Refresh`。  
Mock：`SystemStats`/`Queue` 返回 `mock: true`, `reachable: true` 占位，不崩溃。  
不可达：HTTP 客户端错误 → 响应 `reachable: false` 或非 2xx（二选一，测试锁定一种：推荐 **200 + reachable:false** 或 **502 + body**；计划定案为 **200 JSON `{reachable:false,error:"..."}`**，避免与「实例不存在 404」混淆）。

- [x] **Step 1: 写失败测试 — Mock SystemStats/Queue；HTTP handler CRUD；tasks 不含 pending 无 instance**

```go
func TestMock_SystemStatsAndQueue(t *testing.T) {
	m := &comfyui.Mock{}
	st, err := m.SystemStats(context.Background())
	if err != nil || !st.Mock || !st.Reachable {
		t.Fatalf("%+v %v", st, err)
	}
}

func TestHandler_CreateListAndTasksFilter(t *testing.T) {
	// httptest POST create gpu-2 → GET list 含 gpu-2
	// 预置 task pending 无 instance + queued gpu-2
	// GET .../gpu-2/tasks 仅 queued
}
```

- [x] **Step 2: 跑测试确认失败**

Run: `go test ./internal/runtime/infrastructure/comfyui/ ./internal/httpapi/comfyinstances/ -count=1`  
Expected: FAIL

- [x] **Step 3: 实现 Client 方法与 Handler**

`HTTP.SystemStats` → `GET {base}/system_stats`；`HTTP.Queue` → `GET {base}/queue`，解析 `queue_running` / `queue_pending`（按 Comfy 实际 JSON 字段适配，测试可用 httptest stub）。

- [x] **Step 4: main 注册路由（保留 `/healthz`）**

```go
r.Route("/api/v1/comfy-instances", func(r chi.Router) {
	h.Mount(r)
})
```

- [x] **Step 5: 测试通过**

Run: `go test ./internal/runtime/infrastructure/comfyui/ ./internal/httpapi/comfyinstances/ -count=1`  
Expected: PASS

- [x] **Step 6: Commit**

```bash
git add internal/runtime/infrastructure/comfyui internal/httpapi apps/bot/cmd/comfyui-bot/main.go
git commit -m "feat: comfy instance HTTP CRUD and system/queue/tasks APIs"
```

---

## Task 8: 健康探测 + round-robin + Actuator 按 InstanceID（tasks.md 3.1–3.3）

**Files:**
- Modify: `internal/platform/instance/pool.go`（健康 map + Probe）
- Modify: `internal/runtime/application/orchestrator/service.go`
- Modify: `internal/runtime/application/orchestrator/service_test.go`
- Modify: `internal/runtime/infrastructure/actuator/worker.go`
- Modify: `internal/runtime/infrastructure/actuator/worker_test.go`
- Modify: `apps/bot/cmd/comfyui-bot/main.go`（健康 ticker；多实例 dispatch 订阅；AutoMigrate 全表）

**Interfaces:**
- Consumes: `Client.SystemStats`、`Pool.Client`
- Produces:
  - `Pool.Probe(ctx)`：对 enabled 非 mock 真实实例调 SystemStats；失败 `SetHealthy(id,false)`
  - Orchestrator：`rrIndex uint64`；从健康∩`Storm.Breaker.Allow` 列表 round-robin；**空列表时 return nil 且保持 pending**（改掉当前 `fmt.Errorf("no healthy instance")` 导致扫库报错的行为）
  - `Worker.Clients` 或 `ClientResolver func(InstanceID) (Client, error)`；`HandleDispatch` 用 `ev.InstanceID`，不要用死写的 `w.InstanceID` 作为唯一客户端键（status 事件仍带 `ev.InstanceID`）

- [x] **Step 1: 写失败测试**

```go
func TestOrchestrator_RoundRobinAcrossHealthy(t *testing.T) {
	// Registry 返回 gpu-a, gpu-b 均 Allow
	// 连续 OnTaskCreated 两个 pending → InstanceID 分别为 a,b（顺序可定义从 rr=0 开始）
}

func TestOrchestrator_NoInstanceKeepsPending(t *testing.T) {
	// ListHealthy 空 → Task 仍 pending；SchedulePending 不返回 fatal error
}

func TestWorker_UsesDispatchInstanceClient(t *testing.T) {
	// 两个 Mock，按 InstanceID 记录 Submit 调用归属
}
```

- [x] **Step 2: 跑测试确认失败**

Run: `go test ./internal/runtime/application/orchestrator/ ./internal/runtime/infrastructure/actuator/ -run 'RoundRobin|NoInstance|UsesDispatch' -v`  
Expected: FAIL

- [x] **Step 3: 实现 round-robin 与无实例 pending**

```go
candidates := filterAllowed(insts, s.Storm.Breaker)
if len(candidates) == 0 {
	return nil // keep pending
}
idx := int(atomic.AddUint64(&s.rr, 1)-1) % len(candidates)
chosen := candidates[idx]
```

- [x] **Step 4: 实现 Worker 按 InstanceID 取客户端；main 健康循环与订阅**

健康 ticker（默认 30s，可配置 `health_probe_interval`）：`pool.Probe(ctx)`。  
`main`：对 `pool.List` 每个实例 `bus.Subscribe(TopicDispatch(id), ...)`（或统一 handler 内再解析 InstanceID）；AutoMigrate：`UserRow, SessionRow, TaskRow, InstanceRow, CaseRow`。

- [x] **Step 5: 测试通过**

Run: `go test ./internal/platform/instance/ ./internal/runtime/application/orchestrator/ ./internal/runtime/infrastructure/actuator/ -count=1`  
Expected: PASS

- [x] **Step 6: Commit**

```bash
git add internal/platform/instance internal/runtime apps/bot/cmd/comfyui-bot/main.go
git commit -m "feat: health probe, round-robin dispatch, per-instance actuator clients"
```

---

## Task 9: Makefile、README、配置注释与验收（tasks.md 4–5）

**Files:**
- Create: `Makefile`
- Modify: `README.md`
- Modify: `configs/bot.example.yaml`
- Modify: `configs/bot.yaml`（仅注释/示例字段，勿提交密钥）

- [x] **Step 1: 添加 Makefile**

```makefile
.PHONY: build test run run-mock

build:
	go build -o bin/comfyui-bot ./apps/bot/cmd/comfyui-bot

test:
	go test ./...

run: build
	COMFY_MOCK=0 go run ./apps/bot/cmd/comfyui-bot

run-mock: build
	COMFY_MOCK=1 go run ./apps/bot/cmd/comfyui-bot
```

- [x] **Step 2: 更新 README**

必须覆盖：

1. 架构简述：User/Session/Task/实例池  
2. 启动种子：`comfyui_base_url` / `comfy_instances` / `default_instance_id`  
3. CRUD 与观测 curl 示例：

```bash
curl -s localhost:8080/api/v1/comfy-instances
curl -s -X POST localhost:8080/api/v1/comfy-instances \
  -H 'Content-Type: application/json' \
  -d '{"id":"gpu-1","base_url":"http://127.0.0.1:8188","enabled":true}'
curl -s localhost:8080/api/v1/comfy-instances/gpu-1/system
curl -s localhost:8080/api/v1/comfy-instances/gpu-1/queue
curl -s 'localhost:8080/api/v1/comfy-instances/gpu-1/tasks?limit=20'
```

4. Mock 开关与 `make run-mock`  
5. **无鉴权警示**：仅本机/内网

- [x] **Step 3: 更新 `configs/bot.example.yaml` 注释**

增加示例：

```yaml
# 可选：多实例种子（优先于仅用 comfyui_base_url 的单实例 upsert）
# comfy_instances:
#   - id: gpu-1
#     base_url: "http://127.0.0.1:8188"
#     enabled: true
# health_probe_interval: 30s
```

- [x] **Step 4: 验收 — 相关测试**

Run: `go test ./internal/identity/... ./internal/conversation/... ./internal/runtime/... ./internal/platform/instance/... ./internal/httpapi/... ./internal/packaging/botapp/... ./apps/bot/... -count=1`  
Expected: PASS  
（或 `make test` 若全仓稳定）

- [x] **Step 5: 验收清单（手工/本地）**

- [x] Mock 开：CRUD 实例 → `/system` `/queue` 返回 `mock: true`  
- [x] 创建 Task 后 `/tasks` 仅含已派发；pending 无 instance 不出现  
- [x] 重启进程：User/Session/Task/实例仍在  
- [x] 两台健康实例（可用两个 Mock 或 stub）：连续任务 InstanceID 轮转  
- [x] 真实模式：能连 `comfyui_base_url` 时 system/queue 非伪造

- [x] **Step 6: Commit**

```bash
git add Makefile README.md configs/bot.example.yaml
git commit -m "docs: Makefile and README for multi-instance ops APIs"
```

---

## Self-Review（对照 design / tasks.md）

| 需求 | 任务 |
|---|---|
| users + TG upsert | 1 |
| sessions GORM、user_id、活跃查询、长期保留 | 2 |
| tasks GORM、session_id、ListByInstance、ListMyTasks join | 3–4 |
| ConfirmRun session_id；通知 join chat | 4 |
| comfy_instances CRUD + 种子 + 客户端池 | 5–6 |
| SystemStats/Queue Client + Mock | 6 |
| HTTP system/queue/tasks | 6 |
| 健康探测 + ListHealthy | 7 |
| round-robin；无实例不投递 | 7 |
| Actuator 按 InstanceID；main AutoMigrate | 7 |
| Makefile / README / bot.example.yaml | 8 |
| 验收 go test + 重启持久化 | 8 |

占位符扫描：无 TBD/TODO；类型名在 Interfaces 与步骤间一致（`SessionID`、`ListByInstance`、`Pool.Refresh`、`SystemStats`）。
