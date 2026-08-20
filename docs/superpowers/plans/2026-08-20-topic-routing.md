---
change: topic-routing
design-doc: docs/superpowers/specs/2026-08-20-topic-routing-design.md
base-ref: 3f2cc67642628024536fb2baa9ca160a5e2bdf04
---

# Topic 调度（任务分流）实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 Pixoma 落地 Topic 调度：Topic 实体与默认 Topic、Case 一对 N Topic 条件投放（可扩展条件协议）、节点订阅 Topic、并发原子领取（一任务一台消费）、失败/宕机回收，全链路不引入 Redis 依赖。

**Architecture:** 控制面以 DB 为 Task 唯一真相源：Orchestrator 先按条件协议求值 Case 路由确定 `dispatch_topic`，将 Task 置为可领取（不预绑定节点）；Edge 按控制面存储的订阅集合经 `/agent/v1/jobs/claim` 原子抢占（事务内条件 UPDATE）。条件协议为纯领域包（provider 注册 + JSON Schema 描述），新增属性不改引擎。

**Tech Stack:** Go（GORM，SQLite/MySQL/Postgres）、chi、slog、testify；Edge-Agent（Go）；`internal/runtime/domain/condition` 纯领域。

## Global Constraints

- Topic key 必须匹配 `^[a-z0-9]([a-z0-9-]{0,62}[a-z0-9])?$`；`default` 为保留 key，禁止删除。
- 默认派发不依赖 Redis：跨进程走 DB 可领取态 + Edge 长轮询；条件协议不引入 CEL/expr 等新依赖。
- 同一任务在并发 claim 下最多被一台节点消费（事务内 CAS）。
- 条件求值错误（非属性缺失）必须保持 Task pending 并记录原因，不投默认 Topic；缺失属性按 false（`exists` 除外）。
- 失败重试：`max_attempts=3`，退避 5s/15s/45s；租约过期回收不计 attempts。
- Edge 未配置 `subscribe_topics` = 订阅 `default`；presence 首次报到写入，管理端 PATCH 为权威。
- Mock（`comfy_mock`）主路径必须端到端可通；所有新测试用 testify。
- 产物语言 zh-CN；文档同步 `docs/architecture/{data-model,runtime,task-data-walkthrough}.md` 与根 README。

---

## 1. 数据模型与迁移

### Task 1: `topics` 表与 GORM 模型

**Files:**
- Create: `internal/platform/topic/domain.go`（Topic 聚合：Key/Name/Enabled/时间戳）
- Create: `internal/platform/topic/repository.go`（接口：`List/Get/Create/Update/Delete`）
- Create: `internal/platform/topic/infrastructure.go`（GORM 实现 `TopicRepository`，表 `topics`）
- Modify: `internal/platform/db/db.go` 或现有 AutoMigrate 清单，注册 `topic.TopicRow`

**Interfaces:**
- Produces: `type Topic struct { Key string; Name string; Enabled bool; CreatedAt, UpdatedAt time.Time }`
- Produces: `type Repository interface { List(ctx, enabled *bool) ([]Topic, error); Get(ctx, key string) (*Topic, error); Create(ctx, t Topic) error; Update(ctx, t Topic) error; Delete(ctx, key string) error }`

- [x] **Step 1: 写失败测试**（`internal/platform/topic/repository_test.go`）：GORM sqlite 内存库 AutoMigrate 后 Create/Get/List/Update/Delete 圆环通过；Delete 不存在返回 `ErrTopicNotFound`。
- [x] **Step 2: 运行确认失败**：`go test ./internal/platform/topic/ -count=1` → FAIL（包不存在）。
- [x] **Step 3: 实现 domain/repository/infrastructure**：`TopicRow` 用 `gorm:"primaryKey;size:64"` 映射 `key`；`Create` 冲突返回 `ErrTopicConflict`；`Get` 未找到返回 `ErrTopicNotFound`。
- [x] **Step 4: 运行通过**：`go test ./internal/platform/topic/ -count=1` → PASS。
- [x] **Step 5: 提交**：`git add internal/platform/topic && git commit -m "feat(topic): add topics model and repository"`

### Task 2: `edges`/`tasks` 列与 Case routing 模型

**Files:**
- Modify: `internal/platform/edge/`（Edge 记录增加 `SubscribeTopics []string`，持久化为 `subscribe_topics_json`）
- Modify: `internal/runtime/infrastructure/persistence/gorm_task.go`（`TaskRow` 增加 `DispatchTopic string`、`Attempts int`、`RequeueAt *time.Time`；列 `dispatch_topic`/`attempts`/`requeue_at`）
- Modify: `internal/runtime/domain/task.go`（Task 增加同名字段与迁移方法）
- Modify: `internal/catalog/domain/document.go`（Case doc 增加 `Routing *RoutingConfig`；`type RoutingConfig struct { Rules []RoutingRule }`、`type RoutingRule struct { When json.RawMessage; Topic string }`）

**Interfaces:**
- Produces: `func NormalizeTopics(raw []string) []string`（空/缺失 → `["default"]`，去重保序）
- Produces: `func (t *Task) PrepareForTopic(topic string, jobRef BlobRef, now time.Time) error`（pending→queued，写 dispatch_topic/job_ref，清 edge_id/lease/requeue_at）
- Produces: `func (t *Task) ClaimWithLease(edgeID EdgeID, lease time.Duration, now time.Time) error`（queued→running，写 edge_id/lease；复用现有语义）

- [x] **Step 1: 写失败测试**：`Task.PrepareForTopic` 从 pending 成功；重复调用返回 `ErrInvalidTransition`；`NormalizeTopics(nil/[]/{default,x})` 断言。
- [x] **Step 2: 运行确认失败**：`go test ./internal/runtime/domain/... ./internal/platform/edge/... -count=1` → FAIL（字段缺失）。
- [x] **Step 3: 实现**：加列与字段；GORM `Updates` map 增加新列；`toRow/fromRow` 双向映射；AutoMigrate 生效。
- [x] **Step 4: 运行通过**：`go test ./internal/runtime/... ./internal/platform/edge/... -count=1` → PASS。
- [x] **Step 5: 提交**：`git commit -am "feat(runtime): add topic/attempts/requeue columns and case routing model"`

### Task 3: 启动种子与存量迁移

**Files:**
- Modify: `apps/pixoma/cmd/pixoma/main.go` 或现有 bootstrap 组合根（业务库就绪后调用种子）
- Create: `internal/platform/topic/seed.go`（`EnsureDefaultTopic(ctx, repo) error`）
- Modify: `internal/runtime/infrastructure/persistence/gorm_task.go`（迁移函数 `MigrateLegacyTasks(ctx, db) (int, error)`）

**Interfaces:**
- Produces: `func EnsureDefaultTopic(ctx, repo topic.Repository) error`（无 `default` 则创建，幂等）
- Produces: `func MigrateLegacyTasks(ctx, db) (int, error)`：把 `status='queued' AND edge_id != '' AND lease_until < now` 的行清 edge_id/lease_until、置 `dispatch_topic='default'`

- [x] **Step 1: 写失败测试**：空库调用两次 `EnsureDefaultTopic` 只创建一次；`MigrateLegacyTasks` 处理混合行（旧 queued 归 default、running 不动、有效租约不动）。
- [x] **Step 2: 运行确认失败**：`go test ./internal/platform/topic/ ./internal/runtime/infrastructure/persistence/ -count=1` → FAIL。
- [x] **Step 3: 实现**：种子在业务库迁移后调用；迁移用单条 UPDATE + count 返回。
- [x] **Step 4: 运行通过 + 提交**：`go test ./internal/platform/topic/ ./internal/runtime/infrastructure/persistence/ -count=1` → PASS；`git commit -am "feat(topic): seed default topic and migrate legacy tasks"`

---

## 2. 条件协议

### Task 4: 条件包骨架：规则模型、校验与组合

**Files:**
- Create: `internal/runtime/domain/condition/rule.go`（Rule 类型与 JSON 解析）
- Create: `internal/runtime/domain/condition/validate.go`（`Validate(rule, reg) error`）
- Create: `internal/runtime/domain/condition/rule_test.go`

**Interfaces:**
- Produces: `type Rule struct { Field string; Op string; Value any; And []Rule; Or []Rule }`
- Produces: `func ParseRule(raw json.RawMessage) (Rule, error)`（叶子必须含 field/op；组合必须含非空 and/or；混合字段非法）
- Produces: `type OpSet = map[string]bool`；`var BuiltinOps = OpSet{"eq":true,"ne":true,"in":true,"gt":true,"gte":true,"lt":true,"lte":true,"exists":true}`
- Produces: `type Registry interface { ProviderFor(field string) (Provider, error); Attributes() []AttributeDescriptor }`

- [x] **Step 1: 写失败测试**：合法叶子/and/or 解析；缺 field、非法 op、空组合、`{and:[...], field:...}` 混合结构均报可定位错误。
- [x] **Step 2: 运行确认失败**：`go test ./internal/runtime/domain/condition/ -count=1` → FAIL。
- [x] **Step 3: 实现 rule.go/validate.go**：`ParseRule` 用 encoding/json 严格解析；`Validate` 递归检查 field 注册、op ∈ BuiltinOps、value 类型与 descriptor schema 一致（见 Task 5）。
- [x] **Step 4: 运行通过 + 提交**：PASS；`git add internal/runtime/domain/condition && git commit -m "feat(condition): rule model and validation"`

### Task 5: 属性提供方注册表与内置 provider

**Files:**
- Create: `internal/runtime/domain/condition/provider.go`（Provider/AttributeDescriptor/Registry）
- Create: `internal/runtime/domain/condition/user_provider.go`（`user.is_premium`，经注入的 `UserLookup func(ctx, userID) (isPremium *bool, err error)`）
- Create: `internal/runtime/domain/condition/case_provider.go`（`case.category`、`case.tags`，经注入的 `CaseLookup func(ctx, caseID) (category string, tags []string, err error)`）
- Create: `internal/runtime/domain/condition/provider_test.go`

**Interfaces:**
- Produces: `type AttributeDescriptor struct { Key, Context, Label string; Schema map[string]any }`
- Produces: `type Provider interface { Namespace() string; ListAttributes() []AttributeDescriptor; Value(ctx context.Context, field string) (any, error) }`
- Produces: `var ErrAttributeMissing = errors.New("condition: attribute missing")`
- Produces: `func NewRegistry(providers ...Provider) *Registry`；`func (r *Registry) Register(p Provider)`

- [x] **Step 1: 写失败测试**：注册 user/case provider 后 `Attributes()` 含 3 个 descriptor（schema 含 type/enum/description）；`ProviderFor("user.is_premium")` 命中；`ProviderFor("user.level")` 返回 `ErrUnknownField`；lookup 返回 nil 时 `Value` 返回 `ErrAttributeMissing`。
- [x] **Step 2: 运行确认失败** → 实现 provider.go + 两个 provider（schema：`user.is_premium` boolean；`case.category` string+enum `["image","video","audio"]`；`case.tags` array）→ 运行通过。
- [x] **Step 3: 提交**：`git commit -am "feat(condition): attribute providers and registry"`

### Task 6: 求值引擎与协议扩展性测试

**Files:**
- Create: `internal/runtime/domain/condition/evaluate.go`（`Evaluate(ctx, rule, reg) (bool, error)`）
- Create: `internal/runtime/domain/condition/evaluate_test.go`

**Interfaces:**
- Produces: `func Evaluate(ctx context.Context, rule Rule, reg *Registry) (bool, error)`
- Consumes: Task 4/5 的 Rule、Registry、Provider、ErrAttributeMissing

- [x] **Step 1: 写失败测试**：
  - and/or 短路：`and(false, ...)` 不再调用后续 provider（用计数 provider 断言）；
  - 缺失属性：`user.is_premium` 为 nil → `eq true` 求值 false；`exists` 求值 true；
  - provider 返回非 missing 错误 → `Evaluate` 返回该错误（不吞掉）；
  - 比较：eq/ne/in/gt/gte/lt/lte 对 bool/string/number/list 断言。
- [x] **Step 2: 运行确认失败** → 实现 evaluate.go（值比较先按 descriptor 归一；错误包装 `evaluate <field>: %w`）→ PASS。
- [x] **Step 3: 扩展性测试**：测试内 `Register` 一个新 provider（如 `user.region`，string+enum），不改 evaluate 代码即可 `Validate` + `Evaluate` 通过——这是「新增条件不改引擎」的契约测试。
- [x] **Step 4: 提交**：`git commit -am "feat(condition): rule evaluation engine"`

---

## 3. Topic 领域与 Admin API

### Task 7: Topic CRUD API 与 default 保护

**Files:**
- Create: `internal/httpapi/topics/handler.go`（chi 路由：`GET/POST /api/v1/topics`、`GET/PUT/DELETE /api/v1/topics/{key}`）
- Create: `internal/httpapi/topics/handler_test.go`
- Modify: `internal/httpapi/adminhost/server.go`（挂载 topics 路由）

**Interfaces:**
- Consumes: `topic.Repository`
- Produces: 载荷 `{"key","name","enabled","created_at","updated_at"}`；POST 非法 key（regex）→ 400；重复 → 409；DELETE `default` → 409；DELETE 被 Case 规则/Edge 订阅引用 → 409（引用检查经注入的 `ReferenceCount func(ctx, key) (int, error)`，本期可返回 0 并记录 TODO？——不可，见 Step 3 实现引用计数：Case 路由引用与 edges 订阅引用两处 count）

- [x] **Step 1: 写失败测试**：创建→列表可见；非法 key 400；重复 409；PUT 改名/禁用生效；DELETE default 409；DELETE 被引用 409；DELETE 成功后列表消失。
- [x] **Step 2: 运行确认失败** → 实现 handler + service 逻辑（key 校验用共享 regex `var KeyPattern = regexp.MustCompile(...)`；引用计数查询 Case doc JSON 与 edges 订阅 JSON——先实现 edges 引用计数，Case 引用计数在本 Task 一并做，用 GORM `LIKE` 扫 `doc_json`/`subscribe_topics_json`，量小可接受）→ 挂路由 → PASS。
- [x] **Step 3: 提交**：`git commit -am "feat(topics): admin CRUD api with default protection"`

### Task 8: 节点订阅绑定（presence 声明 + 管理端覆盖）

**Files:**
- Modify: `internal/platform/edge/`（Edge 记录 `SubscribeTopics []string`；`EffectiveTopics() []string` 归一化）
- Modify: `internal/httpapi/edges/handler.go`（PATCH 支持 `subscribe_topics`，校验每个 key 存在且启用；列表/详情返回 `subscribe_topics` 与归一化 `effective_topics`）
- Modify: `apps/edge-agent` presence 上报载荷（可选 `subscribe_topics`）与控制面 `/agent/v1/presence` 处理（首次报到写入，之后不覆盖；复用 hardware 首次语义）
- Create: `internal/httpapi/edges/topic_binding_test.go`

**Interfaces:**
- Produces: `func (e *Edge) EffectiveTopics() []string`（空→`["default"]`，去重保序）
- Consumes: `topic.Repository`（校验存在+启用）

- [x] **Step 1: 写失败测试**：PATCH 空数组→`effective_topics=["default"]`；PATCH 含禁用/不存在 key → 400；presence 首次携带 `["fast-gpu"]` 写入；presence 再次不覆盖管理端 PATCH 值；重启 Edge 不重置。
- [x] **Step 2: 运行确认失败** → 实现 → PASS。
- [x] **Step 3: 提交**：`git commit -am "feat(edges): topic subscription binding"`

### Task 9: Case 路由配置校验与 case-admin-api

**Files:**
- Modify: `internal/catalog/domain/document.go`（routing 模型已在 Task 2）
- Modify: `internal/catalog/infrastructure/validation/validate.go`（`ValidateRouting(routing, topicRepo, conditionReg)`）
- Modify: `internal/httpapi/cases/handler.go`（创建/更新/详情支持 `routing`）
- Create: `internal/catalog/infrastructure/validation/routing_test.go`

**Interfaces:**
- Produces: `func ValidateRouting(r *RoutingConfig, topics topic.Repository, reg *condition.Registry) error`（规则非空时每条 when 通过 `condition.Validate`；topic 存在且启用；返回含 rule index 的错误）

- [x] **Step 1: 写失败测试**：合法 routing 通过；引用不存在/禁用 Topic 拒绝（错误含 Topic key 与 rule index）；条件未知字段拒绝；`routing: null` 通过。
- [x] **Step 2: 运行确认失败** → 实现 validate.go 扩展 + handler 载荷字段（`routing` 可选）→ PASS。
- [x] **Step 3: 提交**：`git commit -am "feat(cases): routing config validation and api"`

### Task 10: 条件目录 API

**Files:**
- Create: `internal/httpapi/routing/handler.go`（`GET /api/v1/routing/attributes`）
- Create: `internal/httpapi/routing/handler_test.go`
- Modify: `internal/httpapi/adminhost/server.go`（挂载）

**Interfaces:**
- Consumes: `*condition.Registry`
- Produces: `{"attributes":[{"key","label","context","schema"}]}`

- [x] **Step 1: 写失败测试**：返回 3 个内置属性且 schema 含 type/enum；`task-flow-editor` 契约字段齐全。
- [x] **Step 2: 实现 + PASS + 提交**：`git commit -am "feat(routing): attributes catalog api"`

---

## 4. 调度改造

### Task 11: Router.Resolve 与 Orchestrator 按 Topic 调度

**Files:**
- Create: `internal/runtime/application/routing/router.go`（`Resolve(routing *RoutingConfig, ctx, reg) (string, error)`）
- Create: `internal/runtime/application/routing/router_test.go`
- Modify: `internal/runtime/application/orchestrator/service.go`（`dispatchTask` 改为路由求值；求值错误保持 pending + `error_message`；`PrepareForClaim` 传 topic）
- Modify: `internal/runtime/infrastructure/actuator/snapshot.go` 或 prep 实现（`PrepareJob(ctx, taskID)` 去掉 edgeID，job 包保持 edge 无关）
- Modify: `internal/runtime/domain/task_repository.go` + memory/gorm 实现（`PrepareForClaim(ctx, id, topic, jobRef, now)`）

**Interfaces:**
- Produces: `func Resolve(r *RoutingConfig, ctx context.Context, reg *condition.Registry, evalCtx *condition.EvalContext) (string, error)`——返回首个命中规则的 Topic；无规则/无命中返回 `"default"`；求值错误原样返回
- Produces: `type EvalContext struct { UserID, CaseID string; User *condition.UserProvider; Case *condition.CaseProvider }`（按设计文档 §4.2）
- Consumes: `condition.Evaluate`、`PrepareForTopic`、`JobPreparer`

- [x] **Step 1: 写失败测试（routing/router_test.go）**：首条命中即投；无命中回退 default；无规则回退 default；provider 错误向上传播；顺序敏感（两条规则都命中时取第一条）。
- [x] **Step 2: 写失败测试（orchestrator/service_test.go 改造）**：pending Task 路由命中 `fast-gpu` 后 `PrepareForTopic` 被调且 `dispatch_topic=fast-gpu`、`edge_id` 为空；无在线订阅节点保持 pending；求值错误保持 pending 且 error_message 含 `routing:`；`SchedulePending` 兜底路径同语义。
- [x] **Step 3: 运行确认失败** → 实现 router + service 改造（组合根注入 provider/registry/evalCtx 构建）→ 全绿。
- [x] **Step 4: 提交**：`git commit -am "feat(orchestrator): topic-based routing dispatch"`

---

## 5. 原子领取

### Task 12: ClaimNextWithLease 按 Topic 集合 CAS

**Files:**
- Modify: `internal/runtime/domain/task_repository.go`（接口改 `ClaimNextWithLease(ctx, topics []string, lease time.Duration, now time.Time) (*Task, error)`）
- Modify: `internal/runtime/domain/task_repository.go`（memory 实现）
- Modify: `internal/runtime/infrastructure/persistence/gorm_task.go`（GORM 实现 + 索引迁移）
- Create: `internal/runtime/infrastructure/persistence/gorm_topic_claim_test.go`

**Interfaces:**
- Consumes: Task 2 的 `PrepareForTopic`/`ClaimWithLease`、Task 11 的 `dispatch_topic`
- Produces: 原子领取语义：扫描 `status='queued' AND (dispatch_topic IN (topics) OR dispatch_topic='') AND job_ref 非空 AND (requeue_at IS NULL OR requeue_at <= now)`，ORDER BY created_at, id；事务内条件 UPDATE 抢 `running`

- [x] **Step 1: 写失败测试（单线程语义）**：topic 过滤（只领订阅集合内）；requeue_at 未到期不可领、到期可领；租约写入正确；`dispatch_topic=''` 旧行可领（兜底）。
- [x] **Step 2: 写并发契约测试**：N=8 goroutine 同时 `ClaimNextWithLease(topics, ...)` 抢 M=20 个任务（sqlite 内存库），断言：恰好 20 个成功、结果无重复 task id、失败者返回 nil 且无错。
- [x] **Step 3: 运行确认失败** → 实现 GORM（沿用现有事务内 select-then-conditional-update 循环模式，WHERE 增加 topic 集合与 requeue_at；`Updates` 写 status/edge_id/lease_until）→ 单测 + 并发测试 PASS（并发测试可 `-race`）。
- [x] **Step 4: 提交**：`git commit -am "feat(runtime): atomic topic-scoped claim"`

### Task 13: claim 端点按订阅领取 + 长轮询

**Files:**
- Modify: `internal/httpapi/edges/handler.go` 或现有 agent claim handler（`GET /agent/v1/jobs/claim?edge_id=&wait=`）
- Modify: `internal/platform/edge/`（registry 暴露 `EffectiveTopicsByID(edgeID)`）
- Create: `internal/httpapi/edges/claim_topic_test.go`

**Interfaces:**
- Consumes: Task 12 的仓储方法、Task 8 的订阅绑定
- Produces: claim 响应形状不变（`task_id/edge_id/job_ref/lease_until`）；无任务/竞争落败返回 204；长轮询循环重扫（等待上限默认 30s，可配）

- [x] **Step 1: 写失败测试**：Edge 订阅 `["default","fast-gpu"]` 时可领两个 Topic 的任务、领不到其它 Topic；未配置订阅的 Edge 只领 default；两节点并发 HTTP claim 同一任务恰好一个 200 一个 204。
- [x] **Step 2: 运行确认失败** → 实现（handler 用 edge registry 解析 topics → repo.ClaimNextWithLease；轮询用现有 wait 循环）→ PASS。
- [x] **Step 3: 提交**：`git commit -am "feat(agent): topic-scoped claim endpoint"`

---

## 6. 失败重回与回收

### Task 14: 失败有界重试（attempts + 退避 + 超限收口）

**Files:**
- Modify: `internal/runtime/domain/task.go`（`RequeueAfterFailure` 领域方法）
- Modify: `internal/runtime/domain/task_repository.go` + memory/gorm（`RequeueAfterFailure(ctx, id, backoff, now, reason) error`）
- Modify: `internal/runtime/application/orchestrator/service.go`（`applyStatus` failed 分支接线）
- Create: `internal/runtime/application/orchestrator/retry_test.go`

**Interfaces:**
- Produces: `var MaxAttempts = 3`；`var BackoffSchedule = []time.Duration{5*time.Second, 15*time.Second, 45*time.Second}`
- Produces: `func BackoffFor(attempts int) time.Duration`（`attempts>=len(schedule)` 用最后一项）
- Consumes: Task 12 的 `requeue_at` 过滤（未到期不可领）

- [x] **Step 1: 写失败测试（domain）**：attempts=0→1 回 queued（保留 dispatch_topic、清 edge_id/lease、requeue_at=now+5s）；attempts=2→3 时不再回 queued 而是终态 failed（返回 `ErrMaxAttempts` 或由调用方收口，二选一并写死在实现）；`BackoffFor(1/2/3/99)` 断言。
- [x] **Step 2: 写失败测试（orchestrator）**：status=failed 且未达上限 → Task 重新可领（attempts+1、requeue_at 未来）；达上限 → 终态 failed + 通知；重复 failed status 幂等（已 failed 不再 +1）。
- [x] **Step 3: 运行确认失败** → 实现（applyStatus 统一入口内处理；收口走既有 failed+notify）→ PASS。
- [x] **Step 4: 提交**：`git commit -am "feat(runtime): bounded retry with backoff"`

### Task 15: 租约过期回收（不计 attempts）+ ExecutionQuery 收口

**Files:**
- Modify: `internal/runtime/infrastructure/persistence/gorm_task.go`（`RequeueExpiredLeases` 扩展：清 edge_id/lease、置 requeue_at=now、`attempts` 不变、保留 dispatch_topic）
- Modify: `internal/runtime/application/orchestrator/service.go`（ReconcileStale 周期接线；病理场景复用 ExecutionQuery 对账收口 failed）
- Modify: 既有 `internal/runtime/application/orchestrator/service_test.go`（租约回收场景扩展）

**Interfaces:**
- Consumes: Task 12 的 claim 过滤（回收后立即可领，因为 requeue_at=now）

- [x] **Step 1: 写失败测试**：queued/running 租约过期 → 回收后同 Topic 可再领且 attempts 不变；有效租约不动；回收后 heartbeat 旧 holder 失败（非 holder 拒延）。
- [x] **Step 2: 运行确认失败** → 实现 → PASS。
- [x] **Step 3: 提交**：`git commit -am "feat(runtime): lease expiry requeue without burning attempts"`

### Task 16: StormGuard 复用与回收风暴防护

**Files:**
- Modify: `internal/runtime/application/orchestrator/service.go`（SchedulePending/ReconcileStale/retry 走 StormGuard 限流与分池）
- Create: `internal/runtime/application/orchestrator/storm_retry_test.go`

**Interfaces:**
- Consumes: 现有 `StormGuard`（`AllowSchedule` 等）

- [x] **Step 1: 写失败测试**：批量失败任务进入回收时，单轮回收数量受限；调度/回收分池互不阻塞（并发断言）。
- [x] **Step 2: 实现 + PASS + 提交**：`git commit -am "feat(orchestrator): storm guards for requeue and reconcile"`

---

## 7. 端到端与兼容

### Task 17: allinone/Mock 主路径（默认 Topic）

**Files:**
- Modify: `apps/pixoma/internal/app/`（组合根：topic 仓储、条件 provider、router 注入；本地 Edge 默认订阅 default）
- Modify: `test/integration/smoke_test.go`（无路由 Case → default Topic → 领取执行成功）
- Modify: `apps/pixoma/internal/app/smoke_test.go`

**Interfaces:**
- Consumes: Task 3/8/11/12/13 全部产物

- [x] **Step 1: 写失败测试**：allinone + `comfy_mock`：ConfirmRun → Task 路由 default → 本地 Edge 领取 → running → succeeded → 通知意图；断言 `dispatch_topic=default`。
- [x] **Step 2: 运行确认失败** → 修组合根装配 → `go test ./apps/pixoma/... ./test/integration/ -count=1` PASS。
- [x] **Step 3: 提交**：`git commit -am "test(e2e): allinone mock default topic path"`

### Task 18: split 多 Topic 订阅端到端

**Files:**
- Modify: `apps/edge-agent` 配置解析（`subscribe_topics` yaml + `EDGE_SUBSCRIBE_TOPICS` 环境变量逗号分隔）
- Modify: `test/integration/`（split 场景：两个 Edge 各订阅不同 Topic；Case 路由 `user.is_premium=true → fast-gpu`）
- Create: `apps/edge-agent/internal/pull/config_test.go`

**Interfaces:**
- Produces: `func ParseSubscribeTopics(raw string) []string`（yaml 列表 / 逗号分隔 → 归一化）

- [x] **Step 1: 写失败测试**：配置解析（空→default 语义）；split 集成：Task 命中 fast-gpu → 仅订阅 fast-gpu 的 Edge 可领并成功；另一 Edge 领不到。
- [x] **Step 2: 运行确认失败** → 实现 → `go test ./apps/edge-agent/... ./test/integration/ -count=1` PASS。
- [x] **Step 3: 提交**：`git commit -am "test(e2e): split edge multi-topic subscription"`

### Task 19: 兼容收尾与文档同步

**Files:**
- Modify: `docs/architecture/data-model.md`（topics 表、edges/tasks 新列、Case routing）
- Modify: `docs/architecture/runtime.md`（调度按 Topic、claim 语义、失败重回/回收）
- Modify: `docs/architecture/task-data-walkthrough.md`（样例数据含 dispatch_topic）
- Modify: `README.md`（EDGE_SUBSCRIBE_TOPICS、Topic 概念）
- Modify: `configs/`（样例配置补 `subscribe_topics` 示例）

**Interfaces:**
- 无（文档与配置）

- [x] **Step 1: 写文档与样例**：按设计文档 §10 同步；样例配置补 `subscribe_topics: []` 注释说明默认语义。
- [x] **Step 2: 全量验证**：`go build ./...` + `go test ./... -count=1`（确认既有测试未被破坏）。
- [x] **Step 3: 提交**：`git commit -am "docs(topic): sync architecture docs and config samples"`

---

## Self-Review 结论

- **Spec 覆盖**：tasks.md 7 组 31 项全部映射到 Task 1-19（数据模型 1-3、条件协议 4-6、Topic/API 7-10、调度 11、原子领取 12-13、失败回收 14-16、端到端 17-19）；delta spec 的求值错误场景由 Task 11 覆盖、并发互斥由 Task 12/13 覆盖、默认 Topic 由 Task 3/8/17 覆盖。
- **无占位符**：所有代码步骤给出签名或测试断言；HTTP 载荷形状引用设计文档 §7。
- **类型一致性**：`dispatch_topic`/`attempts`/`requeue_at`、`PrepareForTopic`、`ClaimNextWithLease(topics,...)`、`BackoffFor` 在 Task 2/11/12/14 间保持一致；`NormalizeTopics`/`EffectiveTopics` 在 Task 2/8 间一致。
