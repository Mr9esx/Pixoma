---
change: case-delete-cleanup
role: technical-design
canonical_spec: openspec
---

# Case 删除改为清理式删除 深度技术设计

## 1. 目标

删除 Case（工作流）不再设置任何前置阻止条件。用户在弹窗中确认后，系统在同一笔事务内自动完成清理并删除：

- 解除菜单/卡片引用：把被删工作流 ID 从 `open_workflow.WorkflowIDs` 中移除；列表清空则删除整个按钮/菜单项。
- pending 任务终态失败：错误码 `case_deleted`，失败原因「工作流已删除」，**不重试**，并走现有任务失败通知告知发起用户。
- 活跃会话（collecting/confirming）终止：标记为 `exited`，并给会话中的用户发送一条提示消息。
- 删除 case 行；任务历史行保留（现状即如此）。

running / queued 任务不干预：它们已持有 `job_ref` 快照，不依赖 case 行，自然跑完并正常回写状态。

## 2. 现状与问题

当前 `internal/httpapi/cases/handler.go` 的 `delete` 依次检查四个条件，任一命中返回 409：

| 条件 | 问题 |
|---|---|
| `enabled` 必须先停用 | `enabled` 只是「能否触发新运行」的开关（`botapp/facade.go`、`confirm_run.go` 拦截）；删除本身不依赖它，先停用是多余一步 |
| 被菜单/卡片引用 | 引用不是外键，而是菜单/卡片 JSON 中 `open_workflow` 动作的 `workflow_ids`；可以自动解除，但改的是线上菜单/卡片，需要用户明示确认 |
| 有活跃会话 | 会话只有 `exited`（用户主动退出）终态；删除后用户下一步会撞 404，且残留活跃态会锁住该聊天下次开新会话 |
| 有 pending/queued/running 任务 | 任务行不随 case 删除，且 running/queued 已有 `job_ref` 快照，可脱离 case 行运行；当前一刀切阻止导致「一直有任务就永远删不掉」 |

另外两个技术细节：

- 任务失败事件默认走 `RequeueAfterFailure`（有界重试）；删除场景必须绕过重试，直接终态失败。
- 调度循环遇到「case 不存在」目前是无限重试（`resolveTopic` 失败后保持 pending 等下一轮），需要防御性改为终态失败，避免僵尸任务。

## 3. 技术决策

### 3.1 删除保护全部移除，改为确认制

- `DELETE /api/v1/cases/{id}` 不再因 enabled/引用/会话/任务而 409。
- 请求体支持 `{"ack_references": true}`。当存在引用且未带确认时返回 409（错误码 `case_delete_needs_ack`），提示「该工作流仍被 N 处菜单/卡片引用，请确认删除后将自动移除」。脚本误调也有保护。
- 删除成功返回 200 摘要：`{deleted: true, removed_placements: [...], failed_tasks: N, terminated_sessions: N}`。

### 3.2 后端新增删除清理服务

新建应用层服务（如 `internal/catalog/application/case_delete.go`），签名：

```go
type DeleteSummary struct {
	RemovedPlacements  []mcpersist.WorkflowPlacement
	FailedTasks        int
	TerminatedSessions int
}

func DeleteCaseWithCleanup(ctx context.Context, id sharedkernel.CaseID, ack bool) (DeleteSummary, error)
```

在单个 `gdb.Transaction` 内执行四步，任一失败整体回滚：

1. **失败 pending 任务**：`WHERE case_id = ? AND status = 'pending'`，逐个 `MarkFailed("case_deleted", "工作流已删除", now)` + Update。不经过 `OnStatus` 的失败分支（避免重试）。queued/running 不触碰。
2. **终止活跃会话**：`WHERE case_id = ? AND status IN ('collecting','confirming')`，逐个 `Exit(now)` + Save，收集 ChatID 列表供 commit 后通知。
3. **解除引用**：扫描全部菜单/卡片 JSON，`open_workflow` 动作中移除该 `workflow_id`；`WorkflowIDs` 清空则删除该项/按钮；保存有变更的菜单/卡片，收集移除清单。
4. **删除 case 行**（复用现有 `CaseRepository.Delete`）。

### 3.3 通知（commit 之后，best-effort）

- 被终止的会话：通过 `NotifyRouter` 发布 `UserNotify{Kind: "session_terminated", ErrorMsg: "该工作流已被管理员删除，当前会话已结束。"}`；TG adapter 的 `HandleUserNotify` 增加该 kind 的干净文案（现有逻辑只认识任务成功/失败，会拼出「任务 x: ...」）。
- 被失败的任务：复用 orchestrator `publishNotify` 等价逻辑（`Kind: "task_failed"` + `ErrorMsg`），让发起用户看到失败原因。
- 通知失败只记日志，不影响删除结果。

### 3.4 前端删除弹窗

- 打开弹窗时拉取现有 `/menu-placements` 接口，展示引用清单（频道/菜单项/按钮名）。
- 弹窗内容：删除确认 + 引用影响面（「将自动移除以下 N 处引用」）+ 影响摘要（「X 个排队任务将标记为失败，Y 个进行中会话将被终止」）。
- 「我已知悉引用将被移除」checkbox 勾选后确认按钮才可点；确认时 `DELETE` 带 `ack_references: true`。
- 删除成功 toast 展示摘要；失败展示本地化错误。

### 3.5 防御性修复

调度循环 `resolveTopic` 遇到 `case not found` 时，直接把该 pending 任务终态失败（`case_deleted`），而不是保留 pending 无限重试。

## 4. 数据流

```text
前端弹窗（引用清单 + 影响摘要 + 我已知悉）
  → DELETE /api/v1/cases/{id} {ack_references: true}
  → DeleteCaseWithCleanup（gorm.Transaction）
      1. pending 任务终态失败（case_deleted，不重试）
      2. 活跃会话 Exit + 收集 ChatID
      3. 解除菜单/卡片引用 + 收集移除清单
      4. 删除 case 行
  → commit 后：session_terminated / task_failed 通知（best-effort）
  → 200 摘要 → toast
```

## 5. 契约

### API

| 项 | 值 |
|---|---|
| 请求 | `DELETE /api/v1/cases/{id}`，body `{"ack_references": true}` |
| 成功 | 200 `{"deleted": true, "removed_placements": [...], "failed_tasks": N, "terminated_sessions": N}` |
| 有引用未确认 | 409 `{"error": "...", "code": "case_delete_needs_ack"}` |
| case 不存在 | 404 `{"error": "case not found"}` |

### 事件

`UserNotify.Kind` 新增 `session_terminated`；TG adapter 渲染固定文案，不再按任务模板拼接。

### i18n

删除弹窗新增 keys（zh/en 成对，沿用现有 locale 合同测试）：

- `cases.deleteWillRemoveRefs`：将自动移除以下引用
- `cases.deleteWillFailTasks`：X 个排队任务将标记为失败
- `cases.deleteWillEndSessions`：Y 个进行中会话将被终止
- `cases.deleteAckRefs`：我已知悉引用将被移除
- `cases.deleteNeedsAck`：该工作流仍被引用，请确认后将自动移除
- `cases.sessionTerminated`：该工作流已被管理员删除，当前会话已结束

上一轮加的四个 `deleteConflict*` 文案随本改动删除（不再有对应 409 分支）。

## 6. 边界与异常

- 事务失败：整体回滚，返回 500，前端提示「删除失败，请重试」。
- 通知失败：记日志，不阻断。
- 删除与调度并发：事务内对 pending 的终态失败与调度循环冲突时，以事务结果为准；调度侧另有 case 不存在即失败的防御修复。
- 删除期间新任务进来：入口已删，`StartCase`/`ConfirmRun` 取不到 case 直接失败，不会出现「一直有任务导致永远删不掉」。
- running 任务在删除后完成：任务行保留，正常回写 succeeded，历史可查。

## 7. 测试

- 后端 handler/服务测试：无引用直删；有引用未 ack 409；有引用 ack 后删除并返回摘要；pending 终态失败且不重试；queued/running 不受影响；会话终止并触发通知；事务失败回滚。
- 前端 contract 测试：弹窗展示引用清单；checkbox 门控；删除调用携带 ack。
- i18n locale 成对测试。

## 8. 不做的事（YAGNI）

- Comfy interrupt：running 任务不打断，自然跑完。
- 软删除/归档：本设计仍是硬删除 + 清理。
- 会话新增失败状态：复用 `exited` 终态。
- 引用改为用户手动逐个处理：确认后自动解除。
