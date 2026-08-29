---
comet_change: quick-config-node-flow-refactor
role: technical-design
canonical_spec: openspec
---

# 快速配置向导四步重构 · 深度技术设计

## 1. Context

现有快速配置为 4 个 Formity 屏（`quick-config-flow.tsx`）：Step1 工作流编辑（`WorkflowEditor`）、Step2 处理流程画布（`TaskFlowEditor`）、Step3 投放、完成页就绪清单与发布门禁。节点模型为 `ComfyEdge`，`EdgeRecord` 含 `subscribe_topics` / `effective_topics`，`patchEdge(id, { subscribe_topics })` 已支持直接更新节点 Topic 绑定（`lib/api/edges.ts`）。调度语义为「规则首中命中选 Topic、无命中回退 `default` Topic，订阅 default 的节点竞争消费」。

本次重构由 open 阶段 proposal/design/specs/tasks 约束，经 Brainstorming 确认三点核心调整：
- 采纳「节点自绑定 default、竞争消费」（非独占），并**取消 Case 级归属字段**设计，后端零新增。
- **`TaskFlowEditor` 本期不接入向导**；「需要特殊规则」保留入口但**跳转既有独立规则编辑页**。
- 旧会话清空（BREAKING）。

## 2. Goals / Non-Goals

**Goals**
- 四步向导 + 完成页：工作流编辑 → 运行节点 → 特殊规则 → 投放 → 完成页发布。
- 运行节点步骤：选择已有或新建 Edge 节点（草稿态 `selectedEdgeId`），不写 Case 模型。
- Default 分支：完成提交时对选定节点 `patchEdge` 写入 `default` 订阅。
- 规则分支：先落库工作流，跳转既有独立规则编辑页，向导退出。
- 完成页四就绪 + 发布门禁；会话 schemaVersion 版本化、旧会话清空。

**Non-Goals**
- 不引入 Case 级 `assignedEdgeId`，不改 `Case` 模型/DTO/PATCH 载荷、不改调度/数据面与执行面。
- 本期不在向导内重建或内嵌 Flow 编辑器；不做独占派发。
- 不改 `WorkflowEditor`、`CreateEdgeWizard`、edge-agent 订阅机制本身。

## 3. 深度技术决策

### D1 — 向导步骤组件与共享状态
`WizardShared`（`features/quick-config/types.ts`）增加运行节点草稿：`selectedEdgeId: string | null`。其余 `caseRecord`、`routing`、`placements`、`pendingEntries` 复用。四屏映射调整：
- Step1 `Step1Workflow`（复用 `WorkflowEditor`，不变）。
- Step2 新建 `Step2Node`（运行节点选择）。
- Step3 新建 `Step3Rules`（特殊规则分支）。
- Step4 `Step3Channels`（原投放组件，屏序后移保留）。
- 完成页 `DoneScreen`（就绪项更新）。

### D2 — 运行节点步骤实现
`Step2Node` 复用 `listEdges()` 可选中列表 + `listPresence()` 展示在线/消费状态；无 `subscribe_topics` 相关交互（订阅在 Default 分支统一写）。无匹配节点时内嵌复用 `CreateEdgeWizard` 对话框，成功后回填 `selectedEdgeId`。本步仅为草稿态，不做任何写请求。

### D3 — Default 分支实现（default 订阅写入）
「不需要特殊规则」分支不写伪规则：`routing` 保持默认（空规则 → 引擎回退 default）。完成页统一提交阶段对选定节点执行：
`PATCH /edges/{id} { subscribe_topics: [...(已有), 'default'] }`
- 幂等：读取节点现有 `subscribe_topics`，仅当不含 `default` 时追加；无操作则跳过。
- 失败处理：提交中断，保留草稿并显示可读错误，允许重试；不产生部分已发布状态。
- 数据面不新增逻辑：回退 default 由现有调度承担，订阅 default 节点竞争消费。

### D4 — 规则分支跳转（TaskFlowEditor 不接入）
「需要特殊规则」选择后：
1. 向导先把 Step1 收集的工作流落库（创建/更新 Case），取得 `caseId`。
2. 跳转既有独立规则编辑页（任务 Flow 独立页）继续进行规则/Topic/节点订阅配置。
3. 向导退出；会话标记该 Case 交由规则路径接管（写入会话 `handledBy='editor'` 且不再恢复旧草稿），避免旧会话干扰。
- 取舍：向导不再负责规则路径的完成页/发布；规则与订阅在独立页闭环。投放与本分支的衔接在独立编辑流程中处理（不在本期向导范围）。
- 约束：跳转目标页复用现有 task-flow 独立页能力，本期不重建编辑器。

### D5 — 完成页提交与四就绪
完成页四项就绪（`lib/readiness.ts` 扩展）：
1. 工作流已导入（`bindings.workflow`、`inputs`、`input_schema` 完整）。
2. 运行节点已选（`selectedEdgeId` 非空）。
3. 处理流程就绪：默认路由（default 就绪项）——Default 分支满足；规则分支由独立页负责。
4. 至少一个投放（`pendingEntries` 非空）。
发布提交按依赖顺序：① 工作流（创建/更新 Case）→ ② 运行节点 default 订阅（仅 Default 分支）→ ③ 投放菜单。三态就绪（就绪/警告/缺口）与回跳沿用 `done-screen` / `readiness`。

### D6 — 会话版本化清空
`lib/session.ts` 载荷增加 `schemaVersion: 2`；`QuickConfigPage` 的 `loadQuickConfigSession` 与恢复路径检测旧版/缺失版本即清除并按无会话处理，从第一步重新开始（BREAKING）。

### D7 — 复用与 i18n 边界
组件复用：`WorkflowEditor`（Step1）、Edge 列表/`CreateEdgeWizard`（Step2）、`TaskFlowEditor` 不在向导内使用、投放与完成页组件现成。新建 `Step2Node`、`Step3Rules` 组件与对应文案；i18n 同步 `zh.json`/`en.json`（新增步骤标题与分支文案）。

## 4. 数据流

**Default 路径**
`工作流编辑` → `运行节点(selectedEdgeId)` → `不放行规则(默认)` → `投放` → 完成页：`POST/PATCH Case` → `patchEdge 追加 default` → 写投放菜单 → 发布启用 Case。

**规则路径**
`工作流编辑` → `运行节点` → `需要规则` → 落库 Case 取 `caseId` → 跳既有独立规则编辑页（向导退出，会话标记 `handledBy='editor'`）。

## 5. 模块与接口映射

- 改 `features/quick-config/quick-config-flow.tsx`：四屏置换 + `selectedEdgeId` 草稿。
- 改 `features/quick-config/types.ts`：`WizardShared.selectedEdgeId`、`handledBy?`。
- 新增 `features/quick-config/step2-node.tsx`（运行节点选择）。
- 新增 `features/quick-config/step3-rules.tsx`（特殊规则分支：默认路由 / 跳独立页）。
- 改 `features/quick-config/done-screen.tsx`、`lib/readiness.ts`、`lib/session.ts`（就绪项 + 版本化）。
- 复用 `lib/api/edges.ts`（`listEdges`/`listPresence`/`patchEdge`）、`features/edges/create-edge-wizard.tsx`。
- 后端：无改动（复用 `patchEdge` 的 `subscribe_topics` 与既有调度）。

## 6. 错误处理与边界

- `patchEdge` 失败：显示可读错误、保留草稿、可重试；不产生部分发布。
- 运行节点离线：完成页该就绪项为「警告（可发布）」。
- 旧会话：打开即清空。
- 规则分支：跳转独立页失败且工作流已落库 → 提示继续到独立页，不反复重试。
- Default 分支空 `selectedEdgeId`：完成页阻塞发布并回跳。

## 7. 测试策略

- 前端合同测试（`quick-config.contract.test.ts`、`session.test.ts`、`readiness.test.ts`）：新屏序、运行节点步骤、default 分支绑定、完成页四就绪、会话版本化清空。
- 后端 `go test ./...`：仅回归（无 Case 模型改动时验证无回归）。
- 冒烟：默认路径完整链路（工作流→节点→default→投放→发布）；规则路径（落库→跳独立页）。

## 8. Risks / Trade-offs

- [非独占派发] → 接受竞争消费语义，文档与 UI 措辞不承诺独占；如需独占留待后续引入 Case 归属。
- [规则分支跳转依赖既有独立页能力] → 不重建编辑器；若独立页能力缺失，本期以「工作流落库 + 跳转」为交付边界，编辑器强化后续补。
- [default 订阅已存在则跳过] → 幂等合并，避免覆盖节点既有订阅。
- [会话清空丢失未完成配置] → 属明确 BREAKING，交付说明提示。

## 9. Spec Patch 摘要

已回写两个 delta spec：`quick-config-node-assignment`（运行节点选择 + Default 绑定与派发）。`quick-config-wizard`（步骤驱动、处理流程→默认路由/独立页、完成页四就绪、分步保存与恢复、新增运行节点步骤）。
