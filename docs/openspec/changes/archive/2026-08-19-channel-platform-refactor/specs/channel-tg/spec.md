## MODIFIED Requirements

### Requirement: TG 适配器将应用 DTO 渲染为 Bot API 消息
系统 MUST 提供 Telegram 适配器：作为消息平台运行时端口实现，接收 Bot Update，调用消息平台无关的应用用例，并将菜单/Case 列表/会话提示/错误/结果渲染为 Telegram 支持的消息形态（文本、Photo、InlineKeyboard 等）。应用层 MUST NOT 依赖 Telegram SDK 类型。发送 Photo 时文件名 MUST 为无路径分隔符的安全基名，避免因 blob key 含 `/` 导致投递失败。

#### Scenario: Case 列表以按钮呈现
- **WHEN** 用户进入某分类的 Case 列表
- **THEN** 适配器发送包含 Case 入口的 InlineKeyboard（可分页）

#### Scenario: 完成后发送图片结果
- **WHEN** 适配器收到 succeeded 的用户通知且输出含 image BlobRef
- **THEN** 适配器向对应用户发送图片（或等价媒体消息）

#### Scenario: 产物 Photo 使用安全文件名
- **WHEN** BlobRef.Key 含目录前缀（如 `outputs/task/0_out.png`）
- **THEN** 发往 Telegram 的上传文件名仅为基名（如 `0_out.png`）

### Requirement: 主菜单来自 Menu 配置
Telegram 适配器展示的主 ReplyKeyboard MUST 由**消息平台作用域**菜单根项配置（含默认种子）生成，MUST NOT 再以源码常量作为唯一长期配置源；菜单读取以消息平台 id 为作用域。

#### Scenario: /start 展示配置中的主键盘
- **WHEN** 用户发送 `/start` 或打开主菜单，且该消息平台菜单配置可用
- **THEN** 用户收到与配置根项顺序/文案一致的 ReplyKeyboard

## ADDED Requirements

### Requirement: 适配器由消息平台配置装配
TG 适配器 MUST 由消息平台实体装配启动：Bot Token 来自消息平台凭证；启用消息平台对应适配器运行，禁用消息平台对应适配器停止。凭证 MUST NOT 从环境变量或 `platform_settings` 读取。

#### Scenario: 消息平台凭证驱动 Bot
- **WHEN** 管理员在消息平台中填写/更新 Telegram Bot Token 并启用
- **THEN** Bot 使用该凭证运行，无需手工修改环境变量后重启

#### Scenario: 禁用消息平台停止适配器
- **WHEN** 消息平台被禁用
- **THEN** TG 适配器不再处理该消息平台事件
