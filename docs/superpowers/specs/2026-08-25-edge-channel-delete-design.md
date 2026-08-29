---
change: edge-channel-delete
role: technical-design
canonical_spec: openspec
---

# 计算节点与消息平台删除：确认制清理式删除 深度技术设计

## 1. 目标

对齐工作流（Case）删除的交互模式，把计算节点（Edge）与消息平台（Channel）的删除改为「**先展示真实状态，再二次确认，确认后自动清理**」：

- **计算节点**：补充前端删除入口；删除时把该节点 running 任务终态标记失败（原因「节点已删除」）并通知发起用户；有任务时需用户勾选确认。
- **消息平台**：删除不再要求先停用；删除时在同一事务内删除消息平台行、终止该消息平台活跃会话（标记 exited）并通知会话用户、清理该消息平台菜单/卡片行；弹窗先展示该消息平台进行中会话数与执行中/排队任务数，有影响时需勾选确认。
- 两者的确认弹窗都必须**先拉取当前任务/会话状态展示给用户**，再要求二次确认，而不是删完才报。

## 2. 现状与问题

### 计算节点

当前 `DELETE /api/v1/edges/{id}`（`internal/httpapi/edges/handler.go`）**没有任何限制与善后**：直接删行 + 刷新调度池。前端删除按钮只藏在「编辑」弹窗内，列表页与详情页头部无入口。

删除的实际连带影响：

| 影响 | 说明 |
|---|---|
| agent token 立即失效 | 所有 agent 接口（claim/心跳/状态/presence）依赖节点行中的 token 鉴权（`VerifyAgentToken`），行删除后 agent 立刻 401 |
| running 任务 | 节点正在执行的任务无法上报结果；不处理会等 90s 租约过期后重排，可能被其他节点重复执行 |
| queued/pending 任务 | pending 未绑定节点；queued 可由其他订阅同 topic 的节点继续 claim，无需处理 |
| presence/metrics | 内存 presence 记录无 Remove 方法（小残留）；metrics 历史行保留 |
| 订阅 topics | 节点删除后其订阅的 topic 可能没有消费者，queued 任务无人 claim |

### 消息平台

当前 `channelapp.Service.Delete`（`internal/channel/application/service.go`）要求：

1. `Enabled == false` 才能删（前端删除按钮在启用时直接禁用）；
2. `HasActiveRefs == false`（只统计 collecting/confirming 会话）。

问题：

- 「必须先停用」对用户不友好，删除一个启用中的消息平台本可以靠确认弹窗完成。
- 错误文案写「active sessions/tasks」，实际只查会话；执行中/排队任务（会话已 submitted）并不会被统计，与文案不符。
- 删除消息平台不清理 `channel_main_menus` / `channel_cards` 孤儿数据。
- 活跃会话未终止，用户会被悬挂在流程中。

## 3. 技术决策

### 3.1 计算节点：`edgeadmin` 清理服务

新建 `internal/edgeadmin`（仿 `internal/caseadmin`），签名：

```go
var ErrNeedsAck = errors.New("edge delete needs ack for running tasks")

type DeleteSummary struct {
	FailedTasks int `json:"failed_tasks"`
}

func DeleteEdge(ctx context.Context, id sharedkernel.EdgeID, ack bool) (DeleteSummary, error)
```

在单个 `gdb.Transaction` 内：

1. `edgeRepo.Get` 不存在 → `edge.ErrNotFound`。
2. 列出该节点 running 任务（`taskRepo.List(AdminListQuery{EdgeID: id, Status: running})`）。
3. 若 running 任务数 > 0 且未带 `ack` → 返回 `ErrNeedsAck`（整体回滚）。
4. 逐个 `MarkFailed(edge_deleted, "节点已删除", now)` + Update（跳过已终态任务，不重试），收集失败任务的通知（ChatID 为空时经 session 回查，仿 caseadmin）。
5. 删除节点行。

commit 之后 best-effort：

- 发布 `task_failed` 通知（`Kind: "task_failed"`，`ErrorMsg: "节点已删除"`）。
- 清理 presence：`presence.Store` 新增 `Remove(id)`，由 edges handler 在删除成功后调用。

queued/pending 任务不处理；metrics 历史保留。

### 3.2 edges handler 与 main.go 接线

- `edges.Handler` 新增注入字段 `DeleteWithCleanup func(ctx, id sharedkernel.EdgeID, ack bool) (edgeadmin.DeleteSummary, error)`。
- `delete` 方法改为：调 `DeleteWithCleanup` → `ErrNeedsAck` 返回 409（code `edge_delete_needs_ack`）→ `edge.ErrNotFound` 返回 404 → 成功后续 `refreshPool` + `h.Presence.Remove(id)` → 返回 `{"deleted": true, "failed_tasks": N}`。
- main.go：`edgeDeleteSvc := edgeadmin.NewService(gdb, botRT.Notify)`，`DeleteWithCleanup: edgeDeleteSvc.DeleteEdge`。
- `sharedkernel` 新增常量：`TaskErrorEdgeDeleted = "edge_deleted"`、`EdgeDeletedMessage = "节点已删除"`。

### 3.3 消息平台：直接删除 + 事务清理

`channelapp.Service` 变更：

- 删除 `Enabled` 检查与 `HasActiveRefs` 字段。
- 新增字段：

```go
Notify notify.Publisher
// DeleteWithCleanup deletes the channel row, terminates its active sessions
// (collecting/confirming → exited) and removes channel-scoped menu/card rows
// in one transaction; returns chats to notify after commit.
DeleteWithCleanup func(ctx context.Context, channelID string) ([]sharedkernel.ChatID, error)
```

- `Delete` 变为：`Store.Get`（404）→ 调 `DeleteWithCleanup` → commit 后对返回的 chat 发布 `UserNotify{Kind: "session_terminated", ErrorMsg: "该消息平台已被管理员删除，当前会话已结束。"}`（best-effort）。
- 没有 `DeleteWithCleanup` 时退化为 `Store.Delete`（兼容测试）。

main.go 中 `DeleteWithCleanup` 闭包用 `gdb.Transaction`：

1. 删除 `channels` 行；
2. `UPDATE sessions SET status='exited', updated_at=? WHERE channel_id=? AND status IN ('collecting','confirming')`，并查出受影响 chat 列表；
3. 删除 `channel_main_menus`、`channel_cards` 中 `channel_id = ?` 的行。

channels HTTP handler 的 `Delete` 删除 409 分支，保留 404/500/200。

### 3.4 删除前状态统计接口

确认弹窗需要删除前的实时数字：

- **节点 running 任务数**：前端已有 `listEdgeTasks(id, {status:'running'})`，无需新接口。
- **消息平台进行中会话数**：sessions 管理 API 增加 `channel_id` 过滤（`SessionRow` 已有该列，仿 case_id 实现）。
- **消息平台执行中/排队任务数**：tasks 管理 API 增加 `channel_id` 过滤；`TaskRow` 无 channel 列，通过 `JOIN sessions ON sessions.id = tasks.session_id` 过滤 `sessions.channel_id`。
- 前端 `listSessions` / `listTasks` 参数增加 `channel_id`。

### 3.5 前端弹窗

#### 节点详情页

- 详情页头部（编辑按钮旁）新增「删除」按钮；编辑弹窗（`EdgeForm`）里的删除按钮移除，收敛为单一入口。
- 删除弹窗内容（打开时实时拉取）：
  - 「N 个执行中任务将标记为失败并通知用户」（N=0 时不展示该行或展示“无执行中任务”）；
  - 只读展示该节点订阅的 topics；
  - N>0 时勾选「我已知悉这些任务将标记为失败」才能确认；
  - 确认调用 `deleteEdge(id, ack)`，成功 toast 展示 `failed_tasks`。
- 节点不直接拥有会话，弹窗不展示会话维度。

#### 消息平台详情页

- 删除按钮不再因启用状态禁用（停用/启用开关保留）。
- 删除弹窗内容（打开时实时拉取）：
  - 「M 个进行中会话将被终止并通知用户」；
  - 「K 个执行中/排队任务将继续运行，但消息平台删除后完成通知无法送达」；
  - M>0 或 K>0 时勾选「我已知悉」才能确认。
- 确认弹窗文案同步更新。

### 3.6 i18n

新增 keys（zh/en 成对，登记进 `locale.test.ts` 成对列表）：

- `edges.deleteWillFailRunning`：N 个执行中任务将标记为失败并通知用户
- `edges.deleteSubscribedTopics`：该节点订阅的 Topics
- `edges.deleteAckRunning`：我已知悉这些任务将标记为失败
- `channels.deleteWillEndSessions`：M 个进行中会话将被终止并通知用户
- `channels.deleteInFlightTasks`：K 个执行中/排队任务将继续运行，但完成后通知无法送达
- `channels.deleteAckImpact`：我已知悉上述影响

## 4. 数据流

```text
节点：详情页删除按钮
  → 实时拉取 running 任务数 + 订阅 topics
  → 弹窗展示 + 勾选确认
  → DELETE /api/v1/edges/{id} {ack_references: true}
  → edgeadmin.DeleteEdge（事务：running 任务终态失败 → 删节点行）
  → commit 后 task_failed 通知 + presence.Remove + refreshPool

消息平台：详情页删除按钮（不再因 enabled 禁用）
  → 实时拉取进行中会话数 + 执行中/排队任务数
  → 弹窗展示 + 勾选确认
  → DELETE /api/v1/channels/{id}
  → channelapp.Service.Delete（事务：删消息平台行 + 会话 exited + 清理菜单/卡片）
  → commit 后 session_terminated 通知
```

## 5. 契约

### API

| 项 | 值 |
|---|---|
| 节点删除 | `DELETE /api/v1/edges/{id}`，body `{"ack_references": true}`；成功 200 `{"deleted": true, "failed_tasks": N}` |
| 节点有 running 任务未确认 | 409 `{"error": "...", "code": "edge_delete_needs_ack"}` |
| 节点不存在 | 404 |
| 消息平台删除 | `DELETE /api/v1/channels/{id}`；成功 200 `{"deleted": true}`；不再有启用/会话 409 |
| 消息平台不存在 | 404 |
| sessions 列表 | 新增 `?channel_id=` 过滤 |
| tasks 列表 | 新增 `?channel_id=` 过滤（join sessions） |

### 事件

复用 `UserNotify.Kind = "session_terminated"`（TG adapter 已支持，渲染 `ErrorMsg`）；消息平台场景 ErrorMsg 为「该消息平台已被管理员删除，当前会话已结束。」。

## 6. 边界与异常

- 删除与任务上报并发：agent 鉴权依赖节点行，删除后上报被 401 拒绝，不存在状态回跳；删除前瞬时上报若已落库（终态任务）则在 `MarkFailed` 时跳过。
- 事务失败：整体回滚，返回 500，前端提示「删除失败，请重试」。
- 通知失败：记日志，不阻断。
- 消息平台删除后运行中任务：继续执行（job_ref 快照），但完成通知因消息平台不存在无法送达（路由会跳过并记日志）；弹窗已提前告知用户。
- presence 为内存态，进程重启即清空；`Remove` 只清理当前进程。
- 节点删除不停止远程 agent 进程：agent 会持续 401 刷日志，需人工停进程；弹窗文案提示。

## 7. 测试

- `edgeadmin` 单测：无 running 任务直删；有 running 未 ack 409 且回滚；有 running ack 后终态失败（`edge_deleted`）且通知；queued/pending 不受影响；节点不存在 404。
- `presence.Store.Remove` 单测。
- edges handler 测试：409 code、200 摘要、404。
- channels 服务测试：enabled 消息平台可直接删；事务清理（会话 exited、菜单/卡片删除、返回 chat）；无清理注入时退化。
- sessions/tasks handler 测试：`channel_id` 过滤。
- 前端 contract 测试：节点详情页删除入口与影响文案；消息平台删除按钮不再禁用、弹窗影响文案；i18n 成对。

## 8. 不做的事（YAGNI）

- 不停远程 agent 进程（无法从控制面保证；仅提示）。
- 不级联清理节点 metrics 历史（保留可查）。
- 消息平台删除不中断执行中/排队任务（与 workflow 一致）。
- 不新增「消息平台任务数」专用聚合接口（复用 tasks 列表的 channel_id 过滤）。
