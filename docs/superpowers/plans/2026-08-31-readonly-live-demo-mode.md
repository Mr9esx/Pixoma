# Readonly Live Demo Mode Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

```yaml
---
change: readonly-live-demo-mode
design-doc: docs/superpowers/specs/2026-08-31-readonly-live-demo-mode-design.md
base-ref: 090642943372d1b457257b58b736c174700e0c46
---
```

**Goal:** `-livedemo` 启动一个全后台只读、无持久化副作用的 Pixoma 演示站点。

**Architecture:** 独立 Demo 启动分支打开进程内 ephemeral SQLite，复用现有模型、GORM 仓储和后台 handler；Demo Auth Gate 与只读中间件在业务 handler 前强制固定账号认证和拒绝变更请求。前端通过 `live_demo` 状态展示登录提醒并隐藏设置入口。

**Tech Stack:** Go、chi、GORM、SQLite、React 19、TanStack Router、TanStack Query、Vitest。

## Global Constraints

- Demo 不创建、打开或持久化 bootstrap 数据库、业务数据库或 Blob 存储。
- 进程内 ephemeral SQLite 仅用于当前进程演示状态。
- 固定 Demo 账号为 `admin / 123456`，登录页必须展示提醒。
- 服务端强制只读；仅放行 Demo 登录和登出写请求。
- 设置、配置、注册、修改密码、重启、重试、取消和 Token 轮换必须拒绝。
- 拒绝响应使用 `403` 和 `readonly_live_demo` 错误码。
- 前端不得新增裸 hex；优先复用 shadcn/ui 语义组件与 variants。

---

### Task 1: Demo Auth And Readonly Middleware

**Files:**

- Create: `apps/pixoma/cmd/pixoma/internal/livedemo/auth.go`
- Create: `apps/pixoma/cmd/pixoma/internal/livedemo/middleware.go`
- Test: `apps/pixoma/cmd/pixoma/internal/livedemo/livedemo_test.go`

**Interfaces:**

- Produces: `type AuthStore`、`func NewAuthStore() *AuthStore`、`func (s *AuthStore) Handler(next http.Handler) http.Handler`。
- Produces: `type SetupHandler`、`func NewSetupHandler(*AuthStore) http.Handler`、`func Mount(r chi.Router, auth *AuthStore)`。
- Produces: `func Readonly(next http.Handler) http.Handler`。

- [x] **Step 1: 写失败测试**，覆盖正确凭据登录、错误凭据拒绝、受保护 API 未认证返回 `401`、认证后变更请求返回 `403 readonly_live_demo`、登录/登出放行。
- [x] **Step 2: 运行 `go test ./apps/pixoma/cmd/pixoma/internal/livedemo`**，确认新测试因缺少实现失败。
- [x] **Step 3: 实现 AuthStore 与内存会话**，返回现有前端兼容的 status/login/logout/me 响应形状。
- [x] **Step 4: 实现 Readonly Middleware**，先显式拒绝敏感路径，再默认拒绝非安全方法，仅放行 `/api/v1/setup/login` 和 `/api/v1/setup/logout`。
- [x] **Step 5: 运行 `go test ./apps/pixoma/cmd/pixoma/internal/livedemo`**，确认通过。

### Task 2: Ephemeral Demo Database And Seed

**Files:**

- Create: `apps/pixoma/cmd/pixoma/internal/livedemo/database.go`
- Create: `apps/pixoma/cmd/pixoma/internal/livedemo/seed.go`
- Test: `apps/pixoma/cmd/pixoma/internal/livedemo/database_test.go`

**Interfaces:**

- Consumes: 现有 GORM row models。
- Produces: `func OpenDatabase() (*gorm.DB, func() error, error)`。
- Produces: `func Seed(ctx context.Context, db *gorm.DB) error`。

- [x] **Step 1: 写失败测试**，验证打开的数据库不是文件、迁移成功、Seed 后各领域都能查询到演示数据、统计页可读取聚合行。
- [x] **Step 2: 运行 `go test ./apps/pixoma/cmd/pixoma/internal/livedemo`**，确认失败。
- [x] **Step 3: 实现进程内 SQLite 打开函数**，使用 shared-cache memory DSN，并设置单连接或等效连接复用策略。
- [x] **Step 4: 实现确定性中文演示数据**，覆盖 edges、cases、channels、topics、tasks、sessions、users、console users、menus、daily/edge/error/case stats。
- [x] **Step 5: 运行 `go test ./apps/pixoma/cmd/pixoma/internal/livedemo`**，确认通过。

### Task 3: Demo Router Assembly

**Files:**

- Create: `apps/pixoma/cmd/pixoma/internal/livedemo/server.go`
- Modify: `apps/pixoma/cmd/pixoma/main.go`
- Test: `apps/pixoma/cmd/pixoma/internal/livedemo/server_test.go`

**Interfaces:**

- Consumes: `OpenDatabase`、`Seed`、`Mount`、`Readonly`。
- Produces: `type Server`、`func New(ctx context.Context, addr, publicURL string) (*Server, error)`、`func (s *Server) Shutdown(ctx context.Context) error`。

- [x] **Step 1: 写失败测试**，验证 `New` 不接收 `DATA_DIR`、迁移并注入数据、主要资源 `GET` 返回演示数据、写请求和敏感接口被拦截。
- [x] **Step 2: 运行 `go test ./apps/pixoma/cmd/pixoma/internal/livedemo`**，确认失败。
- [x] **Step 3: 装配内存数据库和现有 admin handlers**，挂载 Demo setup/auth 路由，不装配 bootstrap、Blob、Bot、Edge Agent 或外部队列。
- [x] **Step 4: 修改 `main.go` 解析 `-livedemo` 并进入 `runLiveDemo`**；常规模式行为不变。
- [x] **Step 5: 运行 `go test ./apps/pixoma/cmd/pixoma/...`**，确认通过。

### Task 4: Frontend Live Demo State

**Files:**

- Modify: `web/admin/src/lib/api/setup.ts`
- Modify: `web/admin/src/routes/login.tsx`
- Modify: `web/admin/src/features/setup/login-page.tsx`
- Modify: `web/admin/src/config/menu.ts`
- Modify: `web/admin/src/components/layout/app-sidebar.tsx`
- Test: `web/admin/src/lib/api/setup.test.ts`
- Test: `web/admin/src/features/setup/login-page.test.tsx`
- Test: `web/admin/src/components/layout/app-sidebar.test.tsx`

**Interfaces:**

- Consumes: 后端 `live_demo: boolean`。
- Produces: `SetupStatus.live_demo?: boolean`、`CurrentUser.live_demo?: boolean`。
- Produces: `filterMenuGroupsForDemo(groups, isLiveDemo)`。

- [x] **Step 1: 写失败 API/组件测试**，覆盖 `live_demo` 状态、登录 alert 渲染账号密码、Demo 模式过滤设置菜单。
- [x] **Step 2: 运行 `pnpm --dir web/admin test -- setup live-demo app-sidebar`**，确认新测试失败。
- [x] **Step 3: 扩展类型并透传状态**，在登录页使用 `Alert` 展示 Demo 提醒。
- [x] **Step 4: 侧边栏通过用户/状态过滤 `settings` 菜单项**，保持普通模式不变。
- [x] **Step 5: 运行 `pnpm --dir web/admin test -- setup live-demo app-sidebar`**，确认通过。

### Task 5: Persistence And Startup Isolation Verification

**Files:**

- Test: `apps/pixoma/cmd/pixoma/internal/livedemo/server_test.go`

**Interfaces:**

- Consumes: `New`、`Shutdown`。

- [x] **Step 1: 写失败隔离测试**，设置临时 `DATA_DIR`，启动 Demo Server 后检查目录内没有 bootstrap、business database 或 Blob 文件。
- [x] **Step 2: 运行 `go test ./apps/pixoma/cmd/pixoma/internal/livedemo`**，确认隔离断言生效。
- [x] **Step 3: 补充实现缺口**，确保测试与 Global Constraints 一致。
- [x] **Step 4: 运行 `go test ./apps/pixoma/cmd/pixoma/...`**，确认通过。

### Task 6: Full Validation

**Files:**

- No new implementation files.

- [x] **Step 1: 运行 `gofmt -w apps/pixoma/cmd/pixoma`**。
- [x] **Step 2: 运行 `go test ./...`**。
- [x] **Step 3: 运行 `pnpm --dir web/admin test`**。
- [x] **Step 4: 运行 `pnpm --dir web/admin build`**。
- [x] **Step 5: 用 `HTTP_ADDR=127.0.0.1:18082 DATA_DIR=$(mktemp -d) go run ./apps/pixoma/cmd/pixoma -livedemo` 做人工只读冒烟验证**，随后停止进程。
