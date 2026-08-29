## Why

主键盘（tgmenu）从领域模型到管理台全面耦合 Telegram：`row/col`、64 字节回调编码、`tg_*` 表与 `/api/v1/tg-menu` 路径直接进入领域模型，交互配置（5 种 kind + 挂载 Case + 行/列）对运营难以理解。后续要接入飞书、企业微信、钉钉时，现有结构无法复用。本次以「消息平台」为一级实体完整重构消息平台层：管理台先建消息平台（选平台、填 Token），菜单配置收进消息平台详情，运行时抽象为可插拔端口，为多平台铺路。

## What Changes

- **消息平台管理（新）**：管理台一级模块「消息平台」——创建消息平台（平台 + 凭证，如 TG Bot Token）、列表、启停；Bot 运行时从消息平台配置装配启动。
- **BREAKING** 菜单收进消息平台：移除顶级「主键盘」模块，菜单编辑进入消息平台详情；`tg_menus` / `tg_menu_items` / `tg_menu_item_cases` 替换为消息平台作用域新表；菜单 API 由 `/api/v1/tg-menu` 改为 `/api/v1/channels/{id}/menu`。
- **BREAKING** 领域模型平台中立化：去除 `row/col`（改为有序列表）、TG 消息长度约束（如 `intro_text` 上限）与 `bot_id=default` 语义；`kind` 收敛为 `folder` / `open_case` / `placeholder` / `reply_media`（移除 `list_cases_by_tag` 兼容动作），平台专有约束下沉适配层。
- **消息平台运行时端口（新）**：定义入站事件、出站消息、媒体桥、身份映射端口；`channel/tg` 重构为端口实现（ReplyKeyboard、Inline、回调编码、媒体下载全部收进 TG 适配器）。
- **身份消息平台化**：`tg_user_id` 唯一约束扩展为消息平台作用域外部身份（`channel + external_id`），用户主键仍为内部 UUID。
- **通知投递按消息平台**：notify 按消息平台适配器投递，为多消息平台并存预留。
- 不改 Case 工作流本体（预览/填表/确认/执行/通知），不改 ComfyUI/队列/存储。
- 无存量数据：不保留旧表、旧字段、旧 API 路径的迁移与兼容层。

## Capabilities

### New Capabilities
- `channel-management`: 消息平台实体（平台、凭证、启停）持久化、admin API 与管理台页面；Bot 运行时从消息平台配置装配
- `channel-menu-config`: 消息平台级菜单配置模型：平台中立的菜单树与动作语义，收进消息平台详情，含迁移与 API
- `channel-runtime-ports`: 消息平台运行时端口（事件/消息/媒体/身份）与适配器注册；TG 作为首个实现

### Modified Capabilities
- `channel-tg`: TG 适配器改为实现消息平台端口；主菜单来源改为消息平台作用域菜单；回调编码、媒体、身份下沉适配层
- `tg-menu`: 菜单模型改为消息平台作用域并平台中立化（去 row/col、TG 约束），Case 挂载与反查语义保留
- `tg-menu-admin-api`: 菜单管理 API 改为消息平台作用域路径（`/api/v1/channels/{id}/menu`），原路径迁移/兼容
- `user-directory`: 外部身份改为消息平台作用域（`channel + external_id`），保留内部 UUID 主键
- `admin-web-shell`: 侧栏移除「主键盘」，新增「消息平台」入口

## Impact

- 领域/包：`internal/channel`（新增消息平台领域与端口）、`internal/tgmenu`（重构为消息平台菜单）、`internal/channel/tg`（适配器化）、`internal/identity`（external_id 消息平台化）、`internal/sharedkernel`（消息平台类型）
- 持久化：新增消息平台表；迁移 `tg_menus` / `tg_menu_items` / `tg_menu_item_cases`；`users` 外部身份消息平台化
- API：admin-api 新增消息平台 CRUD；菜单 API 路径与作用域变化
- 管理台：`web/admin` 新增消息平台页；主键盘页移入消息平台详情
- 配置：Bot Token 统一由消息平台凭证管理，`platform_settings` / env 不再作为配置源
- 测试：领域/持久化/API/适配器测试随重构更新；TG 行为等价回归
