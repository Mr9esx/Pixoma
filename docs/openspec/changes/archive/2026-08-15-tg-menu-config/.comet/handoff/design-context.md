# Comet Design Handoff

- Change: tg-menu-config
- Phase: design
- Mode: compact
- Context hash: 1278ddf43d341660ab4844df54ecb9061791a421f11f815abea3674e84be9b42

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## docs/openspec/changes/tg-menu-config/proposal.md

- Source: docs/openspec/changes/tg-menu-config/proposal.md
- Lines: 1-29
- SHA256: 919dd758036260baf7c7fb9be75fb5132132656062a1aa7b065772b2bfd92982

```md
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

```

## docs/openspec/changes/tg-menu-config/design.md

- Source: docs/openspec/changes/tg-menu-config/design.md
- Lines: 1-59
- SHA256: f7dc0b7d65a061af50f774745c0e5ca14cfb7fd23fd90087e0407cde69605dc8

```md
## Context

参见 `proposal.md`。当前单 Bot：`telegram_bot_token` / `TG_BOT_TOKEN`；主菜单硬编码于 `internal/channel/tg/menu.go`；Case 为全局目录；admin-api 与 `web/admin` 已具备 Case 等资源管理，但无 Menu。本期**不做**多 Bot / `bot_id` 租户。

## Goals / Non-Goals

**Goals:**
- Menu 落库 + 默认种子
- Bot 读配置生成主键盘并路由动作（含 Case 绑定）
- admin-api + 控制台可编辑 Menu

**Non-Goals:**
- 多 Bot / 租户字段改造
- Bot Token 后台管理
- 充值/签到等业务实现（占位即可）
- 前端 mock

## Decisions

1. **单租户 Menu 文档**  
   采用「一份当前 Menu 配置」（单行文档或版本表取最新）而非多 Bot 作用域。理由：明确降难度；后续若做多 Bot 再加 `bot_id`。  
   备选：YAML 文件 — 不利于后台编辑与校验，否决。

2. **动作模型**  
   - `open_case`：必填 `case_id` → 现有 Case 工作流  
   - `list_cases_by_tag`：必填 `tag`（种子「图片」=`image`）  
   - `placeholder`：可选提示文案  
   - `reply_media`：`reply.text` / `reply.images[]`（http(s) URL）至少其一；点击后发文字与图片；本期不做管理端上传  

3. **包边界**  
   - 领域+持久化：新建小模块（如 `internal/tgmenu`）或挂在 catalog 邻域；**禁止** admin-api 依赖 `channel/tg`。  
   - `channel/tg` 只依赖 menu 读取端口。  
   - HTTP：`internal/httpapi/tgmenu` 挂到 admin-api。

4. **刷新策略**  
   Adapter 每次构建主键盘时读仓储（或进程内短缓存 ≤ 几秒）。不要求热推送。

5. **控制台 IA**  
   侧栏增加「TG 菜单」；页用列表+编辑（可 Master–Detail 或单页表单）。Case 选择复用 `listCases`。

## Risks / Trade-offs

- [点击文案匹配脆弱] → 菜单项保留稳定 `id`；路由优先用 id/callback，ReplyKeyboard 文案变更时用「文案→项」映射表并在保存时校验唯一文案。  
- [与现网「图片→tag 列表」差异] → 种子保留 `list_cases_by_tag`；纯 `open_case` 不够表达列表时以该动作为准。  
- [空/坏配置导致 Bot 无键盘] → 读取失败回退默认种子并打日志。  
- [无鉴权写 Menu] → 与现 admin-api 一致，仅内网；文档警示。

## Migration Plan

1. 建表 + 默认种子 upsert（对齐现 `MainMenuRows`）。  
2. Adapter 切读配置；保留短期兼容或直接替换硬编码布局。  
3. admin-api + 控制台上线后用 UI 改绑定验证。  
4. 回滚：关读库开关或回退镜像；表可保留。

## Open Questions

- （已关闭）多 Bot：本期不做  
- （已关闭）Token 后台管理：本期不做  
- `list_cases_by_tag` 是否保留：做，用于兼容「图片」入口

```

## docs/openspec/changes/tg-menu-config/tasks.md

- Source: docs/openspec/changes/tg-menu-config/tasks.md
- Lines: 1-29
- SHA256: d3ce6a0b394cca7116ee607f22a6731030636ccb9315051235b4fa5e81f8220e

```md
## 1. Menu 领域与持久化

- [ ] 1.1 新增 Menu 领域模型（项 id、文案、顺序、启用、动作类型、case_id/tag、reply_media 的 reply 等）与仓储接口
- [ ] 1.2 实现 SQLite/GORM 持久化 + 迁移；空库默认种子对齐现网主菜单（图片=list_cases_by_tag:image，其余 placeholder）
- [ ] 1.3 校验：open_case 需存在 case_id；reply_media 至少 text 或 images；images 须 http(s)；文案唯一；非法配置拒绝写入
- [ ] 1.4 领域/持久化单测（种子、更新、校验失败、reply_media）

## 2. Bot 运行时接入

- [ ] 2.1 `channel/tg` 构建主 ReplyKeyboard 改为读取 Menu 仓储（失败回退种子）；去掉对硬编码布局的唯一依赖
- [ ] 2.2 点击路由：open_case → 既有 Case 工作流；list_cases_by_tag → 既有列表；placeholder → 提示文案；reply_media → 发文本与按 URL 发图（单张失败不崩）
- [ ] 2.3 组合根注入 Menu 仓储；补充/调整 TG 相关测试

## 3. admin-api

- [ ] 3.1 实现 `httpapi` Menu handler：GET/PUT（或等价）`/api/v1/tg-menu`
- [ ] 3.2 挂载到 admin-api；保持不依赖 `channel/tg`
- [ ] 3.3 handler 单测（合法更新、非法 case_id、读回一致）

## 4. 管理控制台

- [ ] 4.1 侧栏增加「TG 菜单」入口 + 中英 i18n
- [ ] 4.2 Menu 管理页：列表/编辑/排序/动作与 Case 选择、tag、placeholder、reply_media（文本+图片 URL）；调用 admin-api
- [ ] 4.3 空态/错误/保存反馈与现有资源页一致；README 联调补充 Menu

## 5. 文档与验收

- [ ] 5.1 同步 `docs/architecture/data-model.md`（及必要时 runtime/overview）中 Menu 表与主链路说明
- [ ] 5.2 联调验收：改文案/绑 Case 后 TG `/start` 键盘与入口行为符合预期；Network 仅打 admin-api

```

## docs/openspec/changes/tg-menu-config/specs/admin-resource-pages/spec.md

- Source: docs/openspec/changes/tg-menu-config/specs/admin-resource-pages/spec.md
- Lines: 1-23
- SHA256: e8788d2e3c9007a48fd18bfee7b1024baa41daebc55e315a87c5c1e418338669

```md
## ADDED Requirements

### Requirement: TG Menu 管理页
控制台 MUST 提供 TG Menu 管理页：展示当前菜单项列表，支持编辑文案、顺序、启用、动作类型，以及 Case 绑定（选择已有 Case id）、按 tag 列表、占位提示，以及回复媒体（文本与图片 URL 列表）。保存 MUST 调用 admin-api Menu 接口；失败时 MUST 展示错误且不假装成功。本期 MUST NOT 要求浏览器本地上传图片文件到对象存储（图片以 URL 配置）。

#### Scenario: 编辑并保存绑定 Case
- **WHEN** 运维将某菜单项动作设为进入 Case，选择合法 Case 并保存
- **THEN** 页面提示成功，刷新后仍显示该绑定

#### Scenario: 编辑并保存回复媒体
- **WHEN** 运维将某菜单项动作设为回复媒体，填写文本与图片 URL 并保存成功
- **THEN** 页面提示成功，刷新后仍显示该回复配置

#### Scenario: 保存失败展示错误
- **WHEN** admin-api 返回校验或网络错误
- **THEN** 页面展示错误信息，本地不进入「已保存」误导态

### Requirement: 请求仅指向 admin-api
Menu 页的读写请求 MUST 仅使用 `VITE_ADMIN_API_BASE` 指向的 admin-api；MUST NOT 引入前端 mock 作为验收路径。

#### Scenario: Network 指向 admin-api
- **WHEN** 运维在 Menu 页加载或保存
- **THEN** 浏览器请求前缀为配置的 admin-api 基址 + Menu API 路径

```

## docs/openspec/changes/tg-menu-config/specs/admin-web-shell/spec.md

- Source: docs/openspec/changes/tg-menu-config/specs/admin-web-shell/spec.md
- Lines: 1-12
- SHA256: a7689230c2325bfb047e4dc1cb386b18086ca556819eeb843363f159c731a957

```md
## ADDED Requirements

### Requirement: 侧栏包含 TG Menu 入口
管理控制台侧栏 MUST 提供 TG Menu（或产品约定中文名，如「TG 菜单」）入口，并映射到对应路由。本期侧栏在既有六项基础上增加该项；顺序建议置于 Case 附近（具体顺序在实现中与 i18n 文案一并固定）。

#### Scenario: 侧栏可见 Menu 入口
- **WHEN** 用户打开管理控制台
- **THEN** 侧栏可见 TG Menu 管理入口且可点击进入

#### Scenario: 中英切换含 Menu 文案
- **WHEN** 用户切换中/英文
- **THEN** TG Menu 侧栏与页面关键文案随语言切换

```

## docs/openspec/changes/tg-menu-config/specs/channel-tg/spec.md

- Source: docs/openspec/changes/tg-menu-config/specs/channel-tg/spec.md
- Lines: 1-31
- SHA256: ce2ce7ed0e57fe545c3bc1dc362560fd6208050617af5576428a7bb86ceb85e1

```md
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

```

## docs/openspec/changes/tg-menu-config/specs/tg-menu/spec.md

- Source: docs/openspec/changes/tg-menu-config/specs/tg-menu/spec.md
- Lines: 1-49
- SHA256: 57a7302e5799b9f7c212eefc0f2e30a006c293c8bdf0d6ed08a2cf5683888e83

```md
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

```

## docs/openspec/changes/tg-menu-config/specs/tg-menu-admin-api/spec.md

- Source: docs/openspec/changes/tg-menu-config/specs/tg-menu-admin-api/spec.md
- Lines: 1-31
- SHA256: 25cdb33254feaf5614469d3eb270fa399fc0767a702325d7a367616797ae48a2

```md
## Purpose

为管理端提供 Telegram 主菜单配置的 HTTP API，使运维可在 admin-api 上查询与更新 Menu，无需改代码发版。

## ADDED Requirements

### Requirement: Menu 管理 HTTP 接口
admin-api MUST 暴露 Menu 管理接口（路径约定在设计中固定，例如 `/api/v1/tg-menu`）：至少支持获取当前完整配置，以及用完整配置或等价更新语义写回。接口 MUST 无鉴权策略与现有 admin-api 一期一致（仅内网使用约定不变）。

#### Scenario: 获取 Menu
- **WHEN** 客户端 GET Menu 配置且服务可用
- **THEN** 返回 200 与当前菜单项列表 JSON

#### Scenario: 更新 Menu
- **WHEN** 客户端提交合法 Menu 配置进行更新
- **THEN** 返回成功，且随后 GET 可见更新结果

#### Scenario: 非法配置被拒绝
- **WHEN** 客户端提交缺必填字段、非法动作或无效 `case_id` 的配置
- **THEN** 返回 4xx 与错误说明，且不破坏既有已保存配置

#### Scenario: 回复媒体非法 URL 被拒绝
- **WHEN** 客户端提交回复媒体项且图片列表含非 http(s) 绝对 URL
- **THEN** 返回 4xx，且不破坏既有已保存配置

### Requirement: 仅经 admin-api 管理
Menu 的管理写路径 MUST 仅通过 admin-api 提供；Bot 进程 MUST NOT 再依赖改源码作为常规配置手段（默认种子除外）。

#### Scenario: 控制台经 admin-api 读写
- **WHEN** 管理前端保存 Menu
- **THEN** 请求指向 admin-api 的 Menu 接口，而非 bot `:8080` 管理残留路径

```
