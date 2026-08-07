# Comet Design Handoff

- Change: workflow-engine-core
- Phase: design
- Mode: compact
- Context hash: c2fb6cca8b1ff63779f0c1c088be4d06a327002e8a75b9ee3749281cfed2c95d

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## docs/openspec/changes/workflow-engine-core/proposal.md

- Source: docs/openspec/changes/workflow-engine-core/proposal.md
- Lines: 1-32
- SHA256: 48f0e610dbe5558168842b6106779005eb20bab9ecb464492287e85e0bf684f1

```md
## Why

项目要做成可部署的 ComfyUI Telegram Bot 服务。本期落地可组装的工作流核心：Case 协议与注册、私聊 Dialog Session、异步 Task，以及 Bot（渠道）/ Orchestrator（控制面）/ Actuator（执行面）模块；Queue/Blob 端口化，便于同进程组装或后续拆分。

## What Changes

- 新增 **Workflow Case 协议**：元数据、有序 input/output、**JSON Schema 校验**（含媒体扩展）、Comfy 绑定、价格数值、可扩展 tags。
- 新增 **工作流注册与仓储**：GORM + SQLite，预留 MySQL。
- 新增 **Dialog Session**：按 `chat_id` 填表锁；浏览不上锁；未 Exit/提交不得开新 Case；生成中 Task 不挡新 Case。
- 新增 **异步 Task + Orchestrator + Actuator**：ConfirmRun 后事件驱动；Task 终态仅 Orchestrator 写库；Actuator 发 status 并对账 Query；风暴防护与温和取消。
- 新增 **TG Adapter**：菜单/分类/Case/填表/拦截/结果通知（Application DTO → TG 消息）。
- **本期不做**：真实积分扣费、多租户部署平台、强取消（interrupt running）、生产级多云 MQ/OSS（保留端口）。

## Capabilities

### New Capabilities

- `workflow-protocol`: Case 协议与 JSON Schema 校验契约。
- `workflow-registry`: Case 注册与持久化。
- `dialog-session`: 私聊填表会话与锁。
- `task-orchestrator`: Task 生命周期、调度、对账、notify 驱动、风暴防护。
- `comfyui-executor`: Actuator 执行面（注入、ComfyUI、status、本地 ledger、Query）。
- `channel-tg`: Telegram 渠道适配与通知投递。

### Modified Capabilities

- （无既有主规格）

## Impact

- Go 模块化工程；GORM/SQLite；`go-telegram/bot`；JSON Schema 库；Memory Queue + LocalFS。
- 外部：Telegram Bot API、ComfyUI HTTP API。

```

## docs/openspec/changes/workflow-engine-core/design.md

- Source: docs/openspec/changes/workflow-engine-core/design.md
- Lines: 1-94
- SHA256: 30a2adcfbfcc12cb67b9ecb258a13748416a30489c7e926edba7fd426c295f2a

[TRUNCATED]

```md
## Context

空仓库起步，技术栈为 Golang。产品最终形态是可部署的 TG Bot 服务，终端用户通过对话调用 ComfyUI（文生图、图像/视频编辑等）。本期只交付工作流核心：协议、基于 ORM 的注册仓储、单实例 ComfyUI 执行器。TG 对话、积分扣费、ComfyUI 调度均明确延后，但协议需预留对话与展示所需元数据（preview、描述、价格数值、可跳过输入等）。

## Goals / Non-Goals

**Goals:**

- 定义可扩展的 Case/Workflow 协议（input/output schema、校验、元数据、类别标签、ComfyUI 绑定）
- 用 ORM + SQLite 注册/查询/更新/停用 case，模型可迁移到 MySQL
- 单实例 ComfyUI：注入输入、执行、按 output schema 回收 image/text/file
- 覆盖已知场景标签示例，并以 schema 驱动扩展，而不是写死 case 类型实现

**Non-Goals:**

- TG 对话状态机与 Bot 部署平台
- 真实积分/支付
- ComfyUI 多实例调度
- 管理后台 UI

## Decisions

### D1. 模块划分

采用清晰分层：

```
protocol/     # 协议模型、schema、校验
registry/     # ORM 实体、仓储、迁移
comfyui/      # HTTP 客户端、注入、轮询、产物回收
app/api 或 cmd/ # 最小调用入口（HTTP 或 CLI，实现阶段二选一偏 HTTP 便于联调）
```

协议与执行解耦：registry 存协议文档 + 绑定；executor 只消费「已校验输入 + case 快照」。

### D2. ORM 选型：GORM

- **选择**：GORM + SQLite 驱动，配置层可切换 `mysql` DSN。
- **理由**：生态成熟、迁移工具够用、后续换 MySQL 成本低。
- **备选**：ent（更强类型，样板更多）、sqlx（偏手写 SQL）。本期优先交付速度与可切换性，选 GORM。

### D3. 协议载体

- **选择**：领域用 Go struct；持久化可将完整 case 文档存 JSON 列，并用关键列索引 `id`、状态、类别标签。
- **理由**：schema 会演进，整文档 JSON 降低改表频率；标签可用关联表或 JSON 查询策略（SQLite 下关联表更稳）。
- **备选**：每个 schema 字段拆列——过早结构化，迁移重。

### D4. 类别标签 vs 固定枚举

- **选择**：`tags/categories` 为字符串集合；预置文档示例含 text2img、text2video、img2img、img2video、videoedit、imgedit，并可扩展 upscale、inpaint、remove-bg 等。
- **理由**：真实差异在 input/output schema 与工作流图，不在类型名。

### D5. Input / Output 类型基线

- Input：`string` | `image` | `video` | `number` | `boolean` | `enum`
- Output：`image` | `text` | `file`（视频走 file，可附 `media_type`）
- 每个 input：`key`、`type`、`required`、`skip_allowed`（仅当非必填）、`description`、`preview`、校验约束

### D6. ComfyUI 绑定模型

- Case 保存：工作流模板（API prompt graph）+ `input_bindings`（logical key → node_id + field_path）+ `output_bindings`（logical key → 产物定位方式，如 node/output index 或 filename 前缀规则）。
- 执行时：深拷贝模板 → 注入 → `/prompt` 提交 → 轮询 history/等待 → 拉产物。

### D7. 价格字段

- 仅 `price`（数值，单位由上层约定，本期不解释货币/积分换算）。
- 不调用任何扣费接口。

### D8. 本期入口形态

- 提供最小 HTTP API 或内部 Service 接口用于注册、查询、校验、执行；不实现 TG webhook。
- 具体路由在实现任务中细化，设计层要求「可被后续 Bot 层直接调用的应用服务接口」。

## Risks / Trade-offs

- **[Risk] ComfyUI 工作流图与节点字段因版本/自定义节点而异** → 绑定外置到 case 定义；提供清晰映射错误；用 1～2 个样例工作流做集成测试。
- **[Risk] 大文件（视频）内存压力** → 产物落本地临时目录/对象路径，结果返回引用；避免无缓冲超大文件。
- **[Risk] SQLite 与 MySQL 方言差异** → 只用 GORM 可移植特性；避免原始 SQLite JSON 特有 SQL。
- **[Risk] 协议过早绑死 TG 交互** → 协议只表达数据与校验语义；对话编排留给后续模块。
- **[Trade-off] JSON 整文档存储 vs 强列结构** → 灵活优先，列表过滤靠索引列/标签表补偿。

```

Full source: docs/openspec/changes/workflow-engine-core/design.md

## docs/openspec/changes/workflow-engine-core/tasks.md

- Source: docs/openspec/changes/workflow-engine-core/tasks.md
- Lines: 1-53
- SHA256: d1cf305a7bcbf1eea6924b7d0750939955326273b6dd915c7742de485d6eeb11

```md
## 1. 工程与端口

- [ ] 1.1 初始化 Go module、`cmd` wire、config/slog、chi health
- [ ] 1.2 实现 `port/queue` + Memory 适配器
- [ ] 1.3 实现 `port/blob` + LocalFS 适配器
- [ ] 1.4 实现 `port/notify`、`port/instance`（一期单实例注册表）
- [ ] 1.5 GORM + SQLite 连接与 AutoMigrate 骨架

## 2. 协议与注册

- [ ] 2.1 Case 文档模型与 JSON Schema 校验 + 媒体钩子
- [ ] 2.2 Case Registry 仓储（CRUD、按 tag/菜单过滤、停用）
- [ ] 2.3 协议与仓储单元/集成测试

## 3. Session 与 Task 领域

- [ ] 3.1 Dialog Session 状态机与按 chat_id 锁
- [ ] 3.2 Task 聚合与合法迁移（含温和取消）
- [ ] 3.3 领域单测（锁、非法边、幂等前提）

## 4. App 与 ConfirmRun

- [ ] 4.1 app.Facade：菜单/Case/Session/ConfirmRun/ListTasks
- [ ] 4.2 ConfirmRun：校验、物化 Blob、建 pending、发 `task.created`、解锁 Session
- [ ] 4.3 DeliverNotify 用例入口（供 TG 适配）

## 5. Orchestrator

- [ ] 5.1 OnTaskCreated + SchedulePending 双触发调度
- [ ] 5.2 applyStatus 幂等写库 + 发 notify.user
- [ ] 5.3 超时对账 + ExecutionQuery
- [ ] 5.4 风暴防护：限流、退避+jitter、熔断、分池、redispatch 配额
- [ ] 5.5 温和取消（pending/queued）
- [ ] 5.6 Orchestrator 单测（假 Queue/Query）

## 6. Actuator 与 ComfyUI

- [ ] 6.1 ComfyUI HTTP client（submit/wait/fetch）
- [ ] 6.2 HandleDispatch：注入、执行、ledger、发 status
- [ ] 6.3 ExecutionQuery 与 status 重发
- [ ] 6.4 假 ComfyUI 单测 + 可选真机联调

## 7. TG Adapter

- [ ] 7.1 go-telegram/bot 接入与 Update 路由
- [ ] 7.2 菜单/分类/Case/会话/锁拦截渲染
- [ ] 7.3 消费 notify.user 发结果（去重）
- [ ] 7.4 样例 Case 种子与 README 跑通说明

## 8. 验收

- [ ] 8.1 Memory all-in-one 冒烟：text2img 路径（可 mock Comfy）
- [ ] 8.2 对照 specs 清单勾验；确认无强取消/扣费/Actuator 直写终态

```

## docs/openspec/changes/workflow-engine-core/specs/channel-tg/spec.md

- Source: docs/openspec/changes/workflow-engine-core/specs/channel-tg/spec.md
- Lines: 1-26
- SHA256: b123ccfb4c1c02a8867b515d0a79c5200ab910a93af20691a62b049401e1bf55

```md
## ADDED Requirements

### Requirement: TG 适配器将应用 DTO 渲染为 Bot API 消息
系统 MUST 提供 Telegram 适配器：接收 Bot Update，调用渠道无关的应用用例，并将菜单/Case 列表/会话提示/错误/结果渲染为 Telegram 支持的消息形态（文本、Photo、InlineKeyboard 等）。应用层 MUST NOT 依赖 Telegram SDK 类型。

#### Scenario: Case 列表以按钮呈现
- **WHEN** 用户进入某分类的 Case 列表
- **THEN** 适配器发送包含 Case 入口的 InlineKeyboard（可分页）

#### Scenario: 完成后发送图片结果
- **WHEN** 适配器收到 succeeded 的用户通知且输出含 image BlobRef
- **THEN** 适配器向对应用户发送图片（或等价媒体消息）

### Requirement: 适配器执行 Session 锁拦截文案
当应用层返回会话锁定时，适配器 MUST 向用户展示当前流程提示，并提供继续与退出当前流程的操作入口。

#### Scenario: 锁定时展示退出选项
- **WHEN** 用户在填表中尝试开新 Case
- **THEN** 用户收到锁定说明及退出/继续类可点击操作

### Requirement: 消费 notify 完成对用户投递
适配器 MUST 订阅或接收 Orchestrator 的 notify 意图，并完成对 `chat_id` 的消息投递。终态通知 MUST 幂等处理，避免重复刷屏。

#### Scenario: 重复 notify 不重复刷终态消息策略
- **WHEN** 同一 task 终态 notify 重复到达
- **THEN** 适配器不产生重复的骚扰性终态推送（按去重策略合并或忽略）

```

## docs/openspec/changes/workflow-engine-core/specs/comfyui-executor/spec.md

- Source: docs/openspec/changes/workflow-engine-core/specs/comfyui-executor/spec.md
- Lines: 1-48
- SHA256: 570ac730da24c0af894a83e79496c52f99aaa5c584ecc46b0a9576830e7d4657

```md
## ADDED Requirements

### Requirement: Actuator 作为执行面调用 ComfyUI
系统 MUST 提供 Actuator 模块：消费指向本实例的 dispatch 命令，按 Case 快照中的绑定将输入注入工作流图，调用配置的 ComfyUI HTTP API，并等待完成或失败。Actuator MUST NOT 直接更新全局 Task 表的终态；MUST 通过 status 事件向 Orchestrator 上报。

#### Scenario: 收到 dispatch 后执行文生图
- **WHEN** Actuator 收到合法 dispatch 且 ComfyUI 可用
- **THEN** 系统向 ComfyUI 提交工作流，并上报至少一次 running 与一次终态 status（succeeded 或 failed）

#### Scenario: ComfyUI 不可达时上报失败 status
- **WHEN** ComfyUI 实例不可达或返回连接错误
- **THEN** Actuator 上报 failed status（含可诊断信息），且不假装成功

### Requirement: 按绑定映射注入输入
系统 MUST 依据 case 中声明的节点/字段注入映射，将逻辑 input 键映射到 ComfyUI 工作流图中的具体节点入参。缺少映射或映射目标不存在时，系统 MUST 在调用 ComfyUI 前失败并上报 failed status。

#### Scenario: 文本注入到指定节点
- **WHEN** case 将 `prompt` 映射到某节点的文本字段
- **THEN** 提交给 ComfyUI 的工作流中该节点字段等于已物化输入中的 prompt 值

#### Scenario: 映射缺失导致拒绝执行
- **WHEN** 某必填 input 没有有效的 ComfyUI 注入映射
- **THEN** 不调用 ComfyUI，并上报指出映射问题的 failed status

### Requirement: 按 output schema 回收产物到对象存储
执行完成后，系统 MUST 根据 output schema 与输出绑定，从 ComfyUI 历史/产物中提取结果，写入 Blob，并在 succeeded status 中携带 BlobRef。`image`/`text`/`file`（含视频）MUST 可被后续 notify 与查询使用。

#### Scenario: 回收单张输出图片引用
- **WHEN** 工作流成功且 output schema 定义了一个 image 字段
- **THEN** succeeded status 中包含该字段对应的可读取 BlobRef

#### Scenario: 回收视频文件引用
- **WHEN** 工作流成功且 output schema 定义了一个 file 字段指向视频产物
- **THEN** succeeded status 中包含对应 BlobRef

### Requirement: 本地执行账与对账查询
Actuator MUST 维护本地执行账（ledger），在 Publish status 前记录进度；MUST 提供 ExecutionQuery，供 Orchestrator 在 status 丢失时查询执行真相。Actuator MAY 在 Publish 失败后重发 status，消费侧 MUST 幂等。

#### Scenario: 对账查询返回已成功执行
- **WHEN** status 消息丢失但本地 ledger 显示已成功且产物已在 Blob
- **THEN** ExecutionQuery 返回 succeeded 及输出引用，供 Orchestrator 补写 Task

### Requirement: 执行前输入须已通过协议校验
系统 MUST 保证进入 dispatch 的 Task 在 ConfirmRun 时已通过 workflow-protocol 校验。Actuator MUST NOT 接受未物化必填输入的任务作为成功路径。

#### Scenario: 缺少物化输入则失败
- **WHEN** dispatch 指向的 input Blob 前缀缺失必填对象
- **THEN** Actuator 上报 failed，且不向 ComfyUI 假装成功提交

```

## docs/openspec/changes/workflow-engine-core/specs/dialog-session/spec.md

- Source: docs/openspec/changes/workflow-engine-core/specs/dialog-session/spec.md
- Lines: 1-34
- SHA256: 24b63f62bebeaf65ec9d1fbfb0c54c26915dbc06f779a7c4ad3f978d86d1a328

```md
## ADDED Requirements

### Requirement: 每聊天最多一个进行中的填表会话
系统 MUST 以 Telegram `chat_id` 为键，保证同一时刻最多一个非终态 Dialog Session。浏览菜单与 Case 列表 MUST NOT 创建 Session。仅 `StartCase` 成功后进入 `collecting` 并上锁。

#### Scenario: 浏览菜单不创建会话
- **WHEN** 用户仅打开菜单或 Case 列表
- **THEN** 系统不创建 Dialog Session，用户仍可自由导航

#### Scenario: StartCase 上锁
- **WHEN** 用户对某 active Case 执行 StartCase 且当前无进行中 Session
- **THEN** 系统创建 `collecting` 会话并禁止再次 StartCase，直至退出或提交

### Requirement: 上锁期间拦截新 Case 并提供退出
当 Session 处于 `collecting` 或 `confirming` 时，系统 MUST 拒绝新的 StartCase，并返回当前 Case 摘要及「继续 / 退出并重选」类选项语义。用户执行 ExitSession 后，会话 MUST 进入 `exited`（或等价清除），允许新开 Case。

#### Scenario: 上锁时开新 Case 被拒绝
- **WHEN** 用户已在 collecting 中又请求 StartCase
- **THEN** 系统返回锁定错误/视图，不创建第二个 Session

#### Scenario: 退出后可重新开始
- **WHEN** 用户 ExitSession 后再 StartCase
- **THEN** 系统允许创建新的 collecting 会话

### Requirement: 确认执行后结束填表会话
ConfirmRun 成功创建 Task 后，系统 MUST 结束填表 Session（`submitted` 或清除 active），从而解锁。进行中的执行 Task MUST NOT 阻止用户开启新的填表 Session。

#### Scenario: 提交后解锁
- **WHEN** ConfirmRun 成功
- **THEN** 该 chat 不再处于填表锁，可 StartCase

#### Scenario: 生成中可开新 Case
- **WHEN** 用户已有 running Task 且无填表 Session
- **THEN** StartCase 被允许

```

## docs/openspec/changes/workflow-engine-core/specs/task-orchestrator/spec.md

- Source: docs/openspec/changes/workflow-engine-core/specs/task-orchestrator/spec.md
- Lines: 1-54
- SHA256: cded9083d720f78b24199885ce7617c15149257f3581478865f36a12928ad532

```md
## ADDED Requirements

### Requirement: ConfirmRun 创建异步 Task 并物化输入
系统 MUST 在校验通过后创建 Task（`pending`），将输入物化到对象存储，发布可编排信号（如 `task.created`），并立即向调用方返回 `task_id`，MUST NOT 在 ConfirmRun 调用栈内阻塞等待 ComfyUI 完成。

#### Scenario: 确认后立即返回任务号
- **WHEN** 用户 ConfirmRun 且校验通过
- **THEN** 系统持久化 pending Task，返回 task_id，并异步进入编排

### Requirement: Orchestrator 独占 Task 终态写入
系统 MUST 仅由 Orchestrator 将 Task 迁移至 `succeeded`/`failed`/`cancelled`（以及中间态 `queued`/`running` 的控制面确认）。Actuator 上报的 status 与对账结果 MUST 经统一幂等入口 `applyStatus` 写库。

#### Scenario: status 驱动成功
- **WHEN** Orchestrator 收到合法 succeeded status
- **THEN** Task 变为 succeeded，并触发用户通知意图

#### Scenario: 重复 status 不重复副作用
- **WHEN** 同一终态 status 被重复投递
- **THEN** Task 保持已收敛状态，且不重复发送终态通知

### Requirement: 调度与 pending 扫描双触发
系统 MUST 支持事件触发调度，并 MUST 提供定时扫描 `pending` 的兜底，以免创建事件丢失导致任务无人认领。调度选择实例后 MUST 投递 `dispatch.<instance_id>` 并将 Task 置为 `queued`（幂等）。

#### Scenario: 创建事件丢失仍被扫到
- **WHEN** Task 长期处于 pending 且无成功调度
- **THEN** SchedulePending 仍能将其投递（若实例可用）

### Requirement: Status 丢失时对账兜底
对处于 `queued`/`running` 且超时的 Task，Orchestrator MUST 通过 ExecutionQuery（或等价真相源）探测执行结果，并经 `applyStatus` 补写；多次失败或超过最大存活时间 MUST 收口为 failed 并通知。

#### Scenario: 对账补写成功
- **WHEN** status 丢失但 Query 显示已成功
- **THEN** Task 被标记 succeeded 并通知用户

### Requirement: 温和取消
系统 MUST 允许取消仍处于 `pending` 或 `queued` 的 Task。MUST NOT 在本期对 `running` 任务执行 ComfyUI interrupt。

#### Scenario: 取消排队中任务
- **WHEN** 用户取消 queued Task
- **THEN** Task 变为 cancelled，且不再被 Actuator 成功执行（或执行结果被控制面忽略为取消策略所定义）

### Requirement: 调用风暴防护
Orchestrator MUST 对调度、对账、重派、实例探测与通知实施限流、退避（含抖动）、错误分类、实例熔断，以及对账与调度分池，避免大面积故障时放大重试。

#### Scenario: 实例连续失败触发熔断
- **WHEN** 某实例连续调度/探测失败超过阈值
- **THEN** 系统在熔断期内停止向该实例投递新任务，并记录可观测状态

### Requirement: 通过 notify 驱动渠道回用户
Task 到达终态（及可选进度）时，Orchestrator MUST 发布与渠道无关的 notify 意图；MUST NOT 在 Orchestrator 内直接调用 Telegram API。

#### Scenario: 成功后发出通知意图
- **WHEN** Task 变为 succeeded
- **THEN** 系统发布含 chat_id、task_id、输出引用的 notify，供渠道模块投递

```

## docs/openspec/changes/workflow-engine-core/specs/workflow-protocol/spec.md

- Source: docs/openspec/changes/workflow-engine-core/specs/workflow-protocol/spec.md
- Lines: 1-71
- SHA256: 85951e731724a3fbce63eda064c42121e9c8787ca627eab83624ce910a8473b0

```md
## ADDED Requirements

### Requirement: Case 协议定义可执行的工作流契约
系统 MUST 提供 Case/Workflow 协议，用于描述一个可注册、可校验、可执行的工作流 case。每个 case MUST 至少包含：稳定标识、显示名称、描述、可选 preview 资源引用、价格数值字段、一个或多个类别标签、有序的 input schema 列表、有序的 output schema 列表，以及到 ComfyUI 工作流定义的绑定信息（含节点注入映射）。

#### Scenario: 注册前协议字段齐全
- **WHEN** 调用方提交一个完整的 text2img case 定义
- **THEN** 系统接受该定义，并保留其元数据、input/output schema 与 ComfyUI 绑定信息以供后续查询与执行

#### Scenario: 缺少必填协议字段被拒绝
- **WHEN** 调用方提交缺少标识、input schema 或 ComfyUI 绑定的 case 定义
- **THEN** 系统拒绝该定义并返回可定位缺失字段的错误

### Requirement: Input schema 支持多类型采集与可选跳过
系统 MUST 将 input 定义为有序数组。每个 input 项 MUST 包含：字段键、类型、是否必填、描述、可选 preview，以及在非必填时可被跳过的标记。本阶段类型 MUST 至少支持：`string`（文本）、`image`、`video`、`number`、`boolean`、`enum`。后续 TG 对话可按该数组顺序逐项引导；本阶段仅保证协议与校验语义成立。

#### Scenario: 必填文本输入不可跳过
- **WHEN** text2img case 的 prompt 输入标记为必填
- **THEN** 协议将该输入视为不可跳过，缺少值时校验失败

#### Scenario: 非必填输入允许跳过
- **WHEN** case 中某输入标记为非必填且允许跳过
- **THEN** 调用方可不提供该输入值，校验在其余必填项满足时仍可通过

#### Scenario: img2img 需要图片与可选文本
- **WHEN** case 的 input schema 依次定义 image（必填）与 string（可选）
- **THEN** 协议要求先满足图片输入，文本可缺省或跳过

### Requirement: Output schema 支持图片、文本与文件扩展
系统 MUST 将 output 定义为有序数组。每个 output 项 MUST 包含：字段键、类型、描述。类型 MUST 至少支持：`image`、`text`、`file`。视频等二进制产物 MAY 以 `file`（或带媒体提示的 file 元数据）表示，以便调用方按 schema 回收结果。

#### Scenario: 文生图产出图片
- **WHEN** text2img case 的 output schema 声明一个 image 输出
- **THEN** 执行成功后结果中必须能按该字段键取回图片内容或可访问引用

#### Scenario: 文生视频产出文件
- **WHEN** text2video case 的 output schema 声明一个 file 输出（视频）
- **THEN** 执行成功后结果中必须能按该字段键取回对应文件内容或可访问引用

### Requirement: 输入校验遵循 schema 的表单规则
系统 MUST 使用 **JSON Schema**（或与其语义等价的引擎）对提交的输入集合执行校验，规则至少覆盖：必填、类型匹配、`number` 的最小/最大（若配置）、`string` 的最小/最大长度（若配置）、`enum` 的枚举集合约束。对 `image`/`video` 等媒体输入，系统 MUST 在 Schema 校验之外提供扩展钩子校验引用可解析性与允许的 MIME（若配置）。校验失败 MUST 返回按字段聚合的错误信息，且不得创建执行 Task 或向 ComfyUI 提交工作流。

#### Scenario: 类型不匹配导致失败
- **WHEN** schema 要求 image，但调用方提供了纯文本值
- **THEN** 校验失败并指出该字段类型错误，不触发执行

#### Scenario: 全部输入合法则通过
- **WHEN** 调用方按 schema 提供全部必填输入且类型与约束均满足
- **THEN** 校验通过，输入可进入 ConfirmRun 物化与后续编排

#### Scenario: JSON Schema 引擎拒绝非法枚举
- **WHEN** enum 字段取值不在 schema 枚举集合内
- **THEN** 校验失败且不创建 Task

### Requirement: 价格字段仅为协议数值元数据
系统 MUST 在 case 上提供数值型价格/费用字段以供展示或后续积分系统读取。本阶段 MUST NOT 实现扣费、余额校验或支付流程。

#### Scenario: 查询 case 可见价格数值
- **WHEN** 调用方读取已注册 case
- **THEN** 响应包含协议中的价格数值字段，且不产生任何扣费副作用

### Requirement: 场景类别可扩展而不锁死实现集合
系统 MUST 允许 case 使用类别标签表达场景（例如 text2img、text2video、img2img、img2video、videoedit、imgedit，以及后续可能的 upscale、inpaint、remove-bg 等）。协议 MUST NOT 将可运行能力硬编码为固定枚举实现集合；具体能力由该 case 的 input/output schema 与 ComfyUI 绑定决定。

#### Scenario: 使用已知标签注册 case
- **WHEN** 调用方以 `text2img` 标签注册 case
- **THEN** 系统保存该标签，并仍以 schema 为准决定输入输出行为

#### Scenario: 使用新标签扩展场景
- **WHEN** 调用方以尚未内置示例的标签（如 `upscale`）注册合法 case
- **THEN** 系统接受该 case，不因标签不在预置列表示例中而拒绝

```

## docs/openspec/changes/workflow-engine-core/specs/workflow-registry/spec.md

- Source: docs/openspec/changes/workflow-engine-core/specs/workflow-registry/spec.md
- Lines: 1-34
- SHA256: bec4e42e7258d0be39350b23a32810617ecaa8bc87ca52bb248a47568ea02601

```md
## ADDED Requirements

### Requirement: 通过仓储注册与持久化 Case
系统 MUST 提供工作流/case 注册能力，将符合协议的 case 定义持久化到数据库。默认实现 MUST 使用 SQLite；领域模型与仓储接口 MUST 不绑定 SQLite 专有 SQL，以便后续切换 MySQL。系统 MUST 使用 ORM 管理实体映射与迁移。

#### Scenario: 成功注册新 case
- **WHEN** 调用方提交合法的 case 定义进行注册
- **THEN** 系统将其写入数据库，并可用同一标识再次查询到完整定义

#### Scenario: 重复标识注册冲突
- **WHEN** 调用方使用已存在的 case 标识再次注册且策略为禁止覆盖
- **THEN** 系统拒绝注册并返回冲突错误

### Requirement: 查询与更新已注册 Case
系统 MUST 支持按标识获取单个 case、列出 case（至少支持按类别标签过滤），以及更新已有 case 的元数据、schema 或 ComfyUI 绑定。更新后的定义 MUST 成为后续校验与执行的唯一来源。

#### Scenario: 按标识读取
- **WHEN** 调用方请求某个已注册 case 的标识
- **THEN** 系统返回完整协议字段与绑定信息

#### Scenario: 按类别列出
- **WHEN** 调用方按 `img2img` 类别过滤列表
- **THEN** 系统仅返回带有该标签的 case

#### Scenario: 更新 ComfyUI 绑定
- **WHEN** 调用方更新某 case 的节点注入映射并保存
- **THEN** 之后的执行使用更新后的绑定，而不是旧映射

### Requirement: 禁用或删除 Case 不影响历史执行记录边界
系统 MUST 支持将 case 标记为停用（或等价不可用状态），停用后 MUST NOT 接受新的执行请求。若提供删除，删除策略 MUST 明确；本阶段允许硬删除或停用二选一，但行为 MUST 文档化且一致。

#### Scenario: 停用后拒绝新执行
- **WHEN** 某 case 已被停用，调用方仍请求执行该 case
- **THEN** 系统拒绝执行并说明 case 不可用

```
