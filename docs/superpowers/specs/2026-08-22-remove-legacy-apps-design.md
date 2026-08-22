---
comet_change: remove-legacy-apps
role: technical-design
canonical_spec: openspec
archived-with: 2026-08-22-remove-legacy-apps
status: final
---

# remove-legacy-apps 深度技术设计

## 1. 目标与范围

删除旧部署入口 `apps/bot`（旧拆分 bot）与 `apps/admin-api`（过渡期独立管理 HTTP）及其专属配置与 `internal/platform/adminconfig`；Makefile 只构建 `pixoma` / `pixoma-edge-agent`；backfill 运维命令迁移到 `apps/pixoma/cmd/backfill-task-stats`；同步文档。新部署只保留控制面 `pixoma` + 执行面 `pixoma-edge-agent`。

非目标：不动 `internal/httpapi/adminhost`（pixoma 使用）、`internal/packaging/botapp`（领域门面，被 pixoma 与渠道 capability 使用）、pixoma 管理 API 行为、`web/admin` 前端、历史归档文档与旧 verify 报告。

## 2. 影响面审计（无功能影响）

只读审计结论（证据见 brainstorming 记录）：

- 存活代码中 import `apps/bot` / `apps/admin-api` 的引用为 0（排除被删目录后）。
- `internal/platform/adminconfig` 无其它消费者；`.github/workflows` 与 `scripts/` 无引用；仅 Makefile 的 `build` 与 `run-admin-api` 目标引用。
- `internal/packaging/botapp` 仍被 `apps/pixoma/internal/app/telegram.go` 与 `internal/channel/capability/open_case.go` 使用，保留。
- `scripts/dev.sh` 只启动 `pixoma` + Vite，不依赖被删入口。

因此删除不会破坏现有构建、运行、测试或前端链路。

## 3. 删除清单与迁移方案

| 删除项 | 迁移方案 |
|---|---|
| `apps/bot/` | bot 运行时已内嵌 pixoma；无外部引用 |
| `apps/admin-api/`（含 `internal/server` 包装） | 管理 HTTP 由 pixoma（adminhost）提供，接口面与门禁不变；`make dev` 本就只跑 pixoma + Vite |
| `internal/platform/adminconfig/` | 无其它消费者；backfill 改用环境变量 |
| `configs/admin-api.yaml` / `configs/admin-api.example.yaml` | 随 admin-api 删除，无其它读取方 |
| Makefile `bin/admin-api` 构建与 `run-admin-api` 目标 | 删除；`build` 只产出 pixoma / pixoma-edge-agent |
| `apps/admin-api/cmd/backfill-task-stats` | 迁移至 `apps/pixoma/cmd/backfill-task-stats`，配置改为 `DB_DRIVER`（默认 sqlite）/ `DATABASE_DSN` / `DATA_DIR`；`DATA_DIR` 语义与 pixoma 一致，`DATABASE_DSN` 直接兼容旧值；行为不变 |

## 4. backfill 迁移实现

`apps/pixoma/cmd/backfill-task-stats/main.go`：

- `DB_DRIVER` 环境变量（默认 `sqlite`），`DATABASE_DSN` 缺省为 `DATA_DIR/app.db`（`DATA_DIR` 缺省 `data`）。
- 复用 `internal/platform/db.Open` / `appboot` 式 AutoMigrate（四张统计表 + `tasks`）。
- 逻辑不变：按 `STATS_TIMEZONE` 归天，遍历终态任务重算 daily / edge / error / case 四表绝对值并覆盖，最后 Prune。
- 删除旧 `apps/admin-api/cmd/backfill-task-stats` 与 `internal/platform/adminconfig` 依赖。

## 5. 文档同步

- README：服务表改为 pixoma / pixoma-edge-agent；环境变量表移除 admin-api 相关行，补充 `DB_DRIVER` / `DATABASE_DSN`（backfill）；删除 `make run-admin-api` 描述。
- `docs/architecture/overview.md`：去掉"过渡期独立管理 HTTP"行与 bot 拆分形态。
- `docs/architecture/data-model.md`：`相关代码入口` 表把 admin-api 行改为 pixoma / backfill。
- `docs/architecture/runtime.md`：确认无独立 admin-api 描述残留（若有则更新）。

## 6. 测试策略

- `go build ./...` + `go test ./...` 全绿，且不包含被删包。
- `rg 'apps/admin-api|apps/bot'` 仅剩历史归档 / 旧 verify 报告（预期残留，不改动）。
- backfill 冒烟：临时库 seed 终态任务 → 运行新命令 → 校验四表聚合正确。

## 7. 风险与回滚

- 文档残留引用 → 第 5 节全仓核对；rg 复检。
- 有人仍依赖 `bin/admin-api` → 迁移方案已给出（pixoma 提供同一管理 HTTP）；README 标注。
- 回滚：从 git 恢复被删目录与 Makefile 目标即可（无数据迁移，纯代码/配置删除）。
