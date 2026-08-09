## ADDED Requirements

### Requirement: 主菜单来自 Menu 配置
Telegram 适配器展示的主 ReplyKeyboard MUST 由 Menu 配置（含默认种子）生成，MUST NOT 再以源码常量作为唯一长期配置源。点击菜单项时 MUST 按该项动作执行：绑定 Case 则进入既有 Case 工作流；占位动作则回复说明或无害提示。

#### Scenario: /start 展示配置中的主键盘
- **WHEN** 用户发送 `/start` 或打开主菜单，且 Menu 配置可用
- **THEN** 用户收到与配置顺序/文案一致的 ReplyKeyboard

#### Scenario: 绑定 Case 的入口进入工作流
- **WHEN** 用户点击动作为「进入 Case」且已绑定合法 `case_id` 的菜单项
- **THEN** 系统进入该 Case 的既有预览或填表工作流（与现网 Case 入口语义一致）

#### Scenario: 占位入口提示
- **WHEN** 用户点击占位动作菜单项
- **THEN** 用户收到非崩溃的提示文案，且不错误打开无关 Case

#### Scenario: 回复媒体发送文字与图片
- **WHEN** 用户点击动作为回复媒体的菜单项，且配置含文本与图片 URL
- **THEN** 适配器向该用户发送文本消息，并尝试按配置发送图片；单张图片失败 MUST NOT 导致进程崩溃，且在仍有文本时应已发出文本

#### Scenario: 仅图片 URL 的回复媒体
- **WHEN** 用户点击仅配置了图片 URL、无文本的回复媒体项
- **THEN** 适配器尝试发送图片；若全部失败则向用户发送可理解的错误或降级提示

### Requirement: 配置变更对运行中 Bot 可见
在单 Bot 进程模型下，Menu 配置更新后，Bot MUST 在合理策略下使用新配置（例如下次构建键盘时读取最新配置，或短 TTL 刷新）；MUST NOT 要求重启进程才能使文案/绑定生效（若实现选择启动缓存，则 MUST 在设计中写明刷新策略并在验收中可测）。

#### Scenario: 更新文案后再次打开主菜单
- **WHEN** 管理面修改某菜单项文案并保存成功，用户再次触发主菜单展示
- **THEN** 键盘展示更新后的文案（按既定刷新策略，可接受「下一次构建键盘」生效）
