---
comet_change: config-topology-overview
role: technical-design
canonical_spec: openspec
---

# 配置拓扑总览 深度技术设计

## 1. 目标

把已有的配置关系画成只读四层图：消息平台 → 工作流 → 任务队列 → 计算节点。工作台左栏一张卡片里直接铺全量画布；四类详情页用同一套图画「过当前资源的路径」。点选只高亮路径，不跳走；「打开详情」才进详情。节点绿/黄与链路健康同一套结论。

Canonical 行为见 OpenSpec delta：`docs/openspec/changes/config-topology-overview/specs/`。高层选型见该 change 的 `design.md`。本文是实现细化。

## 2. 现状

| 块 | 现状 |
|---|---|
| 关系与健康 | `features/link-health/lib/references.ts` |
| 工作台左栏 | `features/dashboard/workbench-data-board.tsx` |
| 详情 | cases / channels / topics / edges 的 `*-detail-panel.tsx` |
| 菜单 | `getMenu(channelId)`，`open_workflow` + `workflow_id`；卡片按钮可嵌套 |
| 挂载列表 | `getCaseMenuPlacements` 按工作流，不适合全图 |
| xyflow | 依赖已有；任务分流编辑器不把它引进表格文件 |

## 3. 模块切分

新建 `web/admin/src/features/config-topology/`：

```text
config-topology/
  lib/walk-menu-workflows.ts   # 菜单树 → workflow_id 集合
  lib/build-link-graph.ts      # 输入 → GraphModel
  lib/path-through.ts          # 点选 id → 路径上的 node/edge id
  use-topology-source.ts       # 组合查询
  topology-node.tsx            # 自定义节点 + StatusDot
  link-graph.tsx               # React Flow 只读画布
  topology-card.tsx            # 工作台 Card
  topology-dialog.tsx          # 详情 Dialog
  *.test.ts / *.contract.test.ts
```

不把图逻辑写进 `link-health-section.tsx`。健康规则仍只住在 `references.ts`。

## 4. 数据流

```text
useTopologySource
  listChannels + getMenu(each)
  listCases / listTopics / listEdges / listPresence
  各 channel reachability（与频道详情同一 queryKey）
        │
        ▼
walkMenuWorkflows(menu) → channelId → caseId[]
        │
        ▼
buildLinkGraph(source) → { nodes, edges }  全量（度数 0 丢掉）
        │
        ├─ TopologyCard：直接渲染
        └─ TopologyDialog(focus)：filter 过 focus 的路径；无边时仍留 focus 节点
        │
        ▼
LinkGraph：四列 x/y、点选 path-through、打开详情 Link
```

菜单边：递归 `items` 与 `action.card.buttons`，收集 `type === 'open_workflow'` 且带 `workflow_id` 的边。同一平台到同一工作流多入口合并为一条边。

健康：对每个仍在图上的实体调用对应 `*References`，把 `health.state` 写进 node.data。`useTopologySource` 未齐（loading 或任一必需 query error）时：卡片走骨架/ErrorBanner；节点不得把 `StatusDot` 调成 `problems === 0`。

## 5. 查询

`useTopologySource` 使用已有 `queryKeys`（`channels.all`、`channels.menu(id)`、`cases.all`、`topics.all`、`edges.all`、`edges.presence`、频道 reachability 与详情页相同）。`useQueries` 拉每个通道的菜单。失败隔离：拓扑卡片自己的 error；不拖垮贡献图。

不新增 admin-api。通道数预期很小；若以后变多再单开聚合接口。

## 6. React Flow

- 从 `@xyflow/react` 引入，应用入口或本 feature 模块顶层 `import '@xyflow/react/dist/style.css'`。
- 父级明确高度：工作台卡片 `h-[360px]`；Dialog 内容区 `h-[min(70vh,640px)]`。
- `nodeTypes` 定义在模块作用域。自定义节点含类型名、资源名、`StatusDot`（`problems` = breakpoints.length；`healthReady === false` 时不渲染 success 点）。
- `nodesDraggable={false}` `nodesConnectable={false}`；`Controls` + 初始 `fitView`。无 MiniMap。
- 点选：`onNodeClick` 更新 `selectedId`；`pathThrough(graph, selectedId)` 得到高亮集合；路径外 opacity 降低。不 `navigate`。
- 「打开详情」：节点内 `nodrag` 的 `Link`（`data.to` 已有：`/channels/:id` `/cases/:id` `/topics/:key` `/edges/:id`）。弹窗内点击后 Dialog 随路由卸载即可。
- 布局：`COL = [platform, case, topic, edge]`，`x = col * 220`，`y = index * 72`，同列按名称排序。

禁止使用 task-flow 编辑器的布局/连线 hook。

## 7. 页面接入

- 工作台：`WorkbenchDataBoard` 最上方渲染 `TopologyCard`。
- 详情：四页 header 动作放 `Button variant="outline"` 文案「拓扑」，打开 `TopologyDialog`。不改动 `LinkHealthSection`。

文案：`topology.title` = 配置拓扑，`topology.open` = 拓扑，`topology.openDetail` = 打开详情。无说明句。

## 8. 测试

| 层 | 覆盖 |
|---|---|
| `walk-menu-workflows` | 根按钮、卡片嵌套、忽略非 open_workflow |
| `build-link-graph` | 四层边、多入口合并、全量丢度数 0、缺入口但有下游仍出现、焦点无边只留自己、health 来自 references |
| `path-through` | 点队列高亮上下游；点无关节点不在集合 |
| 合同 | 工作台含 `data-testid=topology-card`；四页含打开按钮；详情仍含 `link-health-section`；`LinkGraph` 无 `onConnect` 写配置 |
| 登记 | 新测试加入 `vitest.config.ts` include |

浏览器：工作台能平移看到连线；点节点不高亮以外的路径发灰；打开详情进对的页。弹窗从工作流打开不含无关工作流。

## 9. 边界

- Topic 节点 id 用 `topic:${key}`，避免与 edge id 撞车。
- 健康 `bad`（实体缺失）在图上按黄档 `StatusDot`（规范只有绿/黄）。
- 空全图（全是孤立点）：卡片画布空白，无解释文案。
- `prefers-reduced-motion`：不高亮动画。

## 10. 发布与回滚

纯前端。回滚删除 `features/config-topology` 与四处入口。不改架构文档。
