## Context

`topic-routing` 已定义 Case `routing`（有序 rules：`{when, topic}`）、Topic 管理 API、节点订阅 API 与条件目录 API（`/api/v1/routing/attributes`）。本 change 是纯前端：在 `web/admin`（React + TanStack Router/Query + shadcn/ui，pnpm）上落地 React Flow 画布与 schema 驱动表单，只消费上述 API，不改后端。

## Goals / Non-Goals

**Goals:**
- 管理员能直观编辑/查看「Case → 条件 → Topic」单步分流图，并保存为合法 `routing` 载荷。
- 新增条件属性后，前端表单自动支持，零代码改动。
- Topic 管理页与节点订阅编辑随画布一起可用，形成完整后台闭环。

**Non-Goals:**
- 不改后端调度/求值（依赖 `topic-routing` 先落地其 API）。
- 不做 ComfyUI workflow 画布（现有表单式 Case 工作流编辑器保持）。
- 不做多步流水线编辑（单步分流语义由 `topic-routing` 固定）。

## Decisions

### D1. 画布库：`@xyflow/react`（React Flow 12）

- 引入官方包 `@xyflow/react`，按需导入组件与样式，避免引入全套示例代码；包体积由构建分析确认。
- **备选**：自研 SVG/HTML 连线编辑器——与"可编辑查看"目标相比工程量与维护成本高，拒绝。

### D2. 画布数据模型与序列化

- 内部图模型：固定拓扑 = 1 个 Case 起始节点 + N 个条件分支节点 + N 个 Topic 目标节点 + 默认 Topic 回退（虚线边，只读展示）。
- 分支顺序：以显式 `order` 数组（而非画布坐标）作为 `routing.rules` 顺序真相源；自动布局按顺序纵向排布，用户可拖动微调，保存时重排顺序数组。
- 序列化：`{rules: [{when: <条件 JSON>, topic: <key>}, ...]}`；回读：按 `rules` 渲染节点与连线，未知/禁用 Topic 标红并阻断保存。
- 校验发生在保存前：条件合法性（复用条件目录 schema）、目标 Topic 存在于已启用列表、每条规则非空。

### D3. 条件表单渲染器（schema 驱动）

- 启动/进入编辑器时经 TanStack Query 拉取 `/api/v1/routing/attributes`，按属性 key 索引缓存。
- schema → 控件映射：`enum` → Select（含 UI 标签与描述）、`boolean` → Switch、`number` → NumberInput、`string` → TextInput；未知类型回退文本输入并在 UI 上弱提示。
- 组合编辑：`and`/`or` 嵌套行编辑器，支持增删子条件；保存输出协议 JSON。
- 未来新增属性 = 后端注册 provider + schema，前端自动获得控件（spec 验收点）。

### D4. 集成位置

- Case 创建/编辑表单新增「任务分流」区块（与导入/输入/输出区块并列），保存时 `buildPayload` 统一带上 `routing`；Case 详情页以只读画布展示。
- 计算节点详情页新增「订阅 Topic」多选（数据源 `/api/v1/topics`；空 = 默认 Topic 语义），保存走 `PATCH /api/v1/edges/{id}`。
- 新增 Topic 管理页：列表 + 新建/编辑/启用禁用（对话框表单，风格与计算节点页一致），`default` 不提供删除操作。
- 侧栏新增 Topic 菜单项（Dashboard、实例、Case、渠道、Task、User、Session、Topic），i18n zh/en 成对。

### D5. 状态与契约测试

- `draft.routing` 由画布实时推导，保存时统一序列化；错误定位到具体分支卡片。
- 沿用现有 contract 测试模式：锁定「画布区存在于 Case 表单」「条件表单不出现未注册属性」「默认 Topic 不可删」「节点订阅空=默认」等关键约束。

## Risks / Trade-offs

- [React Flow 增加包体积与学习成本] → 按需导入、限定使用子集；构建体积纳入验收。
- [条件 schema 演进导致渲染缺类型] → 渲染器严格 schema 驱动 + 文本输入兜底，禁止硬编码属性名。
- [与既有 Case 表单冲突] → 独立区块、统一保存入口；contract 测试守住既有导入/绑定行为不被破坏。
- [`topic-routing` API 未就绪时页面不可用] → 画布区显示空态/禁用并给出前置依赖提示（两个 change 顺序交付）。

## Migration Plan

- 纯前端增量：新增依赖、路由与区块，无数据迁移；与 `topic-routing` 联调通过后再发布。
- 回滚：移除菜单项与画布区块即可，后端 API 不受影响。

## Open Questions

- 画布自动布局细节（纵向间距、分支横向排布、缩放范围）——build 期按可用性调优，不改行为契约。
