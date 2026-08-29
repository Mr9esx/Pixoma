## MODIFIED Requirements

### Requirement: 菜单模型平台中立
菜单领域模型 MUST 使用平台中立的**能力入口**语义：菜单项 = {展示字段（label/intro/占位/回复媒体）、能力入口（capability_id + params）、children}；中立核心字段 MUST NOT 包含平台专有字段（如 Telegram 行/列网格、回调编码、消息长度上限）。平台渲染差异 MUST 由能力的消息平台渲染声明承载，MUST NOT 进入中立核心字段，且 MUST NOT 因通用化抹平平台特色能力。

#### Scenario: 中立字段不含 TG 专有字段
- **WHEN** 通过管理 API 读取或提交菜单的中立字段
- **THEN** 中立字段中不存在 row/col、回调前缀或 TG 消息长度约束；平台差异仅存在于能力的消息平台渲染声明

#### Scenario: TG 特色能力保留
- **WHEN** 管理员为 TG 消息平台配置能力入口的消息平台展示参数（如根层每行按钮数）
- **THEN** 参数存于该能力的消息平台渲染声明，TG 适配器按声明渲染；其他消息平台读取不到且不受影响

#### Scenario: 同一菜单可被多平台适配器渲染
- **WHEN** 同一份消息平台菜单（能力入口 + 分组）分别由 TG 与后续平台适配器消费
- **THEN** 各适配器按各自平台形态渲染，无需改写菜单配置

### Requirement: Case 挂载与反查
系统 MUST 支持 open_case 能力入口引用 Case（作为能力参数）并从 Case 反查其出现的菜单路径；引用的 Case 必须存在，否则拒绝保存。

#### Scenario: 保存文件夹并挂载 Case
- **WHEN** 管理面在消息平台菜单中创建文件夹项，其 open_case 能力参数引用已存在的 Case 后保存
- **THEN** 持久化成功，后续读取可见该文件夹及其 Case 引用

#### Scenario: 绑定不存在的 Case 被拒绝
- **WHEN** 提交的 open_case 能力参数引用不存在的 `case_id`
- **THEN** 系统拒绝保存并返回可理解的校验错误

#### Scenario: 查询 Case 的菜单挂载
- **WHEN** 调用方请求某 Case 的菜单挂载
- **THEN** 系统返回零条或多条路径；每条能标识所在消息平台、菜单项与可读路径

### Requirement: 默认种子按消息平台
系统在消息平台无自定义菜单时 MUST 提供可用的默认种子菜单；种子根层入口 MUST 使用 open_case 能力并挂载图片类 Case。

#### Scenario: 空配置使用默认种子
- **WHEN** 消息平台尚无自定义菜单
- **THEN** 运行时仍能得到可用的根菜单项集合（open_case 能力入口）

## REMOVED Requirements

### Requirement: 消息平台差异数据（extras）管理
**Reason**: 平台差异数据并入能力的消息平台渲染声明，不再以游离 extras 存储。
**Migration**: 既有 extras（如 tg_root_layout）迁移为 open_case 等能力的消息平台渲染声明参数；管理台不再提供 extras 独立编辑入口。
