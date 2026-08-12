## Why

家里/内网的 ComfyUI 无法被云上 Bot、Admin 直连，且 Comfy 无现成鉴权、不宜对公网暴露。需要把执行面外置为 Edge-Agent（主动出网消费 MQ、只调本机 Comfy），并用 Topic + 投放表达式按会员等级与 Case 分类把任务分到不同机器。

本 change 刻意保持为单一需求包（用户明确不拆 batch）：共享存储、跨进程队列、方案 A 任务包、Edge 进程、Topic 管理与表达式路由一并纳入，以便端到端闭环。

## What Changes

- 新增 **Edge-Agent** 独立进程：按配置订阅一个或多个 dispatch Topic；拉取任务包与输入图；本机 Upload/Submit/Wait；产物写回共享 Blob；发布 `task.status`。
- **方案 A 任务包**：云上调度/prep 在发 dispatch 前组装半成品（已注入非图片字段的 workflow + 图片 BlobRef 列表），写入共享 Blob；`DispatchCommand` **BREAKING** 增加 `job_ref`（可废弃 Edge 侧对业务 DB 的依赖）。
- Blob 从「仅 Bot 本机 localfs 可用」升级为 **云与 Edge 均可 Put/Get 的共享对象存储**（接口仍为 `blob.Store`；具体后端在 design 选定）。
- Queue 从进程内 Memory 扩展为 **可跨进程的 MQ 适配器**（保留 Port；Memory 可继续服务单机/mock）。
- 后台支持 **Topic 管理**与 **投放表达式**（输入维度含用户会员等级如 Lv1/Lv2、Case 分类如图片/视频等）；调度器按表达式选择 dispatch Topic。
- Edge 配置声明自身订阅的 Topic 列表；与后台 Topic 目录对齐。
- **Mock 同演进**：`comfy_mock` 路径仍须端到端可通（可用进程内或本地 MQ + mock Comfy/Edge 替身）。

## Capabilities

### New Capabilities

- `edge-agent`: Edge 进程职责、配置（含订阅 Topic）、本机 Comfy 调用、状态回传与鉴权边界（不连业务 DB）。
- `dispatch-job-package`: 方案 A 任务包格式、云上 prep、`job_ref` 与 Edge 消费约定。
- `shared-blob-store`: 跨进程/跨主机共享 Blob 的行为与可达性要求。
- `cross-process-queue`: 跨进程 Publish/Subscribe 语义与 topic 命名约定。
- `dispatch-topic-routing`: Topic 目录、投放表达式求值、调度选 Topic、与 Edge 订阅的匹配关系。
- `topic-admin`: 后台对 Topic 与投放表达式的管理能力（API/控制台范围在 design 收束）。

### Modified Capabilities

- `task-orchestrator`: 认领后按表达式选 Topic 再 Publish dispatch；与 prep/job_ref 衔接。
- `comfyui-executor`: 执行面从同进程 Actuator 迁移/对接到 Edge 消费模型（同进程可保留为开发/mock 形态）。
- `comfy-instance-pool`: 实例与 Topic/Edge 的关联方式调整（不再假设云侧直连探活 Comfy 为唯一健康来源；可演进为 Edge 心跳等，细节在 design）。

## Impact

- 进程拓扑：Bot/调度（云）与 Edge（Comfy 侧）分离；Comfy 仅本机可达。
- 事件：`dispatch.*` 载荷与 topic 命名；可能新增 Topic 实体与表达式配置持久化。
- 依赖：对象存储、跨进程 MQ；Admin API/Web 增加 Topic/表达式管理。
- 风险：表达式错误导致误投；Blob/MQ 未就绪时 Edge 无法工作；需明确 mock 与真实路径开关。
