---
comet_change: channel-platform-refactor
role: technical-design
canonical_spec: openspec
archived-with: 2026-08-19-channel-platform-refactor
status: final
---

# 消息平台层完整重构：深度技术设计（channel-platform-refactor）

> 上游事实源：OpenSpec change `channel-platform-refactor` 的 proposal.md / design.md / specs/**。本文是对 open 阶段 design.md 高层框架的深化，聚焦实现细节、边界条件与测试策略。

## 1. 包结构与分层

```
internal/channel/
  domain/            Channel 实体、Platform 枚举、Repository/Service 端口
  application/       Create/List/Get/Update/Disable/Delete(受限)、凭证加解密编排
  runtime/           装配器：扫描、watch、生命周期、事件路由
  ports/             端口契约：InboundEvent/Outbound/MediaBridge/IdentityResolver
internal/menu/       原 internal/tgmenu 重命名：MenuTree/MenuNode/kind/validate/seed/Repository
internal/channel/tg/ TG 适配器：BotMessenger、回调翻译、媒体桥、装配入口
internal/identity/   users + user_external_identities
internal/sharedkernel/ ChannelAddr 与 ChatID 类型、notify 事件
internal/httpapi/channels/ + internal/httpapi/channelmenu/  管理 API
web/admin/src/features/channels/ + 迁移后的菜单编辑器
```

边界约束：
- `internal/menu` 与 `internal/channel` MUST NOT import `internal/channel/tg`
- `botapp.Facade` 保持消息平台无关；适配器只做翻译与渲染
- `runtime` 依赖 ports 与适配器工厂（registry: platform → adapter factory），不依赖具体平台 SDK

## 2. 端口契约

### 2.1 InboundEvent（规范化入站事件）

```go
type ChannelAddr struct {
    ChannelID     string // 如 "tg-default"
    ExternalChatID string // 如 "123456789" / "oc_xxx"
}

type InboundEvent struct {
    Addr   ChannelAddr
    Kind   EventKind // Text | Media | Callback
    Text   string
    Media  *InboundMedia // {ExternalFileID, MIME, ...}
    Action Action        // 回调/按钮翻译后的规范化动作
    Raw    string        // 原始载荷（仅调试/审计）
}
```

### 2.2 规范化动作与 Facade 映射

| Action | 含义 | 应用方法 |
|---|---|---|
| OpenMenu | 回主菜单 | 渲染层 SendMenu |
| OpenFolder(itemID) | 进目录 | 菜单读取 + SendList |
| OpenCase(caseID, back) | Case 预览 | GetCase + 预览渲染 |
| StartCase(caseID) | 开始 | StartCase |
| SubmitText / SubmitMedia | 填表 | SubmitInput |
| Confirm / Skip / Exit / Continue | 会话控制 | ConfirmRun / SkipInput / ExitSession / GetSession |
| ReplaceStart(caseID) | 冲突后重开 | ExitSession + StartCase |

### 2.3 Outbound

```go
type Outbound interface {
    SendText(ctx, addr, text) error
    SendMenu(ctx, addr, title string, items []MenuEntry) error // 根层入口（TG: ReplyKeyboard）
    SendList(ctx, addr, title string, rows [][]Button) error   // 目录/Case 列表（TG: Inline）
    SendMedia(ctx, addr, ref BlobRef, caption string) error
}
type Button struct { Text string; Action Action }
```

- TG 实现：SendMenu → ReplyKeyboard（按 extras 网格布局），SendList → InlineKeyboard（64 字节编码内部化），SendMedia → SendPhoto
- 飞书实现（后续）：SendMenu → 机器人菜单/欢迎卡片，SendList → 交互卡片，SendMedia → 上传 image_key 后发送

### 2.4 MediaBridge / IdentityResolver

```go
type MediaBridge interface {
    Download(ctx, extFileID, mime) ([]byte, error) // 平台媒体 → 字节
    Upload(ctx, data, mime) (externalFileID, error) // 预留（飞书 image_key）
}
type IdentityResolver interface {
    Resolve(ctx, addr ChannelAddr, profile Profile) (UserID, error) // upsert
}
```

TG MediaBridge 用 file_id → blob；出图走 blob → photo。本期 Upload 仅预留。

## 3. 运行时装配器（热生效）

- 启动：扫描 `channels WHERE enabled=1` 且 platform 已知 → 逐消息平台 `factory.Create(platform, credential)` → `Start(ctx)`
- watch：每 5s（可配置）拉取消息平台快照（id/platform/credential_hash/enabled/updated_at），与内存期望状态 diff：
  - 新增/启用 → start
  - 停用 → stop
  - 凭证变化（credential_hash 不同）→ stop → start（重建）
  - 删除 → stop
- 状态机：absent → starting → running → stopping → absent；error 状态带退避重试（5s / 30s / 5min 封顶）
- 事件路由：适配器 `Receive(ctx)` 产出 InboundEvent → 统一 handler（panic recover + 日志）→ 翻译动作 → Facade
- 并发：单装配器串行 diff，消息平台 goroutine 独立；退出用 context cancel + WaitGroup
- 竞态：watch 周期内连续变更以最新快照为准（每轮全量 diff，不做增量队列）

测试注入：`AdapterFactory` 接口可注入 fake；watch 周期可缩短（注入 ticker）。

## 4. 数据层

（表结构见 design.md「数据库调整」；以下为索引/约束/实现要点）

- channels：`credential_ciphertext` 为 AES-GCM（nonce 前置）；masked 格式 `前4 + "****" + 后4`
- channel_menu_items：UNIQUE(channel_id, parent_id, label)、UNIQUE(channel_id, parent_id, order)
- channel_menu_item_extras：UNIQUE(channel_id, menu_item_id, extra_type)；extra_type 由平台适配器注册白名单（TG: `tg_root_layout` 等）；未知类型忽略
- user_external_identities：UNIQUE(channel_id, external_user_id)；users.id 为内部 UUID
- sessions：索引 (channel_id, chat_external_id, status)；单活跃会话由应用层保证
- 删除受限：DELETE 前校验 enabled=false 且无活跃 session/task 引用；冲突返回 409；或软删除 deleted_at

凭证加解密复用 `bootstrap.EncKey()`（enc_key_b64），与 settings 的 AES-GCM 同套实现（抽公共 crypto helper）。

## 5. 会话寻址与身份

- sharedkernel：`ChatID = string`（`tg:<chat_id>`）；`ChatAddr{ChannelID, ExternalChatID}` 为结构化类型；提供 `ParseChatID / FormatChatID` 转换
- sessions：channel_id + chat_external_id（由 ChatID 拆分落库）
- notify：UserNotify.ChatID 保持 string 地址，装配器按 ChannelID 前缀路由到适配器 Outbound
- 身份：适配器入口 `IdentityResolver.Resolve(addr, profile)`；TG 的 From → (tg-default, 字符串化 tg_user_id)，profile 存 profile_json；is_bot/is_premium 等 TG 特有字段进 profile_json

## 6. 管理 API 与 DTO

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | /api/v1/channels | 列表（masked token、enabled） |
| POST | /api/v1/channels | 创建（platform+name+token）→ 热生效 |
| GET/PUT | /api/v1/channels/{id} | 详情/更新（token 留空=不变） |
| POST | /api/v1/channels/{id}/disable · enable | 启停（或 PUT enabled 字段） |
| DELETE | /api/v1/channels/{id} | 受限删除（409 带原因） |
| GET/PUT | /api/v1/channels/{id}/menu | 中立菜单树 |
| GET/PUT | /api/v1/channels/{id}/menu/extras | extras 读写 |
| GET | /api/v1/cases/{id}/menu-placements | 含消息平台标识 |

错误码：404 消息平台不存在；400 校验失败（非法平台/空凭证/非法菜单树）；409 删除受限；422 凭证无效（创建时可用 getMe 预检）。

## 7. 管理台

- 路由：`/channels`（列表）、`/channels/new`（向导）、`/channels/:id`（详情 tabs：基本信息 / 菜单 / extras）
- 侧栏「消息平台」替代「主键盘」；顺序 Dashboard/实例/Case/消息平台/Task/User/Session
- 菜单编辑器：中立字段（名称、排序、类型、Case 挂载、启停）通用；extras 面板按平台 schema 注册渲染（TG 根层网格布局表单、其他键值 JSON 编辑器）
- 热生效提示：保存后 toast「已生效」或「将在数秒内生效」

## 8. 错误处理与边界条件

- Bot 启动失败（token 无效/网络）：适配器进入 error 状态，退避重试；消息平台详情展示状态与最后错误；创建时可用 getMe 预检提示
- 凭证为空：禁用不启动
- extras 无效：适配器忽略；管理台保存时按平台 schema 校验并提示
- 删除受限：409 + 具体原因（未禁用/有会话/有任务）
- 热生效竞态：以最新快照为准，不重放旧状态
- 通知投递失败：幂等去重保留，重试策略沿用现有

## 9. 测试策略

- 单元：menu 领域（order/extras 校验）、Channel 凭证加解密、ChatID 转换
- 装配器：fake factory/ticker 覆盖 start/stop/restart/删除/失败退避
- 适配器：TG 回调前缀 → Action 翻译、chat 映射、extras 消费（网格布局）、媒体
- 持久化：extras 唯一/隔离、删除限制、身份唯一
- API：channels/menu/extras/placements/409/422
- 手工回归：六键主键盘、文件夹下钻、Case 预览/开跑/出图；热生效（改 token 数秒内不重启可用；停用后停止）

## 10. 实施顺序（与 tasks 对齐）

1. sharedkernel ChatAddr + menu 重命名/中立化（tasks 2/3）
2. channels 领域 + 持久化 + 凭证（tasks 1）
3. 端口 + TG 适配器重构（tasks 5）
4. 装配器热生效（tasks 6）
5. 身份消息平台化（tasks 4）
6. API + 管理台（tasks 7/8）
7. 测试与回归（tasks 9）

## 11. 风险与缓解

| 风险 | 缓解 |
|---|---|
| 热生效装配器生命周期 bug 导致 bot 丢失 | fake 适配器单测 + 状态机 + 退避重试 + 手工回归 |
| ChatID 类型切换波及面大 | 无存量数据直接切换；编译期类型检查 |
| extras 编辑器复杂化 | 平台 schema 注册驱动动态表单；中立字段与差异字段分离 |
| 凭证泄露 | AES-GCM + masked + 文档警示 |
| 删除误删 | 受限删除 + 409 提示 |
