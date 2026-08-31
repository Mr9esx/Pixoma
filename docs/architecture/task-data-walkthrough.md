# 任务全生命周期数据走查

> 从用户选 Case、采集输入、确认生成，到 Task 终态与 TG 回图。  
> 本文只讲**数据落在哪、长什么样**；控制流见 [runtime.md](./runtime.md)，表结构见 [data-model.md](./data-model.md)。

默认按 **`runtime_mode=allinone`**（`queue=Memory`、`blob=localfs`）写完整样例；文末对照 **`split`**（Redis Streams + S3 或 TOS + edge-agent）差异。

---

## 0. 样例约定（Example）

固定一组 ID，便于对照各阶段快照：

| 角色 | 值 |
|---|---|
| Telegram `from.id` / `chat.id` | `10001` |
| 内部 `users.id` | `usr-alice-001` |
| Case | `text2img-demo`（`configs/cases/text2img.example.json`） |
| Session | `sess-20260811-001` |
| Task | `task-a1b2c3d4` |
| Comfy 实例 | `local` |
| 用户输入 | `prompt` = `"a red cat"`；`seed` 跳过 |
| 时钟 | `2026-08-11T06:00:00Z` 起（下文用 `T0`、`T0+1s`…） |

Case 输入定义（摘要）：

```json
{
  "id": "text2img-demo",
  "inputs": [
    {"key": "prompt", "type": "string", "required": true},
    {"key": "seed", "type": "number", "required": false, "skip_allowed": true}
  ],
  "bindings": {
    "workflow": {"1": {"class_type": "Stub", "inputs": {}}},
    "inputs": [{"key": "prompt", "node_id": "1", "field_path": "text"}]
  }
}
```

---

## 1. 前置数据（任务开始前已存在）

这些**不是**本次会话写入的主路径，但调度/执行会读到。

### 1.1 `catalog_cases`

| 列 | 样例值 |
|---|---|
| `id` | `text2img-demo` |
| `name` | `Text to Image Demo` |
| `enabled` | `1` |
| `doc_json` | 完整 CaseDocument（含 `bindings.workflow`） |
| `tags_json` | `["image","text2img"]` |

### 1.2 `edges`

| 列 | 样例值 |
|---|---|
| `id` | `local` |
| `name` | `local` |
| `base_url` | `http://127.0.0.1:8188` |
| `enabled` | `1` |
| `capabilities_json` | `[]` |
| `agent_token_enc` | AES-GCM 密文 |
| `hardware_json` | CPU / 显卡（首次 presence 可写） |

### 1.3 菜单（可选）

`tg_menus` / `tg_menu_items` / `tg_menu_item_cases` 决定 TG 如何点到该 Case；与 Task 行无直接外键。

---

## 2. 阶段总览

```text
选 Case → 采输入 → 确认
    → Blob stage inputs + INSERT tasks(pending) + sessions.submitted
    → MQ task.created
    → ClaimQueued + PrepareJob(Blob job) + MQ dispatch.local
    → Actuator Submit/Wait Comfy
    → MQ task.status(running → succeeded)
    → UPDATE tasks + Notify(TG 发图)
```

| 阶段 | 主要写入 | 主要读出 |
|---|---|---|
| A 身份 | `users` upsert | — |
| B 开 Session | `sessions` INSERT | `catalog_cases` |
| C 采输入 | `sessions` UPDATE；图片则 Blob `tg/...` | — |
| D ConfirmRun | Blob `inputs/<task>/…`；`tasks` INSERT；`sessions` → submitted；MQ `task.created` | session draft + case |
| E 调度 | `tasks` → queued；Blob `jobs/<task>/job.json`；MQ `dispatch.<id>` | edges / online |
| F 执行 | Blob `outputs/…`；MQ `task.status`；Comfy HTTP | Blob job + images |
| G 收敛 | `tasks` UPDATE；进程内 Notify → TG API | session.chat_id |

---

## 3. 阶段 A — 用户发消息 / 点按钮（身份）

**入口**：Telegram Update → `channel/tg.Adapter` → `UpsertFromTG`。

### 写入：`users`

```json
{
  "id": "usr-alice-001",
  "tg_user_id": 10001,
  "username": "alice",
  "first_name": "Alice",
  "last_name": "",
  "language_code": "zh-hans",
  "last_seen_at": "2026-08-11T06:00:00Z"
}
```

- 行为：按 `tg_user_id` upsert；刷新 `last_seen_at`。
- **无 MQ**。

---

## 4. 阶段 B — 开始 Case（创建 Session）

**入口**：用户选 Case → `botapp.StartCase` → `conversation.StartCase`。

### 写入：`sessions`（INSERT）

| 列 | 值 |
|---|---|
| `id` | `sess-20260811-001` |
| `user_id` | `usr-alice-001` |
| `chat_id` | `10001` |
| `case_id` | `text2img-demo` |
| `status` | `collecting` |
| `current_input_index` | `0` |
| `input_keys_json` | `["prompt","seed"]` |
| `draft_json` | `{}` |
| `created_at` / `updated_at` | `T0` |

**读**：`catalog_cases`（校验 enabled，取 inputs 顺序）。

**接口（观测，非主路径）**：Admin `GET /api/v1/sessions` 可看到该行。

此时**尚无** `tasks` 行。

---

## 5. 阶段 C — 采集输入（改 Session Draft）

### 5.1 提交 `prompt`（文本）

用户发：`a red cat` → `SubmitInput`。

### 写入：`sessions`（UPDATE）

```json
{
  "status": "collecting",
  "current_input_index": 1,
  "draft_json": {
    "prompt": {
      "Key": "prompt",
      "Text": "a red cat"
    }
  }
}
```

（JSON 字段名以 GORM 序列化的 `DraftValue` 为准；领域结构见 `conversation/domain.DraftValue`。）

### 5.2 跳过 `seed`

用户点「跳过」→ `SkipInput`。

```json
{
  "status": "confirming",
  "current_input_index": 2,
  "draft_json": {
    "prompt": { "Key": "prompt", "Text": "a red cat" },
    "seed": { "Key": "seed", "Skipped": true }
  }
}
```

采齐后 `status` 变为 **`confirming`**（本样例在 Skip 时触发）。

### 5.3 若输入是图片（对照说明）

以图生图 Case 为例，用户发图时：

1. TG `getFile` 下载字节  
2. **Blob Put**：`tg/10001/<nanos>.jpg`  
3. Session draft 写入 `Blob: { key, mime, size }`  
4. ConfirmRun 时再 stage 到 `inputs/<task_id>/<field>.blob.json`（见阶段 D）

本样例无图，无 `tg/...` 对象。

---

## 6. 阶段 D — ConfirmRun（创建 Task + 暂存输入）

**入口**：用户点「✅ 确认生成」→ `botapp.ConfirmRun`。

顺序（代码路径）：校验 session=`confirming` → 校验 Case/inputs → 生成 `task_id` → **stage Blob** → **INSERT tasks** → session `submitted` → **Publish `task.created`**。

### 6.1 Blob：`inputs/<task_id>/…`

| Key | MIME | 内容 |
|---|---|---|
| `inputs/task-a1b2c3d4/prompt.txt` | `text/plain` | `a red cat` |

- `seed` 已 Skipped → **不写** stage 文件。  
- allinone 物理路径示例：`data/blob/inputs/task-a1b2c3d4/prompt.txt`。

图片字段时还会写：

| Key | 内容 |
|---|---|
| `inputs/<task>/<key>.blob.json` | 原 draft 中 `BlobRef` 的 JSON（指向已有 `tg/...` 或其它 blob） |

### 6.2 DB：`tasks` INSERT

| 列 | 值 |
|---|---|
| `id` | `task-a1b2c3d4` |
| `session_id` | `sess-20260811-001` |
| `case_id` | `text2img-demo` |
| `status` | `pending` |
| `edge_id` | `""` |
| `prompt_id` | `""` |
| `input_prefix` | `inputs/task-a1b2c3d4` |
| `outputs_json` | `[]` |
| `error_code` / `error_message` | 空 |
| `created_at` / `updated_at` | `T0+5s` |

> `ChatID` 在领域对象上会缓存，**不作为 `tasks` 必填列落库**；通知时优先 `session_id` → `sessions.chat_id`。

### 6.3 DB：`sessions` UPDATE

| 列 | 新值 |
|---|---|
| `status` | `submitted` |
| `updated_at` | `T0+5s` |

Draft **保留**（不删），便于审计；Session 不再是 active。

### 6.4 MQ：`task.created`

| 字段 | 值 |
|---|---|
| Topic | `task.created` |
| Key | `task-a1b2c3d4` |
| Payload | 见下 |

```json
{
  "task_id": "task-a1b2c3d4",
  "chat_id": 10001,
  "case_id": "text2img-demo",
  "created_at": "2026-08-11T06:00:05Z"
}
```

**allinone**：Memory bus **同步**调用 Orchestrator `OnTaskCreated`（ConfirmRun 返回前可能已跑完整条链路）。  
**split**：Redis Stream 名通常为 `q:task.created`（前缀默认 `q:`），fields：`topic` / `key` / `payload`。

### 6.5 TG 即时回复（非库）

Adapter：`已排队\ntask=task-a1b2c3d4\n完成后会把图片发回来。`（Telegram Bot API `sendMessage`）。

---

## 7. 阶段 E — Orchestrator 调度（认领 + 任务包 + dispatch）

> **Topic 分流（2026-08-20 起）**：调度改为「求值 Case 路由 → `dispatch_topic`（无命中 `default`）→ prep edge 无关任务包 → `PrepareForTopic`（queued + topic，**不绑定节点**）」。`/agent/v1/jobs/claim` 按节点订阅集合原子抢占（同一任务只被一台消费）；失败有界重试（5s/15s/45s），租约过期回收不烧 attempts。下文旧示例保留作历史对照。

**入口**：`OnTaskCreated` → `dispatchTask`。

### 7.1 读

- `tasks`：须仍为 `pending`  
- 实例：`ListHealthy`（allinone）或 `ListEnabled` + `edge:online:<id>` TTL（split）  
- Case + staged inputs（PrepareJob）

### 7.2 DB：`tasks` ClaimQueued（CAS）

条件：`id=? AND status='pending'`。

| 列 | 新值 |
|---|---|
| `status` | `queued` |
| `edge_id` | `local` |
| `updated_at` | `T0+5.1s` |

认领失败（已被别人抢走）则不再发 dispatch。

### 7.3 Blob：`jobs/<task_id>/job.json`（方案 A）

`PrepareJob` → `BuildJobPackage` → `Blob.Put`。

**BlobRef 返回值**（进入 dispatch）：

```json
{
  "key": "jobs/task-a1b2c3d4/job.json",
  "mime": "application/json",
  "size": 312
}
```

**job.json 内容（概念样例）**：

```json
{
  "task_id": "task-a1b2c3d4",
  "edge_id": "local",
  "workflow": {
    "1": {
      "class_type": "Stub",
      "inputs": { "text": "a red cat" }
    }
  },
  "images": [],
  "output_prefix": "outputs/task-a1b2c3d4"
}
```

说明：

- 非 image 字段已注入 `workflow` 节点。  
- image 字段放在 `images[]`，由执行面本机 `UploadImage` 后再写节点。  
- **执行面不读 Case/Task DB** 拼装图。

Prepare 失败会 **rollback** claim：`status` 回到 `pending`，`edge_id` 清空。

### 7.4 MQ：`dispatch.local`

| 字段 | 值 |
|---|---|
| Topic | `dispatch.local`（`TopicDispatch(edge_id)`） |
| Key | `task-a1b2c3d4` |

```json
{
  "task_id": "task-a1b2c3d4",
  "edge_id": "local",
  "input_prefix": "inputs/task-a1b2c3d4",
  "job_ref": {
    "key": "jobs/task-a1b2c3d4/job.json",
    "mime": "application/json",
    "size": 312
  }
}
```

- `input_prefix`：遗留字段；成功路径以 `job_ref` 为准。  
- **split**：Stream `q:dispatch.local`；Bot **不**订阅生产 dispatch，由 `edge-agent` 消费。  
- **allinone**：同进程 Worker 订阅并处理。

### 7.5 split 额外：在线心跳（非 Task 表）

Edge 每 5 秒本机探 Comfy，`POST /agent/v1/presence` `{ edge_id, comfy_running, hardware? }`，响应 `{ refresh_hardware }`。控制面内存记 `last_seen`；超过约 15 秒无报到视为工人掉线，界面 Comfy 也显示未启动。claim 长轮询与任务 heartbeat 只刷新 `last_seen`，不改 Comfy 状态。不落库、不用 Redis `edge:online:*`。

管理页 `GET /api/v1/edges/presence` 读这套内存，不是控制面 ping 画图机。

---

## 8. 阶段 F — Actuator 执行（Comfy + 产物）

**入口**：Worker `HandleDispatch`。

### 8.1 读 Blob job

`Blob.Get(job_ref)` → 解析 `JobPackage` →（若有）逐张 `Get` 图片并 `UploadImage` 到 Comfy。

### 8.2 Comfy 接口（HTTP）

| 调用 | 样例 |
|---|---|
| `Submit(graph)` | 调用 `/prompt` 并返回真实 `prompt_id` |
| `Wait(prompt_id)` | 轮询任务产物并返回 PNG 字节 |

业务**不以** Comfy history 为真相源。

### 8.3 MQ：`task.status` → `running`

```json
{
  "task_id": "task-a1b2c3d4",
  "edge_id": "local",
  "status": "running",
  "prompt_id": "prompt-run-001",
  "at": "2026-08-11T06:00:06Z"
}
```

### 8.4 Blob：产物

| Key | MIME | 内容 |
|---|---|---|
| `outputs/task-a1b2c3d4/0_out.png` | `image/png` | ComfyUI 生成的 PNG 字节 |

### 8.5 MQ：`task.status` → `succeeded`

```json
{
  "task_id": "task-a1b2c3d4",
  "edge_id": "local",
  "status": "succeeded",
  "prompt_id": "prompt-run-001",
  "outputs": [
    {
      "key": "outputs/task-a1b2c3d4/0_out.png",
      "mime": "image/png",
      "size": 4096
    }
  ],
  "at": "2026-08-11T06:00:07Z"
}
```

失败时发 `status=failed` + `error_code` / `error_msg`（如 `comfy_submit`），不写 outputs。

---

## 9. 阶段 G — Orchestrator 收敛 + 通知用户

**入口**：订阅 `task.status` → `OnStatus` / `applyStatus`。

### 9.1 `running` 回写

| 列 | 新值 |
|---|---|
| `status` | `running` |
| `prompt_id` | `prompt-run-001` |
| `updated_at` | event.`at` |

### 9.2 `succeeded` 回写

| 列 | 新值 |
|---|---|
| `status` | `succeeded` |
| `outputs_json` | 见下 |
| `updated_at` | event.`at` |

```json
[
  {
    "Key": "out-0",
    "Blob": {
      "key": "outputs/task-a1b2c3d4/0_out.png",
      "mime": "image/png",
      "size": 4096
    }
  }
]
```

（`OutputRef` 序列化字段名以 Go 结构为准。）

### 9.3 Notify（**不走 Queue topic**）

进程内 `notify.Publisher` → `tg/notifybridge` → `Adapter.HandleUserNotify`。

`UserNotify` 载荷：

```json
{
  "chat_id": 10001,
  "task_id": "task-a1b2c3d4",
  "kind": "task_succeeded",
  "outputs": [
    {
      "key": "outputs/task-a1b2c3d4/0_out.png",
      "mime": "image/png",
      "size": 4096
    }
  ]
}
```

`chat_id` 来源：Task 缓存或 `sessions` join。

### 9.4 Telegram Bot API

1. `sendPhoto`：读 Blob 首图，caption `✅ Case 完成\ntask=task-a1b2c3d4`  
2. `sendMessage` + 主菜单：引导再选 Case  

去重：内存 map（`task_id:kind`）；进程重启可能重复通知。

---

## 10. 终态快照（对照）

### `tasks` 终行

```text
id            = task-a1b2c3d4
session_id    = sess-20260811-001
case_id       = text2img-demo
status        = succeeded
edge_id       = local
prompt_id     = prompt-run-001
input_prefix  = inputs/task-a1b2c3d4
outputs_json  = [{"Key":"out-0","Blob":{"key":"outputs/task-a1b2c3d4/0_out.png",...}}]
```

### `sessions` 终行

```text
status = submitted
draft_json 仍含 prompt / skipped seed
```

### Blob 对象集合

```text
inputs/task-a1b2c3d4/prompt.txt
jobs/task-a1b2c3d4/job.json
outputs/task-a1b2c3d4/0_out.png
```

### MQ 消息序列（成功路径）

```text
task.created
dispatch.local
task.status (running)
task.status (succeeded)
```

---

## 11. 观测 HTTP（读路径，不改变主链路）

| API | 数据源 |
|---|---|
| `GET /api/v1/tasks` / `GET /api/v1/tasks/{id}` | `tasks`（可 join session 看 chat） |
| `POST /api/v1/tasks/{id}/cancel` | 仅 `pending`/`queued` → `cancelled` + notify |
| `GET /api/v1/sessions` | `sessions` |
| `GET /api/v1/cases` | `catalog_cases` |
| `GET /api/v1/edges/{id}/tasks` | `tasks WHERE edge_id=?` |
| `GET /api/v1/edges/{id}/system` | 该节点 Comfy `system_stats`（非 DB） |
| `GET /api/v1/edges/{id}/queue` | 该节点 Comfy `queue`（非 DB） |
| `GET /api/v1/edges/{id}/stats` | 该节点任务数 / 累计耗时 / 成功率 |

---

## 12. allinone vs split（同一业务数据，不同载体）

| 项 | allinone | split |
|---|---|---|
| Blob 驱动 | localfs（如 `data/blob/`） | S3 兼容或火山 TOS（同 key 空间） |
| Queue | Memory 同步 handler | Redis Streams `q:<topic>` |
| 谁消费 `dispatch.*` | Bot 同进程 Worker | `apps/edge-agent` |
| 实例可选条件 | `ListHealthy`（探活） | `ListEnabled` + `edge:online:<id>` |
| DB（SQLite） | **仅云/Bot 侧**；Edge **不写** Task/Case | 同左 |
| job / inputs / outputs | 同 key | 同 key（跨进程共享对象存储） |

业务表行与事件 JSON **形状相同**；变的是「文件存在哪台盘 / 消息进哪条 Stream」。

---

## 13. 状态机对照（数据视角）

**Session**

```text
collecting → confirming → submitted
             ↘ exited
```

**Task**

```text
pending → queued → running → succeeded
                           ↘ failed
         pending|queued ──► cancelled
```

谁写 Task：

| 迁移 | 写入方 |
|---|---|
| → `pending` | ConfirmRun |
| → `queued` | Orchestrator `ClaimQueued` |
| → `running` / 终态 | Orchestrator `OnStatus`（源自 Actuator 的 `task.status`） |
| → `cancelled` | Admin/API `RequestCancel`（限 pending/queued） |

---

## 14. 相关代码入口

| 步骤 | 路径 |
|---|---|
| StartCase / SubmitInput | `internal/packaging/botapp/facade.go` |
| ConfirmRun + stage | `internal/packaging/botapp/confirm_run.go` |
| 事件 DTO | `internal/sharedkernel/events.go` |
| 调度 / OnStatus / Notify | `internal/runtime/application/orchestrator/service.go` |
| PrepareJob / JobPackage | `internal/runtime/infrastructure/actuator/snapshot.go`、`job.go` |
| Worker | `internal/runtime/infrastructure/actuator/worker.go` |
| TG 适配 | `internal/channel/tg/adapter.go` |
| Edge 在线 | `internal/platform/presence/store.go` |
