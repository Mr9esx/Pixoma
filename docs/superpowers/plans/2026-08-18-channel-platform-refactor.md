---
change: channel-platform-refactor
design-doc: docs/superpowers/specs/2026-08-18-channel-platform-refactor-design.md
base-ref: 9b95701d047b591711b76658722c8937013982bb
archived-with: 2026-08-19-channel-platform-refactor
---

# 渠道层完整重构 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 以「渠道」为一级实体重构渠道层：渠道管理（平台+凭证+启停）、菜单收进渠道并平台中立化、渠道运行时端口化 + 热生效装配，为飞书/企微/钉钉铺路。

**Architecture:** 中立领域（`internal/menu` 菜单意图、`internal/channel` 渠道与端口契约）+ 平台适配器（`internal/channel/tg` 实现端口）+ 热生效装配器（`internal/channel/runtime` 监听渠道表动态启停）。应用内核 `botapp.Facade` 保持渠道无关。差异数据走 `channel_menu_item_extras`，不抹平 TG 特色能力。

**Tech Stack:** Go 1.24 / GORM / SQLite；React 18 + TanStack Router + shadcn-admin（pnpm）；go-telegram/bot（long polling）。

## Global Constraints

- 模块名：`github.com/mr9esx/comfyui_tgbot`
- 无存量数据：不做任何旧数据迁移；`tg_menus*`、`users.tg_user_id`、`sessions.chat_id`、`platform_settings.telegram_token_cipher`、`/api/v1/tg-menu` 全部移除
- `internal/menu` 与 `internal/channel` MUST NOT import `internal/channel/tg`
- 应用层/共享内核 MUST NOT 依赖任何平台 SDK 类型
- 凭证 AES-GCM 加密存储（复用 bootstrap enc key），API 只回显 masked token（`前4 + "****" + 后4`）
- 删除渠道受限：仅 enabled=false 且无活跃 session/task 引用可删（否则 409）
- 菜单中立核心字段无 row/col、无 TG 长度约束、无 `list_cases_by_tag`；TG 网格布局存 extras
- 会话地址：`sharedkernel.ChatID` 为 `tg:<chat_id>` 字符串；`sessions` 表存 `channel_id` + `chat_external_id`
- 测试命令：`go test ./...`；管理台：`cd web/admin && pnpm test && pnpm build`
- 每个任务结束时提交，commit message 用 conventional 风格

### 删除清单（本 change 必须物理删除，不保留兼容层/冻结表/死代码）

- 表模型与迁移：`tg_menus` / `tg_menu_items` / `tg_menu_item_cases` / `tg_menu_configs`（LegacyMenuRow）、`MigrateFromLegacyIfNeeded`
- 字段：`users.tg_user_id`、`sessions.chat_id`、`platform_settings.telegram_token_cipher`
- 领域：`internal/tgmenu` 整包（重命名为 `internal/menu` 并中立化）、`list_cases_by_tag` kind、`tag` 字段、`row/col`
- 代码：`internal/httpapi/tgmenu` 整包、`/api/v1/tg-menu` 路由、`internal/channel/tg/notifybridge` 旧 publisher（被 Task 11 泛化替换）
- 前端：`web/admin` 中 `tg-menu` 路由/页面/API 客户端、`menu.tgMenu` 与「主键盘」文案（Task 13 替换为渠道体系）
- 配置读取：env `telegram_bot_token` 与 settings 的 token 读取路径（Token 仅存 `channels.credential_ciphertext`）

删除后 `rg -n "tg_menu|tg-menu|tgMenu|TgUserID|telegram_token_cipher|list_cases_by_tag" internal apps web/admin/src --hidden` 除本计划与文档外应无命中。

---

## 文件结构（目标形态）

```
internal/channel/
  domain/channel.go            // Channel 实体、Platform、Repository
  domain/credential.go         // 凭证加解密、Masked()
  application/service.go       // Create/List/Get/Update/Disable/Delete(受限)
  application/service_test.go
  ports/ports.go               // InboundEvent/Outbound/MediaBridge/IdentityResolver/Action
  runtime/assembler.go         // 热生效装配器（watch + start/stop/restart）
  runtime/assembler_test.go
  runtime/adapter.go           // AdapterFactory / Adapter 生命周期接口
  infrastructure/persistence/gorm_channel.go
  infrastructure/persistence/gorm_channel_test.go
internal/menu/                 // 原 internal/tgmenu 重命名 + 中立化
  domain/document.go           // MenuTree/MenuNode/Kind（folder/open_case/placeholder/reply_media）
  domain/validate.go           // 去 row/col、去 tag；extras 白名单钩子
  domain/seed.go               // 按渠道种子
  domain/repository.go         // GetTree(ctx, channelID)/ReplaceTree/ListPlacementsByCase/ListExtras/SaveExtras
  infrastructure/persistence/gorm_repository.go   // channel_menu_* 表
internal/channel/tg/           // TG 适配器（端口实现）
  adapter.go                   // 回调翻译为 Action；chat_id ↔ ChatID
  messenger.go                 // Outbound 实现（ReplyKeyboard/Inline/媒体）
  media.go                     // MediaBridge（file_id → blob）
  extras.go                    // 读取 tg_root_layout 等 extras
internal/httpapi/channels/handler.go
internal/httpapi/channelmenu/handler.go
internal/identity/infrastructure/persistence/gorm_user.go   // 去 tg_user_id + user_external_identities
internal/sharedkernel/ids.go   // ChatID string + ChannelAddr + ParseChatID/FormatChatID
web/admin/src/config/menu.ts   // 侧栏「渠道」替代「主键盘」
web/admin/src/features/channels/*        // 渠道列表/新建/详情 + extras 编辑
web/admin/src/routes/_app/channels/*
```

---

### Task 1: sharedkernel 渠道地址类型

**Files:**
- Modify: `internal/sharedkernel/ids.go`
- Test: `internal/sharedkernel/ids_test.go`（新建）

**Interfaces:**
- Produces: `type ChannelAddr struct { ChannelID string; ExternalChatID string }`
- Produces: `func FormatChatID(addr ChannelAddr) string`（`tg:123`）；`func ParseChatID(s string) (ChannelAddr, error)`
- Produces: `type ChatID string`（替换 `int64`；本期仅改类型定义与编译错误，逐调用点随各任务迁移）

- [x] **Step 1: 写失败测试**

```go
package sharedkernel

import "testing"

func TestChatIDRoundTrip(t *testing.T) {
	addr := ChannelAddr{ChannelID: "tg-default", ExternalChatID: "123456789"}
	got, err := ParseChatID(FormatChatID(addr))
	if err != nil { t.Fatal(err) }
	if got != addr { t.Fatalf("roundtrip: %+v != %+v", got, addr) }
}

func TestParseChatIDInvalid(t *testing.T) {
	if _, err := ParseChatID("noprefix"); err == nil { t.Fatal("want error") }
	if _, err := ParseChatID("tg:"); err == nil { t.Fatal("want error for empty external id") }
}
```

- [x] **Step 2: 运行确认失败** — `go test ./internal/sharedkernel/ -run TestChatID -v`，预期编译失败/函数不存在
- [x] **Step 3: 实现**
  - `ids.go`：`ChatID int64` → `type ChatID string`；新增 `ChannelAddr`、`FormatChatID`（`channelID + ":" + externalChatID`）、`ParseChatID`（`strings.Cut`，两部分均非空，否则 error）
- [x] **Step 4: 运行通过** — 上述测试 PASS；`go build ./...` 记录编译错误清单（后续任务逐个修复）
- [x] **Step 5: 提交** — `git commit -m "refactor(sharedkernel): add channel chat address types"`

---

### Task 2: menu 领域重命名与中立化（tgmenu → menu）

**Files:**
- Rename: `internal/tgmenu` → `internal/menu`（domain/application/infrastructure 全部文件）
- Modify: `internal/menu/domain/document.go`、`tree.go`、`validate.go`、`seed.go`、`repository.go`
- Test: `internal/menu/domain/*_test.go`（沿用现有测试并更新）

**Interfaces:**
- Produces: `type MenuKind string` 仅含 `folder/open_case/placeholder/reply_media`
- Produces: `MenuNode{ID, ParentID, Label, Order int, Enabled, Kind, PlaceholderText, IntroText, CaseIDs, Reply, Children}`（无 Row/Col/Tag）
- Produces: `type MenuTree struct { ChannelID string; Items []MenuNode; UpdatedAt time.Time }`
- Produces: `Repository interface { GetTree(ctx, channelID string) (MenuTree, error); ReplaceTree(ctx, tree MenuTree) error; ListPlacementsByCase(ctx, caseID string) ([]MenuPlacement, error) }`
- Consumes: Task 1 的 `ChatID` 无关；本任务只动菜单领域

- [x] **Step 1: git mv 目录** — `git mv internal/tgmenu internal/menu`，更新全仓 import（`internal/tgmenu` → `internal/menu`）
- [x] **Step 2: 写失败测试（领域中立化）** — 在 `internal/menu/domain/validate_test.go` 追加：含 `Row`/`Col`/`Tag` 字段的节点编译失败；`KindListCasesByTag` 常量不存在；`Validate` 接受 `Order` 排序且同层 `Order` 重复报错
- [x] **Step 3: 运行确认失败** — `go test ./internal/menu/...`，预期编译失败
- [x] **Step 4: 实现中立模型**
  - `document.go`：删除 `Row/Col/Tag` 与 `KindListCasesByTag`；`MenuItem`/`MenuNode` 增加 `Order int`；`BotID` → `ChannelID`
  - `tree.go`：`nodeToItem`/`itemToNode` 同步；排序逻辑改为按 `Order`（平铺时先按 Order）
  - `validate.go`：去 row/col/tag 校验；保留同层 label 唯一、深度 ≤5、kind 关联约束（folder 子项仅 folder；open_case 恰 1 个 case 且无 children；placeholder 无 case；reply_media 需 text 或 http(s) 图片）；新增同层 Order 唯一校验
  - `seed.go`：`DefaultSeedTree()` → `DefaultSeedTree(channelID string)`，根项带 `Order`
  - `repository.go`：`GetTree(ctx, channelID string)`；删除 `DocumentIDDefault`/`BotIDDefault` 语义（改用调用方传入 channelID）
- [x] **Step 5: 运行通过** — `go test ./internal/menu/...` 全绿（原有测试更新为 Order 语义）
- [x] **Step 6: 提交** — `git commit -m "refactor(menu): rename tgmenu to menu and neutralize model"`

---

### Task 3: channels 领域与持久化（含凭证）

**Files:**
- Create: `internal/channel/domain/channel.go`、`internal/channel/domain/credential.go`
- Create: `internal/channel/application/service.go`、`service_test.go`
- Create: `internal/channel/infrastructure/persistence/gorm_channel.go`、`gorm_channel_test.go`
- Create: `internal/platform/crypto/crypto.go`（从 settings 提取 AES-GCM helper）
- Modify: `internal/platform/settings/store.go`（改用 crypto helper）

**Interfaces:**
- Produces: `type Platform string`（`telegram` 本期；预留 `feishu/wecom/dingtalk`）
- Produces: `type Channel struct { ID, Platform, Name, CredentialCiphertext string; Enabled bool; CreatedAt, UpdatedAt time.Time }`
- Produces: `func EncryptCredential(key []byte, cred Credential) (string, error)`；`func DecryptCredential(key []byte, ct string) (Credential, error)`；`func MaskedToken(token string) string`
- Produces: `type Repository interface { Create(ctx, Channel) error; Get(ctx, id) (Channel, error); List(ctx) ([]Channel, error); Update(ctx, Channel) error; Delete(ctx, id) error }`
- Produces: `application.Service{ Create/Get/List/Update/Disable/Delete }`；`Delete` 校验 enabled=false 且无活跃 session/task（通过注入 `HasActiveRefs(ctx, channelID) (bool, error)` 端口）

- [x] **Step 1: 写失败测试（凭证）** — `credential_test.go`：Encrypt→Decrypt 往返一致；错误 key 解密失败；`MaskedToken("1234567890") == "1234****7890"`，短 token 用 `"****"` 全遮
- [x] **Step 2: 写失败测试（持久化）** — `gorm_channel_test.go`（sqlite memory + AutoMigrate）：Create→Get 往返；List 含 enabled 过滤；Update 凭证后 Get 为密文新值；Delete 删除
- [x] **Step 3: 运行确认失败** — `go test ./internal/channel/...`，预期编译失败
- [x] **Step 4: 实现**
  - `internal/platform/crypto`：`Encrypt(key []byte, plaintext string) (string, error)` / `Decrypt(key, ct) (string, error)`（AES-GCM，nonce 前置 base64）；`settings` 复用并删除私有 encrypt/decrypt
  - `domain/channel.go` + `gorm_channel.go`：`channels` 表（id/platform/name/credential_ciphertext/enabled/created_at/updated_at）；`platform` 校验白名单
  - `application/service.go`：Create 时凭证加密；Get/List 回显 `MaskedToken`（DTO 层）；Update 支持 token 留空=不变；Delete 先查 `HasActiveRefs`
- [x] **Step 5: 运行通过** — `go test ./internal/channel/... ./internal/platform/settings/...` 全绿
- [x] **Step 6: 提交** — `git commit -m "feat(channel): add channel domain, credential crypto, and persistence"`

---

### Task 4: 菜单持久化渠道化（channel_menu_* 表 + extras）

**Files:**
- Modify: `internal/menu/infrastructure/persistence/gorm_repository.go`
- Test: `internal/menu/infrastructure/persistence/gorm_repository_test.go`

**Interfaces:**
- Consumes: Task 2 的 `MenuTree{ChannelID}`；Task 3 的 `channels` 表
- Produces: `type Extra struct { ChannelID, MenuItemID, ExtraType, ExtraJSON string; UpdatedAt time.Time }`
- Produces: `Repository` 追加 `ListExtras(ctx, channelID string) (map[string][]Extra, error)`、`SaveExtras(ctx, channelID string, extras map[string][]Extra) error`（按 menu_item 全量替换）

- [x] **Step 1: 写失败测试** — 更新 `gorm_repository_test.go`：AutoMigrate 换成 `channel_menus/channel_menu_items/channel_menu_item_cases/channel_menu_item_extras`；`GetTree(ctx, "tg-default")` 读取树含 Order；`SaveExtras` 后 `ListExtras` 按 item 返回且跨渠道隔离；UNIQUE(channel_id, parent_id, label/order) 冲突报错
- [x] **Step 2: 运行确认失败** — `go test ./internal/menu/...`，预期旧表模型编译失败/测试失败
- [x] **Step 3: 实现**
  - 表模型：`ChannelMenuRow{ChannelID PK}`、`ChannelMenuItemRow{ID PK, ChannelID, ParentID *string, Label, Order, Enabled, Kind, PlaceholderText, IntroText, ReplyJSON}`、`ChannelMenuItemCaseRow{MenuItemID, CaseID PK, Sort}`、`ChannelMenuExtraRow{ChannelID, MenuItemID, ExtraType PK, ExtraJSON, UpdatedAt}`
  - 删除 `MenuHeaderRow/BotID/MenuItemRow.Row/Col/Tag` 与 `LegacyMenuRow`；删除 `MigrateFromLegacyIfNeeded`
  - `GetTree` 按 channelID 查询并 `BuildTree`；`ReplaceTree` 事务内按 channelID 整树替换（items+links+extras 保留策略：extras 由 SaveExtras 单独管理，ReplaceTree 不动 extras）
  - `ListPlacementsByCase` 返回含 `ChannelID` 的路径
- [x] **Step 4: 运行通过** — `go test ./internal/menu/...` 全绿
- [x] **Step 5: 提交** — `git commit -m "feat(menu): channel-scoped menu persistence with extras table"`

---

### Task 5: 渠道与菜单管理 API（channels CRUD + menu + extras + placements）

**Files:**
- Create: `internal/httpapi/channels/handler.go`、`handler_test.go`
- Create: `internal/httpapi/channelmenu/handler.go`、`handler_test.go`
- Modify: `internal/httpapi/adminhost/server.go`
- Delete: `internal/httpapi/tgmenu/`（迁移 placements 到 channelmenu）

**Interfaces:**
- Consumes: Task 3 Service、Task 4 Repository
- Produces: `GET/POST /api/v1/channels`、`GET/PUT/DELETE /api/v1/channels/{id}`、`POST /api/v1/channels/{id}/disable|enable`、`GET/PUT /api/v1/channels/{id}/menu`、`GET/PUT /api/v1/channels/{id}/menu/extras`、`GET /api/v1/cases/{id}/menu-placements`

- [x] **Step 1: 写失败测试** — `handler_test.go`（sqlite + AutoMigrate channels/menu 表）：POST 创建返回 masked token；非法 platform 400；GET 列表；PUT 更新 token 留空不变；DELETE 启用中 409；disable 后无引用可删；GET/PUT menu 校验非法树 400；GET/PUT extras 按渠道隔离；placements 含渠道
- [x] **Step 2: 运行确认失败** — `go test ./internal/httpapi/channels/... ./internal/httpapi/channelmenu/...`
- [x] **Step 3: 实现**
  - channels handler：DTO `{id, platform, name, token_masked, enabled, created_at, updated_at}`；写入时校验 `platform` 与凭证；删除检查 `HasActiveRefs`（注入查询 `sessions/tasks` 是否存在该 channel 引用）
  - channelmenu handler：`GET/PUT /api/v1/channels/{id}/menu`（校验存在渠道，404）；`/menu/extras` 读写（extra_type 白名单由适配器注册，未知类型忽略）；placements 从旧 tgmenu handler 迁移
  - adminhost：挂载 `/api/v1/channels`；移除 `/api/v1/tg-menu` 路由；`Options.TGMenu` 删除
- [x] **Step 4: 运行通过** — 上述测试全绿；`go build ./...`
- [x] **Step 5: 提交** — `git commit -m "feat(httpapi): channel and channel menu management APIs"`

---

### Task 6: 身份渠道化（user_external_identities）

**Files:**
- Modify: `internal/identity/domain/user.go`、`repository.go`
- Modify: `internal/identity/infrastructure/persistence/gorm_user.go`
- Test: `internal/identity/infrastructure/persistence/gorm_user_test.go`

**Interfaces:**
- Produces: `type UpsertFrom struct { ChannelID, ExternalUserID string; Username, FirstName, LastName, LanguageCode string; ProfileJSON string; LastSeenAt time.Time }`
- Produces: `Repository.UpsertByChannelExternal(ctx, in UpsertFrom) (*User, error)`；`GetByID`、`List(ctx, ListQuery{ChannelID, ExternalUserID, Q, ...})`
- Removes: `UpsertByTgUserID`、`TgUserID` 字段

- [x] **Step 1: 写失败测试** — `UserRow` 无 `TgUserID` 编译失败；AutoMigrate `users + user_external_identities`；同 (channel, external_id) 两次 upsert 返回同一内部 id；不同渠道外部 id 并存；`List` 按 ChannelID 过滤
- [x] **Step 2: 运行确认失败** — `go test ./internal/identity/...`
- [x] **Step 3: 实现**
  - `user_external_identities(id PK, user_id, channel_id, external_user_id, profile_json, last_seen_at)`，UNIQUE(channel_id, external_user_id)
  - upsert 流程：按 (channel_id, external_user_id) 查外部身份 → 无则建内部 user（uuid）+ 外部身份；有则刷新 users 通用资料快照与 last_seen_at
  - `ListQuery` 增加 `ChannelID *string`、`ExternalUserID *string`，参数化查询
- [x] **Step 4: 运行通过** — `go test ./internal/identity/...`；修复 `internal/httpapi/users` 与 adapter 调用点（`UpsertByTgUserID` 引用改为新方法；httpapi/users 接线延至 Task 12）
- [x] **Step 5: 提交** — `git commit -m "refactor(identity): channel-scoped external identities"`

---

### Task 7: 会话寻址渠道化（sessions 拆列 + 事件载荷）

**Files:**
- Modify: `internal/conversation/infrastructure/persistence/gorm_session.go`
- Modify: `internal/conversation/domain/*`（`ChatID` 使用点）
- Modify: `internal/runtime/infrastructure/persistence/gorm_task.go`（如需）、`internal/sharedkernel/events.go`
- Test: `internal/conversation/infrastructure/persistence/gorm_session_test.go`

**Interfaces:**
- Consumes: Task 1 `ChatID string`
- Produces: `SessionRow{ChannelID string; ChatExternalID string}`（替代 `ChatID int64`；索引 (channel_id, chat_external_id, status)）

- [x] **Step 1: 写失败测试** — `SessionRow` 无 `ChatID int64`；AutoMigrate 后按 (channel_id, chat_external_id) 查询；conversation 领域 `Session.ChatID` 为 string
- [x] **Step 2: 运行确认失败** — `go test ./internal/conversation/... ./internal/runtime/...`
- [x] **Step 3: 实现**
  - `gorm_session.go`：`ChatID int64` → `ChannelID` + `ChatExternalID`（均 TEXT，index）
  - `sharedkernel/events.go`：`TaskCreated.ChatID`、`UserNotify.ChatID` 类型已是 `ChatID`（string）；JSON 序列化无需改 key
  - 修复全仓编译错误：`internal/channel/tg` 中 `int64(chatID)` 使用点暂时以 `FormatChatID` 组装/`ParseChatID` 拆分（最终形态在 Task 9 完成）
- [x] **Step 4: 运行通过** — `go test ./internal/conversation/... ./internal/runtime/... ./internal/packaging/...`
- [x] **Step 5: 提交** — `git commit -m "refactor(conversation): channel-scoped session addressing"`

---

### Task 8: 渠道端口契约（ports）

**Files:**
- Create: `internal/channel/ports/ports.go`
- Test: `internal/channel/ports/ports_test.go`（Action 序列化/校验）

**Interfaces:**
- Produces:

```go
type ActionType string
const (
    ActionOpenMenu ActionType = "open_menu"
    ActionOpenFolder  ActionType = "open_folder"
    ActionOpenCase    ActionType = "open_case"
    ActionStartCase   ActionType = "start_case"
    ActionSubmitText  ActionType = "submit_text"
    ActionSubmitMedia ActionType = "submit_media"
    ActionConfirm     ActionType = "confirm"
    ActionSkip        ActionType = "skip"
    ActionExit        ActionType = "exit"
    ActionContinue    ActionType = "continue"
    ActionReplaceStart ActionType = "replace_start"
)

type Action struct {
    Type     ActionType `json:"type"`
    MenuItemID string   `json:"menu_item_id,omitempty"`
    CaseID   string     `json:"case_id,omitempty"`
    BackRef  string     `json:"back_ref,omitempty"`
    Text     string     `json:"text,omitempty"`
    Media    *InboundMedia `json:"media,omitempty"`
}

type InboundMedia struct { ExternalFileID string `json:"external_file_id"`; MIME string `json:"mime,omitempty"` }

type InboundEvent struct {
    Addr   sharedkernel.ChannelAddr `json:"addr"`
    Kind   EventKind                `json:"kind"` // text | media | callback
    Text   string                   `json:"text,omitempty"`
    Media  *InboundMedia            `json:"media,omitempty"`
    Action Action                   `json:"action,omitempty"`
}

type MenuEntry struct { ID, Label string }
type Button struct { Text string; Action Action }

type Outbound interface {
    SendText(ctx context.Context, addr sharedkernel.ChannelAddr, text string) error
    SendMenu(ctx context.Context, addr sharedkernel.ChannelAddr, title string, items []MenuEntry) error
    SendList(ctx context.Context, addr sharedkernel.ChannelAddr, title string, rows [][]Button) error
    SendMedia(ctx context.Context, addr sharedkernel.ChannelAddr, ref sharedkernel.BlobRef, caption string) error
}

type MediaBridge interface {
    Download(ctx context.Context, externalFileID, mime string) ([]byte, error)
    Upload(ctx context.Context, data []byte, mime string) (string, error)
}

type IdentityResolver interface {
    Resolve(ctx context.Context, addr sharedkernel.ChannelAddr, profile identitydomain.UpsertFrom) (string, error)
}
```

- [x] **Step 1: 写失败测试** — Action JSON 往返；未知 ActionType 校验失败；`InboundEvent` 携带 Addr 完整
- [x] **Step 2: 运行确认失败** — `go test ./internal/channel/ports/...`
- [x] **Step 3: 实现** — 按上述接口写 `ports.go`（依赖 `sharedkernel` 与 `identitydomain`）；`Action.Validate()` 按 Type 校验必填（open_folder 需 MenuItemID；open_case/start_case 需 CaseID）
- [x] **Step 4: 运行通过** — 测试全绿
- [x] **Step 5: 提交** — `git commit -m "feat(channel): define channel runtime ports"`

---

### Task 9: TG 适配器重构为端口实现

**Files:**
- Modify: `internal/channel/tg/adapter.go`、`bot.go`、`messenger.go`、`menu_runtime.go`
- Create: `internal/channel/tg/extras.go`、`media.go`
- Test: `internal/channel/tg/adapter_test.go`、`bot_test.go`

**Interfaces:**
- Consumes: Task 8 ports；Task 1 ChatID；Task 2 menu；Task 6 identity
- Produces: `type Adapter struct { App *botapp.Facade; Out ports.Outbound; Media ports.MediaBridge; Users ports.IdentityResolver; Menu MenuReader; Extras ExtrasReader }`
- Produces: `func TranslateCallback(data string) (ports.Action, error)`（`mf:`/`mb:`/`cp:`/`cs:`/`cf`/`sk`/`ex`/`ct`/`rs:`/`il` → Action）

- [x] **Step 1: 写失败测试（回调翻译）** — `TranslateCallback("mf:btn-image") == Action{OpenFolder, MenuItemID:"btn-image"}`；`mb:root` → OpenMenu；`cpf:folder:case` → OpenCase{BackRef:"mb:folder"}；未知前缀 → error；`cs:case1` → StartCase
- [x] **Step 2: 写失败测试（chat 映射与渲染）** — 适配器 `HandleText(ctx, "tg:123", text, userID)` 入口解析 Addr；`SendMenu` 对 root items 按 extras `tg_root_layout` 排布（缺省两列）；`SendList` 生成 Inline 且回调编码内部化；`SendMedia` 走 blob→photo
- [x] **Step 3: 运行确认失败** — `go test ./internal/channel/tg/...`
- [x] **Step 4: 实现**
  - `adapter.go`：`chatID int64` 参数改为 `addr sharedkernel.ChannelAddr`（或 `chatID string`）；入口 `ParseChatID`；`HandleCallback` 改调 `TranslateCallback` 后统一走 Action 分发（复用现有 Facade 调用：OpenFolder→showMenuFolder、OpenCase→showCasePreview、StartCase→startCase、Confirm/Skip/Exit/Continue/ReplaceStart 同现有）
  - `extras.go`：`ExtrasReader` 从 `internal/menu` 读 extras；`RootLayout(extras)` 返回 `[]int`（每行按钮数）或 nil（缺省两列）；`BuildReplyKeyboard(tree, extras)`
  - `messenger.go`：实现 `ports.Outbound`（`SendMenu`→ReplyKeyboard+文本；`SendList`→InlineKeyboard 用 `TranslateCallback` 反向编码；`SendMedia`→SendPhoto；`SendText`→SendMessage）；`AnswerCallback` 逻辑保留在 bot.go 层
  - `media.go`：`Download` 用现有 file_id 下载逻辑；`Upload` 返回 `not implemented`（本期仅预留）
  - `bot.go`：`RegisterHandlers` 中 chat id 用 `FormatChatID` 组装传给 Adapter；`BotMessenger` 适配 Outbound 签名
- [x] **Step 5: 运行通过** — `go test ./internal/channel/tg/...` 全绿
- [x] **Step 6: 提交** — `git commit -m "refactor(tg): implement channel ports and internalize callback protocol"`

---

### Task 10: 热生效装配器（runtime）

**Files:**
- Create: `internal/channel/runtime/adapter.go`、`assembler.go`、`assembler_test.go`

**Interfaces:**
- Produces:

```go
type Adapter interface {
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
}
type AdapterFactory interface {
    Create(platform string, credential string) (Adapter, error)
}
type Assembler struct {
    Store   ChannelSnapshotStore // ListChannels(ctx) ([]ChannelSnapshot, error)
    Factory AdapterFactory
    Interval time.Duration // 默认 5s
}
func (a *Assembler) Run(ctx context.Context) error // 每 Interval 拉快照 diff
```

- [x] **Step 1: 写失败测试（fake）** — fake factory 记录 Create/Start/Stop 调用：空库→无启动；新增 enabled 渠道→start；禁用→stop；credential 变化→stop+start（重建）；删除→stop；Start 失败→退避重试（Interval 注入可缩短）；ctx cancel→全部停止
- [x] **Step 2: 运行确认失败** — `go test ./internal/channel/runtime/...`
- [x] **Step 3: 实现**
  - `adapter.go`：`Adapter`/`AdapterFactory` 接口；`credential_hash` 由凭证密文 sha256 计算
  - `assembler.go`：`Run(ctx)` 循环；内存 `map[channelID]runtimeState`（absent/starting/running/stopping/error）；每轮 diff：新增/启用→`factory.Create`+`Start`；停用/删除→`Stop`；凭证 hash 变化→`Stop` 后重建；error 状态记录 lastErr 并按 5s/30s/5min 退避重试
  - 事件路由：`Adapter` 扩展 `Events() <-chan ports.InboundEvent`（或 `Receive(ctx, handler)`），统一 handler 内 `recover()` 防 panic 崩溃
- [x] **Step 4: 运行通过** — 测试全绿
- [x] **Step 5: 提交** — `git commit -m "feat(channel): hot-reload runtime assembler"`

---

### Task 11: notify 按渠道投递

**Files:**
- Modify: `internal/channel/tg/notifybridge/publisher.go`
- Create: `internal/channel/runtime/notify.go`、`notify_test.go`

**Interfaces:**
- Produces: `type NotifyRouter struct { OutByChannel func(channelID string) (ports.Outbound, bool) }`；`func (r *NotifyRouter) Publish(ctx, n sharedkernel.UserNotify) error`（`ParseChatID(n.ChatID)` → 找 Outbound → `SendMedia/SendText`）

- [x] **Step 1: 写失败测试** — `UserNotify{ChatID:"tg:123", Kind:"task_succeeded", Outputs:[blob]}` → 对应 channel 的 fake Outbound 收到 SendMedia；未知渠道→记录并返回 nil（不崩溃）；重复终态通知去重（沿用 adapter 内 `notified` map，按 channel 隔离）
- [x] **Step 2: 运行确认失败** — `go test ./internal/channel/runtime/... -run Notify`
- [x] **Step 3: 实现** — `notify.go` 按地址路由；`notifybridge` 包废弃删除，装配器在 Start 时把 router 注入 orchestrator 的 notify.Publisher
- [x] **Step 4: 运行通过** — 测试全绿；`go build ./...`
- [x] **Step 5: 提交** — `git commit -m "refactor(channel): route notifications by channel"`

---

### Task 12: 进程装配接线（main / telegram.go / adminhost）

**Files:**
- Modify: `apps/pixoma/cmd/pixoma/main.go`、`apps/pixoma/internal/app/telegram.go`
- Modify: `apps/admin-api/cmd/admin-api/main.go`、`internal/httpapi/adminhost/server.go`

**Interfaces:**
- Consumes: Task 3/5/10/11
- Produces: `StartBotRuntime` 改为：读 `channels`（enabled）→ `assembler.Run(ctx)`；`adminhost.Options` 增加 `Channels *channels.Handler`、`ChannelMenu *channelmenu.Handler`，移除 `TGMenu`

- [x] **Step 1: 改 main.go 接线** — `appboot.Bootstrap` Models 换成 `channels/channel_menus/channel_menu_items/channel_menu_item_cases/channel_menu_item_extras/user_external_identities` 等新表模型；移除 `tgmenupersist.*` 与 `platform_settings` token 读取；`settings.ApplyEnv` 不再覆盖 token
- [x] **Step 2: 改 telegram.go** — `StartBotRuntime(ctx, deps)`：构建 facade → `assembler.Run(ctx)`（goroutine）；`TG adapter` 工厂注册进 `AdapterFactory`（从 channel 凭证解密拿 token → `bot.New` + `RegisterHandlers` + `Start`）
- [x] **Step 3: 改 adminhost + admin-api main** — 挂载 channels/channelmenu；删除 tg-menu 引用；跑 `go build ./...` 与 `go test ./...`，修复全部编译错误（含 `sharedkernel.ChatID` 使用点）
- [x] **Step 4: 冒烟** — 本地 `go run ./apps/pixoma`（mock 模式）启动成功；`curl /api/v1/channels` 返回空列表或种子
- [x] **Step 5: 提交** — `git commit -m "feat(app): wire channel runtime and admin APIs"`

---

### Task 13: 管理台改版（渠道 + 菜单 + extras）

**Files:**
- Modify: `web/admin/src/config/menu.ts`、`menu.test.ts`
- Create: `web/admin/src/routes/_app/channels/index.tsx`、`new.tsx`、`$id.tsx`、`$id/menu.tsx`、`$id/extras.tsx`
- Create: `web/admin/src/features/channels/channels-list.tsx`、`channel-wizard.tsx`、`channel-detail.tsx`、`channel-menu-editor.tsx`、`extras-editor.tsx`
- Modify: `web/admin/src/lib/api/`（新增 `channels.ts`；`tg-menu.ts` 迁移为 `channel-menu.ts`）、`query-keys.ts`
- Modify: `web/admin/src/lib/i18n/locales/zh.json`、`en.json`

**Interfaces:**
- Consumes: Task 5 API
- Produces: 侧栏顺序 Dashboard/实例/Case/渠道/Task/User/Session；渠道详情 tabs（基本信息/菜单/extras）

- [x] **Step 1: 写契约测试** — `features/channels/channels.contract.test.ts`：创建向导 POST 载荷 `{platform:"telegram", name, token}`；详情 PUT token 留空不发 token；菜单 PUT 载荷无 row/col/tag；extras PUT 载荷 `{extra_type, extra_json}`
- [x] **Step 2: 运行确认失败** — `cd web/admin && pnpm test`
- [x] **Step 3: 实现**
  - `menu.ts`：`tg-menu` 项替换为 `{ id:'channels', titleKey:'menu.channels', path:'/channels', icon: Radio }`；顺序调整；`menu.test.ts` 同步
  - `lib/api/channels.ts`：`listChannels/createChannel/getChannel/updateChannel/deleteChannel/setChannelEnabled`；`channel-menu.ts`：`getChannelMenu/putChannelMenu/getChannelMenuExtras/putChannelMenuExtras`；`query-keys.ts` 增加 `channels.all`、`channels.detail(id)`、`channelMenu.all(id)`
  - 路由：`/channels` 列表；`/channels/new` 向导（平台选择 TG → 名称+Token → 创建后跳详情）；`/channels/$id` 详情（tabs：基本信息：masked token/启停/删除受限提示；菜单：迁移原 `TgMenuEditor`（去 row/col，改 order 上下移动；文件夹/Case 挂载/占位/reply_media 保留）；extras：按平台 schema 渲染（TG 根层网格布局表单 + 键值 JSON 编辑器））
  - i18n：新增 `menu.channels`、`channels.*`（列表/新建/详情/菜单/extras 文案），移除 `menu.tgMenu` 与 `tgMenu.*` 中「主键盘」入口文案
- [x] **Step 4: 运行通过** — `cd web/admin && pnpm test && pnpm build`；`rg -n "tg-menu|tgMenu" web/admin/src` 仅剩测试夹具或已清理
- [x] **Step 5: 提交** — `git commit -m "feat(admin): channel management with menu and extras editors"`

---

### Task 14: 全量测试与回归

**Files:**
- Modify: `docs/openspec/changes/channel-platform-refactor/tasks.md`（逐项勾选）

- [x] **Step 1: 后端全量** — `go test ./...` 全绿；`go vet ./...` 无新告警
- [x] **Step 2: 前端全量** — `cd web/admin && pnpm test && pnpm build` 全绿
- [x] **Step 3: 手工回归（TG 行为等价）** — 创建 TG 渠道填 token → 5s 内 bot 启动；六键主键盘（根层两列/按 extras 网格）；文件夹下钻 + 返回；Case 预览/开始/填表/确认/出图通知；停用渠道 → bot 停止；改 token → 重建（旧连接停止）；删除启用渠道被拒（注：真实 Token 手工回归待上线前执行；装配/适配器由单测覆盖）
- [x] **Step 4: 无残留检查** — `rg -n "tg_menu|tg-menu|tgMenu|TgUserID|telegram_token_cipher|list_cases_by_tag|row|col" internal apps web/admin/src --hidden` 仅剩允许的迁移注释/文档
- [x] **Step 5: 提交收尾** — 勾选 tasks.md 全部任务；`git commit -m "test(channel): full regression for channel platform refactor"`

---

## Self-Review（writing-plans 要求）

**Spec 覆盖检查：**
- `channel-management`（渠道实体/CRUD/热生效/删除受限/运行时装配）→ Task 3/5/10/12
- `channel-menu-config`（渠道作用域/中立模型/extras/Case 挂载/默认种子）→ Task 2/4/5/13
- `channel-runtime-ports`（端口契约/身份渠道化/通知投递/专有隔离）→ Task 8/9/11/6
- `channel-tg`（端口实现/主菜单来源/渠道装配）→ Task 9/12
- `tg-menu` / `tg-menu-admin-api`（中立模型/API 路径/placements）→ Task 2/4/5
- `user-directory`（渠道化外部身份）→ Task 6
- `admin-web-shell`（侧栏渠道替代主键盘）→ Task 13

**类型一致性：** `ports.Action` 在 Task 8 定义、Task 9 生产、Task 13 前端仅消费 JSON；`ChannelAddr`/`ChatID` 在 Task 1 定义、Task 7/9 消费；`channel_menu_item_extras` 在 Task 4 建表、Task 5 API、Task 9 消费、Task 13 编辑。
