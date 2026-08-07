# comfyui-executor Specification

## Purpose
TBD - created by archiving change workflow-engine-core. Update Purpose after archive.
## Requirements
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

