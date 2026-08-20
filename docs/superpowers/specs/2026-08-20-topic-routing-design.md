---
comet_change: topic-routing
role: technical-design
canonical_spec: openspec
---

# Topic 调度（任务分流）— 技术设计

> OpenSpec 上游：`docs/openspec/changes/topic-routing/`（proposal / design / delta specs / tasks）。

## 1. 背景与目标

见 proposal.md（Why）。本设计深化 open 阶段的 design.md：把「条件协议、按 Topic 调度、原子领取、失败/宕机回收」落成可实现的模块边界、数据流与算法，并锁定 build 期可直接执行的测试矩阵。

现状关键约束：

- Task 表是执行态唯一真相源；默认跨进程派发是 DB 可领取态 + Edge 长轮询，不依赖 Redis。
- 已有原子领取骨架：`PrepareForClaim`（pending→queued+edge_id+job_ref）、`ClaimNextWithLease`（事务内 CAS，queued→running+lease）、`RequeueExpiredLeases`、`HeartbeatLease`。
- Case doc 存放 Comfy workflow 与 bindings；用户属性目前只有 `is_premium`。
- 控制面进程为 Go（GORM + SQLite，可切 MySQL/Postgres）；Edge-Agent 独立进程，经 `/agent/v1` 长轮询。

## 2. 模块边界

```text
internal/runtime/domain/condition        # 条件协议（纯领域，无 DB/HTTP）
internal/runtime/application/routing     # 路由求值编排（Case 路由 → dispatch_topic）
internal/runtime/application/orchestrator# 调度、claim、applyStatus、对账、回收（复用 StormGuard）
internal/runtime/domain                  # Task 状态机扩展（dispatch_topic/attempts/requeue_at）
internal/runtime/infrastructure/persistence# GORM Task/Topic 仓储（原子 claim 落库）
internal/catalog/domain                  # Case doc 增加 routing 字段与校验
internal/platform/edge                   # Edge 记录增加订阅绑定（presence 声明 + 管理端覆盖）
internal/httpapi/topics                  # Topic 管理 API
internal/httpapi/routing                 # 条件目录 API
apps/edge-agent                          # subscribe_topics 配置与 claim 适配
```

依赖规则不变：`domain/condition` 不引用仓储；`orchestrator` 只经端口（TaskRepository / Router / EdgeRegistry / Prep / Query）工作。

## 3. 数据模型与迁移

### 3.1 新表 `topics`

| 列 | 类型 | 说明 |
|---|---|---|
| `key` | varchar PK | `^[a-z0-9]([a-z0-9-]{0,62}[a-z0-9])?$`，不可变 |
| `name` | varchar | 显示名 |
| `enabled` | bool | 禁用后不可作新路由目标/新订阅 |
| `created_at` / `updated_at` | time | |

种子：控制面启动（业务库就绪后）幂等插入 `default`（enabled=true）；删除 `default` 返回冲突错误。

### 3.2 `edges`

新增 `subscribe_topics_json`（TEXT，JSON 字符串数组）。归一化规则：空串 / `[]` / NULL → 有效集合 `["default"]`。写入时机（复用 hardware 首次报到语义）：

1. Edge presence 首次携带 `subscribe_topics`（或首次报到时库中为空）→ 写入声明值；
2. 之后心跳不覆盖；
3. 管理端 `PATCH /api/v1/edges/{id}` 覆盖；再次「从机器更新」不重置（订阅不随硬件刷新）。

### 3.3 `tasks`

| 新增列 | 说明 |
|---|---|
| `dispatch_topic` | varchar，索引；空值查询层按 `default`（迁移为主，查询兜底） |
| `attempts` | int 默认 0；失败/回收累计执行尝试 |
| `requeue_at` | nullable time；到期前不可被领取（退避） |

复用 `edge_id`（当前持有节点）与 `lease_until`（租约）。

### 3.4 Case doc

`routing: { "rules": [ { "when": <condition>, "topic": "<topic-key>" } ] }`（有序，首个命中即投）。无该字段 = 全部回退 `default`。校验见 §7.3。

### 3.5 启动迁移

1. AutoMigrate 新表/列；种入 `default`。
2. 存量 `queued` 且 `edge_id` 非空、`lease_until` 已过期的行：清 `edge_id`/`lease_until`，置 `dispatch_topic='default'`（历史任务归默认池）。
3. 存量 `running` 行不动，交由 ReconcileStale 按既有路径处理。
4. 查询层兜底：`dispatch_topic=''` 等价 `default`（滚动升级期保险）。

## 4. 条件协议详细设计（关键设计点）

### 4.1 规则语法

```json
{
  "and": [
    { "field": "user.is_premium", "op": "eq", "value": true },
    { "or": [
        { "field": "case.category", "op": "in", "value": ["image", "video"] },
        { "field": "case.tags",     "op": "in", "value": ["fast"] }
    ]}
  ]
}
```

- 叶子：`{field, op, value}`；组合：`{"and": [<rule>, ...]}` / `{"or": [<rule>, ...]}`；组合数组非空。
- 内置运算：`eq` `ne` `in` `gt` `gte` `lt` `lte` `exists`。运算集在引擎内固定（协议能力边界），属性集可扩展。
- `exists`：只判断属性是否有值，忽略 `value`。

### 4.2 属性提供方

```go
type AttributeDescriptor struct {
    Key     string         // "user.is_premium"
    Context string         // "user" | "case" | "input"
    Label   string         // UI 标签
    Schema  map[string]any // JSON Schema 子集：type/enum/description
}

type Provider interface {
    Namespace() string                          // "user" / "case" / "input"
    ListAttributes() []AttributeDescriptor
    Value(ctx context.Context, field string) (any, error)
}
```

注册表：`registry.Register(p Provider)`；`Registry.ProviderFor(field)` 按 `namespace.attr` 解析。本期内置：

| field | 来源 | 取值 |
|---|---|---|
| `user.is_premium` | task→session→user | `*bool`；nil → 缺失 |
| `case.category` | case doc | string 或缺失 |
| `case.tags` | case doc | []string |

`input.*` 命名空间预留（无 provider）。

### 4.3 求值语义

- `Validate(rule)`：字段已注册、op 合法、value 与 schema 类型一致（enum 成员校验）；组合递归。
- `Evaluate(ctx, rule)`：
  - `and`：全真为真，短路；`or`：任一为真，短路。
  - 叶子：`Value` 返回 `ErrAttributeMissing` → 条件为 false（`exists` 为 true）；其它错误 → 返回错误（**不静默**）。
  - 值比较按 descriptor 类型归一（bool/string/number/list containment）。
- 调度层对求值错误的处置（已确认 1A）：保持 `pending`，`error_message` 记录「routing: <field> evaluate: <err>」，下一轮 SchedulePending 重试；错误自愈前不投递、不误投默认池。

## 5. 调度流程

```text
OnTaskCreated / SchedulePending
  → Task.Get(id)，非 pending 跳过
  → caseDoc = CaseRepo.Get(task.CaseID)
  → evalCtx = buildContext(task, session, user, caseDoc)   // user/case providers
  → topic, err = Router.Resolve(caseDoc.Routing, evalCtx)
       // 首个命中即投；无规则/无命中 → default
       // err=评估错误 → 保持 pending + 记录原因，return
  → ref = Prep.PrepareJob(ctx, taskID)                     // edge-agnostic，见 §5.1
  → PrepareForClaim(ctx, taskID, topic, ref, now)          // queued + dispatch_topic + job_ref，edge_id 留空
```

`PrepareForClaim` 签名变更（BREAKING 内部）：`(ctx, id, topic, jobRef, now)`，不再收 `edgeID`；条件 `status='pending'` 不变。

### 5.1 PrepareJob 去实例化

现有 `PrepareJob(ctx, taskID, edgeID)` 的 edgeID 仅用于任务包归属展示；job 包内容（已注入 workflow + 图片 BlobRef + `outputs/<taskID>/` 前缀）本就 edge 无关。改为 `PrepareJob(ctx, taskID)`，任务包可被任意订阅节点领取后执行，产物回写共享 Blob。

## 6. 原子领取详细设计

### 6.1 接口

```go
// 领域仓储
ClaimNextWithLease(ctx context.Context, topics []TopicKey, lease time.Duration, now time.Time) (*Task, error)
RequeueExpiredLeases(ctx context.Context, now time.Time) (int, error)
RequeueAfterFailure(ctx context.Context, id TaskID, backoff time.Duration, now time.Time, reason string) error
HeartbeatLease(ctx context.Context, id TaskID, edgeID EdgeID, lease time.Duration, now time.Time) (bool, error)
```

### 6.2 CAS 算法（SQLite/MySQL/Postgres 同一语义）

```text
事务内循环：
  row = SELECT * FROM tasks
        WHERE status='queued'
          AND (dispatch_topic IN (:topics) OR dispatch_topic='')
          AND job_ref_json IS NOT NULL AND job_ref_json != ''
          AND (requeue_at IS NULL OR requeue_at <= :now)
        ORDER BY created_at ASC, id ASC
        LIMIT 1
  if not found: return nil
  n = UPDATE tasks SET status='running', edge_id=:edgeID, lease_until=:now+:lease, updated_at=:now
      WHERE id=:row.id AND status='queued'
  if n == 0: continue            // 被并发领取者先改走
  return row
```

- SQLite：单写者天然串行。
- MySQL/Postgres：默认隔离级别下，两个事务可能读到同一行，但败者的 UPDATE 被行锁阻塞后 `RowsAffected=0` → 重扫下一行；同一任务最多一次成功置 `running`。
- 索引：`(status, dispatch_topic, created_at)`；`requeue_at` 为范围条件，随行数增长评估（本期规模可控）。
- 长轮询：现有 `/agent/v1/jobs/claim?edge_id=&wait=` 的循环重扫机制复用，等待上限可配（默认 30s）。

### 6.3 订阅解析

- claim 请求只带 `edge_id`（形状不变，BREAKING 语义：不再按 instance 专属任务过滤）。
- 控制面从 edge 注册表读取该节点有效订阅集合（`subscribe_topics_json` 归一化，空=default）。
- Edge 首次 presence 声明写入订阅；管理端 PATCH 为权威覆盖（§3.2）。

### 6.4 竞态契约

`ClaimNextWithLease` 必须在测试中证明：N 个 goroutine 并发抢 M 个任务，结果集每任务恰好出现一次，无重复发放。

## 7. Admin API 契约

### 7.1 Topic 管理

| 方法/路径 | 行为 |
|---|---|
| `GET /api/v1/topics` | 列表（含 enabled 过滤） |
| `POST /api/v1/topics` | `{key,name}`；key 模式校验；重复冲突 |
| `GET /api/v1/topics/{key}` | 详情 |
| `PUT /api/v1/topics/{key}` | 更新 name/enabled；key 不可改 |
| `DELETE /api/v1/topics/{key}` | 禁用删除；`default` 拒绝；被引用（Case 规则/Edge 订阅）时拒绝或提示先解除 |

### 7.2 节点订阅

`PATCH /api/v1/edges/{id}` 增加 `subscribe_topics: []string`；校验每个 key 存在且启用；`[]`/缺省 = 默认语义。列表/详情返回有效订阅集合（归一化后）。

### 7.3 Case 路由

case-admin-api 创建/更新载荷增加 `routing`；`catalog/infrastructure/validation` 扩展校验：

- 每条规则 `when` 通过 condition.Validate（注册表注入）；
- `topic` 存在且启用（Topic 仓储查询）；
- `rules` 为空或缺失合法（= 默认回退）。

### 7.4 条件目录

`GET /api/v1/routing/attributes` → `{"attributes": [{"key","label","context","schema"}]}`，来自 provider 注册表；`task-flow-editor` 据此自动渲染表单。

## 8. Edge-Agent 变更

- 配置：`subscribe_topics: []string`（yaml）/ `EDGE_SUBSCRIBE_TOPICS`（逗号分隔环境变量）；空 = 未声明。
- presence 载荷增加可选 `subscribe_topics`（仅首次写入，见 §3.2）。
- claim 适配：请求仍带 `edge_id`，控制面按存储订阅领取；Edge 不解析表达式。
- 兼容：旧 Edge（无该字段）presence 后存储为空 → 有效订阅 = default，行为等价现状。

## 9. 失败重回与回收

### 9.1 任务失败（status=failed 上报）

```text
applyStatus(failed)
  → attempts = attempts + 1
  → if attempts >= max_attempts(3): 最终 failed（保留 error，走 notify）
  → else: status=queued；edge_id/lease 清空；dispatch_topic 不变；
           requeue_at = now + backoff(attempts)   // 5s / 15s / 45s，可配
```

### 9.2 租约过期（节点宕机/失联）

```text
RequeueExpiredLeases(now)
  → 对 status IN (queued, running) 且 lease_until 非零且已过期：
      status=queued；edge_id/lease 清空；requeue_at=now（不计 attempts）
  → 病理场景（反复无法完成）由既有 ExecutionQuery 对账在超时后收口 failed
```

说明：租约过期代表节点问题而非任务问题，不烧 attempts（已确认）；防无限循环依赖对账收口与节点在线观测。

### 9.3 风暴防护

调度/回收沿用 StormGuard 分池与限流；失败重回的退避表独立可配；claim 空转不触发通知。

## 10. 兼容与升级

1. 控制面先行：新表/列、种子、迁移、Topic 调度与 claim 新语义；旧 Edge 未声明订阅 → default，可继续工作。
2. Edge 升级：声明 `subscribe_topics`（不声明行为不变）。
3. 回滚：调度开关回退「全部按 default 领取」，不依赖表达式；既有 claim/lease 路径保持兼容。
4. 文档同步：`docs/architecture/{data-model,runtime,task-data-walkthrough}.md`、README。

## 11. 测试策略

| 层 | 覆盖 |
|---|---|
| condition 单测 | 校验（未知字段/非法 op/类型不符/组合）、求值（and/or 短路、缺失=false、exists）、provider 注册扩展性（新 provider 不改引擎） |
| 仓储契约 | 并发抢单（N goroutine × M 任务，恰好一次）、topic 过滤（多 Topic 集合）、requeue_at 未到期不可领、租约过期回收、失败重回 attempts/backoff、超限 failed |
| Orchestrator | 首个命中、无命中回退 default、求值错误保持 pending 记原因、无在线节点不投递、job_ref 存在 |
| HTTP API | topics CRUD + default 保护、edges 订阅绑定 + 归一化、case routing 校验、attributes 目录 |
| E2E | allinone/mock 默认 Topic 主路径；split Edge 多 Topic 订阅领取并执行 |
| 兼容 | 存量 queued 迁移、空 dispatch_topic 兜底、旧 Edge 未声明订阅 |

## 12. 风险与缓解

- [规则误配导致任务进错池] → 保存校验 + 默认 Topic 兜底 + 求值错误保守 pending + 规则变更审计。
- [并发领取热点/索引膨胀] → `(status, dispatch_topic, created_at)` 索引；claim 批大小与等待上限可配。
- [重试风暴] → 有界重试 + 指数退避 + StormGuard。
- [节点反复宕机任务滞留] → 租约回收 + ExecutionQuery 对账收口 + 在线观测。
- [升级窗口不一致] → 迁移清旧持有者 + 查询层 default 兜底；升级顺序文档化。

## 13. build 期实现细节开关

- `max_attempts`（默认 3）、退避表（默认 5s/15s/45s）、claim 等待上限（默认 30s）与扫描批大小：配置化，不改行为契约。
- Topic key 校验实现为共享 regex；`default` 为保留 key。
