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
