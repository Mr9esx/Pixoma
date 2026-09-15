---
change: wecom-aibot-channel
design-doc: docs/superpowers/specs/2026-09-15-wecom-aibot-channel-design.md
base-ref: 6c30b89ecfe0064e74b4fe3eb2018f548c461a3b
---

# 企业微信智能机器人渠道实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use `executing-plans` to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 通过企业微信智能机器人长连接提供与 Telegram 一致的工作流会话能力，并修正三种 IM 平台的运行时装配。

**Architecture:** 从 Telegram 提炼平台无关的会话核心，Telegram、飞书、企微各自实现事件解码、媒体桥接和平台渲染。企微协议通过本地 `Client` 接口封装 `wecom-aibot-go`，任务结果以持久化会话地址主动推送。

**Tech Stack:** Go 1.25、`wecom-aibot-go`（固定提交 `0d971de840ad587bc90096992c88310b04cd52d8`）、Telegram Bot API、飞书 OAPI、React、TanStack Query、shadcn/ui、Vitest。

## 全局约束

- 所有生产代码先写失败测试，确认 RED 后再写最小实现。
- 不把 Secret、AES key、媒体下载 URL 或完整消息内容写入日志或 API 响应。
- 企微只支持智能机器人 WSS，不实现 HTTP 回调或第二条探测连接。
- `single/<userid>` 和 `group/<chatid>` 是企微 `ChannelAddr.ExternalChatID` 的持久化格式。
- 后台沿用 `web/admin/src/components/ui` 的 shadcn/ui 组件、Pixoma 语义令牌、`StatusDot` 与「已启用 / 已停用」文案。
- 每个提交只暂存本 change 文件，且使用 `李卓洲 <1138099359@qq.com>` 作为 author 与 committer。

---

### Task 1: 企微凭证、HTTP API 与健康探测

**Files:**
- Modify: `internal/channels/domain/credential.go`
- Modify: `internal/channels/domain/credential_test.go`
- Modify: `internal/channels/application/service.go`
- Modify: `internal/channels/application/service_test.go`
- Modify: `internal/httpapi/channels/handler.go`
- Modify: `internal/httpapi/channels/handler_test.go`

**Interfaces:**
- Produces: `domain.Credential{WeComBotID, WeComSecret, WeComWSURL}`。
- Produces: `Service.CheckReachability` 对 `PlatformWeCom` 仅读取 `AdapterStatus`。
- Produces: 渠道 API 的 `bot_id`、`ws_url`、`bot_secret` 请求字段；响应只回显 Bot ID 与 WSS 地址。

- [x] **Step 1: 写凭证与探测的失败测试**

```go
func TestService_CreateWeComEncryptsAndMasks(t *testing.T) {
    svc := &Service{Store: &memStore{rows: map[string]domain.Channel{}}, Key: make([]byte, 32)}
    got, err := svc.CreateWithCredential(context.Background(), "wc-1", domain.PlatformWeCom, "企微助手", domain.Credential{
        WeComBotID: "aibot_1", WeComSecret: "secret-12345678", WeComWSURL: "wss://private.example/ws",
    }, "")
    if err != nil || got.CredentialCiphertext == "" { t.Fatalf("create = %#v, %v", got, err) }
    if masked, _ := svc.Masked(context.Background(), "wc-1"); masked == "secret-12345678" { t.Fatal("secret leaked") }
}

func TestService_CheckReachabilityWeComUsesAdapterStatus(t *testing.T) {
    svc := &Service{Store: storeWithWeCom(t), Key: make([]byte, 32), AdapterStatus: func(context.Context, string) (string, string, bool) {
        return "running", "", true
    }}
    got, err := svc.CheckReachability(context.Background(), "wc-1")
    if err != nil || !got.OK || got.Kind != ReachabilityOK { t.Fatalf("got %#v, %v", got, err) }
}
```

- [x] **Step 2: 运行测试确认 RED**

Run: `go test ./internal/channels/application ./internal/channels/domain -run 'TestService_(CreateWeCom|CheckReachabilityWeCom)' -count=1`

Expected: FAIL，因为 `Credential` 尚无企微字段且服务尚未实现企微状态探测。

- [x] **Step 3: 实现凭证验证、掩码和健康映射**

```go
type Credential struct {
    BotToken string `json:"bot_token,omitempty"`
    AppID string `json:"app_id,omitempty"`
    AppSecret string `json:"app_secret,omitempty"`
    WeComBotID string `json:"wecom_bot_id,omitempty"`
    WeComSecret string `json:"wecom_secret,omitempty"`
    WeComWSURL string `json:"wecom_ws_url,omitempty"`
}

case domain.PlatformWeCom:
    if strings.TrimSpace(cred.WeComBotID) == "" || strings.TrimSpace(cred.WeComSecret) == "" {
        return fmt.Errorf("channel: wecom bot_id/secret required")
    }
```

在 `CheckReachability` 中将 `running` 映射到 `ReachabilityOK`，`starting`、`stopping`、`error` 或找不到状态映射到 `ReachabilityOther`，并把 `lastErr` 作为诊断消息。不得调用网络。

- [x] **Step 4: 扩展创建、编辑和 DTO 测试后实现 HTTP 字段映射**

```go
type createBody struct {
    ID string `json:"id"`; Platform string `json:"platform"`; Name string `json:"name"`
    Token string `json:"token"`; AppID string `json:"app_id"`; AppSecret string `json:"app_secret"`
    BotID string `json:"bot_id"`; BotSecret string `json:"bot_secret"`; WSURL string `json:"ws_url"`
    ExtraInfo string `json:"extra_info"`
}
```

为 `PlatformWeCom` 构造专属 `Credential`；更新时读取原凭证，只覆盖请求中非空的企微字段。DTO 只增加 `bot_id` 和 `ws_url`，绝不增加 Secret 字段。

- [x] **Step 5: 运行后端相关测试**

Run: `go test ./internal/channels/domain ./internal/channels/application ./internal/httpapi/channels -count=1`

Expected: PASS。

- [x] **Step 6: 提交本任务**

```bash
git add internal/channels/domain/credential.go internal/channels/domain/credential_test.go internal/channels/application/service.go internal/channels/application/service_test.go internal/httpapi/channels/handler.go internal/httpapi/channels/handler_test.go
GIT_AUTHOR_NAME='李卓洲' GIT_AUTHOR_EMAIL='1138099359@qq.com' GIT_COMMITTER_NAME='李卓洲' GIT_COMMITTER_EMAIL='1138099359@qq.com' git commit -m 'feat(channels): add wecom credentials and health probe'
```

### Task 2: 共享会话核心与 Telegram 行为迁移

**Files:**
- Create: `internal/channels/conversation/controller.go`
- Create: `internal/channels/conversation/controller_test.go`
- Create: `internal/channels/conversation/store.go`
- Modify: `internal/channels/tg/adapter.go`
- Modify: `internal/channels/tg/callback.go`
- Modify: `internal/channels/tg/invoke_store.go`
- Modify: `internal/channels/tg/*_test.go`

**Interfaces:**
- Produces: `conversation.Controller`，持有 `Registry`、`Users`、`Blob`、`MenuReader`、`TemplateRenderer` 与逻辑渲染器。
- Produces: `HandleText`、`HandleMedia`、`HandleAction`、`HandleNotify`，全部以 `sharedkernel.ChannelAddr` 和平台无关事件输入工作。
- Consumes: `protocol.Result`、`protocol.Outbound`、`protocol.MediaBridge`。

- [x] **Step 1: 用 Telegram 既有场景写共享核心失败测试**

```go
func TestController_SubmitsImageOnlyForActiveImageStep(t *testing.T) {
    ctl, calls := newController(t, activeImageSession())
    err := ctl.HandleMedia(context.Background(), inbound("tg-1", "123"), sharedkernel.BlobRef{Key: "tg/123/a.png", MIME: "image/png"})
    if err != nil { t.Fatal(err) }
    if calls.step != "media" || calls.blob.Key != "tg/123/a.png" { t.Fatalf("calls = %#v", calls) }
}

func TestController_ExpiredActionReturnsReselectEffect(t *testing.T) {
    ctl, out := newController(t, noSession())
    _ = ctl.HandleAction(context.Background(), inbound("tg-1", "123"), "missing")
    if out.lastText != "操作已过期，重新选择。" { t.Fatalf("text = %q", out.lastText) }
}
```

- [x] **Step 2: 运行测试确认 RED**

Run: `go test ./internal/channels/conversation -run 'TestController_(SubmitsImageOnlyForActiveImageStep|ExpiredActionReturnsReselectEffect)' -count=1`

Expected: FAIL，因为 `conversation` 包不存在。

- [ ] **Step 3: 实现最小核心并迁移 Telegram 的状态容器**

```go
type Inbound struct {
    Addr sharedkernel.ChannelAddr
    ExternalUserID string
}

func (c *Controller) HandleMedia(ctx context.Context, in Inbound, ref sharedkernel.BlobRef) error {
    return c.invoke(ctx, in, "open_case", map[string]any{"step": "media", "blob": map[string]any{"key": ref.Key, "mime": ref.MIME}})
}
```

将 Telegram 的回退栈与一次性动作令牌移入 `conversation/store.go`；移动 `actionDispatch`、`sendCard`、`renderResult`、`HandleUserNotify` 的平台无关决策。核心只调用渲染端口，不导入 Telegram SDK。

- [ ] **Step 4: 改写 Telegram 适配器为事件解码和渲染层**

保留 `tg.BotMessenger`、`tgMediaBridge` 和 Bot handler 注册；`Adapter.HandleText`、`HandleUserMedia`、`HandleCallback`、`HandleNotify` 只把 Telegram 数据转换成核心输入或调用核心。迁移前后的现有 Telegram 测试必须不改断言语义。

- [ ] **Step 5: 运行 Telegram 与共享核心测试**

Run: `go test ./internal/channels/conversation ./internal/channels/tg -count=1`

Expected: PASS，且菜单、图片输入、回调、媒体 MIME 路由和通知测试保持通过。

- [ ] **Step 6: 提交本任务**

```bash
git add internal/channels/conversation internal/channels/tg
GIT_AUTHOR_NAME='李卓洲' GIT_AUTHOR_EMAIL='1138099359@qq.com' GIT_COMMITTER_NAME='李卓洲' GIT_COMMITTER_EMAIL='1138099359@qq.com' git commit -m 'refactor(channels): extract shared conversation core'
```

### Task 3: 飞书薄适配层与显式平台工厂

**Files:**
- Modify: `internal/channels/feishu/adapter.go`
- Modify: `internal/channels/feishu/feishu_test.go`
- Modify: `apps/pixoma/internal/telegram/telegram.go`
- Modify: `apps/pixoma/internal/telegram/telegram_test.go`

**Interfaces:**
- Consumes: `conversation.Controller`。
- Produces: `channelFactory.Create(ChannelSnapshot)` 对 Telegram、飞书、MCP 明确分发。
- Produces: `notifyRegistry` 中飞书渠道的注册与注销。

- [ ] **Step 1: 写工厂和飞书委托的失败测试**

```go
func TestChannelFactory_CreatesFeishuAdapterWithoutTelegramBot(t *testing.T) {
    f := newTestFactory(t)
    ad, err := f.Create(channelapp.ChannelSnapshot{ID: "fs-1", Platform: "feishu", AppID: "cli_1", AppSecret: "secret"})
    if err != nil { t.Fatal(err) }
    if _, ok := ad.(*feishuAdapterWrapper); !ok { t.Fatalf("adapter = %T", ad) }
}

func TestFeishuAdapter_DelegatesImageBlobToConversation(t *testing.T) {
    ctl := newRecordingController(t)
    ad := newFeishuAdapterForTest(t, ctl, &fakeIM{download: []byte("PNG")})
    ad.dispatchInbound(context.Background(), inbound{ChatID: "oc_1", SenderUserID: "ou_1", ImageKey: "img_1", MIME: "image/png"})
    got := ctl.LastInput()
    if got.Step != conversation.StepMedia || got.Blob.MIME != "image/png" { t.Fatalf("input = %#v", got) }
}
```

- [ ] **Step 2: 运行测试确认 RED**

Run: `go test ./apps/pixoma/internal/telegram ./internal/channels/feishu -run 'Test(ChannelFactory_CreatesFeishuAdapterWithoutTelegramBot|FeishuAdapter_DelegatesImageBlobToConversation)' -count=1`

Expected: FAIL，因为现有工厂将飞书按 Telegram 创建。

- [ ] **Step 3: 迁移飞书入站、菜单、卡片与通知逻辑**

让 `FeishuAdapter` 保存 `*conversation.Controller`。图片仍由飞书 IM 下载并写 Blob，之后调用 `Controller.HandleMedia`；文本和飞书卡片事件分别调用 `HandleText`、`HandleAction`。实现 `conversation.Renderer` 的飞书渲染器，将菜单和卡片转换为飞书支持的消息结构。新增 `feishuAdapterWrapper`：其 `Start` 注册飞书 `NotifyHandler` 后启动适配器，`Stop` 先停止适配器再注销处理器，使工厂返回的类型满足 `application.Adapter`。

- [ ] **Step 4: 实现显式工厂路由和凭证快照**

```go
switch domain.Platform(snap.Platform) {
case domain.PlatformTelegram:
    return f.newTelegram(snap)
case domain.PlatformFeishu:
    return f.newFeishu(snap)
case domain.PlatformMCP:
    return nil, fmt.Errorf("channel: mcp has no IM adapter")
default:
    return nil, fmt.Errorf("channel: unsupported IM platform %q", snap.Platform)
}
```

`channelSnapshotStore` 需要填充飞书 App ID/App Secret，并以完整序列化凭证哈希；不要继续只散列 Telegram Token。

- [ ] **Step 5: 运行迁移测试**

Run: `go test ./apps/pixoma/internal/telegram ./internal/channels/feishu ./internal/channels/conversation -count=1`

Expected: PASS。

- [ ] **Step 6: 提交本任务**

```bash
git add apps/pixoma/internal/telegram/telegram.go apps/pixoma/internal/telegram/telegram_test.go internal/channels/feishu
GIT_AUTHOR_NAME='李卓洲' GIT_AUTHOR_EMAIL='1138099359@qq.com' GIT_COMMITTER_NAME='李卓洲' GIT_COMMITTER_EMAIL='1138099359@qq.com' git commit -m 'feat(channels): route feishu through shared runtime'
```

### Task 4: Assembler 的停止优先级与运行时状态

**Files:**
- Modify: `internal/channels/application/assembler.go`
- Modify: `internal/channels/application/assembler_test.go`
- Modify: `internal/channels/application/adapter.go`

**Interfaces:**
- Produces: 每个待停止实例携带渠道 ID 的 `stopJob`。
- Produces: 停止失败时对应 `AdapterStatus{State: stateError, LastErr: err}`，且本轮不调用 `Factory.Create`。

- [ ] **Step 1: 写停止失败不得替换的失败测试**

```go
func TestAssembler_CredentialChangeDoesNotStartReplacementWhenStopFails(t *testing.T) {
    as, store, factory := newAssemblerWithStopError(t)
    runOnce(as, context.Background())
    store.upsert("wc-1", "new-secret", true)
    runOnce(as, context.Background())
    if factory.createCount("new-secret") != 0 { t.Fatal("replacement started after stop failure") }
    if as.Status()["wc-1"].State != stateError { t.Fatalf("status = %#v", as.Status()["wc-1"]) }
}
```

- [ ] **Step 2: 运行测试确认 RED**

Run: `go test ./internal/channels/application -run TestAssembler_CredentialChangeDoesNotStartReplacementWhenStopFails -count=1`

Expected: FAIL，因为当前协调器记录 Stop 错误后仍启动 replacement。

- [ ] **Step 3: 以带渠道 ID 的停止任务替代裸 Adapter 列表**

```go
type stopJob struct { id string; adapter Adapter }

for _, job := range toStop {
    if err := job.adapter.Stop(ctx); err != nil {
        a.finishStopError(job.id, err)
        continue
    }
    if start, ok := startsByID[job.id]; ok {
        a.startOne(ctx, job.id, start.snap)
    }
}
```

为删除和停用渠道保留「停止失败只记录错误、不创建 replacement」语义；启动候选须等关联停止任务成功后才执行。

- [ ] **Step 4: 运行应用层测试**

Run: `go test ./internal/channels/application -count=1`

Expected: PASS。

- [ ] **Step 5: 提交本任务**

```bash
git add internal/channels/application/adapter.go internal/channels/application/assembler.go internal/channels/application/assembler_test.go
GIT_AUTHOR_NAME='李卓洲' GIT_AUTHOR_EMAIL='1138099359@qq.com' GIT_COMMITTER_NAME='李卓洲' GIT_COMMITTER_EMAIL='1138099359@qq.com' git commit -m 'fix(channels): wait for adapter shutdown before replacement'
```

### Task 5: 企微协议客户端封装与媒体桥

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`
- Create: `internal/channels/wecom/client.go`
- Create: `internal/channels/wecom/client_test.go`
- Create: `internal/channels/wecom/media.go`
- Create: `internal/channels/wecom/media_test.go`

**Interfaces:**
- Produces: `wecom.Client`，包括 `Run(context.Context) error`、`SendReply`、`SendPush`、`UploadMedia`、`DownloadMedia`、`OnMessage`、`OnEvent`。
- Produces: `wecom.InboundMessage{ReqID, UserID, ChatID, ChatType, Text, Image}` 与 `wecom.Event{ReqID, Type, EventKey}`。
- Consumes: 固定提交的 `github.com/seastart/wecom-aibot-go`。

- [ ] **Step 1: 写客户端包装和 AES 图片下载失败测试**

```go
func TestClientAdapter_MapsGroupMessageAndReqID(t *testing.T) {
    raw := fakeSDKMessage{ReqID: "req-1", ChatType: groupChat, ChatID: "room-1", From: "user-1", Text: "生成图片"}
    got := mapMessage(raw)
    if got.ReqID != "req-1" || got.ChatType != Group || got.ChatID != "room-1" { t.Fatalf("got %#v", got) }
}

func TestMediaBridge_DownloadRejectsOversizedPlaintext(t *testing.T) {
    bridge := MediaBridge{Download: oversizedDecryptedFile}
    if _, err := bridge.Download(context.Background(), "https://example.invalid/file", "aes-key"); err == nil { t.Fatal("want size error") }
}
```

- [ ] **Step 2: 运行测试确认 RED**

Run: `go test ./internal/channels/wecom -run 'Test(ClientAdapter_MapsGroupMessageAndReqID|MediaBridge_DownloadRejectsOversizedPlaintext)' -count=1`

Expected: FAIL，因为 `wecom` 包不存在。

- [ ] **Step 3: 添加固定提交依赖并实现 SDK 包装**

Run: `go get github.com/seastart/wecom-aibot-go@0d971de840ad587bc90096992c88310b04cd52d8`

包装层必须把 SDK 回调转换为本地类型；所有媒体下载经过 AES 解密、30 秒超时和与 Telegram 相同的 20 MiB 明文上限。上传 MIME 映射为 `image`、`file`、`video` 或 `voice`；动图使用图片素材类型。不要在业务代码直接导入 SDK。

- [ ] **Step 4: 运行企微协议单元测试**

Run: `go test ./internal/channels/wecom -count=1`

Expected: PASS。

- [ ] **Step 5: 提交本任务**

```bash
git add go.mod go.sum internal/channels/wecom
GIT_AUTHOR_NAME='李卓洲' GIT_AUTHOR_EMAIL='1138099359@qq.com' GIT_COMMITTER_NAME='李卓洲' GIT_COMMITTER_EMAIL='1138099359@qq.com' git commit -m 'feat(wecom): add intelligent bot protocol client'
```

### Task 6: 企微薄适配层、群聊与异步投递

**Files:**
- Create: `internal/channels/wecom/adapter.go`
- Create: `internal/channels/wecom/adapter_test.go`
- Create: `internal/channels/wecom/render.go`
- Create: `internal/channels/wecom/render_test.go`
- Modify: `apps/pixoma/internal/telegram/telegram.go`
- Modify: `apps/pixoma/internal/telegram/telegram_test.go`

**Interfaces:**
- Produces: `wecom.Adapter`，实现 `application.Adapter` 与 `application.NotifyHandler`。
- Produces: `encodeConversation(ChatType, chatID, userID string) sharedkernel.ChannelAddr`。
- Consumes: `conversation.Controller` 和 `wecom.Client`。

- [ ] **Step 1: 写企微地址、欢迎事件和主动推送失败测试**

```go
func TestEncodeConversation(t *testing.T) {
    if got := encodeConversation(Single, "", "u-1"); got.ExternalChatID != "single/u-1" { t.Fatalf("got %#v", got) }
    if got := encodeConversation(Group, "g-1", "u-1"); got.ExternalChatID != "group/g-1" { t.Fatalf("got %#v", got) }
}

func TestAdapter_EnterChatRepliesWelcomeBeforeDispatch(t *testing.T) {
    client := &fakeClient{}
    ad := newAdapterForTest(t, client)
    ad.handleEvent(context.Background(), Event{ReqID: "req-1", Type: EnterChat, UserID: "u-1"})
    if !client.hasReply("req-1") { t.Fatal("welcome reply missing") }
}

func TestAdapter_TaskSucceededPushesToOriginalGroup(t *testing.T) {
    ad, client := newAdapterForTest(t, &fakeClient{})
    _ = ad.HandleNotify(context.Background(), notifyFor("wc-1:group/g-1"))
    if !client.hasPush(Group, "g-1") { t.Fatal("group push missing") }
}
```

- [ ] **Step 2: 运行测试确认 RED**

Run: `go test ./internal/channels/wecom -run 'Test(EncodeConversation|Adapter_EnterChatRepliesWelcomeBeforeDispatch|Adapter_TaskSucceededPushesToOriginalGroup)' -count=1`

Expected: FAIL，因为适配器尚不存在。

- [ ] **Step 3: 实现适配器生命周期与事件映射**

`Start` 注册 SDK 回调并在 goroutine 中运行 `Client.Run`；`Stop` 取消 context 后等待 Run 退出。文本、图片和图片文档进入共享核心；群聊未 @ 机器人直接忽略。`enter_chat` 先发送欢迎效果，模板卡片事件先在 4 秒内确认/更新后交给核心动作令牌。未知或过期卡片动作返回重新选择提示。

- [ ] **Step 4: 实现渲染器和媒体路由**

`render.go` 将核心效果转成被动回复或主动推送：同步结果使用事件 `ReqID`，任务通知解析 `single/`、`group/` 前缀后调用 `SendPush`。读取 Blob 后上传媒体，再按 MIME 发送 `image`、`video` 或 `file`；无法映射的 URL 与文本使用文本回退。

- [ ] **Step 5: 将企微加入运行时工厂**

在 Task 3 的 `switch` 增加：

```go
case domain.PlatformWeCom:
    return f.newWeCom(snap)
```

创建后注册企微 `NotifyHandler`，停止时注销。工厂测试必须断言 `wecom` 不初始化 Telegram Bot，且 Bot ID、Secret、WSS 地址均来自快照。

- [ ] **Step 6: 运行企微与工厂测试**

Run: `go test ./internal/channels/wecom ./apps/pixoma/internal/telegram -count=1`

Expected: PASS。

- [ ] **Step 7: 提交本任务**

```bash
git add internal/channels/wecom apps/pixoma/internal/telegram/telegram.go apps/pixoma/internal/telegram/telegram_test.go
GIT_AUTHOR_NAME='李卓洲' GIT_AUTHOR_EMAIL='1138099359@qq.com' GIT_COMMITTER_NAME='李卓洲' GIT_COMMITTER_EMAIL='1138099359@qq.com' git commit -m 'feat(channels): add wecom intelligent bot adapter'
```

### Task 7: 后台创建、编辑和统一健康状态

**Files:**
- Modify: `web/admin/src/lib/api/channels.ts`
- Modify: `web/admin/src/lib/api/channels.test.ts`
- Modify: `web/admin/src/features/channels/create-channel-form.tsx`
- Modify: `web/admin/src/features/channels/channel-detail-panel.tsx`
- Modify: `web/admin/src/features/channels/channel-layout.contract.test.ts`
- Modify: `web/admin/src/lib/i18n/locales/zh.json`
- Modify: `web/admin/src/lib/i18n/locales/en.json`
- Modify: `web/admin/src/lib/i18n/copy-quality.test.ts`

**Interfaces:**
- Produces: `Channel{bot_id?: string; ws_url?: string}`。
- Produces: `createChannel` 与 `updateChannel` 支持 `botId`、`botSecret`、`wsUrl`。
- Consumes: 后端的 `adapter_state` 与 `last_check_kind`，复用既有健康计算函数。

- [ ] **Step 1: 写企微创建表单和 API 序列化失败测试**

```ts
it('serializes WeCom bot credentials without a Telegram token', async () => {
  await createChannel({ platform: 'wecom', name: '企微助手', botId: 'aibot_1', botSecret: 'secret', wsUrl: 'wss://private/ws' })
  expect(apiFetch).toHaveBeenCalledWith('/api/v1/channels', expect.objectContaining({
    body: JSON.stringify({ platform: 'wecom', name: '企微助手', bot_id: 'aibot_1', bot_secret: 'secret', ws_url: 'wss://private/ws' }),
  }))
})
```

在布局契约测试中断言 `platformWeCom`、Bot ID、Secret、WSS 地址字段存在，且企微编辑不展示 Telegram Token 标签。

- [ ] **Step 2: 运行测试确认 RED**

Run: `pnpm --dir web/admin test -- channels.test.ts channel-layout.contract.test.ts`

Expected: FAIL，因为前端类型、字段和翻译尚未存在。

- [ ] **Step 3: 实现 API 类型与表单分支**

```ts
type Platform = 'telegram' | 'mcp' | 'feishu' | 'wecom'

function toCreatePayload(): CreateChannelInput {
  if (platform === 'wecom') {
    return { platform, name: name.trim(), botId: botId.trim(), botSecret, wsUrl: wsUrl.trim() || undefined }
  }
  if (platform === 'feishu') {
    return { platform, name: name.trim(), appId: appId.trim(), appSecret }
  }
  if (platform === 'telegram') {
    return { platform, name: name.trim(), botToken }
  }
  return { platform, name: name.trim() }
}
```

使用 `Field`、`Input` 与 `SecretInput`；WSS 地址为可选字段。详情编辑页按平台展示 Bot ID、Secret 与 WSS 地址。新增中英文 i18n 键，中文文案使用「企业微信智能机器人」「Bot ID」「Secret」「WSS 地址」。

- [ ] **Step 4: 对齐健康展示**

复用 `alignPlatformHealth` 与 `resolvedEntityHealth`；当企微 `adapter_state` 不是 `running`，或没有成功的探测记录时，不渲染绿色成功状态。状态文本与按钮保持「已启用 / 已停用」「启用 / 停用」。

- [ ] **Step 5: 运行前端测试和类型检查**

Run: `pnpm --dir web/admin test -- channels.test.ts channel-layout.contract.test.ts copy-quality.test.ts`

Run: `pnpm --dir web/admin typecheck`

Expected: PASS。

- [ ] **Step 6: 提交本任务**

```bash
git add web/admin/src/lib/api/channels.ts web/admin/src/lib/api/channels.test.ts web/admin/src/features/channels/create-channel-form.tsx web/admin/src/features/channels/channel-detail-panel.tsx web/admin/src/features/channels/channel-layout.contract.test.ts web/admin/src/lib/i18n/locales/zh.json web/admin/src/lib/i18n/locales/en.json web/admin/src/lib/i18n/copy-quality.test.ts
GIT_AUTHOR_NAME='李卓洲' GIT_AUTHOR_EMAIL='1138099359@qq.com' GIT_COMMITTER_NAME='李卓洲' GIT_COMMITTER_EMAIL='1138099359@qq.com' git commit -m 'feat(admin): configure wecom intelligent bots'
```

### Task 8: 集成验收、架构文档与 OpenSpec 任务同步

**Files:**
- Create: `internal/channels/wecom/integration_test.go`
- Modify: `docs/architecture/overview.md`
- Modify: `docs/architecture/runtime.md`
- Modify: `docs/openspec/changes/wecom-aibot-channel/tasks.md`

**Interfaces:**
- Consumes: `wecom.Adapter`、共享会话核心、`comfy_mock`。
- Produces: 覆盖单聊、群 @、媒体输入、出图主动推送和热重载的集成验收。

- [ ] **Step 1: 写企微端到端失败测试**

```go
func TestWeComAdapter_ComfyMockGroupImageResultIsPushedToSameGroup(t *testing.T) {
    h := newWeComComfyMockHarness(t)
    h.ReceiveGroupMention("g-1", "u-1", "生成图片")
    h.CompleteTaskWithImage("outputs/result.png")
    h.AssertPush(Group, "g-1", "image")
}

func TestWeComAdapter_ReloadStopsOldConnectionBeforeStartingNewOne(t *testing.T) {
    h := newReloadHarness(t)
    h.RotateSecret("wc-1", "new-secret")
    h.AssertStopBeforeStart("wc-1")
}
```

- [ ] **Step 2: 运行测试确认 RED**

Run: `go test ./internal/channels/wecom -run 'TestWeComAdapter_(ComfyMockGroupImageResultIsPushedToSameGroup|ReloadStopsOldConnectionBeforeStartingNewOne)' -count=1`

Expected: FAIL，直到适配器、工厂和 harness 已完整接通。

- [ ] **Step 3: 完成既定 harness 后运行全量相关测试**

Run: `go test ./internal/channels/... ./apps/pixoma/internal/telegram/... -count=1`

Run: `pnpm --dir web/admin test -- channels.test.ts channel-layout.contract.test.ts`

Expected: PASS。

- [ ] **Step 4: 更新架构说明和勾选 OpenSpec 任务**

在 `docs/architecture/overview.md` 记录共享会话核心及 Telegram、飞书、企微三层适配器边界；在 `docs/architecture/runtime.md` 记录企微智能机器人只使用单条 WSS、探测复用运行状态、会话地址编码、被动回复与主动推送的分界。逐条勾选 `tasks.md` 中已由测试覆盖的 1.1–3.2 项，未验证项保持未勾选。

- [ ] **Step 5: 提交本任务**

```bash
git add internal/channels/wecom/integration_test.go docs/architecture/overview.md docs/architecture/runtime.md docs/openspec/changes/wecom-aibot-channel/tasks.md
GIT_AUTHOR_NAME='李卓洲' GIT_AUTHOR_EMAIL='1138099359@qq.com' GIT_COMMITTER_NAME='李卓洲' GIT_COMMITTER_EMAIL='1138099359@qq.com' git commit -m 'test(wecom): verify intelligent bot channel flow'
```

## 计划自检

- OpenSpec 的适配器、媒体、单连接、菜单/工作流、后台、`comfy_mock` 和架构文档要求分别覆盖于 Task 2–8。
- 企微主动推送依赖地址编码，已由 Task 6 单测和 Task 8 集成测试覆盖。
- 所有新增生产接口均在其首次出现任务中定义，且每项任务先有明确 RED 命令，再进入最小 GREEN 实现。
