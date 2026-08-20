# Brainstorm Summary

- Change: topic-routing
- Date: 2026-08-20

## 确认的技术方案

### 数据模型
- 新增 `topics` 表：`key`（主键，`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`）、`name`、`enabled`、时间戳；启动幂等种入 `default`，`default` 不可删除。
- `edges.subscribe_topics_json`（JSON 数组，空/缺失 = `["default"]`）；`tasks` 增加 `dispatch_topic`（索引）、`attempts`、`requeue_at`（可选重试延迟）。
- Case doc 增加 `routing.rules: [{when: <条件 JSON>, topic: <key>}]`，保存时校验。

### 条件协议（关键设计点）
- 包位：`internal/runtime/domain/condition`（纯领域，无 DB/HTTP 依赖）。
- 规则语法：叶子 `{field, op, value}`；组合 `{"and":[...]}` / `{"or":[...]}`；运算 `eq/ne/in/gt/gte/lt/lte/exists`。
- 属性提供方：`ListAttributes() []AttributeDescriptor` + `Value(ctx, field)`；descriptor 含 key、JSON Schema、UI 标签、上下文来源（user/case/input）。
- 内置 provider：`user.is_premium`（task→session→user）、`case.category`、`case.tags`；`input.*` 预留。
- 引擎：`Validate(rule)` + `Evaluate(ctx, rule)`；缺失属性值按 false（`exists` 除外）；**provider 错误向上传播**（区别于缺失）。
- 新增条件 = 注册 provider + schema，引擎与前端零改动。

### 调度与求值
- dispatchTask：读 Case 路由 → 按顺序求值首个命中 → `dispatch_topic`（无命中 = `default`）→ prep job → `PrepareForClaim`（queued + topic + job_ref，**不再绑定 edge_id**）。
- 求值错误 → 保持 pending 并记录原因，下一轮 SchedulePending 重试（可自愈，不误投）。**已确认 1A。**
- 事件触发 + SchedulePending 双通道保持。

### 原子领取
- 仓储接口：`ClaimNextWithLease(ctx, topics []TopicKey, lease, now)`；控制面按节点订阅集合解析 topics（空=default）。
- 原子性：沿用现有事务内「SELECT 最老 queued（status + dispatch_topic IN (...) + job_ref 存在 + requeue_at 到期）→ 条件 UPDATE（status=queued→running + lease + edge_id）→ RowsAffected 0 则 continue」；SQLite 单写者串行，Postgres/MySQL 行锁 + status 守卫。索引 `(status, dispatch_topic, created_at)`。
- claim 响应形状不变（task_id/edge_id/job_ref/lease_until）；长轮询重扫沿用。
- 备选：Redis Streams 消费组抢占——与「默认不依赖 Redis」约束冲突，拒绝。

### 失败重回与回收
- failed（任务自身失败）：attempts+1 < max（默认 3）→ 回 queued（保留 dispatch_topic，清 edge_id/lease，写 requeue_at 退避）；达上限 → 最终 failed。
- 退避：指数 5s/15s/45s（可配）。**已确认 2A。**
- 租约过期（节点宕机，非任务失败）：RequeueExpiredLeases 回收，**不计 attempts**；结合既有 ExecutionQuery 对账收口病理场景。
- 失败重回沿用原 `dispatch_topic`，不重新求值。

### 节点订阅绑定（补充确认）
- Edge-Agent 配置 `subscribe_topics` 作为声明（空 = default）；presence 首次报到时写入控制面（复用 hardware 语义：首次写入、重启不覆盖、管理端 PATCH 可覆盖）。
- claim 端点以控制面存储的订阅集合为准（空 = default）；Edge 配置只做启动声明，管理端为权威。

### Admin API
- `/api/v1/topics` CRUD（default 不可删、禁用后不可作新路由目标/新订阅）。
- `PATCH /api/v1/edges/{id}` 支持 `subscribe_topics`。
- Case 创建/更新/详情支持 `routing` + 校验。
- `GET /api/v1/routing/attributes` 条件目录。

### 兼容与迁移
- 启动迁移：存量 queued 且 edge_id 已绑定、租约过期的任务清持有者并置 `dispatch_topic=default`；存量空 dispatch_topic 查询层按 default。
- 旧 Edge 未配置订阅 = default，claim 形状不变；先升级控制面后升级 Edge。

## 关键取舍与风险

- [规则误配进错池] → 保存校验 + 默认 Topic 兜底 + 求值错误保守 pending。
- [并发领取热点] → DB CAS + 索引 + 并发契约测试。
- [重试风暴] → 有界重试 + 指数退避 + 复用 StormGuard。
- [节点离线滞留] → 租约回收 + 观测。
- [升级不兼容] → 迁移清旧持有者 + 查询层兜底；升级顺序文档化。

## 测试策略

- 条件协议单测（校验/求值/组合/新 provider 扩展性）。
- 仓储并发契约测试（多 goroutine 抢单，每任务恰好一成功）。
- Orchestrator（首个命中/回退/求值错误/无在线节点）。
- API 测试（topics CRUD、default 保护、订阅绑定、routing 校验、attributes）。
- E2E：allinone mock 默认 Topic；split Edge 多 Topic 订阅。

## Spec Patch（已确认，将回写）

- `condition-protocol`：补充场景「provider 求值错误向上返回错误（区别于上下文缺失）」，MUST NOT 静默按 false。
- `dispatch-topic-routing`：补充场景「规则求值错误时任务保持 pending 并记录原因，不投递默认 Topic」。
