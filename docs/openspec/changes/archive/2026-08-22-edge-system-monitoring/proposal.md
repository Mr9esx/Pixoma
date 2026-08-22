## Why

节点详情页的「系统」观测目前由控制面反向连接 Edge 本机 ComfyUI 的 `/system_stats` 取数。Edge 部署在内网时控制面连不上，页面长期显示「连接被拒绝」。需要改成 Edge 在自己心跳（presence 上报）时主动携带实时系统指标，控制面记录这些信息，页面从服务端数据画图，彻底摆脱对 Edge 入站连接的依赖。

## What Changes

- 节点详情页「系统」节文案改为「系统监控」。
- Edge-Agent 新增实时指标采集：CPU 占用率、内存占用（已用字节）、内存占用率、GPU 占用率、GPU 显存占用（已用字节）、GPU 显存占用率、磁盘 I/O 信息（读/写速率）。采集结果随 presence 心跳上报到 `/agent/v1/presence`。
- 控制面持久化指标：新增 `edge_metrics` 记录每次上报的指标快照，并提供管理端查询接口 `GET /api/v1/edges/{id}/metrics`。
- Admin 节点详情页「系统监控」节改用 MHTML（Shadcnblocks Admin Kit）里的 chart 组件与样式：同一套 shadcn `ChartContainer` 组件（recharts + Tailwind v4，`data-slot="chart"` 与同款 class 串），卡片、图例、渐变、配色与参考页一致，不只是抄布局。
- 管理端系统节不再直连 Edge 取数；「连接被拒绝」类错误不再出现在该节。

## Capabilities

### New Capabilities
- `edge-metrics`: Edge 心跳上报实时系统指标、控制面持久化并提供查询 API、Admin 节点详情页以参考页同款 chart 组件展示系统监控图表。

### Modified Capabilities
<!-- 本次为新增能力，不修改既有 spec 级行为。 -->

## 范围与非目标

范围：Edge-Agent 指标采集与心跳上报、控制面 `edge_metrics` 持久化与查询接口、节点详情页「系统监控」图表（CPU / 内存 / GPU / I/O）。不修改「队列」与「最近任务」节、不修改右列静态机器规格语义。

非目标：不做实时推送（WebSocket/SSE）；不引入 NVML / ROCm（GPU 占用率用 `nvidia-smi`，无 NVIDIA 时字段留空）；不改调度、健康检查与任务执行路径；不做多节点聚合或全局监控页。

## Impact

- `apps/edge-agent`：新增 `internal/metrics` 采集器（CPU/内存/GPU/磁盘 I/O），presence 载荷扩展 `metrics` 字段；新增 `gopsutil` 依赖（GPU 占用率用 `nvidia-smi` 查询，不引入 NVML）。
- 控制面：`internal/platform` 新增指标领域与 `edge_metrics` 持久化（AutoMigrate）；`internal/httpapi/agent` 的 presence 处理接收并落库指标；`internal/httpapi/edges` 新增 metrics 查询端点。
- Admin 前端：新增 `web/admin/src/components/ui/chart.tsx`（shadcn chart 组件）；`features/edges` 的系统监控图表、i18n 文案、API 类型与合同测试。
- 文档：`docs/architecture/data-model.md`、`runtime.md` 等同步 `edge_metrics` 与上报路径。
