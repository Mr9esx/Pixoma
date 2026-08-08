# Task 2 Brief: admin-api 进程骨架

## Full plan task text

### Task 2: admin-api 进程骨架（health + CORS + 配置）

**Files:**
- Create: `apps/admin-api/cmd/admin-api/main.go`
- Create: `internal/platform/adminconfig/config.go`（或与 botconfig 共享薄封装；优先独立 adminconfig 避免拖入 TG 字段）
- Create: `configs/admin-api.example.yaml`
- Modify: `apps/admin-api/README.md`、`Makefile`（若已有 run 目标则增加 `run-admin-api`）

**Interfaces:**
- Consumes: appboot
- Produces: 可监听 `http_addr`（默认 `:8081`）、`GET /healthz` 返回 ok、CORS 中间件允许本地开发源

Steps:
- Step 1: 写失败测试或 smoke — 配置 Load 默认端口；router 对 `/healthz` 返回 200（可用 httptest）
- Step 2: 实现配置 + main 接线（开库 migrate，先不挂实例路由也可，但建议同任务挂空路由组）
- Step 3: 测试/本地启动通过
- Step 4: Commit — `feat(admin-api): add host process with health and CORS`

## Allowed

- Create/modify: `apps/admin-api/**`, `internal/platform/adminconfig/**`, `configs/admin-api.example.yaml`, `Makefile`, README under apps/admin-api
- Consume: `internal/platform/appboot`
- May read botconfig for patterns only
- Forbidden: moving comfyinstances off bot (Task 3); bot Probe Refresh (Task 4); frontend

## TDD

You MUST follow TDD. Load `/Users/mr9esx/Documents/Pixoma/.agents/skills/test-driven-development/SKILL.md`.
Prefer testable packages (adminconfig Load; chi router builder returning http.Handler) so tests don't require full ListenAndServe.

## Tests

```bash
go test ./internal/platform/adminconfig/...
# and any package that owns /healthz router if extracted
go test ./apps/admin-api/...
```

Report DONE|... with commit hash, files, RED/GREEN evidence, risk signals.
