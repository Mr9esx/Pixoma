## ADDED Requirements

### Requirement: 主菜单来自 Menu 配置
Telegram 适配器展示的主 ReplyKeyboard MUST 由 Menu 根项配置（含默认种子）生成，MUST NOT 再以源码常量作为唯一长期配置源。

#### Scenario: /start 展示配置中的主键盘
- **WHEN** 用户发送 `/start` 或打开主菜单，且 Menu 配置可用
- **THEN** 用户收到与配置根项顺序/文案一致的 ReplyKeyboard

### Requirement: 文件夹下钻浏览 Case
当用户点击 kind 为文件夹的菜单项时，适配器 MUST 以 InlineKeyboard 展示该文件夹的子文件夹与挂载的 Case，并提供返回上一级或主菜单的入口。点击 Case MUST 进入既有 Case 预览或填表工作流。

#### Scenario: 进入文件夹看到子项与 Case
- **WHEN** 用户点击根层文件夹项，且该文件夹配置了子文件夹与/或挂载 Case
- **THEN** 用户收到 Inline 按钮，其中可区分文件夹与 Case，且可返回

#### Scenario: 进入子文件夹并可返回
- **WHEN** 用户在 Inline 中点击子文件夹，再点击返回
- **THEN** 用户回到上一层文件夹视图或主菜单（按设计的返回语义），过程 MUST NOT 崩溃

#### Scenario: 占位与回复媒体
- **WHEN** 用户点击 placeholder 或 reply_media 根项
- **THEN** 行为与既有占位提示 / 发文本与图片 URL 语义一致；单张图片失败 MUST NOT 导致进程崩溃

### Requirement: 配置变更对运行中 Bot 可见
Menu 配置更新后，Bot MUST 在下次构建键盘或进入文件夹时使用新配置；MUST NOT 要求重启进程才能使文案/结构生效（若使用短 TTL 缓存，MUST ≤ 数秒量级）。

#### Scenario: 更新后再次打开主菜单或文件夹
- **WHEN** 管理面保存新树后用户再次打开主菜单或进入文件夹
- **THEN** 展示与最新配置一致（按刷新策略）
