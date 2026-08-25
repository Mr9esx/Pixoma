## Purpose

让 Pixoma 控制面的业务库可选用 SQLite、MySQL 或 Postgres 三种驱动，并在 Setup 向导完成配置，保证 MySQL/Postgres 下启动装配与核心读写可用。

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

### Requirement: 设置页只读展示业务库信息
系统 MUST 在设置页只读展示当前业务库驱动与 DSN 信息，MUST NOT 提供修改业务库连接或切换业务库的入口；业务库连接只允许在初始化向导（Setup）中配置。

#### Scenario: 设置页查看业务库信息
- **WHEN** 平台已初始化，管理员打开设置页的账户页签
- **THEN** 页面展示当前业务库驱动与 DSN，且不提供编辑或切换控件

#### Scenario: 不可达数据库不落盘
- **WHEN** Setup 向导提交的数据库 DSN 不可达、鉴权失败或连接超时
- **THEN** 向导拒绝该配置并返回可诊断错误，引导态业务库连接不被修改
