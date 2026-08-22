# 运行时：调度、执行与事件

> ConfirmRun 之后的控制面 / 执行面。数据落库见 [data-model.md](./data-model.md)；按阶段的样例数据快照见 [task-data-walkthrough.md](./task-data-walkthrough.md)。

---

## 1. 主链路（端到端）

```mermaid
sequenceDiagram
  participant U as Telegram User
  participant TG as channel/tg
  participant APP as botapp.ConfirmRun
  participant O as Orchestrator
  participant E as pixoma-edge-agent
  participant C as Comfy/Mock
  participant N as notify→TG

  U->>TG: 确认生成
  TG->>APP: ConfirmRun
  APP->>APP: Create Task(pending) + blob inputs
  APP->>O: OnTaskCreated（同进程）
  O->>O: 选实例 + prep job + 可领取（queued）
  E->>O: GET /agent/v1/jobs/claim
  O-->>E: task + job_ref
  E->>E: Blob.Get(job_ref) + 本机 UploadImage
  E->>C: Submit / Wait
  E->>O: POST .../status
  O->>N: 终态 UserNotify
  N->>U: 发图/文案
```

默认跨进程派发是 **DB 可领取态 + Edge 长轮询**，不是 Redis Topic。

### 1.0 本机与远程

| 位置 | 进程 | Blob | 执行面 |
|---|---|---|---|
| 本机（默认） | `pixoma` 自动 spawn Edge | localfs 共用目录 | `pixoma-edge-agent` |
| 远程 | 控制面 + 独立 Edge | s3 或 tos（禁止 localfs） | Edge 出站 claim |

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
    TG->>U: SendInline（正文 intro_text 或 label；Case 在前，子 folder，再返回）
    U->>TG: callback mf:&lt;item_id&gt; / mb:root|&lt;parent_id&gt;
    TG->>TG: showMenuFolder 下钻/返回
  else open_case / list_cases_by_tag / placeholder / reply_media
    TG->>TG: 既有 Case 列表或单 Case 预览链路
  end
```

要点：

- **根键盘**：仅 `parent_id` 为空的启用节点；`BuildReplyKeyboard` 按 `(row, col)` 排序。
- **folder**：根按钮或 Inline「📁」进入 `showMenuFolder`；消息正文优先 `intro_text`（空则用 `label`）；按钮顺序为同节点 `case_ids`（`CBCasePreview`）→ 子 folder（`mf:`）→ 「⬅️ 返回」（`mb:root` / `mb:<parent_id>`）。
- **list_cases_by_tag**：仍走既有按 tag 列表 Inline（兼容旧 `btn-image` 行为）；与 folder 内挂 Case 可并存。
- **callback 编码**：`mf:<menu_item_id>` 下钻；`mb:root` / `mb:<parent_item_id>` 返回（见 `channel/tg/menu.go`）。

---

## 2. 控制面 vs 执行面

| 角色 | 包 | 职责 |
|---|---|---|
| **Orchestrator** | `runtime/application/orchestrator` | 路由求值、按 Topic 置可领取、收敛 status、失败有界重试、终态 notify、对账 |
| **Actuator** | `runtime/infrastructure/actuator` | 按实例客户端跑 workflow、产物入 blob、上报 status |
| **Edge Pool** | `platform/edge` | CRUD 元数据、健康探测、持有 per-edge Client、RR 候选 |

Task 表是**执行态唯一真相源**（无独立 Actuator Ledger）。

---

## 3. 事件与端口

### Queue topics（`sharedkernel`）

同进程编排仍可用这些名字；跨进程向 Edge **不要求** Publish `dispatch.<edge_id>`。

| Topic | 载荷 | 方向 |
|---|---|---|
| `task.created` | `TaskCreated` | ConfirmRun → Orchestrator（可同进程） |
| `task.status` | `TaskStatusEvent` | Edge Agent API → Orchestrator |

跨进程投递：`GET /agent/v1/jobs/claim` 返回含 `job_ref` 的任务。

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
| 组合根 | `apps/pixoma/cmd/pixoma/main.go` |

---

## 4. 调度策略（Topic 分流）

```text
pending Task
    → 求值 Case 路由（首个命中即投；无命中 → default）
    → 目标 Topic 无在线订阅节点 → 保持 pending 记原因
    → prep job（edge 无关）+ PrepareForTopic（queued + dispatch_topic，不绑定节点）
    → Edge GET /agent/v1/jobs/claim（按节点订阅 Topic 集合原子抢占 + lease）
```

- **同一任务只被一台消费**：claim 端点按订阅集合在事务内条件 UPDATE 抢占。
- **失败有界重试**：`attempts+1`，退避 5s/15s/45s，超限收敛 failed；沿用原 `dispatch_topic`。
- **租约过期**：回到 queued 可再领。
- **默认 Topic**：启动幂等创建 `default`；节点未配置订阅 = 订阅 `default`；无规则命中回退 `default`。
- **条件协议**：声明式 JSON 规则 + 属性提供方（`user.is_premium` / `case.category` / `case.tags`），新增属性只注册 provider + schema，不改引擎。

---

## 5. Comfy 客户端选型

```text
cfg.ComfyMock ──► Pool.Refresh ──► comfyui.NewClient(Options{Mock, BaseURL})
                                      ├─ true  → Mock（进程内，返回示例 PNG）
                                      └─ false → HTTP(BaseURL)
```

- 产品链路变更须保持 **Mock 端到端可通**（项目规则 `comfy-mock-parity`）。
- 观测 API：`GET .../system` → `/system_stats`；`GET .../queue` → `/queue`；业务历史**不用** Comfy `/history`，用 DB `tasks`。
- 管理列表/详情的「节点在线」「Comfy运行中」两个 Tag **不以** 控制面探 Comfy 为准：Edge 本机 `SystemStats` 后 `POST /agent/v1/presence`，控制面内存 15 秒无报到视为掉线。`GET .../system` 仍是硬件明细，远程可能打不通。

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
| `queued` + `edge_id` | Orchestrator `ClaimQueued` |
| `running` + `prompt_id` | Actuator / status 回写 |
| 终态 + outputs/error | Orchestrator `OnStatus` |

Session 状态机（独立）：`collecting` → `confirming` → `submitted` | `exited`。Session **不含** Task 执行字段。

---

## 7. 观测与运维面

| HTTP | 数据源 |
|---|---|
| `GET/POST/PATCH/DELETE /api/v1/edges` | `edges` |
| `POST .../{id}/rotate-token` | 换发该节点 AGENT_TOKEN |
| `POST /agent/v1/presence` | Edge 上报 `{ edge_id, comfy_running, started_at, comfy_version, hardware?, metrics? }` → `{ refresh_hardware }`；`started_at` 首次心跳只记一次，`comfy_version` 来自 `/system_stats`；`metrics` 按 `METRICS_INTERVAL`（默认 30s）采样随心跳携带，控制面写入 `edge_metrics` 与 `edges` |
| `GET /api/v1/edges/presence` | 内存；15s 无报到视为掉线，Comfy 一并显示未启动 |
| `GET .../{id}/system` | 该节点 Comfy system_stats（硬件明细；远程可能不通） |
| `GET .../{id}/queue` | 该节点 Comfy queue |
| `GET .../{id}/metrics` | `edge_metrics` 窗口查询（默认 1h），返回 `{ latest, series }`；详情页「系统监控」图表数据源 |
| `GET .../{id}/tasks` | `tasks WHERE edge_id=?` |
| `GET .../{id}/stats` | 该节点任务数 / 累计耗时 / 成功率 |
| `GET /api/v1/stats/tasks/daily?from&to` | `task_daily_stats` 按天查询（零填充，默认近 30 天、上限 365 天），附成功率汇总；任务终态由 orchestrator 写入统计表 |
| `GET /api/v1/stats/tasks/errors?from&to&limit` | `task_error_daily_stats` 错误码 Top-N |
| `GET /api/v1/stats/tasks/edges?from&to` | `task_edge_daily_stats` 每节点已处理任务数与 total |
| `GET /api/v1/stats/cases/top?from&to&limit` | `task_case_daily_stats` 按 Case 聚合的终态任务数与平均耗时 |
| `GET /api/v1/stats/fleet` | `edge_metrics` 每节点最新快照聚合：在线数、平均 CPU/内存/GPU、VRAM 占用、最热节点（实时，无时间范围） |
| `GET /healthz` | 进程存活 |

> 计算节点 API **当前无鉴权**，仅本机/可信内网。
