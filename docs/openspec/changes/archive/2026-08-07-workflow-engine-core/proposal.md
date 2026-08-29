## Why

项目要做成可部署的 ComfyUI Telegram Bot 服务。本期落地可组装的工作流核心：Case 协议与注册、私聊 Dialog Session、异步 Task，以及 Bot（消息平台）/ Orchestrator（控制面）/ Actuator（执行面）模块；Queue/Blob 端口化，便于同进程组装或后续拆分。

## What Changes

- 新增 **Workflow Case 协议**：元数据、有序 input/output、**JSON Schema 校验**（含媒体扩展）、Comfy 绑定、价格数值、可扩展 tags。
- 新增 **工作流注册与仓储**：GORM + SQLite，预留 MySQL。
- 新增 **Dialog Session**：按 `chat_id` 填表锁；浏览不上锁；未 Exit/提交不得开新 Case；生成中 Task 不挡新 Case。
- 新增 **异步 Task + Orchestrator + Actuator**：ConfirmRun 后事件驱动；Task 终态仅 Orchestrator 写库；Actuator 发 status 并对账 Query；风暴防护与温和取消。
- 新增 **TG Adapter**：菜单/分类/Case/填表/拦截/结果通知（Application DTO → TG 消息）。
- **本期不做**：真实积分扣费、多租户部署平台、强取消（interrupt running）、生产级多云 MQ/OSS（保留端口）。

## Capabilities

### New Capabilities

- `workflow-protocol`: Case 协议与 JSON Schema 校验契约。
- `workflow-registry`: Case 注册与持久化。
- `dialog-session`: 私聊填表会话与锁。
- `task-orchestrator`: Task 生命周期、调度、对账、notify 驱动、风暴防护。
- `comfyui-executor`: Actuator 执行面（注入、ComfyUI、status、本地 ledger、Query）。
- `channel-tg`: Telegram 消息平台适配与通知投递。

### Modified Capabilities

- （无既有主规格）

## Impact

- Go 模块化工程；GORM/SQLite；`go-telegram/bot`；JSON Schema 库；Memory Queue + LocalFS。
- 外部：Telegram Bot API、ComfyUI HTTP API。
