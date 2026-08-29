# channel-tg Specification

## Purpose
TBD - created by archiving change workflow-engine-core. Update Purpose after archive.
## Requirements
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

### Requirement: 适配器执行 Session 锁拦截文案
当应用层返回会话锁定时，适配器 MUST 向用户展示当前流程提示，并提供继续与退出当前流程的操作入口。

#### Scenario: 锁定时展示退出选项
- **WHEN** 用户在填表中尝试开新 Case
- **THEN** 用户收到锁定说明及退出/继续类可点击操作

### Requirement: 消费 notify 完成对用户投递
适配器 MUST 订阅或接收 Orchestrator 的 notify 意图，并完成对 `chat_id` 的消息投递。终态通知 MUST 幂等处理，避免重复刷屏。

#### Scenario: 重复 notify 不重复刷终态消息策略
- **WHEN** 同一 task 终态 notify 重复到达
- **THEN** 适配器不产生重复的骚扰性终态推送（按去重策略合并或忽略）

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

### Requirement: 主菜单来自 Menu 配置
Telegram 适配器展示的主 ReplyKeyboard MUST 由**消息平台作用域**菜单的**能力入口**根项生成（含默认种子），根项数量 MUST 不超过 6 个直达入口；MUST NOT 再以源码常量作为唯一长期配置源；菜单读取以消息平台 id 为作用域。

#### Scenario: /start 展示配置中的主键盘
- **WHEN** 用户发送 `/start` 或打开主菜单，且该消息平台菜单配置可用
- **THEN** 用户收到与配置能力入口顺序/文案一致的 ReplyKeyboard（直达入口 ≤6）

### Requirement: 文件夹下钻浏览 Case
当用户点击分组（文件夹）菜单项时，适配器 MUST 以消息按钮展示该分组的子项与 open_case 能力入口，并提供返回上一级或主菜单的入口。点击能力入口 MUST 进入对应能力流程（open_case 进入既有 Case 预览或填表工作流）。

#### Scenario: 进入文件夹看到子项与 Case
- **WHEN** 用户点击根层分组项，且该分组配置了子分组与/或 open_case 能力入口
- **THEN** 用户收到消息按钮，其中可区分分组与 Case 入口，且可返回

#### Scenario: 进入子文件夹并可返回
- **WHEN** 用户在消息按钮中点击子分组，再点击返回
- **THEN** 用户回到上一层分组视图或主菜单（按设计的返回语义），过程 MUST NOT 崩溃

#### Scenario: 占位与回复媒体
- **WHEN** 用户点击占位或回复媒体展示项
- **THEN** 行为与既有占位提示 / 发文本与图片 URL 语义一致；单张图片失败 MUST NOT 导致进程崩溃

### Requirement: 配置变更对运行中 Bot 可见
Menu 配置更新后，Bot MUST 在下次构建键盘或进入文件夹时使用新配置；MUST NOT 要求重启进程才能使文案/结构生效（若使用短 TTL 缓存，MUST ≤ 数秒量级）。

#### Scenario: 更新后再次打开主菜单或文件夹
- **WHEN** 管理面保存新树后用户再次打开主菜单或进入文件夹
- **THEN** 展示与最新配置一致（按刷新策略）

### Requirement: 适配器由消息平台配置装配
TG 适配器 MUST 由消息平台实体装配启动：Bot Token 来自消息平台凭证；启用消息平台对应适配器运行，禁用消息平台对应适配器停止。凭证 MUST NOT 从环境变量或 `platform_settings` 读取。

#### Scenario: 消息平台凭证驱动 Bot
- **WHEN** 管理员在消息平台中填写/更新 Telegram Bot Token 并启用
- **THEN** Bot 使用该凭证运行，无需手工修改环境变量后重启

#### Scenario: 禁用消息平台停止适配器
- **WHEN** 消息平台被禁用
- **THEN** TG 适配器不再处理该消息平台事件

### Requirement: 能力入口与流程渲染
TG 适配器 MUST 按能力的消息平台渲染声明渲染入口与流程（消息按钮、返回、结果），MUST NOT 为具体业务编写分支；新注册能力加入菜单后，适配器无需改动即可完成事件翻译与结果渲染。

#### Scenario: 新能力入口直接可用
- **WHEN** 管理台把新注册能力加入 TG 菜单并保存
- **THEN** 主键盘或消息按钮出现该入口，点击触发对应能力，无需改适配器

