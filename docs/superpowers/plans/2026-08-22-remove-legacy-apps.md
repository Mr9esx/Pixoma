---
change: remove-legacy-apps
design-doc: docs/superpowers/specs/2026-08-22-remove-legacy-apps-design.md
base-ref: dc04ae28c9ee876d0fcf9961afd7092141c915c2
archived-with: 2026-08-22-remove-legacy-apps
---

# remove-legacy-apps Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans or subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax.

**Goal:** 删除旧入口 `apps/bot` 与 `apps/admin-api` 及其配置/依赖，backfill 迁移到 pixoma 命令，Makefile 只构建 pixoma / pixoma-edge-agent，文档同步；保证现有功能不受影响。

**Architecture:** 整目录删除 + backfill 迁移（环境变量 DSN）+ Makefile 收敛 + 文档同步。`adminhost`、`botapp` 门面、pixoma 行为不动。

**Global Constraints:**
- 不删除 `internal/httpapi/adminhost`、`internal/packaging/botapp`、pixoma 管理 API 行为。
- 删除前后 `go build ./...` + `go test ./...` 必须全绿；`rg 'apps/admin-api|apps/bot'` 仅剩历史归档/旧报告。
- backfill 行为不变（四张统计表绝对值重算覆盖），仅入口与配置变化。
- 提交信息用中文，按任务粒度提交。

## Task 1: 迁移 backfill 到 pixoma 命令

**Files:**
- Create: `apps/pixoma/cmd/backfill-task-stats/main.go`
- Delete: `apps/admin-api/cmd/backfill-task-stats/main.go`

**Interfaces:**
- Produces: `go run ./apps/pixoma/cmd/backfill-task-stats`（env：`DB_DRIVER` 默认 sqlite、`DATABASE_DSN` 缺省 `DATA_DIR/app.db`、`DATA_DIR` 缺省 `data`）

- [x] Step 1: 新建 `apps/pixoma/cmd/backfill-task-stats/main.go`

```go
package main

// 读取 DB_DRIVER（默认 sqlite）/ DATABASE_DSN（缺省 DATA_DIR/app.db）/ DATA_DIR（缺省 data），
// 复用 internal/platform/db 与 taskstats persistence；逻辑沿用原 admin-api 版 backfill：
// 按 STATS_TIMEZONE 归天，遍历终态任务重算 task_daily_stats / task_edge_daily_stats /
// task_error_daily_stats / task_case_daily_stats 四表绝对值并覆盖，最后 Prune。
```

> 以现有 `apps/admin-api/cmd/backfill-task-stats/main.go` 为底稿：替换 adminconfig 加载为 env 解析（`resolveDSN` 改为读 `DATABASE_DSN`，否则 `DATA_DIR/app.db`），`appboot.Bootstrap` 的 Driver 用 `DB_DRIVER`，其余（四表聚合、OnConflict 覆盖、Prune）保持不变。

- [x] Step 2: 删除旧 `apps/admin-api/cmd/backfill-task-stats/main.go`（目录随 Task 2 一并移除）
- [x] Step 3: 编译验证

Run: `go build ./apps/pixoma/cmd/backfill-task-stats/...`
Expected: PASS

- [x] Step 4: Commit

```bash
git add apps/pixoma/cmd/backfill-task-stats apps/admin-api/cmd/backfill-task-stats
git commit -m "feat(remove-legacy-apps): backfill 迁移到 pixoma 命令（环境变量 DSN）"
```

## Task 2: 删除旧入口与配置

**Files:**
- Delete: `apps/bot/`
- Delete: `apps/admin-api/`
- Delete: `internal/platform/adminconfig/`
- Delete: `configs/admin-api.yaml`、`configs/admin-api.example.yaml`

- [x] Step 1: 用 apply_patch 逐文件删除（或 git rm）上述目录与文件，保留 `internal/httpapi/adminhost` 与 `internal/packaging/botapp`
- [x] Step 2: 全仓引用核对

Run: `rg -n 'apps/(bot|admin-api)|platform/adminconfig' --glob '!docs/openspec/changes/archive/**' --glob '!docs/superpowers/reports/**'`
Expected: 仅剩 README 与架构文档的待更新描述（Task 4 处理）与历史归档

- [x] Step 3: Commit

```bash
git add -A apps/bot apps/admin-api internal/platform/adminconfig configs
git commit -m "refactor(remove-legacy-apps): 删除旧 bot 与 admin-api 入口及配置"
```

## Task 3: Makefile 收敛

**Files:**
- Modify: `Makefile`

- [x] Step 1: `build` 删除 admin-api 行，只保留 pixoma / pixoma-edge-agent；`.PHONY` 移除 `run-admin-api`；删除 `run-admin-api` 目标
- [x] Step 2: 验证

Run: `make build`
Expected: 产出 `bin/pixoma` 与 `bin/pixoma-edge-agent`，无 admin-api

- [x] Step 3: Commit

```bash
git add Makefile
git commit -m "chore(remove-legacy-apps): Makefile 只构建 pixoma 与 edge-agent"
```

## Task 4: 文档同步

**Files:**
- Modify: `README.md`
- Modify: `docs/architecture/overview.md`
- Modify: `docs/architecture/data-model.md`
- Modify: `docs/architecture/runtime.md`

- [x] Step 1: README 服务表改为 pixoma / pixoma-edge-agent；移除 admin-api 过渡期描述与 `run-admin-api`；环境变量表补充 backfill 的 `DB_DRIVER` / `DATABASE_DSN` / `DATA_DIR`
- [x] Step 2: overview.md 去掉"过渡期独立管理 HTTP"行；data-model.md `相关代码入口` 表改为 pixoma / backfill；runtime.md 核对无独立 admin-api 残留
- [x] Step 3: 复核

Run: `rg -n '独立.*admin-api|make run-admin-api|bin/admin-api' README.md docs/architecture`
Expected: 无残留（历史归档除外）

- [x] Step 4: Commit

```bash
git add README.md docs/architecture
git commit -m "docs(remove-legacy-apps): 同步部署形态与 backfill 用法"
```

## Task 5: 验证

- [x] Step 1: 全量构建与测试

Run: `go build ./... && go test ./...`
Expected: 全绿

- [x] Step 2: 残留核对

Run: `rg -n 'apps/admin-api|apps/bot|comfyui-bot' --glob '!docs/openspec/changes/archive/**' --glob '!docs/superpowers/reports/**'`
Expected: 无代码/配置引用（历史文档除外）

- [x] Step 3: backfill 冒烟（临时库）

Run: 临时库 seed 终态任务 + 指标 → `DB_DRIVER=sqlite DATABASE_DSN=file:... go run ./apps/pixoma/cmd/backfill-task-stats` → 校验四表
Expected: 聚合正确

- [x] Step 4: Commit（如有修复）

```bash
git add -A
git commit -m "chore(remove-legacy-apps): 全量验证通过"
```
