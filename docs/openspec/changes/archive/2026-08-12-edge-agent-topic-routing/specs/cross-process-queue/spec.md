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
