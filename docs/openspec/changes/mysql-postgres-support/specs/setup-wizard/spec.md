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
