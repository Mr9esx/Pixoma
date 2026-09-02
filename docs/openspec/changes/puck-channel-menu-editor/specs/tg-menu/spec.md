## MODIFIED Requirements

### Requirement: Menu 树持久化
系统 MUST 将消息平台菜单持久化为该平台作用域的**一份嵌套树文档**。根层 MUST 可用于入口 ReplyKeyboard 渲染；按钮下内嵌的卡片 MUST 可用于发送消息与 inline 按钮。文档 MUST 包含运行 Telegram 所需的展示字段（主键盘列数、按钮文案、卡片正文/媒体、动作参数）。

#### Scenario: 读取当前菜单树
- **WHEN** 调用方请求某消息平台的菜单配置
- **THEN** 系统返回嵌套树（含键盘按钮、内嵌卡片与动作），字段完整可读

#### Scenario: 保存按钮并挂载 Case
- **WHEN** 管理面保存一棵树，其中某按钮打开工作流并引用已存在的 Case
- **THEN** 持久化成功；后续读取可见该按钮及其 Case 引用

#### Scenario: 绑定不存在的 Case 被拒绝
- **WHEN** 提交的打开工作流动作引用的 Case 在目录中不存在
- **THEN** 系统拒绝保存并返回可理解的校验错误

### Requirement: Case 反查菜单挂载
系统 MUST 能根据 Case id 列出其出现的菜单路径（菜单文档与由根键盘到该动作的标签路径）。

#### Scenario: 查询 Case 的菜单挂载
- **WHEN** 调用方请求某 Case 的菜单挂载
- **THEN** 系统返回零条或多条路径；每条能标识所在菜单节点与可读路径

### Requirement: 默认种子与兼容
系统在消息平台无自定义菜单时 MUST 提供可用的默认嵌套树；其中图片类入口 MUST 以「打开工作流」按钮表达（可再挂卡片），MUST NOT 再依赖独立卡片表或旧文件夹+多 Case 关联。旧格式菜单/卡片数据 MUST NOT 被读取或隐式转换。

#### Scenario: 空配置使用默认种子
- **WHEN** 消息平台库中尚无自定义菜单
- **THEN** 运行时仍能得到可用的根键盘按钮集合

### Requirement: 非 Case 动作允许占位与回复媒体
系统 MUST 允许按钮动作为发文字、发媒体、打开链接、复制文本、列出任务；发媒体保存时 MUST 要求至少一条带 http(s) URL 的媒体。本期 MUST NOT 要求管理端本地上传图片。

#### Scenario: 保存回复媒体
- **WHEN** 管理面保存合法发媒体按钮
- **THEN** 持久化成功

#### Scenario: 空回复被拒绝
- **WHEN** 发媒体的媒体列表为空或 URL 非法
- **THEN** 系统拒绝保存

## ADDED Requirements

### Requirement: 运行时编译嵌套树
Bot 消费菜单时 MUST 使用由嵌套树编译出的键盘与卡片消息，MUST NOT 直接把可视化编辑器的内部 JSON 发给 Telegram API。编译结果与树上结构 MUST 一致：根按钮对应主键盘，打开卡片对应发送该子树卡片。

#### Scenario: 用户点键盘打开卡片
- **WHEN** 用户点击主键盘上动作为打开卡片的按钮
- **THEN** Bot 发送该按钮子树中的卡片消息（正文/媒体与卡片按钮）
