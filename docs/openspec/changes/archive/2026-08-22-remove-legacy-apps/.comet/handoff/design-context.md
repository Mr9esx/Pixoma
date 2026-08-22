# Comet Design Handoff

- Change: remove-legacy-apps
- Phase: design
- Mode: compact
- Context hash: b732d1a3183885b13e4379f505ad33e32fc48b015fabd1fe318b2a66a12057a5

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## docs/openspec/changes/remove-legacy-apps/proposal.md

- Source: docs/openspec/changes/remove-legacy-apps/proposal.md
- Lines: 1-28
- SHA256: 64433073a9c3d553a0db22f60afcda663ff185b3fc70ff3f9803e367eb4cc440

```md
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

```

## docs/openspec/changes/remove-legacy-apps/design.md

- Source: docs/openspec/changes/remove-legacy-apps/design.md
- Lines: 1-44
- SHA256: d21591e12d6adf4adb54eeff909b05c833748b1dee1a04c3b0de3ab2113b6dff

```md
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

```

## docs/openspec/changes/remove-legacy-apps/tasks.md

- Source: docs/openspec/changes/remove-legacy-apps/tasks.md
- Lines: 1-17
- SHA256: 15cd389fb952076b42e7449d866755c043509d8b19d4b40f30ba09002cacf433

```md
## 1. 代码与构建清理

- [ ] 1.1 迁移 backfill 至 `apps/pixoma/cmd/backfill-task-stats`（环境变量 DSN：`DB_DRIVER` / `DATABASE_DSN` / `DATA_DIR`）
- [ ] 1.2 删除 `apps/bot`、`apps/admin-api` 目录
- [ ] 1.3 删除 `internal/platform/adminconfig` 与 `configs/admin-api.yaml`、`configs/admin-api.example.yaml`
- [ ] 1.4 Makefile：`build` 仅产出 pixoma / pixoma-edge-agent，删除 `run-admin-api`

## 2. 文档同步

- [ ] 2.1 README：服务表与环境变量表移除 admin-api 过渡期描述，补充 backfill 环境变量
- [ ] 2.2 `docs/architecture/overview.md` / `data-model.md` / `runtime.md` 去掉独立 admin-api / bot 入口描述

## 3. 验证

- [ ] 3.1 `go build ./...` + `go test ./...` 通过且无被删包引用
- [ ] 3.2 `rg apps/admin-api|apps/bot` 仅剩历史归档/报告文档
- [ ] 3.3 backfill 冒烟：临时库 seed 终态任务后重算统计表正确

```

## docs/openspec/changes/remove-legacy-apps/specs/admin-api-host/spec.md

- Source: docs/openspec/changes/remove-legacy-apps/specs/admin-api-host/spec.md
- Lines: 1-22
- SHA256: d4f02c72b51c2eb19b19746483d1ea72bcffb356b601339f0e46630482c6eda1

```md
## REMOVED Requirements

### Requirement: 独立 admin-api 进程可启动并提供健康检查
**Reason**: 过渡期结束，独立 `admin-api` 二进制与 `apps/admin-api` 入口已删除；健康检查与管理 API 由 `pixoma` 一体托管。
**Migration**: 使用 `pixoma` 的 `/healthz` 与管理 API；`make run-admin-api` / `bin/admin-api` 不再存在。

### Requirement: 管理 HTTP 无鉴权（本期）
**Reason**: 独立 admin-api 删除后管理 HTTP 统一走 `pixoma` 门禁，初始化后要求管理员会话，不再存在无鉴权管理入口。
**Migration**: 使用已初始化平台的登录与管理员会话访问管理 API。

## MODIFIED Requirements

### Requirement: 控制面可一体托管管理 API
`pixoma` 控制面入口 MUST 托管管理 HTTP 能力（健康检查、向导、管理 API、Agent API 同进程）；独立 `admin-api` 二进制已移除，新部署 MUST NOT 依赖或启动独立管理进程。

#### Scenario: 一体入口提供健康检查与管理 API
- **WHEN** 用户仅启动 `pixoma` 且已初始化
- **THEN** 可通过该进程提供的地址访问健康检查与管理 API

#### Scenario: 无独立管理进程可启动
- **WHEN** 运维查找 `apps/admin-api` 或 `bin/admin-api`
- **THEN** 该入口不存在，文档与 Makefile 不再提供独立 admin-api 构建或运行目标

```
