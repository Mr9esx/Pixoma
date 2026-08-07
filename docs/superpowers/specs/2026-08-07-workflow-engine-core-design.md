---
comet_change: workflow-engine-core
role: technical-design
canonical_spec: openspec
---

# 工作流引擎核心 — 技术设计

## 1. 背景与目标

在空仓库上用 Go 落地 ComfyUI Telegram Bot 的**工作流核心**：Case 协议与注册、私聊 Dialog Session、异步 Task，以及可组装的 **Bot / Orchestrator / Actuator** 模块。模块边界优先于进程个数；同进程可 all-in-one，后续按模块拆分。

**目标**

- Case 协议（元数据、有序 input/output、JSON Schema 校验、Comfy 绑定、价格数值、可扩展 tags）
- Registry：GORM + SQLite（可切 MySQL）
- TG 适配：菜单→分类→Case→上锁填表→ConfirmRun
- 异步执行：Blob 物化 → Orchestrator 调度 → Actuator 调 ComfyUI → status 收敛 → notify 回用户
- Queue / Blob 端口化，一期 Memory + LocalFS

**非目标**

- 真实积分扣费、多租户 Bot 部署平台、强取消（interrupt running）
- Orchestrator / Actuator 直接依赖 Telegram SDK（仅 adapter/tg）

## 2. 模块架构

```text
adapter/tg → app → domain{protocol, case, session, task}
                      │
         orchestrator ┼─ ports{queue, blob, notify, instance}
         actuator ────┘─ comfyui.Client
```

| 模块 | 职责 |
|------|------|
| protocol | Case 文档模型；JSON Schema 校验 + 媒体扩展 |
| case | 注册/查询/上下架 |
| session | chat_id 填表锁与草稿 |
| task | Task 聚合与合法状态迁移 |
| app | 用例门面（ConfirmRun 等） |
| orchestrator | 选实例、投递、applyStatus、对账、取消、风暴防护、notify |
| actuator | 消费 dispatch、执行 Comfy、本地 ledger、发 status、ExecutionQuery |
| adapter/tg | Update↔用例；notify→发图/改消息 |
| port/* | queue / blob / notify / instance 接口 |

依赖规则：`domain` 不引用 TG/MQ SDK；`orchestrator` 不发 TG；`actuator` 不写全局 Task 终态、不碰 Bot。

## 3. 关键状态机

### 3.1 Dialog Session（每 chat_id 最多一条非终态）

- 浏览菜单：**不建** Session  
- `collecting` / `confirming`：**上锁**（禁止 StartCase）  
- `submitted` / `exited`：终态，解锁  
- 有 running Task **不阻止** 新开 Case  

### 3.2 Task

`pending → queued → running → succeeded | failed`  
`cancelled` 仅自 `pending|queued`（温和取消）

**写库**：仅 Orchestrator 调用 `applyStatus`（含 OnStatus 与对账）。

## 4. ConfirmRun 后事件流

```text
app.ConfirmRun
  → Validate + Blob stage + Task(pending) + Clear Session
  → Publish task.created
orchestrator
  → 选实例 + MarkQueued + Publish dispatch.<instance_id>
actuator
  → Comfy 执行 + Publish task.status（running/终态）
orchestrator.applyStatus
  → 写 Task + Publish notify.user
adapter/tg
  → 回用户
```

**双触发**：`task.created`（快）+ `SchedulePending` 扫库（稳）。  
**Status 丢失**：超时扫描 + `ExecutionQuery.GetRun` / Blob 线索 → 同一 `applyStatus`。  
**Actuator**：本地 RunLedger；可重发 status；**不**直接 UPDATE tasks 终态。

事件 Topic：`task.created` | `dispatch.<id>` | `task.status` | `notify.user`。Payload 以 ID + BlobRef 为主。

## 5. Orchestrator 风暴防护

- 全局 / 按实例：调度、对账、重派、Query 分配额  
- 指数退避 + jitter；可重试 vs 不可重试分类  
- 实例熔断 + 半开  
- SchedulePending 与 ReconcileStale **分池**  
- bulk redispatch 令牌桶  
- notify 去重；进度合并  
- 指标：积压、失败率、熔断状态  

端口：`RetryPolicy` / `RateLimiter` / `CircuitBreaker`。

## 6. 技术选型

| 项 | 选择 |
|----|------|
| 语言 | Go |
| ORM/DB | GORM + SQLite（modernc）；预留 MySQL |
| 协议校验 | JSON Schema 引擎 + 媒体/OSS 钩子 |
| TG | `github.com/go-telegram/bot` |
| HTTP | chi（Webhook/health） |
| 日志 | slog |
| Queue | Port + Memory 适配器 |
| Blob | Port + LocalFS |
| ComfyUI | 自研 net/http client |
| 测试 | testify；假 Queue/Comfy/Query |

## 7. 工程目录（DDD + 多应用预留）

完整树、限界上下文与依赖规则见：  
`docs/openspec/changes/workflow-engine-core/.comet/handoff/project-layout.md`。

**要点：**

- **Monorepo**：`apps/bot`（本期）、`apps/admin-api` + `web/admin`（后台预留）
- **限界上下文**：Catalog / Conversation / Runtime / ChannelTG / Platform；（预留 Identity、Billing）
- 每 BC 内 DDD 分层：`domain` → `application` → `infrastructure`
- 跨 BC 用例（如 ConfirmRun）放在 `packaging/botapp`，不放进 domain
- Admin **复用** Catalog/Runtime 应用服务，**禁止**依赖 `channel/tg`；前端只调 admin-api

```text
apps/bot | apps/admin-api（预留）
web/admin（预留）
internal/
  sharedkernel/
  catalog/{domain,application,infrastructure}
  conversation/{...}
  runtime/{domain,application/orchestrator,infrastructure/actuator|comfyui}
  channel/tg/
  platform/{queue,blob,notify,instance}
  packaging/botapp/
  identity/（预留）
```

## 8. TG 应用与渲染

- Application API 返回 DTO；adapter 渲染为文本/Photo + InlineKeyboard  
- 导航：菜单大类 → 子分类 → Case 列表/详情 → StartCase  
- `callback_data` 短编码；会话状态在服务端按 `chat_id`  

## 9. 测试策略

1. protocol：合法/非法 input、skip、enum、媒体钩子  
2. session：锁、Exit、Confirm 后解锁  
3. task：合法边；非法 cancelled(running)  
4. orchestrator：调度幂等、applyStatus 幂等、熔断与限流、对账补写  
5. actuator：注入、ledger、status 重发、Query  
6. 冒烟：Memory all-in-one text2img（Comfy 可 mock）  

## 10. 风险与缓解

| 风险 | 缓解 |
|------|------|
| 控制面重试风暴 | §5 强制策略 |
| status 丢失 | Query 对账 + 统一 applyStatus |
| Comfy 节点漂移 | 绑定外置 Case；样例工作流测 |
| 大文件 | 只传 BlobRef |
| Open 文档与现架构偏差 | 本 Doc + Spec Patch 对齐 |

## 11. 实现顺序建议

1. kernel + ports + Memory/LocalFS  
2. protocol + case + task + session  
3. app.ConfirmRun + 事件  
4. orchestrator（含限流/对账）  
5. actuator + comfyui（可 mock）  
6. adapter/tg 主路径  
7. 样例 Case 种子与 README  

## 12. 参考手稿

`docs/openspec/changes/workflow-engine-core/.comet/handoff/`：  
domain-modules、module-ports-and-events、orchestrator-full-roles、tech-stack-and-layout、dialog-session / task 状态机、phase-boundaries。
