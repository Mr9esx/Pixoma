# Brainstorm Summary

- Change: remove-legacy-apps
- Date: 2026-08-22

## 已确认事实

- 新部署只保留 `pixoma` + `pixoma-edge-agent`；`apps/bot`（旧拆分 bot）与 `apps/admin-api`（过渡期管理 HTTP）无运行引用。
- `internal/platform/adminconfig` 唯一消费者是 admin-api；`bin/` 为 gitignore 产物。
- backfill 命令位于 admin-api 下，需随迁移保留；`adminhost` 与 `botapp` 门面继续由 pixoma 使用。

## 候选方案（推荐 1）

1. 整目录删除 + backfill 迁移到 `apps/pixoma/cmd/backfill-task-stats`（环境变量 DSN）——推荐。
2. 保留目录仅停用构建——仍需维护，易误用。
3. 仅删 bot、保留 admin-api——未收敛到新形态，不推荐。

## 关键取舍与风险

- 文档残留引用 → 全仓 rg 核对并同步 README / overview / data-model / runtime。
- backfill 配置入口变化 → README 运维节补充 `DATABASE_DSN` / `DB_DRIVER`。

## 测试策略

- `go build ./...` + `go test ./...` 全绿且无被删包引用；`rg apps/admin-api|apps/bot` 仅剩历史归档/报告。
- backfill 冒烟：临时库 seed 终态任务重算统计表。

## Spec Patch

无（open 阶段 delta 已定稿：admin-api-host REMOVED + MODIFIED）。
