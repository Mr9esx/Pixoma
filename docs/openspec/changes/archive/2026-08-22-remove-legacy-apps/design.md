## Context

参见 proposal.md - Why：部署形态已收敛为 `pixoma` + `pixoma-edge-agent`。`apps/bot` 与 `apps/admin-api` 是历史拆分/过渡入口，`internal/platform/adminconfig` 仅被 admin-api 消费；backfill 运维命令位于 `apps/admin-api/cmd/backfill-task-stats`，需要随迁移保留。

## Goals / Non-Goals

**Goals:**
- 删除 `apps/bot`、`apps/admin-api`、`internal/platform/adminconfig` 与 admin-api 配置文件，Makefile 只构建 `pixoma` / `pixoma-edge-agent`。
- backfill 命令迁移到 `apps/pixoma/cmd/backfill-task-stats`，配置改为环境变量（`DATABASE_DSN` / `DATA_DIR` / `DB_DRIVER`）。
- 同步文档，使新部署入口描述一致。

**Non-Goals:**
- 不删除 `internal/httpapi/adminhost`（pixoma 仍在用）与 `internal/packaging/botapp`（领域门面，非旧入口）。
- 不改 `pixoma` 管理 API 行为与 `web/admin` 前端。
- 不动历史归档文档与旧 verify 报告。

## Decisions

### 1. 整目录删除，不做"保留但不再构建"

`apps/bot` 与 `apps/admin-api` 已无任何运行引用（Makefile 不构建 bot；admin-api 仅文档提及过渡期），直接删除目录与其专属配置、`adminconfig` 包。备选：保留目录仅停用构建——仍需维护且易被误用，放弃。

### 2. backfill 迁移到 pixoma 命令并改用环境变量

新建 `apps/pixoma/cmd/backfill-task-stats/main.go`：读取 `DB_DRIVER`（默认 sqlite）、`DATABASE_DSN`（缺省 `DATA_DIR/app.db`），复用 `db.Open` / AutoMigrate；逻辑与现实现一致（按 completed_at 归天重算四张统计表绝对值并覆盖）。删除旧路径。

### 3. Makefile 收敛

`build` 只产出 `bin/pixoma` 与 `bin/pixoma-edge-agent`；删除 `run-admin-api` 目标。

## Risks / Trade-offs

- 遗漏文档引用导致新部署者仍找 admin-api → 用 rg 全仓核对并同步 README / overview / data-model / runtime。
- backfill 配置入口变化 → README 运维节补充 `DATABASE_DSN` / `DB_DRIVER` 用法。

## Migration Plan

1. 迁移 backfill 命令并删除旧入口目录/配置/包。
2. 更新 Makefile 与文档。
3. 全量 `go build ./...` + `go test ./...` 验证无残留引用。

## Open Questions

无。
