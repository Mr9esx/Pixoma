# Comet Design Handoff

- Change: edge-agent-topic-routing
- Phase: design
- Mode: compact
- Context hash: df09265333a7249fe8b48e4d8d98667109a57cdf071fdea4643a48e9721b3bc9

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## docs/openspec/changes/edge-agent-topic-routing/proposal.md

- Source: docs/openspec/changes/edge-agent-topic-routing/proposal.md
- Lines: 1-39
- SHA256: 6dbb3495b982f7214308660e8768a11b1e59a354e4dc5cd7298d62f8b4671b4d

```md
## Why

家里/内网的 ComfyUI 无法被云上 Bot、Admin 直连，且 Comfy 无现成鉴权、不宜对公网暴露。需要把执行面外置为 Edge-Agent（主动出网消费 MQ、只调本机 Comfy），并用 Topic + 投放表达式按会员等级与 Case 分类把任务分到不同机器。

本 change 刻意保持为单一需求包（用户明确不拆 batch）：共享存储、跨进程队列、方案 A 任务包、Edge 进程、Topic 管理与表达式路由一并纳入，以便端到端闭环。

## What Changes

- 新增 **Edge-Agent** 独立进程：按配置订阅一个或多个 dispatch Topic；拉取任务包与输入图；本机 Upload/Submit/Wait；产物写回共享 Blob；发布 `task.status`。
- **方案 A 任务包**：云上调度/prep 在发 dispatch 前组装半成品（已注入非图片字段的 workflow + 图片 BlobRef 列表），写入共享 Blob；`DispatchCommand` **BREAKING** 增加 `job_ref`（可废弃 Edge 侧对业务 DB 的依赖）。
- Blob 从「仅 Bot 本机 localfs 可用」升级为 **云与 Edge 均可 Put/Get 的共享对象存储**（接口仍为 `blob.Store`；具体后端在 design 选定）。
- Queue 从进程内 Memory 扩展为 **可跨进程的 MQ 适配器**（保留 Port；Memory 可继续服务单机/mock）。
- 后台支持 **Topic 管理**与 **投放表达式**（输入维度含用户会员等级如 Lv1/Lv2、Case 分类如图片/视频等）；调度器按表达式选择 dispatch Topic。
- Edge 配置声明自身订阅的 Topic 列表；与后台 Topic 目录对齐。
- **Mock 同演进**：`comfy_mock` 路径仍须端到端可通（可用进程内或本地 MQ + mock Comfy/Edge 替身）。

## Capabilities

### New Capabilities

- `edge-agent`: Edge 进程职责、配置（含订阅 Topic）、本机 Comfy 调用、状态回传与鉴权边界（不连业务 DB）。
- `dispatch-job-package`: 方案 A 任务包格式、云上 prep、`job_ref` 与 Edge 消费约定。
- `shared-blob-store`: 跨进程/跨主机共享 Blob 的行为与可达性要求。
- `cross-process-queue`: 跨进程 Publish/Subscribe 语义与 topic 命名约定。
- `dispatch-topic-routing`: Topic 目录、投放表达式求值、调度选 Topic、与 Edge 订阅的匹配关系。
- `topic-admin`: 后台对 Topic 与投放表达式的管理能力（API/控制台范围在 design 收束）。

### Modified Capabilities

- `task-orchestrator`: 认领后按表达式选 Topic 再 Publish dispatch；与 prep/job_ref 衔接。
- `comfyui-executor`: 执行面从同进程 Actuator 迁移/对接到 Edge 消费模型（同进程可保留为开发/mock 形态）。
- `comfy-instance-pool`: 实例与 Topic/Edge 的关联方式调整（不再假设云侧直连探活 Comfy 为唯一健康来源；可演进为 Edge 心跳等，细节在 design）。

## Impact

- 进程拓扑：Bot/调度（云）与 Edge（Comfy 侧）分离；Comfy 仅本机可达。
- 事件：`dispatch.*` 载荷与 topic 命名；可能新增 Topic 实体与表达式配置持久化。
- 依赖：对象存储、跨进程 MQ；Admin API/Web 增加 Topic/表达式管理。
- 风险：表达式错误导致误投；Blob/MQ 未就绪时 Edge 无法工作；需明确 mock 与真实路径开关。

```

## docs/openspec/changes/edge-agent-topic-routing/design.md

- Source: docs/openspec/changes/edge-agent-topic-routing/design.md
- Lines: 1-92
- SHA256: 59de22d465120a41a36126fad0bad642c0744968b597fc86f718cc891ea999f5

[TRUNCATED]

```md
## Context

当前 Bot 进程内同时包含 Orchestrator（调度）与 Actuator（执行），Queue 为进程内 Memory，Blob 为 Bot 本机 `localfs`。调度按健康实例 round-robin，向 `dispatch.<instance_id>` 投递瘦载荷 `{task_id, instance_id, input_prefix}`；Actuator 同进程读 DB+Blob 拼图并直连 Comfy HTTP。

约束：家里/内网 Comfy 不可被云直连；Comfy 不宜对公网暴露；大文件不进 MQ；`comfy_mock` 须继续端到端可通；用户确认本 change 不拆 batch。

已锁定：**方案 A**（云上 prep 半成品 + `job_ref`）；Edge = 外置 Actuator；Topic 由后台管理；投放表达式基于会员等级与 Case 分类；Edge 配置订阅 Topic。

## Goals / Non-Goals

**Goals:**

- 云侧 prep 写出可被 Edge 消费的任务包到共享 Blob；`dispatch` 携带 `job_ref` 与目标 Topic。
- Edge 独立进程：订阅配置的 Topic → 拉 job/图 → 本机 Comfy → 产物回写 Blob → `task.status`。
- 共享 Blob、跨进程 MQ 适配器落地（Port 不变，换实现/配置）。
- Topic 目录 + 投放表达式；调度按表达式选 Topic；Admin 可管理。
- Mock/单机开发路径仍可闭环。

**Non-Goals:**

- Edge 直连业务 Postgres 拼装（方案 C）。
- 本期强取消 running（Comfy interrupt）不变。
- 不在本期绑定某一云厂商专有服务为唯一实现（设计选定默认开源/可自托管组合即可）。
- 不把完整会员计费/支付做进本 change（只消费已有或可扩展的会员等级字段）。

## Decisions

### D1. 执行面外置为 Edge-Agent（复用 Actuator 语义）

- **选择**：独立二进制/进程承载今日 `actuator.Worker` 的执行职责；云 Bot 默认不再同进程订阅生产 dispatch（开发/mock 可同机嵌入）。
- **备选**：反向隧道把家里 Comfy 暴露给云 — 拒绝（鉴权与暴露面差）。
- **理由**：与既有「Actuator 跟实例走」模型一致，只改部署与材料获取。

### D2. 方案 A：`job_ref` + Blob 内任务包

- **选择**：prep 在 Publish dispatch 前写入例如 `jobs/<task_id>/job.json`；`DispatchCommand` 增加 `job_ref`（`BlobRef`）。任务包含：已注入非图片字段的 `workflow`、`images[]`（node/field + 输入 BlobRef）、`output_prefix` 等。
- **备选**：整包塞 MQ；Edge 读 DB — 拒绝。
- **理由**：MQ 保持小；Edge 不碰业务库。

### D3. 大文件通道：共享对象存储（`blob.Store`）

- **选择**：引入可跨主机的 Blob 后端（默认倾向 S3 兼容如 MinIO；接口仍 Put/Get）。Edge 与云使用同一逻辑 bucket/前缀约定。
- **备选**：仅自研 HTTP 传文件 API — 可作为适配器，但语义仍是对象存储。
- **理由**：与现有 BlobRef 模型一致。

### D4. 跨进程 Queue

- **选择**：保留 `queue.Publisher`/`Subscriber`；新增至少一种跨进程适配器（具体中间件在实现前在 Open Questions 收口，候选 NATS / Redis Streams）。Topic 字符串与业务名对齐：`task.created`、`task.status`、以及可配置的 `dispatch.<topic_key>`。
- **备选**：继续仅 Memory — 无法支撑家里 Edge。
- **理由**：端口已存在，二期换适配器的原设计意图。

### D5. Topic 与投放表达式

- **选择**：
  - **Topic**：后台可 CRUD 的逻辑投递目标（稳定 `key`、显示名、启用状态等）；Edge 配置 `subscribe_topics: [key, ...]`。
  - **表达式**：对「用户会员等级 + Case 分类（图片/视频等）」求值，产出目标 Topic key。支持类似「Lv1 → topicA」「Lv2 AND category=image → topicB」的规则集；求值失败走显式默认 Topic 或保持 pending 并记原因（实现选一种并在 spec 写死）。
  - **调度**：Claim 前/时求值 Topic；Publish 到该 Topic；Task 记录实际 `dispatch_topic`（及可选 instance/edge 标识）。
- **备选**：仅 `dispatch.<instance_id>` — 无法表达会员×分类策略。
- **理由**：与用户描述一致；Edge 只订配置 Topic，不解析表达式。

### D6. 健康与选路

- **选择**：生产路径以 Edge **心跳/在线**（或「该 Topic 有活跃消费者」信号）作为可投递条件；云侧对家里 Comfy 的 HTTP 探活降为可选/仅局域网。`comfy_mock` 仍可用进程内 Mock 执行面。
- **备选**：继续云直连探 Comfy — 对家里机器不可行。

### D7. Admin

- **选择**：Admin API（及必要 Web 页）管理 Topic 与投放规则；实例/Edge 注册信息可关联默认订阅 Topic（细节实现阶段与现有 `comfy-instance-*` 对齐或新增 edge 资源）。

## Risks / Trade-offs

- [表达式误配导致全员进错池] → 管理端校验、默认 Topic、审计日志、干跑/试算 API（可分期）。
- [Blob/MQ 未就绪 Edge 饿死] → 启动自检；Task 保持 pending；可观测告警。
- [job 与输入图生命周期] → 约定 TTL/前缀清理策略；失败重试幂等。
- [单 change 范围大] → tasks 按里程碑切片；用户明确要求不拆 OpenSpec change。
- [BREAKING dispatch 载荷] → 版本字段或双读过渡期；mock/单机路径同步改。

## Migration Plan

1. 落地共享 Blob + 跨进程 MQ，双写/开关切换。

```

Full source: docs/openspec/changes/edge-agent-topic-routing/design.md

## docs/openspec/changes/edge-agent-topic-routing/tasks.md

- Source: docs/openspec/changes/edge-agent-topic-routing/tasks.md
- Lines: 1-28
- SHA256: 34e0ccfe392aa9efc300ed3c3e50c2cffb47ff692cbafcb18cc5bca66821a588

```md
## 1. 模式、Blob、Queue

- [ ] 1.1 引入 `runtime_mode=allinone|split` 与 queue/blob 驱动配置；启动组合校验
- [ ] 1.2 保留 localfs；新增 S3 兼容 `blob.Store`；前缀 `inputs/` `jobs/` `outputs/`
- [ ] 1.3 保留 Memory；新增 Redis Streams `queue.Bus`；topic ↔ stream 映射与 Ack
- [ ] 1.4 适配器单测/集成测（Memory、localfs、Redis、S3 或测试替身）

## 2. 方案 A 任务包与 Dispatch

- [ ] 2.1 定义 job JSON 与 `job_ref`；扩展 `DispatchCommand`
- [ ] 2.2 实现 prep（非图片注入 + images 清单）；禁止 prep 调家里 UploadImage
- [ ] 2.3 执行面改为读 job → 本机 Upload → Submit/Wait → outputs → status
- [ ] 2.4 allinone 同进程接线：prep + 执行面 + Memory + localfs
- [ ] 2.5 缺输入/坏 job_ref 失败路径测试

## 3. Edge（split）

- [ ] 3.1 新增 `edge-agent` 入口（Redis、S3、本机 Comfy、instance_id、subscribe topic、mock）
- [ ] 3.2 云侧 split 默认不订阅生产 dispatch；Edge 心跳/在线信号最小实现
- [ ] 3.3 无在线 Edge 时保持 pending；相关测试
- [ ] 3.4 split 冒烟：Redis + S3 + Edge + mock/真 Comfy（可文档化手动步骤）

## 4. Mock、文档与验收

- [ ] 4.1 allinone + `comfy_mock` 端到端：确认生成 → 收到产物
- [ ] 4.2 更新 `docs/architecture/`（runtime/overview）：双模式、job_ref、Edge
- [ ] 4.3 回归 orchestrator/actuator/confirm_run 等相关测试
- [ ] 4.4 （延后，不阻塞本期）Topic Admin / 投放表达式 / 会员×分类分流

```

## docs/openspec/changes/edge-agent-topic-routing/specs/comfy-instance-pool/spec.md

- Source: docs/openspec/changes/edge-agent-topic-routing/specs/comfy-instance-pool/spec.md
- Lines: 1-12
- SHA256: dbf794be32649ab117c70e380eb56cc6167e1914257c67b28a71575985af8ebb

```md
## ADDED Requirements

### Requirement: 实例或 Edge 与 Topic 关联
系统 MUST 能表达执行端（Comfy 实例元数据与/或 Edge 注册信息）与其订阅/服务的逻辑 Topic 之间的关联，供调度判断 Topic 是否可投递，并供运维查看。云侧 MUST NOT 将「对家里 Comfy 的 HTTP 探活成功」作为家里部署场景下唯一的健康信号。

#### Scenario: 记录 Edge 订阅的 Topic
- **WHEN** 某 Edge 配置订阅 Topic `vip` 并成功上报在线
- **THEN** 控制面能查询到 `vip` 存在可用消费者

#### Scenario: 家里场景不以云直连 Comfy 为唯一健康条件
- **WHEN** 实例位于不可被云直连的网络，但对应 Topic 的 Edge 在线
- **THEN** 该 Topic 仍可作为可投递目标（在其他条件满足时）

```

## docs/openspec/changes/edge-agent-topic-routing/specs/comfyui-executor/spec.md

- Source: docs/openspec/changes/edge-agent-topic-routing/specs/comfyui-executor/spec.md
- Lines: 1-39
- SHA256: d91c414aab567ee1cd563268e418a581c9bb63ce37db74230024aaebe1542e84

```md
## MODIFIED Requirements

### Requirement: Actuator 作为执行面调用 ComfyUI
系统 MUST 提供执行面（同进程 Actuator 与/或独立 Edge-Agent）：消费指向本进程订阅的 dispatch Topic 上的命令。成功主路径 MUST 通过 dispatch 中的 `job_ref` 加载方案 A 任务包，完成本机图片 Upload 与 Submit/Wait（Mock 或真实 HTTP），MUST NOT 以读取业务 Case/Task 数据库拼装 workflow 作为成功主路径。执行面 MUST NOT 直接更新全局 Task 表的终态；MUST 通过 status 事件向 Orchestrator 上报。当配置为真实多机 Edge 时，各 Edge MUST 仅处理其订阅 Topic 上的消息。

#### Scenario: 收到含 job_ref 的 dispatch 后执行
- **WHEN** 执行面收到合法 dispatch 且 `job_ref` 可读，本机 Comfy（或 Mock）可用
- **THEN** 系统提交工作流，并上报至少一次 running 与一次终态 status（succeeded 或 failed）

#### Scenario: ComfyUI 不可达时上报失败 status
- **WHEN** 本机 ComfyUI 不可达或返回连接错误（真实 HTTP 模式）
- **THEN** 执行面上报 failed status（含可诊断信息），且不假装成功

#### Scenario: Mock 开关开启时仍完成主路径
- **WHEN** `comfy_mock`（或等价环境变量）为真
- **THEN** 同一任务包注入与 status 路径可成功完成并产生可投递图片产物

#### Scenario: 不维护独立 Ledger 真相源
- **WHEN** 执行面执行任务并上报 status（含 prompt_id / 终态）
- **THEN** 系统不以 MemoryLedger/LocalRun 作为可恢复执行态的唯一或权威存储；可恢复状态落在 Task（经 Orchestrator 写回）

### Requirement: 按绑定映射注入输入
系统 MUST 在云上 prep 阶段依据 Case 绑定将非图片输入注入工作流图，并将图片以「节点/字段 + BlobRef」形式写入任务包。执行面 MUST 依据任务包完成图片 Upload 与字段写入。缺少映射、映射目标不存在、或必填物化输入缺失时，系统 MUST 在调用 ComfyUI 前失败并上报 failed status（失败可发生在 prep 或执行面，但 MUST 可观测）。

#### Scenario: 文本在任务包中已注入
- **WHEN** case 将 `prompt` 映射到某节点文本字段，且 staged 输入含该文本
- **THEN** 任务包内 workflow 中该节点字段等于已物化 prompt 值

#### Scenario: 图片由执行面注入
- **WHEN** 任务包 images 列表含某节点图片绑定与 BlobRef
- **THEN** Submit 前执行面完成本机上传，且工作流中该字段指向可用的图片输入

#### Scenario: 映射缺失导致拒绝执行
- **WHEN** 某必填 input 没有有效的 ComfyUI 注入映射
- **THEN** 不调用 ComfyUI，并上报指出映射问题的 failed status

#### Scenario: 空 workflow 拒绝执行
- **WHEN** 任务包中 workflow 为空或不含可提交图
- **THEN** 不调用 ComfyUI，并上报失败 status

```

## docs/openspec/changes/edge-agent-topic-routing/specs/cross-process-queue/spec.md

- Source: docs/openspec/changes/edge-agent-topic-routing/specs/cross-process-queue/spec.md
- Lines: 1-26
- SHA256: 2f0bcacd1e6d64848e00dfce8bc38041e43798a47392e809284632ec5a753bc4

```md
## ADDED Requirements

### Requirement: Queue 双驱动（Memory 与 Redis Streams）
系统 MUST 提供可配置的 Queue 适配器：至少支持进程内 Memory 与 Redis Streams。`allinone` MUST 使用 Memory；`split` MUST 使用 Redis Streams（或启动校验拒绝 Memory）。控制面 Publish 的消息在 split 下 MUST 能被 Edge 进程 Subscribe 收到。

#### Scenario: allinone Memory 闭环
- **WHEN** runtime_mode 为 allinone 且 queue 为 memory
- **THEN** 同进程内 Publish/Subscribe 可完成主路径

#### Scenario: split 跨进程收到 dispatch
- **WHEN** 调度进程向 `dispatch.<instance_id>` Publish，Edge 已订阅对应 Redis Stream
- **THEN** Edge handler 收到等价 payload

### Requirement: 模式与驱动一致性校验
系统 MUST 在启动时校验 runtime_mode 与 queue/blob 驱动组合合法；非法组合 MUST 失败并给出可诊断错误。

#### Scenario: split 配置 Memory 被拒绝
- **WHEN** runtime_mode 为 split 且 queue 驱动为 memory
- **THEN** 进程拒绝就绪或退出，并说明原因

### Requirement: 业务 Topic 命名稳定
系统 MUST 保持 `task.created`、`task.status` 稳定；dispatch MUST 使用 `dispatch.<instance_id>`（或实例上配置的 DispatchTopic）。本期 MUST NOT 要求投放表达式派生 Topic。

#### Scenario: 按实例 Topic 投递
- **WHEN** 调度选定实例 `gpu-1`
- **THEN** 消息发布到该实例 dispatch Topic

```

## docs/openspec/changes/edge-agent-topic-routing/specs/dispatch-job-package/spec.md

- Source: docs/openspec/changes/edge-agent-topic-routing/specs/dispatch-job-package/spec.md
- Lines: 1-30
- SHA256: 597d75fa81f222adf3e66138da579ca32a2316c765909f50156239dbd6e9c037

```md
## ADDED Requirements

### Requirement: 云上组装任务包并写入 Blob（双模式）
在发布 dispatch 之前，控制面 MUST 组装方案 A 任务包并写入当前模式配置的 Blob。任务包 MUST 包含：已注入非图片输入的 workflow；图片的节点/字段与输入 `BlobRef`；产物前缀约定。任务包 MUST NOT 内嵌图片二进制。allinone 与 split MUST 共用该语义。

#### Scenario: prep 写出 job 对象
- **WHEN** 调度准备投递某 Task 且 Case 与 staged 输入完整
- **THEN** Blob 中存在该 Task 的任务包，且 workflow 已含非图片字段值

#### Scenario: 缺少必填输入则不发布 dispatch
- **WHEN** 必填 staged 输入缺失
- **THEN** 不发布 dispatch，并记录可诊断原因

### Requirement: Dispatch 携带 job_ref
`DispatchCommand` MUST 包含 `job_ref`。执行面 MUST 仅依据 `job_ref` 与任务包获取执行材料，成功主路径 MUST NOT 依赖 `input_prefix` 回查 Case 库。

#### Scenario: 执行面凭 job_ref 拉取材料
- **WHEN** 执行面收到含有效 `job_ref` 的 dispatch
- **THEN** 能 Get 到任务包，并按 images 列表从 Blob 拉取图片

#### Scenario: job_ref 无效则失败 status
- **WHEN** `job_ref` 指向不存在对象或任务包无法解析
- **THEN** 上报 failed status，且不向 ComfyUI 假装成功提交

### Requirement: 执行面完成本机图片注入
对任务包中的每张图片，执行面 MUST 上传到本机 ComfyUI（或 Mock），将本地引用写入 workflow 后再 Submit。

#### Scenario: 图片注入后提交
- **WHEN** 任务包含一张 image 绑定
- **THEN** Submit 前该节点字段为本机 Upload 返回的可用引用

```

## docs/openspec/changes/edge-agent-topic-routing/specs/dispatch-topic-routing/spec.md

- Source: docs/openspec/changes/edge-agent-topic-routing/specs/dispatch-topic-routing/spec.md
- Lines: 1-15
- SHA256: be7626f64e47f03e24920f14f0433b4df79aada49fd242089eb4b0c4df3af727

```md
## ADDED Requirements

### Requirement: 本期不实现投放表达式选路
本期实现 MUST NOT 将「用户会员等级 × Case 分类投放表达式」作为调度选路的验收路径。调度 MUST 继续按实例维度选择目标，并向 `dispatch.<instance_id>`（或实例配置的 DispatchTopic）投递。Topic 表达式分流延后到后续 change。

#### Scenario: 调度按实例 Topic 投递
- **WHEN** 某 pending Task 被成功调度到实例 `gpu-1`
- **THEN** dispatch 发布到该实例对应 Topic，且不依赖会员/分类表达式求值

### Requirement: Topic 无消费者时不假装投递成功
当目标实例的 dispatch Topic 在 split 模式下无在线 Edge（或等价消费者）时，系统 MUST 保持 Task 可重试并记录原因，MUST NOT 假装已可靠交付执行。

#### Scenario: split 下无在线 Edge
- **WHEN** runtime_mode 为 split 且目标实例无在线 Edge
- **THEN** Task 保持 pending（或未被标记为已可靠 queued）

```

## docs/openspec/changes/edge-agent-topic-routing/specs/edge-agent/spec.md

- Source: docs/openspec/changes/edge-agent-topic-routing/specs/edge-agent/spec.md
- Lines: 1-29
- SHA256: 62768b5a73bc600effd2083cd31e4f680597f33ce5eb1cde568d242678f8d644

```md
## ADDED Requirements

### Requirement: Edge-Agent 独立进程消费 dispatch（split）
在 `runtime_mode=split`（或等价）下，系统 MUST 提供可独立部署的 Edge-Agent 进程。该进程 MUST 订阅其 `instance_id` 对应的 dispatch Topic，消费含合法 `job_ref` 的 dispatch，调用本机可达的 ComfyUI（Mock 或真实 HTTP），并通过 `task.status` 回报。Edge-Agent MUST NOT 连接业务 Task/Case 数据库以拼装工作流。

#### Scenario: 订阅实例 Topic 后执行任务
- **WHEN** Edge 订阅 `dispatch.<instance_id>` 且收到合法 `job_ref` 的 dispatch
- **THEN** Edge 拉取任务包并完成本机执行，上报至少一次 running 与一次终态 status

### Requirement: ALLINONE 同进程执行面
在 `runtime_mode=allinone` 下，系统 MUST 在单一 OS 进程内组装 Bot、调度与执行面；执行面 MUST 消费本进程内投递的 dispatch，并使用与 Edge 相同的方案 A 任务包语义（`job_ref`）。

#### Scenario: allinone 单进程完成主路径
- **WHEN** runtime_mode 为 allinone，queue 为 memory，blob 为 localfs，且 comfy_mock 或本机 Comfy 可用
- **THEN** ConfirmRun 后任务可收敛并产生可投递产物引用

### Requirement: Edge/执行面仅访问本机 Comfy 与配置的 Blob/Queue
执行面 MUST 通过配置的 Blob 读写输入/产物，通过配置的 Queue 收发消息。split 模式下 MUST NOT 要求云对家里 Comfy 发起入站连接。

#### Scenario: split 无公网 Comfy 暴露仍可完成
- **WHEN** ComfyUI 仅监听 localhost，Edge 与 Comfy 同机，S3 与 Redis 出网可达
- **THEN** 任务仍可成功执行并回写产物引用

### Requirement: Mock 路径在双模式下仍可通
当 `comfy_mock` 为真时，系统 MUST 能在不依赖真实 ComfyUI 的情况下完成「dispatch → 执行 → 产物 Blob → status」主路径（allinone 同进程或 split 下 mock 执行面）。

#### Scenario: mock 开关开启完成主路径
- **WHEN** `comfy_mock` 为真并投递一条合法任务
- **THEN** 产生可读取的输出 BlobRef 且 Task 可收敛为成功

```

## docs/openspec/changes/edge-agent-topic-routing/specs/shared-blob-store/spec.md

- Source: docs/openspec/changes/edge-agent-topic-routing/specs/shared-blob-store/spec.md
- Lines: 1-23
- SHA256: 36e67f043f0b9e56f6c05a9a30b9d679adb368556c831ab7e103b2d59b1d1193

```md
## ADDED Requirements

### Requirement: Blob 双驱动（localfs 与 S3 兼容）
系统 MUST 提供可配置的 `blob.Store` 后端：至少支持 localfs 与 S3 兼容实现。`allinone` 默认使用 localfs；`split` 跨机主路径 MUST 使用 S3 兼容（或等价共享对象存储），使得云 Put 的对象可被 Edge Get，Edge Put 的产物可被云通知路径 Get。逻辑 key 前缀（`inputs/`、`jobs/`、`outputs/`）MUST 一致。

#### Scenario: allinone 使用 localfs
- **WHEN** runtime_mode 为 allinone 且 blob 驱动为 localfs
- **THEN** 输入、任务包与产物均可通过同一 localfs 根读写

#### Scenario: split 云写入 Edge 可读
- **WHEN** 控制面将对象 Put 到 S3 兼容存储的 key `inputs/<task_id>/...`
- **THEN** Edge 使用同一 key 能 Get 到相同内容

#### Scenario: Edge 写入产物云可读
- **WHEN** Edge 将产物 Put 到 `outputs/<task_id>/...`
- **THEN** Bot 通知路径能 Get 该对象并投递给用户

### Requirement: 引用仍使用 BlobRef
跨进程传递文件 MUST 继续使用 `BlobRef`，MUST NOT 在 MQ 消息中携带文件二进制作为成功主路径。

#### Scenario: status 仅带引用
- **WHEN** 执行成功
- **THEN** `task.status` 的 outputs 为 BlobRef 列表而非文件字节

```

## docs/openspec/changes/edge-agent-topic-routing/specs/task-orchestrator/spec.md

- Source: docs/openspec/changes/edge-agent-topic-routing/specs/task-orchestrator/spec.md
- Lines: 1-25
- SHA256: 146c81ade65d8c632e82e21bcb395d40411d8609a60649202b8873cd132ac088

```md
## ADDED Requirements

### Requirement: 投递前组装 job 并携带 job_ref
Orchestrator（或其 prep 步骤）在 Publish dispatch 之前 MUST 组装方案 A 任务包并获得 `job_ref`。dispatch 载荷 MUST 包含该 `job_ref`。成功主路径 MUST NOT 再依赖执行面读取业务 Case/Task 库拼装 workflow。该要求在 allinone 与 split 下均生效。

#### Scenario: 调度发出的 dispatch 含 job_ref
- **WHEN** 某 pending Task 被成功调度
- **THEN** 对应 dispatch 消息含有效 `job_ref`，且 Blob 中存在可读任务包

## MODIFIED Requirements

### Requirement: 调度与 pending 扫描双触发
系统 MUST 支持事件触发调度，并 MUST 提供定时扫描 `pending` 的兜底。本期调度 MUST 按实例选择目标（健康/熔断/在线消费者等条件），向 `dispatch.<instance_id>`（或实例 DispatchTopic）Publish 含 `job_ref` 的 dispatch，并将 Task 置为 `queued`（幂等）。本期 MUST NOT 以实现投放表达式选 Topic 为验收条件。当没有可投递实例/消费者时 MUST 保持 Task 为 `pending` 并记录原因，MUST NOT 假装已排队。split 模式下 MUST NOT 假设云侧一定能直连家里 ComfyUI。

#### Scenario: 创建事件丢失仍被扫到
- **WHEN** Task 长期处于 pending 且无成功调度
- **THEN** SchedulePending 仍能将其投递（若存在可投递实例/消费者）

#### Scenario: 按实例 Topic 投递
- **WHEN** 选定实例 `gpu-1` 且可投递
- **THEN** dispatch 发布到该实例 Topic，且含 `job_ref`

#### Scenario: 无可投递目标时不投递
- **WHEN** 无健康可投递实例（或 split 下无在线 Edge）
- **THEN** Task 仍保持 pending，且不假装成功投递

```

## docs/openspec/changes/edge-agent-topic-routing/specs/topic-admin/spec.md

- Source: docs/openspec/changes/edge-agent-topic-routing/specs/topic-admin/spec.md
- Lines: 1-8
- SHA256: 4b3dc2172c0a66146f7aa863c869a8354497a318be59acec9ba8117a9487ceb4

```md
## ADDED Requirements

### Requirement: 本期不实现 Topic 与投放规则管理验收
本期 MUST NOT 将 Topic 目录 CRUD、投放表达式管理、试算 API 作为交付验收项。相关后台能力延后到后续 change。若代码中预留表结构，MUST NOT 阻塞 allinone/split 与方案 A 主路径。

#### Scenario: 主路径不依赖 Topic Admin
- **WHEN** 未配置任何 Topic Admin 资源
- **THEN** allinone 或按实例 dispatch 的 split 主路径仍可完成任务（在实例/Edge 可用时）

```
