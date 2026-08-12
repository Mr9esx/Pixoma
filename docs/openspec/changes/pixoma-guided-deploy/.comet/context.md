# Comet Design Handoff

- Change: pixoma-guided-deploy
- Phase: design
- Mode: compact
- Context hash: 122a8a3ebb1fcd2cfd3e142125f3f1090eb8b2e1533ff2ae301417f8f074ad43

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## docs/openspec/changes/pixoma-guided-deploy/proposal.md

- Source: docs/openspec/changes/pixoma-guided-deploy/proposal.md
- Lines: 1-38
- SHA256: 97ef8c63fb934dd656c3ea6c0c84629bbddfc801b3f4e7fa717141afc994c122

```md
## Why

当前部署依赖多进程、YAML/环境变量，以及 `allinone`/`split`、queue、blob 组合矩阵，上手成本高。需要一版「下载即跑、后台向导配完」的体验：默认无 Redis、无用户可见 queue，执行面统一为 Edge，派活改为数据库领取 + 长轮询。

**不拆 change 的原因（用户确认）**：本版一次性交付引导启动、配置落库、Edge 拉取与向导，避免半成品拓扑并存过久；接受单 change 体量大、验证周期长。

## What Changes

- 引入控制面一体入口（`pixoma`）：零配置可启动；日志输出后台地址与默认账号密码；未初始化仅开放登录与向导。
- 配置从「YAML 为主」转为「引导态 + 业务库 settings」；向导完成库、本机/远程、存储、节点/Comfy、渠道等步骤。
- **BREAKING（默认路径）**：取消用户必选的 `runtime_mode` / `queue.driver`；默认派发不再依赖 Redis Streams / 进程内 memory 跨进程总线。
- Edge 统一为 `pixoma-edge-agent`：主动连接控制面，长轮询/领取任务（claim + lease），回报状态；同机可用 localfs，远程用对象存储。
- 本机可自动拉起 Edge（推荐）或文档手起；远程仅提供安装说明与节点登记。
- 保留紧急覆盖（env）用于运维；旧 YAML 可过渡期兼容，但新部署以向导为准。

## Capabilities

### New Capabilities

- `platform-bootstrap`：零配置启动、引导态存储、默认管理员凭证与首启门闩、控制面一体二进制边界。
- `setup-wizard`：后台初始化向导步骤、校验矩阵（本机 vs 远程）、设置读写 API。
- `agent-pull-dispatch`：基于 DB 的任务领取/租约/心跳与 Edge↔控制面拉取协议（替代默认跨进程 queue）。

### Modified Capabilities

- `cross-process-queue`：默认部署不再要求 Redis/memory 作为跨进程总线；queue 配置项对用户隐藏或移除。
- `edge-agent`：改为拉取式领取与状态回传；二进制命名/最小配置对齐 `pixoma-edge-agent`。
- `shared-blob-store`：以「本机共用目录 / 远程 OSS」表达，不再绑定 allinone|split 矩阵文案。
- `task-orchestrator`：派发改为「任务可被 Edge claim」，不再默认 Publish 到 Redis topic。
- `admin-api-host` / `admin-web-shell`：未初始化门闩、向导路由与默认登录体验。

## Impact

- 进程：`apps/bot`、`apps/admin-api`、`apps/edge-agent` 与未来 `pixoma` / `pixoma-edge-agent` 入口；`web/admin` 向导页。
- 运行时：`queue` 适配器降级为可选/内部；新增 agent HTTP API；Task 状态机增加 lease 字段语义。
- 配置：`botconfig` / `adminconfig` 读取路径迁移；新增 settings 持久化。
- 文档：README 部署章节改为新故事；架构 `runtime` / `overview` 需同步。
- 依赖：默认路径可去掉 Redis 运行时依赖；对象存储仍为远程必需。

```

## docs/openspec/changes/pixoma-guided-deploy/design.md

- Source: docs/openspec/changes/pixoma-guided-deploy/design.md
- Lines: 1-78
- SHA256: 27337c6cf9bba52aeb5d1b803d7130d6f4c4ce48f16eb9a12a585388866ecd4a

```md
## Context

当前控制面依赖 YAML/`runtime_mode`/`queue`/`blob` 矩阵，跨进程默认走 Redis Streams，同机「allinone」则把执行面塞进 Bot 进程。用户已确认：**永远控制面 + Edge**；**默认无 Redis、无用户可见 queue**；派活改为 **DB + Edge 长轮询领取**；配置经后台向导落库；提供零配置 `pixoma` 与 `pixoma-edge-agent`。

本 change **不拆分**（用户选项 2）：一次性交付引导启动、拉取运行时、向导与落库，避免半成品拓扑长期并存。

## Goals / Non-Goals

**Goals:**

- `pixoma` 零配置可起：日志打印后台 URL + 默认管理员账号密码；未初始化仅放行登录与向导
- 引导态（bootstrap）解决「配置进库 vs 库未就绪」；业务 settings 进业务库
- 向导：库 → 本机/远程 → 存储 → 节点/Comfy 指引 → 渠道（含 TG Token）；远程禁 localfs
- Edge 经 HTTP 长轮询 claim/lease/heartbeat/status；任务可调度态在 DB
- 同机 localfs 共用目录可跑通主路径；远程 OSS（s3|tos）
- 本机可自动拉起 Edge；文档亦支持手起
- 架构/README 改为新部署故事

**Non-Goals:**

- 保留用户必选的 allinone/split、queue.driver 作为一等配置
- 默认路径继续依赖 Redis / 跨进程 memory Bus
- NATS、预签名 CDN、running 任务 Comfy interrupt
- 一次删光所有旧 YAML 兼容（过渡期可 env/YAML 紧急覆盖）

## Decisions

1. **拓扑：始终双进程语义**  
   控制面不内嵌生产执行面；执行只在 Edge。同机也起 Edge（可自动拉起）。  
   - 备选：保留 allinone 同进程执行 → 否决（与「简化心智」冲突）。

2. **跨进程总线：DB + Agent HTTP，不用 Redis**  
   调度把 Task 置为可领取；Edge `GET` 长轮询 claim（带 lease）；`POST` status/heartbeat。  
   - 备选：WebSocket 推送 → 二期；短轮询 → 可作实现细节但契约按长轮询写。  
   - 备选：继续 Redis → 否决为默认。

3. **控制面内部事件**  
   `task.created` 等同进程信号可用函数调用或内存通道，**不**暴露为用户 queue 配置。

4. **两段存储**  
   - Bootstrap：本机小库/文件（initialized、管理员哈希、业务 DSN、向导步）。  
   - App DB：向导所选库；settings + 业务表。

5. **本机 vs 远程（取代 mode 矩阵）**  
   - 本机：localfs + Agent 拉活。  
   - 远程：s3|tos + Agent 出站连控制面；UI 禁止 localfs。

6. **二进制**  
   - `pixoma`：Bot 调度 + 管理 API +（可内嵌）静态后台。  
   - `pixoma-edge-agent`：现 edge-agent 协议改为拉取；最小配置：控制面 URL、instance_id、凭证、blob。

7. **安全**  
   默认监听本机；默认密码仅未初始化时打日志；首次登录强制改密；密钥进库须加密或至少与引导密钥派生。本期管理 API 从「完全无鉴权」升级为「初始化后需管理员会话」（向导/登录）。

8. **生效方式**  
   一期向导「保存并重启生效」；不承诺热切换 Redis/OSS 中途无感。

## Risks / Trade-offs

- [单 change 过大] → 任务按里程碑勾选；Verify 分主路径（本机 mock）与远程（可手工/标签）  
- [claim 双领] → lease + 实例维度唯一领取；单测覆盖  
- [Edge 失联] → lease 过期回可领取；心跳更新在线  
- [密钥进库] → 加密与备份引导态风险写入文档与威胁说明  
- [旧 split+Redis 部署] → 过渡兼容或迁移说明；默认文档只讲新故事  
- [admin 无鉴权 → 有登录] → **BREAKING** 对现有联调脚本；提供默认账密与改密流

## Migration Plan

1. 新部署只走 `pixoma` + 向导 + Edge 拉取。  
2. 旧 YAML/`RUNTIME_MODE`：过渡可读，新 UI 不展示 queue/mode。  
3. 回滚：切回旧二进制/配置（文档标明版本边界）。  
4. 数据：Task 增加 lease/claimed_by 等字段需迁移。

## Open Questions

- Agent 鉴权：共享 token vs mTLS（一期共享 token 即可）  
- 长轮询默认 wait（建议 25s）与 lease（建议 60–120s）具体值 Build 时定  
- 静态后台内嵌进 `pixoma` 还是仍独立 `web/admin` 开发、生产由一体托管（倾向一体托管生产构建产物）

```

## docs/openspec/changes/pixoma-guided-deploy/tasks.md

- Source: docs/openspec/changes/pixoma-guided-deploy/tasks.md
- Lines: 1-35
- SHA256: 6c1193f30ce004f5a28e4a360a993598e4f1eaad60847d7da25e3ab4a58a51e3

```md
## 1. 引导启动与一体控制面

- [ ] 1.1 新增/调整 `pixoma` 入口：零配置启动、引导态存储、日志输出后台 URL 与默认管理员凭证
- [ ] 1.2 未初始化门闩：仅放行登录与向导 API；初始化后启用管理员会话鉴权
- [ ] 1.3 一体托管管理 HTTP（同进程或明确子服务）；健康检查可用
- [ ] 1.4 首次登录强制改密；改密后启动日志不再打印明文密码

## 2. Agent 拉取派发（替换默认 Redis 跨进程队列）

- [ ] 2.1 Task/调度模型：可领取态、lease、claimed_by、心跳字段与迁移
- [ ] 2.2 控制面 Agent API：长轮询 claim、续约/心跳、status 上报（幂等接入 applyStatus）
- [ ] 2.3 Orchestrator：prep `job_ref` 后改为「可领取」而非默认 Publish Redis dispatch
- [ ] 2.4 租约过期回收与无在线 Edge 时保持 pending 的行为与测试
- [ ] 2.5 默认路径去掉对 Redis 的运行时依赖（旧适配器可残留但非默认）

## 3. Edge 二进制与本机/远程

- [ ] 3.1 `pixoma-edge-agent`：拉取循环 + Comfy mock/真机 + blob 读写 + status/heartbeat
- [ ] 3.2 本机 localfs 共用目录主路径（控制面 + Edge）跑通
- [ ] 3.3 远程 s3/tos 配置校验；拒绝远程 localfs
- [ ] 3.4 本机自动拉起 Edge（可配置关闭）；远程向导文案与节点登记

## 4. 向导与配置落库

- [ ] 4.1 settings 持久化（业务库）与引导态字段；启动时装配读取顺序（引导态 → settings → env 紧急覆盖）
- [ ] 4.2 向导 API：库连通、本机/远程、存储、节点/Comfy、TG Token 等步骤与校验
- [ ] 4.3 `web/admin`：未初始化向导流、登录页、完成后进入业务壳
- [ ] 4.4 「保存并重启生效」的产品行为与提示

## 5. 文档与验收

- [ ] 5.1 更新 README 部署故事与架构 `runtime`/`overview`（取消 allinone/split 用户矩阵为默认叙事）
- [ ] 5.2 验收：空目录 `pixoma` → 日志账密 → 向导本机 mock → Edge 领取 → 任务成功
- [ ] 5.3 验收：远程配置矩阵（禁 localfs）与 Edge 出站拉取说明可执行
- [ ] 5.4 Comfy mock 开关在新拓扑下仍可端到端成功

```

## docs/openspec/changes/pixoma-guided-deploy/specs/admin-api-host/spec.md

- Source: docs/openspec/changes/pixoma-guided-deploy/specs/admin-api-host/spec.md
- Lines: 1-15
- SHA256: dfe8387660afd4bfdee6d5e46bc3c6c926aa539cd78adc21d680a104eed84236

```md
## ADDED Requirements

### Requirement: 初始化后管理接口需管理员会话
平台完成初始化后，admin 管理 HTTP MUST 要求有效管理员会话（或等价凭证）；未认证请求 MUST 被拒绝。未初始化阶段 MUST 仅放行登录与向导所必需的接口（见 `platform-bootstrap` / `setup-wizard`）。

#### Scenario: 初始化后无会话访问被拒
- **WHEN** 平台已初始化，客户端未提供管理员会话调用受保护管理 API
- **THEN** 请求因未认证被拒绝

### Requirement: 控制面可一体托管管理 API
`pixoma` 控制面入口 MUST 能托管原 admin-api 的管理 HTTP 能力（可同进程），使新部署不必再单独配置并启动第二个管理进程作为默认路径。过渡期 MAY 保留独立 `admin-api` 二进制。

#### Scenario: 一体入口提供健康检查与管理 API
- **WHEN** 用户仅启动 `pixoma` 且已初始化
- **THEN** 可通过该进程提供的地址访问健康检查与管理 API

```

## docs/openspec/changes/pixoma-guided-deploy/specs/admin-web-shell/spec.md

- Source: docs/openspec/changes/pixoma-guided-deploy/specs/admin-web-shell/spec.md
- Lines: 1-15
- SHA256: 2af778077798be77a941293087c49b79ac4d31e8b1160825580bda76e170b3e5

```md
## ADDED Requirements

### Requirement: 未初始化进入向导而非业务壳
当平台未完成初始化时，管理前端 MUST 引导用户进入登录（若需要）与初始化向导，MUST NOT 将未初始化用户默认送入完整业务资源壳并假装系统已就绪。

#### Scenario: 未初始化打开控制台
- **WHEN** 用户打开管理控制台且平台未初始化
- **THEN** 界面进入初始化/向导流程，而非直接展示完整业务侧栏资源页为唯一内容

### Requirement: 初始化后需登录
平台已初始化后，管理前端 MUST 在无有效会话时展示登录页；使用向导设定（或默认后已改密）的管理员账号登录成功后方可进入业务壳。

#### Scenario: 未登录访问根路径
- **WHEN** 平台已初始化且用户无会话访问控制台根路径
- **THEN** 用户被引导至登录，而非直接进入需鉴权的业务页

```

## docs/openspec/changes/pixoma-guided-deploy/specs/agent-pull-dispatch/spec.md

- Source: docs/openspec/changes/pixoma-guided-deploy/specs/agent-pull-dispatch/spec.md
- Lines: 1-26
- SHA256: 2da140ca15de8490025b9844a125fa8e50adbf2f6f89f27e4eb817038156d960

```md
## ADDED Requirements

### Requirement: Edge 长轮询领取任务
系统 MUST 提供 Edge 可调用的控制面 Agent API：Edge MUST 能对指定 `instance_id` 发起长轮询（或等价等待）以领取待执行任务；成功领取 MUST 返回含合法 `job_ref` 的任务描述，并将该 Task 置于带租约的已领取状态。默认跨进程派发 MUST NOT 依赖 Redis Streams。

#### Scenario: 在线 Edge 领到可执行任务
- **WHEN** 控制面存在可调度到某实例的待领取任务，且该实例 Edge 正在长轮询领取
- **THEN** Edge 获得含 `job_ref` 的任务，且控制面记录领取租约与实例归属

#### Scenario: 无任务时等待后空返回
- **WHEN** 该实例暂无待领取任务，Edge 发起带等待时限的领取请求
- **THEN** 在等待时限内若仍无任务则返回空结果且不报错为系统故障

### Requirement: 租约、心跳与回收
已领取任务 MUST 具备租约到期时间；Edge MUST 能通过心跳延长租约或申报在线。租约过期且未终态时，系统 MUST 使任务重新可被领取（或按策略失败收口），MUST NOT 永久卡在已领取态。

#### Scenario: Edge 宕机后任务可再领
- **WHEN** 任务已被领取但 Edge 在租约内未续约且未上报终态
- **THEN** 租约到期后任务可被同一或其他合格实例再次领取（或进入产品定义的失败收口）

### Requirement: Edge 回报状态
Edge MUST 能通过控制面 API 上报执行状态（至少 running 与终态）；控制面 MUST 经既有幂等 status 入口更新 Task，MUST NOT 要求 Edge 连接业务库拼装工作流。

#### Scenario: 上报成功终态
- **WHEN** Edge 完成执行并上报合法 succeeded status（含产物 BlobRef）
- **THEN** Task 收敛为成功并触发既有通知意图路径

```

## docs/openspec/changes/pixoma-guided-deploy/specs/cross-process-queue/spec.md

- Source: docs/openspec/changes/pixoma-guided-deploy/specs/cross-process-queue/spec.md
- Lines: 1-30
- SHA256: 6e1ede56eb3ce60d331804965a94889f2a81e15fd7bc1a1e9056ab02ba41a0e4

```md
## MODIFIED Requirements

### Requirement: Queue 双驱动（Memory 与 Redis Streams）
系统 MUST NOT 将 Redis Streams 或跨进程 Memory Bus 作为默认部署的跨进程派发依赖。默认跨进程派发 MUST 通过 Agent 拉取（见 `agent-pull-dispatch`）完成。进程内 MAY 保留内存事件通道仅供控制面同进程编排，但 MUST NOT 作为用户可配置的 `queue.driver` 一等选项。

#### Scenario: 默认路径无 Redis 仍可派发
- **WHEN** 部署未配置 Redis，控制面已产生可领取任务且 Edge 在线拉取
- **THEN** Edge 仍能领取并执行该任务主路径

#### Scenario: 用户无需选择 queue.driver
- **WHEN** 用户通过向导完成新部署初始化
- **THEN** 向导不要求用户选择 memory/redis 作为跨进程队列驱动

### Requirement: 模式与驱动一致性校验
系统 MUST 以「本机 / 远程」校验存储与拓扑合法性，MUST NOT 再以用户必选的 `runtime_mode=allinone|split` + `queue.driver` 矩阵作为默认校验入口。远程部署 MUST 拒绝 localfs 作为跨机对象存储；本机部署 MUST 允许 localfs。

#### Scenario: 远程配置 localfs 被拒绝
- **WHEN** 部署位置为远程且对象存储驱动为 localfs
- **THEN** 校验失败并说明原因

#### Scenario: 本机 localfs 允许
- **WHEN** 部署位置为本机且对象存储为 localfs
- **THEN** 校验通过（在其它必填项满足时）

### Requirement: 业务 Topic 命名稳定
控制面同进程内编排信号（如 `task.created`）命名 MAY 保持稳定。跨进程向 Edge 的「投递」MUST 不再要求 Publish 到 `dispatch.<instance_id>` Redis Topic；改为 Agent 领取 API 下发含 `job_ref` 的任务描述。若过渡期保留旧 Redis 适配器，MUST NOT 作为默认新部署路径。

#### Scenario: 按实例领取而非 Redis Topic
- **WHEN** 调度选定实例 `gpu-1` 且任务可领取
- **THEN** 该实例 Edge 可通过领取 API 获得含 `job_ref` 的任务，而无需订阅 Redis `dispatch.gpu-1`

```

## docs/openspec/changes/pixoma-guided-deploy/specs/edge-agent/spec.md

- Source: docs/openspec/changes/pixoma-guided-deploy/specs/edge-agent/spec.md
- Lines: 1-29
- SHA256: e809f0da210f6131b67a6acb3891a4b3729ac430f71608618e32b723f45a626a

```md
## MODIFIED Requirements

### Requirement: Edge-Agent 独立进程消费 dispatch（split）
系统 MUST 提供可独立部署的 Edge-Agent 进程（产品名 `pixoma-edge-agent`）。该进程 MUST 主动连接控制面 Agent API，按 `instance_id` 长轮询领取含合法 `job_ref` 的任务，调用本机可达的 ComfyUI（Mock 或真实 HTTP），并经控制面 API 回报 `task.status`。Edge-Agent MUST NOT 连接业务 Task/Case 数据库以拼装工作流。MUST NOT 将「仅 split 模式才需要 Edge」作为产品前提——本机与远程均使用 Edge 执行面。

#### Scenario: 领取任务后执行
- **WHEN** Edge 对控制面长轮询领取成功并获得合法 `job_ref`
- **THEN** Edge 拉取任务包并完成本机执行，上报至少一次 running 与一次终态 status

### Requirement: ALLINONE 同进程执行面
系统 MUST NOT 再将「单一 OS 进程内嵌生产执行面」作为默认部署形态。本机部署 MUST 仍通过 Edge 进程（可自动拉起）执行；控制面进程 MUST NOT 在默认路径下同进程消费跨进程 dispatch 以替代 Edge。

#### Scenario: 本机亦经 Edge 执行
- **WHEN** 用户完成本机部署初始化且 Comfy mock 或本机 Comfy 可用
- **THEN** 任务由 Edge 进程领取执行，而非控制面进程内嵌 Actuator 作为唯一执行面

### Requirement: Edge/执行面仅访问本机 Comfy 与配置的 Blob/Queue
执行面 MUST 通过配置的 Blob 读写输入/产物，并通过控制面 Agent API 领取任务与回报状态。MUST NOT 要求控制面对家里 Comfy 发起入站连接。MUST NOT 要求执行面依赖 Redis 才能领取任务。

#### Scenario: 无公网 Comfy 暴露仍可完成
- **WHEN** ComfyUI 仅监听 localhost，Edge 与 Comfy 同机，控制面与（若远程）对象存储出网可达
- **THEN** 任务仍可成功执行并回写产物引用

### Requirement: Mock 路径在双模式下仍可通
当 `comfy_mock` 为真时，系统 MUST 能在不依赖真实 ComfyUI 的情况下完成「领取 → 执行 → 产物 Blob → status」主路径（本机或远程拓扑下 mock 执行面）。

#### Scenario: mock 开关开启完成主路径
- **WHEN** `comfy_mock` 为真并存在一条可被 Edge 领取的合法任务
- **THEN** 产生可读取的输出 BlobRef 且 Task 可收敛为成功

```

## docs/openspec/changes/pixoma-guided-deploy/specs/platform-bootstrap/spec.md

- Source: docs/openspec/changes/pixoma-guided-deploy/specs/platform-bootstrap/spec.md
- Lines: 1-22
- SHA256: 1fc84193d5ea94aa426f97d88ff8f9c73d30d91a7d078b599c0f90431e83a51e

```md
## ADDED Requirements

### Requirement: 零配置控制面可启动
系统 MUST 提供控制面入口二进制（产品名 `pixoma`），在无用户预先编写业务 YAML 的情况下能够启动；启动成功后 MUST 在日志中输出可打开的后台地址，以及仅用于首启的默认管理员账号与密码（或一次性初始密码）。

#### Scenario: 空目录首次启动
- **WHEN** 用户在空数据目录启动 `pixoma` 且未提供业务库配置
- **THEN** 进程就绪，日志含后台 URL 与默认管理员凭证信息

### Requirement: 引导态与未初始化门闩
系统 MUST 在本机维护引导态（bootstrap），至少包含：是否已完成初始化、管理员凭证、业务库连接信息、向导进度。未完成初始化时，管理 HTTP MUST 仅允许登录与向导相关接口；MUST NOT 对外提供完整业务管理与任务主路径能力。

#### Scenario: 未初始化拒绝业务 API
- **WHEN** 平台尚未完成初始化向导，客户端调用非向导/非登录的管理业务 API
- **THEN** 请求被拒绝或重定向到初始化流程，且不执行该业务副作用

### Requirement: 默认凭证仅首启可见
系统 MUST 仅在未初始化（或尚未完成首次改密）时于日志打印默认/初始管理员密码；初始化完成且管理员已改密后，MUST NOT 再次在常规启动日志中打印明文密码。

#### Scenario: 初始化后不再打印明文密码
- **WHEN** 初始化完成且管理员已修改密码后再次启动控制面
- **THEN** 启动日志不包含该管理员明文密码

```

## docs/openspec/changes/pixoma-guided-deploy/specs/setup-wizard/spec.md

- Source: docs/openspec/changes/pixoma-guided-deploy/specs/setup-wizard/spec.md
- Lines: 1-26
- SHA256: f05424394aa3c6542869947c3fb0564fea4c7e8b310001df14b2b07da3e281c7

```md
## ADDED Requirements

### Requirement: 初始化向导多步配置
系统 MUST 在管理后台提供初始化向导，引导用户完成至少：业务数据库、部署位置（本机或远程）、对象存储、执行节点/Comfy 指引、渠道（含 Telegram Bot Token）等步骤；每步 MUST 做可达性或合法性校验，失败时 MUST 给出可诊断错误。

#### Scenario: 本机路径完成向导
- **WHEN** 用户选择本机部署、配置可用业务库与 localfs 目录，并完成节点与渠道必要项
- **THEN** 向导可标记初始化完成，设置持久化到业务库（或引导态约定位置）

#### Scenario: 远程禁止 localfs
- **WHEN** 用户选择远程部署并尝试将对象存储选为 localfs
- **THEN** 向导拒绝该组合并说明原因

### Requirement: 配置落库而非用户手改 YAML
初始化与后续平台设置的主路径 MUST 将配置写入持久化存储（业务库 settings / 引导态），MUST NOT 要求用户手改 `bot.yaml` 才能完成新部署主路径。运维紧急覆盖（环境变量）MAY 存在，但 MUST NOT 作为向导成功的前提。

#### Scenario: 向导保存后进程可读设置
- **WHEN** 向导成功保存设置并按产品约定重启或重新加载
- **THEN** 控制面按所存设置装配 blob 与 Agent 派发行为

### Requirement: 本机与远程部署说明
选择远程时，向导 MUST 展示 `pixoma-edge-agent` 的部署要点（控制面地址、instance_id、鉴权、blob）；选择本机时，系统 MUST 支持自动拉起本机 Edge，或提供等价的一键/明确手起说明。

#### Scenario: 远程展示 Edge 部署指引
- **WHEN** 用户在向导中选择远程部署
- **THEN** 界面展示 Edge 独立进程启动与登记节点所需信息

```

## docs/openspec/changes/pixoma-guided-deploy/specs/shared-blob-store/spec.md

- Source: docs/openspec/changes/pixoma-guided-deploy/specs/shared-blob-store/spec.md
- Lines: 1-35
- SHA256: ff2a97c45fc7a597bc0fcb8a239d0acc92d975d33e00cac454e477286df7ab6e

```md
## MODIFIED Requirements

### Requirement: Blob 双驱动（localfs 与 S3 兼容）
系统 MUST 提供可配置的 `blob.Store` 后端：至少支持 localfs、S3 兼容实现，以及火山引擎 TOS。本机部署默认使用 localfs（控制面与 Edge 共用目录）；远程跨机主路径 MUST 使用 `s3` 或 `tos`，使得控制面 Put 的对象可被 Edge Get，Edge Put 的产物可被控制面通知路径 Get。逻辑 key 前缀（`inputs/`、`jobs/`、`outputs/`）MUST 一致。MUST NOT 再以 `runtime_mode=allinone|split` 作为 blob 选型文案前提。

#### Scenario: 本机使用 localfs
- **WHEN** 部署位置为本机且 blob 驱动为 localfs
- **THEN** 输入、任务包与产物均可通过同一 localfs 根读写

#### Scenario: 远程云写入 Edge 可读
- **WHEN** 控制面将对象 Put 到 S3 兼容存储的 key `inputs/<task_id>/...`
- **THEN** Edge 使用同一 key 能 Get 到相同内容

#### Scenario: 远程云写入 Edge 可读（TOS）
- **WHEN** 控制面将对象 Put 到 TOS 的 key `inputs/<task_id>/...`
- **THEN** Edge 使用同一 key 能 Get 到相同内容

#### Scenario: Edge 写入产物云可读
- **WHEN** Edge 将产物 Put 到 `outputs/<task_id>/...`
- **THEN** Bot 通知路径能 Get 该对象并投递给用户

### Requirement: 模式与驱动一致性校验（blob 维）
系统 MUST 在保存设置或启动时校验部署位置与 blob 驱动组合合法；非法组合 MUST 失败并给出可诊断错误。远程下 blob 驱动 MUST 为 `s3` 或 `tos`。MUST NOT 要求与 `queue.driver=redis` 搭配作为合法前提。

#### Scenario: 远程允许 TOS
- **WHEN** 部署位置为远程、blob 驱动为 tos
- **THEN** 校验通过（在其它必填项满足时）

#### Scenario: 远程允许 S3
- **WHEN** 部署位置为远程、blob 驱动为 s3
- **THEN** 校验通过（在其它必填项满足时）

#### Scenario: 远程拒绝 localfs
- **WHEN** 部署位置为远程且 blob 驱动为 localfs
- **THEN** 校验失败并说明原因

```

## docs/openspec/changes/pixoma-guided-deploy/specs/task-orchestrator/spec.md

- Source: docs/openspec/changes/pixoma-guided-deploy/specs/task-orchestrator/spec.md
- Lines: 1-31
- SHA256: 1b4eda50223cc5945fb3d06dccd97439906434b70c134c68c4ab47a7fbcbfba3

```md
## MODIFIED Requirements

### Requirement: 调度与 pending 扫描双触发
系统 MUST 支持事件触发调度，并 MUST 提供定时扫描 `pending` 的兜底。本期调度 MUST 按实例选择目标（健康/熔断/在线 Edge 等条件），在可选集合上使用轮询（round-robin）分配，将 Task 置为可被该实例领取的状态（并准备含 `job_ref` 的任务包），MUST NOT 默认向 Redis `dispatch.<instance_id>` Publish 作为唯一派发手段。当没有可投递实例/在线 Edge 时 MUST 保持 Task 为 `pending` 并记录原因，MUST NOT 假装已排队。MUST NOT 假设控制面一定能直连家里 ComfyUI。

#### Scenario: 创建事件丢失仍被扫到
- **WHEN** Task 长期处于 pending 且无成功调度
- **THEN** SchedulePending 仍能将其变为可领取（若存在可投递实例/在线 Edge）

#### Scenario: 多健康实例轮询
- **WHEN** 存在至少两台健康且未熔断的可投递实例，连续调度多个 pending Task
- **THEN** 连续选定的 InstanceID 在轮询意义上分散到这些实例，而非每次都是同一台

#### Scenario: 按实例可领取
- **WHEN** 选定实例 `gpu-1` 且可投递
- **THEN** 任务对该实例可领取且含有效 `job_ref`，且 Blob 中存在可读任务包

#### Scenario: 全部不可用时不投递
- **WHEN** 没有任何健康且未熔断的可投递实例（或无在线 Edge）
- **THEN** Task 仍保持 pending（或未被标记为可领取），且不向任意实例交付任务

#### Scenario: 无可投递目标时不投递
- **WHEN** 无健康可投递实例（或无在线 Edge）
- **THEN** Task 仍保持 pending，且不假装成功投递

### Requirement: 投递前组装 job 并携带 job_ref
Orchestrator（或其 prep 步骤）在使任务可被 Edge 领取之前 MUST 组装方案 A 任务包并获得 `job_ref`。领取下发的任务描述 MUST 包含该 `job_ref`。成功主路径 MUST NOT 再依赖执行面读取业务 Case/Task 库拼装 workflow。该要求在本机与远程部署下均生效。

#### Scenario: 可领取任务含 job_ref
- **WHEN** 某 pending Task 被成功调度为可领取
- **THEN** 对应领取载荷含有效 `job_ref`，且 Blob 中存在可读任务包

```
