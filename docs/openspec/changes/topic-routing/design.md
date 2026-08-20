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

- 失败重试的退避策略：立即重入 vs 指数退避（build 期按可观测性要求定，不改 spec）。
- claim 长轮询的单次等待上限与扫描批大小（实现细节，不改行为契约）。
- Topic key 命名规则与校验（如小写字母数字连字符），build 期在 topic-admin 定。
