## Why

新部署形态已收敛为 `pixoma`（控制面一体：引导、向导、管理 API、Agent API、内嵌 bot 运行时、发布时 `go:embed` 管理前端）与 `pixoma-edge-agent`（执行面）。`apps/bot`（旧 Redis / runtime_mode 拆分时代的独立 bot）与 `apps/admin-api`（过渡期独立管理 HTTP）不再被任何构建或运行路径使用，仅残留代码、配置与文档，造成维护负担和入口混淆。

## What Changes

- 删除 `apps/bot/` 整个目录（旧版拆分 bot 入口；bot 运行时已内嵌 pixoma）。
- 删除 `apps/admin-api/` 整个目录（过渡期独立管理 HTTP；管理 API 已由 pixoma 托管）。
- 删除 `internal/platform/adminconfig`（唯一消费者是 admin-api）。
- 删除 `configs/admin-api.yaml` 与 `configs/admin-api.example.yaml`。
- backfill 命令迁移至 `apps/pixoma/cmd/backfill-task-stats`，改为读取 `DATABASE_DSN` / `DATA_DIR` / `DB_DRIVER` 环境变量，不再依赖 adminconfig。
- Makefile 移除 `bin/admin-api` 构建目标与 `run-admin-api`；`build` 只产出 `pixoma` 与 `pixoma-edge-agent`。
- 文档同步：README、`docs/architecture/overview.md`、`data-model.md`、`runtime.md` 去掉独立 admin-api / bot 入口描述。

## Capabilities

### New Capabilities
<!-- 无 -->

### Modified Capabilities
- `admin-api-host`: 独立 admin-api 进程与无鉴权管理 HTTP 需求移除（过渡期结束）；管理 HTTP 仅由 pixoma 一体托管并受管理员会话门禁。

## Impact

- 构建：Makefile 目标收敛为 `pixoma` + `pixoma-edge-agent`；`go build ./...` / `go test ./...` 不再包含被删目录。
- 运行：新部署只启动 `pixoma`（+ 按需 `pixoma-edge-agent`）；`bin/admin-api` 不再产出。
- 运维工具：backfill 迁移到 pixoma 命令下，环境变量入口变化（`DATABASE_DSN` / `DATA_DIR` / `DB_DRIVER`）。
- 文档：README 服务表与环境变量表、架构文档同步。
