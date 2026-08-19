# Comet Design Handoff

- Change: channel-platform-refactor
- Phase: design
- Mode: compact
- Context hash: 61bd8b96c6a9f5e9227753fdf0a40dc16e3018d80dc07ede88c05d635d23185e

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## docs/openspec/changes/channel-platform-refactor/proposal.md

- Source: docs/openspec/changes/channel-platform-refactor/proposal.md
- Lines: 1-37
- SHA256: 589f37eab96c67e4bb173003933840bc4ea090e16552173a5bceba01b6f5be3d

```md
## Why

主键盘（tgmenu）从领域模型到管理台全面耦合 Telegram：`row/col`、64 字节回调编码、`tg_*` 表与 `/api/v1/tg-menu` 路径直接进入领域模型，交互配置（5 种 kind + 挂载 Case + 行/列）对运营难以理解。后续要接入飞书、企业微信、钉钉时，现有结构无法复用。本次以「渠道」为一级实体完整重构渠道层：管理台先建渠道（选平台、填 Token），菜单配置收进渠道详情，运行时抽象为可插拔端口，为多平台铺路。

## What Changes

- **渠道管理（新）**：管理台一级模块「渠道」——创建渠道（平台 + 凭证，如 TG Bot Token）、列表、启停；Bot 运行时从渠道配置装配启动。
- **BREAKING** 菜单收进渠道：移除顶级「主键盘」模块，菜单编辑进入渠道详情；`tg_menus` / `tg_menu_items` / `tg_menu_item_cases` 替换为渠道作用域新表；菜单 API 由 `/api/v1/tg-menu` 改为 `/api/v1/channels/{id}/menu`。
- **BREAKING** 领域模型平台中立化：去除 `row/col`（改为有序列表）、TG 消息长度约束（如 `intro_text` 上限）与 `bot_id=default` 语义；`kind` 收敛为 `folder` / `open_case` / `placeholder` / `reply_media`（移除 `list_cases_by_tag` 兼容动作），平台专有约束下沉适配层。
- **渠道运行时端口（新）**：定义入站事件、出站消息、媒体桥、身份映射端口；`channel/tg` 重构为端口实现（ReplyKeyboard、Inline、回调编码、媒体下载全部收进 TG 适配器）。
- **身份渠道化**：`tg_user_id` 唯一约束扩展为渠道作用域外部身份（`channel + external_id`），用户主键仍为内部 UUID。
- **通知投递按渠道**：notify 按渠道适配器投递，为多渠道并存预留。
- 不改 Case 工作流本体（预览/填表/确认/执行/通知），不改 ComfyUI/队列/存储。
- 无存量数据：不保留旧表、旧字段、旧 API 路径的迁移与兼容层。

## Capabilities

### New Capabilities
- `channel-management`: 渠道实体（平台、凭证、启停）持久化、admin API 与管理台页面；Bot 运行时从渠道配置装配
- `channel-menu-config`: 渠道级菜单配置模型：平台中立的菜单树与动作语义，收进渠道详情，含迁移与 API
- `channel-runtime-ports`: 渠道运行时端口（事件/消息/媒体/身份）与适配器注册；TG 作为首个实现

### Modified Capabilities
- `channel-tg`: TG 适配器改为实现渠道端口；主菜单来源改为渠道作用域菜单；回调编码、媒体、身份下沉适配层
- `tg-menu`: 菜单模型改为渠道作用域并平台中立化（去 row/col、TG 约束），Case 挂载与反查语义保留
- `tg-menu-admin-api`: 菜单管理 API 改为渠道作用域路径（`/api/v1/channels/{id}/menu`），原路径迁移/兼容
- `user-directory`: 外部身份改为渠道作用域（`channel + external_id`），保留内部 UUID 主键
- `admin-web-shell`: 侧栏移除「主键盘」，新增「渠道」入口

## Impact

- 领域/包：`internal/channel`（新增渠道领域与端口）、`internal/tgmenu`（重构为渠道菜单）、`internal/channel/tg`（适配器化）、`internal/identity`（external_id 渠道化）、`internal/sharedkernel`（渠道类型）
- 持久化：新增渠道表；迁移 `tg_menus` / `tg_menu_items` / `tg_menu_item_cases`；`users` 外部身份渠道化
- API：admin-api 新增渠道 CRUD；菜单 API 路径与作用域变化
- 管理台：`web/admin` 新增渠道页；主键盘页移入渠道详情
- 配置：Bot Token 统一由渠道凭证管理，`platform_settings` / env 不再作为配置源
- 测试：领域/持久化/API/适配器测试随重构更新；TG 行为等价回归

```

## docs/openspec/changes/channel-platform-refactor/design.md

- Source: docs/openspec/changes/channel-platform-refactor/design.md
- Lines: 1-274
- SHA256: c5d26e779d72efb7024bd500bb9bc75691fc83aeb1942b205f7060df79cef4f9

[TRUNCATED]

```md
## Context

现状（动机见 proposal.md）：`internal/tgmenu` 领域模型携带 TG 专有字段（`row/col`、`intro_text ≤3500`、`bot_id=default`），持久化表与 API 路径以 `tg_` 命名；`internal/channel/tg` 的 `Messenger` 接口形状即 TG 渲染抽象，回调编码（64 字节前缀）散落在适配器编排逻辑；`identity` 以 `tg_user_id` 唯一约束；`notifybridge` 位于 tg 包内。Bot Token 当前存于 `platform_settings.telegram_token_cipher`（AES-GCM 密文）。完整 Case 工作流（`botapp.Facade`）已渠道无关，是本次重构的稳定内核。

约束：单仓库 Go 项目，admin-api 与 bot 可同进程；SQLite/GORM；管理台为 React（shadcn）；TG 为唯一现网平台，行为必须等价保留。

## Goals / Non-Goals

**Goals:**
- 「渠道」成为领域与管理台一级实体：平台、凭证、启停，驱动对应 Bot 运行时装配
- 菜单模型平台中立：无 row/col、无 TG 长度/回调约束，按渠道作用域存储
- 渠道运行时端口契约：入站事件、出站消息、媒体桥、身份映射；TG 适配器化
- 身份与通知渠道化；既有 TG 行为与数据平滑迁移

**Non-Goals:**
- 不实现飞书/企微/钉钉适配器（端口就绪即可）
- 本期不支持多渠道并行运行与多机器人切换 UI（数据模型预留）
- 不改 Case 工作流、ComfyUI/队列/存储
- 不做管理端图片上传

## Decisions

### D1. 领域分层：中立领域 + 渠道端口 + 平台适配器

```
internal/menu（原 tgmenu，重命名）   ← 平台中立菜单领域
internal/channel                     ← 渠道实体 + 端口契约（新）
internal/channel/tg                  ← TG 适配器（端口实现 + 渲染/回调/媒体）
internal/botapp + sharedkernel       ← 渠道无关应用内核（不动）
```

`internal/tgmenu` 重命名为 `internal/menu`，消除 tg 命名残留；目录移动 + import 更新为机械重构。菜单领域只表达「目录/动作意图」，渲染差异（键盘形态、按钮布局、回调编码、媒体上传）全部留在适配器。

备选：保留 `internal/tgmenu` 包名。放弃原因：命名即耦合，完整重构应同步清理。

### D2. 渠道实体与凭证

新增 `channels` 表：`id`（稳定字符串）、`platform`（如 `telegram`）、`name`、`credential_ciphertext`、`enabled`、时间戳。凭证以 AES-GCM 加密存储（密钥来自 env，开发环境可退化为明文并告警），API 回显 `masked_token`（如 `12345****`），永不回显完整 token。

渠道一律由管理台创建（选平台 + 填凭证）；无存量数据，不做 env / `platform_settings` 迁移，Bot Token 仅存于 `channels.credential_ciphertext`。

### D3. 菜单模型中立化

- `row/col` → 同层有序列表（`order`）；TG 适配器自行决定根层两列排布
- `intro_text` 长度上限移出领域校验，由 TG 适配器渲染时处理
- `bot_id` → `channel_id`，菜单文档唯一绑定渠道
- `kind` 收敛为四种：`folder` / `open_case` / `placeholder` / `reply_media`；移除 `list_cases_by_tag` 与 `tag` 字段（无存量兼容需求）
- 表：`channel_menus` / `channel_menu_items` / `channel_menu_item_cases`（替换 `tg_menus` 等）

### D3a. 不同渠道类型的菜单存放

**决策：中立核心模型（跨平台共有意图）+ 渠道扩展表（平台差异数据）+ 适配器收敛渲染；不抹平平台特色能力。**

- 每个渠道实例（`channels.id`）拥有一份菜单树：`channel_menus`（1:1）+ `channel_menu_items`（树）+ `channel_menu_item_cases`（Case 挂载）。不同渠道的数据以 `channel_id` 隔离，互不串扰。
- 中立核心字段只承载跨平台共有的「意图」：`label`、`order`、`enabled`、`kind`、`case_ids`。
- **渠道扩展表 `channel_menu_item_extras`**：按渠道存放平台差异数据（`channel_id` + `menu_item_id` + `extra_type` + JSON）。TG 特色能力在此保留：根层网格布局（row/col 精确排布）、每行按钮数、Inline 行为提示等；只对对应渠道生效，其他渠道不读取。
- 平台能力差异由适配器收敛：

| 平台 | 入口渲染 | 文件夹下钻 | 差异数据 |
|---|---|---|---|
| Telegram | ReplyKeyboard（优先读 extras 网格布局，缺省按 order 两列） | InlineKeyboard + 回调 | `tg_root_layout` 等 |
| 飞书（后续） | 机器人菜单 / 欢迎卡片按钮 | 交互卡片按钮（可原地更新） | 卡片标题/风格等 |
| 企微（后续） | 自定义菜单（列表式） | 菜单层级映射 | 层级上限提示等 |

- **不抹平原则**：中立模型只承载所有渠道都有的语义；TG 特有交互（根层网格布局、Inline 下钻、回复媒体、占位）不得因通用化而丢失，以 extras + 适配器完整保留。若某平台差异字段多到需要强类型列，可为该平台建专用扩展表（如 `tg_channel_extras`），本期以通用 extras 表实现并预留此扩展点。

备选方案与放弃原因：
- 把全部差异塞进中立核心模型：模型被平台字段污染（正是本次要解决的痛点），放弃；
- 按平台拆整套菜单表（`tg_menu_items` / `feishu_menu_items` …）：Case 挂载与反查需跨表 union，维护成本高；核心菜单共用一套，差异走 extras（用户已确认该方案）。

### D4. 会话寻址渠道化

`sharedkernel.ChatID` 由 `int64` 改为渠道命名的字符串地址（`tg:<chat_id>`，未来 `feishu:<chat_id>`），conversation/task/notify 的列与消息类型同步迁移。`sessions` 表拆分存储 `channel_id` + `chat_external_id` 两列（避免前缀解析查询，并为按渠道过滤建索引）；跨进程事件（notify）继续使用完整地址字符串。TG 适配器在入口处完成 `chat_id → "tg:xxx"` 映射，出口处反向映射。

备选：保留 int64 + 独立映射表。放弃原因：映射表增加间接层且 notify 跨进程传递时仍需转换；直接渠道化更彻底、更利于飞书字符串 chat_id。

## 数据库调整（现状 → 修改后）

### 现状（相关表）


```

Full source: docs/openspec/changes/channel-platform-refactor/design.md

## docs/openspec/changes/channel-platform-refactor/tasks.md

- Source: docs/openspec/changes/channel-platform-refactor/tasks.md
- Lines: 1-66
- SHA256: 6b67549b6048067877ffa8620aadd42c0e92769ead99084f1c8df5e43ef449c2

```md
## 1. 渠道领域与持久化

- [ ] 1.1 新增 `internal/channel` 领域：Channel 实体（id/platform/name/credential/enabled/时间戳）与 Repository 端口
- [ ] 1.2 GORM 实现 `channels` 表与 CRUD（含凭证加密/解密与 masked 回显）
- [ ] 1.3 凭证密钥注入：AES-GCM 密钥经 env 注入（开发可退化并告警）；无 env / `platform_settings` token 迁移
- [ ] 1.4 渠道应用服务：Create/List/Get/Update/Delete，校验平台合法、凭证非空、删除/停用语义

## 2. 菜单模型中立化（tgmenu → menu）

- [ ] 2.1 重命名 `internal/tgmenu` → `internal/menu` 并更新全部 import（domain/application/infrastructure/httpapi）
- [ ] 2.2 领域模型去 TG 字段：`row/col` → 有序列表 `order`；`intro_text` 上限移出领域校验；`bot_id` → `channel_id`；移除 `list_cases_by_tag` kind 与 `tag` 字段
- [ ] 2.3 校验规则更新：同层 label 唯一、深度上限、kind 关联约束保留；去除 row/col 与 tag 相关检查
- [ ] 2.4 新建 `channel_menus` / `channel_menu_items` / `channel_menu_item_cases` / `channel_menu_item_extras`；删除旧 `tg_menus*` 与 legacy JSON 表模型（无存量数据，不做迁移）
- [ ] 2.5 删除 `MigrateFromLegacyIfNeeded` 等旧数据迁移逻辑（无存量兼容需求）
- [ ] 2.6 默认种子按渠道生成：图片文件夹入口 + 挂载图片 Case；空渠道可读

## 3. 会话寻址渠道化

- [ ] 3.1 `sharedkernel.ChatID` 由 int64 改为渠道命名 string（`tg:` 前缀）并新增 ChannelAddr 辅助类型
- [ ] 3.2 `sessions` 新表直接使用 `channel_id` + `chat_external_id`（无存量迁移）；task/notify 事件载荷同步
- [ ] 3.3 notify 事件载荷 ChatID 字段切换并保持 JSON 兼容
- [ ] 3.4 TG 适配器入口/出口完成 chat_id ↔ `tg:xxx` 映射

## 4. 身份渠道化

- [ ] 4.1 新增 `user_external_identities`：`channel + external_id` 唯一约束；`users` 新模型不含 `tg_user_id` 列
- [ ] 4.2 upsert 按（channel, external_id）执行并刷新资料字段与 `last_seen_at`
- [ ] 4.3 用户查询/列表接口支持按渠道外部身份过滤（兼容旧 tg_user_id 查询）

## 5. 渠道端口契约与 TG 适配器重构

- [ ] 5.1 `internal/channel` 端口定义：EventPort / Messenger / MediaBridge / IdentityResolver
- [ ] 5.2 规范化事件与动作：文本/媒体/回调动作（OpenFolder/OpenCase/Back/StartCase/Confirm 等）
- [ ] 5.3 `channel/tg` 实现端口：回调前缀协议解析 → 规范化动作（64 字节编码留在 tg 内部）
- [ ] 5.4 TG 渲染下沉：ReplyKeyboard 优先读 extras 网格布局（缺省按 order 两列）、Inline、媒体上传与安全文件名、reply_media 图片；不抹平 TG 特色能力
- [ ] 5.5 适配器装配：从 Channel 凭证启动 bot（启用/禁用生命周期）

## 6. 运行时装配与通知投递

- [ ] 6.1 启动装配器：扫描启用渠道 → 启动对应适配器（独立 goroutine + graceful stop）
- [ ] 6.2 notifybridge 泛化到 channel 层：按渠道地址路由投递
- [ ] 6.3 通知幂等去重保留并按渠道隔离

## 7. 管理 API

- [ ] 7.1 `/api/v1/channels` CRUD（GET 列表 / POST / GET:id / PUT / DELETE）
- [ ] 7.2 `/api/v1/channels/{id}/menu` GET/PUT（渠道作用域校验，不存在渠道 404）
- [ ] 7.3 `/api/v1/cases/{id}/menu-placements` 返回含渠道标识的路径
- [ ] 7.4 移除旧 `/api/v1/tg-menu` 路由与 `internal/httpapi/tgmenu` 引用（无兼容需求）
- [ ] 7.5 admin-api host 挂载新渠道路由

## 8. 管理台改版

- [ ] 8.1 侧栏：「主键盘」→「渠道」；顺序为 Dashboard、实例、Case、渠道、Task、User、Session
- [ ] 8.2 渠道列表页 + 新建向导（选平台 + 名称 + Token）
- [ ] 8.3 渠道详情页：基本信息（masked token、启用/停用）tab
- [ ] 8.4 渠道菜单配置页：迁移原主键盘编辑器（排序控件；TG 渠道可在 extras 中配置根层网格布局）
- [ ] 8.5 i18n 中英文案：新增渠道相关文案，移除「主键盘」入口文案

## 9. 测试与回归

- [ ] 9.1 领域测试：order 排序、渠道作用域、去 row/col 后校验规则
- [ ] 9.2 持久化测试：渠道 CRUD、菜单迁移幂等、外部身份唯一
- [ ] 9.3 API 测试：channels CRUD、channel menu、placements 错误语义
- [ ] 9.4 适配器测试：回调动作翻译、chat_id 映射、通知路由与幂等
- [ ] 9.5 全量 `go test ./...` 通过 + TG 手工回归（六键主键盘、文件夹下钻、Case 流程、出图通知）

```

## docs/openspec/changes/channel-platform-refactor/specs/admin-web-shell/spec.md

- Source: docs/openspec/changes/channel-platform-refactor/specs/admin-web-shell/spec.md
- Lines: 1-35
- SHA256: 4fbb4bae6c5981c64268328e19622ae73789af6ea053f9211a67263154a6049e

```md
## MODIFIED Requirements

### Requirement: 菜单模块
系统 MUST 提供后台菜单模块，将菜单项映射到路由，并在侧栏展示 Dashboard 与全部资源入口；菜单顺序 MUST 为 Dashboard、实例、Case、渠道、Task、User、Session；「主键盘」不再作为一级菜单项。

#### Scenario: 侧栏展示菜单
- **WHEN** 用户打开控制台
- **THEN** 侧栏可见 Dashboard、实例、Case、渠道、Task、User、Session 菜单项，且顺序如上，且不含「主键盘」

#### Scenario: 点击菜单进入对应路由
- **WHEN** 用户点击某一菜单项
- **THEN** 导航到该菜单绑定的页面路由

## REMOVED Requirements

### Requirement: 侧栏包含 TG Menu 入口
**Reason**: 管理台改版为「渠道」一级实体，主键盘菜单配置收进渠道详情。
**Migration**: 侧栏「渠道」入口替代；原 TG Menu 编辑能力迁移至渠道详情内的菜单配置页。

## ADDED Requirements

### Requirement: 侧栏包含渠道入口
管理控制台侧栏 MUST 提供「渠道」入口并映射到渠道列表路由；渠道详情内 MUST 提供该渠道的菜单配置入口（原「主键盘」编辑能力）。

#### Scenario: 侧栏可见渠道入口
- **WHEN** 用户打开管理控制台
- **THEN** 侧栏可见「渠道」管理入口且可点击进入渠道列表

#### Scenario: 渠道详情可编辑菜单
- **WHEN** 用户进入某渠道详情
- **THEN** 可进入该渠道的菜单配置页进行编辑

#### Scenario: 中英切换含渠道文案
- **WHEN** 用户切换中/英文
- **THEN** 「渠道」相关侧栏与页面关键文案随语言切换

```

## docs/openspec/changes/channel-platform-refactor/specs/channel-management/spec.md

- Source: docs/openspec/changes/channel-platform-refactor/specs/channel-management/spec.md
- Lines: 1-68
- SHA256: 1aaafa063c600dcc560835dbbc37c392e783e285f485ec98eaa7c37536574630

```md
## Purpose

渠道是管理台的一级实体：代表接入某个消息平台（Telegram、飞书、企业微信、钉钉等）的配置实例，包含平台类型、凭证与启停状态。渠道管理负责创建、查看、启停渠道并驱动对应 Bot 运行时装配，使菜单配置与消息投递获得明确归属。

## ADDED Requirements

### Requirement: 渠道实体与持久化
系统 MUST 将渠道持久化为独立实体，字段至少包含：渠道 id（稳定字符串）、平台类型（如 `telegram`）、显示名称、凭证（如 TG Bot Token）、启用状态、创建/更新时间。同一平台 MAY 存在多个渠道实例（数据模型层面预留）。

#### Scenario: 创建渠道
- **WHEN** 管理员在管理台创建渠道并选择平台 Telegram、填写 Bot Token
- **THEN** 系统保存该渠道并可列出；凭证以安全方式存储，不直接明文回显

#### Scenario: 渠道列表与详情
- **WHEN** 管理员打开渠道管理页
- **THEN** 可见全部渠道的平台、名称、启用状态，并可进入详情查看与编辑

### Requirement: 渠道 CRUD 管理 API
admin-api MUST 暴露渠道管理 HTTP 接口，支持创建、列表、详情、更新与删除（或等价启停语义）；鉴权策略与既有 admin-api 一致（仅内网约定）。

#### Scenario: 创建并读取渠道
- **WHEN** 客户端 POST 创建渠道（平台 + 凭证 + 名称）后 GET 列表
- **THEN** 返回 200 且列表包含新渠道

#### Scenario: 非法平台被拒绝
- **WHEN** 客户端提交未知平台类型或空凭证
- **THEN** 返回 4xx 与错误说明，且不创建渠道

### Requirement: 运行时按渠道装配
系统 MUST 依据启用的渠道配置装配 Bot 运行时：渠道启用且凭证有效时对应平台适配器启动；渠道禁用或凭证缺失时对应适配器 MUST NOT 启动。Bot Token 仅存于渠道凭证，MUST NOT 从环境变量或 `platform_settings` 读取。

#### Scenario: 启用渠道后 Bot 可启动
- **WHEN** 存在启用且凭证完整的 Telegram 渠道
- **THEN** 运行时启动对应 TG Bot，主菜单可响应

#### Scenario: 禁用渠道后 Bot 不启动
- **WHEN** 渠道被禁用
- **THEN** 对应平台适配器不启动（或停止），不再处理该渠道事件

#### Scenario: 无凭证渠道不启动
- **WHEN** 渠道启用但凭证缺失或为空
- **THEN** 对应平台适配器 MUST NOT 启动，并记录可诊断错误

### Requirement: 渠道变更热生效
渠道创建、启用/停用或凭证更新后，系统 MUST 在无需重启进程的情况下使对应适配器生效：启用/创建 → 启动适配器；停用 → 停止适配器；凭证更新 → 重建适配器。生效延迟 MUST 在数秒量级。

#### Scenario: 创建渠道后热启动
- **WHEN** 管理员创建启用且凭证完整的 Telegram 渠道
- **THEN** 无需重启进程，适配器在数秒内启动并可响应消息

#### Scenario: 停用渠道热停止
- **WHEN** 管理员停用运行中的渠道
- **THEN** 对应适配器在数秒内停止，不再处理事件

#### Scenario: 凭证更新后重建
- **WHEN** 管理员更新运行中渠道的 Bot Token
- **THEN** 适配器以新凭证重建，旧连接停止

### Requirement: 渠道删除受限
系统 MUST 禁止删除未禁用或仍被引用的渠道：删除仅在渠道已禁用且无活跃会话/任务引用时允许（或采用软删除）；违规删除 MUST 返回 4xx 与可理解原因。

#### Scenario: 启用中删除被拒
- **WHEN** 客户端删除启用中的渠道
- **THEN** 返回 4xx，渠道保持不变

#### Scenario: 禁用且无引用可删除
- **WHEN** 渠道已禁用且无活跃会话/任务引用
- **THEN** 删除成功（或软删除后不可见）

```

## docs/openspec/changes/channel-platform-refactor/specs/channel-menu-config/spec.md

- Source: docs/openspec/changes/channel-platform-refactor/specs/channel-menu-config/spec.md
- Lines: 1-68
- SHA256: 01e6ce9156381374baba295056cfa83d35190ae94e7753d4455a1d34aebc03e2

```md
## Purpose

渠道菜单配置定义每个渠道作用域下的可配置菜单树：平台中立的目录与动作语义、Case 挂载与反查。菜单编辑入口位于渠道详情内，管理台不再存在独立的「主键盘」顶级模块。

## ADDED Requirements

### Requirement: 菜单按渠道作用域存储
系统 MUST 将菜单配置归属到渠道：每个渠道一份菜单文档，包含树形菜单项与菜单项–Case 关联；菜单读取与替换均以渠道 id 为作用域，不同渠道互不影响。

#### Scenario: 渠道各自菜单独立
- **WHEN** 存在两个渠道并分别保存菜单
- **THEN** 各渠道读取到的菜单互不串扰

#### Scenario: 不存在渠道被拒绝
- **WHEN** 对不存在的渠道 id 读取或保存菜单
- **THEN** 返回 404 且不创建隐式菜单

### Requirement: 菜单模型平台中立
菜单领域模型 MUST 使用平台中立的目录/动作语义（文件夹、打开 Case、占位提示、回复媒体），中立核心字段 MUST NOT 包含平台专有字段（如 Telegram 行/列网格、回调编码、消息长度上限）；平台差异数据 MAY 通过渠道扩展数据（extras）按渠道存放，MUST NOT 进入中立核心字段；平台渲染差异（键盘形态、按钮布局、回调数据、媒体上传）MUST 由对应渠道适配器处理，且 MUST NOT 因通用化抹平平台特色能力。

#### Scenario: 中立字段不含 TG 专有字段
- **WHEN** 通过管理 API 读取或提交菜单的中立字段
- **THEN** 中立字段中不存在 row/col、回调前缀或 TG 消息长度约束；平台差异仅出现在该渠道的 extras 数据中

#### Scenario: TG 特色能力保留
- **WHEN** 管理员为 TG 渠道配置根层网格布局等 TG 特色数据
- **THEN** 数据存入该渠道 extras，TG 适配器按 extras 渲染；其他渠道读取不到且不受影响

#### Scenario: 同一菜单可被多平台适配器渲染
- **WHEN** 同一份渠道菜单（folder + open_case + placeholder）分别由 TG 与后续平台适配器消费
- **THEN** 各适配器能按各自平台形态渲染，无需改写菜单配置

### Requirement: Case 挂载与反查
系统 MUST 支持菜单项挂载 Case 并从 Case 反查其出现的菜单路径；挂载的 Case 必须存在，否则拒绝保存。

#### Scenario: 保存文件夹并挂载 Case
- **WHEN** 管理面在渠道菜单中创建文件夹并关联已存在的 Case 后保存
- **THEN** 持久化成功，后续读取可见文件夹及其 Case 关联

#### Scenario: 绑定不存在的 Case 被拒绝
- **WHEN** 提交的关联 `case_id` 不存在
- **THEN** 系统拒绝保存并返回可理解的校验错误

#### Scenario: 查询 Case 的菜单挂载
- **WHEN** 调用方请求某 Case 的菜单挂载
- **THEN** 系统返回零条或多条路径；每条能标识所在渠道、菜单项与可读路径

### Requirement: 默认种子按渠道
系统在渠道无自定义菜单时 MUST 提供可用的默认种子菜单；种子包含「图片」文件夹入口并可挂载图片类 Case。

#### Scenario: 空配置使用默认种子
- **WHEN** 渠道尚无自定义菜单
- **THEN** 运行时仍能得到可用的根菜单项集合

### Requirement: 渠道差异数据（extras）管理
系统 MUST 支持按渠道为菜单项保存平台差异数据（extras）：每条 extras 以（channel_id, menu_item_id, extra_type）唯一标识，内容为 JSON；仅对应渠道的适配器读取，其他渠道不得受影响。无效或未知的 extras MUST 被忽略或按适配器校验规则拒绝，MUST NOT 影响中立字段语义。管理台 MUST 提供 extras 的可视化编辑入口。

#### Scenario: 按渠道保存与读取 extras
- **WHEN** 管理面为 TG 渠道某根菜单项保存网格布局 extras 后再次读取
- **THEN** 该渠道可读回该 extras，其他渠道读取不到

#### Scenario: 管理台编辑 extras
- **WHEN** 管理员在 TG 渠道菜单中编辑根层网格布局
- **THEN** 变更写入该渠道 extras，保存后按适配器刷新策略生效

#### Scenario: 无效 extras 不影响中立字段
- **WHEN** extras 内容为无效或未知类型
- **THEN** 适配器忽略或拒绝该 extras，菜单中立字段行为不受影响

```

## docs/openspec/changes/channel-platform-refactor/specs/channel-runtime-ports/spec.md

- Source: docs/openspec/changes/channel-platform-refactor/specs/channel-runtime-ports/spec.md
- Lines: 1-48
- SHA256: 3d4380bd2335999001f68b692f51eb242471b0511c439ba716529d19acc06ead

```md
## Purpose

渠道运行时端口定义消息平台接入的统一契约：入站事件、出站消息、媒体桥与身份映射。`channel/tg` 重构为这些端口的一个实现，使飞书、企业微信、钉钉等新平台可通过实现端口接入，而不改动应用层与菜单模型。

## ADDED Requirements

### Requirement: 渠道端口契约
系统 MUST 定义渠道运行时端口，至少覆盖：入站事件（消息/回调查询/媒体）、出站消息（文本/菜单/内联按钮/媒体）、媒体桥（上传/下载/转存）与身份映射（外部用户 → 内部用户）。应用层 MUST 仅依赖端口类型，MUST NOT 依赖具体平台 SDK。

#### Scenario: 应用层不感知平台
- **WHEN** 新增一个平台适配器且不改动 Case 工作流应用层
- **THEN** 应用层代码不出现该平台 SDK 类型

### Requirement: 适配器注册与装配
系统 MUST 按渠道实例选择对应平台适配器实现，并完成事件订阅与消息发送绑定；渠道停用时对应适配器停止消费。

#### Scenario: 渠道装配对应适配器
- **WHEN** 存在平台类型为 telegram 的启用渠道
- **THEN** 运行时装配 TG 适配器并处理该渠道事件

#### Scenario: 渠道变更驱动适配器启停
- **WHEN** 渠道由启用变为停用，或凭证更新
- **THEN** 装配器停止旧适配器（凭证更新时以新配置重建），无需重启进程

### Requirement: 身份渠道化
用户外部身份 MUST 以（渠道 + 渠道外部用户 id）唯一标识；系统 MUST 将其映射到内部用户主键（UUID），用于 Session/Task 归属；同一内部用户 MAY 关联多个渠道的外部身份。

#### Scenario: 渠道外部 id 唯一
- **WHEN** 同一渠道出现相同外部用户 id
- **THEN** 仍对应同一条内部用户记录（id 不变）

#### Scenario: 不同渠道身份并存
- **WHEN** 同一自然人在 TG 与后续平台各有一个外部 id
- **THEN** 分别存在对应渠道的外部身份记录，且不因外部 id 格式冲突

### Requirement: 通知按渠道投递
系统 MUST 将用户通知（任务状态、结果媒体）按渠道适配器投递到对应用户/会话；同一通知 MUST 幂等处理，避免重复投递。

#### Scenario: 结果按渠道送达
- **WHEN** 任务成功且输出含图片 BlobRef
- **THEN** 对应渠道适配器向用户发送图片（或等价媒体消息），重复通知不重复刷屏

### Requirement: 平台专有实现隔离
平台专有逻辑（键盘形态、回调数据编码、消息长度限制、媒体上传/下载、凭证类型）MUST 位于各渠道适配器内部，MUST NOT 泄漏到菜单模型、应用用例或共享内核。

#### Scenario: 新增平台只改适配器
- **WHEN** 接入新平台（如飞书）且不改变菜单模型
- **THEN** 变更范围限于新适配器与渠道装配/注册，既有 TG 行为不受影响

```

## docs/openspec/changes/channel-platform-refactor/specs/channel-tg/spec.md

- Source: docs/openspec/changes/channel-platform-refactor/specs/channel-tg/spec.md
- Lines: 1-36
- SHA256: e7027772c4a089d5574bbc2d52dfb68137c45326bc728d58151918d29d0e0ade

```md
## MODIFIED Requirements

### Requirement: TG 适配器将应用 DTO 渲染为 Bot API 消息
系统 MUST 提供 Telegram 适配器：作为渠道运行时端口实现，接收 Bot Update，调用渠道无关的应用用例，并将菜单/Case 列表/会话提示/错误/结果渲染为 Telegram 支持的消息形态（文本、Photo、InlineKeyboard 等）。应用层 MUST NOT 依赖 Telegram SDK 类型。发送 Photo 时文件名 MUST 为无路径分隔符的安全基名，避免因 blob key 含 `/` 导致投递失败。

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
Telegram 适配器展示的主 ReplyKeyboard MUST 由**渠道作用域**菜单根项配置（含默认种子）生成，MUST NOT 再以源码常量作为唯一长期配置源；菜单读取以渠道 id 为作用域。

#### Scenario: /start 展示配置中的主键盘
- **WHEN** 用户发送 `/start` 或打开主菜单，且该渠道菜单配置可用
- **THEN** 用户收到与配置根项顺序/文案一致的 ReplyKeyboard

## ADDED Requirements

### Requirement: 适配器由渠道配置装配
TG 适配器 MUST 由渠道实体装配启动：Bot Token 来自渠道凭证；启用渠道对应适配器运行，禁用渠道对应适配器停止。凭证 MUST NOT 从环境变量或 `platform_settings` 读取。

#### Scenario: 渠道凭证驱动 Bot
- **WHEN** 管理员在渠道中填写/更新 Telegram Bot Token 并启用
- **THEN** Bot 使用该凭证运行，无需手工修改环境变量后重启

#### Scenario: 禁用渠道停止适配器
- **WHEN** 渠道被禁用
- **THEN** TG 适配器不再处理该渠道事件

```

## docs/openspec/changes/channel-platform-refactor/specs/tg-menu/spec.md

- Source: docs/openspec/changes/channel-platform-refactor/specs/tg-menu/spec.md
- Lines: 1-34
- SHA256: 08c0f93cb89f56daab27bf5a46280388f3511a20da8f28a1ff8fb9077fada775

```md
## MODIFIED Requirements

### Requirement: Menu 树持久化
系统 MUST 将**渠道菜单**持久化为渠道作用域的菜单文档、菜单项（含可选父项）与菜单项–Case 关联。根项（无父项）MUST 可用于渠道入口渲染；子项与关联 Case MUST 可用于目录下的浏览与选择。模型 MUST NOT 包含平台专有字段（如 Telegram 行/列网格、回调编码、消息长度上限）。

#### Scenario: 读取当前菜单树
- **WHEN** 调用方请求某渠道的菜单配置
- **THEN** 系统返回树形结构（含子项与挂载的 Case 标识），字段完整可读

#### Scenario: 保存文件夹并挂载 Case
- **WHEN** 管理面在渠道菜单中创建或更新一文件夹项，并关联一个或多个已存在的 Case 后保存
- **THEN** 持久化成功；后续读取可见该文件夹及其 Case 关联

#### Scenario: 绑定不存在的 Case 被拒绝
- **WHEN** 提交的关联 `case_id` 在 Case 目录中不存在
- **THEN** 系统拒绝保存并返回可理解的校验错误

### Requirement: 默认种子与兼容
系统在渠道无自定义菜单时 MUST 提供可用的默认入口种子；其中「图片」类入口 MUST 以文件夹（或等价可下钻节点）表达，并可挂载图片类 Case。

#### Scenario: 空配置使用默认种子
- **WHEN** 渠道库中尚无自定义菜单
- **THEN** 运行时仍能得到可用的根菜单项集合

### Requirement: 非 Case 动作允许占位与回复媒体
系统 MUST 允许占位（placeholder）与回复媒体（reply_media：文本与/或 http(s) 图片 URL）类动作；回复媒体保存时 MUST 要求文本与图片列表至少其一非空。本期 MUST NOT 要求管理端本地上传图片；图片 URL 的合法性校验（http(s) 绝对地址）保持。

#### Scenario: 保存回复媒体
- **WHEN** 管理面保存合法 reply_media 项
- **THEN** 持久化成功

#### Scenario: 空回复被拒绝
- **WHEN** reply_media 的文本与图片皆空
- **THEN** 系统拒绝保存

```

## docs/openspec/changes/channel-platform-refactor/specs/tg-menu-admin-api/spec.md

- Source: docs/openspec/changes/channel-platform-refactor/specs/tg-menu-admin-api/spec.md
- Lines: 1-38
- SHA256: 4dff3e2d3bf650ee8534e16848dda5f042c1f97b09aeea1d8ba3aaaef4d4b061

```md
## MODIFIED Requirements

### Requirement: Menu 树管理 HTTP 接口
admin-api MUST 暴露渠道菜单管理接口（路径约定 `/api/v1/channels/{id}/menu`）：支持按渠道获取完整树形配置，以及整棵树写回。接口鉴权策略与现有 admin-api 一致（仅内网约定）。

#### Scenario: 获取渠道菜单树
- **WHEN** 客户端 GET `/api/v1/channels/{id}/menu` 且服务可用
- **THEN** 返回 200 与当前树形菜单 JSON（含子项与 Case 关联标识）

#### Scenario: 更新渠道菜单树
- **WHEN** 客户端提交合法菜单树进行更新
- **THEN** 返回成功，且随后 GET 可见更新结果

#### Scenario: 非法配置被拒绝
- **WHEN** 客户端提交缺必填、非法 kind、无效 `case_id`、空 reply_media 或违反同层 label 唯一等规则的配置
- **THEN** 返回 4xx 与错误说明，且不破坏既有已保存配置

#### Scenario: 回复媒体非法 URL 被拒绝
- **WHEN** 客户端提交 reply_media 且图片列表含非 http(s) 绝对 URL
- **THEN** 返回 4xx，且不破坏既有已保存配置

### Requirement: Case 菜单挂载查询接口
admin-api MUST 暴露按 Case id 查询菜单挂载的接口（路径约定 `/api/v1/cases/{id}/menu-placements`），返回该 Case 出现的渠道与菜单路径列表。

#### Scenario: 查询已挂载 Case
- **WHEN** 客户端 GET 某已挂到菜单的 Case 的 placements
- **THEN** 返回 200 与至少一条可读路径（含渠道标识）

#### Scenario: 未挂载 Case
- **WHEN** 客户端 GET 未出现在任何菜单项关联中的 Case 的 placements
- **THEN** 返回 200 与空列表

### Requirement: 仅经 admin-api 管理
菜单的管理写路径 MUST 仅通过 admin-api；Bot 进程 MUST NOT 再依赖改源码作为常规配置手段（默认种子除外）。

#### Scenario: 控制台经 admin-api 读写
- **WHEN** 管理前端在渠道详情中保存菜单或加载 Case 挂载
- **THEN** 请求指向 admin-api 的渠道菜单路径，而非 bot 管理残留路径

```

## docs/openspec/changes/channel-platform-refactor/specs/user-directory/spec.md

- Source: docs/openspec/changes/channel-platform-refactor/specs/user-directory/spec.md
- Lines: 1-19
- SHA256: 8358d7d610d6cf560920575841617708dd85c41fcee45bb771e20006d2e0c3e8

```md
## MODIFIED Requirements

### Requirement: User 持久化与主键
系统 MUST 将用户记录持久化到数据库。每条记录 MUST 使用内部 UUID（或等价内部 id）作为主键，并 MUST 对**渠道外部身份**（渠道 + 渠道外部用户 id）建立唯一约束；Telegram 用户 id 作为 `telegram` 渠道的外部身份。系统 MUST NOT 仅用 `chat_id` 充当唯一用户主键。

#### Scenario: 同一渠道外部身份不重复建档
- **WHEN** 同一渠道同一外部用户 id 再次触发 upsert
- **THEN** 仍对应同一条内部用户记录（id 不变）

### Requirement: TG 资料字段与 upsert
系统 MUST 在处理渠道消息或回调查询的路径上，根据平台来源（如 Telegram `From`）按（渠道 + 外部 id）upsert 用户资料。Telegram 渠道存储字段 MUST 覆盖 From 可获得的常用身份信息，至少包括：`username`、`first_name`、`last_name`、`language_code`、`is_bot`、`is_premium`（若 API 提供；不可得则可空）。系统 MUST 维护 `last_seen_at`（或等价），并在 upsert 时刷新可变资料字段。其他渠道 MAY 存储其平台提供的等价资料字段。

#### Scenario: 消息到达后可查到用户资料
- **WHEN** 用户（telegram 渠道）首次向 bot 发消息且 From 含 username 与 first_name
- **THEN** 数据库中存在对应渠道外部身份的用户行，且上述字段已写入

#### Scenario: 资料变更被刷新
- **WHEN** 同一用户稍后消息中 username 已变更
- **THEN** upsert 后该用户行的 username 为新值，且 last_seen_at 更新

```
