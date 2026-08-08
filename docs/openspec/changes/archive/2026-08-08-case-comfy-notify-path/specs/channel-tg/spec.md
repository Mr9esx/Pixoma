## ADDED Requirements

### Requirement: 当前输入为图片时采集 Telegram 媒体
当会话当前待填字段类型为 `image` 时，适配器 MUST 接受用户发送的 Photo（以及可识别的图片 Document），下载并写入 Blob，再以 Blob Draft 提交给应用层。此时 MUST NOT 把任意纯文本当作该图片字段的合法值（引导文案与跳过规则除外）。

#### Scenario: 用户发送照片填入参考图
- **WHEN** 当前 Case 字段为必填 `image`，用户发送一张 Photo
- **THEN** 适配器将该图物化为 Blob 并推进会话到下一输入或确认态

#### Scenario: 图片字段收到无关文本时提示
- **WHEN** 当前字段为 `image` 且用户发送非命令普通文本
- **THEN** 适配器提示需要发送图片，而不把该文本写入图片 Draft

### Requirement: 按字段类型解析标量文本
当当前字段类型为 `number` 或 `boolean` 时，适配器 MUST 将用户文本解析为对应 Draft 类型后再提交；解析失败 MUST 提示用户重试，不得以错误类型进入 ConfirmRun 校验。

#### Scenario: 数字种子解析成功
- **WHEN** 当前字段为 `number`，用户发送合法整数字符串
- **THEN** 草稿以数值形式保存且后续校验可通过

## MODIFIED Requirements

### Requirement: TG 适配器将应用 DTO 渲染为 Bot API 消息
系统 MUST 提供 Telegram 适配器：接收 Bot Update，调用渠道无关的应用用例，并将菜单/Case 列表/会话提示/错误/结果渲染为 Telegram 支持的消息形态（文本、Photo、InlineKeyboard 等）。应用层 MUST NOT 依赖 Telegram SDK 类型。发送 Photo 时文件名 MUST 为无路径分隔符的安全基名，避免因 blob key 含 `/` 导致投递失败。

#### Scenario: Case 列表以按钮呈现
- **WHEN** 用户进入某分类的 Case 列表
- **THEN** 适配器发送包含 Case 入口的 InlineKeyboard（可分页）

#### Scenario: 完成后发送图片结果
- **WHEN** 适配器收到 succeeded 的用户通知且输出含 image BlobRef
- **THEN** 适配器向对应用户发送图片（或等价媒体消息）

#### Scenario: 产物 Photo 使用安全文件名
- **WHEN** BlobRef.Key 含目录前缀（如 `outputs/task/0_out.png`）
- **THEN** 发往 Telegram 的上传文件名仅为基名（如 `0_out.png`）
