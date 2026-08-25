# Comet Design Handoff

- Change: mysql-postgres-support
- Phase: design
- Mode: compact
- Context hash: 80f0977c1a93a5029abd14104751cce0c2070a4fd3766df0fabe4fdd8636c545

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## docs/openspec/changes/mysql-postgres-support/proposal.md

- Source: docs/openspec/changes/mysql-postgres-support/proposal.md
- Lines: 1-32
- SHA256: 57b2a2e094c8b45de5376ad357f475eb29fba14ef2d7760244f31d368180b591

```md
## Why

平台业务库目前默认 SQLite，向导里已预留 MySQL/Postgres 选项且 ORM 层支持三种驱动，但设置页只能只读查看数据库连接，初始化后无法在界面上切换业务库；换库时还存在「设置存在业务库里、空库无法启动」的引导死锁。多库支持停留在半成品，无法形成完整可交付能力。

## What Changes

- **设置页支持配置/切换业务库**：设置页新增数据库配置入口（驱动 + DSN + 测连通），保存时校验目标库可达、把当前平台设置预置到目标库、更新引导态业务库连接并标记重启；重启后按新库装配（**BREAKING**：设置页的 `PUT /api/v1/setup/settings` 不再忽略 `db_driver`/`db_dsn`，而是按换库流程处理）。
- **Setup 向导数据库步骤完善**：保留已有 sqlite/mysql/postgres 选项，补齐驱动对应的 DSN 示例/占位与可诊断错误提示，确保新部署可选择 MySQL/Postgres 完成向导。
- **MySQL/Postgres 启动链路兼容加固**：核对并修正启动期迁移/修复逻辑（`RenameLegacy`、全模型 `AutoMigrate`、legacy 迁移、统计 upsert），保证控制面可在 MySQL/Postgres 上启动并完成核心读写；补充 MySQL/Postgres 集成测试覆盖全量迁移与关键链路。
- **文档与规格**：更新 `setup-wizard`、`platform-bootstrap` 规格，补充架构文档与 README 中多库说明。

## Non-Goals

- 不做跨数据库引擎的数据迁移（换库指向已有或全新库，旧库数据保留不动）。
- 不把 bootstrap 引导库改为 MySQL/Postgres（保持本地 SQLite）。
- 不为控制面新增 `DB_DRIVER/DATABASE_DSN` 环境变量覆盖（保持 backfill 专用）。
- 不做连接池、读写分离、集群等高阶数据库运维配置。

## Capabilities

### New Capabilities
- `multi-database-support`: 业务库支持 sqlite/mysql/postgres 三种驱动；Setup 向导与设置页均支持配置并安全切换业务库。

### Modified Capabilities
- `setup-wizard`: 数据库步骤明确支持 MySQL/Postgres，并以可诊断错误呈现连通性/合法性校验。
- `platform-bootstrap`: 引导态保存业务库连接信息；已初始化后业务库连接可在设置页修改并触发重启装配。

## Impact

- 后端：`internal/platform/db`、`internal/platform/settings`、`internal/platform/bootstrap`、`internal/httpapi/setup`、`apps/pixoma/cmd/pixoma` 启动装配链路与集成测试。
- 前端：`web/admin` 设置页数据库配置、Setup 向导数据库步骤、`lib/api/setup.ts` 契约与合同测试。
- 文档：`setup-wizard`、`platform-bootstrap` 规格、`docs/architecture/data-model.md`、`README.md`。

```

## docs/openspec/changes/mysql-postgres-support/design.md

- Source: docs/openspec/changes/mysql-postgres-support/design.md
- Lines: 1-73
- SHA256: 06497557cf31b3c54ce5a01ba71e4830a3a78dcb2b53e8474cd269ae614e32c3

```md
## Context

现状（动机见 `proposal.md`）：

- `internal/platform/db` 的 `Open` 已支持 sqlite/mysql/postgres 三种 dialector，`settings.Settings` 也校验三种驱动；Setup 向导后端（`POST /api/v1/setup/database`）与前端数据库步骤已可选 MySQL/Postgres。
- 设置页 `GET/PUT /api/v1/setup/settings` 只读展示 DB，`putSettings` 强制把 `DBDriver/DBDSN` 覆盖为引导态值，无法换库。
- 平台设置行存在业务库 `platform_settings` 中；引导态 `bootstrap_meta` 只存 `app_db_driver/app_db_dsn` 指针。因此换库后若目标库没有 settings 行，重启时 `loadSavedSettings` 会失败，形成引导死锁。
- 启动期 `RenameLegacy`、全模型 `AutoMigrate`、`MigrateLegacyTasks`、统计 upsert 尚未在 MySQL/Postgres 上做过端到端验证；`drivers_integration_test.go` 仅 ping + 单表 migrate，且依赖环境变量。

## Goals / Non-Goals

**Goals:**

- 设置页可配置/切换业务库（驱动 + DSN + 测连通 + 重启生效），并保证换库后平台设置不丢失、重启不失败。
- Setup 向导数据库步骤保持可选 MySQL/Postgres，并补齐 DSN 示例与可诊断错误。
- MySQL/Postgres 下启动装配（迁移、修复、统计、任务对账）端到端可用；用集成测试锁定。

**Non-Goals:**

- 不做跨引擎数据迁移（换库 = 指向已有或全新数据库，旧库数据保留不动）。
- 不把 bootstrap 引导库迁到 MySQL/Postgres（保持本地 SQLite 信任边界）。
- 不为控制面新增 `DB_DRIVER/DATABASE_DSN` 环境变量覆盖（保持 backfill 专用）。
- 不做连接池、读写分离、集群等高阶数据库运维配置。

## Decisions

### D1：复用 `POST /api/v1/setup/database` 承载换库语义

现有端点已具备「驱动校验 → 打开目标库 → ping → `SetAppDB`」流程，首配与换库是同一操作。扩展它：

- 请求体仍是 `{driver, dsn}`；非法驱动/空 DSN/不可达照旧返回 400 且不改引导态。
- 若 `boot.Initialized()` 且目标连接与当前 `AppDB()` 不同：先在目标库 AutoMigrate `platform_settings`，用同一 encKey 写入当前平台设置行，再 `SetAppDB` + `SetRestartRequired(true)`。
- 若目标连接与当前相同：幂等成功，不触发重启标记。

不新增专用端点、不让 `PUT /api/v1/setup/settings` 承担换库，是因为 settings 保存在业务库内，换库必须先切连接再写 settings，拆到数据库端点职责更清晰，前端也复用现有 `testDatabase` 契约。

### D2：设置页数据库节改为可编辑，前端契约不变

设置页 account tab 的数据库展示替换为编辑表单：

- driver Select（sqlite/mysql/postgres）+ DSN Input（按 driver 显示占位示例）+「测连通」按钮。
- 保存调 `POST /api/v1/setup/database`；成功后显示重启提示，走现有 `waitForSetupReady` 重载。
- 存储/网络 tab 保存继续走 `PUT /api/v1/setup/settings`，该端点维持「DB 只读」语义，避免双写。

Setup 向导数据库步骤保持现有流程，仅补 DSN 占位/示例与错误文案。

### D3：MySQL/Postgres 启动链路兼容策略

- `RenameLegacy`：legacy 重命名仅当检测到旧表/旧列时执行；全新 MySQL/Postgres 库均为 no-op，保留现状，不引入 driver 分支（降低兼容面）。
- 全模型 `AutoMigrate`：在集成测试中对全部业务模型（cases/users/sessions/tasks/topics/edges/metrics/channels/menu/settings/stats）跑一遍，锁定列类型与索引长度兼容性。
- `MigrateLegacyTasks` 中把 `lease_until`/`requeue_at` 重置为 `time.Time{}` 的写法改为写 `NULL`：MySQL 严格模式拒绝 `0000-00-00`，零值 time.Time 会触发该问题；Postgres 无此限制但统一为 NULL 更安全。
- 统计 upsert（`clause.OnConflict`）与任务 claim 更新均为标准 SQL，MySQL/Postgres 通用，不做改动，仅测试覆盖。

### D4：换库预置 settings 采用「复制当前行」

目标库的 `platform_settings` 行直接复制当前设置（含加密后的 blob 密钥密文，密文由引导态 encKey 加密，跨库可复用同一把 key），避免要求管理员重填整套配置。复制后 target 库仅需 settings 行即可通过 `loadSavedSettings` 启动，其余业务表由启动期 AutoMigrate 补齐。

## Risks / Trade-offs

- [MySQL 严格模式与零值时间] → `MigrateLegacyTasks` 与同类更新统一写 NULL；集成测试在 MySQL 上覆盖该路径。
- [换库后旧库数据看起来“丢失”] → 前端在设置页展示「切换后使用新库，旧库数据不迁移」提示；README 与架构文档同步说明。
- [MySQL 8.0 以下不支持 `RENAME COLUMN`] → 该语句只在检测到旧 SQLite schema 时才执行，新 MySQL/PG 库不触发；文档标注 MySQL 建议 8.0+。
- [目标库 settings 预置与 encKey 绑定] → encKey 存于本地引导态，不随库迁移；若引导态丢失则密文不可解，属既有约束，不在本 change 范围。

## Migration Plan

- 无破坏性 schema 变更；改动为设置页交互、数据库端点扩展、启动期兼容修正与集成测试。
- 部署：发布后新部署在向导中选库；既有部署在设置页换库，保存后自动重启生效。
- 回滚：恢复上一版本二进制即可；引导态连接未变时行为与旧版一致。

## Open Questions

- MySQL 版本下限（建议 8.0+）与 MariaDB 兼容性：集成测试以 MySQL 8 为准，是否需兼容 MariaDB 可在 verify 阶段再确认，不影响本 change 的规格与任务拆分。

```

## docs/openspec/changes/mysql-postgres-support/tasks.md

- Source: docs/openspec/changes/mysql-postgres-support/tasks.md
- Lines: 1-21
- SHA256: 4c703e39b13de3d3a2fcbf4663c26153fb42b26d6e2f434b8ded2b97e5792a33

```md
## 1. 后端换库 API 与引导态

- [ ] 1.1 扩展 `POST /api/v1/setup/database`：平台已初始化且目标连接与引导态不同时，先在目标库 AutoMigrate `platform_settings` 并复制当前设置行，再 `SetAppDB` + `SetRestartRequired(true)`；连接相同则幂等成功
- [ ] 1.2 `MigrateLegacyTasks` 中 `lease_until`/`requeue_at` 重置改为写 NULL，兼容 MySQL 严格模式
- [ ] 1.3 单测覆盖：不可达/非法 DSN 不改引导态；已初始化换库后目标库存在 settings 行且引导态更新；相同连接幂等；未初始化向导路径不回归

## 2. 设置页与 Setup 数据库配置

- [ ] 2.1 设置页 account tab 数据库节改为编辑表单（driver 下拉 + DSN 输入 + 测连通 + 保存调 `POST /api/v1/setup/database` + 重启提示）
- [ ] 2.2 Setup 向导数据库步骤补齐各驱动的 DSN 占位/示例与可诊断错误文案
- [ ] 2.3 合同测试：设置页数据库配置保存成功/失败、重启提示；Setup 向导 MySQL/Postgres 步骤可完成

## 3. MySQL/Postgres 集成测试与兼容

- [ ] 3.1 扩展 `drivers_integration_test.go`：MySQL/Postgres 上对全部业务模型 AutoMigrate，并覆盖 settings 存取、Topic、Case、Task claim/update 核心读写 roundtrip（环境变量门控）
- [ ] 3.2 在 MySQL/Postgres 上验证启动链路：`RenameLegacy` no-op、`DropTable`、统计 upsert、任务对账可正常执行

## 4. 文档与验证

- [ ] 4.1 README 与 `docs/architecture/data-model.md` 更新多库说明：Setup/设置页配置入口、换库语义（不迁移数据）、MySQL 8.0+ 建议
- [ ] 4.2 `go build ./...` + `go test ./...`（含 integration tags 的驱动测试）；`pnpm tsc -b` + `pnpm vitest run`（web/admin）

```

## docs/openspec/changes/mysql-postgres-support/specs/multi-database-support/spec.md

- Source: docs/openspec/changes/mysql-postgres-support/specs/multi-database-support/spec.md
- Lines: 1-46
- SHA256: a87d8b590ffe07cca608391cc23ba51f7c09517bcf45a9de410fd7e20e0ce8c8

```md
## Purpose

让 Pixoma 控制面的业务库可选用 SQLite、MySQL 或 Postgres 三种驱动，并在 Setup 向导与设置页完成配置与安全切换，避免初始化后无法更换数据库或换库后无法启动。

## ADDED Requirements

### Requirement: 业务库支持多种数据库驱动
系统 MUST 支持 `sqlite`、`mysql`、`postgres` 三种业务库驱动，启动时按引导态存储的驱动与 DSN 打开业务库并完成迁移；未知驱动 MUST 被拒绝并给出可诊断错误。

#### Scenario: 使用 MySQL 启动
- **WHEN** 引导态配置 `db_driver=mysql` 且 DSN 指向可达的 MySQL 实例
- **THEN** 控制面使用该库装配全部业务表并正常启动，核心读写可用

#### Scenario: 使用 Postgres 启动
- **WHEN** 引导态配置 `db_driver=postgres` 且 DSN 指向可达的 Postgres 实例
- **THEN** 控制面使用该库装配全部业务表并正常启动，核心读写可用

#### Scenario: 未知驱动被拒绝
- **WHEN** 配置的业务库驱动不在 `sqlite`、`mysql`、`postgres` 之列
- **THEN** 配置被拒绝，错误信息指明非法驱动值，且不写入引导态

### Requirement: 设置页可配置并切换业务库
系统 MUST 在设置页提供业务库驱动与 DSN 的配置入口，支持连通性测试；保存成功后 MUST 更新引导态业务库连接并标记重启，重启后按新连接装配。

#### Scenario: 初始化后切换到 MySQL
- **WHEN** 平台已初始化，管理员在设置页把业务库从 SQLite 改为 MySQL 并测试连通后保存
- **THEN** 引导态业务库连接更新为 MySQL，平台提示重启；重启后控制面使用 MySQL 并保持后台可用

#### Scenario: 初始化后切换到 Postgres
- **WHEN** 平台已初始化，管理员在设置页把业务库改为 Postgres 并测试连通后保存
- **THEN** 引导态业务库连接更新为 Postgres，平台提示重启；重启后控制面使用 Postgres 并保持后台可用

#### Scenario: 不可达数据库不落盘
- **WHEN** 设置页或 Setup 向导提交的数据库 DSN 不可达、鉴权失败或连接超时
- **THEN** 请求被拒绝并返回可诊断错误，引导态业务库连接不被修改

### Requirement: 换库后平台设置不丢失
切换业务库时，系统 MUST 把当前平台设置写入目标库的 `platform_settings` 行后再更新引导态，确保重启后不因设置缺失而启动失败。

#### Scenario: 切换到空库
- **WHEN** 目标库没有任何业务表且平台已初始化，管理员保存新业务库连接
- **THEN** 目标库写入包含当前设置的 `platform_settings` 行，重启后控制面按新库启动并继续使用既有平台设置

#### Scenario: 切回原有 SQLite 库
- **WHEN** 管理员把业务库从 MySQL/Postgres 切回原有 SQLite 库
- **THEN** 引导态更新为 SQLite，重启后按原库启动且既有数据仍可访问

```

## docs/openspec/changes/mysql-postgres-support/specs/platform-bootstrap/spec.md

- Source: docs/openspec/changes/mysql-postgres-support/specs/platform-bootstrap/spec.md
- Lines: 1-16
- SHA256: a902ee4935008491ad57eb08028d0f6388e9ff97b8a6a26a79a1b531cd1a8225

```md
## ADDED Requirements

### Requirement: 已初始化后业务库连接可更新
平台初始化完成后，系统 MUST 允许通过设置页更新引导态中的业务库驱动与 DSN；更新成功后 MUST 标记重启，重启装配按新连接执行，且更新前 MUST 保证目标库已具备可读取的平台设置。

#### Scenario: 设置页更新业务库
- **WHEN** 平台已初始化，管理员在设置页保存新的业务库连接（驱动 + DSN）
- **THEN** 引导态 `app_db_driver` / `app_db_dsn` 更新为新的值，平台标记重启；重启后按新连接装配

#### Scenario: 重启后使用新库
- **WHEN** 管理员重启 pixoma 且引导态业务库连接已更新
- **THEN** 控制面按新连接打开业务库、读取平台设置并继续提供管理后台

#### Scenario: 更新失败不改引导态
- **WHEN** 新业务库连接不可达或目标库设置写入失败
- **THEN** 请求返回错误，引导态业务库连接保持原值，平台不标记重启

```

## docs/openspec/changes/mysql-postgres-support/specs/setup-wizard/spec.md

- Source: docs/openspec/changes/mysql-postgres-support/specs/setup-wizard/spec.md
- Lines: 1-24
- SHA256: 643ea95b370b03d246d8bce4a29ac0fd170547e1f86482d42c2626d8209ba64e

```md
## MODIFIED Requirements

### Requirement: 初始化向导多步配置
系统 MUST 在管理后台提供初始化向导，引导用户完成至少：业务数据库、部署位置（本机或远程）、对象存储、执行节点/Comfy 指引、渠道（含 Telegram Bot Token）等步骤；业务数据库步骤 MUST 支持 `sqlite`、`mysql`、`postgres` 三种驱动；每步 MUST 做可达性或合法性校验，失败时 MUST 给出可诊断错误。

#### Scenario: 本机路径完成向导
- **WHEN** 用户选择本机部署、配置可用业务库与 localfs 目录，并完成节点与渠道必要项
- **THEN** 向导可标记初始化完成，设置持久化到业务库（或引导态约定位置）

#### Scenario: 远程禁止 localfs
- **WHEN** 用户选择远程部署并尝试将对象存储选为 localfs
- **THEN** 向导拒绝该组合并说明原因

#### Scenario: 使用 MySQL 完成向导
- **WHEN** 用户在数据库步骤选择 MySQL、填写可达 DSN 并测连通
- **THEN** 向导继续后续步骤并可完成初始化，重启后控制面使用该 MySQL 库

#### Scenario: 使用 Postgres 完成向导
- **WHEN** 用户在数据库步骤选择 Postgres、填写可达 DSN 并测连通
- **THEN** 向导继续后续步骤并可完成初始化，重启后控制面使用该 Postgres 库

#### Scenario: 数据库不可达提示
- **WHEN** 用户在数据库步骤填写不可达或非法的 DSN
- **THEN** 向导显示可诊断错误且不进入下一步，引导态业务库连接不被修改

```
