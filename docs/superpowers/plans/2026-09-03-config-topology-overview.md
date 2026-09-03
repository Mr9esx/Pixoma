---
change: config-topology-overview
design-doc: docs/superpowers/specs/2026-09-03-config-topology-overview-design.md
base-ref: 15878a784532800bfd04edfc32da6cd70965a5bf
---

# 配置拓扑总览 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 后台工作台卡片 + 四类详情弹窗，用只读 React Flow 画出消息平台 → 工作流 → 任务队列 → 计算节点。

**Architecture:** `features/config-topology` 组装现有列表与每通道菜单，`buildLinkGraph` 出图，健康复用 `link-health` 的 `*References`。`LinkGraph` 共用；工作台 `Card`、详情 `Dialog`。工作台不主动 `POST /channels/:id/check`；无缓存则平台节点不标绿。

**Tech Stack:** React、TanStack Query、`@xyflow/react`、shadcn Card/Dialog/Button、`StatusDot`、vitest 合同测试。

## Global Constraints

- 只读；不画菜单入口节点；全量丢度数 0；焦点图无边仍留自身。
- 健康绿/黄与 `*References` 一致；未就绪不标绿；禁止红色档。
- 文案：配置拓扑 / 拓扑 / 打开详情；无解释句。
- 语义令牌 + `flex`/`gap`；Dialog 可有阴影，Card 无投影。
- 新测试登记 `web/admin/vitest.config.ts`。
- 不改 `docs/architecture/`。

## File map

- Create: `web/admin/src/features/config-topology/lib/walk-menu-workflows.ts`
- Create: `web/admin/src/features/config-topology/lib/build-link-graph.ts`
- Create: `web/admin/src/features/config-topology/lib/path-through.ts`
- Create: `web/admin/src/features/config-topology/use-topology-source.ts`
- Create: `web/admin/src/features/config-topology/topology-node.tsx`
- Create: `web/admin/src/features/config-topology/link-graph.tsx`
- Create: `web/admin/src/features/config-topology/topology-card.tsx`
- Create: `web/admin/src/features/config-topology/topology-dialog.tsx`
- Modify: `workbench-data-board.tsx`、四类 `*-detail-panel.tsx`、`zh.json`/`en.json`、`vitest.config.ts`、相关合同测试

---

### Task 1: walk-menu-workflows

**Files:**
- Create: `web/admin/src/features/config-topology/lib/walk-menu-workflows.ts`
- Test: `web/admin/src/features/config-topology/lib/walk-menu-workflows.test.ts`

**Produces:** `walkMenuWorkflows(menu: MenuTree): string[]` — 递归 items 与 card.buttons，收集 `open_workflow` 的 `workflow_id`，去重。

- [x] **Step 1: 写失败测试**（根按钮 + 卡片嵌套 + 忽略其它 action）
- [x] **Step 2: 跑测确认失败** `cd web/admin && pnpm vitest run src/features/config-topology/lib/walk-menu-workflows.test.ts`
- [x] **Step 3: 实现并登记 vitest include**
- [x] **Step 4: 测试通过**
- [x] **Step 5: Commit** `test: walk menu open_workflow ids for topology graph`

### Task 2: buildLinkGraph + pathThrough

**Files:**
- Create: `lib/build-link-graph.ts`, `lib/path-through.ts` 及测试
- Consumes: `walkMenuWorkflows`、`caseRoutingTopics`/`edgeTopics`/`*References`

**Produces:**

```ts
type TopologyKind = 'platform' | 'case' | 'topic' | 'edge'
type GraphNode = {
  id: string
  kind: TopologyKind
  name: string
  to: string
  health: 'ok' | 'warn' | 'pending'
}
type GraphEdge = { id: string; source: string; target: string }
function buildLinkGraph(input, mode: { type: 'all' } | { type: 'focus'; kind; id }): { nodes: GraphNode[]; edges: GraphEdge[] }
function pathThrough(graph, nodeId: string): { nodes: Set<string>; edges: Set<string> }
```

节点 id：`platform:${id}` `case:${id}` `topic:${key}` `edge:${id}`。

- [x] **Step 1: 失败测试** — 四层边、多入口合并、全量丢孤立、缺入口有下游仍出现、焦点无边只留自己、health 来自 references、pending 当 reachability 缺失
- [x] **Step 2: 跑测失败**
- [x] **Step 3: 实现**
- [x] **Step 4: 通过**
- [x] **Step 5: Commit** `feat: build read-only config topology graph model`

### Task 3: LinkGraph 只读画布

**Files:** `topology-node.tsx`, `link-graph.tsx`, `link-graph.contract.test.ts`

- 自定义节点 + `StatusDot`（pending 不渲染 success）
- 四列坐标、固定高度、`fitView`/`Controls`、不可连接
- 点选 `pathThrough` 降透明；`Link`「打开详情」`nodrag`
- 合同：无 `onConnect` 持久化；含 `data-testid='link-graph'`

- [x] **Step 1: 合同测试先写**
- [x] **Step 2: 确认失败**
- [x] **Step 3: 实现画布（nodeTypes 在模块作用域，引入 xyflow css）**
- [x] **Step 4: 通过**
- [x] **Step 5: Commit** `feat: add read-only React Flow config topology canvas`

### Task 4: useTopologySource + 工作台卡片

**Files:** `use-topology-source.ts`, `topology-card.tsx`；Modify `workbench-data-board.tsx`, `workbench.contract.test.ts`, i18n

- 查询：channels/cases/topics/edges/presence + 每通道 `getMenu`
- **不要**对工作台批量 `checkChannelReachability`
- Card `h-[360px]`，`ErrorBanner`/`LoadingSkeleton` 隔离
- 合同：`data-testid='topology-card'` 在 data-board 顶部

- [x] **Step 1: 合同测试（卡片存在、不调用 checkChannelReachability）**
- [x] **Step 2: 失败**
- [x] **Step 3: 实现**
- [x] **Step 4: 通过**
- [x] **Step 5: Commit** `feat: show config topology card on workbench`

### Task 5: 详情 TopologyDialog

**Files:** `topology-dialog.tsx`；四类 detail-panel；对应 contract tests

- 按钮文案「拓扑」`data-testid='topology-open'`
- Dialog 标题「配置拓扑」，画布 focus 模式
- 不移除 `link-health-section`

- [x] **Step 1: 四页合同测试（按钮 + 仍含 link-health-section）**
- [x] **Step 2: 失败**
- [x] **Step 3: 实现 Dialog 与入口**
- [x] **Step 4: `pnpm tsc -b` + 相关 vitest**
- [x] **Step 5: Commit** `feat: open focused topology dialog from resource details`
