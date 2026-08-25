---
comet_change: mysql-postgres-support
role: technical-design
canonical_spec: openspec
archived-with: 2026-08-25-mysql-postgres-support
status: final
---

# mysql-postgres-support 深度设计

## 背景与范围

目标（详见 `docs/openspec/changes/mysql-postgres-support/proposal.md`）：业务库完整支持 SQLite/MySQL/Postgres，Setup 向导支持配置业务库，设置页只读展示业务库信息。

现状约束：

- `internal/platform/db.Open` 已支持三种 dialector；`settings.Settings.Validate` 已校验三驱动；Setup 向导前后端已可选 MySQL/Postgres（`POST /api/v1/setup/database` 校验连接并写引导态）。
- 设置页 `GET/PUT /api/v1/setup/settings` 只读展示 DB，`putSettings` 强制把 `DBDriver/DBDSN` 覆盖为引导态值。本期维持该边界，不做换库。
- Task 仓储多处把 `time.Time{}` 写入 `Updates` map（`PrepareForClaim`、`RequeueExpiredLeases`、`MigrateLegacyTasks`），MySQL 严格模式拒绝零值 datetime；`RequeueExpiredLeases` 用 `lease_until != time.Time{}` 比较可空列，语义有误。
- `drivers_integration_test.go` 仅 ping + 单表 migrate，MySQL/Postgres 上的全量迁移与核心读写未验证。

## 目标 / 非目标

**目标**

- Setup 向导数据库步骤明确支持三驱动，补齐 DSN 示例与可诊断错误。
- MySQL/Postgres 下启动装配（迁移、修复、统计、任务对账）端到端可用，用集成测试锁定。
- 设置页保持只读展示业务库驱动与 DSN，不提供修改/切换入口。

**非目标**

- 业务库切换/换库与跨引擎数据迁移（迁移场景走数据迁移 + 重跑 Setup）。
- bootstrap 引导库迁移到 MySQL/Postgres（保持本地 SQLite 信任边界）。
- 控制面新增 `DB_DRIVER/DATABASE_DSN` 环境变量覆盖（backfill 专用不变）。
- 连接池、读写分离、集群等高阶运维配置。

## 技术方案

### 1. 零值时间 NULL 化（Task 仓储）

`internal/runtime/infrastructure/persistence/gorm_task.go` 中以下位置从 `time.Time{}` 改为写 `gorm.Expr("NULL")`：

- `PrepareForClaim`：`lease_until`、`requeue_at`
- `RequeueExpiredLeases`：`lease_until`
- `MigrateLegacyTasks`：`lease_until`、`requeue_at`

条件改写：

```go
// 旧
Where("status = ? AND lease_until != ? AND lease_until < ?", status, time.Time{}, now)
// 新
Where("status = ? AND lease_until IS NOT NULL AND lease_until < ?", status, now)
```

理由：MySQL 严格模式拒绝 `0000-00-00` 写入；`NULL != value` 在 SQL 中结果为 NULL，原条件语义不正确。行模型字段保持 `time.Time`（可空列），不做指针化改造，避免扩散。`domain` 层对 `LeaseUntil/RequeueAt` 的零值赋值保持不变（内存语义），仅在仓储落库时归一为 NULL。

### 2. Setup 向导数据库步骤

- 保持现有流程：driver Select（sqlite/mysql/postgres）+ DSN Input + 测连通（`POST /api/v1/setup/database`）。
- 数据库步骤改为**结构化连接表单**（不再让用户手拼一行 DSN）：
  - SQLite：数据库文件路径（默认 `data/app.db`）。
  - MySQL：Host / 端口（默认 3306）/ 用户 / 密码 / 数据库；组装为 `user:password@tcp(host:port)/db?charset=utf8mb4&parseTime=True&loc=Local`。
  - Postgres：Host / 端口（默认 5432）/ 用户 / 密码 / 数据库 / SSL 模式（disable/require/prefer）；组装为 `host=... port=... user=... password=... dbname=... sslmode=...`。
  - 密码字段 `type=password` 不回显；每个服务型驱动提供「附加参数」输入框（MySQL 示例 `timeout=5s&readTimeout=10s`，Postgres 示例 `connect_timeout=10 application_name=pixoma`），直接追加到组装结果，不提供整串 DSN 编辑入口。
  - 切换驱动时端口默认值自动切换（3306↔5432），已填字段保留可编辑；组装逻辑放在纯函数模块 `db-dsn.ts`（可单测）。
- 错误展示用 `Alert variant="destructive"`：`db-error.ts` 把常见与边界数据库错误（拒绝连接/鉴权失败/库不存在/超时/DNS 与路由不可达/连接中断/连接数满/文件锁与磁盘/死锁/角色与表缺失/SSL-TLS/未知驱动/缺 DSN/登录与密码/向导步骤前置/远程 localfs/代理/未授权等约 28 类）映射为中文友好标题，AlertDescription 展示实际错误详情；未知错误回退「操作失败，请重试」。
- 数据库步骤拆两个按钮：「连通性测试」调 `POST /api/v1/setup/database` 并显示结果（通过以 success Alert 显示「连接正常」并带 `CircleCheck` 图标，失败以 destructive Alert 显示错误），「继续」始终可点；连接配置变更后 `dbTested` 自动失效。`Alert` 组件新增 `success` 变体（emerald 系）。
- 连通性与自动建库：`internal/platform/db.EnsureDatabase` 先连服务器验证可达与账号密码（MySQL 用 `mysql.ParseDSN` 去掉库名后连；Postgres 用 `pgx.ParseConfig` 连维护库 `postgres`→`template1`），目标库缺失时自动创建（标识符正确转义），再连目标库；SQLite no-op。业务表由向导保存/启动 AutoMigrate 自动创建。`setup.Handler.openDB` 统一先 ensure 再 open。「继续」不要求先测连通（无 `submitDisabled` 门控），问题在后续步骤以 Alert 报错。
- 错误文案可诊断：非法驱动、空 DSN、连接失败均返回具体原因（后端已具备，前端透传）。

### 3. 设置页只读展示

- account tab 数据库节维持只读展示 `db_driver` / `db_dsn`（现状已如此），不新增编辑/切换控件。
- 用合同测试锁定该行为：页面展示驱动与 DSN 文本，且不存在可编辑控件或切换入口。

### 4. 集成测试

`internal/platform/db/drivers_integration_test.go`（`//go:build integration`）扩展：

- `PIXOMA_MYSQL_DSN` / `PIXOMA_POSTGRES_DSN` 门控（缺省 skip，保持现状）。
- 全业务模型 AutoMigrate：cases/users/sessions/tasks/topics/edges/metrics/channels/menu/settings/stats。
- 核心读写 roundtrip：`settings.Store` Save/Load、Topic 创建/查询、Case 创建/Get、Task Create → PrepareForClaim → ClaimNextWithLease → 终态 Update。
- 启动链路：`db.RenameLegacy` no-op、`DropTable` 遗留表、统计 upsert（`clause.OnConflict`）、`MigrateLegacyTasks`。

不引入 docker-compose（本机无 docker）；如需本地一键起库，作为后续可选小项。

## 数据流：首次部署装配

```text
管理员（Setup 向导）
  → 数据库步骤选驱动 + 填 DSN → POST /api/v1/setup/database
  → 校验 + 打开目标库 + ping（失败 → 可诊断错误，不改引导态）
  → SetAppDB(driver, dsn) + 向导步骤推进
  → 后续步骤保存平台设置到业务库（AutoMigrate platform_settings）
  → finalize → MarkInitialized + restart_required
  → pixoma 重启 → loadSavedSettings(业务库) → 全模型装配
```

## 边界条件

- 未知驱动/空 DSN/不可达/鉴权失败/超时 → 向导报可诊断错误，引导态不变。
- SQLite DSN 相对路径：父目录确保存在；MySQL/PG 不做目录处理。
- 设置页对业务库信息只读；`PUT /api/v1/setup/settings` 继续忽略 `db_driver`/`db_dsn`。
- MySQL 8.0+ 建议：`RenameLegacy` 的 `RENAME COLUMN` 只在旧 SQLite schema 上触发，新库 no-op。

## 测试策略

- Go 单测：Task 仓储零值时间写 NULL 与 `IS NOT NULL` 条件（SQLite 上行为等价，回归保护）；`setup` handler 现有测试保持（`putSettings` DB 只读断言保留）。
- 前端合同测试（`web/admin`）：设置页只读展示业务库信息且无编辑控件；Setup 向导三驱动选项与 DSN 占位。
- 集成测试：env-gated MySQL/PG 全量迁移 + 核心读写。
- 回归：`go test ./...`、`pnpm vitest run`（web/admin）全绿。

## 风险与缓解

- [MySQL 严格模式零值时间] → 统一 NULL + 条件重写；集成测试覆盖。
- [换库场景缺失] → 本期明确非目标；README 与架构文档说明「迁移走数据迁移 + 重跑 Setup」。
- [MySQL 8.0 以下 RENAME COLUMN 不支持] → 仅旧 SQLite schema 触发；文档标注 MySQL 8.0+。

## 交付清单

- `internal/runtime/infrastructure/persistence/gorm_task.go`：零值时间 NULL 化 + 条件重写 + 单测。
- `web/admin/src/features/setup/setup-wizard.tsx`：DSN 占位/示例。
- `web/admin/src/features/settings/settings-page.tsx` 相关合同测试：只读展示锁定。
- `internal/platform/db/drivers_integration_test.go`：全模型集成测试。
- README / `docs/architecture/data-model.md`：多库配置说明与换库语义（非目标）说明。
