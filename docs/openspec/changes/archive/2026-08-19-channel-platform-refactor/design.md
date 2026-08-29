## Context

现状（动机见 proposal.md）：`internal/tgmenu` 领域模型携带 TG 专有字段（`row/col`、`intro_text ≤3500`、`bot_id=default`），持久化表与 API 路径以 `tg_` 命名；`internal/channel/tg` 的 `Messenger` 接口形状即 TG 渲染抽象，回调编码（64 字节前缀）散落在适配器编排逻辑；`identity` 以 `tg_user_id` 唯一约束；`notifybridge` 位于 tg 包内。Bot Token 当前存于 `platform_settings.telegram_token_cipher`（AES-GCM 密文）。完整 Case 工作流（`botapp.Facade`）已消息平台无关，是本次重构的稳定内核。

约束：单仓库 Go 项目，admin-api 与 bot 可同进程；SQLite/GORM；管理台为 React（shadcn）；TG 为唯一现网平台，行为必须等价保留。

## Goals / Non-Goals

**Goals:**
- 「消息平台」成为领域与管理台一级实体：平台、凭证、启停，驱动对应 Bot 运行时装配
- 菜单模型平台中立：无 row/col、无 TG 长度/回调约束，按消息平台作用域存储
- 消息平台运行时端口契约：入站事件、出站消息、媒体桥、身份映射；TG 适配器化
- 身份与通知消息平台化；既有 TG 行为与数据平滑迁移

**Non-Goals:**
- 不实现飞书/企微/钉钉适配器（端口就绪即可）
- 本期不支持多消息平台并行运行与多机器人切换 UI（数据模型预留）
- 不改 Case 工作流、ComfyUI/队列/存储
- 不做管理端图片上传

## Decisions

### D1. 领域分层：中立领域 + 消息平台端口 + 平台适配器

```
internal/menu（原 tgmenu，重命名）   ← 平台中立菜单领域
internal/channel                     ← 消息平台实体 + 端口契约（新）
internal/channel/tg                  ← TG 适配器（端口实现 + 渲染/回调/媒体）
internal/botapp + sharedkernel       ← 消息平台无关应用内核（不动）
```

`internal/tgmenu` 重命名为 `internal/menu`，消除 tg 命名残留；目录移动 + import 更新为机械重构。菜单领域只表达「目录/动作意图」，渲染差异（键盘形态、按钮布局、回调编码、媒体上传）全部留在适配器。

备选：保留 `internal/tgmenu` 包名。放弃原因：命名即耦合，完整重构应同步清理。

### D2. 消息平台实体与凭证

新增 `channels` 表：`id`（稳定字符串）、`platform`（如 `telegram`）、`name`、`credential_ciphertext`、`enabled`、时间戳。凭证以 AES-GCM 加密存储（密钥来自 env，开发环境可退化为明文并告警），API 回显 `masked_token`（如 `12345****`），永不回显完整 token。

消息平台一律由管理台创建（选平台 + 填凭证）；无存量数据，不做 env / `platform_settings` 迁移，Bot Token 仅存于 `channels.credential_ciphertext`。

### D3. 菜单模型中立化

- `row/col` → 同层有序列表（`order`）；TG 适配器自行决定根层两列排布
- `intro_text` 长度上限移出领域校验，由 TG 适配器渲染时处理
- `bot_id` → `channel_id`，菜单文档唯一绑定消息平台
- `kind` 收敛为四种：`folder` / `open_case` / `placeholder` / `reply_media`；移除 `list_cases_by_tag` 与 `tag` 字段（无存量兼容需求）
- 表：`channel_menus` / `channel_menu_items` / `channel_menu_item_cases`（替换 `tg_menus` 等）

### D3a. 不同消息平台类型的菜单存放

**决策：中立核心模型（跨平台共有意图）+ 消息平台扩展表（平台差异数据）+ 适配器收敛渲染；不抹平平台特色能力。**

- 每个消息平台实例（`channels.id`）拥有一份菜单树：`channel_menus`（1:1）+ `channel_menu_items`（树）+ `channel_menu_item_cases`（Case 挂载）。不同消息平台的数据以 `channel_id` 隔离，互不串扰。
- 中立核心字段只承载跨平台共有的「意图」：`label`、`order`、`enabled`、`kind`、`case_ids`。
- **消息平台扩展表 `channel_menu_item_extras`**：按消息平台存放平台差异数据（`channel_id` + `menu_item_id` + `extra_type` + JSON）。TG 特色能力在此保留：根层网格布局（row/col 精确排布）、每行按钮数、Inline 行为提示等；只对对应消息平台生效，其他消息平台不读取。
- 平台能力差异由适配器收敛：

| 平台 | 入口渲染 | 文件夹下钻 | 差异数据 |
|---|---|---|---|
| Telegram | ReplyKeyboard（优先读 extras 网格布局，缺省按 order 两列） | InlineKeyboard + 回调 | `tg_root_layout` 等 |
| 飞书（后续） | 机器人菜单 / 欢迎卡片按钮 | 交互卡片按钮（可原地更新） | 卡片标题/风格等 |
| 企微（后续） | 自定义菜单（列表式） | 菜单层级映射 | 层级上限提示等 |

- **不抹平原则**：中立模型只承载所有消息平台都有的语义；TG 特有交互（根层网格布局、Inline 下钻、回复媒体、占位）不得因通用化而丢失，以 extras + 适配器完整保留。若某平台差异字段多到需要强类型列，可为该平台建专用扩展表（如 `tg_channel_extras`），本期以通用 extras 表实现并预留此扩展点。

备选方案与放弃原因：
- 把全部差异塞进中立核心模型：模型被平台字段污染（正是本次要解决的痛点），放弃；
- 按平台拆整套菜单表（`tg_menu_items` / `feishu_menu_items` …）：Case 挂载与反查需跨表 union，维护成本高；核心菜单共用一套，差异走 extras（用户已确认该方案）。

### D4. 会话寻址消息平台化

`sharedkernel.ChatID` 由 `int64` 改为消息平台命名的字符串地址（`tg:<chat_id>`，未来 `feishu:<chat_id>`），conversation/task/notify 的列与消息类型同步迁移。`sessions` 表拆分存储 `channel_id` + `chat_external_id` 两列（避免前缀解析查询，并为按消息平台过滤建索引）；跨进程事件（notify）继续使用完整地址字符串。TG 适配器在入口处完成 `chat_id → "tg:xxx"` 映射，出口处反向映射。

备选：保留 int64 + 独立映射表。放弃原因：映射表增加间接层且 notify 跨进程传递时仍需转换；直接消息平台化更彻底、更利于飞书字符串 chat_id。

## 数据库调整（现状 → 修改后）

### 现状（相关表）

**`tg_menus`**（菜单文档）

| 列 | 类型 | 说明 |
|---|---|---|
| id | TEXT PK | 恒 `default` |
| bot_id | TEXT NOT NULL | 恒 `default` |
| updated_at | DATETIME | |

**`tg_menu_items`**（菜单项）

| 列 | 类型 | 说明 |
|---|---|---|
| id | TEXT PK | 稳定字符串 |
| menu_id | TEXT NOT NULL, idx | → tg_menus.id |
| parent_id | TEXT NULL, idx | 空 = 根层 |
| label | TEXT NOT NULL | 文案 |
| row / col | INT NOT NULL | TG 网格排布 |
| enabled | BOOL NOT NULL | |
| kind | TEXT NOT NULL | folder/open_case/placeholder/reply_media/list_cases_by_tag |
| placeholder_text | TEXT | |
| intro_text | TEXT | 领域限 ≤3500 |
| tag | TEXT | list_cases_by_tag |
| reply_json | TEXT | reply_media |

**`tg_menu_item_cases`**

| 列 | 类型 |
|---|---|
| menu_item_id | TEXT PK |
| case_id | TEXT PK |
| sort | INT NOT NULL |

**`users`**

| 列 | 类型 | 说明 |
|---|---|---|
| id | TEXT PK | 内部 UUID |
| tg_user_id | INT UNIQUE NOT NULL | TG 用户 id |
| username / first_name / last_name / language_code | TEXT | TG 资料 |
| is_bot / is_premium | BOOL NULL | TG 资料 |
| last_seen_at | DATETIME NOT NULL | |

**`sessions`**

| 列 | 类型 | 说明 |
|---|---|---|
| id | TEXT PK | |
| user_id | TEXT NOT NULL, idx | |
| chat_id | INT NOT NULL, idx | TG chat id |
| case_id / status / current_input_index / input_keys_json / draft_json | | 会话状态 |

**`platform_settings`**（单行）：含 `telegram_token_cipher`（AES-GCM 密文，当前 Bot Token 来源）。

### 修改后（相关表）

**`channels`（新）**

| 列 | 类型 | 说明 |
|---|---|---|
| id | TEXT PK | 稳定 id，如 `tg-default` |
| platform | TEXT NOT NULL | `telegram` / `feishu` / `wecom` / `dingtalk` |
| name | TEXT NOT NULL | 显示名称 |
| credential_ciphertext | TEXT NOT NULL | AES-GCM 加密凭证 JSON，如 `{"bot_token":"..."}` |
| enabled | BOOL NOT NULL | |
| created_at / updated_at | DATETIME | |

**`channel_menus`（替换 tg_menus）**

| 列 | 类型 | 说明 |
|---|---|---|
| channel_id | TEXT PK | → channels.id，1:1 |
| updated_at | DATETIME | |

**`channel_menu_items`（替换 tg_menu_items）**

| 列 | 类型 | 说明 |
|---|---|---|
| id | TEXT PK | 稳定字符串（消息平台内唯一） |
| channel_id | TEXT NOT NULL, idx | → channels.id |
| parent_id | TEXT NULL, idx | 空 = 根层 |
| label | TEXT NOT NULL | 同层唯一 |
| order | INT NOT NULL | 同层顺序（替代 row/col） |
| enabled | BOOL NOT NULL | |
| kind | TEXT NOT NULL | folder/open_case/placeholder/reply_media |
| placeholder_text | TEXT | |
| intro_text | TEXT | 领域不限长，适配器渲染时限制 |
| reply_json | TEXT | |
| render_hints_json | TEXT NULL | 平台展示提示（见 D3a） |

约束：UNIQUE(channel_id, parent_id, label)；UNIQUE(channel_id, parent_id, order)。

**`channel_menu_item_cases`（替换 tg_menu_item_cases）**

| 列 | 类型 |
|---|---|
| menu_item_id | TEXT PK |
| case_id | TEXT PK |
| sort | INT NOT NULL |

**`channel_menu_item_extras`（新）**

| 列 | 类型 | 说明 |
|---|---|---|
| channel_id | TEXT NOT NULL | → channels.id |
| menu_item_id | TEXT NOT NULL | → channel_menu_items.id |
| extra_type | TEXT NOT NULL | 如 `tg_root_layout` |
| extra_json | TEXT NOT NULL | 平台差异数据（如 row/col 网格布局） |
| updated_at | DATETIME | |

约束：UNIQUE(channel_id, menu_item_id, extra_type)。

**`user_external_identities`（新）**

| 列 | 类型 | 说明 |
|---|---|---|
| id | TEXT PK | |
| user_id | TEXT NOT NULL, idx | → users.id |
| channel_id | TEXT NOT NULL | → channels.id |
| external_user_id | TEXT NOT NULL | 消息平台作用域用户 id（TG 存字符串化的 tg user id） |
| profile_json | TEXT | 平台资料（is_bot/is_premium 等） |
| last_seen_at | DATETIME NOT NULL | |

约束：UNIQUE(channel_id, external_user_id)。

**`users`（修改）**

| 列 | 类型 | 说明 |
|---|---|---|
| id | TEXT PK | 内部 UUID 保持不变 |
| username / first_name / last_name / language_code | TEXT | 保留为最新通用资料快照 |
| is_bot / is_premium | BOOL NULL | 迁移到 external profile_json 后可移除 |
| last_seen_at | DATETIME NOT NULL | |

**`sessions`（修改）**

| 列 | 类型 | 说明 |
|---|---|---|
| channel_id | TEXT NOT NULL, idx | 新 |
| chat_external_id | TEXT NOT NULL, idx | 新；索引 (channel_id, chat_external_id) |

**`platform_settings`（修改）**：移除 `telegram_token_cipher` 列，其余设置保留。

未涉及的表保持不变：`catalog_cases`、`tasks`、`edges`、`edge_metrics`、`bootstrap_meta`。

> 无存量数据：以上「修改后」表直接在全新库中创建，不实施任何数据迁移；现状表结构仅作重构前后对照参考。

### D5. 消息平台端口契约

定义端口（`internal/channel`）：

- `EventPort`：规范化入站事件（文本、媒体、回调动作、命令）
- `Messenger`：`SendText / SendMenu / SendList / SendMedia / SendImage`
- `MediaBridge`：上传/下载/转存（TG file_id ↔ blob；飞书 image_key ↔ blob）
- `IdentityResolver`：`(channel, external_user_id) → internal user`，upsert 资料

TG 回调前缀协议（`mf:`/`mc:`/`mb:`/`cp:`/`cs:`…）保留在 `internal/channel/tg` 内部，解析为规范化动作（OpenFolder/OpenCase/Back/StartCase/Confirm…）后再调用应用用例。新增平台只需实现端口 + 回调动作翻译。

### D6. 运行时装配与通知投递

进程启动时扫描启用的消息平台，为每个消息平台启动对应适配器（独立轮询/长连接 goroutine，带 graceful stop）；消息平台禁用触发对应适配器停止。`notifybridge` 从 tg 包泛化到 channel 层：通知携带消息平台地址，按消息平台路由到对应 `Messenger`。

### D7. 管理台与 API

- 侧栏「消息平台」替代「主键盘」；消息平台列表 → 新建向导（平台 + Token）→ 详情（基本信息 + 菜单配置 tab）
- 菜单编辑器去除 row/col 输入（改上下移动排序），kind 文案沿用现有五类但弱化平台词
- API：`/api/v1/channels` CRUD、`/api/v1/channels/{id}/menu` GET/PUT；`/api/v1/cases/{id}/menu-placements` 保持；旧 `/api/v1/tg-menu` 路由直接移除

## Risks / Trade-offs

- [ChatID 类型迁移波及 conversation/task/notify 持久化与跨进程消息] → 分两步：先加消息平台前缀语义与迁移脚本，再切换类型；旧值统一 `tg:` 前缀；notify JSON 兼容字段
- [凭证入库带来泄露面] → AES-GCM 加密 + masked 回显 + 文档警示；密钥经 env 注入
- [表结构替换引入回归] → 无存量数据，新 schema 全新建表；旧表模型代码随重构删除，测试覆盖等价行为
- [多适配器生命周期管理复杂] → 本期单 TG 消息平台；装配器以注册表 + 启停接口收敛，为多消息平台预留
- [重命名包引入大 diff] → 与行为改动分开提交，便于回归

## Migration Plan

无存量数据，本 change 不做数据迁移与兼容层，按依赖顺序直接落地：

1. 新增 `channels` 表与 `internal/channel` 领域；消息平台由管理台创建，凭证仅存消息平台
2. 新增 `channel_menus` / `channel_menu_items` / `channel_menu_item_cases`（row/col → order、无 `list_cases_by_tag`）；删除旧 `tg_menus*` 与 legacy JSON 表模型
3. 新增 `user_external_identities`；`users` 去掉 `tg_user_id`；`sessions` 直接使用 `channel_id` + `chat_external_id`
4. ChatID 类型切换（sharedkernel/conversation/task/notify 同步）
5. 新 API 上线并移除 `/api/v1/tg-menu`；管理台改版（消息平台 → 菜单）
6. 端口化重构 `internal/channel/tg`；notifybridge 泛化
7. 全量测试与 TG 手工回归后收尾

回滚：无数据迁移，直接回退代码（git revert）即可；不做旧路径/旧表兼容。

## Open Questions

- 凭证加密密钥的来源与管理（env key vs 外部 secret 管理）可在实现期按部署形态确定，不影响规格与任务拆分
- 多消息平台并行的真实优先级（本期仅单 TG）不影响本设计的数据模型预留
- 飞书对接细节（事件订阅方式、卡片交互、媒体桥）在 `feishu-channel` change 中细化
