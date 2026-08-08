---
change: admin-resources-api
design-doc: docs/superpowers/specs/2026-08-08-admin-resources-api-design.md
base-ref: 7f13625f8de256859429b79059c03ad2f023966d
archived-with: 2026-08-08-admin-resources-api
---

# admin-resources-api Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在已落地的 `apps/admin-api` 上补齐 Case / User / Session / Task 管理 HTTP（含过滤列表与有限写动作），并拆清 admin-api / bot 的 HTTP 宿主目录。

**Architecture:** Handler 落在 `internal/httpapi/{cases,users,sessions,tasks}`，模式对齐 `comfyinstances`；领域侧补齐 catalog `Enable` + 扩展 `ListQuery`、identity/conversation 管理 `List`、runtime 管理 `List` + 取消只调 `orchestrator.RequestCancel`。`apps/admin-api` 拆 `router`/`middleware` 挂载四资源；bot 抽出 `apps/bot/internal/server` 仅 healthz，不恢复管理 CRUD。

**Tech Stack:** Go、chi、GORM/sqlite、既有 catalog/identity/conversation/runtime、`internal/httpapi` 先例、`orchestrator.RequestCancel`、`appboot`

## Global Constraints

- 产物语言：zh-CN
- **禁止** `apps/admin-api` / `internal/httpapi/{cases,users,sessions,tasks}` 依赖 `channel/tg`
- **禁止** 提供代用户 `ConfirmRun` / 管理创建 Task 的入口
- User：**只读**（仅 `GET` 列表/详情）
- Session：**只读**（仅 `GET` 列表/详情；无通用 Update/Delete）
- Case 写动作路径：`POST /api/v1/cases/{id}/disable`、`POST /api/v1/cases/{id}/enable`（另有 `GET/POST/PATCH` 常规 CRUD）
- Task 取消路径：`POST /api/v1/tasks/{id}/cancel`；取消语义**只**复用 `orchestrator.RequestCancel`，不在 admin 另写态机
- 统一前缀 `/api/v1`；分页 `limit`/`offset`；错误 JSON `{"error":"..."}` 对齐 `comfyinstances`
- 列表过滤用显式 `ListQuery` 结构体 + 参数化 SQL；**禁止**把客户端键名直接拼进 SQL
- 无鉴权；README 必须警示仅本机/受信网络；默认绑定沿用 foundation（如 `127.0.0.1`）
- Comfy Mock 开关与本 change 无关的执行链路不改；不破坏既有 comfy-instances API
- 遵循仓库 golang skills；改动范围限于本 change；触及架构表述时同步 `docs/architecture/`

---

## 文件结构（先锁定职责）

| 路径 | 职责 |
|---|---|
| `internal/catalog/domain/repository.go` | `ListQuery` 加 `Q`/`CreatedFrom`/`CreatedTo`；接口加 `Enable` |
| `internal/catalog/infrastructure/persistence/gorm_repository.go` | `Enable` + List 过滤实现 |
| `internal/identity/domain/repository.go` | `ListQuery` + `List` |
| `internal/identity/infrastructure/persistence/gorm_user.go` | User `List` |
| `internal/conversation/domain/service.go`（或新建 `repository.go`） | Session `ListQuery` + `List`；补 `ErrNotFound`（Get 缺失映射） |
| `internal/conversation/infrastructure/persistence/gorm_session.go` | Session `List`；MemoryRepository 同步 |
| `internal/runtime/domain/task_repository.go` | 管理端 `AdminListQuery` + `List` |
| `internal/runtime/infrastructure/persistence/gorm_task.go` | Task 管理 `List`（含 chat JOIN） |
| `internal/platform/notify/nop.go` | 无 TG 的 `notify.Publisher`（admin cancel 通知旁路） |
| `internal/httpapi/cases/` | Case HTTP Mount |
| `internal/httpapi/users/` | User 只读 HTTP |
| `internal/httpapi/sessions/` | Session 只读 HTTP |
| `internal/httpapi/tasks/` | Task HTTP + cancel |
| `apps/admin-api/internal/server/{middleware,router,server}.go` | CORS / 路由挂载 / Options |
| `apps/admin-api/cmd/admin-api/main.go` | 组装 repos + validators + orch + handlers |
| `apps/bot/internal/server/` | bot chi + healthz 抽出 |
| `apps/bot/cmd/comfyui-bot/main.go` | 改用 bot server 包 |
| `apps/admin-api/README.md` | 四类资源 curl + 无鉴权警示 |
| `docs/architecture/overview.md` | admin-api 能力表述从「预留」改为资源管理 HTTP |

---

### Task 1: catalog — Enable + ListQuery 过滤扩展

**Files:**
- Modify: `internal/catalog/domain/repository.go`
- Modify: `internal/catalog/infrastructure/persistence/gorm_repository.go`
- Modify: `internal/catalog/infrastructure/persistence/gorm_repository_test.go`
- Modify（若有 Memory stub）: 任何实现 `domain.Repository` 的测试假对象（如 `internal/runtime/infrastructure/actuator/snapshot_test.go` 的 `memCases`）补 `Enable`

**Interfaces:**
- Produces:
  - `domain.ListQuery` 字段：`Tag, MenuKey, Category string`；`Enabled *bool`；`Q string`；`CreatedFrom, CreatedTo *time.Time`；`Limit, Offset int`
  - `Repository.Enable(ctx context.Context, id sharedkernel.CaseID) error`（不存在 → `ErrNotFound`）
  - `Repository.List` 对 `Q`：在 `id`/`name` 上参数化 `LIKE`；时间窗过滤 `created_at`

- [x] **Step 1: 写失败测试**

在 `gorm_repository_test.go` 追加：

```go
func TestEnableAndListFilters(t *testing.T) {
	repo := openTestDB(t)
	ctx := context.Background()
	c1 := sampleCase("alpha-case", "t1")
	c1.Document.Name = "Alpha Workflow"
	c1.Document.Categories = []string{"gen"}
	_ = repo.Create(ctx, c1)
	c2 := sampleCase("beta-case", "t2")
	c2.Document.Name = "Beta Other"
	_ = repo.Create(ctx, c2)

	if err := repo.Disable(ctx, "alpha-case"); err != nil {
		t.Fatal(err)
	}
	if err := repo.Enable(ctx, "alpha-case"); err != nil {
		t.Fatal(err)
	}
	got, _ := repo.Get(ctx, "alpha-case")
	if !got.Enabled {
		t.Fatal("expected enabled after Enable")
	}
	if err := repo.Enable(ctx, "missing"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}

	list, err := repo.List(ctx, domain.ListQuery{Q: "Alpha", Limit: 10})
	if err != nil || len(list) != 1 || list[0].Document.ID != "alpha-case" {
		t.Fatalf("q filter: err=%v list=%+v", err, list)
	}
	en := true
	list, err = repo.List(ctx, domain.ListQuery{Enabled: &en, Category: "gen", Limit: 10})
	if err != nil || len(list) != 1 {
		t.Fatalf("enabled+category: err=%v n=%d", err, len(list))
	}
}
```

- [x] **Step 2: 跑测试确认失败**

Run: `go test ./internal/catalog/infrastructure/persistence/ -run TestEnableAndListFilters -count=1`

Expected: FAIL（`Enable` undefined 和/或 `Q` 字段不存在）

- [x] **Step 3: 最小实现**

1. `ListQuery` 增加 `Q string`、`CreatedFrom/CreatedTo *time.Time`（需 `import "time"`）。
2. 接口增加 `Enable`。
3. `Enable` 镜像 `Disable`，`Update("enabled", true)`，`RowsAffected==0` → `ErrNotFound`。
4. `List` 中：

```go
if q.Q != "" {
	like := "%" + q.Q + "%"
	tx = tx.Where("id LIKE ? OR name LIKE ?", like, like)
}
if q.CreatedFrom != nil {
	tx = tx.Where("created_at >= ?", *q.CreatedFrom)
}
if q.CreatedTo != nil {
	tx = tx.Where("created_at <= ?", *q.CreatedTo)
}
```

（既有 Tag/Category 过滤保持参数化；**不要**把原始 `q.Tag` 拼进无占位符字符串以外的危险拼接——现有 `LIKE "%\""+tag+"\"%"` 对内部 tag 可接受，但 `Q` 必须用 `?`。）

5. 所有 `Repository` 假实现补空 `Enable`。

- [x] **Step 4: 测试通过**

Run: `go test ./internal/catalog/... -count=1`

Expected: PASS

- [x] **Step 5: Commit**

```bash
git add internal/catalog/
git commit -m "$(cat <<'EOF'
feat(catalog): add Enable and extend ListQuery filters for admin

EOF
)"
```

---

### Task 2: identity — 管理端 User List

**Files:**
- Modify: `internal/identity/domain/repository.go`
- Modify: `internal/identity/infrastructure/persistence/gorm_user.go`
- Modify: `internal/identity/infrastructure/persistence/gorm_user_test.go`

**Interfaces:**
- Produces:
  - `domain.ListQuery`：`Q string`；`TgUserID *int64`；`CreatedFrom, CreatedTo *time.Time`；`Limit, Offset int`
  - `Repository.List(ctx context.Context, q ListQuery) ([]*User, error)`
  - 过滤：`tg_user_id` 精确；`Q` 模糊匹配 `id`/`username`/`first_name`/`last_name`（参数化 `LIKE`）

- [x] **Step 1: 写失败测试**

```go
func TestUserListFilters(t *testing.T) {
	// open migrate users table; Upsert two users with distinct username/tg ids
	// List with TgUserID exact → 1
	// List with Q matching username → 1
	// List with Limit 1 Offset 0 → len==1
}
```

具体断言用既有 `NewUserRepository` + `UpsertByTgUserID` 造数。

- [x] **Step 2: 跑测试确认失败**

Run: `go test ./internal/identity/infrastructure/persistence/ -run TestUserListFilters -count=1`

Expected: FAIL（`List` undefined）

- [x] **Step 3: 最小实现**

```go
// domain/repository.go
type ListQuery struct {
	Q           string
	TgUserID    *int64
	CreatedFrom *time.Time
	CreatedTo   *time.Time
	Limit       int
	Offset      int
}

type Repository interface {
	UpsertByTgUserID(ctx context.Context, in UpsertFrom) (*User, error)
	GetByID(ctx context.Context, id string) (*User, error)
	List(ctx context.Context, q ListQuery) ([]*User, error)
}
```

GORM：`Order("created_at desc")`；`TgUserID != nil` → `Where("tg_user_id = ?", *q.TgUserID)`；`Q` → `id/username/first_name/last_name LIKE ?`。

- [x] **Step 4: 测试通过**

Run: `go test ./internal/identity/... -count=1`

- [x] **Step 5: Commit**

```bash
git add internal/identity/
git commit -m "$(cat <<'EOF'
feat(identity): add admin List query with filters

EOF
)"
```

---

### Task 3: conversation — 管理端 Session List

**Files:**
- Modify: `internal/conversation/domain/service.go`（`Repository` 接口所在文件）
- Modify: `internal/conversation/domain/session.go`（增加 `ErrNotFound`，与 `ErrNoActiveSession` 并存）
- Modify: `internal/conversation/infrastructure/persistence/gorm_session.go`
- Modify: `internal/conversation/domain/service.go` 内 `MemoryRepository` 实现 `List`
- Create/Modify: `internal/conversation/infrastructure/persistence/gorm_session_test.go`（若无则新建）

**Interfaces:**
- Produces:
  - `domain.ListQuery`：`Q string`；`UserID string`；`ChatID *int64`（或 `sharedkernel.ChatID` 零值表示未设）；`Status Status`；`CreatedFrom, CreatedTo *time.Time`；`Limit, Offset int`
  - `Repository.List(ctx, q ListQuery) ([]*Session, error)`
  - `GetByID` 缺失时：持久化层改为返回 `domain.ErrNotFound`（新建 sentinel）；`GetActiveByChat` 仍用 `ErrNoActiveSession`。更新所有依赖 `errors.Is(..., ErrNoActiveSession)` 的 GetByID 断言

**说明：** 管理详情需要区分「无活跃会话」与「ID 不存在」。`GetByID` 用 `ErrNotFound`；活跃查询保持原语义。

- [x] **Step 1: 写失败测试**

```go
func TestSessionListFilters(t *testing.T) {
	// migrate SessionRow; Save two sessions with不同 user_id/chat_id/status
	// List by user_id → 1
	// List by status=submitted → 匹配
	// List by chat_id → 1
	// Q 匹配 case_id 或 id 前缀
}
```

- [x] **Step 2: 跑测试确认失败**

Run: `go test ./internal/conversation/... -run TestSessionListFilters -count=1`

Expected: FAIL

- [x] **Step 3: 最小实现**

GORM List：按字段 `Where`；`Q` → `id LIKE ? OR case_id LIKE ? OR user_id LIKE ?`；`Order("updated_at desc")`。

`GetByID`：

```go
if errors.Is(err, gorm.ErrRecordNotFound) {
	return nil, domain.ErrNotFound
}
```

`MemoryRepository.List`：线性过滤，供 handler 单测可选用。

- [x] **Step 4: 测试通过**

Run: `go test ./internal/conversation/... -count=1`

- [x] **Step 5: Commit**

```bash
git add internal/conversation/
git commit -m "$(cat <<'EOF'
feat(conversation): add admin Session List and ErrNotFound for GetByID

EOF
)"
```

---

### Task 4: runtime — Admin List + cancel 可被 admin 复用

**Files:**
- Modify: `internal/runtime/domain/task_repository.go`
- Modify: `internal/runtime/infrastructure/persistence/gorm_task.go`
- Modify: `internal/runtime/infrastructure/persistence/gorm_task_test.go`
- Create: `internal/platform/notify/nop.go`
- Modify: `internal/runtime/application/orchestrator/service_test.go`（若需覆盖 RequestCancel + Nop；可复用既有 cancel 测试）

**Interfaces:**
- Produces:
  - `domain.AdminListQuery`：
    - `Q string`
    - `Status sharedkernel.TaskStatus`
    - `InstanceID sharedkernel.InstanceID`
    - `ChatID sharedkernel.ChatID`（0 = 未过滤）
    - `SessionID sharedkernel.SessionID`
    - `CaseID sharedkernel.CaseID`
    - `CreatedFrom, CreatedTo *time.Time`
    - `Limit, Offset int`
  - `TaskRepository.List(ctx context.Context, q AdminListQuery) ([]*Task, error)`
  - `notify.Nop`：`type Nop struct{}`；`func (Nop) Publish(context.Context, sharedkernel.UserNotify) error { return nil }`
  - Cancel：**不新写取消逻辑**；admin 侧将注入 `orchestrator.Service{Tasks, Notify: notify.Nop{}, Sessions: sessionRepo, Now: ...}` 并调用 `RequestCancel`
  - HTTP 映射约定（本任务在仓储/通知就绪；handler 任务实现）：
    - `ErrTaskNotFound` → 404
    - `ErrCancelNotAllowed` → 409
    - 成功 → 200 + task DTO

**chat_id 过滤：** 与既有 `ListByChat` 相同，`JOIN sessions ON tasks.session_id = sessions.id` + `sessions.chat_id = ?`。其它过滤在 `tasks` 列上。

- [x] **Step 1: 写失败测试**

在 `gorm_task_test.go`：

```go
func TestTaskAdminListFilters(t *testing.T) {
	// migrate TaskRow + SessionRow；插入 session(chat=42) + tasks
	// List AdminListQuery{SessionID: ...} 
	// List AdminListQuery{ChatID: 42} 经 JOIN
	// List AdminListQuery{Status: pending, CaseID: ...}
	// List AdminListQuery{Q: 前缀}
}
```

另加/确认：

```go
func TestRequestCancelViaOrchestrator(t *testing.T) {
	tasks := domain.NewMemoryTaskRepository()
	// create pending task
	orch := orchestrator.New(tasks, nil, nil, notify.Nop{})
	if err := orch.RequestCancel(ctx, "t1"); err != nil {
		t.Fatal(err)
	}
	// create succeeded task → errors.Is(ErrCancelNotAllowed)
}
```

（若 `New` 对 nil Registry 不安全，测试里传最小假对象；`RequestCancel` 路径不碰 Instances/Dispatch。）

- [x] **Step 2: 跑测试确认失败**

Run: `go test ./internal/runtime/infrastructure/persistence/ -run TestTaskAdminListFilters -count=1`

Expected: FAIL

- [x] **Step 3: 最小实现**

1. 接口加 `List`；`MemoryTaskRepository.List` 内存过滤（含 ChatID 精确匹配 `t.ChatID`，无 session JOIN——内存测可设 `ChatID`）。
2. GORM `List`：有 `ChatID != 0` 时用 Table+Joins；否则 `Model(&TaskRow{})`；全部条件用 `?`。
3. 新增 `notify.Nop`。

- [x] **Step 4: 测试通过**

Run: `go test ./internal/runtime/... ./internal/platform/notify/ -count=1`

- [x] **Step 5: Commit**

```bash
git add internal/runtime/ internal/platform/notify/
git commit -m "$(cat <<'EOF'
feat(runtime): add admin Task List and notify.Nop for cancel wiring

EOF
)"
```

---

### Task 5: httpapi/cases — Case 管理 HTTP

**Files:**
- Create: `internal/httpapi/cases/handler.go`
- Create: `internal/httpapi/cases/handler_test.go`

**Interfaces:**
- Consumes: `catalog/domain.Repository`；`validation.Validator.ValidateDocument`
- Produces: `type Handler struct { Repo domain.Repository; Validate func(domain.CaseDocument) error }`；`func (h *Handler) Mount(r chi.Router)`
- 路由（挂在 `/api/v1/cases`）：
  - `GET /` list（query: `q,created_from,created_to,enabled,category,tag,menu_key,limit,offset`）
  - `POST /` create（body = CaseDocument + optional `enabled`；校验失败 400）
  - `GET /{id}` get
  - `PATCH /{id}` update document（Save；校验失败 400；不存在 404）
  - `POST /{id}/disable`
  - `POST /{id}/enable`
- 错误：`writeErr`/`writeJSON` 复制 `comfyinstances` 风格（包内私有即可）
- **禁止** import `channel/tg`

- [x] **Step 1: 写失败测试**

表驱动 httptest + 内存/sqlite repo：

```go
func TestCasesHandler_CRUDEnableDisable(t *testing.T) {
	// migrate CaseRow; Handler{Repo, Validate: validation.New().ValidateDocument}
	// POST valid → 201
	// POST invalid (empty name) → 400，库中无脏数据
	// GET list ?q= → 200
	// POST /{id}/disable → enabled=false
	// POST /{id}/enable → enabled=true
	// GET missing → 404
}
```

合法最小文档可复用 catalog 测试的 `sampleCase` 字段形状。

- [x] **Step 2: 跑测试确认失败**

Run: `go test ./internal/httpapi/cases/ -count=1`

Expected: FAIL（包不存在）

- [x] **Step 3: 实现 Handler**

参考 `internal/httpapi/comfyinstances/handler.go` 的 `Mount`/`writeJSON`/`writeErr`/`chi.URLParam`。

时间解析：`created_from`/`created_to` 用 `time.RFC3339`；非法 → 400。  
`enabled` query：`true`/`false`/`1`/`0`；空 = 不过滤。

Create：`Repo.Create`；`ErrAlreadyExists` → 409。  
Update：先 `Get`，合并 document JSON，`Validate`，`Save`。

- [x] **Step 4: 测试通过**

Run: `go test ./internal/httpapi/cases/ -count=1`

- [x] **Step 5: Commit**

```bash
git add internal/httpapi/cases/
git commit -m "$(cat <<'EOF'
feat(httpapi): add cases admin CRUD enable/disable handlers

EOF
)"
```

---

### Task 6: httpapi/users — User 只读 HTTP

**Files:**
- Create: `internal/httpapi/users/handler.go`
- Create: `internal/httpapi/users/handler_test.go`

**Interfaces:**
- Consumes: `identity/domain.Repository`
- Produces: `Handler{Repo}`；`Mount`：`GET /`、`GET /{id}`
- Query：`q, tg_user_id, created_from, created_to, limit, offset`
- **禁止** POST/PATCH/DELETE 路由
- DTO 字段：`id, tg_user_id, username, first_name, last_name, language_code, last_seen_at, created_at, updated_at`

- [x] **Step 1: 写失败测试**

```go
func TestUsersHandler_ListGetReadOnly(t *testing.T) {
	// Upsert user; GET /api/v1/users → 含该用户
	// GET /api/v1/users/{id} → 200
	// GET missing → 404
	// 路由上 Method POST /api/v1/users → 405 或 404（未注册即可）
}
```

- [x] **Step 2: 跑测试确认失败**

Run: `go test ./internal/httpapi/users/ -count=1`

- [x] **Step 3: 实现**

`tg_user_id` 用 `strconv.ParseInt`；非法 → 400。`ErrNotFound` → 404。

- [x] **Step 4: 测试通过**

Run: `go test ./internal/httpapi/users/ -count=1`

- [x] **Step 5: Commit**

```bash
git add internal/httpapi/users/
git commit -m "$(cat <<'EOF'
feat(httpapi): add read-only users admin handlers

EOF
)"
```

---

### Task 7: httpapi/sessions — Session 只读 HTTP

**Files:**
- Create: `internal/httpapi/sessions/handler.go`
- Create: `internal/httpapi/sessions/handler_test.go`

**Interfaces:**
- Consumes: `conversation/domain.Repository`
- Produces: `Handler{Repo}`；`Mount`：`GET /`、`GET /{id}`
- Query：`q, user_id, chat_id, status, created_from, created_to, limit, offset`
- DTO：含 `id, user_id, chat_id, case_id, status, current_input_index, input_keys, draft, created_at, updated_at`（排障所需）
- **禁止** 通用写路由

- [x] **Step 1: 写失败测试**

造两条 Session（不同 status/user）；列表过滤；详情含 draft；缺失 → 404（`ErrNotFound`）。

- [x] **Step 2: 跑测试确认失败**

Run: `go test ./internal/httpapi/sessions/ -count=1`

- [x] **Step 3: 实现**

与 users 同风格；`status` 原样传入 `domain.Status`。

- [x] **Step 4: 测试通过**

Run: `go test ./internal/httpapi/sessions/ -count=1`

- [x] **Step 5: Commit**

```bash
git add internal/httpapi/sessions/
git commit -m "$(cat <<'EOF'
feat(httpapi): add read-only sessions admin handlers

EOF
)"
```

---

### Task 8: httpapi/tasks — Task 列表/详情/取消

**Files:**
- Create: `internal/httpapi/tasks/handler.go`
- Create: `internal/httpapi/tasks/handler_test.go`

**Interfaces:**
- Consumes: `runtimedomain.TaskRepository`；取消端口：

```go
type Canceller interface {
	RequestCancel(ctx context.Context, taskID sharedkernel.TaskID) error
}
```

- Produces: `Handler{Tasks TaskRepository; Cancel Canceller}`；`Mount`：
  - `GET /`
  - `GET /{id}`
  - `POST /{id}/cancel`
- Query：`q, status, instance_id, chat_id, session_id, case_id, created_from, created_to, limit, offset`
- **禁止** 任何 ConfirmRun / `POST /` 创建 Task
- 取消映射：`ErrTaskNotFound`→404；`ErrCancelNotAllowed`→409；成功后 `Get` 再返回 DTO（200）

- [x] **Step 1: 写失败测试**

```go
func TestTasksHandler_ListGetCancel(t *testing.T) {
	tasks := runtimedomain.NewMemoryTaskRepository()
	orch := orchestrator.New(tasks, nil, nil, notify.Nop{})
	h := &tasksapi.Handler{Tasks: tasks, Cancel: orch}
	// create pending + succeeded
	// GET list ?status=pending
	// POST /{pending}/cancel → 200 status=cancelled
	// POST /{succeeded}/cancel → 409
	// POST /missing/cancel → 404
	// 确认没有 POST / 创建路由（405/404）
}
```

- [x] **Step 2: 跑测试确认失败**

Run: `go test ./internal/httpapi/tasks/ -count=1`

- [x] **Step 3: 实现**

DTO 对齐 comfyinstances 的 `taskDTO`，并加上 `chat_id`（若有）。取消**只**调用 `h.Cancel.RequestCancel`，不要在 handler 里直接 `MarkCancelled`。

- [x] **Step 4: 测试通过**

Run: `go test ./internal/httpapi/tasks/ -count=1`

- [x] **Step 5: Commit**

```bash
git add internal/httpapi/tasks/
git commit -m "$(cat <<'EOF'
feat(httpapi): add tasks list/get and RequestCancel endpoint

EOF
)"
```

---

### Task 9: admin-api — 拆分 router/middleware 并挂载四资源

**Files:**
- Create: `apps/admin-api/internal/server/middleware.go`（从现 `server.go` 移出 `corsMiddleware`）
- Create: `apps/admin-api/internal/server/router.go`（或把 `NewHandler` 主体迁入并保留 `server.go` 导出 Options）
- Modify: `apps/admin-api/internal/server/server.go` — `Options` 扩展
- Modify: `apps/admin-api/internal/server/server_test.go` — 四资源挂载冒烟
- Modify: `apps/admin-api/cmd/admin-api/main.go` — migrate + 组装
- Modify: `docs/architecture/overview.md` — admin-api 角色描述

**Interfaces:**
- Produces `server.Options`：

```go
type Options struct {
	CORSOrigins []string
	Instances   *comfyinstances.Handler
	Cases       *cases.Handler
	Users       *users.Handler
	Sessions    *sessions.Handler
	Tasks       *tasks.Handler
}
```

- `NewHandler` 注册：
  - `/healthz`
  - `/api/v1/comfy-instances`（既有）
  - `/api/v1/cases` / `users` / `sessions` / `tasks`（handler 非 nil 时 Mount）
- `main.go` Bootstrap `Models` 至少包含：`CaseRow`、`UserRow`、`SessionRow`、`TaskRow`（外加 `MigrateInstances: true`）
- 组装：
  - cases: `persistence.NewGormRepository` + `validation.New().ValidateDocument`
  - users/sessions: 对应 GORM repo
  - tasks: `taskpersist.NewTaskRepository` + `orchestrator.New(tasks, pool, memoryBusOrNop, notify.Nop{})`，并设置 `orch.Sessions = sessionRepo`（cancel 通知旁路用 Nop，但仍可解析 chat）
- **禁止** import `channel/tg`

- [x] **Step 1: 写失败/扩展测试**

在 `server_test.go` 增加：

```go
func TestNewHandler_MountsResourceRoutes(t *testing.T) {
	// 用真实或轻量 Handler（Repos repo）注入 Options
	// GET /api/v1/cases → 200（空列表）
	// GET /api/v1/users → 200
	// GET /api/v1/sessions → 200
	// GET /api/v1/tasks → 200
	// GET /healthz → 200
}
```

- [x] **Step 2: 拆分文件 + 实现挂载 + main 接线**

把 CORS 挪到 `middleware.go`；路由集中在 `NewHandler`/`router.go`。更新 overview：

```markdown
| `apps/admin-api` | 管理 HTTP（实例 + Case/User/Session/Task；约定不依赖 `channel/tg`） |
```

- [x] **Step 3: 测试通过**

Run: `go test ./apps/admin-api/... ./internal/httpapi/... -count=1`

- [x] **Step 4: Commit**

```bash
git add apps/admin-api/ docs/architecture/overview.md
git commit -m "$(cat <<'EOF'
feat(admin-api): split router/middleware and mount resource APIs

EOF
)"
```

---

### Task 10: bot — 抽出 `apps/bot/internal/server`（仅 healthz）

**Files:**
- Create: `apps/bot/internal/server/server.go`
- Create: `apps/bot/internal/server/server_test.go`
- Modify: `apps/bot/cmd/comfyui-bot/main.go` — 用 `botserver.NewHandler()` 替代内联 chi

**Interfaces:**
- Produces: `func NewHandler() http.Handler` — `RequestID`/`RealIP`/`Recoverer` + `GET /healthz` → `ok`
- **禁止** 挂载 `/api/v1/cases|users|sessions|tasks|comfy-instances`

- [x] **Step 1: 写失败测试**

```go
func TestBotServer_HealthzOnly(t *testing.T) {
	h := server.NewHandler()
	// GET /healthz → 200 ok
	for _, path := range []string{
		"/api/v1/cases", "/api/v1/users", "/api/v1/sessions", "/api/v1/tasks", "/api/v1/comfy-instances",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("%s status=%d, want 404", path, rec.Code)
		}
	}
}
```

- [x] **Step 2: 跑测试确认失败**

Run: `go test ./apps/bot/internal/server/ -count=1`

Expected: FAIL（包不存在）

- [x] **Step 3: 实现并改 main**

```go
// apps/bot/internal/server/server.go
func NewHandler() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.RealIP, middleware.Recoverer)
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	return r
}
```

`main.go`：`srv := &http.Server{Addr: addr, Handler: botserver.NewHandler(), ...}`

- [x] **Step 4: 测试通过**

Run: `go test ./apps/bot/... -count=1`

- [x] **Step 5: Commit**

```bash
git add apps/bot/
git commit -m "$(cat <<'EOF'
refactor(bot): extract healthz-only HTTP server package

EOF
)"
```

---

### Task 11: README curl + 联调验收 + 勾选 OpenSpec tasks

**Files:**
- Modify: `apps/admin-api/README.md`
- Modify: `docs/openspec/changes/admin-resources-api/tasks.md`（勾选已完成项）

**Interfaces:**
- Produces: README 含四类资源主路径 curl、分页/过滤示例、取消/enable/disable、**无鉴权警示**；明确无 ConfirmRun、User/Session 只读

- [x] **Step 1: 更新 README 示例**

追加类似：

```bash
# Case
curl -s 'localhost:8081/api/v1/cases?limit=20'
curl -s -X POST localhost:8081/api/v1/cases/{id}/disable
curl -s -X POST localhost:8081/api/v1/cases/{id}/enable

# User / Session（只读）
curl -s 'localhost:8081/api/v1/users?tg_user_id=123'
curl -s 'localhost:8081/api/v1/sessions?user_id=...'

# Task
curl -s 'localhost:8081/api/v1/tasks?status=pending'
curl -s -X POST localhost:8081/api/v1/tasks/{id}/cancel
```

强调：**本期无鉴权，勿对公网暴露。**

- [x] **Step 2: 跑全量相关测试**

Run:

```bash
go test ./apps/admin-api/... ./apps/bot/internal/server/ \
  ./internal/httpapi/... \
  ./internal/catalog/... ./internal/identity/... \
  ./internal/conversation/... ./internal/runtime/... \
  ./internal/platform/notify/ -count=1
```

Expected: PASS

- [x] **Step 3: 本地冒烟（可选但建议）**

```bash
go build -o "$TMPDIR/admin-api" ./apps/admin-api/cmd/admin-api
HTTP_ADDR=127.0.0.1:18081 DATABASE_DSN="file:$TMPDIR/admin-res.db?cache=shared" \
  COMFY_MOCK=1 "$TMPDIR/admin-api" &
curl -fsS http://127.0.0.1:18081/healthz
curl -fsS http://127.0.0.1:18081/api/v1/cases
curl -fsS http://127.0.0.1:18081/api/v1/users
curl -fsS http://127.0.0.1:18081/api/v1/sessions
curl -fsS http://127.0.0.1:18081/api/v1/tasks
```

- [x] **Step 4: 勾选 `docs/openspec/changes/admin-resources-api/tasks.md` 对应项**

- [x] **Step 5: Commit**

```bash
git add apps/admin-api/README.md docs/openspec/changes/admin-resources-api/tasks.md
git commit -m "$(cat <<'EOF'
docs(admin-api): document resource API curls and mark tasks done

EOF
)"
```

---

## 执行注意

- 每个 Task 可独立审查；不要把 `web/admin` / 鉴权 / ConfirmRun 带进来
- TDD：仓储与 handler 优先红绿；纯文档步骤可 direct
- Cancel 竞态以领域状态为准；已终态 → 409，勿伪造成功
- Case 校验失败必须 4xx 且不半写入
- 提交信息用英文 conventional 前缀，与仓库近期风格一致

## Spec 覆盖自检

| 规格 / 设计点 | 任务 |
|---|---|
| Case 列表/详情/过滤 | Task 1, 5 |
| Case 创建/更新/校验拒绝 | Task 5 |
| Case disable/enable | Task 1, 5 |
| User 只读列表/详情/过滤 | Task 2, 6 |
| 无 User 写入口 | Task 6, Global Constraints |
| Session 只读列表/详情/过滤 | Task 3, 7 |
| 无 Session 通用写 | Task 7, Global Constraints |
| Task 列表/详情/过滤 | Task 4, 8 |
| Task cancel via RequestCancel | Task 4, 8 |
| 无 ConfirmRun | Task 8, Global Constraints |
| admin-api router/middleware 挂载 | Task 9 |
| bot healthz 抽出、无管理 CRUD | Task 10 |
| README curl + 无鉴权 | Task 11 |
| 无 channel/tg | Global Constraints + Task 5–9 |
| 架构 overview 同步 | Task 9 |
