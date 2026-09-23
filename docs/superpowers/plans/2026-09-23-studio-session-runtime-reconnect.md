# Studio Session Runtime Reconnect Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让 Studio 的后台 Run 独立于页面连接，支持用户切换页面、刷新或关闭浏览器后重新进入，并自动接回同一个 Run，补齐断线期间的事件后继续流式输出。

**Architecture:** 将 Session、Run、Connection、assistant-ui Runtime 分层。Run 状态和进度由服务端持久化，WebSocket 只负责临时传输；页面重新进入时先读取权威 Run 状态，再通过 AG-UI attach 模式连接原 Run。侧边栏从最新 Run 状态显示执行中、等待用户输入和已完成状态。

**Tech Stack:** Go Studio domain/application、GORM、AG-UI WebSocket、`@assistant-ui/react-ag-ui` `ThreadHistoryAdapter.unstable_resume`、TanStack Query、AI Elements、Vitest、Go test。

**Spec:** `docs/superpowers/specs/2026-09-20-ai-studio-technical-design.md`

## Global Constraints

- 保持 AG-UI 作为事件协议，不把 AG-UI 事件直接当作领域模型。
- AI Elements 继续负责 Chat UI，assistant-ui 只负责 runtime、history 和 AG-UI 适配。
- 所有跨页面、跨连接、跨进程需要恢复的状态必须由服务端持久化，不能依赖 React state、WebSocket 生命周期或 goroutine 生命周期。
- 前端优先复用 `web/admin/src/components/ui` 和 Pixoma 语义令牌；状态点使用 `StatusDot` 能力扩展，不新增裸色。
- 保留当前工作区已有修改，不使用破坏性 git 操作覆盖其他任务的改动。
- 每个任务先写失败测试，再写最小实现；每个任务完成后运行对应的 Go/Vitest/TypeScript 检查。

## 文件结构与职责

本次工作按以下边界拆分：

- `internal/studio/domain/repository.go`：增加批量读取 Session 最新 Run 和 Run 进度的 Port。
- `internal/studio/infrastructure/persistence/gorm_repository.go`：实现批量最新 Run、Run 进度快照和事件游标持久化。
- `internal/httpapi/studio/handler.go`：把最新 Run 状态返回给 Session 列表和详情。
- `internal/httpapi/studio/agui.go`：增加 attach 已有 Run 的 AG-UI 传输分支。
- `internal/studio/application/executor.go`：维护流式 assistant/reasoning 进度快照，不改变最终消息语义。
- `internal/studio/application/event_stream.go`：补充 attach 所需的实时历史和订阅边界。
- `web/admin/src/lib/api/studio.ts`：同步 Session 最新 Run、进度和 attach 类型。
- `web/admin/src/lib/agui-websocket-agent.ts`：保留新 Run 创建路径，增加 attach 参数和连接生命周期。
- `web/admin/src/lib/studio-run-connection.ts`：把 AG-UI attach 事件转换为 assistant-ui 所需的累计快照。
- `web/admin/src/features/studio/studio-chat.tsx`：用 history resume 接入活动 Run，避免重新发送用户消息。
- `web/admin/src/features/studio/studio-workspace.tsx`：以服务端 Run 状态初始化 Chat，并刷新 Session 列表。
- `web/admin/src/features/studio/studio-sidebar.tsx`：显示执行中、等待输入和完成状态点。
- `web/admin/src/components/status-dot.tsx`：向后兼容地增加运行态 tone/pulse 能力。

## 运行状态和协议约定

稳定 Run ID 与临时客户端 Run ID 必须分开：

```text
studioRunId  服务端持久化 Run ID，页面重连始终复用
clientRunId  当前 AG-UI 连接的临时 runId，重连时可以变化
sequence     已持久化 Studio event 的单调序号
```

AG-UI 初始请求继续使用现有格式；重新连接时增加 Pixoma 的协议扩展字段：

```json
{
  "threadId": "session-id",
  "runId": "client-run-id",
  "attachRunId": "studio-run-id",
  "afterSequence": 42,
  "messages": []
}
```

`resume` 仍然只用于解决审批中断。`attachRunId` 只观察/接回已有 Run，绝不调用 `SendMessage`，也不创建第二个 Run。

### Task 1: 建立最新 Run 和 Session Runtime API 契约

**Files:**
- Modify: `internal/studio/domain/repository.go`
- Modify: `internal/studio/infrastructure/persistence/gorm_repository.go`
- Modify: `internal/httpapi/studio/handler.go`
- Modify: `internal/httpapi/studio/handler_test.go`
- Modify: `web/admin/src/lib/api/studio.ts`
- Test: `internal/studio/infrastructure/persistence/gorm_repository_test.go`

**Interfaces:**
- Produces `ListLatestSessionRuns(ctx context.Context, accountID string, sessionIDs []string) (map[string]*domain.Run, error)`。
- Produces `sessionView.LatestRun *runView`，字段包含 `id`、`status`、`updated_at`、`completed_at`。
- `GET /sessions` 和 `GET /sessions/:sessionID` 都返回 `latest_run`；没有 Run 时返回 `null`。

- [ ] **Step 1: 写失败测试，验证 Session 列表返回最新 Run**

在 `handler_test.go` 增加以下场景：同一个 Session 有一个已完成 Run 和一个 running Run，`GET /sessions` 必须返回更新时间最新的 running Run；无 Run 的 Session 返回 `latest_run: null`。

```go
func TestListSessionsIncludesLatestRun(t *testing.T) {
    // 创建 session、succeeded run、running run，确认只返回 running run。
    // 同时确认另一个没有 run 的 session 返回 null。
}
```

- [ ] **Step 2: 写失败测试，验证批量查询不会逐 Session 调用 ListSessionRuns**

在 GORM repository 测试中插入多个 Session 和多个 Run，确认结果按 `created_at DESC, id DESC` 选择每个 Session 的唯一最新 Run。

- [ ] **Step 3: 增加 Repository Port 和 GORM 实现**

实现批量读取：先按账号和 Session ID 查询每个 Session 的最大 `created_at`，再用 `created_at` 与 `id` 处理同一时间戳的并列记录，最终返回 `map[sessionID]*domain.Run`。禁止在 HTTP handler 中循环调用 `ListSessionRuns`。

- [ ] **Step 4: 扩展 HTTP view 和 TypeScript 类型**

```go
type sessionView struct {
    // existing fields...
    LatestRun *runView `json:"latest_run,omitempty"`
}
```

```ts
export type StudioSession = {
  // existing fields...
  latest_run?: StudioRun
}
```

- [ ] **Step 5: 运行测试**

运行：`go test ./internal/studio/infrastructure/persistence ./internal/httpapi/studio`

预期：全部通过，且现有 Session API 响应兼容没有 Run 的旧数据。

- [ ] **Step 6: 提交独立变更**

提交信息：`feat(studio): expose latest run runtime state`

### Task 2: 增加 AG-UI attach 已有 Run 的后端协议

**Files:**
- Modify: `internal/httpapi/studio/agui.go`
- Modify: `internal/studio/application/event_stream.go`
- Modify: `internal/httpapi/studio/handler_test.go`
- Test: `internal/studio/application/event_stream_test.go`

**Interfaces:**
- `aguiRunInput.AttachRunID string json:"attachRunId"`。
- `aguiRunInput.AfterSequence uint64 json:"afterSequence"`。
- `prepareAGUIRun` 在 attach 分支返回已存在的 `studioRunID`，不调用 `Service.SendMessage`。
- attach 必须校验 `accountID`、`ThreadID`、`AttachRunID` 三者关系。

- [ ] **Step 1: 写失败测试，验证 attach 不创建新 Run**

测试请求只带 `threadId`、临时 `runId`、`attachRunId`，不带用户消息。断言：

1. 响应包含 `RUN_STARTED`；
2. `metadata.studioRunId` 等于原 Run；
3. Session 的 Run 数量不增加；
4. 原 Run 的事件继续被输出。

- [ ] **Step 2: 写失败测试，覆盖安全和状态边界**

覆盖以下情况：

- Run 不属于当前账号；
- Run 不属于 `threadId`；
- `attachRunId` 为空但没有用户文本和审批 resume；
- Run 在 attach 前已经进入 `waiting_approval`；
- Run 在详情查询后已经完成；
- 重复 attach 不会创建新 Run。

对于已完成 Run，允许只读回放并正常结束；对于不存在或越权 Run 返回明确错误。

- [ ] **Step 3: 实现 prepareAGUIRun attach 分支**

分支顺序固定为：

```text
attachRunId != ""  → 校验并订阅原 Run
resume 非空        → 解决审批并订阅原 Run
否则               → SendMessage 创建新 Run
```

attach 分支读取当前最大持久化事件序号，优先使用请求中的 `afterSequence`，但不能允许客户端跳过尚未确认的序号。返回的 `after` 必须取服务端确认值。

- [ ] **Step 4: 让 EventHub 的订阅边界可重复使用**

保留现有 `SubscribeAfter(runID, after)` 语义，补充测试确认：

- 已持久化事件不会重复回放；
- `Sequence == 0` 的流式 delta 在 after 之后仍然可以回放；
- 订阅建立前和建立后的事件不会丢失；
- Run 完成后连接能收到唯一的 `RUN_FINISHED`。

- [ ] **Step 5: 运行后端测试**

运行：`go test ./internal/studio/application ./internal/httpapi/studio`

- [ ] **Step 6: 提交独立变更**

提交信息：`feat(studio): attach agui stream to existing run`

### Task 3: 持久化运行中的流式进度快照

**Files:**
- Modify: `internal/studio/domain/model.go`
- Modify: `internal/studio/domain/repository.go`
- Modify: `internal/studio/infrastructure/persistence/gorm_repository.go`
- Modify: `internal/studio/application/executor.go`
- Modify: `internal/httpapi/studio/agui.go`
- Modify: `internal/httpapi/studio/handler.go`
- Modify: `internal/studio/infrastructure/persistence/gorm_repository_test.go`
- Test: `internal/studio/application/executor_stream_test.go`

**Interfaces:**
- 新增 `domain.RunProgress`：`RunID`、`SessionID`、`AssistantMessageID`、`AssistantText`、`ReasoningText`、`ToolCallsJSON`、`LastSequence`、`UpdatedAt`。
- Repository 增加：

```go
GetRunProgress(ctx context.Context, accountID, runID string) (*domain.RunProgress, error)
UpsertRunProgress(ctx context.Context, progress *domain.RunProgress) error
DeleteRunProgress(ctx context.Context, accountID, runID string) error
```

- `executionWriter` 在流式 assistant/reasoning 过程中维护内存累计值，并按时间或字节阈值写入快照。

- [ ] **Step 1: 写失败测试，验证断开后仍能恢复半截 assistant 文本**

测试执行器发出三次文本 delta，只持久化一次快照，然后模拟没有 EventHub 历史的 attach。断言 attach 输出 `TEXT_MESSAGE_START` 和累计文本，而不是空 assistant message。

- [ ] **Step 2: 写失败测试，验证快照不会制造最终重复消息**

Run 完成后，最终 `Message` 和 `TEXT_MESSAGE_END` 只能出现一次；progress snapshot 被删除或标记为完成。

- [ ] **Step 3: 增加 GORM progress 表模型**

在 `gorm_repository.go` 增加 `RunProgressRow` 和 `Models()` 注册，唯一键使用 `run_id`，并为 `account_id`、`session_id` 建索引。不要把每个 token 写成一条数据库记录。

- [ ] **Step 4: 实现批量/节流保存**

保存条件使用以下任一条件：距上次快照至少 200ms，或累计文本新增至少 1024 字节，或进入 reasoning/text/tool 边界，或 Run 状态发生变化。`ToolCallsJSON` 至少记录每个活动 tool call 的 ID、名称、累计 args 和是否已有 result。保存失败不能让模型流直接崩溃，但必须记录结构化错误并在下一次阈值触发时重试。

- [ ] **Step 5: 在 attach 无内存历史时发送快照**

如果 EventHub 没有可回放的 live history，先根据 `RunProgress` 生成标准 AG-UI start/content 事件，再订阅后续实时事件。快照只恢复当前未结束消息，不发送 `TEXT_MESSAGE_END`。

详情接口同时返回当前活动 Run 的 `run_progress`；Session 列表只返回轻量的 `latest_run`，避免侧边栏携带大段文本。

- [ ] **Step 6: 运行持久化测试**

运行：`go test ./internal/studio/application ./internal/studio/infrastructure/persistence ./internal/httpapi/studio`

- [ ] **Step 7: 提交独立变更**

提交信息：`feat(studio): persist in-flight run progress`

### Task 4: 实现前端 attach 连接和 assistant-ui resume 适配

**Files:**
- Modify: `web/admin/src/lib/agui-websocket-agent.ts`
- Create: `web/admin/src/lib/studio-run-connection.ts`
- Modify: `web/admin/src/features/studio/studio-chat.tsx`
- Test: `web/admin/src/lib/studio-run-connection.test.ts`
- Test: `web/admin/src/features/studio/studio-chat.test.tsx`

**Interfaces:**
- `StudioWebSocketAgent` 保持当前新 Run API，并增加 attach 配置：

```ts
type StudioAttachOptions = {
  studioRunId: string
  afterSequence?: number
}
```

- 新增 `StudioRunConnection.resume(options): AsyncGenerator<ChatModelRunResult>`，负责把 AG-UI 事件转换成累计快照。
- 不导入 `@assistant-ui/react-ag-ui` 的私有 `RunAggregator` 或深层内部模块；事件聚合逻辑放在 Pixoma 自己的适配文件中。

- [ ] **Step 1: 写失败测试，验证事件聚合输出累计快照**

输入 start/content/content/end 事件，断言每次 yield 的文本是累计值：`你`、`你好`、`你好！`，而不是单个 delta。reasoning、tool args、tool result 和 interrupt 也必须保留累计状态。

- [ ] **Step 2: 写失败测试，验证 WebSocket 断线重连不发送用户消息**

模拟第一次连接关闭，再次调用 resume。断言第二次请求只携带 `attachRunId` 和游标，不携带新的用户文本，且 `clientRunId` 可以重新生成。

- [ ] **Step 3: 实现 StudioRunConnection**

连接步骤固定为：

```text
打开 WS
→ 发送 attach request
→ 校验 threadId / studioRunId
→ 读取 AG-UI 事件
→ 更新累计消息快照
→ 收到 RUN_FINISHED success/interrupt/error 后结束 generator
```

连接关闭时只结束当前 generator，不调用“取消 Run”接口。可见性恢复和网络恢复由外层重新调用 resume 完成。

- [ ] **Step 4: 接入 ThreadHistoryAdapter**

当 `props.latestRun.status` 为 `queued`、`running` 或 `waiting_approval` 时，`history.load()` 返回：

```ts
return {
  ...ExportedMessageRepository.fromArray(messages),
  unstable_resume: true,
}
```

并实现 `history.resume()`：读取当前 `latestRun.id`，调用 `StudioRunConnection.resume()`。`waiting_approval` 也必须 attach，使中断卡片重新出现，但不自动提交审批。

- [ ] **Step 5: 防止运行中的 props 更新重建 runtime**

`StudioChat` 的 agent 实例不能因为 transcript 或 latest run 查询刷新而重复创建。将“新 Run 配置”和“attach 状态”分离，使用稳定 ref 保存当前连接，只有 Session ID 变化时才替换 runtime。

- [ ] **Step 6: 运行前端测试**

运行：`pnpm --dir web/admin exec vitest run src/lib/studio-run-connection.test.ts src/features/studio/studio-chat.test.tsx`

预期：覆盖文本续流、reasoning 聚合、审批中断和 attach 请求体。

- [ ] **Step 7: 提交独立变更**

提交信息：`feat(studio): resume active runs in assistant ui`

### Task 5: 页面可见性、网络恢复和 Query 对账

**Files:**
- Modify: `web/admin/src/features/studio/studio-workspace.tsx`
- Modify: `web/admin/src/features/studio/studio-chat.tsx`
- Modify: `web/admin/src/lib/api/studio.ts`
- Test: `web/admin/src/features/studio/studio-workspace.contract.test.ts`

**Interfaces:**
- `getStudioSession(sessionId)` 返回 `latest_run` 和可选 `run_progress`。
- Studio Chat 对外提供 `onRuntimeStateChange`，由 Workspace 更新 Session 列表缓存。

- [ ] **Step 1: 写失败测试，验证回到页面先读服务端状态**

模拟本地 runtime 为空、服务端 latest run 为 running，断言 Chat 在可发送之前进入 attach/resuming 状态，不显示空闲 composer 状态。

- [ ] **Step 2: 增加详情查询的恢复策略**

对当前 Chat 详情设置 `refetchOnMount: 'always'`、`refetchOnWindowFocus: true`、`staleTime: 0`。详情请求完成前保留 skeleton 或 reconnecting 状态，不能把旧缓存渲染成“待输入”。

- [ ] **Step 3: 增加连接恢复触发器**

监听 `visibilitychange` 和 `online`：页面重新可见或网络恢复时，重新读取 Session 详情；如果 Run 仍非终态，调用 attach；如果已经终态，只刷新消息和状态，不建立无意义连接。

- [ ] **Step 4: 同步 Query 缓存**

Run 开始、等待审批、完成或失败时，同时 invalidate：

```ts
['studio', 'sessions']
['studio', 'session', sessionId]
```

切换 Session 时取消前一 Session 的 attach 监听，禁止旧响应覆盖新 Session。

- [ ] **Step 5: 运行类型和契约测试**

运行：`pnpm --dir web/admin exec vitest run src/features/studio/studio-workspace.contract.test.ts && pnpm --dir web/admin exec tsc -b --pretty false`

- [ ] **Step 6: 提交独立变更**

提交信息：`feat(studio): reconcile runtime state on visibility changes`

### Task 6: 左侧 Session 状态点和运行态语义

**Files:**
- Modify: `web/admin/src/components/status-dot.tsx`
- Modify: `web/admin/src/features/studio/studio-sidebar.tsx`
- Modify: `web/admin/src/features/studio/studio-workspace.tsx`
- Test: `web/admin/src/features/studio/studio-workspace.contract.test.ts`

**Interfaces:**
- `StatusDot` 增加可选 `state: 'ok' | 'warn' | 'active'` 和 `pulse?: boolean`，保持现有 `problems` 调用方式不变。
- Studio 状态映射函数固定为：

```ts
function studioRunTone(status?: StudioRunStatus) {
  if (status === 'succeeded') return { state: 'ok', label: '本轮已完成' }
  if (status === 'waiting_approval') return { state: 'warn', label: '等待你的批准' }
  if (status === 'queued' || status === 'running') {
    return { state: 'active', label: '正在执行', pulse: true }
  }
  if (status === 'failed' || status === 'cancelled') {
    return { state: 'warn', label: '本轮未完成' }
  }
  return null
}
```

- [ ] **Step 1: 写失败契约测试**

断言 Session 行右侧渲染 `StatusDot`，并且包含 waiting、active、succeeded 的 label 和 `data-state`。断言没有 latest Run 时不渲染点。

- [ ] **Step 2: 扩展 StatusDot**

继续使用 `bg-success`、`bg-warning`、`bg-info` 等语义令牌，不写裸色。`active` 允许脉冲，但必须提供静态 aria-label 和 title。

- [ ] **Step 3: 在 StudioSidebar 显示状态点**

状态点放在 Session 行最右侧，不能挤压标题；移动端 Sheet 和桌面 Sidebar 使用同一组件。黄色只表示需要用户处理或本轮未完成，绿色只表示本轮成功完成。

- [ ] **Step 4: 增加列表刷新**

`useQuery(['studio', 'sessions'])` 设置轻量 `refetchInterval: 2500`，页面不可见时暂停，重新可见时立即刷新。当前 Chat 的事件状态变化也主动 invalidate 列表。

- [ ] **Step 5: 运行 UI 测试**

运行：`pnpm --dir web/admin exec vitest run src/features/studio/studio-workspace.contract.test.ts && pnpm --dir web/admin exec tsc -b --pretty false`

- [ ] **Step 6: 提交独立变更**

提交信息：`feat(studio): show run status in session sidebar`

### Task 7: 端到端恢复场景和回归验证

**Files:**
- Modify: `internal/httpapi/studio/handler_test.go`
- Modify: `internal/studio/application/approval_test.go`
- Create: `web/admin/src/features/studio/studio-runtime-reconnect.test.tsx`
- Modify: `web/admin/src/features/studio/studio-workspace.contract.test.ts`
- Modify: `docs/superpowers/specs/2026-09-20-ai-studio-technical-design.md`（补充最终 attach/checkpoint 约定）

**Interfaces:**
- 测试必须通过公开 API、模拟 WebSocket/EventHub 和 Session query 验证行为，不能依赖私有 React state。

- [ ] **Step 1: 增加“页面切走后继续流式”集成测试**

场景：创建 Run，发送 `TEXT_MESSAGE_START` 和部分 delta，关闭连接，继续发布 delta，再 attach。断言 attach 后先得到已有文本，再得到新 delta 和唯一 `RUN_FINISHED`。

- [ ] **Step 2: 增加“关闭浏览器后重新进入”测试**

场景：不保留前端 runtime，只保留数据库中的 Session、Run、Progress；重新加载详情并 attach。断言半截文本、reasoning 和运行状态恢复。

- [ ] **Step 3: 增加审批场景测试**

场景：Run 进入 `waiting_approval`，页面关闭；重新进入显示黄色状态点和审批卡片；批准后仍复用原 Run，继续输出，不新增 Run。

- [ ] **Step 4: 增加终态场景测试**

场景：用户离开期间 Run 成功、失败或取消；回来后不重新建立流连接，直接显示终态消息和左侧状态。

- [ ] **Step 5: 增加重复 attach 和多标签测试**

两个客户端同时 attach 同一个 Run，只允许一个审批响应生效；两个客户端都能收到终态；任何客户端都不能重复触发模型执行。

- [ ] **Step 6: 执行完整验证**

运行：

```bash
go test ./...
pnpm --dir web/admin exec vitest run
pnpm --dir web/admin exec tsc -b --pretty false
git diff --check
```

预期：Go、Vitest、TypeScript 全部通过，且没有新的 diff whitespace 错误。

- [ ] **Step 7: 提交并更新设计文档**

提交信息：`test(studio): cover run reconnect and resume scenarios`

## 验收标准

1. 用户发送问题后切换到资产、Flow 或设置页面，后台 Run 继续执行。
2. 用户关闭浏览器，稍后重新打开同一 Session，页面能识别 `running` Run 并自动 attach。
3. 断开期间产生的 assistant 文本、reasoning、tool call 和 approval interrupt 不丢失、不重复。
4. attach 不会重复发送原始用户消息，不会创建第二个 Run。
5. Run 等待审批时，左侧显示黄色点，Chat 显示可操作审批卡片。
6. Run 成功完成后，左侧显示绿色点；回来时不再显示正在执行或待输入状态。
7. 网络断开、页面刷新和多标签页不会让本地 `isRunning` 覆盖服务端真实状态。
8. 服务进程重启后，后台 Runner 能恢复 queued/running Run；已有进度快照能恢复到最近检查点。
9. 所有 Session/Run/Approval 权限校验都基于当前账号，不能通过 attachRunId 读取其他账号数据。

## 风险和取舍

- `ThreadHistoryAdapter.unstable_resume` 是 assistant-ui 的实验性能力，必须封装在 `studio-run-connection.ts`，后续升级只影响这一层。
- 当前 EventHub 是进程内内存结构；没有 progress snapshot 时，服务重启会丢失未完成的 token 级增量。因此进度快照是核心能力的一部分，不应只做前端重连。
- 运行态轮询先采用 2.5 秒轻量查询，避免第一版引入全局 Session SSE；如果 Session 数量增长，再将 `/sessions` runtime 字段拆成全局活动 Run endpoint。
- 不允许把“页面关闭”映射为 Run cancel；取消只能由用户主动点击停止或后端明确失败。
