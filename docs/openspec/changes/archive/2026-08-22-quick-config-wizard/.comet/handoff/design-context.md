# Comet Design Handoff

- Change: quick-config-wizard
- Phase: design
- Mode: compact
- Context hash: 72d85d85449e8db8bb47189d194ba8dde1b7bccd26e4c9dca00c8fe9be289060

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## docs/openspec/changes/quick-config-wizard/proposal.md

- Source: docs/openspec/changes/quick-config-wizard/proposal.md
- Lines: 1-31
- SHA256: 1ed2dc0760006b584625c652ed8bec223fc1131ffbcebe0f3ba72bdd23cb6b90

```md
## Why

管理后台要把「一个工作流从创建到真正被用户使用」配通，需要横跨 Case 编辑、任务分流画布（React Flow）、Topic/计算节点管理、渠道菜单四个页面，操作链路长且管理员不知道「还差什么」。需要一个菜单级的「快速配置」入口：用 **Formity 的 flow 步骤机制**（参照其官方 examples 形态）编排三步向导——库一次渲染一个步骤屏幕，配合完成页就绪清单与发布门禁，让不熟悉内部结构的运营能明确知道下一步该做什么。

## What Changes

- 管理后台侧栏新增「快速配置」入口（`admin-web-shell`），落地页 `/quick-config`：继续上次配置 / 新建工作流 / 选择已有工作流；**落地页的选择即向导模式**（`mode` 变量带入流程），Step 1 不再重复询问新建/已有。
- **三步由 Formity flow 步骤机制驱动**（参照 Formity examples：一次只渲染当前步骤屏幕、进度指示「Step X of 3」、`next`/`back`/`jump` 切换、已完成步骤可回跳；不做独立路由页、不堆叠在同一页面）：
  - **Step 1 工作流配置**：按落地页选择的模式直接呈现——新建（复用 `CaseForm`，**强制导入合法 JSON**，杜绝空壳 Case）或调整已有（Case 已选中并加载到同一表单编辑）。
  - **Step 2 处理流程**：独立页面内嵌 `TaskFlowCanvas` 配置模式，行内新建/选择 Topic 与计算节点，保存 `routing` 与节点订阅。
  - **Step 3 投放**：独立页面复用 `MenuPlacementsSection` 与 `action-form`（锁定 `open_workflow`），配置渠道菜单入口。
  - **完成页**：就绪清单（工作流已导入 / 处理流程已配置 / 至少一个投放）+ **发布门禁**（发布 = 启用 Case），缺口项可回跳对应步骤补齐。
- 引入 **`@formity/react`** 作为向导编排引擎（form/variables/condition/return + 步骤屏幕切换），步骤组件自行校验与落库（官方推荐模式）。
- 分步保存与恢复：前进即保存；Formity 状态 + 本地会话支持刷新/重进恢复；落地页「继续上次配置」恢复最近会话。

## Capabilities

### New Capabilities

- `quick-config-wizard`: 快速配置多步向导——落地页、三步独立步骤页（工作流配置/处理流程/投放）、完成页就绪清单与发布门禁、Formity 编排与分步保存恢复。

### Modified Capabilities

- `admin-web-shell`: 侧栏菜单新增「快速配置」项（含路由与 zh/en i18n 文案）；不改既有菜单项与页面布局。

## Impact

- 前端：`web/admin` 新增 `features/quick-config`（落地页、三步步骤页、完成页、就绪计算、Formity flow）；路由 `/quick-config` 与步骤寻址参数；侧栏 `config/menu.ts` 与 i18n 更新；依赖新增 `@formity/react`（`@xyflow/react` 已存在）。
- 消费 API（不改后端契约）：`/api/v1/cases`（list/create/patch/get）、`/api/v1/topics`（CRUD）、`/api/v1/edges`（list/create/patch、`subscribe_topics`）、`/api/v1/routing/attributes`、`/api/v1/channels`（list）、`/api/v1/channels/{id}/menu`（PUT）、`/api/v1/cases/{id}/menu-placements`。
- 前置依赖：`task-flow-editor` change 提供 `TaskFlowCanvas` 配置模式；本 change 在其上做向导编排。
- 非目标：不改后端调度/求值；不新增后端草稿态或模板 API；不做 ComfyUI workflow 画布编辑（完整 Case 编辑器保留）；不做独立路由页或单页堆叠式工作台（步骤由 Formity flow 管理）。

```

## docs/openspec/changes/quick-config-wizard/design.md

- Source: docs/openspec/changes/quick-config-wizard/design.md
- Lines: 1-157
- SHA256: a74ff66b4412c5229c2d66658df78812e7fa33b318814f56e8d1632835978561

[TRUNCATED]

```md
## Context

后台已具备全部底层能力：`BasicsSection` / `WorkflowImportSection` / `FieldCards` / `MenuPlacementsSection`（`features/cases/sections/`）、`TaskFlowCanvas` + `condition-form` + `topic-binding`（`features/task-flow/`，原型已可用）、`action-form` / `card-picker`（`features/menu/`）、`create-edge-wizard`（`features/edges/`）。API 侧 `POST/PATCH /cases`、`/topics` CRUD、`/edges`（含 `subscribe_topics`）、`PUT /channels/{id}/menu`、`GET /cases/{id}/menu-placements` 均存在，Case 文档含 `Routing` 字段。

**关键约束（已核验）**：后端 `catalog` 校验要求 Case 创建时 `inputs`、`bindings.workflow`、`input_schema` 完整合法，且没有草稿概念——「先建壳再补全」会产出启用但不可运行的空壳 Case。`configs/cases/*.json` 是启动种子（`SeedCasesDir`），无模板 API。菜单保存为整树 `PUT`（读改写）。

**用户决策（已确认）**：用 **Formity 的 flow 步骤机制**（`form` 元素）渲染每一步：库一次只渲染当前步骤屏幕，通过 `next`/`back`/`jump` 切换，参照 [formity.app/examples](https://formity.app/examples) 的形态（进度「Step X of 3」、可回跳已完成步骤）；**不做独立路由页**，也不把 ①②③ 堆叠在同一页面。同时保留此前核验结论：不产生空壳 Case、就绪清单与发布门禁放在完成页。

## Goals / Non-Goals

**Goals:**
- 三步（工作流配置 / 处理流程 / 投放）由 **Formity flow 步骤机制**渲染，一次只显示当前步骤屏幕，进度「Step X of 3」，已完成步骤可回跳。
- Formity 编排步骤流（condition 分支 新建/已有、variables 携带 caseId、return 汇总），步骤组件自行校验与落库。
- 完成页就绪清单 + 发布门禁（发布 = 启用 Case）；不产生空壳 Case（新建强制导入）。
- 分步保存与恢复：前进即保存；Formity 状态 + 本地会话支持刷新/重进恢复。

**Non-Goals:**
- 不做独立路由页、不做单页堆叠式工作台（步骤由 Formity flow 管理）。
- 不做第三套编辑实现：所有编辑都是现有组件的组合与内嵌。
- 不新增后端草稿态、不新增模板 API；不改菜单 `PUT` 协议（读改写）；不做乐观锁。

## Decisions

### D1. 入口与路由

- 侧栏在 Dashboard 后新增「快速配置」（icon `Zap`）→ `/quick-config` 落地页：继续上次配置 / 新建工作流 / 选择已有工作流。
- **落地页的选择即向导模式**：点击「新建工作流」→ flow 变量 `mode='create'`；「选择已有工作流」选中 Case → `mode='existing'` 且 `caseId` 已确定。Step 1 不再重复询问模式。
- 向导为 `/quick-config` 内的 **Formity flow**：三个 `form` 步骤 + `return` 完成页，由 `useFormity` 渲染与切换（`next`/`back`/`jump`）；`mode`/`caseId` 在 flow 变量与本地会话间传递。

### D2. 向导壳（Formity examples 形态）

```
┌─ 快速配置 · 新建工作流 ──────────────────────────── [×] ─┐
│ Step 1 of 3        ● ① 工作流配置 → ○ ② 处理流程 → ○ ③ 投放 │
├─────────────────────────────────────────────────────────┤
│                    [当前步骤全屏内容]                     │
│   （Step 2 时画布占满宽度，≥560px）                       │
├─────────────────────────────────────────────────────────┤
│  [上一步]                            [下一步：保存并继续]  │
└─────────────────────────────────────────────────────────┘
```

- 顶部进度「Step X of 3」+ 步骤指示器：通过 Formity `params` + `jump` 实现（参照官方 sidebar-navigation 示例）；已完成步骤可点击回跳，未完成步骤锁定。
- 每步全屏切换（一次只显示一步）；底部主按钮随步骤变化：Step 1/2 = 「下一步：保存并继续」，Step 3 = 「完成并查看摘要」。
- 顶栏摘要 chips：当前工作流名 / 规则数 / 投放数（跨步骤保持上下文）。

### D3. Step 1 工作流配置页（新建 / 调整已有）

```
┌─ Step 1 of 3 · 工作流配置（模式：新建工作流）          ┐
│ [拖拽 JSON / 选择文件 / 粘贴]   ← 强制导入，无导入不可前进 │
│ ✓ 已识别 12 个节点 · 2 输入 · 1 输出                   │
│ 名称 * [________]  说明 [________]  启用 [●]          │
└─────────────────────────────────────────────────────┘
（选择已有进入时：名称/说明/启用已加载，可重新导入替换 workflow）
```

- **模式由落地页入口决定**（Formity 变量 `mode`），Step 1 不再提供模式切换；顶部仅展示当前模式标签与「返回落地页更换」的轻量出口。
- **新建**：复用 `CaseForm` 创建流程（`WorkflowImportSection` 导入 → `FieldCards` 输入/输出 → `BasicsSection` 名称/说明/启用），**未导入合法 JSON 不得前进**（无空壳）。
- **调整已有**：搜索选择 Case（复用 `CaseListPanel`），选中后加载 `name/description/enabled` 到同一表单编辑，可重新导入 JSON 替换 workflow；进入 Step 2 时读取该 Case 的 `routing` 与绑定。
- 提交语义：`下一步` = `POST /cases`（新建）或 `PATCH /cases/{id}`（调整）→ `caseId` 写入 Formity 变量并推进。

### D4. Step 2 处理流程页（React Flow 画布）

- 步骤屏幕内嵌 `TaskFlowCanvas` 配置模式（工具栏：添加规则 / 新建 Topic / 选择 Topic / 新建节点 / 选择节点；无只读切换）。
- 行内新建 Topic（key 自动 slug、`[a-z0-9-]` 校验）与计算节点（精简 `create-edge-wizard`）即时落库（连线需要真实 ID）。
- 分支→Topic 拖线（写 `rule.topic`）、Topic→节点订阅（写 `subscribe_topics`）、拖拽 Y 重排（复用 `reorderByY`）。
- 状态提示复用 `topic-binding`（ready / bound-offline / unbound）；离线节点弱化显示。
- 校验（下一步前）：条件缺失 / 规则未连 Topic / 非法 key 阻止并定位到分支；unbound / 离线仅警告放行（完成页体现）。
- 提交语义：`下一步` = `PATCH /cases/{id}` 写 `routing` + `PATCH /edges/{id}` 写订阅。

### D5. Step 3 投放页

- 「当前已投放」复用 `MenuPlacementsSection`（`GET /cases/{id}/menu-placements`），支持移除。
- 渠道卡片复用 `action-form` 语义并锁定 `type=open_workflow`、`workflow_ids=[caseId]`、direct/list 模式；保存 = 读 `GET /channels/{id}/menu` → 追加/更新按钮 → `PUT` 全量写回。
- 未启用渠道置灰；允许零投放进入完成页（由完成页就绪清单提示缺口，而不是在向导中强制）。

### D6. 完成页：就绪清单与发布门禁

```

```

Full source: docs/openspec/changes/quick-config-wizard/design.md

## docs/openspec/changes/quick-config-wizard/tasks.md

- Source: docs/openspec/changes/quick-config-wizard/tasks.md
- Lines: 1-49
- SHA256: 0e6e0a8fe2dc18bb733d312ed9121a906883a1240f229a0ca64fe672af91840f

```md
## 1. 依赖与入口

- [ ] 1.1 引入 `@formity/react`；spike 验证三步 flow（form/variables/condition/return）嵌入画布与 `CaseForm` 自定义组件、RHF 校验后 `next()` 模式（参照 formity.app/examples）；受阻则记录 `setup-wizard` 回退方案
- [ ] 1.2 侧栏在 Dashboard 后新增「快速配置」项（icon Zap）与落地页 `/quick-config`（支持 `?caseId=` 深链）
- [ ] 1.3 Formity flow 步骤状态：`useFormity`（history/jump/params）与本地会话恢复；zh/en i18n 文案成对

## 2. 落地页与向导壳

- [ ] 2.1 落地页：继续上次配置卡片（本地会话 caseId + 步骤）、新建工作流入口、已有 Case 搜索列表（复用 `CaseListPanel`）
- [ ] 2.2 向导壳：进度「Step X of 3」+ 步骤指示器（已完成可回跳、未完成锁定）、每步全屏切换不堆叠、顶栏摘要 chips
- [ ] 2.3 底部导航：上一步 / 下一步（保存并继续），Step 3 为「完成并查看摘要」

## 3. Step 1 工作流配置页

- [ ] 3.1 模式由落地页入口决定（Formity 变量 `mode` 传入），Step 1 不再提供模式切换；顶部仅展示当前模式标签与「返回落地页更换」出口
- [ ] 3.2 新建：复用 `CaseForm` 创建流程（`WorkflowImportSection` 导入 → `FieldCards` 字段 → `BasicsSection`），**未导入合法 JSON 不得前进**
- [ ] 3.3 调整已有：`CaseListPanel` 选择后加载基础信息到表单编辑，可重新导入 JSON 替换 workflow
- [ ] 3.4 提交：`POST /cases`（新建）或 `PATCH /cases/{id}`（调整），成功后 `caseId` 写入 Formity 变量并推进

## 4. Step 2 处理流程页

- [ ] 4.1 `TaskFlowCanvas` 配置模式嵌入 Formity 步骤屏幕（工具栏：添加规则/新建 Topic/选择 Topic/新建节点/选择节点）
- [ ] 4.2 行内新建 Topic（key slug + `[a-z0-9-]` 校验）与计算节点（精简 create-edge-wizard）即时落库
- [ ] 4.3 保存：`PATCH /cases/{id}` 写 `routing` + `PATCH /edges/{id}` 写 `subscribe_topics`；保存前校验并定位错误到分支
- [ ] 4.4 状态提示（ready / bound-offline / unbound）与离线弱化，供完成页就绪计算使用

## 5. Step 3 投放页

- [ ] 5.1 复用 `MenuPlacementsSection` 展示已有投放并支持移除
- [ ] 5.2 渠道卡片：启用渠道可配置（`action-form` 锁定 `open_workflow` + direct/list），未启用置灰
- [ ] 5.3 保存：读 `GET /channels/{id}/menu` → 追加/更新按钮 → `PUT` 全量写回

## 6. 完成页：就绪清单与发布门禁

- [ ] 6.1 就绪计算（G1/G2/G3）与三态渲染（就绪/警告/缺口）
- [ ] 6.2 发布按钮门禁：缺口禁用并列出原因，点击缺口项回跳对应步骤；全绿可用
- [ ] 6.3 发布操作：`PATCH /cases/{id}` 设置 `enabled: true`；已发布状态展示

## 7. 保存与恢复

- [ ] 7.1 前进即保存、后退加载已存状态；本地会话记录 caseId + 当前步骤
- [ ] 7.2 刷新/重进恢复：Formity 状态 + 本地会话（caseId + 步骤下标）`jump` 到已保存步骤并标记完成状态

## 8. 质量与联调

- [ ] 8.1 Contract 测试：落地页三入口、Formity 步骤一次只显示一步、进度与回跳门控、未导入阻止前进、就绪清单与发布门禁、离线警告、刷新恢复、i18n 成对
- [ ] 8.2 单测：就绪计算（G1/G2/G3）、菜单按钮追加的 PUT 载荷、Formity flow 分支（新建/已有）、key slug
- [ ] 8.3 `pnpm build` + `pnpm test` 通过；bundle 观察 `@formity/react` 体积
- [ ] 8.4 与 `task-flow-editor` 联调冒烟：导入工作流建 Case → 画布配置规则/Topic/节点 → 投放渠道 → 完成页全绿 → 发布 → 回读一致

```

## docs/openspec/changes/quick-config-wizard/specs/admin-web-shell/spec.md

- Source: docs/openspec/changes/quick-config-wizard/specs/admin-web-shell/spec.md
- Lines: 1-20
- SHA256: 9173552d4ec79813f4fd80924af6a57c91cfbd37d64c3b17b2d0f47bb420ec5f

```md
## MODIFIED Requirements

### Requirement: 菜单模块
系统 MUST 提供后台菜单模块，将菜单项映射到路由，并在侧栏展示 Dashboard 与全部资源入口；菜单顺序 MUST 为 Dashboard、快速配置、实例、Case、渠道、Task、User、Session、设置；「主键盘」不再作为一级菜单项；「快速配置」MUST 可导航到三步快速配置向导页，并随语言切换提供中英文案。

#### Scenario: 侧栏展示菜单
- **WHEN** 用户打开控制台
- **THEN** 侧栏可见 Dashboard、快速配置、实例、Case、渠道、Task、User、Session、设置菜单项，且顺序如上，且不含「主键盘」

#### Scenario: 点击菜单进入对应路由
- **WHEN** 用户点击某一菜单项
- **THEN** 导航到该菜单绑定的页面路由

#### Scenario: 快速配置入口打开向导
- **WHEN** 用户点击侧栏「快速配置」
- **THEN** 内容区展示三步快速配置向导页，且第一步可操作

#### Scenario: 快速配置文案随语言切换
- **WHEN** 用户切换中/英文
- **THEN** 侧栏「快速配置」入口与向导页关键文案随语言切换

```

## docs/openspec/changes/quick-config-wizard/specs/quick-config-wizard/spec.md

- Source: docs/openspec/changes/quick-config-wizard/specs/quick-config-wizard/spec.md
- Lines: 1-121
- SHA256: 7de73f83d6ed1fabe8770b8b54ece8f8e3fbd9ff0d834ecd09e7acce3e9c3294

[TRUNCATED]

```md
## Purpose

提供「快速配置」多步向导：三个步骤由 Formity flow 步骤机制渲染（工作流配置 → 处理流程 → 投放），一次只显示一个步骤屏幕，配合完成页就绪清单与发布门禁，让运营无需理解内部数据结构即可完成配置。

## ADDED Requirements

### Requirement: 入口与落地页
系统 MUST 在侧栏提供「快速配置」入口，打开落地页；落地页 MUST 展示「继续上次配置」卡片（基于本地会话）、「新建工作流（进入三步向导）」入口与可搜索的已有工作流列表。用户选择新建或已有后 MUST 进入三步向导。

#### Scenario: 从侧栏进入落地页
- **WHEN** 用户点击侧栏「快速配置」
- **THEN** 打开落地页，可见继续上次配置、新建工作流与选择已有工作流三个入口

#### Scenario: 选择已有工作流进入向导
- **WHEN** 用户在落地页搜索并选中一个 Case
- **THEN** 进入三步向导的第一步，且该 Case 的基础信息已加载到可编辑表单

#### Scenario: 继续上次配置
- **WHEN** 用户存在本地会话且点击「继续上次配置」
- **THEN** 直接恢复到上次所在步骤（或完成页）

### Requirement: 三步由 Formity 步骤机制驱动
向导 MUST 由 Formity flow 定义三个步骤（① 工作流配置、② 处理流程、③ 投放）与完成页；库 MUST 一次只渲染当前步骤屏幕（不得在同一页面堆叠展示多个步骤）；向导 MUST 展示步骤进度（如「Step 1 of 3」）与步骤指示器；已完成步骤 MUST 可通过 Formity 跳转回改，未完成步骤 MUST 不可跳入。

#### Scenario: 一次只渲染当前步骤
- **WHEN** 用户位于向导第二步
- **THEN** 页面只渲染第二步内容与步骤进度，不展示第一步/第三步内容

#### Scenario: 步骤进度与回跳
- **WHEN** 用户完成第二步后点击步骤指示器返回第一步
- **THEN** 系统展示第一步已保存状态，可修改后重新保存

#### Scenario: 未完成步骤不可跳入
- **WHEN** 用户未完成第一步时点击步骤指示器的第二步
- **THEN** 系统禁止跳入并保持当前步骤

### Requirement: 工作流配置步骤
第一步的模式 MUST 由落地页入口决定（新建或选择已有），第一步页面不得再次要求用户选择模式。新建模式 MUST 要求填写名称（必填）并**强制导入合法工作流 JSON**（未导入不得前进，不得使用空模板创建），导入后输入/输出字段配置自动生成；选择已有模式进入时 Case MUST 已确定，名称/说明/启用状态 MUST 已加载到可编辑表单，用户可修改并 MAY 重新导入 JSON 替换 workflow。本步提交 MUST 创建或更新 Case 并持久化。

#### Scenario: 未导入工作流阻止前进
- **WHEN** 用户在新建模式未导入合法 JSON 即点击「下一步」
- **THEN** 系统阻止前进并提示先导入工作流

#### Scenario: 新建工作流并进入下一步
- **WHEN** 用户导入合法 JSON、填写名称并点击「下一步」
- **THEN** 系统创建 Case，步骤推进到第二步并展示已创建工作流的摘要

#### Scenario: 从已有工作流继续并编辑
- **WHEN** 用户在落地页选择已有 Case，进入第一步后修改名称并点击「下一步」
- **THEN** 修改随提交持久化到该 Case，随后进入第二步

#### Scenario: 第一步不重复询问模式
- **WHEN** 用户从落地页「新建工作流」进入第一步
- **THEN** 第一步直接展示新建表单（导入 + 名称/说明/启用），不出现新建/已有模式选择

### Requirement: 处理流程步骤
第二步 MUST 以独立页面呈现任务分流画布（React Flow）：支持新增/删除条件分支、编辑条件、行内新建或选择已有 Topic 与计算节点、拖线连接分支到 Topic 与 Topic 到节点；保存 MUST 持久化 Case `routing` 与节点订阅。非法配置（条件缺失、规则未连 Topic、非法 key）MUST 阻止保存并定位到对应分支；未绑定/离线状态 MUST 以视觉状态展示。

#### Scenario: 配置规则并保存
- **WHEN** 用户在画布新增规则、编辑条件并连接 Topic 后点击「下一步」
- **THEN** 规则按顺序持久化到 Case `routing`，步骤推进到第三步

#### Scenario: 行内新建 Topic 并绑定节点
- **WHEN** 用户新建 Topic（key 符合 `[a-z0-9-]`）并绑定启用节点
- **THEN** Topic 与绑定即时保存，画布显示可消费状态

#### Scenario: 非法配置阻止前进
- **WHEN** 存在条件缺失或未连 Topic 的规则且用户点击「下一步」
- **THEN** 前进被阻止，错误定位到对应分支卡片

### Requirement: 投放步骤
第三步 MUST 以独立页面展示该工作流已有的渠道投放位置（复用现有投放组件）并支持移除；MUST 支持为已启用渠道添加 `open_workflow` 菜单入口（direct/list 模式），保存 MUST 持久化渠道菜单；未启用渠道 MUST 置灰不可配置。

#### Scenario: 查看与移除已有投放
- **WHEN** 用户在投放步骤打开页面
- **THEN** 展示该工作流全部投放位置，且每项可移除

#### Scenario: 为渠道添加工作流入口
- **WHEN** 用户为已启用渠道添加标签并选择 direct 模式后保存
- **THEN** 该渠道菜单持久化包含打开当前工作流的按钮

```

Full source: docs/openspec/changes/quick-config-wizard/specs/quick-config-wizard/spec.md
