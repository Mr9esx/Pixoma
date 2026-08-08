## MODIFIED Requirements

### Requirement: Actuator 作为执行面调用 ComfyUI
系统 MUST 提供 Actuator 模块：消费指向本实例的 dispatch 命令，按 Case 快照中的绑定将输入注入工作流图，调用配置的 ComfyUI 客户端（Mock 或真实 HTTP，由配置开关选择），并等待完成或失败。Actuator MUST NOT 使用与 Case 无关的空 stub 图作为成功主路径的默认行为；MUST NOT 直接更新全局 Task 表的终态；MUST 通过 status 事件向 Orchestrator 上报。

#### Scenario: 收到 dispatch 后执行文生图
- **WHEN** Actuator 收到合法 dispatch 且 ComfyUI 客户端可用，且 Case 含有效 workflow 与注入后的图
- **THEN** 系统向 ComfyUI 提交工作流，并上报至少一次 running 与一次终态 status（succeeded 或 failed）

#### Scenario: ComfyUI 不可达时上报失败 status
- **WHEN** ComfyUI 实例不可达或返回连接错误（真实 HTTP 模式）
- **THEN** Actuator 上报 failed status（含可诊断信息），且不假装成功

#### Scenario: Mock 开关开启时仍完成主路径
- **WHEN** `comfy_mock`（或等价环境变量）为真
- **THEN** 同一注入与 status 路径可成功完成并产生可投递图片产物

### Requirement: 按绑定映射注入输入
系统 MUST 依据 case 中声明的节点/字段注入映射，将逻辑 input 键映射到 ComfyUI 工作流图中的具体节点入参。文本 MUST 注入为对应字段值；图片 MUST 基于已物化 Blob，经客户端要求的上传/引用步骤后注入。缺少映射、映射目标不存在、或必填物化输入缺失时，系统 MUST 在调用 ComfyUI 前失败并上报 failed status。

#### Scenario: 文本注入到指定节点
- **WHEN** case 将 `prompt` 映射到某节点的文本字段，且 staged 输入含该文本
- **THEN** 提交给 ComfyUI 的工作流中该节点字段等于已物化输入中的 prompt 值

#### Scenario: 图片注入到指定节点
- **WHEN** case 将 `reference`（image）映射到某节点图片字段，且 staged 输入含对应 Blob
- **THEN** 提交前完成上传或等价引用，且工作流中该字段指向可用的图片输入

#### Scenario: 映射缺失导致拒绝执行
- **WHEN** 某必填 input 没有有效的 ComfyUI 注入映射
- **THEN** 不调用 ComfyUI，并上报指出映射问题的 failed status

#### Scenario: 空 workflow 拒绝执行
- **WHEN** Case 快照中 `bindings.workflow` 为空或不含可提交图
- **THEN** 不调用 ComfyUI，并上报失败 status
