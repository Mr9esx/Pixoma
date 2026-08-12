## Context

当前 Bot 进程内同时包含 Orchestrator（调度）与 Actuator（执行），Queue 为进程内 Memory，Blob 为 Bot 本机 `localfs`。调度按健康实例 round-robin，向 `dispatch.<instance_id>` 投递瘦载荷 `{task_id, instance_id, input_prefix}`；Actuator 同进程读 DB+Blob 拼图并直连 Comfy HTTP。

约束：家里/内网 Comfy 不可被云直连；Comfy 不宜对公网暴露；大文件不进 MQ；`comfy_mock` 须继续端到端可通；用户确认本 change 不拆 batch。

已锁定：**方案 A**（云上 prep 半成品 + `job_ref`）；Edge = 外置 Actuator；Topic 由后台管理；投放表达式基于会员等级与 Case 分类；Edge 配置订阅 Topic。

## Goals / Non-Goals

**Goals:**

- 云侧 prep 写出可被 Edge 消费的任务包到共享 Blob；`dispatch` 携带 `job_ref` 与目标 Topic。
- Edge 独立进程：订阅配置的 Topic → 拉 job/图 → 本机 Comfy → 产物回写 Blob → `task.status`。
- 共享 Blob、跨进程 MQ 适配器落地（Port 不变，换实现/配置）。
- Topic 目录 + 投放表达式；调度按表达式选 Topic；Admin 可管理。
- Mock/单机开发路径仍可闭环。

**Non-Goals:**

- Edge 直连业务 Postgres 拼装（方案 C）。
- 本期强取消 running（Comfy interrupt）不变。
- 不在本期绑定某一云厂商专有服务为唯一实现（设计选定默认开源/可自托管组合即可）。
- 不把完整会员计费/支付做进本 change（只消费已有或可扩展的会员等级字段）。

## Decisions

### D1. 执行面外置为 Edge-Agent（复用 Actuator 语义）

- **选择**：独立二进制/进程承载今日 `actuator.Worker` 的执行职责；云 Bot 默认不再同进程订阅生产 dispatch（开发/mock 可同机嵌入）。
- **备选**：反向隧道把家里 Comfy 暴露给云 — 拒绝（鉴权与暴露面差）。
- **理由**：与既有「Actuator 跟实例走」模型一致，只改部署与材料获取。

### D2. 方案 A：`job_ref` + Blob 内任务包

- **选择**：prep 在 Publish dispatch 前写入例如 `jobs/<task_id>/job.json`；`DispatchCommand` 增加 `job_ref`（`BlobRef`）。任务包含：已注入非图片字段的 `workflow`、`images[]`（node/field + 输入 BlobRef）、`output_prefix` 等。
- **备选**：整包塞 MQ；Edge 读 DB — 拒绝。
- **理由**：MQ 保持小；Edge 不碰业务库。

### D3. 大文件通道：共享对象存储（`blob.Store`）

- **选择**：引入可跨主机的 Blob 后端（默认倾向 S3 兼容如 MinIO；接口仍 Put/Get）。Edge 与云使用同一逻辑 bucket/前缀约定。
- **备选**：仅自研 HTTP 传文件 API — 可作为适配器，但语义仍是对象存储。
- **理由**：与现有 BlobRef 模型一致。

### D4. 跨进程 Queue

- **选择**：保留 `queue.Publisher`/`Subscriber`；新增至少一种跨进程适配器（具体中间件在实现前在 Open Questions 收口，候选 NATS / Redis Streams）。Topic 字符串与业务名对齐：`task.created`、`task.status`、以及可配置的 `dispatch.<topic_key>`。
- **备选**：继续仅 Memory — 无法支撑家里 Edge。
- **理由**：端口已存在，二期换适配器的原设计意图。

### D5. Topic 与投放表达式

- **选择**：
  - **Topic**：后台可 CRUD 的逻辑投递目标（稳定 `key`、显示名、启用状态等）；Edge 配置 `subscribe_topics: [key, ...]`。
  - **表达式**：对「用户会员等级 + Case 分类（图片/视频等）」求值，产出目标 Topic key。支持类似「Lv1 → topicA」「Lv2 AND category=image → topicB」的规则集；求值失败走显式默认 Topic 或保持 pending 并记原因（实现选一种并在 spec 写死）。
  - **调度**：Claim 前/时求值 Topic；Publish 到该 Topic；Task 记录实际 `dispatch_topic`（及可选 instance/edge 标识）。
- **备选**：仅 `dispatch.<instance_id>` — 无法表达会员×分类策略。
- **理由**：与用户描述一致；Edge 只订配置 Topic，不解析表达式。

### D6. 健康与选路

- **选择**：生产路径以 Edge **心跳/在线**（或「该 Topic 有活跃消费者」信号）作为可投递条件；云侧对家里 Comfy 的 HTTP 探活降为可选/仅局域网。`comfy_mock` 仍可用进程内 Mock 执行面。
- **备选**：继续云直连探 Comfy — 对家里机器不可行。

### D7. Admin

- **选择**：Admin API（及必要 Web 页）管理 Topic 与投放规则；实例/Edge 注册信息可关联默认订阅 Topic（细节实现阶段与现有 `comfy-instance-*` 对齐或新增 edge 资源）。

## Risks / Trade-offs

- [表达式误配导致全员进错池] → 管理端校验、默认 Topic、审计日志、干跑/试算 API（可分期）。
- [Blob/MQ 未就绪 Edge 饿死] → 启动自检；Task 保持 pending；可观测告警。
- [job 与输入图生命周期] → 约定 TTL/前缀清理策略；失败重试幂等。
- [单 change 范围大] → tasks 按里程碑切片；用户明确要求不拆 OpenSpec change。
- [BREAKING dispatch 载荷] → 版本字段或双读过渡期；mock/单机路径同步改。

## Migration Plan

1. 落地共享 Blob + 跨进程 MQ，双写/开关切换。
2. 上线 prep + `job_ref`；旧瘦 dispatch 过渡或开关关闭。
3. 部署 Edge；云侧停止生产路径同进程 Actuator 订阅。
4. 导入 Topic 与表达式；切调度选路。
5. 回滚：开关回到 Memory+同进程 Actuator+直连（仅适用于仍能直连 Comfy 的环境）。

## Open Questions

1. 跨进程 MQ 默认选型（NATS vs Redis Streams 等）。
2. 表达式语法：自研 JSON 规则 vs CEL/表达式库。
3. Case「分类」字段来源：现有 tag/类型字段复用还是新枚举。
4. 会员等级字段落在 User 的现有模型何处。
5. 同一 Topic 多 Edge 时是否竞争消费（建议是）及与「绑定特定机器规格」的关系。

## Implementation Divergence

**确认日期：** 2026-08-11（Verify 阶段用户选择 A）

Open 阶段本文曾将 Topic 目录、投放表达式、会员×分类分流与 Admin 管理一并纳入 Goals / D5 / D7。Design 确认后范围收窄：**本期不做 Topic 分流**。

| 项 | Open 本文原述 | 实际交付（以 Design Doc 为准） |
|---|---|---|
| Topic Admin / 投放表达式 | 本期交付 | **延后**；delta `topic-admin` / `dispatch-topic-routing` 已 Spec Patch 为「MUST NOT 作为验收」 |
| 调度选路 | 按表达式选 Topic | 仍 `dispatch.<instance_id>`（或实例 `DispatchTopic`） |
| 双模式 + 方案 A | 隐含于材料通道 | **已交付**：allinone Memory+localfs；split Edge+Redis Streams+S3；`job_ref` |

权威技术设计：`docs/superpowers/specs/2026-08-10-edge-agent-dual-mode-design.md`。归档时以 Design Doc + 已 Patch 的 delta 为准同步主 spec；Open Questions 1–5 中与 Topic/表达式相关的项随分流需求另开 change。
