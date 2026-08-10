---
comet_change: edge-agent-topic-routing
role: technical-design
canonical_spec: openspec
---

# Edge-Agent 双模式 + 方案 A（Topic 分流延后）

## 1. 目标与范围

### 1.1 目标

- 执行面可外置为 Edge（家里/内网 Comfy，云不直连、Comfy 不公网暴露）。
- **方案 A**：云上 prep 写任务包到 Blob，`dispatch` 携带 `job_ref`；执行面不连业务 DB。
- **双模式**：
  - `allinone`：单 OS 进程组装 Bot + 调度 + 执行面；MQ = Memory；Blob = localfs。
  - `split`：云控制面 + 独立 Edge 进程；MQ = Redis Streams；Blob = S3 兼容。
- 两种模式共用同一套 prep / `job_ref` / 执行语义，仅替换 Queue、Blob 适配器与进程边界。

### 1.2 非目标（本期）

- Topic 目录管理、投放表达式、会员等级 × Case 分类分流（延后）。
- 真实会员体系（`member_level` 占位若出现，仅默认配置，不作为本期分流验收）。
- Edge 直连业务库拼装（方案 C）。
- running 任务强取消（Comfy interrupt）。

### 1.3 与 Open 阶段差异

Open 曾纳入 Topic 分流；Design 确认后**本期不做**。delta spec 已 Spec Patch。change 目录名保持 `edge-agent-topic-routing`，文档与 tasks 以本 Design Doc 为准。

## 2. 模式与配置

### 2.1 模式枚举

建议配置键（实现可微调命名，须文档化）：

```yaml
runtime_mode: allinone | split   # 或 deploy_mode
queue:
  driver: memory | redis         # allinone 强制 memory；split 强制 redis（可启动校验）
blob:
  driver: localfs | s3           # allinone 默认 localfs；split 默认 s3
comfy_mock: true | false
```

启动校验：

| 模式 | 允许的 queue | 允许的 blob | 执行面 |
|---|---|---|---|
| allinone | memory | localfs（主路径） | 同进程订阅 dispatch |
| split | redis | s3（跨机主路径） | 独立 Edge；云侧默认不订阅生产 dispatch |

误配（如 split + memory、allinone 多进程指望 Memory 互通）MUST 启动失败或明确拒绝就绪。

### 2.2 ALLINONE

- 单一入口进程内 wiring：ConfirmRun、Orchestrator、prep、执行面（现有 Actuator Worker）、notify。
- Memory Bus + localfs；`comfy_mock` 可走 Mock Comfy。
- 仍执行方案 A：prep 写 `jobs/<task_id>/job.json`（localfs），dispatch 带 `job_ref`。

### 2.3 split

- 云：Bot/调度/prep/Publish；不跑生产执行面（或仅管理/观测）。
- Edge：配置 `instance_id`、本机 Comfy URL、Redis、S3、订阅 `dispatch.<instance_id>`（或实例记录中的 DispatchTopic）。
- 心跳/在线信号：供调度判断该实例 Topic 是否可投递（最小：周期性向控制面或约定 Redis key 报活；细节实现可选 DB 字段或 Redis TTL）。

## 3. 数据流

```text
ConfirmRun
  → Blob.Put inputs/<task_id>/...
  → Task pending + task.created

Orchestrator
  → 选健康可投递实例（本期：实例维度，非表达式）
  → prep: Case+staged → job.json → Blob.Put jobs/<task_id>/job.json
  → ClaimQueued + Publish dispatch.<instance_id> { task_id, instance_id, job_ref }
  → （不再依赖执行面读 Case/Task DB 拼装）

执行面（同进程或 Edge）
  → Get job_ref
  → 对 images[]: Get 图 → 本机 UploadImage → 写节点
  → Submit / Wait
  → Blob.Put outputs/<task_id>/...
  → task.status { outputs: BlobRef[] }

Orchestrator
  → 写 Task 终态 → UserNotify → TG 从 Blob.Get 发图
```

### 3.1 任务包（job）最小形状

```json
{
  "task_id": "...",
  "instance_id": "...",
  "workflow": { },
  "images": [
    { "node_id": "...", "field_path": "...", "blob": { "key": "inputs/..." } }
  ],
  "output_prefix": "outputs/<task_id>"
}
```

图片字节不进 MQ、不进 job 正文。

### 3.2 DispatchCommand（BREAKING）

```json
{
  "task_id": "...",
  "instance_id": "...",
  "job_ref": { "key": "jobs/<task_id>/job.json", "mime": "application/json" }
}
```

`input_prefix` 可保留过渡期只读，成功主路径 MUST NOT 依赖其回查 Case 库。

## 4. 适配器设计

### 4.1 Queue：Memory | Redis Streams

- 端口不变：`Publish` / `Subscribe`。
- Redis：每个业务 Topic 映射到 Stream key（例如 `q:dispatch.<instance_id>`）；消费者组按实例/Edge 稳定命名；成功处理再 Ack；失败可重试与死信策略实现阶段定最小可用。
- `task.created` / `task.status` 同样经适配器；allinone 下 Memory 同步语义可保留（注意 ConfirmRun 返回前可能跑完链路的现有行为）。

### 4.2 Blob：localfs | S3

- 端口不变：`Put` / `Get` + `BlobRef`。
- S3：bucket + key 前缀；凭证来自配置/环境。
- 前缀约定：`inputs/`、`jobs/`、`outputs/`。

### 4.3 Prep

- 从现有 `CaseSnapshot` 拆分：云侧完成非图片注入 + 图片清单；**禁止**在 prep 中调用家里 Comfy `UploadImage`。
- 执行面只做图片 Upload + Submit/Wait + 产物 Put。

## 5. 进程与包布局（建议）

- 保持 `internal/runtime/infrastructure/actuator` 为执行核心库。
- 新增 Edge 入口（如 `apps/edge-agent/...`）注入 Redis、S3、本机 Comfy、订阅配置。
- ALLINONE 入口继续 `apps/bot`（或统一 packaging）同进程挂执行面。
- Admin 进程：ALLINONE 可不强制同进程；若「全家桶」仅指 bot 运行时，admin-api 可仍独立——本期不要求 admin 与 bot 同进程，除非已有一键脚本需要列出。

## 6. 健康与调度（本期无 Topic 表达式）

- 选路：健康实例集合 + round-robin（或既有熔断），Publish `dispatch.<instance_id>`。
- split：以 Edge 在线为准，不以云直连家里 Comfy 为唯一健康信号。
- allinone：同进程执行面即消费者；探活可继续用 Mock/本地 HTTP。

## 7. 测试策略

| 层级 | 内容 |
|---|---|
| 单测 | prep 产出 job；执行面读 job；DispatchCommand 序列化 |
| 适配 | Redis Streams 跨 goroutine/进程；S3 或 minio testcontainer/本地；localfs 回归 |
| E2E | allinone + `comfy_mock`：确认生成 → 收图 |
| 冒烟 | split：Redis+S3+Edge+mock/真 Comfy（可选 CI） |
| 负向 | 坏 job_ref；模式误配启动失败；无在线 Edge 时 Task 保持 pending |

## 8. 迁移与开关

1. 落地双 Blob/双 Queue 驱动与启动校验。
2. 上线 prep + `job_ref`；执行面改读 job（allinone 先切）。
3. 发布 Edge；split 配置切换；云侧关闭生产同进程 dispatch 订阅。
4. 回滚：`runtime_mode=allinone` + memory + localfs + mock。

## 9. 延后工作（明确不在本期验收）

- Topic CRUD、投放规则、试算、会员等级真实数据、按 categories 分流。
- 相关 OpenSpec 能力保留目录时可标「延后」，tasks 中 Topic Admin 组取消或移入「延后」。
