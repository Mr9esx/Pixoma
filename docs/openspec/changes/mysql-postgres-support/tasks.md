## 1. MySQL/Postgres 启动链路兼容

- [x] 1.1 Task 仓储 `PrepareForClaim` / `RequeueExpiredLeases` / `MigrateLegacyTasks` 中 `lease_until`、`requeue_at` 的零值 `time.Time{}` 改为写 NULL（`gorm.Expr("NULL")`）
- [x] 1.2 `RequeueExpiredLeases` 的 `lease_until != ?` 条件改写为 `lease_until IS NOT NULL AND lease_until < ?`，并补单测验证（SQLite 行为等价回归）

## 2. Setup 数据库配置

- [x] 2.1 Setup 向导数据库步骤按 driver 切换 DSN 占位/示例（sqlite/mysql/postgres）
- [x] 2.2 合同测试：Setup 向导数据库步骤包含三驱动选项、对应 DSN 占位与可诊断错误提示
- [x] 2.3 切换驱动时若 DSN 仍为默认/示例值则同步替换为所选驱动示例连接（自定义 DSN 不覆盖），合同测试锁定

## 3. 设置页只读展示

- [x] 3.1 合同测试：设置页只读展示业务库驱动与 DSN，且不存在编辑/切换控件（保持现状行为）

## 4. MySQL/Postgres 集成测试

- [x] 4.1 扩展 `drivers_integration_test.go`：MySQL/Postgres 上全业务模型 AutoMigrate + settings/Topic/Case/Task 核心读写 roundtrip（`PIXOMA_MYSQL_DSN` / `PIXOMA_POSTGRES_DSN` 门控）
- [x] 4.2 在 MySQL/Postgres 上验证启动链路：`RenameLegacy` no-op、遗留表 `DropTable`、统计 upsert、`MigrateLegacyTasks`

## 5. 文档与验证

- [x] 5.1 README 与 `docs/architecture/data-model.md` 更新多库说明：Setup 配置入口、MySQL 8.0+ 建议、换库为非目标
- [x] 5.2 `go build ./...` + `go test ./...`（含 integration tags）；`pnpm tsc -b` + `pnpm vitest run`（web/admin）

## 代码审查记录（review_mode: standard）

- 审查方式：内联轻量审查（reviewer subagent 三次派发均因消息投递失败未收到任务，已记录降级原因）。
- 范围：`88eb0fa..de9bd62` 全部实现 diff。
- 结论：实现与设计文档/tasks 对齐（无换库、设置页保持只读）；零值时间 NULL 化三处一致，`lease_until IS NOT NULL` 条件与 `ClaimNextWithLease` 的 `requeue_at IS NULL OR requeue_at <= ?` 语义兼容；测试验证真实 DB 行为（`*time.Time` 扫描 NULL 断言）；集成测试 env 门控、覆盖全模型迁移与核心链路。未发现 Critical/Important 问题。
- 接受的小项（Minor，不影响交付）：`DB_DSN_PLACEHOLDER[driver]` 的回退值为 sqlite 占位（driver 由 Select 约束，实际恒合法）；合同测试 `mysql:`/`postgres:` 断言略宽，但与 `DB_DSN_PLACEHOLDER` 断言组合后足够精确。接受原因：均为防御性写法，不改变行为。
