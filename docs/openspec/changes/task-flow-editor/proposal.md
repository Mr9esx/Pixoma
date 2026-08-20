## Why

`topic-routing` 引入的 Case→条件→Topic 投放规则目前只能靠 JSON/表单表达，管理员无法直观理解"任务从 Case 出发按什么条件流向哪个 Topic"；条件属性又要求随协议演进自动获得表单，不能每加一个属性就改一遍前端代码。需要一个基于 React Flow 的可视化编辑器：画布编辑/查看任务分流，条件表单由 schema 自动渲染。

## What Changes

- 引入 **React Flow（@xyflow/react）** 到 `web/admin`，实现任务分流画布：Case 起始节点 → 条件判定（分支）→ Topic 目标节点，编辑/查看单步分流语义。
- 新增 **条件表单渲染器**：消费 `topic-routing` 的 `GET /api/v1/routing/attributes` 条件目录，按属性 JSON Schema 自动渲染条件配置表单（类型、枚举、说明），新增属性无需改前端代码。
- 画布产出 **路由配置载荷**：保存时序列化为 `topic-routing` Case `routing` 字段（有序规则列表），并从 API 回读渲染已有配置；无路由配置的 Case 显示默认 Topic 回退语义。
- **Topic 管理页**：Topic 列表/新建/编辑/启用禁用（消费 `/api/v1/topics`），为画布提供可选目标。
- **节点订阅编辑**：计算节点详情支持查看/编辑订阅 Topic 列表（空 = 默认 Topic）。
- 只读查看与编辑模式切换；非法配置（无目标 Topic、条件缺字段、引用禁用 Topic）阻止保存并给出可读错误。

## Capabilities

### New Capabilities
- `task-flow-canvas`: React Flow 任务分流画布——Case→条件→Topic 图的渲染、编辑、校验与序列化（生成/消费 Case `routing` 配置）。
- `condition-form-renderer`: 按条件属性 schema 自动渲染条件配置表单（消费 `/api/v1/routing/attributes`，新增属性不改前端代码）。

### Modified Capabilities
- `admin-web-shell`: 侧栏/入口与 i18n 集成任务分流与 Topic 管理入口。
- `admin-resource-pages`: Case 编辑新增任务分流画布区；计算节点详情支持查看/编辑订阅 Topic；新增 Topic 管理页。

## Impact

- 前端：`web/admin` 引入 `@xyflow/react`；新增 `features/task-flow`（画布、条件表单、序列化/校验）、`features/topics`（Topic 管理页）；Case 表单与计算节点详情集成；i18n（zh/en）成对文案。
- API 消费：`/api/v1/topics`、`/api/v1/routing/attributes`、Case 创建/更新/详情的 `routing` 字段、`/api/v1/edges/{id}` 的 `subscribe_topics`。
- 依赖：`@xyflow/react`（React Flow）。
- 非目标：不改后端调度/求值（属 `topic-routing`）；不做 ComfyUI workflow 画布编辑器（现有表单式 Case 工作流编辑器保持）；不做多步流水线编辑。
