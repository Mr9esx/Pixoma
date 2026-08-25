# setup-wizard Specification

## Purpose
TBD - created by archiving change pixoma-guided-deploy. Update Purpose after archive.
## Requirements
### Requirement: 初始化向导多步配置
系统 MUST 在管理后台提供初始化向导，引导用户完成至少：业务数据库、对象存储、执行节点/Comfy 指引、渠道（含 Telegram Bot Token）等步骤；业务数据库步骤 MUST 支持 `sqlite`、`mysql`、`postgres` 三种驱动；每步 MUST 做可达性或合法性校验，失败时 MUST 给出可诊断错误；对象存储步骤 MUST 提供连通性测试。

#### Scenario: 本机路径完成向导
- **WHEN** 用户配置可用业务库与 localfs 目录，并完成节点与渠道必要项
- **THEN** 向导可标记初始化完成，设置持久化到业务库（或引导态约定位置），`placement` 推断为本机

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

#### Scenario: 切换数据库驱动后 DSN 输入更新
- **WHEN** 用户在数据库步骤把驱动从 SQLite 切换为 MySQL/Postgres
- **THEN** 输入区切换为对应驱动的结构化字段（Host/端口/用户/密码/数据库等），按字段组装出的连接随之更新，不再沿用 SQLite 的路径值

#### Scenario: 结构化字段配置 MySQL/Postgres
- **WHEN** 用户选择 MySQL 或 Postgres，并填写 Host、端口、用户、密码、数据库等字段
- **THEN** 向导按字段组装连接字符串用于测连通；密码字段不回显明文；用户可在「附加参数」输入框直接追加任意 key=value 参数（MySQL 为 `&a=b`，Postgres 为空格分隔的 `key=value`），无需手拼完整 DSN

#### Scenario: 常见错误中文提示与详情
- **WHEN** 测连通或保存失败，且错误属于常见类型（拒绝连接、鉴权失败、库不存在、超时、未知驱动等）
- **THEN** 向导以 Alert 形式展示中文友好提示，并在其下方展示实际错误详情

#### Scenario: 测连通与继续分离
- **WHEN** 用户在数据库步骤点「连通性测试」且服务器可达、账号密码正确
- **THEN** 显示连接正常；「继续」始终可点，不要求先测连通，连接问题会在后续步骤以 Alert 报错

#### Scenario: 目标数据库不存在时自动创建
- **WHEN** 用户选择 MySQL/Postgres，服务器可达且账号密码正确，但目标数据库不存在
- **THEN** 向导自动创建目标数据库（MySQL `CREATE DATABASE IF NOT EXISTS`，Postgres 检测 `pg_database` 后创建）并继续；业务表在保存/启动时由 AutoMigrate 自动创建

#### Scenario: 向导不询问部署位置
- **WHEN** 用户完成数据库步骤后进入对象存储步骤
- **THEN** 向导直接展示对象存储配置，不再出现「出图机器在哪」步骤；保存时 `placement` 由对象存储驱动推断（`localfs`→本机，`s3`/`tos`→远程）

#### Scenario: 对象存储连通性测试通过
- **WHEN** 用户选择 S3 或 TOS，填写 endpoint/region/bucket/密钥并点「连通性测试」，配置正确且 bucket 可达
- **THEN** 向导以 success Alert 显示连接正常，可继续下一步

#### Scenario: 对象存储连通性测试失败
- **WHEN** 用户点「连通性测试」但 endpoint 不可达、密钥错误或 bucket 不存在
- **THEN** 向导以 destructive Alert 显示中文友好提示与实际错误详情，不标记连接正常

#### Scenario: bucket 不存在时提示自动创建
- **WHEN** 用户点「连通性测试」且 bucket 不存在
- **THEN** 向导提示「bucket `xxx` 不存在，是否帮你创建？」；用户确认后自动创建并复检，成功显示连接正常

#### Scenario: localfs 目录校验
- **WHEN** 用户选择 localfs 并点「连通性测试」
- **THEN** 目录可写时显示连接正常；目录不可写时显示可诊断错误

#### Scenario: 远程 localfs 组合在 API 层仍被拒绝
- **WHEN** 提交 `placement=remote` 且 `blob.driver=localfs` 的设置
- **THEN** 设置校验失败并说明原因（向导 UI 不提供该组合，API 层保留原约束）

### Requirement: 配置落库而非用户手改 YAML
初始化与后续平台设置的主路径 MUST 将配置写入持久化存储（业务库 settings / 引导态），MUST NOT 要求用户手改 `bot.yaml` 才能完成新部署主路径。运维紧急覆盖（环境变量）MAY 存在，但 MUST NOT 作为向导成功的前提。

#### Scenario: 向导保存后进程可读设置
- **WHEN** 向导成功保存设置并按产品约定重启或重新加载
- **THEN** 控制面按所存设置装配 blob 与 Agent 派发行为

### Requirement: 本机与远程部署说明
系统 MUST 在向导完成页或项目文档中说明本机/远程两种部署与 `pixoma-edge-agent` 的部署要点（控制面地址、instance_id、鉴权、blob）；向导流程 MUST NOT 要求用户先选择部署位置，部署位置由对象存储选型决定。

#### Scenario: 远程展示 Edge 部署指引
- **WHEN** 用户在向导中选择远程部署
- **THEN** 界面展示 Edge 独立进程启动与登记节点所需信息

#### Scenario: 完成向导后可获得部署指引
- **WHEN** 用户完成初始化向导
- **THEN** 完成页/README 提供 Edge 部署要点（控制面地址、节点 token、blob 说明），无需在向导中先选本机/远程

