# Studio Run 连续性修复实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让 Studio Chat 以持久 Run 状态和可重放事件为准，可靠地处理流式续接、审批、取消与进程重启。

**Architecture:** 数据库负责 Run 状态、事件序号和 Eino 审批检查点；EventHub 只通知有新数据。AG-UI 从持久日志按游标补发，前端连接控制器在断线后重连并对账，Chat 只负责展示。

**Tech Stack:** Go、GORM、Eino ADK、AG-UI WebSocket、React、TanStack Query、assistant-ui、AI Elements、Go test、Vitest。

**Spec:** `docs/superpowers/specs/2026-09-23-studio-run-continuity-repair-design.md`

## Global Constraints

- 当前 `feat-studio` 工作区有未提交改动，原位施工；只改本机制相关文件，不清理既有改动，不把原有未提交代码整体纳入新 commit。
- 每个行为先写失败测试并确认失败原因，再改实现；每个阶段跑相关 Go / Vitest / TypeScript 检查。
- AI Elements 负责 Chat UI，assistant-ui 只负责 Runtime 适配；不新增裸色、解释文案或截图测试。
- 服务重启时 `running` 且无安全检查点的 Run 安全中断；`queued` 重新入队；`waiting_approval` 有检查点才允许批准续跑。
- 数据库状态而非 WebSocket 或 assistant-ui `isRunning` 决定 Run 是否可继续发送。

## 文件与职责

- `internal/studio/domain/{model,repository}.go`：Run 状态、仓储端口和幂等请求字段。
- `internal/studio/infrastructure/persistence/gorm_repository.go`：原子事件序号、分页回放、活跃 Run 约束和 Eino 检查点持久化。
- `internal/studio/application/{executor,event_stream,runner,approval,service}.go`：事件提交、轻量通知、后台恢复、审批命令和发送幂等。
- `internal/studio/infrastructure/einoagent/engine.go` 及工具适配器：真正的审批中断、Resume、工具幂等。
- `internal/httpapi/studio/{agui,handler}.go`：AG-UI 按库补发、REST 状态与取消命令。
- `web/admin/src/lib/{studio-run-connection,agui-websocket-agent}.ts`：游标、重连与新 Run 请求幂等键。
- `web/admin/src/features/studio/{studio-chat,studio-workspace,studio-sidebar}.tsx`：不卸载活跃 Chat，以服务端状态驱动操作。

### Task 1：让事件序号由仓储原子分配

**Files:**
- Modify: `internal/studio/domain/repository.go`
- Modify: `internal/studio/infrastructure/persistence/gorm_repository.go`
- Modify: `internal/studio/application/executor.go`
- Modify: `internal/studio/application/approval.go`
- Test: `internal/studio/infrastructure/persistence/gorm_repository_test.go`
- Test: `internal/studio/application/executor_stream_test.go`

**Interfaces:** `AppendRunEvent(ctx context.Context, event *domain.Event) (*domain.Event, error)` 分配并返回事件序号；`ListEventsAfter(ctx, accountID, runID string, after uint64, limit int)` 保持现有分页签名。

- [x] **Step 1: 写失败测试。** 向同一 Run 写入 385 条事件，再由审批服务和执行器各追加一条，断言查询结果为 387 条、序号严格连续；并发写入同一 Run 时也不得覆盖。测试期待实际事件内容，而非只检查方法调用次数。
- [x] **Step 2: 运行红灯。** `go test ./internal/studio/infrastructure/persistence ./internal/studio/application -run 'EventSequence|ApprovalSequence' -count=1`；预期在第 201 条之后发现序号重复或事件缺失。
- [x] **Step 3: 实施最小修复。** 在事务中以数据库真实最大值分配下一序号并写入；冲突返回错误。执行器和审批服务不再自行读取首 200 条。调用形式固定为：

```go
stored, err := repo.AppendRunEvent(ctx, &domain.Event{RunID: run.ID, Type: eventType, Payload: raw})
if err != nil { return err }
publish(stored.Sequence)
```

- [x] **Step 4: 运行绿灯与回归。** 执行本 Task 的定向测试，再执行 `go test ./internal/studio/application ./internal/studio/infrastructure/persistence ./internal/httpapi/studio -count=1`。

### Task 2：把所有可见增量纳入持久事件日志

**Files:**
- Modify: `internal/studio/application/executor.go`
- Modify: `internal/studio/application/event_stream.go`
- Modify: `internal/studio/infrastructure/einoagent/engine.go`
- Test: `internal/studio/application/executor_stream_test.go`
- Test: `internal/studio/application/event_stream_test.go`

**Interfaces:** `executionWriter.FlushOutput(ctx)` 在边界强制提交待发文本；`EventHub.Publish` 只广播最新已提交序号，不保存完整事件历史。

- [ ] **Step 1: 写失败测试。** 模型分三次输出文本和推理；断开连接后只用数据库事件重建完整内容。再用慢订阅者填满通知缓冲，断言 Agent 提交不被阻塞。
- [ ] **Step 2: 运行红灯。** `go test ./internal/studio/application -run 'DurableStream|SlowSubscriber' -count=1`；预期当前文本仅在内存事件流中，或慢订阅者阻塞。
- [ ] **Step 3: 合批持久化。** 对同一消息连续 delta 合并成一条 `TEXT_MESSAGE_CONTENT` / `REASONING_MESSAGE_CONTENT` 事件；在短时间窗口、字节阈值和 start/end/tool/approval/terminal 边界刷新。只有持久化成功的批次可通知客户端：

```go
func (w *executionWriter) FlushOutput(ctx context.Context) error {
    if w.pendingText == "" { return nil }
    delta := w.pendingText
    w.pendingText = ""
    return w.Emit(ctx, EventTextMessageContent, map[string]any{"message_id": w.messageID, "delta": delta})
}
```

- [ ] **Step 4: 移除 EventHub 历史持有及同步背压。** 通知采用有界、非阻塞发送；丢通知时 AG-UI 定时从数据库补读。
- [ ] **Step 5: 运行绿灯与竞态检查。** `go test -race ./internal/studio/application -run 'DurableStream|SlowSubscriber' -count=1`，再运行包级测试。

### Task 3：AG-UI 只从数据库按序补发

**Files:**
- Modify: `internal/httpapi/studio/agui.go`
- Test: `internal/httpapi/studio/handler_test.go`

**Interfaces:** attach 请求继续使用 `attachRunId`、`afterSequence`；输出事件携带持久 `sequence`，协议框架事件不参与 Run 游标。

- [ ] **Step 1: 写失败测试。** 覆盖第 200 条之后补发、补发期间有新事件、空游标从首条读取、已终态 Run 正确结束、等待批准返回唯一可操作中断。断言输出序号有序且无重复或缺口。
- [ ] **Step 2: 运行红灯。** `go test ./internal/httpapi/studio -run 'AGUIAttach|AGUIReplay' -count=1`；确认当前快照/内存历史竞态导致断言失败。
- [ ] **Step 3: 替换补发循环。** 建立通知订阅后用 `ListEventsAfter` 分页读取，直到没有后续事件；收到通知或轮询 tick 时再次读取。删除 `emitRunProgressSnapshot` 的补发职责和由首 200 条推断 attach 游标的分支：

```go
for {
    page, err := repo.ListEventsAfter(ctx, accountID, runID, cursor, 200)
    if err != nil { return err }
    for _, event := range page { cursor = event.Sequence; emit(event.Sequence, mapEvent(event)) }
    if len(page) == 200 { continue }
    if isTerminal(runID) { return emitTerminal(runID) }
    waitForNoticeOrTick(ctx)
}
```

- [ ] **Step 4: 运行绿灯。** 定向测试与 `go test ./internal/httpapi/studio -count=1`。

### Task 4：审批使用持久 Eino 检查点，旧 Run 安全迁移

**Files:**
- Create: `internal/studio/infrastructure/persistence/checkpoint_store.go`
- Modify: `internal/studio/infrastructure/persistence/gorm_repository.go`
- Modify: `internal/studio/infrastructure/einoagent/engine.go`
- Modify: `internal/studio/infrastructure/studiotool/runtime.go`
- Modify: `internal/studio/infrastructure/workflowtool/runtime.go`
- Modify: `internal/studio/application/approval.go`
- Modify: `internal/studio/application/executor.go`
- Modify: `internal/studio/application/runner.go`
- Test: `internal/studio/infrastructure/einoagent/engine_stream_test.go`
- Test: `internal/studio/infrastructure/persistence/gorm_repository_test.go`

**Interfaces:** 持久适配器实现 `adk.CheckPointStore` 的 `Get(ctx, checkpointID)`、`Set(ctx, checkpointID, []byte)`；Agent 新执行调用 `Run(..., adk.WithCheckPointID(run.ID))`，批准后调用 `Resume(ctx, run.ID)`。

- [ ] **Step 1: 写失败测试。** 让 Agent 在已执行一个有副作用工具后等待批准；批准后断言先前工具只执行一次，待批准工具执行一次，检查点从新仓储实例读取仍可续跑。
- [ ] **Step 2: 运行红灯。** `go test ./internal/studio/infrastructure/einoagent ./internal/studio/infrastructure/persistence -run 'ApprovalCheckpoint|ResumeAfterApproval' -count=1`；确认当前从头执行或缺少检查点。
- [ ] **Step 3: 接入 Eino 中断。** 工具在副作用前调用 `compose.Interrupt(ctx, approvalInfo)`；Agent 把 Eino interrupt 结果映射为 Run 等待批准。检查点保存成功后才对外发布等待状态；批准后 `Resume`，拒绝转取消。适配器使用同一个持久 Store：

```go
runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent, EnableStreaming: true, CheckPointStore: store})
if resuming { iterator, err = runner.Resume(ctx, run.ID) } else {
    iterator = runner.Run(ctx, initialMessages, adk.WithCheckPointID(run.ID))
}
```
- [ ] **Step 4: 增加幂等防线。** 内建资产、Flow 节点和工作流启动以 `(runID, stableActionID)` 查找已成功结果；重复调用返回原结果，不生成新资产或提交第二个任务。
- [ ] **Step 5: 兼容旧版等待 Run。** 缺少 checkpoint 时不重放；标记为可诊断的中断失败，保留事件和审批记录。运行定向测试及相关包级测试。

### Task 5：Run 命令幂等和重启恢复

**Files:**
- Modify: `internal/studio/domain/model.go`
- Modify: `internal/studio/domain/repository.go`
- Modify: `internal/studio/infrastructure/persistence/gorm_repository.go`
- Modify: `internal/studio/application/service.go`
- Modify: `internal/studio/application/runner.go`
- Modify: `internal/studio/application/approval.go`
- Modify: `internal/httpapi/studio/agui.go`
- Modify: `internal/httpapi/studio/handler.go`
- Test: `internal/studio/application/runner_test.go`
- Test: `internal/httpapi/studio/handler_test.go`

**Interfaces:** 发送请求带 `requestId`，同账号同 Session 同 ID 返回原 Run；不同请求遇活跃 Run 返回状态冲突。取消继续通过既有 `POST /runs/{runID}/cancel`。

- [ ] **Step 1: 写失败测试。** 重复发送相同请求 ID 只产生一条消息和一个 Run；不同 ID 遇活跃 Run 被拒；取消后后台 context 终止；重启后 queued 重新入队、running 标为中断，超过 200 个待处理 Run 仍全部处理。
- [ ] **Step 2: 运行红灯。** `go test ./internal/studio/application ./internal/httpapi/studio -run 'IdempotentSend|RecoverRuns|CancelRun' -count=1`。
- [ ] **Step 3: 最小实现。** 将请求 ID 与活跃 Run 约束放在仓储事务中；`Recover` 分页扫描，不再自动执行旧 `running`；终态写入和取消事件保持固定顺序：

```go
if previous, err := repo.GetRunByRequestID(ctx, accountID, sessionID, requestID); err == nil {
    return previous, nil
}
if active, err := repo.GetActiveRun(ctx, accountID, sessionID); err == nil {
    return nil, fmt.Errorf("%w: session has active run %s", domain.ErrInvalidTransition, active.ID)
}
```
- [ ] **Step 4: 运行绿灯。** 定向测试与 Studio 全部 Go 测试。

### Task 6：前端运行状态与连接生命周期

**Files:**
- Modify: `web/admin/src/lib/studio-run-connection.ts`
- Modify: `web/admin/src/lib/agui-websocket-agent.ts`
- Modify: `web/admin/src/lib/api/studio.ts`
- Modify: `web/admin/src/features/studio/studio-chat.tsx`
- Modify: `web/admin/src/features/studio/studio-workspace.tsx`
- Modify: `web/admin/src/features/studio/studio-sidebar.tsx`
- Test: `web/admin/src/lib/studio-run-connection.test.ts`
- Test: `web/admin/src/features/studio/studio-workspace.contract.test.ts`

**Interfaces:** 连接控制器保留 `studioRunId` 和 `lastSequence`；断线重连只带 `attachRunId`、游标和空 `messages`；停止调用服务端取消 API。

- [ ] **Step 1: 写失败测试。** 模拟 socket 在两段正文间断开、详情重新获取、批准中断和停止按钮；断言同一 Run 接续、内容不重复、Chat 不卸载、停止会调用后端取消。
- [ ] **Step 2: 运行红灯。** 在 `web/admin` 执行 `pnpm exec vitest run --config vite.config.ts --browser.headless src/lib/studio-run-connection.test.ts src/features/studio/studio-workspace.contract.test.ts`。
- [ ] **Step 3: 重构连接控制器。** 区分传输断开与服务端终态，断开后指数退避重连；每条持久事件更新游标并去重；接入失败时查询权威 Run 状态。新消息使用稳定请求 ID。连接状态只由服务端终态事件结束：

```ts
if (event.sequence <= lastSequence) return
lastSequence = event.sequence
aggregator.handle(event)
// socket close: reconnect with attachRunId and lastSequence; do not yield complete
```
- [ ] **Step 4: 保留 Chat 实例。** `detail.isFetching` 不再替换活跃 Chat；审批中断与成功/失败分别触发对账；取消走后端。保留既有 AI Elements 组件及语义类名。
- [ ] **Step 5: 运行绿灯。** 定向 Vitest、`pnpm exec tsc -b --noEmit` 和相关前端测试。

### Task 7：跨层验收与回归

**Files:**
- Test: `internal/httpapi/studio/handler_test.go`
- Test: `web/admin/src/lib/studio-run-connection.test.ts`
- Modify: `docs/superpowers/specs/2026-09-23-studio-run-continuity-repair-design.md`（仅记录经验证的实施偏差）。

- [ ] **Step 1: 补跨层测试。** 使用真实 GORM 仓储、后台 Runner 与 AG-UI handler，模拟断开再接入、批准、取消和进程重启；前端用可控 WebSocket 模拟同一事件流。新增断言必须对应前面任务未覆盖的跨层故障，并先观察失败，再补最小遗漏。
- [ ] **Step 2: 运行跨层测试。** Go：`go test ./internal/studio/... ./internal/httpapi/studio -count=1`；前端：`pnpm exec vitest run --config vite.config.ts --browser.headless src/lib/studio-run-connection.test.ts`。
- [ ] **Step 3: 全量验证。** `go test ./internal/studio/... ./internal/httpapi/studio -count=1`，`cd web/admin && pnpm exec tsc -b --noEmit && pnpm exec vitest run --config vite.config.ts --browser.headless`；执行 `git diff --check` 并核对已有未提交改动未丢失。
- [ ] **Step 4: 手动验收。** 在 Studio 中间 Chat 分别验证切页、刷新、关闭重开、审批、停止、网络断开恢复与服务重启；记录无法由自动测试覆盖的结果或环境限制。
