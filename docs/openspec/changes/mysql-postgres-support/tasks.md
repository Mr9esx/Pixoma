## 1. MySQL/Postgres 启动链路兼容

- [ ] 1.1 Task 仓储 `PrepareForClaim` / `RequeueExpiredLeases` / `MigrateLegacyTasks` 中 `lease_until`、`requeue_at` 的零值 `time.Time{}` 改为写 NULL（`gorm.Expr("NULL")`）
- [ ] 1.2 `RequeueExpiredLeases` 的 `lease_until != ?` 条件改写为 `lease_until IS NOT NULL AND lease_until < ?`，并补单测验证（SQLite 行为等价回归）

## 2. Setup 数据库配置

- [ ] 2.1 Setup 向导数据库步骤按 driver 切换 DSN 占位/示例（sqlite/mysql/postgres）
- [ ] 2.2 合同测试：Setup 向导数据库步骤包含三驱动选项、对应 DSN 占位与可诊断错误提示

## 3. 设置页只读展示

- [ ] 3.1 合同测试：设置页只读展示业务库驱动与 DSN，且不存在编辑/切换控件（保持现状行为）

## 4. MySQL/Postgres 集成测试

- [ ] 4.1 扩展 `drivers_integration_test.go`：MySQL/Postgres 上全业务模型 AutoMigrate + settings/Topic/Case/Task 核心读写 roundtrip（`PIXOMA_MYSQL_DSN` / `PIXOMA_POSTGRES_DSN` 门控）
- [ ] 4.2 在 MySQL/Postgres 上验证启动链路：`RenameLegacy` no-op、遗留表 `DropTable`、统计 upsert、`MigrateLegacyTasks`

## 5. 文档与验证

- [ ] 5.1 README 与 `docs/architecture/data-model.md` 更新多库说明：Setup 配置入口、MySQL 8.0+ 建议、换库为非目标
- [ ] 5.2 `go build ./...` + `go test ./...`（含 integration tags）；`pnpm tsc -b` + `pnpm vitest run`（web/admin）
