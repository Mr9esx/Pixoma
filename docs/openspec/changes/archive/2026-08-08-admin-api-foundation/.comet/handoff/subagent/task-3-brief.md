# Task 3 Brief: 挂载实例 API 并卸下 bot 管理路由

## Full plan task text

### Task 3: 挂载实例 API 并卸下 bot 管理路由

**Files:**
- Modify: `apps/admin-api/cmd/admin-api/main.go` — Mount `/api/v1/comfy-instances`
- Modify: `apps/bot/cmd/comfyui-bot/main.go` — 移除 comfyinstances Mount；改用 appboot；保留 `/healthz` 若需要
- Modify: `README.md` — 实例管理 curl 改指向 admin-api
- Test: 复用/扩展 `internal/httpapi/comfyinstances/handler_test.go`（语义不变）

**Interfaces:**
- Consumes: `comfyinstances.Handler`、Pool、Repo、Tasks、Mock 开关
- Produces: admin-api 上与迁出前一致的 CRUD/观测行为；bot 不再提供管理路径

Steps:
- Step 1: 确认现有 handler 测试仍绿
- Step 2: admin-api 挂载 Handler + Pool.Refresh 写路径
- Step 3: bot 删除管理路由挂载
- Step 4: 更新 README / admin-api README（无鉴权、同库、新端口）
- Step 5: Commit — `feat(admin-api): migrate comfy-instances HTTP off bot`

## Context

- admin-api already has empty `/api/v1/comfy-instances` route group and appboot with MigrateInstances
- bot currently mounts comfyinstances on its HTTP server — remove that
- Prefer wiring bot to appboot if not already done (Task 1 deferred this)
- Keep handler package in `internal/httpapi/comfyinstances`; do not rewrite protocol
- No auth; no channel/tg in admin-api
- Default admin-api :8081

## Allowed files

- `apps/admin-api/**`, `apps/bot/cmd/comfyui-bot/main.go`, `README.md`, `apps/admin-api/README.md`, `internal/httpapi/comfyinstances/**` (tests only unless tiny mount helper needed)
- Forbidden: Probe-loop Refresh (Task 4), frontend, Case/User APIs

## TDD

You MUST follow TDD. Load `/Users/mr9esx/Documents/Pixoma/.agents/skills/test-driven-development/SKILL.md`
For mount migration: first ensure handler tests green (baseline); if adding mount wiring tests, RED then GREEN. Removing bot routes should have a test or smoke proving management path is gone from bot router if extractable; otherwise document manual verification evidence for Task 5.

## Tests to run

```bash
go test ./internal/httpapi/comfyinstances/...
go test ./apps/admin-api/...
go test ./apps/bot/...   # or package that covers bot main if any; at least build bot
go build -o /tmp/comfyui-bot ./apps/bot/cmd/comfyui-bot
go build -o /tmp/admin-api ./apps/admin-api/cmd/admin-api
```

Report DONE|... with commit hash, files, RED/GREEN (or baseline green + change evidence), risk signals.
