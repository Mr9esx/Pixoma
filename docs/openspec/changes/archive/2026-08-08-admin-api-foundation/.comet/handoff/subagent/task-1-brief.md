# Task 1 Brief: 公共启动包 appboot

## Full plan task text

### Task 1: 公共启动包 appboot

**Files:**
- Create: `internal/platform/appboot/boot.go`（及必要的 options 类型）
- Create: `internal/platform/appboot/boot_test.go`
- Modify（按需）: `apps/bot/cmd/comfyui-bot/main.go` 改为调用 appboot（可在本任务末尾或 Task 3 完成；本任务至少提供可测 API）

**Interfaces:**
- Produces: 类似 `appboot.Open(ctx, opts) (*gorm.DB, error)` 与 `appboot.Migrate(db, models...)` / `Bootstrap(opts) (db, cleanup, error)`；opts 含 DSN、是否 migrate 实例相关模型、可选实例种子列表

Steps:
- Step 1: 写失败测试 — 内存/临时 sqlite DSN 下 Bootstrap 后 `comfy_instances` 可写入/读出（或 migrate 后表存在）
- Step 2: 跑测试确认失败
- Step 3: 实现最小 appboot
- Step 4: 测试通过
- Step 5: Commit — `feat(appboot): extract shared DB bootstrap for admin-api and bot`

## Global constraints

- Language for commits/comments as existing repo; user-facing docs zh-CN later
- No auth; no new tables; do not depend on channel/tg
- Reuse existing `InstanceRow` / db.Open patterns from `internal/platform/db` and `internal/platform/instance/persistence`
- TDD required: RED evidence then GREEN evidence

## Allowed files

- Create/modify: `internal/platform/appboot/**`
- May read: `internal/platform/db/**`, `internal/platform/instance/persistence/**`, `apps/bot/cmd/comfyui-bot/main.go`
- Optional this task: wire bot main to appboot OR leave for Task 3 — prefer delivering testable API only if wiring is large
- Forbidden: admin-api routes, comfyinstances handler moves, frontend, other BC refactors

## Tests to run

```bash
go test ./internal/platform/appboot/...
```

## Report status

DONE | DONE_WITH_CONCERNS | BLOCKED | NEEDS_CONTEXT

Include: commit hash, files changed, RED command+summary, GREEN command+summary, risk signals hit (list or none).
