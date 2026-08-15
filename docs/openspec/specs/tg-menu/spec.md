# tg-menu Specification

## Purpose
定义单 Bot 场景下 Telegram 主键盘的可配置树模型：根层 ReplyKeyboard、文件夹下钻、菜单项与 Case 的持久化关联，并支持从 Case 反查挂载位置。

## Requirements
### Requirement: Menu 树持久化
系统 MUST 将主键盘持久化为菜单文档、菜单项（含可选父项）与菜单项–Case 关联。根项（无父项）MUST 可用于 ReplyKeyboard；子项与关联 Case MUST 可用于文件夹下的 Inline 浏览。

#### Scenario: 读取当前菜单树
- **WHEN** 调用方请求当前 Menu 配置
- **THEN** 系统返回树形结构（含子项与挂载的 Case 标识），字段完整可读

#### Scenario: 保存文件夹并挂载 Case
- **WHEN** 管理面创建或更新一文件夹项，并关联一个或多个已存在的 Case 后保存
- **THEN** 持久化成功；后续读取可见该文件夹及其 Case 关联

#### Scenario: 绑定不存在的 Case 被拒绝
- **WHEN** 提交的关联 `case_id` 在 Case 目录中不存在
- **THEN** 系统拒绝保存并返回可理解的校验错误

### Requirement: Case 反查菜单挂载
系统 MUST 能根据 Case id 列出其出现的菜单路径（菜单文档与由根到该项的标签路径）。

#### Scenario: 查询 Case 的菜单挂载
- **WHEN** 调用方请求某 Case 的菜单挂载
- **THEN** 系统返回零条或多条路径；每条能标识所在菜单项与可读路径

### Requirement: 默认种子与兼容
系统在空库或首次启用时 MUST 提供可用的默认根键盘种子；其中「图片」类入口 MUST 以文件夹（或等价可下钻节点）表达，并可挂载图片类 Case。系统 MAY 保留按 tag 列表动作以兼容旧配置，但 MUST NOT 将其作为新种子的唯一主路径。

#### Scenario: 空配置使用默认种子
- **WHEN** 库中尚无自定义 Menu
- **THEN** 运行时仍能得到可用的根菜单项集合

### Requirement: 非 Case 动作允许占位与回复媒体
系统 MUST 允许 `placeholder` 与 `reply_media`（文本与/或 http(s) 图片 URL）；`reply_media` 保存时 MUST 要求文本与图片列表至少其一非空。本期 MUST NOT 要求管理端本地上传图片。

#### Scenario: 保存回复媒体
- **WHEN** 管理面保存合法 reply_media 项
- **THEN** 持久化成功

#### Scenario: 空回复被拒绝
- **WHEN** reply_media 的文本与图片皆空
- **THEN** 系统拒绝保存
