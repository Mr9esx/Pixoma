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
┌─ 完成 · Step 4/4 ────────────────────────────────────┐
│ 配置摘要：动漫图像生成 · 2 条规则 · 2 个投放            │
│ 就绪清单：                                            │
│   [✓] 工作流已导入（12 节点 · 2 输入 · 1 输出）        │
│   [✕] 处理流程有缺口：Topic B 未绑定节点   → 回 Step 2 │
│   [✓] 已投放 2 个渠道位置                             │
│                                [发布]（缺口时禁用）    │
└──────────────────────────────────────────────────────┘
```

- 就绪模型（纯前端派生，无新后端状态）：
  - **G1 工作流就绪**：`bindings.workflow` 非空、`inputs.length > 0`、`input_schema` 非空。
  - **G2 处理流程就绪**：`routing.rules.length >= 1` 且每条规则已连 Topic 且每个使用中的 Topic 至少绑定一个**启用**节点；全部离线 = 警告不阻塞；未绑定 = 缺口。
  - **G3 投放就绪**：`menu-placements` 非空。
- 发布按钮 = `PATCH /cases/{id}` 设置 `enabled: true`；缺口禁用并列出原因，点击缺口项 `jump` 回对应步骤；黄项允许发布。

### D7. Formity 编排与保存/恢复

```ts
const flow: Flow<WizardSchema> = [
  // mode/caseId 由落地页入口传入（不是步骤内字段）
  { variables: () => ({ mode, caseId, routing: undefined, placements: [] }) },
  {
    form: {
      fields: () => ({}),
      render: Step1Workflow, // 按 mode 渲染新建或已有；强制导入；POST/PATCH /cases
    },
  },
  { form: { fields: () => ({ routing: [undefined, []] }), render: Step2ProcessingFlow } },
  { form: { fields: () => ({ placements: [[], []] }), render: Step3Channels } },
  { return: ({ caseId, routing, placements }) => ({ caseId, rules: routing?.rules.length, placements }) },
]
```

- Formity 状态与恢复：`useFormity` 持有流程状态（history/jump）；本地会话记录 caseId + 当前步骤下标，重新进入时从已保存数据恢复并 `jump` 到对应步骤。
- **完成页统一提交**（用户确认）：前三步只收集草稿（Case 文档、routing、待投放菜单），可自由前进/后退；完成页按依赖顺序一次性落库（建/改 Case → 写 routing → 写菜单 → 可选启用）；本地会话保存完整草稿，落地页「继续上次配置」一键恢复。
- Formity 只做流程编排（condition/jump/variables/return）；校验与落库均在 step 组件内（官方推荐模式：组件用 RHF + zod 校验，通过后调 `next()`）。

### D8. 组件复用清单

| 能力 | 复用来源 |
|---|---|
| 工作流基础信息 / 导入 / 字段配置 | `features/cases/sections/{basics,workflow-import,field-cards}.tsx` + `lib/workflow-parse.ts` |
| Step 1 创建流程 | `features/cases/case-form.tsx`（CaseForm） |
| Case 搜索列表 | `features/cases/list-panel.tsx`（CaseListPanel） |
| 处理流程画布/条件/绑定 | `features/task-flow`（TaskFlowCanvas、condition-form、topic-binding、reorderByY） |
| 节点新建 | `features/edges/create-edge-wizard` 精简 |
| 已投放列表 | `features/cases/sections/menu-placements.tsx`（MenuPlacementsSection） |
| 菜单按钮/卡片 | `features/menu/action-form`、`card-picker`、`menu-card-editor` |
| 向导壳/表单控件 | Formity examples 形态 + shadcn/ui |

### D9. 测试与质量

- Contract 测试：落地页三入口、Formity 步骤一次只显示一步、进度与回跳门控、未导入阻止前进、就绪清单与发布门禁（缺口禁用/全绿可用/离线警告）、刷新恢复、i18n 成对。
- 单测：就绪计算（G1/G2/G3）、菜单按钮追加的 PUT 载荷、Formity flow 分支（新建/已有）、key slug。
- 构建验收：`pnpm build` + `pnpm test` 通过；`@formity/react` 体积观察。

## Risks / Trade-offs

- [Formity 1.x 与自定义组件（画布/CaseForm）集成不确定] → 首个任务为 spike（三步 flow + 嵌入画布与 CaseForm，参照官方 examples）；受阻回退 `setup-wizard` 步骤状态机，交互契约不变。
- [Step 1 与完整 Case 编辑器功能重叠] → 直接复用 `CaseForm`，不新写第二套创建实现。
- [刷新恢复依赖 Formity history 与本地会话的协调] → spike 内验证 `useFormity` history/jump 与本地会话恢复；必要时自定义步骤状态存储。
- [菜单 PUT 全量覆盖并发丢失] → 读-改-写 + 单管理员场景接受；不做乐观锁。
- [G2 依赖 TaskFlowCanvas 配置模式改造] → 与 `task-flow-editor` 顺序交付；组件缺失时步骤页显示前置依赖空态。
- [就绪计算依赖 `routing` 字段的前端类型] → `task-flow-editor` 已定义 `CaseWithRouting`，向导消费该类型。

## Migration Plan

- 纯前端增量：新增 `features/quick-config`（落地页、Formity flow 三步步骤 + 完成页、就绪计算）、侧栏菜单项、`@formity/react` 依赖；无数据迁移。
- 回滚：移除菜单项与路由即整体下线，后端不受影响。
- 依赖顺序：`task-flow-editor` 先提供画布配置模式；本 change 联调三步向导。

## Open Questions

- 「从示例模板新建」：需要模板 API 或前端内置清单（当前 `configs/cases` 仅启动种子）——列为后续增强，不影响本 change。
- 发布语义是否长期等价 `enabled=true`（是否需要独立发布状态）——当前沿用现有字段。
- 浏览器前进/后退与 Formity history 的整合程度（是否需要路由同步）——实现偏好，build 期决定，不影响行为契约。
