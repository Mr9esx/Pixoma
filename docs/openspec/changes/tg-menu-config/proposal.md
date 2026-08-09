## Why

Telegram 主菜单（ReplyKeyboard）目前硬编码在 `menu.go`，运维无法在后台调整入口文案、顺序，也无法把「菜单项 → Case 工作流」做成可配置绑定。管理控制台上一期明确不做 TG Menu，因此后台看不到配置入口。本期在**不做多 Bot / 多租户**的前提下，先把单 Bot 的 Menu 做成可落库、可管理、可驱动对话入口。

## What Changes

- 新增 TG Menu 配置真相源（持久化）：菜单项文案、排序、启用、动作类型；与 Case 相关的入口可绑定 `case_id`（进入既有工作流）；支持 `reply_media`（文字 + 图片 URL 回复，本期不做图片上传）
- Bot 运行时从配置加载主键盘与点击路由，替代硬编码主菜单布局（保留迁移/默认种子，行为与现网对齐可渐进）
- admin-api 提供 Menu 读写（及必要的启停/重排）接口
- `web/admin` 增加 Menu 管理页，并接入侧栏；中英 i18n
- **非目标（本期明确不做）**：多 Bot / `bot_id` 租户改造；Bot Token CRUD；充值/签到/个人中心等业务真逻辑（可为占位动作）；独立前端 mock

## Capabilities

### New Capabilities
- `tg-menu`: TG 主菜单领域模型与持久化（项、排序、动作、Case 绑定）及默认种子约定
- `tg-menu-admin-api`: 管理端 Menu HTTP API（list/get/update 等，对接 admin-api）

### Modified Capabilities
- `channel-tg`: 主 ReplyKeyboard 与菜单入口行为改为读取 Menu 配置；绑定 Case 的入口进入既有 Case 工作流
- `admin-web-shell`: 侧栏增加 TG Menu（或等价命名）入口
- `admin-resource-pages`: 提供 Menu 管理页（列表/编辑，与 Case 选择绑定）

## Impact

- 代码：`internal/channel/tg`（menu 加载与路由）、新 menu 领域/持久化包（或 catalog 邻域）、`apps/admin-api` 路由与 handler、`web/admin` 路由/侧栏/i18n
- 数据：新增 Menu 相关表（或等价持久化）；启动种子/迁移；与 `catalog_cases` 通过 `case_id` 关联（Case 本身仍为全局单 Bot 目录）
- 配置：Token 仍走现有 `bot.yaml` / `TG_BOT_TOKEN`（不改多租户）
- 文档：架构 data-model / runtime 视表与主链路变更同步；admin 联调说明补充 Menu
