# 模块端口（Go interface）与 ConfirmRun 后事件流

> 设计目标：同进程可直接函数调用实现端口；跨进程时端口背后换成 Queue/HTTP 适配器，**领域代码不改。**

---

## 1. 共享内核类型（示意）

```go
package kernel

type CaseID string
type SessionID string
type TaskID string
type ChatID int64
type InstanceID string

type TaskStatus string

const (
    TaskPending   TaskStatus = "pending"
    TaskQueued    TaskStatus = "queued"
    TaskRunning   TaskStatus = "running"
    TaskSucceeded TaskStatus = "succeeded"
    TaskFailed    TaskStatus = "failed"
    TaskCancelled TaskStatus = "cancelled"
)

// BlobRef 对象存储引用，不塞字节进消息。
type BlobRef struct {
    Bucket string
    Key    string
    MIME   string
    Size   int64
}

type InputValue struct {
    Key   string
    // 文本/数字/枚举等标量
    Text  *string
    Number *float64
    Bool  *bool
    // 媒体：已物化后的引用
    Blob  *BlobRef
}
```

---

## 2. 各模块对外端口

### 2.1 `protocol` — 校验（纯领域，无 IO）

```go
package protocol

type Validator interface {
    // doc 为 Case 协议文档；values 为逻辑输入。
    ValidateInputs(doc CaseDocument, values []InputValue) error
}
```

### 2.2 `case`（registry）

```go
package caseregistry

type Repository interface {
    Get(ctx context.Context, id CaseID) (*CaseAggregate, error)
    List(ctx context.Context, q ListQuery) ([]*CaseSummary, error)
    Save(ctx context.Context, c *CaseAggregate) error // 注册/更新
}

// 应用层只读查询可再包一层：
type QueryService interface {
    GetMenu(ctx context.Context) ([]MenuItem, error)
    ListCategories(ctx context.Context, menuKey string) ([]Category, error)
    ListCases(ctx context.Context, menuKey, category string, page Page) (*CasePage, error)
    GetCase(ctx context.Context, id CaseID) (*CaseAggregate, error)
}
```

### 2.3 `session`

```go
package session

type Repository interface {
    GetActiveByChat(ctx context.Context, chatID ChatID) (*Session, error)
    Save(ctx context.Context, s *Session) error
    // 终态后清除 active 索引，保留历史可选
    ClearActive(ctx context.Context, chatID ChatID) error
}

type Service interface {
    StartCase(ctx context.Context, chatID ChatID, caseID CaseID) (*Session, error) // 锁冲突返回 ErrSessionLocked
    SubmitInput(ctx context.Context, chatID ChatID, v InputValue) (*Session, error)
    SkipInput(ctx context.Context, chatID ChatID) (*Session, error)
    Exit(ctx context.Context, chatID ChatID) error
    // Confirm 前：进入 confirming 或返回当前视图
    BeginConfirm(ctx context.Context, chatID ChatID) (*Session, error)
    Get(ctx context.Context, chatID ChatID) (*Session, error)
}
```

### 2.4 `task`

```go
package task

type Repository interface {
    Create(ctx context.Context, t *Task) error
    Get(ctx context.Context, id TaskID) (*Task, error)
    Update(ctx context.Context, t *Task) error
    ListByChat(ctx context.Context, chatID ChatID, q ListTaskQuery) ([]*Task, error)
    // Orchestrator 用：
    ListByStatus(ctx context.Context, st TaskStatus, limit int) ([]*Task, error)
}

// 状态迁移由领域方法保证合法边，不散落在各处乱写 Status。
type Task struct { /* ... */ }

func (t *Task) MarkQueued(instance InstanceID) error
func (t *Task) MarkRunning(promptID string) error
func (t *Task) MarkSucceeded(outputs []OutputRef) error
func (t *Task) MarkFailed(code, msg string) error
func (t *Task) MarkCancelled() error // 仅 pending/queued
```

### 2.5 `app` — 用例门面（给 adapter 用）

```go
package app

type Facade interface {
    caseregistry.QueryService // 或嵌入菜单/列表方法

    StartCase(ctx context.Context, in StartCaseCmd) (*SessionView, error)
    SubmitInput(ctx context.Context, in SubmitInputCmd) (*SessionView, error)
    SkipInput(ctx context.Context, in SkipInputCmd) (*SessionView, error)
    ExitSession(ctx context.Context, in ExitCmd) error
    ConfirmRun(ctx context.Context, in ConfirmRunCmd) (*ConfirmRunResult, error)

    GetSession(ctx context.Context, chatID ChatID) (*SessionView, error)
    ListMyTasks(ctx context.Context, chatID ChatID) ([]TaskView, error)

    // 渠道通知入口（由 notify 适配器回调进 app，或 tg 直接实现 NotifyHandler）
    DeliverNotify(ctx context.Context, n UserNotify) error
}

type ConfirmRunResult struct {
    TaskID TaskID
    Status TaskStatus // pending
}
```

### 2.6 Ports（基础设施，可替换）

```go
package blob

type Store interface {
    Put(ctx context.Context, key string, r io.Reader, opts PutOptions) (BlobRef, error)
    Get(ctx context.Context, ref BlobRef) (io.ReadCloser, error)
    // StageInputs: 把 session 草稿变成 inputs/{task_id}/...
    // 也可放在 app 领域服务里调 Store
}

package queue

type Message struct {
    Topic   string
    Key     string // 分区/幂等键，如 task_id
    Payload []byte // JSON，见下文事件
}

type Publisher interface {
    Publish(ctx context.Context, msg Message) error
}

type Handler func(ctx context.Context, msg Message) error

type Subscriber interface {
    Subscribe(ctx context.Context, topic string, h Handler) error
}

package notifyport

// Orchestrator → 渠道：只表达「请通知用户」，不含 TG API 类型。
type Publisher interface {
    Publish(ctx context.Context, n UserNotify) error
}

type UserNotify struct {
    ChatID   ChatID
    TaskID   TaskID
    Kind     string // task_succeeded | task_failed | task_progress
    Title    string
    Outputs  []BlobRef
    ErrorMsg string
}

package instance

type Registry interface {
    ListHealthy(ctx context.Context, filter CapabilityFilter) ([]Instance, error)
    Get(ctx context.Context, id InstanceID) (*Instance, error)
}

type Instance struct {
    ID           InstanceID
    DispatchTopic string // dispatch.<id>
    Capabilities []string
    // BaseURL 仅 Actuator 本地配置需要，Registry 可只存路由元数据
}
```

### 2.7 `orchestrator`

```go
package orchestrator

// 入站：新任务可编排
type Inbound interface {
    OnTaskCreated(ctx context.Context, ev TaskCreated) error
    // 或 RunOnce 扫描 pending（同进程可用）
    SchedulePending(ctx context.Context, limit int) error
}

// 入站：执行面状态
type StatusHandler interface {
    OnStatus(ctx context.Context, ev TaskStatusEvent) error
}

// 出站依赖（构造注入）
type Deps struct {
    Tasks      task.Repository
    Instances  instance.Registry
    Dispatch   queue.Publisher // 发到 dispatch.<id>
    Notify     notifyport.Publisher
    // 可选：订阅用 Subscriber 在 main 里绑到 StatusHandler
}

type Service interface {
    Inbound
    StatusHandler
    RequestCancel(ctx context.Context, taskID TaskID) error // 温和取消
}
```

### 2.8 `actuator`

```go
package actuator

type Deps struct {
    Tasks   task.Repository // 或只通过 status 上报、由 orchestrator 写库——推荐 Actuator 也写 running 关键字段时用仓库；终态以 status 事件为准亦可双写谨慎
    Blob    blob.Store
    Comfy   comfyui.Client
    Status  queue.Publisher // 发 status
    InstanceID InstanceID
}

type Worker interface {
    // 由 queue.Subscriber 调起
    HandleDispatch(ctx context.Context, ev DispatchCommand) error
}

package comfyui

type Client interface {
    Submit(ctx context.Context, graph WorkflowGraph) (promptID string, err error)
    Wait(ctx context.Context, promptID string) (*ComfyResult, error)
    // 或 Watch + FetchOutputs
}
```

### 2.9 `adapter/tg`

```go
package tgadapter

type Deps struct {
    App app.Facade
    // Bot API client from go-telegram/bot
}

// 实现 notify 消费：
func (a *Adapter) HandleUserNotify(ctx context.Context, n notifyport.UserNotify) error {
    return a.App.DeliverNotify(ctx, n) // 或直接发 TG
}
```

---

## 3. 事件契约（JSON Payload）

Topic 名稳定；跨进程靠 Queue，同进程可用 `bus.Publish` 内存实现同一接口。

| Topic | 生产者 | 消费者 | Payload |
|-------|--------|--------|---------|
| `task.created` | app（ConfirmRun） | orchestrator | `TaskCreated` |
| `dispatch.<instance_id>` | orchestrator | actuator | `DispatchCommand` |
| `task.status` | actuator | orchestrator | `TaskStatusEvent` |
| `notify.user` | orchestrator | tg adapter | `UserNotify` |

```go
type TaskCreated struct {
    TaskID     TaskID    `json:"task_id"`
    ChatID     ChatID    `json:"chat_id"`
    CaseID     CaseID    `json:"case_id"`
    CreatedAt  time.Time `json:"created_at"`
}

type DispatchCommand struct {
    TaskID     TaskID     `json:"task_id"`
    InstanceID InstanceID `json:"instance_id"`
    // 执行所需：快照可放 DB，消息只带 ID；或带只读 snapshot_ref
    CaseSnapshotRef string   `json:"case_snapshot_ref,omitempty"`
    InputPrefix     string   `json:"input_prefix"` // blob key prefix
}

type TaskStatusEvent struct {
    TaskID     TaskID     `json:"task_id"`
    InstanceID InstanceID `json:"instance_id"`
    Status     TaskStatus `json:"status"` // running|succeeded|failed|...
    PromptID   string     `json:"prompt_id,omitempty"`
    Outputs    []BlobRef  `json:"outputs,omitempty"`
    ErrorCode  string     `json:"error_code,omitempty"`
    ErrorMsg   string     `json:"error_msg,omitempty"`
    At         time.Time  `json:"at"`
}
```

**幂等键**：`Message.Key = task_id`（status 可用 `task_id + status + at` 哈希防重）。

---

## 4. ConfirmRun 之后：模块间传递（同进程 / 跨进程同一语义）

```text
ConfirmRun(app)
  │
  ├─1. session.BeginConfirm / 校验 draft
  ├─2. protocol.ValidateInputs(caseDoc, values)
  ├─3. blob 物化 → inputs/{task_id}/…
  ├─4. task.Repository.Create(pending) + case snapshot 落库
  ├─5. session.ClearActive（submitted 解锁）
  └─6. queue.Publish("task.created", TaskCreated)
           │
           ▼
orchestrator.OnTaskCreated
  │
  ├─7. instance.Registry.ListHealthy(filter from case tags)
  ├─8. 选定 InstanceID
  ├─9. task.MarkQueued(instance)
  └─10. queue.Publish("dispatch."+id, DispatchCommand)
           │
           ▼
actuator.HandleDispatch
  │
  ├─11. 拉 blob + 注入 + comfy.Submit
  ├─12. Publish status(running, prompt_id)
  ├─13. Wait / 拉产物 → blob outputs/
  └─14. Publish status(succeeded|failed, outputs|error)
           │
           ▼
orchestrator.OnStatus
  │
  ├─15. task.MarkSucceeded / MarkFailed（幂等）
  └─16. notify.Publish(UserNotify)
           │
           ▼
tgadapter.HandleUserNotify
  └─17. sendPhoto / editMessage → 用户
```

### 同进程组装

```go
// cmd 里：
mem := memoryqueue.New()
orch.Subscribe(mem, "task.created", orch.OnTaskCreated)
orch.Subscribe(mem, "task.status", orch.OnStatus)
act.Subscribe(mem, "dispatch."+localID, act.HandleDispatch)
tg.Subscribe(mem, "notify.user", tg.HandleUserNotify)
```

全部是 `queue.Publisher/Subscriber`；Memory 实现即 all-in-one。

### 跨进程组装

同一接口，换 `nats.Publisher` 等；**app / orchestrator / actuator 源码不改。**

---

## 5. ConfirmRun 时序（谁依赖谁）

```text
tgadapter → app.ConfirmRun
               → session + protocol + caseRepo + blob + taskRepo
               → publisher.Publish(task.created)
             return task_id to user（排队中）

异步：
task.created → orchestrator → dispatch.* → actuator → task.status
 → orchestrator → notify.user → tgadapter → 用户
```

同步路径不阻塞 ComfyUI；异步路径不经过 session 锁。

---

## 6. 设计要点（冻结候选）

1. **事件优于直接交叉调用**：app 不 import actuator；orchestrator 不 import tg。  
2. **同进程也走端口**：避免以后拆开时改领域。  
3. **DB 是任务真相**；消息是驱动；**Task 终态只由 Orchestrator 写库**；Actuator 只发 status。  
4. **消息尽量带 ID + BlobRef**，大对象不进 Queue。  
5. **notify 与 dispatch/status 分离**：渠道可换（TG/其它）不影响执行面。  
6. **status 丢失靠对账兜底**（见 §8）；队列至少 at-least-once + 幂等消费。

---

## 7. 目录落点（与端口对应）

```text
internal/
  kernel/           # IDs, BlobRef, 事件 DTO
  domain/protocol/
  domain/case/
  domain/session/
  domain/task/
  app/
  orchestrator/
  actuator/
  adapter/tg/
  port/queue/
  port/blob/
  port/notify/
  port/instance/
  comfyui/
  adapter/queue/memory/
  adapter/blob/localfs/
```

`port/*` 只含 interface；`adapter/*` 含实现。

---

## 8. Status 丢失时的兜底对账（Orchestrator 独占写库）

主路径仍是：Actuator → `task.status` → Orchestrator 写 DB。  
网络/进程崩溃可能导致 status 没到。兜底不是让 Actuator 改写 Task，而是 **Orchestrator 主动核对「外部事实」**。

### 8.1 分层手段（由近到远）

| 层 | 做什么 | 作用 |
|----|--------|------|
| **队列可靠投递** | at-least-once；Actuator 本地 outbox（先落库再发 status，定期重发未确认） | 减少「发了但丢了」 |
| **消费幂等** | Orchestrator 按 `task_id + status + version/at` 去重 | 重发不重复 notify |
| **超时扫描（控制面对账）** | 定时扫 DB：`queued/running` 且超过阈值 | 发现「该有结果却一直没有」 |
| **向执行面探测** | Orchestrator → Actuator **Query 端口**（或查 Blob 约定路径） | 用真相补写 Task |
| **仍不可知则失败收口** | 多次探测失败 / 超过最大存活时间 → `failed(timeout/unknown)` + notify | 不让任务永远挂起 |

### 8.2 Actuator 侧建议（仍不写 Task 终态）

```go
// actuator 本地执行账（可 SQLite/文件），不是全局 Task 表
type RunLedger interface {
    Save(run LocalRun) error          // task_id, prompt_id, state, output_keys, updated_at
    Get(taskID TaskID) (*LocalRun, error)
    ListNeedStatusRetry(limit int) ([]LocalRun, error)
}
```

- 执行进度先写 **LocalRun**；再 Publish status。  
- 若 Publish 失败或未收到「编排侧 ack」（可选），后台 **重发 status**。  
- 对外提供查询（同进程=函数；跨进程=HTTP/队列 RPC）：

```go
type ExecutionQuery interface {
    // Orchestrator 对账时调用
    GetRun(ctx context.Context, taskID TaskID) (*ExecutionView, error)
}

type ExecutionView struct {
    TaskID   TaskID
    Phase    string // accepted|running|succeeded|failed|unknown
    PromptID string
    Outputs  []BlobRef
    ErrorMsg string
}
```

### 8.3 Orchestrator 对账循环（伪逻辑）

```text
每 N 秒:
  for task in DB where status in (queued, running) and stale(updated_at):
      view := actuatorQuery.GetRun(task.ID)   // 经 instance 路由到对应 Actuator
      if view.succeeded:
          MarkSucceeded + notify          # 补写，与 OnStatus 同一套幂等入口
      else if view.failed:
          MarkFailed + notify
      else if view.running and within_sla:
          续命 / 更新 heartbeat 字段
      else if exceeded_max_age:
          MarkFailed("reconcile_timeout") + notify
      else if actuator unreachable:
          记探测失败次数；超阈值再 failed 或告警人工
```

也可增加 **Blob 启发式**：若 `outputs/{task_id}/` 已出现约定清单文件，即使 status 丢了也可视为成功线索（仍建议以 Actuator Query 为准）。

### 8.4 原则

- **写 Task 终态的入口只有一个**：`orchestrator.applyStatus(ev)`（OnStatus 与 Reconcile 都走它）。  
- Actuator **可以**有本地账本，**不可以**直接 `UPDATE tasks SET status=…`。  
- 对账是补偿，不是主路径；主路径仍是 status 事件。

### 8.5 调用风暴防护（大面积失败 / 重试）

Orchestrator 是控制面汇聚点：对账扫库、重派 dispatch、重问 Actuator、刷 notify，**任一环节在故障时都可能放大成风暴**。必须内建节流，而不是「失败就立刻全量重试」。

#### 风险场景

| 场景 | 若不加限制会发生什么 |
|------|----------------------|
| 某实例宕机 | 大量 running 同时超时 → 同时 Query / 同时重派其它实例 |
| ComfyUI 集体变慢 | 对账误判超时 → 重复 dispatch 打爆实例 |
| status 通道抖动 | Actuator 重发 + 对账补写 + 重复 notify |
| DB 短暂不可用恢复 | SchedulePending 积压一次性打满 |

#### 防护策略（冻结）

1. **分层限流 / 预算**  
   - 全局：每 tick 最多处理 N 条 pending、M 条 reconcile、P 条 redispatch  
   - 按 `instance_id`：单实例同时 in-flight dispatch / Query 上限  
   - 按 `chat_id` / 租户（预留）：避免单用户拖垮控制面  

2. **指数退避 + 抖动（jitter）**  
   - 同一 `task_id` 的重试：`backoff = min(cap, base * 2^attempt) + jitter`  
   - 禁止固定间隔对齐（雷群效应）；对账扫描加随机错峰  

3. **错误分类**  
   - **可重试**（网络超时、429、实例短暂不健康）：进退避队列  
   - **不可重试**（校验失败、Case 停用、输入 Blob 缺失）：直接 `failed`，不重派  
   - **实例级熔断**：连续失败超阈值 → 实例标记 `unhealthy`，调度跳过，半开探测  

4. **幂等 + 去重窗口**  
   - `applyStatus` / `notify` 按 `task_id + status + 结果指纹` 去重  
   - 重派前检查：已 `running/succeeded` 禁止再 dispatch；仅 `pending/queued` 或明确 `retryable_failed` 策略允许  

5. **对账与调度分池**  
   - `SchedulePending` 与 `ReconcileStale` **分开配额**，避免对账占满导致新任务饿死，或相反  
   - 对账优先「最老 / 最超时」，不要每次全表乱序打满  

6. **重派风暴刹车**  
   - 单次故障事件（如实例下线）触发的 bulk redispatch：**令牌桶**分批，而不是 foreach 同步打光  
   - 可配置 `max_redispatch_per_minute`  

7. **Notify 风暴**  
   - 用户通知同样限流；进度类 notify 可合并（同一 task 在窗口内只发最后一条）  
   - 终态 notify 至少一次，但去重保证不刷屏  

8. **可观测**  
   - 指标：pending 积压、reconcile 队列长度、每实例失败率、retry 次数分布、熔断状态  
   - 积压超阈值告警，而不是默默加大重试  

#### 端口级体现（示意）

```go
type RetryPolicy interface {
    NextDelay(taskID TaskID, attempt int, class ErrorClass) (time.Duration, bool) // false=放弃
}

type RateLimiter interface {
    Allow(ctx context.Context, bucket string, n int) bool
}

type CircuitBreaker interface {
    Allow(instanceID InstanceID) bool
    RecordSuccess(instanceID InstanceID)
    RecordFailure(instanceID InstanceID)
}
```

Orchestrator 的 `SchedulePending` / `OnStatus` / `ReconcileStale` / `Redispatch` 都必须经过上述策略，默认 **保守配额**，用配置调大而不是默认打满。

---

## 9. 「task.created 队列」vs「扫 DB pending」是什么意思？

这是 **Orchestrator 怎么知道「有新任务要调度」** 的两种触发方式，不是两套业务。

### 方式 A — 事件驱动（ConfirmRun 后 Publish `task.created`）

```text
ConfirmRun 写 DB(pending) → 立刻发 task.created → Orchestrator.OnTaskCreated → 调度
```

- **优点**：延迟低，收到就调。  
- **风险**：若 `task.created` 丢了，任务停在 pending 没人理（除非有兜底）。

### 方式 B — 拉模式（Orchestrator 定时 `SchedulePending`）

```text
ConfirmRun 只写 DB(pending)
Orchestrator 每几秒：SELECT * FROM tasks WHERE status=pending → 调度
```

- **优点**：不依赖「创建事件」必达；DB 即待办箱。  
- **缺点**：有轮询间隔（最多延迟一个 tick）。

### 推荐组合（完整系统）

```text
主路径：Publish task.created（快）
兜底：  SchedulePending 扫描（慢但稳）——与 status 对账同一思想
```

对应端口里两个方法可以并存：

```go
OnTaskCreated(ctx, TaskCreated) error     // 被队列叫醒
SchedulePending(ctx, limit int) error     // 定时扫 DB，幂等：已 queued 的跳过
```

**同进程**：Memory 队列几乎不丢，扫库仍建议保留作统一模型。  
**跨进程**：创建事件 + pending 扫描双保险，和 status + 执行面对账是同一类「最终一致」设计。
