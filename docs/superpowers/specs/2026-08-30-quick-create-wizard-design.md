---
comet_change: quick-create-wizard
role: technical-design
canonical_spec: openspec
---

# 快速新建向导 · 深度技术设计

## 1. Context

现状：`features/quick-config` 落地页以已有工作流表为主；Formity 四屏为工作流 → 选节点 → 投放 → 完成页门禁。完成页才 `createCase` / `enableCase`，并把选中节点订上 `default`。`WorkflowEditor` 新建路径含第 4 段 `TaskFlowTable`（处理流程）；`emptyCase()` 已带 `{ always: true } → default`。调度仍是规则命中 Topic、否则回退 `default`。

Open 产物与 HTML demo 已确认：入口改名、不要选已有、四屏引导、显式选队列、工作流页处理流程不拆。

## 2. Goals / Non-Goals

**Goals**
- 四屏 Formity：工作流草稿 → 队列 → 节点 → 提交启用 →「还差一步」。
- 选中队列一律写成一条 `{ when: { always: true }, topic }` 规则（含 `default`）。
- 队上已有启用订阅者则可跳过选节点；无人则必选/新建。新建复用 `CreateEdgeWizard`，通线可跳过。
- 投放不挡启用；第四屏跳转 `/channels`；未通线提醒。

**Non-Goals**
- 不改调度引擎、不独占队列、不改工作流页四段结构。
- 不新增 Case 归属节点字段。
- 不在向导里写消息平台菜单。

## 3. 深度技术决策

### D1 — 保留 Formity，替换屏组件
`quick-config-flow.tsx` 仍四 `form` + `return`。映射：
1. `Step1Workflow` — `WorkflowEditor` + `hideProcessing`
2. `Step2Queue`（新建）— 选/建 Topic
3. `Step2Node`（改前进条件）
4. `Step4Next`（替换 `DoneScreen` / `Step3Channels`）

`WizardShared` 增加 `topicKey: string | null`；删除投放相关 `pendingEntries` 提交路径（会话可不再存）。`mode` 只保留 `create`。

### D2 — hideProcessing 与 stepRail
`WorkflowEditor` 增加 `hideProcessing?: boolean`。为 true 时不渲染 `processingSection`。`stepRail` 的底端测量从 `[data-testid="case-section-processing"]` 改为输出段 `[data-testid="case-section-outputs"]`，否则隐藏处理后竖线高度为 0。独立 `/cases` 新建页不传该 prop，处理流程仍在。

### D3 — 队列步
`listTopics()`，默认 key `default` 置顶。可选已有（含自定义）或 `createTopic` 草稿：输入 key（`^[a-z0-9]+(?:-[a-z0-9]+)*$`）+ 显示名，点新建后选中该 key（若尚未 POST，记 `topicDraft`，提交时再 POST）。本步无写 Case。

「队上有人」：`listEdges()` 中 `enabled` 且 `(subscribe_topics ?? effective_topics)` 含该 key。

### D4 — 节点步前进与新建
- 有人：`next` 不要求 `selectedEdgeId`。
- 无人：必须 `selectedEdgeId`。
- 对话框 `CreateEdgeWizard`，`onDone` 回填 id；跳过通线仍记 id。
- 不写扩容文案。不 `patchEdge` 直到提交。

启用后 `jump` 禁止回到 0–2（`Step4Next` 不渲染回跳指示，底部无回到队列）。

### D5 — 提交链（节点步下一步）
函数 `commitQuickCreate(shared)`，顺序：

1. 若 `topicDraft` 且 `listTopics` 无该 key → `createTopic({ key, name })`
2. `createCase({ ...draft, enabled: false, routing: { rules: [{ when: { always: true }, topic: topicKey }] } })`  
   覆盖 `emptyCase` 里预置的 default 规则，避免「选了别的队却仍带 default」。
3. 若 `selectedEdgeId` 且该 edge 订阅列表不含 `topicKey` → `patchEdge` 追加（不删已有）。
4. `enableCase(id)`
5. `clearQuickConfigSession`；进入第四屏。

任一步失败：toast/行内错误，留在节点步，草稿仍在。Topic 已创建但 Case 失败：下次提交 `createTopic` 会 409 时改为视为已存在（读 `getTopic` 或 catch conflict）。

### D6 — 还差一步
不调用 `computeReadiness` 投放项。展示：已启用、队列名。提醒条件：
- `selectedEdgeId` 有值且 presence 不是 `edge_online && comfy_running`；或
- 未选节点且该 topic 没有任何在线订阅者。

按钮 `Link`/`navigate` 到 `/channels`。短教程落到该按钮。底部「离开」回落地页。

### D7 — 会话 version=3
`SESSION_SCHEMA_VERSION = 3`。载荷：`step`、`caseDraft`、`topicKey`、`topicDraft`、`selectedEdgeId`。去掉 `mode: existing` 与 `pendingEntries`。不匹配即 `removeItem`。启用成功后 clear。恢复时若 step 已是 4 但无 caseId（未提交成功），退回节点步。

### D8 — 无草稿直接进向导
侧栏点「快速新建」时，无会话则 `QuickConfigFlow` 立即渲染。有会话才停一屏继续/废弃。向导「上一步」离开第一步时回工作台 `/`，避免再落到空落地页。
去掉工作流表。「创建工作流」直接 `setWizard({ mode: 'create' })`。侧栏 `menu.quickConfig` →「快速新建」/ `Quick create`。路由仍 `/quick-config`。删除或停用投放步组件引用。

`readiness.ts`：向导完成页不再用；可删投放门禁测试，或仅保留给别处。向导契约测试改为四屏文件名与提交链字符串断言，并补队列/节点前进的纯函数（`queueHasSubscribers`、`shouldBlockNodeStep`）。

## 4. 数据流

```
导入 JSON 草稿
  → 选 topicKey
  → 可选 selectedEdgeId
  → createTopic? → createCase(routing.always→topic) → patchEdge? → enableCase
  → 还差一步（提醒 / 去 /channels）
```

运行时：该 Case 规则命中所选 Topic；订了该 Topic 的节点竞争消费。

## 5. 测试策略

- `session.test.ts`：v2 载荷加载后为 null 且 key 被删；v3 含 topicKey 可恢复。
- `queue-binding` 纯函数：无人 / 有人 / 仅 disabled。
- `WorkflowEditor` 契约：`hideProcessing` 时无 `case-section-processing`；无该 prop 时仍有。
- `quick-config.contract.test.ts`：无已有工作流表；无 Step3Channels；有 Step2Queue；commit 含 `always: true` 与 `enableCase`；无 `putMenu`。
- `done-screen` 删除后改测 `step4-next`：离线提醒分支。
- 手工：demo 两条路径（空白新建节点跳过通线；默认队已有人跳过节点）。

## 6. 风险

- [createTopic 成功、createCase 失败] → 提交把 409 当已存在。
- [enable 后用户想改队列] → 禁止回跳；去工作流页改处理流程。
- [emptyCase 预置 default 规则] → 提交必须用所选 topic 整份覆盖 routing，不能 merge 空。
- [stepRail 仍量 processing] → 与 hideProcessing 同改测量选择器。

## 7. Spec Patch

无。OpenSpec delta 已覆盖显式规则与隐藏处理流程。
