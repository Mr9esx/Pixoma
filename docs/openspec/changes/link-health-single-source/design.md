## Context

见 `proposal.md` 的 Why。通道不可用而工作流全绿，只是「各页各算绿黄」的一次发作；工作流页临时去看 `adapter_state` 挡不住队列、节点、拓扑、通道列表/详情继续各用各的信号。架构图：[`diagrams/link-health.html`](./diagrams/link-health.html)。

## Goals / Non-Goals

**Goals:**

- 绿黄只有一个主人：控制面组装器。
- 活路不变式可测：弄坏任一 hop、无旁路 → 下游不得 ok。
- 页面只渲染；新增信号只改组装器。
- 列表 / 详情 / 拓扑同色；缺输入不得当绿。

**Non-Goals:**

- 不把本次做成通道探测落库专项（落库只是让组装器读得到探测，若采用）。
- 不合并四类写模型。
- 不改 Task 调度/执行，Bot 不拦截开跑。
- 不做巡检推送；查询热路径不打外部 Bot API。

## Decisions

### 1. 组装器在 packaging，不新建写侧 BC

- **选择**：`internal/packaging/linkhealth` 只读编排 + `internal/httpapi/linkhealth` 提供 `GET /api/v1/link-health`。注入通道/菜单/Case/Topic/Edge+presence 的只读端口。
- **理由**：关联跨多个已有 BC；写仍留在原模块。健康是读模型。
- **备选**：继续堆前端 `*References` — 就是现在会漏页的结构。塞进 Catalog 或 Channel — 都会反向依赖别的 BC。

节点 id：`platform:{id}` / `case:{id}` / `topic:{key}` / `edge:{id}`。详情按 id 取值，不再为每个 Case 打 `menu-placements` 来算健康。

### 2. 不变式优先于点对点补丁

- **选择**：测试固定「完整活路」，用例覆盖入口不可用、双入口有活路、节点不可用。禁止把「通道 error → case warn」当唯一验收。
- **理由**：下一跳故障类型会变；规则不能绑死通道。
- **备选**：每发现一种漏再给对应 `*References` 加 if — 漏页会再现。

平台是否可作为起点：组装器按它能读到的全部信号合成（启停、适配器、以及它读得到的探测）。工作流页不得另持一套「只看进程」的定义。通道详情也不得只用探测、列表只用进程。

探测若今天只活在详情页 React Query 里，组装器就读不到 — 要么让探测对组装器可见（例如随通道读模型提供上次结论），要么在读不到时 pending。实现期选一种，原则是「组装器看得见或明确未就绪」，不是「做一次通道表迁移需求」。

### 3. 前端删除发明权

- **选择**：一个 `queryKeys.linkHealth`。`LinkHealthSection` / `StatusDot` / 拓扑只吃 DTO。删除运行时对 `caseReferences` / `topicReferences` / `edgeReferences` / `channelReferences` / `listHealthTone` 的调用（测试夹具可留）。
- **理由**：页面一调用本地判定，不变式就会再分叉。
- **备选**：保留纯函数与后端双算 — 两套规则会再漂。

拓扑点选高亮仍可在前端对返回的 `edges` 做 path-through，那是可视化，不是健康判定。

### 4. 架构文档

交付时更新 `bounded-contexts.md`、`data-model.md`、`overview.md`，并把架构图放到 `docs/architecture/diagrams/`。

## 场景（防回归，不是通道专项）

1. **唯一入口不可用** — 工作流/队列不得绿；节点自己活着可以绿，但不能把工作流洗绿。
2. **双入口有活路** — 工作流绿；黄打在坏掉的那一个平台引用上。
3. **唯一节点不可用，或运行信号组装器尚未读到** — 相关工作流/队列不得绿；未就绪不得当绿。

## Risks / Trade-offs

- [查询变慢] → 当前资源量小，一次进程内读。变大再加缓存。
- [presence 仍是内存] → 重启短暂 pending，与现网一致；不把心跳改造成落库专项。
- [与未归档拓扑 change 重叠] → 拓扑画布保留，只换数据源。

## Migration Plan

1. 组装器 + 不变式测试先红后绿。
2. HTTP 挂上；证明热路径不打外部探测。
3. 四类列表/详情/拓扑改吃查询；删掉页面本地绿黄。
4. 更新架构文档。
5. 回滚：前端可暂时再读旧列表（不作为交付态）。

## Open Questions

无。范围已从「修通道」改为「健康唯一主人」；探测如何对组装器可见留给实现，不升格为本 change 标题。
