# 运行时：调度、执行与事件

> ConfirmRun 之后的控制面 / 执行面。数据落库见 [data-model.md](./data-model.md)；按阶段的样例数据快照见 [task-data-walkthrough.md](./task-data-walkthrough.md)。

---

## 1. 主链路（端到端）

```mermaid
sequenceDiagram
  participant U as Telegram User
  participant TG as channel/tg
  participant APP as botapp.ConfirmRun
  participant Q as queue/memory
  participant O as Orchestrator
  participant A as Actuator
  participant C as Comfy/Mock
  participant N as notify→TG

  U->>TG: 确认生成
  TG->>APP: ConfirmRun
  APP->>APP: Create Task(pending) + blob inputs
  APP->>Q: Publish task.created
  Q->>O: OnTaskCreated
  O->>O: ClaimQueued + prep job + round-robin
  O->>Q: Publish dispatch.<instance_id> {job_ref}
  Q->>A: HandleDispatch
  A->>A: Blob.Get(job_ref) + 本机 UploadImage
  A->>C: Submit / Wait
  A->>Q: Publish task.status
  Q->>O: OnStatus（写 Task）
  O->>N: 终态 UserNotify
  N->>U: 发图/文案
```

说明：`allinone` 下 `memory` bus **同步**调用 handler；ConfirmRun 返回前，整条链路（含 notify）可能已完成。

### 1.0 双模式

| 模式 | 进程 | Queue | Blob | 执行面 |
|---|---|---|---|---|
| `allinone`（默认） | Bot 单进程含执行面 | Memory | localfs | 同进程订阅 `dispatch.*` |
| `split` | 云 Bot + `apps/edge-agent` | Redis Streams | S3 兼容 | Edge 订阅；Bot 不订生产 dispatch |

两种模式均为 **方案 A**：调度 `PrepareJob` 写 `jobs/<task_id>/job.json`，dispatch 带 `job_ref`；执行面不读 Case/Task DB 拼装。

对话入口：Telegram 主 ReplyKeyboard 来自 `tg_menus` + `tg_menu_items`（空库种子或自 `tg_menu_configs` 迁移）；`channel/tg` 每次构建键盘时读 `MenuTree`（失败回退 `DefaultSeedTree`）。

### 1.1 TG 主键盘与 folder Inline 浏览

```mermaid
sequenceDiagram
  participant U as User
  participant TG as channel/tg
  participant M as tgmenu.Repository

  U->>TG: /start 或点根按钮
  TG->>M: GetTree(default)
  M-->>TG: MenuTree
  alt 根级 ReplyKeyboard
    TG->>U: SendMenu（按 row/col 布局）
  else kind=folder
    TG->>U: SendInline（子 folder + 挂载 Case + 返回）
    U->>TG: callback mf:&lt;item_id&gt; / mb:root|&lt;parent_id&gt;
    TG->>TG: showMenuFolder 下钻/返回
  else open_case / list_cases_by_tag / placeholder / reply_media
    TG->>TG: 既有 Case 列表或单 Case 预览链路
  end
```

要点：

- **根键盘**：仅 `parent_id` 为空的启用节点；`BuildReplyKeyboard` 按 `(row, col)` 排序。
- **folder**：根按钮或 Inline「📁」进入 `showMenuFolder`；列出子 folder（`mf:` 前缀 callback）与同节点 `case_ids`（`CBCasePreview`）；「⬅️ 返回」用 `mb:root` 或 `mb:<parent_id>`。
- **list_cases_by_tag**：仍走既有按 tag 列表 Inline（兼容旧 `btn-image` 行为）；与 folder 内挂 Case 可并存。
- **callback 编码**：`mf:<menu_item_id>` 下钻；`mb:root` / `mb:<parent_item_id>` 返回（见 `channel/tg/menu.go`）。

---

## 2. 控制面 vs 执行面

| 角色 | 包 | 职责 |
|---|---|---|
| **Orchestrator** | `runtime/application/orchestrator` | 认领 pending、选健康实例、发 dispatch、收敛 status、终态 notify、对账 |
| **Actuator** | `runtime/infrastructure/actuator` | 按实例客户端跑 workflow、产物入 blob、上报 status |
| **Instance Pool** | `platform/instance` | CRUD 元数据、健康探测、持有 per-instance Client、RR 候选 |

Task 表是**执行态唯一真相源**（无独立 Actuator Ledger）。

---

## 3. 事件与端口

### Queue topics（`sharedkernel`）

| Topic | 载荷 | 方向 |
|---|---|---|
| `task.created` | `TaskCreated` | ConfirmRun → Orchestrator |
| `dispatch.<instance_id>` | `DispatchCommand`（含 `job_ref`） | Orchestrator → Actuator/Edge |
| `task.status` | `TaskStatusEvent` | Actuator/Edge → Orchestrator |

`TopicNotifyUser` 常量存在，**未走 queue**。

### Notify 端口

```text
Orchestrator ──Publish(UserNotify)──► platform/notify.Publisher
                                          │
                                          ▼
                                   tg/notifybridge
                                          │
                                          ▼
                                   Adapter.HandleUserNotify
```

### 关键代码

| 步骤 | 路径 |
|---|---|
| Confirm | `internal/packaging/botapp/confirm_run.go` |
| 事件 DTO | `internal/sharedkernel/events.go` |
| Orchestrator | `internal/runtime/application/orchestrator/service.go` |
| Actuator | `internal/runtime/infrastructure/actuator/worker.go` |
| Case→workflow | `internal/runtime/infrastructure/actuator/snapshot.go` |
| 组合根订阅 | `apps/bot/cmd/comfyui-bot/main.go` |

---

## 4. 调度策略

```text
pending Task
    → ListHealthy(enabled ∩ 探测成功 ∩ 未熔断)
    → ClaimQueued(CAS pending→queued, 写 instance_id)
    → Publish dispatch.<instance_id>
```

- **无可用实例**：不投递（Task 保持可重试态，由对账/后续 tick 再试）。
- **Round-robin**：在健康集合上轮转。
- **动态订阅**：实例启用变化时 `SubscriptionSet` 保证 `dispatch.<id>` 有 handler。
- **健康探测**：周期调该实例 `SystemStats`；间隔由 `health_probe_interval` 配置。

---

## 5. Comfy 客户端选型

```text
cfg.ComfyMock ──► Pool.Refresh ──► comfyui.NewClient(Options{Mock, BaseURL})
                                      ├─ true  → Mock（进程内，返回示例 PNG）
                                      └─ false → HTTP(BaseURL)
```

- 产品链路变更须保持 **Mock 端到端可通**（项目规则 `comfy-mock-parity`）。
- 观测 API：`GET .../system` → `/system_stats`；`GET .../queue` → `/queue`；业务历史**不用** Comfy `/history`，用 DB `tasks`。

---

## 6. Task 状态机

```text
pending → queued → running → succeeded
                           ↘ failed
                           ↘ cancelled
```

| 迁移时机 | 谁写 |
|---|---|
| Create `pending` | ConfirmRun |
| `queued` + `instance_id` | Orchestrator `ClaimQueued` |
| `running` + `prompt_id` | Actuator / status 回写 |
| 终态 + outputs/error | Orchestrator `OnStatus` |

Session 状态机（独立）：`collecting` → `confirming` → `submitted` | `exited`。Session **不含** Task 执行字段。

---

## 7. 观测与运维面

| HTTP | 数据源 |
|---|---|
| `GET/POST/PATCH/DELETE /api/v1/comfy-instances` | `comfy_instances` |
| `GET .../{id}/system` | 该实例 Comfy system_stats |
| `GET .../{id}/queue` | 该实例 Comfy queue |
| `GET .../{id}/tasks` | `tasks WHERE instance_id=?` |
| `GET /healthz` | 进程存活 |

> 实例 API **当前无鉴权**，仅本机/可信内网。
