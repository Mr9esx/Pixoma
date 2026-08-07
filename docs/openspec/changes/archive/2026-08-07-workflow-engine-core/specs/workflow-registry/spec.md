## ADDED Requirements

### Requirement: 通过仓储注册与持久化 Case
系统 MUST 提供工作流/case 注册能力，将符合协议的 case 定义持久化到数据库。默认实现 MUST 使用 SQLite；领域模型与仓储接口 MUST 不绑定 SQLite 专有 SQL，以便后续切换 MySQL。系统 MUST 使用 ORM 管理实体映射与迁移。

#### Scenario: 成功注册新 case
- **WHEN** 调用方提交合法的 case 定义进行注册
- **THEN** 系统将其写入数据库，并可用同一标识再次查询到完整定义

#### Scenario: 重复标识注册冲突
- **WHEN** 调用方使用已存在的 case 标识再次注册且策略为禁止覆盖
- **THEN** 系统拒绝注册并返回冲突错误

### Requirement: 查询与更新已注册 Case
系统 MUST 支持按标识获取单个 case、列出 case（至少支持按类别标签过滤），以及更新已有 case 的元数据、schema 或 ComfyUI 绑定。更新后的定义 MUST 成为后续校验与执行的唯一来源。

#### Scenario: 按标识读取
- **WHEN** 调用方请求某个已注册 case 的标识
- **THEN** 系统返回完整协议字段与绑定信息

#### Scenario: 按类别列出
- **WHEN** 调用方按 `img2img` 类别过滤列表
- **THEN** 系统仅返回带有该标签的 case

#### Scenario: 更新 ComfyUI 绑定
- **WHEN** 调用方更新某 case 的节点注入映射并保存
- **THEN** 之后的执行使用更新后的绑定，而不是旧映射

### Requirement: 禁用或删除 Case 不影响历史执行记录边界
系统 MUST 支持将 case 标记为停用（或等价不可用状态），停用后 MUST NOT 接受新的执行请求。若提供删除，删除策略 MUST 明确；本阶段允许硬删除或停用二选一，但行为 MUST 文档化且一致。

#### Scenario: 停用后拒绝新执行
- **WHEN** 某 case 已被停用，调用方仍请求执行该 case
- **THEN** 系统拒绝执行并说明 case 不可用
