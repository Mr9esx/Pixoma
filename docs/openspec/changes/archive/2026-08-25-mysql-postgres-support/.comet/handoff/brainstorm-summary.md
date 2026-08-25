# Brainstorm Summary

- Change: mysql-postgres-support
- Date: 2026-08-25

## 确认的技术方案

- **范围（用户确认收窄）**：不做设置页换库；业务库连接只在 Setup 向导配置，设置页保持只读展示驱动与 DSN。换库/数据迁移为非目标（迁移走数据迁移 + 重跑 Setup）。
- **零值时间**：Task 仓储 `Updates` map 中的 `time.Time{}` 改为 `gorm.Expr("NULL")`；`lease_until != time.Time{}` 条件改写为 `lease_until IS NOT NULL`，兼容 MySQL 严格模式。
- **Setup 向导 UI**：数据库步骤保留三驱动选项，补各驱动 DSN 占位/示例与可诊断错误；设置页以合同测试锁定只读展示。
- **集成测试**：扩展 `drivers_integration_test.go`，`PIXOMA_MYSQL_DSN`/`PIXOMA_POSTGRES_DSN` 环境变量门控，覆盖全模型 AutoMigrate + 核心读写 roundtrip；不引入 docker-compose（本机无 docker），如需本地一键起库可作为后续可选。

## 关键取舍与风险

- MySQL 严格模式拒绝零值 datetime → 统一写 NULL + `IS NOT NULL` 条件重写。
- MySQL 8.0+ `RENAME COLUMN` 只在检测到旧 SQLite schema 时执行，新 MySQL/PG 库 no-op。
- 不做换库 → 迁移场景走数据迁移 + 重跑 Setup，README/架构文档说明。

## 测试策略

- Go 单测：Task 仓储零值时间 NULL 化与 `IS NOT NULL` 条件重写（SQLite 回归等价）。
- 前端合同测试：设置页只读展示业务库信息；Setup 向导三驱动选项与 DSN 占位。
- 集成测试（env-gated）：MySQL/PG 全模型 AutoMigrate + settings/Topic/Case/Task 读写。
- 回归：SQLite 主路径 `go test ./...` + `pnpm vitest run` 全绿。

## Spec Patch

无需额外回写（open 阶段 delta spec 已覆盖验收场景）；如用户在设计确认中提出新边界条件，再回写对应 delta spec。
