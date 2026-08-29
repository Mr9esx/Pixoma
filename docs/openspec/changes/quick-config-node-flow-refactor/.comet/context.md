# Comet Design Handoff

- Change: quick-config-node-flow-refactor
- Phase: design
- Mode: compact
- Context hash: 688052d738871fbc0a71da92b51ecb7bf1adfab2af3d9dea821fceab1b466916

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## docs/openspec/changes/quick-config-node-flow-refactor/proposal.md

- Source: docs/openspec/changes/quick-config-node-flow-refactor/proposal.md
- Lines: 1-31
- SHA256: 7f7e2d7a923d6850acb69c5cb708b53cbe3897b3ddb4318bdad122c7c5ac059a

```md
## Why

现有「快速配置」把运营直接引向「工作流配置 → 处理流程画布 → 投放」三步，对多数无特殊分流需求的用例过重：规则、Topic、节点订阅都埋在画布里，运营必须先理解任务分流概念；节点归属与默认 Topic 缺乏面向普通用例的简化路径。实际创建心智是「工作流怎么用 → 在哪台电脑跑 → 要不要特殊规则」。重构后默认路径更轻，特殊规则按需再进 Flow 编辑器，同时保留投放与完成发布门禁，避免破坏既有发布闭环。

## What Changes

- 保留落地入口「从一个工作流开始」（新建工作流 / 选择已有工作流 / 继续上次配置），保留投放步骤与完成页就绪清单、发布门禁。
- 向导步骤重构为：
  1. **工作流编辑**：沿用统一 `WorkflowEditor`（导入 JSON、名称/说明、输入输出），与现有 Step1 一致。
  2. **节点归属**：指导运营选择「哪台电脑运行」——复用已有的 Edge 节点选择，或新建节点（含部署凭证流程）；**仅记录节点归属，不自动为该节点订阅 Topic**。
  3. **特殊规则分支**：询问「需要特殊规则吗」——不需要则走 **Default Topic**（默认路由，归属节点接收回退 default 主题的任务）；需要则进入 **Flow 可视化编辑器**（`TaskFlowEditor`）配置规则与节点订阅。
  4. **投放**：保留现有投放步骤（为渠道添加打开工作流的菜单入口）。
  5. **完成页**：保留就绪清单与发布门禁（工作流已导入、节点归属完整、处理流程完整、至少一个投放；全部就绪后可发布启用）。
- Case 增加运行节点归属记录，供默认路由投递与规则分支消费。**BREAKING**：旧向导会话不再兼容，打开即清空旧草稿。
- 不改变：调度/条件协议引擎、Edge-Agent 订阅机制、`WorkflowEditor` 工作流编辑组件本身、节点管理页。

## Capabilities

### New Capabilities
- `quick-config-node-assignment`: 向导内选择或新建 Edge 节点并记录到 Case 的运行节点归属；新建节点复用既有节点表单与部署凭证流程。默认分支据此把任务派发到归属节点的 Default Topic 通道。

### Modified Capabilities
- `quick-config-wizard`: 步骤结构重构为「工作流编辑 → 节点归属 → 特殊规则分支（Default Topic / Flow 编辑器）→ 投放 → 完成页」；处理流程画布不再是固定第二屏，改为普通用例走默认路由、特殊用例按需进入编辑器；旧会话不再兼容、打开即清空。

## Impact

- 前端 `web/admin`：`features/quick-config` 的 flow 定义、`step2/step3` 组件重构与置换，新增节点归属选择步骤与「特殊规则」分支画面，保留投放与完成页但调整顺序；进入向导即清理旧会话。
- 数据与接口：`Case` 增加运行节点归属字段（关联 Edge），明确创建/更新/详情载荷契约；default 分支的派发依赖该归属。
- 依赖/复用：`WorkflowEditor`、`TaskFlowEditor`、Edge 选择/`CreateEdgeWizard` 组件、既有投放与完成页组件。
- i18n：新增步骤标题与分支文案（zh/en）。
- 文档：管理配置指引补充新向导步骤说明。

```

## docs/openspec/changes/quick-config-node-flow-refactor/design.md

- Source: docs/openspec/changes/quick-config-node-flow-refactor/design.md
- Lines: 1-58
- SHA256: e43c2a1314a30f444e8c3f78fe3dfe08502506c016fad0236b8c39af2b344438

```md
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

```

## docs/openspec/changes/quick-config-node-flow-refactor/tasks.md

- Source: docs/openspec/changes/quick-config-node-flow-refactor/tasks.md
- Lines: 1-46
- SHA256: 92b229d3b922f13674e200391889b36f3ad55437c8cecedfc9ba3edb5fc9e756

```md
## 1. 会话版本化与旧版清空

- [ ] 1.1 `lib/session.ts` 会话载荷增加 `schemaVersion` 标记（新值），保存路径写入该标记
- [ ] 1.2 向导入口（`QuickConfigPage` 的 `loadQuickConfigSession` 与恢复路径）检测旧版/缺失版本会话时清除并视为无会话
- [ ] 1.3 单品列表：旧版会话不恢复、可重新「从一个工作流开始」（vitest）

## 2. Flow 步骤结构重构

- [ ] 2.1 `quick-config-flow.tsx` 屏序调整为：工作流编辑 → 节点归属 → 特殊规则 → 投放 → 完成页
- [ ] 2.2 既有 Step2 处理流程画布从固定屏移除，迁移为特殊规则分支的按需编辑器内容
- [ ] 2.3 步骤指示器/`wizard-chrome` 进度文案适配「Step N of 4」（含 i18n zh/en）
- [ ] 2.4 合约测试：新屏序、一次只渲染当前屏、已完成步骤可回跳

## 3. 节点归属步骤（Step2）

- [ ] 3.1 `types.ts` 的 `WizardShared` 增加节点归属草稿状态（`assignedEdgeId`）
- [ ] 3.2 新增 Step2 节点归属组件：`listEdges()` 可选中列表 + presence/在线态；无节点时可打开复用 `CreateEdgeWizard`
- [ ] 3.3 新建成功后回填归属并展示就绪；仅记录归属、不做任何订阅操作
- [ ] 3.4 合约测试：选择已有节点、新建节点归属、不产生订阅请求

## 4. 特殊规则分支（Step3）

- [ ] 4.1 新增「规则分支」容器：先展示「不需要（Default Topic）/ 需要（Flow 编辑器）」选择卡
- [ ] 4.2 「不需要」：采用默认路由（不写伪规则），`routing` 保持默认、记录归属后推进到投放
- [ ] 4.3 「需要」：同屏展开复用 `TaskFlowEditor`，支持规则/Topic/节点订阅编辑与非法配置拦截
- [ ] 4.4 提交时按分支持久化 `routing`；合约测试：two 分支路径与非法配置定位

## 5. 完成页与发布次序

- [ ] 5.1 `done-screen`/`readiness` 就绪项增加「节点归属完整」，四就绪项三态展示
- [ ] 5.2 归属缺失时发布禁用并可回跳节点归属步骤
- [ ] 5.3 发布提交按依赖顺序：先 Case（含归属与 routing），再投放菜单
- [ ] 5.4 合约测试：四就绪项、归属缺口、全绿发布

## 6. 后端 Case 节点归属

- [ ] 6.1 `Case` 领域模型与 DTO 增加可选 `assignedEdgeId`（关联 Edge，缺省兼容旧数据）
- [ ] 6.2 `PATCH /cases/{id}` 载荷支持读写归属并随详情返回
- [ ] 6.3 `go build ./...` + `go test ./...`；确认默认路由数据面不因此变更

## 7. 验证与文档

- [ ] 7.1 `pnpm tsc -b` + `pnpm vitest run`（快速配置与相关组件测试通过）
- [ ] 7.2 冒烟：新建工作流 → 选节点 → 默认路由 → 投放 → 发布，Case 启用且归属可读
- [ ] 7.3 冒烟：走特殊规则分支进 Flow 编辑器配置并保存
- [ ] 7.4 管理配置指引补充新四步向导与旧会话清空说明

```

## docs/openspec/changes/quick-config-node-flow-refactor/specs/quick-config-node-assignment/spec.md

- Source: docs/openspec/changes/quick-config-node-flow-refactor/specs/quick-config-node-assignment/spec.md
- Lines: 1-27
- SHA256: d8a0e210249dd63f2f30d77de7c6a40334914c3c585d07e4961510dee0a029c7

```md
## Purpose

让「快速配置」向导在选择「哪台电脑运行工作流」时复用或新建 Edge 节点作为运行节点，并在无需特殊规则时把该节点直接绑定 default Topic，使该 Case 回退 default 的任务能被该节点消费。

## ADDED Requirements

### Requirement: 运行节点选择
向导 MUST 提供「哪台电脑运行你的工作流」步骤，用户 MUST 可选择已有 Edge 节点或新建节点（复用既有节点表单与部署凭证流程）；本步 MUST 仅把选定节点记录为运行节点的向导状态，MUST NOT 写入 Case 模型，MUST NOT 立即持久化节点订阅。

#### Scenario: 选择已有节点
- **WHEN** 用户在运行节点步骤选择一个已有 Edge 节点并推进
- **THEN** 该节点被记录为向导的运行节点，不产生任何持久化副作用

#### Scenario: 新建节点并选定
- **WHEN** 没有匹配节点时用户走新建流程完成注册并部署就绪
- **THEN** 新建节点被记录为运行节点，部署就绪状态对用户可见

### Requirement: Default 分支绑定与派发
「不需要特殊规则」时，向导 MUST 对选定运行节点执行 `PATCH /edges/{id}`，将节点 `subscribe_topics` 加入 `default`（复用该项既有 default 订阅能力），使该 Case 回退 default 的任务由订阅 default 的节点参与竞争消费；向导 MUST NOT 引入 Case 级节点归属字段。

#### Scenario: 默认分支绑定 default
- **WHEN** 用户选择「不需要特殊规则」并在完成页提交
- **THEN** 选定节点 `subscribe_topics` 被写入包含 `default`，任务回退 default 可由该节点消费

#### Scenario: 竞争消费语义
- **WHEN** 另一台节点原本就订阅 default
- **THEN** 该 Case 的回退 default 任务亦可被其消费（非独占），不因本次绑定而受影响

```

## docs/openspec/changes/quick-config-node-flow-refactor/specs/quick-config-wizard/spec.md

- Source: docs/openspec/changes/quick-config-node-flow-refactor/specs/quick-config-wizard/spec.md
- Lines: 1-70
- SHA256: 4dfd3fb1c66c7535ad245b86c51f003170c2a1e56346b5cc9733af9c1078a54c

```md
## MODIFIED Requirements

### Requirement: 三步由 Formity 步骤机制驱动
向导 MUST 由 Formity flow 定义四个配置步骤与完成页：① 工作流编辑、② 运行节点、③ 特殊规则、④ 投放；库 MUST 一次只渲染当前步骤屏幕（不得在同一页面堆叠展示多个步骤）；向导 MUST 展示步骤进度（如「Step N of 4」）与步骤指示器；已完成步骤 MUST 可通过 Formity 跳转回改，未完成步骤 MUST 不可跳入。

#### Scenario: 一次只渲染当前步骤
- **WHEN** 用户位于向导第三步
- **THEN** 页面只渲染第三步内容与步骤进度，不展示第二步/第四步内容

#### Scenario: 步骤进度与回跳
- **WHEN** 用户完成第二步后点击步骤指示器返回第一步
- **THEN** 系统展示第一步已保存状态，可修改后重新保存

#### Scenario: 新增运行节点步骤
- **WHEN** 用户完成工作流编辑后点击步骤指示器进入第二步
- **THEN** 系统展示运行节点步骤（选择或新建 Edge 节点）

### Requirement: 处理流程步骤
本步以「你的工作流需要特殊规则吗」决定处理流程：不需要 MUST 采用默认路由，对选定运行节点写入 `default` 订阅，引擎回退 default 后由订阅 default 的节点竞争消费；需要 MUST 提供进入既有独立规则编辑页的入口（该入口在向导外完成规则/Topic 配置，本期不在向导内重建 Flow 编辑器）。

#### Scenario: 不需要特殊规则走默认路由
- **WHEN** 用户选择「不需要特殊规则」并推进
- **THEN** 采用默认路由，选定节点绑定 default 订阅，回退 default 的任务由订阅 default 的节点消费

#### Scenario: 需要特殊规则进入独立编辑页
- **WHEN** 用户选择「需要特殊规则」并确认
- **THEN** 向导把该工作流交付到既有独立规则编辑页继续配置，不在向导内部渲染编辑器

### Requirement: 完成页就绪清单与发布门禁
完成页 MUST 展示四项就绪状态：工作流已导入（`bindings.workflow`、`inputs`、`input_schema` 完整）、运行节点已选且订阅就绪、处理流程就绪（默认路由或规则已配置）、至少一个渠道投放；状态 MUST 以三态展示（就绪 / 警告 / 缺口）。存在缺口时 MUST 禁用发布按钮、列出缺口，并允许从完成页回跳到对应步骤补齐；全部就绪后发布按钮 MUST 可用，发布操作 MUST 将 Case 设为启用。

#### Scenario: 运行节点缺口阻塞发布
- **WHEN** 用户未选定运行节点就到达完成页
- **THEN** 完成页显示运行节点缺口，发布按钮禁用，可回跳到运行节点步骤补齐

#### Scenario: 全绿后可发布
- **WHEN** 工作流已导入、运行节点已选且订阅就绪、处理流程就绪、存在至少一个投放
- **THEN** 发布按钮可用；点击后 Case 被启用并提示已发布

#### Scenario: 离线节点不阻塞发布
- **WHEN** 运行节点离线但选定与订阅关系完整
- **THEN** 完成页该就绪项显示警告（可发布），发布按钮保持可用

### Requirement: 分步保存与恢复
完成页提交前，向导 MUST 只收集草稿（Case、运行节点、default 订阅待写项、待投放菜单），不强制在步骤间落库；完成页 MUST 按依赖顺序统一提交（先工作流，再运行节点 default 订阅，再投放）。由于流程重构（BREAKING），旧版本地会话结构 MUST NOT 再被兼容或恢复，用户打开向导时 MUST 清理旧版会话并从头开始。

#### Scenario: 旧版会话被清空
- **WHEN** 用户打开重构后的向导且本地存在旧版会话草稿
- **THEN** 向导清空该旧版会话并重新从第一步开始，不回恢复旧草稿

#### Scenario: 刷新恢复当前步骤
- **WHEN** 用户完成运行节点步骤后刷新页面
- **THEN** 按新结构会话恢复到当前步骤，无需重新创建

#### Scenario: 引导中途退出恢复
- **WHEN** 用户完成前两步后退出，稍后从落地页「继续上次配置」进入
- **THEN** 从新结构当前步骤继续，无需重复配置

## ADDED Requirements

### Requirement: 运行节点步骤
向导 MUST 提供「哪台电脑运行你的工作流」运行节点步骤，位于工作流编辑之后、特殊规则之前；用户 MUST 可选择已有 Edge 节点或新建节点（复用既有节点表单与部署凭证流程）。本步 MUST 仅把节点记录为运行节点（向导状态），MUST NOT 写入 Case 模型，MUST NOT 立即持久化节点订阅。

#### Scenario: 选择已有节点
- **WHEN** 用户在运行节点步骤选择已有 Edge 节点并推进
- **THEN** 该节点记录为运行节点且无持久化副作用，随后进入特殊规则步骤

#### Scenario: 新建节点并选定
- **WHEN** 无匹配节点时用户新建节点并部署就绪
- **THEN** 新建节点记录为运行节点并提供就绪状态

```
