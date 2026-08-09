# 数据模型与 ER

> 系统总览见 [overview.md](./overview.md)；运行时链路见 [runtime.md](./runtime.md)。  
> 数据库：SQLite（默认 `data/app.db`），GORM AutoMigrate。  
> 对应能力：User / Session / Task 落库 + 多 Comfy 实例池。

非库内状态（有意不落库）：

| 项 | 存放 | 说明 |
|---|---|---|
| 输入/输出文件 | `data/blob/` | Task 只存 `input_prefix` / outputs BlobRef |
| 事件总线 | 进程内 memory | `task.created` / `dispatch.*` / `task.status` |
| 熔断 / 健康缓存 / RR 游标 | 进程内存 | 可重建 |
| 通知去重 map | 进程内存 | 重启可能重复通知 |

---

## 1. 领域数据结构（内存/领域模型）

### 1.1 User（`internal/identity/domain`）

| 字段 | 类型 | 说明 |
|---|---|---|
| ID | string (UUID) | 内部主键 |
| TgUserID | int64 | Telegram `from.id`，唯一 |
| Username / FirstName / LastName | string | TG 资料 |
| LanguageCode | string | |
| IsBot / IsPremium | *bool | 可空 |
| LastSeenAt | time | upsert 刷新 |

### 1.2 Session（`internal/conversation/domain`）

| 字段 | 类型 | 说明 |
|---|---|---|
| ID | SessionID | |
| UserID | string | → users.id |
| ChatID | int64 | TG chat |
| CaseID | string | → catalog_cases.id（逻辑） |
| Status | collecting / confirming / submitted / exited | |
| CurrentInputIndex | int | |
| InputKeys | []string | 采集顺序 |
| Draft | map[key]DraftValue | 草稿（文本/数/图 Blob 等） |

生命周期：**不含** Task 执行；确认后 → `submitted`，行长期保留。

### 1.3 Task（`internal/runtime/domain`）

| 字段 | 类型 | 说明 |
|---|---|---|
| ID | TaskID | |
| SessionID | SessionID | **必填** → sessions.id |
| ChatID | ChatID | 可选缓存，**非**必需落库列 |
| CaseID | CaseID | 逻辑关联 Case |
| Status | pending→queued→running→终态 | 执行态唯一真相源 |
| InstanceID | InstanceID | 派发后写入 |
| PromptID | string | Comfy prompt id |
| InputPrefix | string | blob 路径前缀 |
| Outputs | []OutputRef | 产物 Blob |
| ErrorCode / ErrorMessage | string | |

### 1.4 Comfy Instance Record（`internal/platform/instance`）

| 字段 | 类型 | 说明 |
|---|---|---|
| ID | InstanceID | 稳定字符串，如 `local` / `gpu-1` |
| BaseURL | string | Comfy HTTP 根 |
| Enabled | bool | |
| Capabilities | []string | 可选标签 |

### 1.5 Case Document（存在 `catalog_cases.doc_json`）

顶层：`id/name/description/.../inputs/outputs/bindings/input_schema`。  
`bindings.workflow` = **整份 Comfy API workflow JSON**；`bindings.inputs/outputs` = 字段↔节点映射。

---

## 2. 数据库表结构

### 2.1 `users`

| 列 | 约束 | 说明 |
|---|---|---|
| id | PK | UUID |
| tg_user_id | UNIQUE NOT NULL | Telegram user id |
| username, first_name, last_name | | |
| language_code | | |
| is_bot, is_premium | nullable bool | |
| last_seen_at | NOT NULL | |
| created_at, updated_at | | |

### 2.2 `sessions`

| 列 | 约束 | 说明 |
|---|---|---|
| id | PK | |
| user_id | NOT NULL, index | → users.id（应用层关联） |
| chat_id | NOT NULL, index | |
| case_id | NOT NULL | → catalog_cases.id（逻辑） |
| status | NOT NULL | |
| current_input_index | NOT NULL | |
| input_keys_json | TEXT | |
| draft_json | TEXT | |
| created_at, updated_at | NOT NULL | |

### 2.3 `tasks`

| 列 | 约束 | 说明 |
|---|---|---|
| id | PK | |
| session_id | NOT NULL, index | → sessions.id |
| case_id | NOT NULL | |
| status | NOT NULL, index | |
| instance_id | index, 可空 | → comfy_instances.id（派发后） |
| prompt_id | 可空 | |
| input_prefix | NOT NULL | |
| outputs_json | TEXT | |
| error_code, error_message | | |
| created_at, updated_at | NOT NULL | |

### 2.4 `comfy_instances`

| 列 | 约束 | 说明 |
|---|---|---|
| id | PK | |
| base_url | NOT NULL | |
| enabled | NOT NULL | |
| capabilities_json | TEXT | |
| created_at, updated_at | NOT NULL | |

### 2.5 `catalog_cases`（既有）

| 列 | 约束 | 说明 |
|---|---|---|
| id | PK | Case id |
| name | NOT NULL | |
| menu_key | index | |
| tags_json / cats_json | | 冗余索引用 |
| doc_json | NOT NULL | 完整 CaseDocument（含 workflow） |
| enabled | index | |
| created_at, updated_at | | |

### 2.6 TG 主键盘树（`tg_menus` / `tg_menu_items` / `tg_menu_item_cases`）

Telegram 主 ReplyKeyboard 配置真相源，**关系型树**（非 JSON 文档）。当前单 Bot、单菜单：`tg_menus.id` 固定为 `default`，`bot_id` 固定为 `default`。

#### `tg_menus`

| 列 | 约束 | 说明 |
|---|---|---|
| id | PK | 菜单文档 id（当前仅 `default`） |
| bot_id | NOT NULL | Bot 标识（当前仅 `default`） |
| updated_at | NOT NULL | UTC |

#### `tg_menu_items`

扁平存储树节点；`parent_id` 为空表示根级（ReplyKeyboard 行）。

| 列 | 约束 | 说明 |
|---|---|---|
| id | PK | 节点 id（如 `btn-image`） |
| menu_id | NOT NULL, index | → `tg_menus.id` |
| parent_id | index, 可空 | → 父节点 `tg_menu_items.id` |
| label | NOT NULL | 按钮文案 |
| row, col | NOT NULL | 根级 ReplyKeyboard 布局（子节点可忽略） |
| enabled | NOT NULL | |
| kind | NOT NULL | `folder` / `open_case` / `list_cases_by_tag` / `placeholder` / `reply_media` |
| placeholder_text | | `placeholder` 提示 |
| intro_text | TEXT | 仅 `folder`：进层消息正文（空则用 `label`） |
| tag | | `list_cases_by_tag` 分区 tag |
| reply_json | TEXT | `reply_media` 的 `{text, images[]}` |

`folder` 子树通过 `parent_id` 表达；`open_case` / `folder` 挂载的 Case 见关联表。

#### `tg_menu_item_cases`

节点 ↔ Case 多对多（有序）；主要用于 `folder` 内 Case 列表与 `open_case` 单 Case。

| 列 | 约束 | 说明 |
|---|---|---|
| menu_item_id | PK (复合) | → `tg_menu_items.id` |
| case_id | PK (复合) | → `catalog_cases.id`（逻辑） |
| sort | NOT NULL | 同节点内排序 |

#### 种子与迁移

- **空库**：`EnsureDefault` 写入 `DefaultSeedTree`（六键根菜单；「🖼 图片」为 `folder`，并挂上 tag=`image` 的 Case id 列表）。
- **旧库**：若关系表为空且存在遗留表 `tg_menu_configs`（`items_json`），一次性导入为扁平根节点后弃用 JSON 表。
- **写入**：`ReplaceTree` 事务整树替换（删旧 items + cases 再插入）。

> SQLite **未声明物理外键**；关联由应用层保证（GORM 默认不强制 FK）。

---

## 3. 表关系 ER 图

```mermaid
erDiagram
  users ||--o{ sessions : "user_id"
  sessions ||--o{ tasks : "session_id"
  catalog_cases ||--o{ sessions : "case_id (逻辑)"
  catalog_cases ||--o{ tasks : "case_id (逻辑)"
  comfy_instances ||--o{ tasks : "instance_id (派发后)"
  tg_menus ||--o{ tg_menu_items : "menu_id"
  tg_menu_items }o--o| tg_menu_items : "parent_id"
  tg_menu_items ||--o{ tg_menu_item_cases : "menu_item_id"
  catalog_cases ||--o{ tg_menu_item_cases : "case_id (逻辑)"

  users {
    string id PK
    int64 tg_user_id UK
    string username
    string first_name
    string last_name
    string language_code
    bool is_bot
    bool is_premium
    datetime last_seen_at
  }

  sessions {
    string id PK
    string user_id FK
    int64 chat_id
    string case_id
    string status
    int current_input_index
    text input_keys_json
    text draft_json
  }

  tasks {
    string id PK
    string session_id FK
    string case_id
    string status
    string instance_id FK
    string prompt_id
    string input_prefix
    text outputs_json
  }

  comfy_instances {
    string id PK
    string base_url
    bool enabled
    text capabilities_json
  }

  catalog_cases {
    string id PK
    string name
    string menu_key
    text doc_json
    bool enabled
  }

  tg_menus {
    string id PK
    string bot_id
    datetime updated_at
  }

  tg_menu_items {
    string id PK
    string menu_id FK
    string parent_id FK
    string label
    text intro_text
    int row
    int col
    bool enabled
    string kind
    text reply_json
  }

  tg_menu_item_cases {
    string menu_item_id PK
    string case_id PK
    int sort
  }
```

读法：

- **强业务链**：`users` ← `sessions` ← `tasks`
- **Case**：Session/Task 用字符串 `case_id` 指向目录
- **实例**：仅在 Task `queued+` 后写入 `instance_id`
- **TG 主菜单**：`tg_menus` + `tg_menu_items` 存树；`tg_menu_item_cases` 挂 Case；folder 子级用 `parent_id`

---

## 4. 实例关系 ER / 运行关系图

「实例」既指 DB 中的 `comfy_instances`，也指进程内 Pool 客户端。

```mermaid
erDiagram
  comfy_instances ||--o| PoolClient : "id → HTTP/Mock Client"
  comfy_instances ||--o{ tasks : "instance_id"
  PoolClient ||--o{ ComfyUI_Remote : "base_url"

  comfy_instances {
    string id PK
    string base_url
    bool enabled
  }

  tasks {
    string id PK
    string instance_id
    string status
    string prompt_id
  }

  PoolClient {
    string instance_id
    bool healthy_memory
    Client comfy_client
  }

  ComfyUI_Remote {
    string host
    endpoint system_stats
    endpoint queue
    endpoint prompt
  }
```

调度关系（非表，运行时）：

```mermaid
flowchart LR
  T[tasks pending] --> O[Orchestrator ClaimQueued]
  O --> H[Pool ListHealthy ∩ 熔断]
  H -->|round-robin| I[comfy_instances.id]
  I --> D[dispatch.instance_id]
  D --> W[Worker clientFor]
  W --> C[Comfy Submit/Wait/Upload]
  C --> S[task.status → 写回 tasks]
```

观测 API（只读）：

| 路径 | 数据源 |
|---|---|
| `GET /api/v1/comfy-instances` | `comfy_instances` |
| `GET .../{id}/system` | 该实例 Comfy `/system_stats` |
| `GET .../{id}/queue` | 该实例 Comfy `/queue` |
| `GET .../{id}/tasks` | `tasks WHERE instance_id=?` |

---

## 5. 状态机（简）

**Session：** `collecting` → `confirming` → `submitted` | `exited`

**Task：** `pending` → `queued`（写 instance_id）→ `running`（写 prompt_id）→ `succeeded` | `failed` | `cancelled`

完整调度/事件语义见 [runtime.md](./runtime.md)。

---

## 6. 相关代码入口

| 表 / 能力 | 包路径 |
|---|---|
| users | `internal/identity/infrastructure/persistence` |
| sessions | `internal/conversation/infrastructure/persistence` |
| tasks | `internal/runtime/infrastructure/persistence` |
| comfy_instances / Pool | `internal/platform/instance` |
| catalog_cases | `internal/catalog/infrastructure/persistence` |
| tg_menus / tg_menu_items / tg_menu_item_cases | `internal/tgmenu`（domain/application/persistence）；HTTP `internal/httpapi/tgmenu`；Case 反查 `GET .../cases/{id}/menu-placements` |
| HTTP API | `internal/httpapi/comfyinstances` 等 |
| 接线 | `apps/bot/cmd/comfyui-bot/main.go`、`apps/admin-api/cmd/admin-api/main.go` |

设计原文：`docs/superpowers/specs/2026-08-08-comfy-multi-instance-design.md`
