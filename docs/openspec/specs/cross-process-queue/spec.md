# cross-process-queue Specification

## Purpose
TBD - created by archiving change edge-agent-topic-routing. Update Purpose after archive.
## Requirements
### Requirement: Queue 双驱动（Memory 与 Redis Streams）
系统 MUST NOT 将 Redis Streams 或跨进程 Memory Bus 作为默认部署的跨进程派发依赖。默认跨进程派发 MUST 通过 Agent 拉取（见 `agent-pull-dispatch`）完成。进程内 MAY 保留内存事件通道仅供控制面同进程编排，但 MUST NOT 作为用户可配置的 `queue.driver` 一等选项。

#### Scenario: allinone Memory 闭环
- **WHEN** 控制面同进程内编排需要事件通道（如 `task.created`）
- **THEN** 同进程内 Publish/Subscribe MAY 完成控制面闭环，且 MUST NOT 作为用户可配置的跨进程 queue.driver

#### Scenario: split 跨进程收到 dispatch
- **WHEN** 控制面已产生可领取任务且对应实例的 Edge 已连接并拉取
- **THEN** Edge 通过领取 API 收到等价任务描述，而无需订阅 Redis Stream

#### Scenario: 默认路径无 Redis 仍可派发
- **WHEN** 部署未配置 Redis，控制面已产生可领取任务且 Edge 在线拉取
- **THEN** Edge 仍能领取并执行该任务主路径

#### Scenario: 用户无需选择 queue.driver
- **WHEN** 用户通过向导完成新部署初始化
- **THEN** 向导不要求用户选择 memory/redis 作为跨进程队列驱动

### Requirement: 模式与驱动一致性校验
系统 MUST 以「本机 / 远程」校验存储与拓扑合法性，MUST NOT 再以用户必选的 `runtime_mode=allinone|split` + `queue.driver` 矩阵作为默认校验入口。远程部署 MUST 拒绝 localfs 作为跨机对象存储；本机部署 MUST 允许 localfs。

#### Scenario: split 配置 Memory 被拒绝
- **WHEN** 新部署尝试把跨进程派发配成仅进程内 Memory
- **THEN** 该配置 MUST NOT 作为合法默认路径；跨进程派发 MUST 走 Agent 拉取

#### Scenario: 远程配置 localfs 被拒绝
- **WHEN** 部署位置为远程且对象存储驱动为 localfs
- **THEN** 校验失败并说明原因

#### Scenario: 本机 localfs 允许
- **WHEN** 部署位置为本机且对象存储为 localfs
- **THEN** 校验通过（在其它必填项满足时）

### Requirement: 业务 Topic 命名稳定
控制面同进程内编排信号（如 `task.created`）命名 MAY 保持稳定。跨进程向 Edge 的「投递」MUST 不再要求 Publish 到 `dispatch.<instance_id>` Redis Topic；改为 Agent 领取 API 下发含 `job_ref` 的任务描述。若过渡期保留旧 Redis 适配器，MUST NOT 作为默认新部署路径。

#### Scenario: 按实例 Topic 投递
- **WHEN** 调度选定实例 `gpu-1` 且任务可领取
- **THEN** 该实例 Edge 可通过领取 API 获得含 `job_ref` 的任务，而无需订阅 Redis `dispatch.gpu-1`

#### Scenario: 按实例领取而非 Redis Topic
- **WHEN** 调度选定实例 `gpu-1` 且任务可领取
- **THEN** 该实例 Edge 可通过领取 API 获得含 `job_ref` 的任务，而无需订阅 Redis `dispatch.gpu-1`

