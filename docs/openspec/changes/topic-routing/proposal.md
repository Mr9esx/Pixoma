## Why

当前调度在「健康 Edge 集合」上 round-robin 选一台机器，无法表达"不同用户/不同 Case 走不同机器池（如高规格机）"的投放策略；同一台机器面对高并发领取时也缺少"任务只被一台消费"的原子保证，失败与节点宕机后的任务回收没有明确闭环。归档的 `edge-agent-topic-routing` 已将 Topic 分流明确延后，现在补上这一层：Topic 作为逻辑投递目标，Case 通过可扩展条件协议把任务投放进不同 Topic，节点订阅 Topic 消费。

## What Changes

- 新增 **Topic 实体**：稳定 `key`、显示名、启用状态；管理端可 CRUD；应用启动初始化时若不存在则自动创建默认 Topic（`default`），默认 Topic 不可删除。
- 新增 **Case 路由投放配置**：一个 Case 声明一对 N Topic 的投放规则（条件规则 + 目标 Topic），无规则命中时回退默认 Topic；Case 协议与 admin API 支持读写该配置。
- 新增 **可扩展条件协议**：投放条件为声明式 JSON 规则（`{field, op, value}`，支持 `and`/`or` 组合）；`field` 来自可注册的属性提供方（本期示例 `user.is_premium` / `case.category` / `case.tags`），每个属性带 JSON Schema 描述；引擎按 schema 求值，新增条件类型只注册 provider + schema，不改引擎代码。
- **调度改为按 Topic 选路**：任务创建/调度时求值路由规则 → 选中目标 Topic → 在订阅该 Topic 的可用节点上抢占领取；不再对全部健康实例 round-robin。
- **节点订阅 Topic**：节点（Edge）可订阅一个或多个 Topic；未配置订阅时默认订阅 `default`；控制面记录并暴露该绑定。
- **原子领取与生命周期**：同一 Topic 下多节点高并发 claim 时，同一任务只被一台消费（原子抢占 + 租约）；节点执行失败后任务按有界重试回到 Topic 可再领取；节点长时间无进展（租约/心跳超时）后任务被回收重回 Topic 或按策略失败收口，不永久卡死。
- **BREAKING**：Agent claim 接口由按 `instance_id` 领取改为按节点订阅的 Topic 领取；Task 记录实际 `dispatch_topic` 与领取/重试信息。

## Capabilities

### New Capabilities
- `condition-protocol`: 可扩展投放条件协议——属性提供方注册、声明式 JSON 规则求值（and/or、比较运算）、JSON Schema 驱动的校验与条件表单描述；新增条件类型不改引擎代码。
- `topic-routing-config`: Case 的路由投放配置（一对 N Topic、条件规则、默认 Topic 回退）的协议、持久化与校验。

### Modified Capabilities
- `topic-admin`: Topic CRUD 与默认 Topic（此前"本期不实现"，改为 MUST 交付）。
- `dispatch-topic-routing`: 按条件表达式选 Topic 投放（此前 MUST NOT，改为 MUST）。
- `edge-agent`: Edge 配置订阅 Topic 列表，未配置默认订阅 `default`，长轮询按订阅 Topic 领取。
- `agent-pull-dispatch`: 按 Topic 的原子 claim（同一任务只被一台消费）、租约/心跳、失败重回 Topic、租约过期回收。
- `task-orchestrator`: 调度流程改为"路由求值 → 选 Topic → 订阅节点抢占"，无可用节点保持 pending 并记录原因。
- `workflow-protocol`: Case 协议增加路由投放配置字段。
- `case-admin-api`: Case 创建/更新支持路由投放配置。
- `task-persistence`: Task 增加 `dispatch_topic`、重试次数与领取租约等字段。
- `comfy-instance-pool`: 节点记录增加订阅 Topic 绑定（含默认）。
- `platform-bootstrap`: 启动初始化自动创建默认 Topic。

## Impact

- 领域/包：`internal/runtime`（domain/application/infrastructure）、`internal/platform/edge`（订阅绑定）、`internal/catalog`（Case 路由配置）、`internal/httpapi`（Topic 管理、Case 路由、Edge 订阅、claim 按 Topic）、`apps/edge-agent`（订阅配置）、`internal/sharedkernel`（事件/Topic 命名与 DTO）、`internal/packaging`（调度装配）。
- 持久化：新增 `topics` 表；`edges` 增加订阅 Topic 字段；`tasks` 增加 `dispatch_topic` / 重试 / 租约字段；Case doc 增加路由配置。
- API：新增 Topic 管理 API；`/agent/v1/jobs/claim` 语义扩展为按节点订阅 Topic 领取（BREAKING）。
- 行为：调度从"全部健康实例 RR"变为"按条件路由到 Topic，再在订阅节点上原子抢占"；失败/超时任务重回 Topic。
- 依赖：条件协议自研声明式 JSON 规则，不引入新外部依赖；与 React Flow 编辑器（`task-flow-editor` change）通过 API 与条件 schema 对接。
