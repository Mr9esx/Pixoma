# edge-agent Specification

## Purpose
TBD - created by archiving change edge-agent-topic-routing. Update Purpose after archive.
## Requirements
### Requirement: Edge-Agent 独立进程消费 dispatch（split）
系统 MUST 提供可独立部署的 Edge-Agent 进程（产品名 `pixoma-edge-agent`）。该进程 MUST 主动连接控制面 Agent API，按 `instance_id` 长轮询领取含合法 `job_ref` 的任务，调用本机可达的 ComfyUI（Mock 或真实 HTTP），并经控制面 API 回报 `task.status`。Edge-Agent MUST NOT 连接业务 Task/Case 数据库以拼装工作流。MUST NOT 将「仅 split 模式才需要 Edge」作为产品前提——本机与远程均使用 Edge 执行面。

#### Scenario: 订阅实例 Topic 后执行任务
- **WHEN** Edge 对控制面长轮询领取成功并获得合法 `job_ref`
- **THEN** Edge 拉取任务包并完成本机执行，上报至少一次 running 与一次终态 status

#### Scenario: 领取任务后执行
- **WHEN** Edge 对控制面长轮询领取成功并获得合法 `job_ref`
- **THEN** Edge 拉取任务包并完成本机执行，上报至少一次 running 与一次终态 status

### Requirement: ALLINONE 同进程执行面
系统 MUST NOT 再将「单一 OS 进程内嵌生产执行面」作为默认部署形态。本机部署 MUST 仍通过 Edge 进程（可自动拉起）执行；控制面进程 MUST NOT 在默认路径下同进程消费跨进程 dispatch 以替代 Edge。

#### Scenario: allinone 单进程完成主路径
- **WHEN** 用户完成本机部署初始化且 Comfy mock 或本机 Comfy 可用
- **THEN** 任务由 Edge 进程领取执行并完成主路径，而非控制面进程内嵌 Actuator 作为唯一执行面

#### Scenario: 本机亦经 Edge 执行
- **WHEN** 用户完成本机部署初始化且 Comfy mock 或本机 Comfy 可用
- **THEN** 任务由 Edge 进程领取执行，而非控制面进程内嵌 Actuator 作为唯一执行面

### Requirement: Edge/执行面仅访问本机 Comfy 与配置的 Blob/Queue
执行面 MUST 通过配置的 Blob 读写输入/产物，并通过控制面 Agent API 领取任务与回报状态。MUST NOT 要求控制面对家里 Comfy 发起入站连接。MUST NOT 要求执行面依赖 Redis 才能领取任务。

#### Scenario: split 无公网 Comfy 暴露仍可完成
- **WHEN** ComfyUI 仅监听 localhost，Edge 与 Comfy 同机，控制面与（若远程）对象存储出网可达
- **THEN** 任务仍可成功执行并回写产物引用

#### Scenario: 无公网 Comfy 暴露仍可完成
- **WHEN** ComfyUI 仅监听 localhost，Edge 与 Comfy 同机，控制面与（若远程）对象存储出网可达
- **THEN** 任务仍可成功执行并回写产物引用

### Requirement: Mock 路径在双模式下仍可通
当 `comfy_mock` 为真时，系统 MUST 能在不依赖真实 ComfyUI 的情况下完成「领取 → 执行 → 产物 Blob → status」主路径（本机或远程拓扑下 mock 执行面）。

#### Scenario: mock 开关开启完成主路径
- **WHEN** `comfy_mock` 为真并存在一条可被 Edge 领取的合法任务
- **THEN** 产生可读取的输出 BlobRef 且 Task 可收敛为成功

### Requirement: Edge 订阅 Topic 配置
Edge-Agent MUST 支持配置订阅 Topic 列表（如 `subscribe_topics`）；未配置时 MUST 默认订阅 `default` Topic。进程启动后 MUST 按实际订阅集合向控制面申报并长轮询领取。Edge MUST NOT 解析投放表达式（规则求值只发生在控制面）。

#### Scenario: 未配置订阅默认
- **WHEN** Edge 配置未声明 subscribe_topics
- **THEN** Edge 以 `default` 为订阅集合领取任务

#### Scenario: 多 Topic 订阅
- **WHEN** Edge 声明订阅 `["default","fast-gpu"]`
- **THEN** Edge 可领取这两个 Topic 下投递的任务，不领取其它 Topic 的任务

