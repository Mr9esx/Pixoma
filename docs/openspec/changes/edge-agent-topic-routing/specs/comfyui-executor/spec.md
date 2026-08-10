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
