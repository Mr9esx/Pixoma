# 限界上下文与包职责

> 设计按模块边界，不按二进制个数。今日组装为 all-in-one；依赖方向保持可拆分。

总览：[overview.md](./overview.md)。数据模型：[data-model.md](./data-model.md)。

---

## 1. 模块地图

```text
┌─ channels/tg ─────────────────────────────────────┐
│  Update→用例；DTO→消息/按钮；Notify→发图          │
│  主菜单只读 ← tgmenu                               │
└───────────────────────┬──────────────────────────┘
                        ▼
┌─ packaging/botapp ───────────────────────────────┐
│  StartCase / ConfirmRun / 跨 BC 应用用例          │
└───────┬───────────────┬───────────────┬──────────┘
        ▼               ▼               ▼
┌─ users ─┐  ┌─ sessions ─┐  ┌─ cases ─┐
│ User upsert│  │ Session 填表   │  │ Case 目录 │
└────────────┘  └────────────────┘  └─────┬─────┘
                                          │ 校验/读 Doc
┌─ menus ──────────────────────────────────────────┐
│  每渠道一份嵌套菜单树；GET/PUT /channels/{id}/menu     │
│  Bot 编译成键盘/卡片；Case 反查 menu-placements         │
└─────────────────────────────────────────────────────┘
┌─ packaging/linkhealth ───────────────────────────┐
│  只读拼图：通道+菜单+Case 路由+Topic+Edge+presence  │
│  合成活路绿黄；HTTP GET /api/v1/link-health         │
└───────────────────────────────────────────────────┘

│ domain.Task │ orchestrator │ actuator │ comfyui │
└───────┬─────────────┬─────────────┬─────────────┘
        ▼             ▼             ▼
   edge           queue/blob   notify 端口
```

---

## 2. Bounded Context 职责

| Context | 包根 | 负责 | 不负责 |
|---|---|---|---|
| **Users** | `internal/users` | User 聚合、按 `tg_user_id` upsert | Session/Task/TG 协议细节 |
| **Sessions** | `internal/sessions` | Session 填表状态机、草稿、Start/Skip/Exit/Confirm 前 | Task 执行、发图 |
| **Cases** | `internal/cases` | Case 持久化、`doc_json` 协议、输入校验 | 调度、Comfy 调用 |
| **Tasks** | `internal/tasks` | Task 状态机、Orchestrator、Actuator、Comfy Client | TG UI、User 资料 |
| **Channels** | `internal/channels` | Telegram 适配、菜单/回调、通知落地 | 领域规则 |
| **Menus** | `internal/menus` | 渠道菜单嵌套树、校验、编译、placements | TG 发送、callback 路由 |
| **Platform** | `internal/platform` | db/blob/queue/notify/edge/botconfig | 业务决策 |
| **Packaging** | `internal/packaging/botapp`、`internal/packaging/linkhealth` | 跨 BC 用例门面；配置链路健康只读组装 | 写侧 CRUD、Task 执行 |
| **HTTP API** | `internal/httpapi` | 计算节点 CRUD/观测；Case/User/Session/Task；TG Menu；`GET /api/v1/link-health` | TG 通道实现 |
| **Shared Kernel** | `internal/sharedkernel` | ID、状态枚举、事件 DTO、topic | 业务行为 |

---

## 3. 关键类型与包

### Users
- `domain.User` / `Repository`
- `infrastructure/persistence` → 表 `users`

### Sessions
- `domain.Session`、`DraftValue`、`Service`
- `infrastructure/persistence` → 表 `sessions`

### Cases
- `domain.Case` / `CaseDocument`（`bindings.workflow` = Comfy API JSON）
- `infrastructure/persistence` → 表 `catalog_cases`
- `infrastructure/validation` → 按 Case 校验采集输入

### Tasks
| 包 | 职责 |
|---|---|
| `tasks/domain` | `Task`、状态迁移、`TaskRepository`（含 `ClaimQueued`） |
| `tasks/application/orchestrator` | 选实例、投递、status 收敛、终态 notify、对账 |
| `tasks/infrastructure/actuator` | 执行 Comfy、写 blob 产物、发 `task.status` |
| `tasks/infrastructure/comfyui` | `NewClient` / HTTP |
| `tasks/infrastructure/persistence` | 表 `tasks` |

### Channels
- `Adapter`、`Messenger` / `BotMessenger`
- `notifybridge`：实现 `platform/notify.Publisher` → `HandleUserNotify`
- `channels/application.ReachabilityProbe`：后台周期探测启用通道并写入 `last_check_*`；GET 热路径与页面渲染只读。打开消息平台时可 `POST /api/v1/channels/probe` 异步再踢一轮，不挡列表。

### Menus
- `domain.MenuTree` / `MenuNode` / `MenuKind` / `MenuPlacement`；`Validate` + `MaxTreeDepth`
- `application.Service`：`Get` / `Replace` / `ListPlacementsByCase` / `EnsureDefault`
- `infrastructure/persistence` → 表 `tg_menus`、`tg_menu_items`、`tg_menu_item_cases`（遗留 `tg_menu_configs` 仅迁移读）
- `httpapi/channels`：`GET/PUT /api/v1/tg-menu`；`GET /api/v1/cases/{id}/menu-placements`

### Platform
| 包 | 职责 |
|---|---|
| `db` | SQLite Open + AutoMigrate |
| `botconfig` | YAML + env |
| `blob` + `blob/localfs` / `blob/s3` / `blob/tos` | 文件端口（localfs；S3 兼容；火山 TOS） |
| `queue` + `queue/memory` | 进程内 Bus；`SubscriptionSet` |
| `notify` | `Publisher` 端口（同步接口，非 bus） |
| `edge` + `persistence` | `Record` / Pool / Seed；表 `edges` |

---

## 4. 依赖规则

1. **`domain/*` 只依赖 `sharedkernel`**，不依赖 adapter、orchestrator、具体 MQ/FS SDK。
2. **BC domain 互不 import**；跨聚合编排放在 `packaging/botapp` 或 application 服务。
3. **`channels/tg` → `botapp` → domains/platform`**；禁止 domain 反向依赖 channel。`channels/tg` 只读依赖 `tgmenu`（窄端口）；`httpapi/channels` 写同一真相源；组合根 `apps/pixoma` 之外的入口 **禁止** import `channels/tg`。
4. **Orchestrator / Actuator** 依赖 `tasks/domain` + platform **ports**；通知只调 `notify.Publisher`，不直接 import TG。
5. **换实现**（Memory→NATS、LocalFS→S3/TOS）= 加 port 适配器，不改 BC 边界。
6. **已知特例**：`edge/application.Pool` 持有 `tasks/infrastructure/comfyui.Client`（池在 edge 应用层建客户端）。
7. **组合根** `apps/pixoma` 是唯一允许全局接线的层（已移除旧 `apps/bot` / `apps/admin-api` 入口）。

```text
sharedkernel ← domain BCs（含 tgmenu）← application / packaging ← channel / httpapi ← apps/*
                     ↑
                platform ports
```

---

## 5. 组合与拆分

**今日：**

```text
all-in-one = tg + botapp + domains + orchestrator + actuator
           + memory queue + local blob + sqlite + edge pool
```

**可拆方向（未做，边界已留）：**

```text
进程 A：tg + botapp + 写库 + Publish task.created
进程 B：orchestrator + Task 真相 + queue
进程 C：actuator + comfyui + blob + 订 dispatch.<id>
```

拆的是进程组装；模块代码与依赖方向不变。
