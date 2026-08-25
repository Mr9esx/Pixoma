## Why

平台业务库目前默认 SQLite，向导里已预留 MySQL/Postgres 选项且 ORM 层支持三种驱动，但 MySQL/Postgres 下的启动装配与关键链路缺少端到端验证，向导数据库步骤也缺少驱动对应的连接示例与可诊断错误文案。多库支持停留在半成品，无法形成完整可交付能力。

## What Changes

- **Setup 向导数据库步骤完善**：保留已有 sqlite/mysql/postgres 选项，补齐驱动对应的 DSN 示例/占位与可诊断错误提示，确保新部署可选择 MySQL/Postgres 完成向导。
- **设置页维持只读展示**：设置页继续只读展示业务库驱动与 DSN，本期不提供修改/切换业务库入口（业务库连接只在 Setup 向导配置）。
- **MySQL/Postgres 启动链路兼容加固**：核对并修正启动期迁移/修复逻辑（`RenameLegacy`、全模型 `AutoMigrate`、legacy 迁移、统计 upsert），保证控制面可在 MySQL/Postgres 上启动并完成核心读写；补充 MySQL/Postgres 集成测试覆盖全量迁移与关键链路。
- **文档与规格**：更新 `setup-wizard`、`multi-database-support` 规格，补充架构文档与 README 中多库说明。

## Non-Goals

- 不做业务库切换/换库（设置页只读展示；迁移场景走数据迁移 + 重跑 Setup）。
- 不做跨数据库引擎的数据迁移。
- 不把 bootstrap 引导库改为 MySQL/Postgres（保持本地 SQLite）。
- 不为控制面新增 `DB_DRIVER/DATABASE_DSN` 环境变量覆盖（保持 backfill 专用）。
- 不做连接池、读写分离、集群等高阶数据库运维配置。

## Capabilities

### New Capabilities
- `multi-database-support`: 业务库支持 sqlite/mysql/postgres 三种驱动；Setup 向导支持配置业务库，设置页只读展示业务库信息。

### Modified Capabilities
- `setup-wizard`: 数据库步骤明确支持 MySQL/Postgres，并以可诊断错误呈现连通性/合法性校验。

## Impact

- 后端：`internal/runtime/infrastructure/persistence`（任务对账时间兼容）、`internal/platform/db` 集成测试、`apps/pixoma/cmd/pixoma` 启动装配链路验证。
- 前端：`web/admin` Setup 向导数据库步骤、设置页只读展示契约与合同测试。
- 文档：`setup-wizard`、`multi-database-support` 规格、`docs/architecture/data-model.md`、`README.md`。
