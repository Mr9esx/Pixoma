## MODIFIED Requirements

### Requirement: Menu 树持久化
系统 MUST 将**渠道菜单**持久化为渠道作用域的菜单文档、菜单项（含可选父项）与菜单项–Case 关联。根项（无父项）MUST 可用于渠道入口渲染；子项与关联 Case MUST 可用于目录下的浏览与选择。模型 MUST NOT 包含平台专有字段（如 Telegram 行/列网格、回调编码、消息长度上限）。

#### Scenario: 读取当前菜单树
- **WHEN** 调用方请求某渠道的菜单配置
- **THEN** 系统返回树形结构（含子项与挂载的 Case 标识），字段完整可读

#### Scenario: 保存文件夹并挂载 Case
- **WHEN** 管理面在渠道菜单中创建或更新一文件夹项，并关联一个或多个已存在的 Case 后保存
- **THEN** 持久化成功；后续读取可见该文件夹及其 Case 关联

#### Scenario: 绑定不存在的 Case 被拒绝
- **WHEN** 提交的关联 `case_id` 在 Case 目录中不存在
- **THEN** 系统拒绝保存并返回可理解的校验错误

### Requirement: 默认种子与兼容
系统在渠道无自定义菜单时 MUST 提供可用的默认入口种子；其中「图片」类入口 MUST 以文件夹（或等价可下钻节点）表达，并可挂载图片类 Case。

#### Scenario: 空配置使用默认种子
- **WHEN** 渠道库中尚无自定义菜单
- **THEN** 运行时仍能得到可用的根菜单项集合

### Requirement: 非 Case 动作允许占位与回复媒体
系统 MUST 允许占位（placeholder）与回复媒体（reply_media：文本与/或 http(s) 图片 URL）类动作；回复媒体保存时 MUST 要求文本与图片列表至少其一非空。本期 MUST NOT 要求管理端本地上传图片；图片 URL 的合法性校验（http(s) 绝对地址）保持。

#### Scenario: 保存回复媒体
- **WHEN** 管理面保存合法 reply_media 项
- **THEN** 持久化成功

#### Scenario: 空回复被拒绝
- **WHEN** reply_media 的文本与图片皆空
- **THEN** 系统拒绝保存
