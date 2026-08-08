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

- [ ] **Step 1: 写失败/扩展测试**

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

- [ ] **Step 2: 拆分文件 + 实现挂载 + main 接线**

把 CORS 挪到 `middleware.go`；路由集中在 `NewHandler`/`router.go`。更新 overview：

```markdown
| `apps/admin-api` | 管理 HTTP（实例 + Case/User/Session/Task；约定不依赖 `channel/tg`） |
```

- [ ] **Step 3: 测试通过**

Run: `go test ./apps/admin-api/... ./internal/httpapi/... -count=1`

- [ ] **Step 4: Commit**

```bash
git add apps/admin-api/ docs/architecture/overview.md
git commit -m "$(cat <<'EOF'
feat(admin-api): split router/middleware and mount resource APIs

EOF
)"
```

---

