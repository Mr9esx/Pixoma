# 验证报告：remove-legacy-apps

- 日期：2026-08-22
- 验证模式：full

## Summary

| 维度 | 状态 |
|---|---|
| Completeness | 8/8 任务完成 |
| Correctness | 删除与迁移无功能影响；backfill 行为等价 |
| Coherence | 与 design（影响面审计 + 迁移方案）一致 |

## 检查项

1. **tasks.md 全部完成**：8/8 `[x]`。
2. **删除清单落地**：`apps/bot`、`apps/admin-api`、`internal/platform/adminconfig`、`configs/admin-api.yaml` / `configs/admin-api.example.yaml` 已删除；Makefile 只构建 `pixoma` / `pixoma-edge-agent`。
3. **backfill 迁移**：`apps/pixoma/cmd/backfill-task-stats`（env：`DB_DRIVER` / `DATABASE_DSN` / `DATA_DIR`）；DSN 解析单测 3 例通过；临时库冒烟：3 条终态任务 → daily（1/1/1、queue/exec 各 3×1h）、edges 成功率、cases、errors 全部正确。
4. **无功能影响**：存活代码 import `apps/bot` / `apps/admin-api` / `platform/adminconfig` 为 0；`.github` / `scripts` 无引用；`adminhost` 与 `botapp` 门面保留；`go build ./...`、`go test ./...` 全绿。
5. **文档同步**：README（服务表、环境变量表、过渡期描述）、overview、data-model、runtime、bounded-contexts（组合根 pixoma）、system.html 图例已更新；残留引用仅剩历史归档/旧设计文档。
6. **代码审查**：`review_mode: standard`。变更以删除与迁移为主，内联复核：backfill 逻辑与旧实现逐段一致（冒烟验证）、删除路径无外部引用、Makefile/文档一致；无 CRITICAL / IMPORTANT 问题。

## 验证证据

- `go build ./...`：exit 0
- `go test ./...`：exit 0，无 FAIL
- `pnpm tsc -b`：exit 0
- `pnpm vitest run`：全绿
- `make build`：只产出 `bin/pixoma` / `bin/pixoma-edge-agent`
- backfill 冒烟：临时库四表聚合正确

## 备注

- `bin/admin-api` 为历史 gitignore 产物，Makefile 已不再构建；如需彻底清除可手动删除该文件。

## Final Assessment

全部检查通过，无关键问题。Ready for archive。
