# 限界上下文与包职责

> 设计按模块边界，不按二进制个数。今日组装为 all-in-one；依赖方向保持可拆分。

总览：[overview.md](./overview.md)。数据模型：[data-model.md](./data-model.md)。

---

## 1. 模块地图

```text
┌─ channel/tg ─────────────────────────────────────┐
│  Update→用例；DTO→消息/按钮；Notify→发图          │
│  主菜单只读 ← tgmenu                               │
└───────────────────────┬──────────────────────────┘
                        ▼
┌─ packaging/botapp ───────────────────────────────┐
│  StartCase / ConfirmRun / 跨 BC 应用用例          │
└───────┬───────────────┬───────────────┬──────────┘
        ▼               ▼               ▼
┌─ identity ─┐  ┌─ conversation ─┐  ┌─ catalog ─┐
│ User upsert│  │ Session 填表   │  │ Case 目录 │
└────────────┘  └────────────────┘  └─────┬─────┘
                                          │ 校验/读 Doc
┌─ tgmenu ────────────────────────────────────────┐
│  MenuTree 真相源；ReplaceTree；Case 反查 Placements │
│  admin GET/PUT /tg-menu；GET .../menu-placements │
│  bot 只读；folder Inline 在 channel/tg          │
└─────────────────────────────────────────────────┘
┌─ runtime ───────────────────────────────────────┐
│ domain.Task │ orchestrator │ actuator │ comfyui │
└───────┬─────────────┬─────────────┬─────────────┘
        ▼             ▼             ▼
   platform/instance  queue/blob   notify 端口
```

---

## 2. Bounded Context 职责

| Context | 包根 | 负责 | 不负责 |
|---|---|---|---|
| **Identity** | `internal/identity` | User 聚合、按 `tg_user_id` upsert | Session/Task/TG 协议细节 |
| **Conversation** | `internal/conversation` | Session 填表状态机、草稿、Start/Skip/Exit/Confirm 前 | Task 执行、发图 |
| **Catalog** | `internal/catalog` | Case 持久化、`doc_json` 协议、输入校验 | 调度、Comfy 调用 |
| **Runtime** | `internal/runtime` | Task 状态机、Orchestrator、Actuator、Comfy Client | TG UI、User 资料 |
| **Channel TG** | `internal/channel/tg` | Telegram 适配、菜单/回调、通知落地 | 领域规则 |
| **TG Menu** | `internal/tgmenu` | 主键盘树持久化、校验、`MenuPlacement` 反查 | TG Inline 发送、callback 路由 |
| **Platform** | `internal/platform` | db/blob/queue/notify/instance/botconfig | 业务决策 |
| **Packaging** | `internal/packaging/botapp` | 跨 BC 用例门面 | 基础设施实现细节 |
| **HTTP API** | `internal/httpapi` | 实例 CRUD/观测；Case/User/Session/Task；TG Menu | TG 通道实现 |
| **Shared Kernel** | `internal/sharedkernel` | ID、状态枚举、事件 DTO、topic | 业务行为 |

---

## 3. 关键类型与包

### Identity
- `domain.User` / `Repository`
- `infrastructure/persistence` → 表 `users`

### Conversation
- `domain.Session`、`DraftValue`、`Service`
- `infrastructure/persistence` → 表 `sessions`

### Catalog
- `domain.Case` / `CaseDocument`（`bindings.workflow` = Comfy API JSON）
- `infrastructure/persistence` → 表 `catalog_cases`
- `infrastructure/validation` → 按 Case 校验采集输入

### Runtime
| 包 | 职责 |
|---|---|
| `runtime/domain` | `Task`、状态迁移、`TaskRepository`（含 `ClaimQueued`） |
| `runtime/application/orchestrator` | 选实例、投递、status 收敛、终态 notify、对账 |
| `runtime/infrastructure/actuator` | 执行 Comfy、写 blob 产物、发 `task.status` |
| `runtime/infrastructure/comfyui` | `NewClient` / HTTP / Mock |
| `runtime/infrastructure/persistence` | 表 `tasks` |

### Channel TG
- `Adapter`、`Messenger` / `BotMessenger`
- `notifybridge`：实现 `platform/notify.Publisher` → `HandleUserNotify`

### TG Menu
- `domain.MenuTree` / `MenuNode` / `MenuKind` / `MenuPlacement`；`Validate` + `MaxTreeDepth`
- `application.Service`：`Get` / `Replace` / `ListPlacementsByCase` / `EnsureDefault`
- `infrastructure/persistence` → 表 `tg_menus`、`tg_menu_items`、`tg_menu_item_cases`（遗留 `tg_menu_configs` 仅迁移读）
- `httpapi/tgmenu`：`GET/PUT /api/v1/tg-menu`；`GET /api/v1/cases/{id}/menu-placements`

### Platform
| 包 | 职责 |
|---|---|
| `db` | SQLite Open + AutoMigrate |
| `botconfig` | YAML + env |
| `blob` + `blob/localfs` | 文件端口 |
| `queue` + `queue/memory` | 进程内 Bus；`SubscriptionSet` |
| `notify` | `Publisher` 端口（同步接口，非 bus） |
| `instance` + `persistence` | `Record` / Pool / Seed；表 `comfy_instances` |

---

## 4. 依赖规则

1. **`domain/*` 只依赖 `sharedkernel`**，不依赖 adapter、orchestrator、具体 MQ/FS SDK。
2. **BC domain 互不 import**；跨聚合编排放在 `packaging/botapp` 或 application 服务。
3. **`channel/tg` → `botapp` → domains/platform`**；禁止 domain 反向依赖 channel。`channel/tg` 只读依赖 `tgmenu`（窄端口）；`httpapi/tgmenu` 写同一真相源；`apps/admin-api` **禁止** import `channel/tg`。
4. **Orchestrator / Actuator** 依赖 `runtime/domain` + platform **ports**；通知只调 `notify.Publisher`，不直接 import TG。
5. **换实现**（Memory→NATS、LocalFS→S3）= 加 port 适配器，不改 BC 边界。
6. **已知特例**：`platform/instance.Pool` 持有 `runtime/.../comfyui.Client`（池在平台层建客户端）。
7. **组合根** `apps/bot` 是唯一允许全局接线的层；`apps/admin-api` 约定不依赖 `channel/tg`。

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
           + memory queue + local blob + sqlite + instance pool
```

**可拆方向（未做，边界已留）：**

```text
进程 A：tg + botapp + 写库 + Publish task.created
进程 B：orchestrator + Task 真相 + queue
进程 C：actuator + comfyui + blob + 订 dispatch.<id>
```

拆的是进程组装；模块代码与依赖方向不变。
