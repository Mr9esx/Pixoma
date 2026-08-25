## Context

现状（动机见 `proposal.md`）：

- `internal/platform/db` 的 `Open` 已支持 sqlite/mysql/postgres 三种 dialector，`settings.Settings` 也校验三种驱动；Setup 向导后端（`POST /api/v1/setup/database`）与前端数据库步骤已可选 MySQL/Postgres。
- 设置页 `GET/PUT /api/v1/setup/settings` 只读展示 DB，`putSettings` 强制把 `DBDriver/DBDSN` 覆盖为引导态值；业务库连接只在 Setup 向导（`POST /api/v1/setup/database`）配置，本期维持该边界。
- 启动期 `RenameLegacy`、全模型 `AutoMigrate`、`MigrateLegacyTasks`、统计 upsert 尚未在 MySQL/Postgres 上做过端到端验证；`drivers_integration_test.go` 仅 ping + 单表 migrate，且依赖环境变量。

## Goals / Non-Goals

**Goals:**

- Setup 向导数据库步骤保持可选 MySQL/Postgres，并补齐 DSN 示例与可诊断错误。
- MySQL/Postgres 下启动装配（迁移、修复、统计、任务对账）端到端可用；用集成测试锁定。
- 设置页保持只读展示业务库驱动与 DSN，不提供修改/切换入口。

**Non-Goals:**

- 不做业务库切换/换库，不做跨引擎数据迁移（迁移场景走数据迁移 + 重跑 Setup）。
- 不把 bootstrap 引导库迁到 MySQL/Postgres（保持本地 SQLite 信任边界）。
- 不为控制面新增 `DB_DRIVER/DATABASE_DSN` 环境变量覆盖（保持 backfill 专用）。
- 不做连接池、读写分离、集群等高阶数据库运维配置。

## Decisions

### D1：Setup 向导为唯一配置入口，设置页只读

业务库连接只在 Setup 向导数据库步骤配置（现状已具备 driver 下拉 + DSN + 测连通，`POST /api/v1/setup/database` 校验并写引导态）；设置页 account tab 维持只读展示 `db_driver`/`db_dsn`，不新增编辑/切换入口。避免为低频、高风险的换库场景引入「设置存业务库、空库无法启动」的引导死锁处理。

向导数据库步骤按 driver 提供**结构化连接表单**：SQLite 为数据库文件路径；MySQL 为 Host/端口/用户/密码/数据库；Postgres 为 Host/端口/用户/密码/数据库/SSL 模式。前端用纯函数按字段组装连接字符串（GORM 兼容格式），密码字段不回显；「高级：直接输入 DSN」折叠项允许覆盖组装结果。切换驱动时端口默认值自动切换（3306↔5432），字段值保留可编辑。

### D2：MySQL/Postgres 启动链路兼容策略

- `RenameLegacy`：legacy 重命名仅当检测到旧表/旧列时执行；全新 MySQL/Postgres 库均为 no-op，保留现状，不引入 driver 分支（降低兼容面）。
- 全模型 `AutoMigrate`：在集成测试中对全部业务模型（cases/users/sessions/tasks/topics/edges/metrics/channels/menu/settings/stats）跑一遍，锁定列类型与索引长度兼容性。
- `MigrateLegacyTasks` 中把 `lease_until`/`requeue_at` 重置为 `time.Time{}` 的写法改为写 `NULL`：MySQL 严格模式拒绝 `0000-00-00`，零值 time.Time 会触发该问题；Postgres 无此限制但统一为 NULL 更安全。
- `RequeueExpiredLeases` 的 `lease_until != time.Time{}` 条件改写为 `lease_until IS NOT NULL`：SQL 中 `NULL != value` 结果为 NULL，原条件在可空列上语义不正确，改写后语义等价且方言安全。
- 统计 upsert（`clause.OnConflict`）与任务 claim 更新均为标准 SQL，MySQL/Postgres 通用，不做改动，仅测试覆盖。

### D3：集成测试环境

`drivers_integration_test.go` 以 `PIXOMA_MYSQL_DSN` / `PIXOMA_POSTGRES_DSN` 门控（缺省 skip，保持现状）；覆盖全业务模型 AutoMigrate、settings/Topic/Case/Task 核心读写 roundtrip 与启动期修复逻辑 no-op 验证。本机无 docker，不引入 compose；本地一键起库作为后续可选。

## Risks / Trade-offs

- [MySQL 严格模式与零值时间] → `MigrateLegacyTasks` 与同类更新统一写 NULL；集成测试在 MySQL 上覆盖该路径。
- [换库场景缺失] → 本期明确非目标；README 与架构文档说明「迁移走数据迁移 + 重跑 Setup」。
- [MySQL 8.0 以下不支持 `RENAME COLUMN`] → 该语句只在检测到旧 SQLite schema 时才执行，新 MySQL/PG 库不触发；文档标注 MySQL 建议 8.0+。

## Migration Plan

- 无破坏性 schema 变更；改动为 Setup 向导文案/占位、启动期兼容修正与集成测试。
- 部署：新部署在 Setup 向导中选库；既有部署不受影响（业务库连接不变）。
- 回滚：恢复上一版本二进制即可；引导态连接未变时行为与旧版一致。

## Open Questions

- MySQL 版本下限（建议 8.0+）与 MariaDB 兼容性：集成测试以 MySQL 8 为准，是否需兼容 MariaDB 可在 verify 阶段再确认，不影响本 change 的规格与任务拆分。
