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

#### Scenario: 不维护独立 Ledger 真相源
- **WHEN** Actuator 执行任务并上报 status（含 prompt_id / 终态）
- **THEN** 系统不以 MemoryLedger/LocalRun 作为可恢复执行态的唯一或权威存储；可恢复状态落在 Task（经 Orchestrator 写回）
