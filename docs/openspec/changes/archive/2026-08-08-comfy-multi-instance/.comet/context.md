# Comet Design Handoff

- Change: comfy-multi-instance
- Phase: design
- Mode: compact
- Context hash: 29df51d575c2be2dfdc6e6dad8cd0cf8315e2bdb467785dd9b31e921330dac50

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## docs/openspec/changes/comfy-multi-instance/proposal.md

- Source: docs/openspec/changes/comfy-multi-instance/proposal.md
- Lines: 1-33
- SHA256: 8667813fa03c929dcfd2ca24140e144d0a551ff84953a4c79eccb4dd4555627c

```md
## Why

当前 bot 只连单一 Comfy、Task/Session 仅内存、不存 TG 用户。需要：同进程多 Comfy（落库 CRUD + 健康选路）、分接口观测 system/queue、按实例查本系统 Task，以及 User/Session/Task 持久化与关联，支撑重启可恢复与运维查询。

## What Changes

- `comfy_instances` 落库 + CRUD；健康探测；客户端池；round-robin
- `GET .../system`、`GET .../queue`、`GET .../tasks`
- **User** 表（内部 UUID + 唯一 tg_user_id，资料 upsert）
- **Session** 落库（user_id + chat_id；长期保留）
- **Task** 落库（只挂 session_id；经 Session join 用户/聊天）
- Makefile + README；保留 `comfy_mock`
- 不拆 catalog_cases bindings；不清队列/interrupt；无管理后台

## Capabilities

### New Capabilities

- `comfy-instance-pool`：实例 CRUD、健康、客户端池、system/queue/tasks 观测
- `user-directory`：TG 用户持久化与 upsert
- `session-persistence`：Session 落库与 User/Chat 关联
- `task-persistence`：Task 落库与 session_id 关联

### Modified Capabilities

- `task-orchestrator`：round-robin 选路
- `comfyui-executor`：按 InstanceID 选用客户端

## Impact

- 代码：仓储、TG upsert、ConfirmRun 关联、API、选路、main 接线
- 文档：Makefile、README
- 非目标：拆 doc_json bindings、强鉴权、多机 Worker、Comfy history 当业务史

```

## docs/openspec/changes/comfy-multi-instance/design.md

- Source: docs/openspec/changes/comfy-multi-instance/design.md
- Lines: 1-43
- SHA256: f7d3a6f8309093379cc6f659c4ef56a611cff1c846fd435c54cb2b0c615dbe10

```md
## Context

单实例静态注册 + 内存 Task/Session 不足以支撑多机 Comfy 与运维观测；亦无法重启恢复填表/工单，且无 TG 用户档案。

## Goals / Non-Goals

**Goals**

- 实例落库 CRUD、健康 + round-robin、客户端池
- system / queue 分接口；按实例 DB tasks
- User / Session / Task 落库；Task→Session→User
- Makefile / README

**Non-Goals**

- 拆 catalog_cases bindings
- Comfy `/history` 当业务史；清队列/interrupt
- 管理后台、强 RBAC、多机 Worker

## Decisions

1. 同进程多 HTTP 客户端  
2. DB 为实例真相源；YAML/单 URL 种子 upsert  
3. User：内部 UUID + 唯一 `tg_user_id`；From 全量可得字段 + upsert  
4. Session：`user_id` + `chat_id`；长期保留  
5. Task：只挂 `session_id`；不冗余 user/chat  
6. 观测：`/system`、`/queue`、`/tasks` 三分  
7. 健康用 system_stats；选路 round-robin  

## Risks / Trade-offs

- [范围大] → 设计已确认同 change 交付  
- [通知需 join] → ConfirmRun/通知路径统一经 Session  
- [Session 删除] → 禁止断链物理删  

## Migration Plan

- AutoMigrate 新表；Memory 仓储换 GORM  
- 空库实例种子；旧内存数据不迁移  

## Open Questions

（无阻塞）

```

## docs/openspec/changes/comfy-multi-instance/tasks.md

- Source: docs/openspec/changes/comfy-multi-instance/tasks.md
- Lines: 1-30
- SHA256: 2b2f070101930807607459be102a6906dbc849bdb197d031531973b9ab09bdb7

```md
## 1. User / Session / Task 持久化

- [ ] 1.1 新增 `users` 表与仓储；TG 路径按 tg_user_id upsert（含 From 全量可得字段与 last_seen_at）
- [ ] 1.2 新增 `sessions` 表与 GORM 仓储；替换 Memory；含 user_id + chat_id；活跃查询；长期保留
- [ ] 1.3 新增 `tasks` 表与 GORM 仓储；替换 Memory；必填 session_id；支持 ListByInstance / 经 Session 列「我的任务」
- [ ] 1.4 ConfirmRun 与通知路径：创建 Task 写 session_id；回图 join Session.chat_id；补测试

## 2. 实例池与观测

- [ ] 2.1 `comfy_instances` 表 + Repository CRUD；启动种子 upsert；刷新客户端池
- [ ] 2.2 HTTP `/api/v1/comfy-instances` CRUD
- [ ] 2.3 Comfy Client：`SystemStats` / `Queue`（Mock 占位）
- [ ] 2.4 `GET .../{id}/system`、`.../queue`、`.../tasks`；不可达明确错误；补测试

## 3. 健康、选路与执行

- [ ] 3.1 enabled 实例周期健康探测；ListHealthy 过滤
- [ ] 3.2 Orchestrator round-robin；无可用实例不投递
- [ ] 3.3 Actuator 按 InstanceID 选用客户端；`main` 接线 AutoMigrate

## 4. Make 与文档

- [ ] 4.1 Makefile：build / test / run / run-mock
- [ ] 4.2 README：架构、种子、CRUD、system/queue/tasks curl、Mock、无鉴权警示
- [ ] 4.3 `configs/bot.example.yaml` 注释更新

## 5. 验收

- [ ] 5.1 相关 `go test` 通过
- [ ] 5.2 按 README 可管理实例、查 system/queue、查实例任务；重启后 Session/Task/User 仍在

```

## docs/openspec/changes/comfy-multi-instance/specs/comfy-instance-pool/spec.md

- Source: docs/openspec/changes/comfy-multi-instance/specs/comfy-instance-pool/spec.md
- Lines: 1-89
- SHA256: 43762f128cc84a2d70011c97bafdac5986a0f5b7678289d2bc23fa57b87bf745

[TRUNCATED]

```md
## Purpose

在同一 bot 进程内将多台远程 ComfyUI 的实例元数据持久化到数据库，提供增删改查，并结合健康探测与进程内 HTTP 客户端池供调度选路；观测面分接口查询 Comfy system/queue，以及按实例关联的本系统 Task。

## ADDED Requirements

### Requirement: 实例元数据持久化
系统 MUST 将 Comfy 实例记录持久化到数据库。每条记录 MUST 至少包含：稳定 `id`、`base_url`、是否启用（enabled）。可选字段可包含能力标签（capabilities）。删除或禁用后 MUST 不再作为默认可调度目标（进行中的已派发任务按既有失败/对账路径处理）。

#### Scenario: 重启后实例仍在
- **WHEN** 管理员已创建实例 `gpu-1` 并成功写入数据库后进程重启
- **THEN** 系统仍能按 id 读出该实例及其 base_url

### Requirement: 实例增删改查 API
系统 MUST 在 bot 进程的 HTTP 服务上提供实例的创建、查询（单个与列表）、更新、删除（或等价禁用）接口。写操作成功后 MUST 反映到后续列表查询与调度所用的注册视图（允许短暂传播延迟，但 MUST 在同一进程内可预期地刷新客户端池）。未认证的公网暴露风险本期可用本地绑定/文档警示；完整鉴权可后续增强，但 API 形状 MUST 稳定可测。

#### Scenario: 创建后可列表查出
- **WHEN** 调用创建接口写入 id=`gpu-2`、合法 base_url
- **THEN** 列表接口返回包含该实例的记录

#### Scenario: 更新 base_url
- **WHEN** 对已存在实例更新 base_url
- **THEN** 后续健康探测与对该 InstanceID 的 Comfy 调用使用新地址

#### Scenario: 删除或禁用后不再被调度
- **WHEN** 实例被删除或标记为 disabled
- **THEN** 调度的健康可选集合不再包含该实例

### Requirement: 实例系统概况只读查询
系统 MUST 提供 `GET`（或等价）按实例 id 查询系统概况的接口。数据 MUST 来自对该实例 Comfy 的 `GET /system_stats`（或文档约定的等价端点）。响应 MUST 能表达连通性，并在可达时包含系统/设备资源摘要（如 RAM/VRAM、版本信息等 Comfy 返回的关键字段）。不可达时 MUST 明确失败或 `reachable=false`，MUST NOT 伪造成功空概况。本期 MUST NOT 经此接口执行写操作。

#### Scenario: 可达时返回 system 摘要
- **WHEN** 实例可达
- **THEN** 接口返回可达标志及源自 system_stats 的摘要字段

#### Scenario: 不可达
- **WHEN** 目标实例网络失败或超时
- **THEN** 接口明确表示不可达（或非 2xx 业务错误）

### Requirement: 实例队列只读查询
系统 MUST 提供独立于系统概况的按实例 id 队列只读接口。数据 MUST 来自对该实例 Comfy 的 `GET /queue`（或文档约定的等价只读端点），并 MUST 暴露 running 与 pending 概况。MUST NOT 用 Comfy `/history` 充当本系统任务历史，MUST NOT 把业务 Task 列表混进该响应作为唯一任务来源。不可达时规则与系统概况接口相同。本期 MUST NOT 经此接口清队列或 interrupt。

#### Scenario: 可达时返回队列概况
- **WHEN** 实例可达且队列中有 running 或 pending
- **THEN** 接口返回 running/pending 概况

#### Scenario: 队列接口与 system 分离
- **WHEN** 调用方只请求队列接口
- **THEN** 不强制同时返回完整 system_stats 载荷

### Requirement: 按实例查询本系统任务
系统 MUST 提供按实例 id 列出本系统 Task 的只读接口。查询 MUST 以数据库中 Task 的 `instance_id` 关联实例，MUST NOT 依赖拉取 Comfy `/history` 作为该列表数据源。接口 SHOULD 支持分页与按任务状态过滤，并返回足以运维对照的字段（如 task id、status、prompt_id、时间戳、错误摘要）。尚未派发、因而没有 `instance_id` 的 pending 任务 MUST NOT 被错误归入某实例的该列表。

#### Scenario: 已派发任务可按实例查出
- **WHEN** 任务已调度到实例 `gpu-1`（Task.InstanceID 已写入）
- **THEN** 对该实例的任务列表接口能返回该任务

#### Scenario: 不混入未关联实例的 pending
- **WHEN** 存在尚未选定实例的 pending 任务
- **THEN** 这些任务不出现在任意实例的按实例任务列表中

### Requirement: 配置种子与 Mock 兼容
系统 MUST 允许启动时从配置（如可选 `comfy_instances` 或单一 `comfyui_base_url`）**upsert** 种子实例到数据库。`comfy_mock` 为真时 MUST 仍提供可用的进程内 Mock 客户端路径；system/queue 查询对 Mock MUST 返回稳定可测占位（标明 mock），不得因缺远程而崩溃。

#### Scenario: 空库用单 URL 种子
- **WHEN** 数据库中尚无实例，且配置了 `comfyui_base_url`，`comfy_mock` 为假
- **THEN** 启动后存在默认实例记录，指向该 URL

#### Scenario: Mock 模式查询 system/queue
- **WHEN** `comfy_mock` 为真并查询默认实例的 system 或 queue
- **THEN** 返回标明 mock 的占位，进程不崩溃

### Requirement: 健康检查过滤不可用实例
系统 MUST 对**已启用**的真实 HTTP 实例做周期性健康探测（例如请求 Comfy 的系统状态端点）。探测失败或超时的实例 MUST 不出现在「健康可选」集合中。探测间隔与超时 MUST 可配置并设有安全默认值。

#### Scenario: 一台宕机被剔除
- **WHEN** 启用实例 `a` 探测失败，启用实例 `b` 探测成功
- **THEN** 健康可选集合仅包含 `b`（及仍健康的其他启用实例）

#### Scenario: 恢复后重新可选

```

Full source: docs/openspec/changes/comfy-multi-instance/specs/comfy-instance-pool/spec.md

## docs/openspec/changes/comfy-multi-instance/specs/comfyui-executor/spec.md

- Source: docs/openspec/changes/comfy-multi-instance/specs/comfyui-executor/spec.md
- Lines: 1-20
- SHA256: daf4ad3627fa6841e5e2e414c3d76802e4c167a534e4ec78b62f12f371e35b49

```md
## MODIFIED Requirements

### Requirement: Actuator 作为执行面调用 ComfyUI
系统 MUST 提供 Actuator 模块：消费指向本进程订阅的 dispatch 命令，按 Case 快照中的绑定将输入注入工作流图，并调用 **dispatch 所指定 InstanceID 对应的** ComfyUI 客户端（Mock 或真实 HTTP），等待完成或失败。当配置了多台真实实例时，客户端 MUST 按实例区分，MUST NOT 忽略 InstanceID 而统一打到单一全局 URL。Actuator MUST NOT 使用与 Case 无关的空 stub 图作为成功主路径的默认行为；MUST NOT 直接更新全局 Task 表的终态；MUST 通过 status 事件向 Orchestrator 上报。

#### Scenario: 收到 dispatch 后执行文生图
- **WHEN** Actuator 收到合法 dispatch 且该 InstanceID 的 ComfyUI 客户端可用，且 Case 含有效 workflow 与注入后的图
- **THEN** 系统向该实例对应的 ComfyUI 提交工作流，并上报至少一次 running 与一次终态 status（succeeded 或 failed）

#### Scenario: ComfyUI 不可达时上报失败 status
- **WHEN** 目标实例的 ComfyUI 不可达或返回连接错误（真实 HTTP 模式）
- **THEN** Actuator 上报 failed status（含可诊断信息），且不假装成功

#### Scenario: Mock 开关开启时仍完成主路径
- **WHEN** `comfy_mock`（或等价环境变量）为真
- **THEN** 同一注入与 status 路径可成功完成并产生可投递图片产物

#### Scenario: 按 InstanceID 选用客户端
- **WHEN** 同一进程注册了多台真实 Comfy 客户端，且 dispatch.InstanceID 为其中一台
- **THEN** Upload/Submit/Wait 只对该实例的 base_url 生效

```

## docs/openspec/changes/comfy-multi-instance/specs/session-persistence/spec.md

- Source: docs/openspec/changes/comfy-multi-instance/specs/session-persistence/spec.md
- Lines: 1-26
- SHA256: bcb4384a0f3b4bac13d1f8a0ed60f326df451d3bf2bb1b9485e2880a33a0b2fd

```md
## Purpose

将 Case 填表会话（Session）持久化到数据库，并与 User、聊天上下文关联；作为 Task 创建前的输入采集态，生命周期不包含 Task 执行阶段。

## ADDED Requirements

### Requirement: Session 持久化
系统 MUST 将 Session 持久化到数据库（替换仅内存实现作为主路径）。记录 MUST 至少包含：session id、关联的 `user_id`、`chat_id`、`case_id`、状态、当前输入进度、草稿输入、时间戳。进程重启后 MUST 仍能按既有查询方式恢复活跃会话（若状态仍为活跃）。

#### Scenario: 重启后活跃 Session 可恢复
- **WHEN** 用户处于 collecting 的 Session 已写入数据库后进程重启
- **THEN** 按该 chat（及实现约定的活跃查询）仍能读出该 Session 及其草稿进度

### Requirement: Session 同时关联 User 与 Chat
每条 Session MUST 同时持久化 `user_id` 与 `chat_id`。系统 MUST NOT 仅用 chat_id 推断用户身份而不写入 `user_id`。

#### Scenario: 创建 Session 时写入 user_id
- **WHEN** 已 upsert 的用户在某 chat 开始 Case
- **THEN** 新建 Session 的 user_id 指向该用户，chat_id 为该聊天

### Requirement: Session 与 Task 生命周期分离且可被 Task 引用
Session 状态机 MUST 表示填表过程（如 collecting / confirming / submitted / exited），MUST NOT 用 Session 状态表示 Comfy 执行中的 running。当 ConfirmRun 成功创建 Task 后，Session MUST 进入已提交（或等价终态），且该 Session 行 MUST 保持可查询以便 Task 通过 `session_id` 关联。系统 MUST NOT 在 Task 仍可能引用时物理删除对应 Session。

#### Scenario: 确认后 Session 结束填表且行仍在
- **WHEN** ConfirmRun 成功
- **THEN** Session 不再处于活跃填表态，且数据库中仍能按 session id 读出该行供 Task 关联

```

## docs/openspec/changes/comfy-multi-instance/specs/task-orchestrator/spec.md

- Source: docs/openspec/changes/comfy-multi-instance/specs/task-orchestrator/spec.md
- Lines: 1-16
- SHA256: 90ab306356861ac2ae0dd9820be98420cacd4a1f4932b74aaaaf7f14dd3cce6f

```md
## MODIFIED Requirements

### Requirement: 调度与 pending 扫描双触发
系统 MUST 支持事件触发调度，并 MUST 提供定时扫描 `pending` 的兜底，以免创建事件丢失导致任务无人认领。调度选择实例时 MUST 仅从健康且未被熔断禁止的实例中选择，并 MUST 在可选集合上使用轮询（round-robin）分配，避免总是固定选择同一台；选定后 MUST 投递 `dispatch.<instance_id>` 并将 Task 置为 `queued`（幂等）。当没有任何可选实例时 MUST 保持 Task 为 `pending`（或等价未投递状态）并记录可诊断原因，MUST NOT 假装已排队到某实例。

#### Scenario: 创建事件丢失仍被扫到
- **WHEN** Task 长期处于 pending 且无成功调度
- **THEN** SchedulePending 仍能将其投递（若实例可用）

#### Scenario: 多健康实例轮询
- **WHEN** 存在至少两台健康且未熔断的实例，连续调度多个 pending Task
- **THEN** 连续选定的 InstanceID 在轮询意义上分散到这些实例，而非每次都是同一台

#### Scenario: 全部不可用时不投递
- **WHEN** 没有任何健康且未熔断的实例
- **THEN** Task 仍保持 pending（或未被标记为 queued），且不向任意实例投递 dispatch

```

## docs/openspec/changes/comfy-multi-instance/specs/task-persistence/spec.md

- Source: docs/openspec/changes/comfy-multi-instance/specs/task-persistence/spec.md
- Lines: 1-30
- SHA256: c9316f7f4c29fcf7d942b5b1befe3c654c0b29ac5794794f618a7b01cb2e8401

```md
## Purpose

将生成工单（Task）持久化到数据库，并通过 `session_id` 关联填表会话，从而间接关联 User 与 chat；支持按实例查询已派发任务。

## ADDED Requirements

### Requirement: Task 持久化
系统 MUST 将 Task 持久化到数据库（替换仅内存实现作为主路径）。记录 MUST 至少包含：task id、`session_id`、`case_id`、状态、可选 `instance_id` / `prompt_id`、输入前缀、产物与错误信息、时间戳。进程重启后 MUST 仍能按 id 读出未完成与历史任务。

#### Scenario: 重启后 Task 仍在
- **WHEN** 已创建的 Task 写入数据库后进程重启
- **THEN** 仍可按 task id 读出其状态与 session_id

### Requirement: Task 通过 Session 关联用户与聊天
创建 Task 时 MUST 写入有效的 `session_id`。Task 行 MUST NOT 将 `user_id` / `chat_id` 作为必需冗余字段；需要通知聊天或用户归属时，系统 MUST 通过 Session（及 User）关联获得。`ListMyTasks` 或等价「我的任务」查询 MUST 在持久化模型上仍按用户或聊天可列出（允许 join）。

#### Scenario: ConfirmRun 写入 session_id
- **WHEN** ConfirmRun 成功创建 Task
- **THEN** 该 Task 的 session_id 指向当次确认所用 Session

#### Scenario: 经 Session 解析通知 chat
- **WHEN** 任务成功需要向 TG 发通知
- **THEN** 系统能通过 Task.session_id → Session.chat_id 得到投递目标（或不依赖已删除的 Session）

### Requirement: 派发后可按 instance 查询
当 Task 被调度并写入 `instance_id` 后，系统 MUST 支持按 `instance_id` 列出这些任务，供实例观测 API 使用。

#### Scenario: 按实例列出已派发任务
- **WHEN** Task 的 instance_id 为 `gpu-1`
- **THEN** 按实例 `gpu-1` 的任务查询包含该 Task

```

## docs/openspec/changes/comfy-multi-instance/specs/user-directory/spec.md

- Source: docs/openspec/changes/comfy-multi-instance/specs/user-directory/spec.md
- Lines: 1-23
- SHA256: 462622d10a54c3e012d4a923d7ce5c855da73aa144dbcee4cff630caa9e543b1

```md
## Purpose

将 Telegram 用户身份持久化到数据库，作为 Session/Task 归属的稳定主体；支持多渠道扩展的内部用户主键。

## ADDED Requirements

### Requirement: User 持久化与主键
系统 MUST 将用户记录持久化到数据库。每条记录 MUST 使用**内部 UUID**（或等价内部 id）作为主键，并 MUST 对 Telegram 用户 id（`tg_user_id`）建立唯一约束。系统 MUST NOT 仅用 `chat_id` 充当唯一用户主键。

#### Scenario: 同一 tg_user_id 不重复建档
- **WHEN** 同一 Telegram 用户再次触发 upsert
- **THEN** 仍对应同一条内部用户记录（id 不变）

### Requirement: TG 资料字段与 upsert
系统 MUST 在处理 Telegram 消息或回调查询的路径上，根据 `From`（或等价来源）upsert 用户资料。存储字段 MUST 覆盖 From 可获得的常用身份信息，至少包括：`username`、`first_name`、`last_name`、`language_code`、`is_bot`、`is_premium`（若 API 提供；不可得则可空）。系统 MUST 维护 `last_seen_at`（或等价），并在 upsert 时刷新可变资料字段。

#### Scenario: 消息到达后可查到用户资料
- **WHEN** 用户首次向 bot 发消息且 From 含 username 与 first_name
- **THEN** 数据库中存在对应该 tg_user_id 的用户行，且上述字段已写入

#### Scenario: 资料变更被刷新
- **WHEN** 同一用户稍后消息中 username 已变更
- **THEN** upsert 后该用户行的 username 为新值，且 last_seen_at 更新

```
