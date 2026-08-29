---
change: topic-delete-cleanup
role: technical-design
canonical_spec: openspec
---

# 任务队列（Topic）删除：确认制清理式删除 深度技术设计

## 1. 目标

把 Topic 删除从「被引用就 409 拒绝 + 英文内联提示」对齐为与工作流/节点/消息平台一致的「**先展示真实引用与影响，二次确认，确认后自动清理**」：

- 删除弹窗实时展示：N 个工作流规则指向它、M 个计算节点订阅它、K 个排队任务将标记失败。
- 确认后在同一事务内：整条删除指向该 Topic 的工作流路由规则、从节点 `subscribe_topics` 移除该 Topic、把排队（queued）任务终态标记失败（`topic_deleted`，原因「任务队列已删除」）并通知发起用户、删除 Topic 行。
- 错误提示改走 toast + 后端 `code` 映射成中文，不再出现英文 header 提示。

## 2. 现状与问题

### 引用形态

- **工作流路由规则**：`CaseDocument.Routing.Rules[].topic`（存于 case 的 doc_json）。规则按顺序第一条命中生效，未命中回退 default。`ValidateRouting` 在保存工作流时校验规则指向的 Topic 必须存在且启用——因此删除 Topic 后不能留悬空引用。
- **计算节点订阅**：节点 `subscribe_topics` 数组。
- **排队任务**：`tasks.dispatch_topic` 指向该 Topic、状态 queued 的任务，在被删除 Topic 的所有订阅节点移除后无人可领取。

### 问题

当前 `internal/httpapi/topics/handler.go` 的 `delete`：

- `CountCaseRefs`（doc_json LIKE）+ `CountEdgeRefs`（subscribe_topics_json LIKE）任一 > 0 直接返回 409，英文硬编码 `topic is referenced by cases or edges`，无 code、无清单、无确认/清理。
- 前端 `topic-detail-panel.tsx` 用内联 `ErrorBanner` 展示 `deleteMutation.error` 原文（页面内容顶部），不是 toast，且无本地化。

## 3. 技术决策

### 3.1 `topicadmin` 清理服务

新建 `internal/topicadmin`（仿 caseadmin/edgeadmin），签名：

```go
var ErrNeedsAck = errors.New("topic delete needs ack for references")

type DeleteSummary struct {
	RemovedCaseRules      int `json:"removed_case_rules"`
	RemovedEdgeSubs       int `json:"removed_edge_subscriptions"`
	FailedTasks           int `json:"failed_tasks"`
}

func DeleteTopic(ctx context.Context, key string, ack bool) (DeleteSummary, error)
```

在单个 `gdb.Transaction` 内：

1. `topicRepo.Get` 不存在 → `topic.ErrTopicNotFound`；`key == topic.DefaultKey` → `ErrDefaultProtected`（409，保留现状）。
2. 收集引用（读全部 cases / edges，Go 侧解析，不用 LIKE）：
   - cases：`routing.rules` 中 `topic == key` 的规则数与所属 case。
   - edges：`subscribe_topics` 包含 key 的节点数与列表。
3. 引用数 > 0 且未带 `ack` → `ErrNeedsAck`（整体回滚）。
4. 清理：
   - **case 规则整条删除**（用户已确认 A）：对每个受影响 case，`routing.rules` 过滤掉 `topic == key` 的规则，保存 doc_json。
   - **节点订阅**：对每个受影响节点，`subscribe_topics` 移除 key（复用 `EdgeRepository.UpdateSubscribeTopics` 或直接改写 doc_json）。
   - **排队任务**：`taskRepo.ListByTopic(ctx, key, ListByTopicQuery{Status: queued})`，逐个 `MarkFailed(topic_deleted, "任务队列已删除", now)` + Update（跳过已终态），收集通知。
5. `topicRepo.Delete`。

commit 后 best-effort：对失败任务发布 `task_failed` 通知（ChatID 为空时经 session 回查）。

running 任务不受影响（已被节点领取，节点仍能上报）；pending 任务在规则删除后重新解析路由（回退 default）。

### 3.2 topics handler 与 main.go 接线

- `topics.Handler` 删除 `CountCaseRefs` / `CountEdgeRefs` 字段，新增 `DeleteWithCleanup func(ctx, key string, ack bool) (topicadmin.DeleteSummary, error)`。
- `delete` 方法：`ErrDefaultProtected` → 409（code `topic_default_protected`）→ `ErrNeedsAck` → 409（code `topic_delete_needs_ack`）→ `ErrTopicNotFound` → 404 → 成功返回摘要。
- main.go：`topicDeleteSvc := topicadmin.NewService(gdb, botRT.Notify)`，`DeleteWithCleanup: topicDeleteSvc.DeleteTopic`。
- `sharedkernel` 新增：`TaskErrorTopicDeleted = "topic_deleted"`、`TopicDeletedMessage = "任务队列已删除"`。

### 3.3 tasks 管理 API 支持 dispatch_topic 过滤

删除弹窗需要展示「K 个排队任务」：`GET /api/v1/tasks?dispatch_topic=X&status=queued`。`TaskRow.dispatch_topic` 已有列与索引，仿 channel_id 加过滤即可。前端 `listTasks` 参数加 `dispatch_topic`。

### 3.4 前端删除弹窗（toast 化）

- `topic-detail-panel.tsx`：
  - `deleteMutation` 增加 `onError` toast，错误经 `localized-errors` 风格映射（code → i18n），移除内联 `ErrorBanner`。
  - 删除弹窗打开时实时拉取：引用清单复用 link-health `topicReferences`（cases + edges，已有）+ queued 任务数（`listTasks({ dispatch_topic, status: 'queued' })`）。
  - 有影响时展示「N 个工作流规则将删除、M 个节点订阅将移除、K 个排队任务将标记失败并通知用户」+「我已知悉」勾选；确认调 `deleteTopic(key, ack)`。
  - 成功 toast 展示摘要。
- `deleteTopic(id, ack?)` 携带 `ack_references`。

### 3.5 i18n

新增 keys（zh/en 成对，登记进 `locale.test.ts`）：

- `topics.deleteWillRemoveRules`：N 个工作流路由规则将被删除
- `topics.deleteWillUnbindNodes`：M 个计算节点的订阅将被移除
- `topics.deleteWillFailQueued`：K 个排队任务将标记为失败并通知用户
- `topics.deleteAckImpact`：我已知悉上述影响
- `topics.deleteNeedsAck`：该任务队列仍被工作流或节点引用，确认后将自动清理
- `topics.deleteDone`：任务队列已删除（移除 N 条规则、解除 M 个订阅、失败 K 个任务）

## 4. 数据流

```text
Topic 详情页删除按钮
  → 实时拉取 topicReferences（工作流/节点）+ queued 任务数
  → 弹窗展示影响 + 勾选确认
  → DELETE /api/v1/topics/{key} {ack_references: true}
  → topicadmin.DeleteTopic（事务：删规则 → 解订阅 → queued 终态失败 → 删 Topic 行）
  → commit 后 task_failed 通知
  → 200 摘要 → toast
```

## 5. 契约

| 项 | 值 |
|---|---|
| 删除 | `DELETE /api/v1/topics/{key}`，body `{"ack_references": true}`；成功 200 `{"deleted": true, "removed_case_rules": N, "removed_edge_subscriptions": M, "failed_tasks": K}` |
| 有引用未确认 | 409 `{"error": "...", "code": "topic_delete_needs_ack"}` |
| default topic | 409 `{"error": "...", "code": "topic_default_protected"}` |
| 不存在 | 404 |
| tasks 列表 | 新增 `?dispatch_topic=` 过滤 |

## 6. 边界与异常

- 事务失败整体回滚，返回 500，前端提示「删除失败，请重试」。
- 通知失败记日志，不阻断。
- 删除后工作流再保存：规则已清理，`ValidateRouting` 不再报未知 Topic。
- 删除期间新任务进来：入口工作流规则已删，pending 解析回退 default，不会路由到已删 Topic。
- running 任务照常完成；历史任务行保留。
- default Topic 始终不可删除（保留现状保护）。

## 7. 测试

- `topicadmin` 单测：无引用直删；有引用未 ack 回滚；有引用 ack 后规则删除/订阅移除/queued 失败/行删除/摘要；default 保护；404。
- topics handler 测试：409 code、404、200 摘要。
- tasks handler 测试：`dispatch_topic` 过滤。
- 前端 contract 测试：弹窗影响文案 + toast 化（不再出现内联 ErrorBanner）+ i18n 成对。

## 8. 不做的事（YAGNI）

- 不重定向规则到 default（用户已选整条删除）。
- 不改 pending 任务（规则删除后自然重新解析）。
- 不清理任务历史。
