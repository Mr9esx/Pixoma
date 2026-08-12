## ADDED Requirements

### Requirement: 初始化向导多步配置
系统 MUST 在管理后台提供初始化向导，引导用户完成至少：业务数据库、部署位置（本机或远程）、对象存储、执行节点/Comfy 指引、渠道（含 Telegram Bot Token）等步骤；每步 MUST 做可达性或合法性校验，失败时 MUST 给出可诊断错误。

#### Scenario: 本机路径完成向导
- **WHEN** 用户选择本机部署、配置可用业务库与 localfs 目录，并完成节点与渠道必要项
- **THEN** 向导可标记初始化完成，设置持久化到业务库（或引导态约定位置）

#### Scenario: 远程禁止 localfs
- **WHEN** 用户选择远程部署并尝试将对象存储选为 localfs
- **THEN** 向导拒绝该组合并说明原因

### Requirement: 配置落库而非用户手改 YAML
初始化与后续平台设置的主路径 MUST 将配置写入持久化存储（业务库 settings / 引导态），MUST NOT 要求用户手改 `bot.yaml` 才能完成新部署主路径。运维紧急覆盖（环境变量）MAY 存在，但 MUST NOT 作为向导成功的前提。

#### Scenario: 向导保存后进程可读设置
- **WHEN** 向导成功保存设置并按产品约定重启或重新加载
- **THEN** 控制面按所存设置装配 blob 与 Agent 派发行为

### Requirement: 本机与远程部署说明
选择远程时，向导 MUST 展示 `pixoma-edge-agent` 的部署要点（控制面地址、instance_id、鉴权、blob）；选择本机时，系统 MUST 支持自动拉起本机 Edge，或提供等价的一键/明确手起说明。

#### Scenario: 远程展示 Edge 部署指引
- **WHEN** 用户在向导中选择远程部署
- **THEN** 界面展示 Edge 独立进程启动与登记节点所需信息
