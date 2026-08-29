## Context

现有「快速配置」为 4 个 Formity 屏：Step1 工作流编辑（`WorkflowEditor`）、Step2 处理流程画布（`TaskFlowEditor` + 节点/Topic 内联弹窗）、Step3 投放、完成页就绪清单与发布门禁；节点模型是 `ComfyEdge`（`features/edges`，含 `CreateEdgeWizard` 部署凭证流与 presence 在线态）。调度语义已支持「规则首中命中选 Topic、无命中回退 `default` Topic」，edge-agent 默认订阅 `default`。动机见 proposal.md。

## Goals / Non-Goals

**Goals**
- 4 个配置步骤 + 完成页：工作流编辑 → 节点归属 → 特殊规则（Default / Flow 编辑器）→ 投放 → 完成页发布。
- 节点归属步骤复用既有 Edge 选择与新建流程，仅记录归属、不引入显式订阅操作。
- 特殊规则按需进入 `TaskFlowEditor`，普通用例走默认路由。
- 旧版会话打开即清空（BREAKING）。

**Non-Goals**
- 不改调度/条件协议引擎、edge-agent 订阅机制与 `WorkflowEditor` 组件本身。
- 不做多节点归属集合（单归属即可满足「哪台电脑运行」）。
- 不改节点管理页与完成后台派发的数据面。

## Decisions

### D1 — 步骤结构与特殊规则分支采用「同屏条件渲染」
`quick-config-flow.tsx` 仍是 4 个配置 Formity 屏 + 完成页。Step3「特殊规则」首屏是一组选择卡：「不需要特殊规则（Default Topic）」/「需要特殊规则（Flow 编辑器）」。选「需要」时**在同一屏内**展开 `TaskFlowEditor`（复用 Ultrastate），选「不需要」则直接以默认路由推进到 Step4 投放。
- 备选：把 Flow 编辑器做成独立 Formity 步骤。否决：会让步骤数与用户感知不一致，且在模板分支切换时增加 wizard-chrome 复杂度。
- 理由：保持 Formity 步数与左手栏「Step N of 4」一致；编辑器作为分支后的内容态而非独立步骤。

### D2 — 节点归属为 Case 上的单归属字段
`CaseRecord` 增加可选归属字段（记为 `assignedEdgeId?: string | null`，指向 `ComfyEdge.id`）。Step2 展示 `listEdges()` 可选列表 + presence 在线/消费状态；无匹配节点时在 Step2 打开复用 `CreateEdgeWizard`，成功后回填 `assignedEdgeId`。归属通过既有 `PATCH /cases/{id}` 随 Case 持久化并在详情返回。
- 备选：新增独立的 case–edge 归属表/多值集合。否决：增加数据面与任务复杂度，超出快速配置范围。
- 风险：后端 Case 模型当前无该字段 → 需扩展 model/DTO/接口载荷，是本 change 最主要的后端点。

### D3 — Default 分支 = 默认路由，复用节点既有 default 订阅能力
「不需要特殊规则」不写伪规则：`routing` 保持默认（空规则 → 引擎回退 `default` Topic），并记录 `assignedEdgeId`。因 edge-agent 既有「默认订阅 `default`」语义，归属节点通过其已有 default 订阅即可消费该 Case 的回退任务，向导**不新增任何显式订阅操作**（呼应「不需要再订阅 Default」）。
- 备选 A：为节点创建一条「case→default」显式订阅。否决：与「不自动订阅」要求冲突，且是多余状态。
- 备选 B：默认分支自动写入一条无条件规则链接 default。否决：制造伪规则，偏离语义。
- 约束：同 `D2` 依赖归属字段；节点选择画面展示 `default` 订阅/在线就绪提示，缺失时在完成页以「节点归属缺口/警告」拦截。

### D4 — 会话版本化实现旧版清空
`lib/session.ts` 的会话载荷增加 `schemaVersion`（新值如上，如 `2`）；向导入口（`QuickConfigPage` 的 `loadQuickConfigSession` 与恢复路径）检测到版本缺失或不匹配的旧载荷时**清除并当作无会话**，从第一步重新开始。
- 备选：字段级迁移旧会话。否决：BREAKING 已明确不兼容，迁移维护成本高。

### D5 — 完成页就绪项与提交顺序
完成页四项就绪：工作流已导入、节点归属完整、处理流程就绪（默认路由或规则/Topic/节点绑定完整）、至少一个投放。发布提交按依赖顺序：先 Case（含 `assignedEdgeId`、`routing`），再投放菜单。就绪/缺口/警告三态与回跳沿用 `done-screen` 与 `readiness` 逻辑并补充归属项。

## Risks / Trade-offs

- [后端 Case 缺少节点归属字段] → 扩展 `Case` model/DTO 与 `PATCH /cases` 载荷，字段可选且缺省兼容旧数据。
- [default 消费依赖节点「默认订阅 default」既有行为，个别节点未订阅则收不到] → 节点选择与完成页展示 default 订阅就绪提示，缺口项回跳。
- [同屏分支切换编辑器增大 Step3 组件复杂度] → 抽取「规则分支」容器组件，编辑器直接复用 `TaskFlowEditor`，不重复实现。
- [旧会话强行清空可能丢失未完成配置] → 属明确的 BREAKING 范围，交付说明中提示。

## Migration Plan

1. 前端先落地：会话版本化清空、Flow 屏置换（Step2 节点归属、Step3 分支、Step4 投放/done 顺序调整），会话与草稿字段兼容旧结构读取。
2. 后端再上：`Case` 归属字段扩展 + `PATCH /cases` 契约，前端接通 `assignedEdgeId` 持久化。
3. 发布：完成页就有新就绪项；回滚风险前端为主（纯前端阶段可先行合入，后端字段后随）。

## Open Questions

- 归属是否允许多节点集群承接（如多个归属节点消费同一 Case 的 default 任务）：本期按单归属设计，多归属作为后续增强，不影响本次 spec/任务划分。
