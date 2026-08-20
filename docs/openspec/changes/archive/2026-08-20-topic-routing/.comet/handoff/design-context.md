# Comet Design Handoff

- Change: topic-routing
- Phase: design
- Mode: compact
- Context hash: 0f6199f4d295ed146081c80fa8d490babadca2528e761f101b19eb4babcf1ee5

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## docs/openspec/changes/topic-routing/proposal.md

- Source: docs/openspec/changes/topic-routing/proposal.md
- Lines: 1-39
- SHA256: 9e6a13114c9ce66864319bbdbc952741eba13ab1aa7b63046ec2aff60fb3eada

```md
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

```

## docs/openspec/changes/topic-routing/design.md

- Source: docs/openspec/changes/topic-routing/design.md
- Lines: 1-84
- SHA256: ad5e510474e11155f27e71c8d7c6bde169891d16f96721e603cb0f182ec9d98a

[TRUNCATED]

```md
## Context

现状（见 proposal.md）：调度在健康 Edge 集合上 round-robin 选一台，`/agent/v1/jobs/claim` 按 `instance_id` 领取，租约/心跳/回收已有雏形；`edges`、`tasks`、Case doc 均无 Topic 概念。约束：默认部署不依赖 Redis 作为跨进程派发；DB 是 Task 执行态唯一真相源；Mock 端到端必须保持可通。

## Goals / Non-Goals

**Goals:**
- Topic 作为逻辑投递目标，贯穿「Case 路由配置 → 调度求值 → 节点订阅 → 原子领取 → 失败/超时回收」全链路。
- 条件协议数据驱动：新增条件类型只注册 provider + schema，引擎与前端表单都不改代码。
- 同一任务并发领取下只被一台消费的原子保证；失败与节点宕机有明确回收闭环。

**Non-Goals:**
- 不做多步任务流水线（任务只投一个 Topic 执行完）。
- 不做用户等级/充值真实字段与会员计费（仅协议示例属性）。
- 不做按机器规格自动选路（规格差异由管理员订阅到不同 Topic 体现）。
- 不做 React Flow 编辑器（归 `task-flow-editor` change，本 change 只提供其消费的 API 与条件 schema）。

## Decisions

### D1. Topic 实体与数据落点

- 新增 `topics` 表：`key`（唯一、稳定、不可变）、`name`（显示名）、`enabled`、时间戳。默认 Topic key 为 `default`，启动幂等创建，禁止删除。
- `edges` 增加 `subscribe_topics_json`（字符串列表，空 = `["default"]`）。
- `tasks` 增加 `dispatch_topic`、`attempts`；复用 `edge_id`（领取节点）与 `lease_until`（租约）。
- Case doc（`catalog_cases.doc_json`）增加 `routing`：`{"rules": [{"when": <condition>, "topic": "<key>"}, ...]}`；无命中回退系统默认 Topic。

### D2. 条件协议：声明式规则 + 属性提供方（关键设计点）

- **规则语法**：叶子 `{field, op, value}`；组合 `{"and":[...]}` / `{"or":[...]}`。内置运算 `eq/ne/in/gt/gte/lt/lte/exists`，运算集在引擎内固定。
- **属性提供方注册**：按命名空间注册（`user.*`、`case.*`、`input.*`），每个属性提供方实现 `ListAttributes()`（返回 `AttributeDescriptor`：key、JSON Schema、UI 标签、上下文来源）与取值函数。本期内置 `user.is_premium`、`case.category`、`case.tags`；`user.level`/`user.paid` 仅作为协议示例保留（不注册真实 provider）。
- **引擎职责**：`Validate(rule)` 检查字段已注册、运算合法、value 与 schema 类型一致；`Evaluate(ctx, rule)` 按 provider 取值求值，缺失上下文按 false（`exists` 除外）。新增条件 = 注册 provider + schema，引擎零改动。
- **契约出口**：Admin API 暴露条件属性目录 `GET /api/v1/routing/attributes`（key、schema、UI 标签），供 `task-flow-editor` 自动渲染条件表单。
- **备选**：引入 CEL/expr 表达式库——更通用但增加外部依赖，且"新增条件不改引擎"仍需要注册函数，收益不成比例；自研 JSON 规则足够覆盖本期组合需求。

### D3. 调度：先路由后抢占

- Orchestrator 调度 pending Task：读 Case 路由配置 → 求值条件（首个命中即投）→ 得到 `dispatch_topic`（无命中 = `default`）→ 组装 job 包 → 将 Task 置为 `queued` 且只携带 `dispatch_topic`（**不**预绑定 edge_id）。
- 无可用节点判断移到领取侧：目标 Topic 无在线订阅节点时任务停在 `queued` 可领取态，靠 claim 空返回与观测暴露；调度器不再对所有健康实例 RR。

### D4. 原子领取：DB 条件更新保证一机一任务

- `GET /agent/v1/jobs/claim?edge_id=...&wait=...`：服务端按 `edge_id` 读取节点订阅集合，从 `queued` 且 `dispatch_topic ∈ 订阅集合`（且未被有效租约持有）的任务中原子抢占一个：
  - SQLite：单写者 + 事务内条件 UPDATE 保证串行；
  - MySQL/Postgres：`UPDATE ... WHERE status='queued' AND dispatch_topic=? AND (edge_id IS NULL OR lease_until < now) ORDER BY id LIMIT 1 RETURNING id` 或等价条件更新，**一行只被一个事务改到**。
- 领取成功写 `edge_id` + `lease_until`；长轮询期间周期性重扫。并发竞争落败返回空结果，不重复发放。
- **备选**：Redis Streams 消费组/XADD 抢占——需要 Redis 成为默认依赖，与现有「DB 是唯一真相源、默认不依赖 Redis」约束冲突，拒绝。

### D5. 失败重回 Topic 与超时回收

- 失败终态：Edge 上报 failed 后，`attempts < max_attempts`（默认 3，可配）时 Task 回到 `queued`（清 edge_id/租约，`attempts+1`，可选退避），可被订阅该 Topic 的节点再领；达上限收敛为最终 failed。
- 节点宕机：复用并强化既有 ReconcileStale——租约过期且无终态的 `queued`/`running` 任务重新可领取（清持有者）或按重试上限收口；心跳续约沿用现有 Agent API。
- 复用 StormGuard 的限流/退避/熔断与分池，避免回收风暴。

### D6. API 形状

| 资源 | 端点 | 说明 |
|---|---|---|
| Topic | `GET/POST /api/v1/topics`；`GET/PUT/DELETE /api/v1/topics/{key}` | DELETE 对 `default` 拒绝；禁用影响新规则/新订阅 |
| 节点订阅 | `PATCH /api/v1/edges/{id}` 增加 `subscribe_topics` | 空数组 = 默认 Topic；列表/详情返回有效订阅集合 |
| Case 路由 | 既有 `case-admin-api` 创建/更新/详情载荷增加 `routing` | 校验规则与 Topic 存在性 |
| 条件目录 | `GET /api/v1/routing/attributes` | 供编辑器渲染条件表单 |
| Claim | `GET /agent/v1/jobs/claim`（**BREAKING**） | 语义从按 instance 变为按节点订阅 Topic 集合 |

## Risks / Trade-offs

- [规则误配导致任务整体进错 Topic] → 保存校验（Topic 存在/启用、条件合法）；默认 Topic 兜底；审计记录规则变更。
- [并发领取在 DB 上的热点竞争] → 条件更新原子 + 领取批大小/等待上限可配；并发契约测试覆盖。
- [节点离线时任务滞留] → 租约过期回收 + pending/queued 观测与告警。
- [失败重试风暴] → 有界重试 + 退避 + 复用 StormGuard；重试次数计入 Task 可观测。
- [旧 Edge 未升级导致 claim 语义不兼容] → 按节点订阅集合兜底（未订阅=default），文档标注 BREAKING；升级顺序：控制面先升级，Edge 后升级。

## Migration Plan

1. 加 `topics` 表并幂等种入 `default`；`edges.subscribe_topics_json`、`tasks.dispatch_topic/attempts` 加列（GORM AutoMigrate）。
2. 存量可领取/未领取任务回填：`dispatch_topic` 空值按 `default` 处理（查询层兜底，不强制全量 UPDATE）。
3. 控制面先上线 Topic 调度与 claim 新语义；旧 Edge 以其订阅集合（空=default）继续工作。
4. Edge 升级为显式声明 `subscribe_topics`（不声明则默认 default，行为不变）。
5. 回滚：若 Topic 路由出问题，调度开关可回退到"全部按 default Topic 领取"，不依赖表达式求值。

## Open Questions

```

Full source: docs/openspec/changes/topic-routing/design.md

## docs/openspec/changes/topic-routing/tasks.md

- Source: docs/openspec/changes/topic-routing/tasks.md
- Lines: 1-51
- SHA256: e763d759ccef6fe1dbad630df82648c2543ebc6bf6cd3d3a53239cf5a3b24b83

```md
## 1. 数据模型与迁移

- [ ] 1.1 新增 `topics` 表（key 唯一、name、enabled、时间戳）与 GORM 模型，纳入 AutoMigrate
- [ ] 1.2 `edges` 增加 `subscribe_topics_json` 列；读取时空值按 `["default"]` 处理
- [ ] 1.3 `tasks` 增加 `dispatch_topic` 与 `attempts` 列；复用 `edge_id`/`lease_until` 表示持有节点与租约
- [ ] 1.4 Case 协议模型与 doc_json 增加可选 `routing`（rules: [{when, topic}]），旧文档无该字段兼容读写
- [ ] 1.5 控制面启动幂等种入默认 Topic `default`（已存在跳过，删除被拒绝）

## 2. 条件协议

- [ ] 2.1 定义 `AttributeDescriptor`（key、JSON Schema、UI 标签、上下文来源）与 provider 注册表（按 user/case/input 命名空间）
- [ ] 2.2 内置 provider：`user.is_premium`、`case.category`、`case.tags`（含取值函数与 schema）
- [ ] 2.3 规则 JSON 解析与校验：叶子 {field, op, value}、and/or 组合；未知字段/非法 op/类型不符给出可定位错误
- [ ] 2.4 求值引擎：eq/ne/in/gt/gte/lt/lte/exists，缺失上下文按 false（exists 除外）
- [ ] 2.5 单元测试：合法/非法规则、组合求值、新 provider 注册后不改引擎即可用（协议扩展性测试）

## 3. Topic 领域与 Admin API

- [ ] 3.1 Topic 仓储与 service：CRUD、启用/禁用、default 不可删
- [ ] 3.2 `GET/POST /api/v1/topics`、`GET/PUT/DELETE /api/v1/topics/{key}` 及校验（key 命名、禁用态引用）
- [ ] 3.3 节点订阅绑定：`PATCH /api/v1/edges/{id}` 支持 `subscribe_topics`，列表/详情返回有效订阅集合
- [ ] 3.4 Case 路由配置读写：case-admin-api 创建/更新/详情载荷支持 `routing`，保存时校验 Topic 存在且启用、条件协议合法
- [ ] 3.5 条件目录 API `GET /api/v1/routing/attributes` 返回属性 key/schema/UI 标签

## 4. 调度改造

- [ ] 4.1 Orchestrator 调度流程改为：读 Case 路由 → 求值首个命中规则 → 确定 `dispatch_topic`（无命中 = default）
- [ ] 4.2 调度不再预绑定 edge_id：Task 置为 `queued` 且携带 `dispatch_topic`，prep job 包（job_ref）语义保持
- [ ] 4.3 SchedulePending/事件触发双通道沿用，按 Topic 可领取态推进

## 5. 原子领取

- [ ] 5.1 `GET /agent/v1/jobs/claim` 改为按节点订阅 Topic 集合扫描可领取任务（BREAKING，edge_id 用于读绑定）
- [ ] 5.2 原子抢占：条件 UPDATE（SQLite 事务 / MySQL/Postgres RETURNING 等价语义）保证同一任务只被一台领取，写入 edge_id + lease_until
- [ ] 5.3 长轮询重扫与空返回；并发竞争落败返回空结果
- [ ] 5.4 并发互斥契约测试：两节点同时 claim 同一 Topic 单任务，恰好一成功一空

## 6. 失败重回 Topic 与超时回收

- [ ] 6.1 failed 有界重试：attempts < max（默认 3）时任务回 `queued`（清持有者、attempts+1、可选退避），超限收敛最终 failed 并记录原因
- [ ] 6.2 ReconcileStale 增强：租约过期且无终态的任务重新可领取或按重试上限收口；心跳续约保持
- [ ] 6.3 复用 StormGuard 限流/退避/熔断与分池，防止回收与重试风暴
- [ ] 6.4 单元测试：失败重回、宕机回收、超限收口、重复 status 不重复副作用

## 7. 端到端与兼容

- [ ] 7.1 allinone/Mock 主路径：默认 Topic 下任务端到端成功
- [ ] 7.2 split 主路径：Edge 配置订阅多个 Topic，按订阅领取并执行成功
- [ ] 7.3 旧数据兼容：空 `dispatch_topic` 按 default 处理；未配置订阅的旧 Edge 按 default 领取
- [ ] 7.4 文档同步：data-model、runtime、task-data-walkthrough、README（Topic、订阅、claim 语义）
- [ ] 7.5 全量 `go test ./...` 与关键冒烟通过

```

## docs/openspec/changes/topic-routing/specs/agent-pull-dispatch/spec.md

- Source: docs/openspec/changes/topic-routing/specs/agent-pull-dispatch/spec.md
- Lines: 1-23
- SHA256: 63bce7c14bff4975520cee2db86a3de824d91f4ba031b0240a147e136989bbb7

```md
## ADDED Requirements

### Requirement: 同一任务只被一台节点消费
控制面 MUST 保证多个节点并发向同一 Topic 领取时，同一 Task 最多被一台节点成功领取（原子抢占）；领取成功后 MUST 记录持有节点与租约。并发竞争落败的领取 MUST 返回空结果（或等价 no-content），MUST NOT 重复发放同一任务。

#### Scenario: 并发领取互斥
- **WHEN** 两台节点同时向同一 Topic 领取且该 Topic 只有一个可领取任务
- **THEN** 恰好一台节点成功领取；另一台获得空结果，且该任务未被重复分配

### Requirement: 失败重回 Topic 与有界重试
节点执行失败（终态 failed）时，任务 MUST 按有界重试策略重新进入其 `dispatch_topic` 可被再次领取；重试次数达到上限后 MUST 收敛为最终 failed 并记录原因。节点领取后长时间无进展（未续约且租约过期、无终态上报）时，任务 MUST 被回收重新可领取或按策略失败收口，MUST NOT 永久卡在已领取态。

#### Scenario: 失败后重回 Topic
- **WHEN** 节点上报 failed 且重试次数未达上限
- **THEN** 任务重新进入 dispatch_topic 可领取队列，重试计数 +1

#### Scenario: 节点宕机租约回收
- **WHEN** 任务已被领取、节点未续约、租约过期且无终态上报
- **THEN** 任务重新可被领取（或按重试上限失败收口）

#### Scenario: 超限失败收口
- **WHEN** 任务重试次数达到上限
- **THEN** 任务收敛为 failed 并记录含重试次数的原因

```

## docs/openspec/changes/topic-routing/specs/case-admin-api/spec.md

- Source: docs/openspec/changes/topic-routing/specs/case-admin-api/spec.md
- Lines: 1-12
- SHA256: 8b13922ae030411f04c79f755afe41061eb419bc13c8aa24c8780ce925dfe81d

```md
## ADDED Requirements

### Requirement: Case 路由投放配置读写
Case 创建/更新 API MUST 接受可选的路由投放配置字段，并按 `topic-routing-config` 校验；非法配置 MUST 被拒绝并返回可定位错误。Case 详情接口 MUST 返回路由配置。

#### Scenario: 保存合法路由配置
- **WHEN** 更新 Case 携带引用已启用 Topic 的路由规则
- **THEN** 更新成功且详情返回该路由配置

#### Scenario: 非法路由配置被拒绝
- **WHEN** 更新 Case 携带引用不存在 Topic 的路由规则
- **THEN** 更新被拒绝，返回含 Topic 相关信息的错误

```

## docs/openspec/changes/topic-routing/specs/comfy-instance-pool/spec.md

- Source: docs/openspec/changes/topic-routing/specs/comfy-instance-pool/spec.md
- Lines: 1-12
- SHA256: 0bc0100269476a59399db1a176e0cb13e129f5b69d17c77e36106dbfe7efa078

```md
## ADDED Requirements

### Requirement: 节点订阅 Topic 绑定
节点记录 MUST 支持声明订阅 Topic 列表；未声明时 MUST 按默认 Topic（`default`）处理。节点管理 API MUST 支持读取与更新该绑定；更新后 MUST 影响后续按 Topic 领取（进行中任务按既有租约/对账路径处理）。

#### Scenario: 节点绑定多个 Topic
- **WHEN** 节点 `gpu-1` 更新订阅为 `["default","fast-gpu"]`
- **THEN** 该节点后续可领取这两个 Topic 的任务

#### Scenario: 未绑定按默认
- **WHEN** 查询未设置订阅的节点
- **THEN** 其有效订阅集合等于 `["default"]`

```

## docs/openspec/changes/topic-routing/specs/condition-protocol/spec.md

- Source: docs/openspec/changes/topic-routing/specs/condition-protocol/spec.md
- Lines: 1-38
- SHA256: 5c4eef1f88cd6a2b474c6b2bd7f1feb654f58c0f88e49b71387557a10493d233

```md
## Purpose

定义可扩展的投放条件协议：属性提供方注册、声明式 JSON 规则结构与求值语义、以及由 JSON Schema 驱动的校验与表单描述，使新增条件类型无需修改任务分流引擎代码。

## ADDED Requirements

### Requirement: 条件规则为声明式 JSON
系统 MUST 以声明式 JSON 表示投放条件。单个条件 MUST 为 `{field, op, value}`；复杂条件 MUST 支持 `{and: [...]}` / `{or: [...]}` 组合。`field` MUST 引用已注册属性提供方提供的属性 key；`op` MUST 为协议内置运算（至少 `eq`、`ne`、`in`、`gt`、`gte`、`lt`、`lte`、`exists`），`value` 的类型 MUST 与该属性 schema 定义一致。

#### Scenario: 合法规则可求值
- **WHEN** 规则为 `{"and":[{"field":"user.is_premium","op":"eq","value":true},{"field":"case.category","op":"in","value":["image","video"]}]}`
- **THEN** 引擎按上下文（用户属性 + Case 属性）求值并返回布尔结果

#### Scenario: 未知字段或非法运算被拒绝
- **WHEN** 规则引用未注册字段或未定义运算
- **THEN** 规则校验失败并返回可定位错误（字段/运算名），不得静默视为 false

### Requirement: 属性提供方注册与 schema 驱动
系统 MUST 提供属性提供方注册机制：每个属性 MUST 声明稳定 key、JSON Schema（类型/枚举/描述/UI 标签）、可选的上下文来源（user/case/input）。新增条件类型 MUST 通过注册 provider + schema 完成，MUST NOT 修改引擎求值代码；管理端条件表单 MUST 由属性 schema 自动渲染（输入控件、可选项、校验）。

#### Scenario: 新属性无需改引擎
- **WHEN** 注册新属性 `user.region`（提供 schema 与取值函数）
- **THEN** 无需改动引擎代码即可在规则中使用并求值，管理端自动出现对应条件配置控件

#### Scenario: 上下文缺失按 false 处理
- **WHEN** 求值时某属性对应上下文数据缺失（如用户无 is_premium 记录）
- **THEN** 该条件求值为 false（`exists` 运算除外），不报错

### Requirement: 条件求值上下文
系统 MUST 为求值提供结构化上下文，至少包含：用户属性命名空间（本期示例 `user.is_premium`）、Case 属性命名空间（`case.category`、`case.tags`），并预留输入属性命名空间；上下文字段由 provider 声明。

#### Scenario: 按用户与 Case 属性路由
- **WHEN** 上下文为 user.is_premium=true、case.category=image
- **THEN** 规则 `user.is_premium eq true` 与 `case.category in [image, video]` 均命中

#### Scenario: provider 求值错误不被吞掉
- **WHEN** 求值某叶子条件时属性 provider 返回错误（区别于属性缺失）
- **THEN** 求值返回错误，不得静默按 false 处理

```

## docs/openspec/changes/topic-routing/specs/dispatch-topic-routing/spec.md

- Source: docs/openspec/changes/topic-routing/specs/dispatch-topic-routing/spec.md
- Lines: 1-22
- SHA256: 031025ff854b9e6ff6bbf7906f2d38fb2408298cc32560d8d56e1439b26e84c9

```md
## REMOVED Requirements

### Requirement: 本期不实现投放表达式选路
**Reason**: 本期交付按条件表达式选 Topic 投放，延后声明作废。
**Migration**: 以本 delta 新增的「按条件表达式选 Topic 投放」要求为准。

## ADDED Requirements

### Requirement: 按条件表达式选 Topic 投放
调度 MUST 在任务调度时求值 Case 路由规则，选中目标 Topic，并仅使订阅该 Topic 的节点可领取；求值结果 MUST 记录在 Task（`dispatch_topic`）。无规则命中时 MUST 回退默认 Topic。

#### Scenario: 条件命中高规格 Topic
- **WHEN** 任务上下文命中 Case 规则指向 Topic `fast-gpu`
- **THEN** Task 的 `dispatch_topic=fast-gpu`，且只有订阅 fast-gpu 的节点可领取

#### Scenario: 回退默认 Topic
- **WHEN** 任务上下文不命中任何规则
- **THEN** Task 的 `dispatch_topic=default`，默认 Topic 订阅者可领取

#### Scenario: 求值错误保持 pending
- **WHEN** 规则求值出现错误（如 provider 读取用户属性失败）
- **THEN** Task 保持 pending 并记录原因，不投递到默认 Topic，后续调度周期重试

```

## docs/openspec/changes/topic-routing/specs/edge-agent/spec.md

- Source: docs/openspec/changes/topic-routing/specs/edge-agent/spec.md
- Lines: 1-12
- SHA256: c5b255f4400d85c4e01adf76f596bb44458dd32f9ea41c3143d8d07a263c464f

```md
## ADDED Requirements

### Requirement: Edge 订阅 Topic 配置
Edge-Agent MUST 支持配置订阅 Topic 列表（如 `subscribe_topics`）；未配置时 MUST 默认订阅 `default` Topic。进程启动后 MUST 按实际订阅集合向控制面申报并长轮询领取。Edge MUST NOT 解析投放表达式（规则求值只发生在控制面）。

#### Scenario: 未配置订阅默认
- **WHEN** Edge 配置未声明 subscribe_topics
- **THEN** Edge 以 `default` 为订阅集合领取任务

#### Scenario: 多 Topic 订阅
- **WHEN** Edge 声明订阅 `["default","fast-gpu"]`
- **THEN** Edge 可领取这两个 Topic 下投递的任务，不领取其它 Topic 的任务

```

## docs/openspec/changes/topic-routing/specs/platform-bootstrap/spec.md

- Source: docs/openspec/changes/topic-routing/specs/platform-bootstrap/spec.md
- Lines: 1-8
- SHA256: cf634be325746cdd1edef754c68702f0c553bb8a04bd7e31dc2370b6a70ff0f3

```md
## ADDED Requirements

### Requirement: 启动初始化默认 Topic
控制面启动 MUST 在业务库就绪后幂等确保默认 Topic（`default`）存在；已存在则跳过。该初始化 MUST NOT 阻塞未完成初始化向导的路径。

#### Scenario: 空库启动创建默认 Topic
- **WHEN** 业务库中无任何 Topic 记录且控制面启动
- **THEN** 创建 `default` Topic；再次启动不重复创建

```

## docs/openspec/changes/topic-routing/specs/task-orchestrator/spec.md

- Source: docs/openspec/changes/topic-routing/specs/task-orchestrator/spec.md
- Lines: 1-20
- SHA256: 4c85de71d069affb3d302050c569edc6b43c060dff9684f1373afbc6c1b34de3

```md
## MODIFIED Requirements

### Requirement: 调度与 pending 扫描双触发
系统 MUST 支持事件触发调度，并 MUST 提供定时扫描 `pending` 的兜底。调度 MUST 先求值 Case 路由规则确定目标 Topic（无命中回退默认 Topic），再仅在订阅该 Topic 的可用节点（健康/熔断/在线 Edge 等条件）上抢占领取，将 Task 置为可被该 Topic 订阅节点领取的状态（并准备含 `job_ref` 的任务包），MUST NOT 默认向 Redis `dispatch.<instance_id>` Publish 作为唯一派发手段。当目标 Topic 没有可投递节点/在线 Edge 时 MUST 保持 Task 为 `pending` 并记录原因，MUST NOT 假装已排队。MUST NOT 假设控制面一定能直连家里 ComfyUI。

#### Scenario: 创建事件丢失仍被扫到
- **WHEN** Task 长期处于 pending 且无成功调度
- **THEN** SchedulePending 仍能将其变为可领取（若目标 Topic 存在可投递节点/在线 Edge）

#### Scenario: 条件路由到目标 Topic
- **WHEN** pending Task 的路由规则命中 Topic `fast-gpu`
- **THEN** 任务置为可领取、`dispatch_topic=fast-gpu`、含有效 `job_ref`，且 Blob 中存在可读任务包

#### Scenario: 同 Topic 多节点并发领取互斥
- **WHEN** 目标 Topic 存在多个在线订阅节点并发领取同一个任务
- **THEN** 恰好一台节点成功领取（带租约），其余获得空结果，任务不被重复分配

#### Scenario: 目标 Topic 全部不可用时不投递
- **WHEN** 目标 Topic 没有任何可投递订阅节点（或无在线 Edge）
- **THEN** Task 仍保持 pending（或未被标记为可领取），且不向任意节点交付任务

```

## docs/openspec/changes/topic-routing/specs/task-persistence/spec.md

- Source: docs/openspec/changes/topic-routing/specs/task-persistence/spec.md
- Lines: 1-8
- SHA256: 6785441c923b2e6e2c6faa70f714062517bc2f79396c7961b4d68554fa5d97e7

```md
## ADDED Requirements

### Requirement: Task 记录投递 Topic 与重试/租约信息
Task 持久化记录 MUST 包含：`dispatch_topic`（实际路由 Topic）、重试次数、当前领取节点与租约截止时间（可复用既有字段语义）。进程重启后 MUST 仍能依据这些字段恢复调度（任务按其 Topic 重新可领取）。

#### Scenario: 重启后按 Topic 恢复
- **WHEN** 进程重启，存在 `dispatch_topic=fast-gpu` 且状态为 pending/可领取的任务
- **THEN** 订阅 fast-gpu 的节点仍可领取该任务

```

## docs/openspec/changes/topic-routing/specs/topic-admin/spec.md

- Source: docs/openspec/changes/topic-routing/specs/topic-admin/spec.md
- Lines: 1-22
- SHA256: 66343787b60821dbe673accc90b2df1e6d16e9e43660aa54802d1c75e8570708

```md
## REMOVED Requirements

### Requirement: 本期不实现 Topic 与投放规则管理验收
**Reason**: 本期交付 Topic 目录管理与投放规则管理，延后声明作废。
**Migration**: 以本 delta 新增的「Topic 管理与默认 Topic」要求为准。

## ADDED Requirements

### Requirement: Topic 管理与默认 Topic
系统 MUST 提供 Topic 的创建、查询（单个与列表）、更新、启用/禁用。Topic MUST 包含稳定 `key`、显示名、启用状态与创建时间。应用启动初始化 MUST 幂等确保默认 Topic（`default`）存在；默认 Topic MUST NOT 可删除。被禁用 Topic MUST NOT 作为新的路由目标，也不可作为新订阅的绑定目标。

#### Scenario: 创建后列表可见
- **WHEN** 管理员创建 Topic `fast-gpu` 成功
- **THEN** 列表接口返回该 Topic，且可被 Case 路由规则引用

#### Scenario: 默认 Topic 自动创建且不可删
- **WHEN** 业务库为空时启动控制面
- **THEN** 自动创建 `default` Topic；对其删除请求被拒绝

#### Scenario: 禁用 Topic 不可路由
- **WHEN** Topic `fast-gpu` 被禁用
- **THEN** 引用它的新规则保存被拒绝（存量任务按既有终态/对账路径处理）

```

## docs/openspec/changes/topic-routing/specs/topic-routing-config/spec.md

- Source: docs/openspec/changes/topic-routing/specs/topic-routing-config/spec.md
- Lines: 1-23
- SHA256: 1ca6e9a7ef3c8300c1113218bf6277039ce530fcda0598889b5ec7b0dbc33c11

```md
## Purpose

定义 Case 与 Topic 之间的路由投放配置：一个 Case 声明一对 N 目标 Topic 与条件规则、首个命中即投、无命中回退默认 Topic，并规定配置的持久化与校验契约。

## ADDED Requirements

### Requirement: Case 路由配置结构
Case 路由配置 MUST 包含有序规则列表（每条 = 条件 + 目标 Topic key）；规则求值 MUST 采用首个命中即投；无任何规则命中或未配置路由时 MUST 回退系统默认 Topic（Case 不单独声明默认 Topic）。Case 可配置的 Topic 数量 MUST 为一个到多个。

#### Scenario: 首个命中即投
- **WHEN** Case 配置规则为 [条件1→Topic A, 条件2→Topic B] 且任务上下文同时满足两条规则
- **THEN** 任务投递到第一条命中规则的目标 Topic A

#### Scenario: 无命中回退默认
- **WHEN** 任务上下文不满足任何规则（或 Case 未配置路由）
- **THEN** 任务投递到默认 Topic

### Requirement: 路由配置校验
路由配置 MUST 在保存时校验：目标 Topic 必须存在且启用；条件必须通过条件协议校验；每条规则 MUST 含条件与目标 Topic；Case 至少有一条可达路径（规则命中或默认 Topic）。

#### Scenario: 引用不存在的 Topic 被拒绝
- **WHEN** 保存引用不存在或已禁用 Topic 的路由规则
- **THEN** 保存被拒绝并返回可定位错误

```

## docs/openspec/changes/topic-routing/specs/workflow-protocol/spec.md

- Source: docs/openspec/changes/topic-routing/specs/workflow-protocol/spec.md
- Lines: 1-12
- SHA256: 82c8a5ffbbb20a4be37cb6a4105eaf56d8fc97021bae16f5da85c51380655c61

```md
## ADDED Requirements

### Requirement: Case 协议包含路由投放配置
Case 协议 MUST 支持可选的路由投放配置（结构见 `topic-routing-config`）：有序规则列表（条件 + 目标 Topic key）；未配置时行为等价于全部回退系统默认 Topic。路由配置 MUST 随 Case 一起注册、更新与查询。

#### Scenario: 注册带路由的 Case
- **WHEN** 提交含路由配置的 Case（规则引用已启用 Topic）
- **THEN** Case 注册成功且路由配置可查询

#### Scenario: 旧 Case 无路由配置兼容
- **WHEN** 查询未配置路由的历史 Case
- **THEN** 返回无路由配置的表示，调度回退默认 Topic，不报错

```
