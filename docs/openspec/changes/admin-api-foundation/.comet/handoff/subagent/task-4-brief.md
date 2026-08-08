# Task 4 Brief: bot 探活周期内 Refresh 名单

## Full plan task text

### Task 4: bot 探活周期内 Refresh 名单

**Files:**
- Modify: `apps/bot/cmd/comfyui-bot/main.go` — Probe 循环内先 `pool.Refresh` 再 `pool.Probe`
- Create/Modify: `internal/platform/instance` 相关测试，或 bot 级可测辅助函数

**Interfaces:**
- Consumes: `Pool.Refresh`、`Pool.Probe`、`health_probe_interval`
- Produces: 管理端写库后，一个探活间隔内 bot 调度视图可见

Steps:
- Step 1: 写失败测试 — 模拟库中新增启用实例后调用 Refresh，Pool 可见新客户端/列表
- Step 2: 实现循环内 Refresh
- Step 3: 测试通过
- Step 4: Commit — `fix(bot): refresh instance pool each health probe tick`

## Design constraint
admin-api write refreshes its own pool immediately; bot must Refresh from DB each health probe tick (same interval as Probe). Not instantaneous RPC.

## Allowed
- `apps/bot/cmd/comfyui-bot/main.go`
- `internal/platform/instance/**` tests/helpers
- Prefer extractable helper like `func tickPool(ctx, pool)` tested without waiting 30s

## Forbidden
- Frontend; other resource APIs; auth

## TDD required
Load test-driven-development skill. RED then GREEN evidence mandatory.

## Tests
```bash
go test ./internal/platform/instance/...
go test ./apps/bot/cmd/comfyui-bot/...
```

Report DONE|... with commit, files, RED/GREEN, risk signals.
