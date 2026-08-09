## Purpose

定义单 Bot 场景下 Telegram 主菜单（ReplyKeyboard）的可配置模型：菜单项文案、排序、启用状态与动作（含绑定 Case 进入工作流），并作为运行时与管理面的共同真相源。

## ADDED Requirements

### Requirement: Menu 配置持久化
系统 MUST 将 TG 主菜单配置持久化为可查询、可更新的真相源（默认与现有 SQLite/app DB 一致）。配置 MUST 至少包含：有序菜单项列表；每项含稳定标识、展示文案、启用标志、动作类型；当动作为「进入 Case 工作流」时 MUST 包含合法 `case_id`。

#### Scenario: 读取当前菜单
- **WHEN** 调用方请求当前 Menu 配置
- **THEN** 系统返回按显示顺序排列的菜单项，且字段完整可读

#### Scenario: 更新菜单项绑定 Case
- **WHEN** 管理面将某启用菜单项动作设为进入 Case，并提交已存在的 `case_id`
- **THEN** 持久化成功；后续读取可见该绑定

#### Scenario: 绑定不存在的 Case 被拒绝
- **WHEN** 提交的 `case_id` 在 Case 目录中不存在
- **THEN** 系统拒绝保存并返回可理解的校验错误

### Requirement: 默认种子与兼容
系统在空库或首次启用时 MUST 提供与现网硬编码主菜单等价的默认种子（文案与行列结构可对齐现网），以便未配置前 Bot 仍可展示主菜单。

#### Scenario: 空配置使用默认种子
- **WHEN** 库中尚无自定义 Menu 或标记为使用默认
- **THEN** 运行时仍能得到可用的主菜单项集合（含现网常见入口）

### Requirement: 非 Case 动作允许占位
系统 MUST 允许菜单项动作为占位/提示类（例如暂未实现的业务入口），占位项 MUST NOT 要求 `case_id`。

#### Scenario: 占位入口可保存
- **WHEN** 管理面保存动作为占位的菜单项且未提供 `case_id`
- **THEN** 持久化成功

### Requirement: 回复文字与图片动作
系统 MUST 支持菜单项动作为「回复媒体」：可配置纯文本、一个或多个图片 URL，或二者兼有；保存时 MUST 要求文本与图片列表至少其一非空；图片 URL MUST 为绝对 `http` 或 `https` 地址。本期 MUST NOT 要求管理端本地上传图片到对象存储。

#### Scenario: 保存纯文字回复
- **WHEN** 管理面将某项动作设为回复媒体，仅填写文本并保存
- **THEN** 持久化成功

#### Scenario: 保存文字加图片 URL
- **WHEN** 管理面为回复媒体项填写文本与至少一个合法图片 URL 并保存
- **THEN** 持久化成功，后续读取可见该 `reply` 内容

#### Scenario: 空回复被拒绝
- **WHEN** 动作为回复媒体但文本与图片列表皆空
- **THEN** 系统拒绝保存并返回校验错误
