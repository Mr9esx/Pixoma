## Context

见 `proposal.md` 的 Why。链路健康已在 `web/admin/src/features/link-health` 用列表数据算出上下游与绿/黄结论（`caseReferences` / `topicReferences` / `edgeReferences` / `channelReferences`），一期明确不做 React Flow。工作台左栏是 `WorkbenchDataBoard`（贡献图 + 概览卡 + 图表）。`@xyflow/react` 已在依赖中，任务分流编辑器刻意不把它引进表格文件。后台状态规范见 `docs/frontend/admin-status-rules.md`。

## Goals / Non-Goals

**Goals:**

- 一套只读画布，工作台卡片与详情弹窗共用。
- 图数据由现有 admin-api 列表在前端组装，健康结论复用 `link-health`。
- 四列确定性布局，容器有明确高度，遵循 `@xyflow/react` 用法（样式、稳定 `nodeTypes`、自定义节点、`StatusDot`）。

**Non-Goals:**

- 不新增后端聚合接口（组合查询过重时再单开 change）。
- 不把菜单入口画成节点，不画运行时任务，不在图上编辑配置。
- 不引入 dagre/elk；四列坐标足够。
- 不改 `docs/architecture/`（不改模块边界、表、事件、外部系统）。

## Decisions

### 1. 前端组装 `buildLinkGraph`，不新开 API

- **选择**：用已有 `listChannels`、菜单挂载、`cases`（含 routing）、topics、edges（订阅）、presence、channel reachability，在前端生成 nodes/edges。
- **理由**：关系已在健康模块算过，全图只是同一关系的可视化。
- **备选**：`GET /api/v1/stats/topology` — 图变大或多次往返成为问题时再做。

连线规则：

- 平台 → 工作流：该平台菜单存在指向该工作流的挂载（多入口合并为一条边）。
- 工作流 → 队列：routing 规则的 topic。
- 队列 → 节点：节点 `effective_topics` / `subscribe_topics` 含该 topic。

全量图：生成边之后丢掉度数为 0 的节点。焦点图：保留焦点节点，再保留所有与它同路径的节点与边。

### 2. 共用 `LinkGraph`，两个壳

- **工作台**：`Card` 包一层固定高度容器（如 `h-[360px]` 或 `min-h-[320px]`），放在 `WorkbenchDataBoard` 顶部，独立 `useQuery`，失败只用卡片内 `ErrorBanner`。
- **详情**：shadcn `Dialog` 大尺寸；四页同一个 `TopologyDialog`，传入 `focus: { type, id }`。
- **备选**：工作台也弹窗 — 已否决；全屏路由 `/topology` — 超出本次入口约定。

入口按钮文案用「拓扑」，不加说明句。详情页按钮放在页头动作区或健康区块标题旁，不新增解释性段落。

### 3. React Flow 只读 + 自定义节点

- 从 `@xyflow/react` 引入，并 `import '@xyflow/react/dist/style.css'`（全局一次）。
- `nodesDraggable={false}`、`nodesConnectable={false}`、`elementsSelectable={true}`；`Controls` + `fitView`；不做 MiniMap。
- 自定义节点：类型名 + 资源名 + `StatusDot`（`ok` / `warn`；数据未齐时不把 tone 设成 success）。
- `nodeTypes` 放组件外。点选写本地 state：计算「经过该 id 的路径」集合，路径外节点/边降透明度；`NodeToolbar` 或节点内 `nodrag` 按钮「打开详情」（`Link`）。
- 布局：`x = column * COL_W`，`y = index * ROW_H`，列顺序 platform / case / topic / edge。打开时 `fitView`。
- **备选**：复用任务分流编辑器的布局 hook — 禁止（一期已否决，那是可编辑路由图）。

### 4. 健康只复用，不平行实现一套规则

图节点的 `state` 必须来自现有 `*References(...).health`（平台用 `channelReferences`）。禁止在图模块里再写一份断点 if/else。

## Risks / Trade-offs

- [资源很多时四列重叠或画布拥挤] → 先缩放/平移；同列按名称排序；必要时以后再加碰撞或虚拟化。
- [多次列表请求] → 与详情页相同的 queryKey，工作台可命中缓存。
- [焦点图在「只有自己」时看起来像坏了] → 弹窗里仍画焦点节点，不加说明文案；无边即无连线。
- [xyflow 默认样式与设计体系冲突] → 自定义节点只用语义令牌与细线框；浮层 Dialog 可以有阴影，卡片本身无投影。

## Migration Plan

纯前端发布。回滚即去掉拓扑卡片、详情按钮与图组件。无数据迁移。

## Open Questions

无。规格未决项已在 Open 阶段对齐。
