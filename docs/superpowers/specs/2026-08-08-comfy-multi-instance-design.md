---
comet_change: comfy-multi-instance
role: technical-design
canonical_spec: openspec
---

# comfy-multi-instance 技术设计

## 1. 目标

在同一 bot 进程内：

1. 多台 ComfyUI：实例落库 CRUD、健康探测、客户端池、round-robin 选路  
2. 观测：按实例查 Comfy **system** / **queue**；按实例查本系统 **Task**  
3. 持久化底座：User、Session、Task 落 SQLite，并理清关联  

非目标：管理后台 UI、强鉴权 RBAC、清队列/interrupt、拆 `catalog_cases` 的 bindings、跨机多 Worker 进程、独立 Actuator **Ledger**（执行态只信 Task 表）。

## 2. 现状基线

| 项 | 现状 |
|---|---|
| SQLite | 仅 `catalog_cases`（`doc_json` 含完整 Case + `bindings.workflow`） |
| Task / Session | 内存仓储 |
| 用户 | 仅用 `chat_id`，不存 TG From |
| Comfy | 单一客户端；无 SystemStats/Queue；无真实健康探测 |
| HTTP | 仅 `/healthz` |
| 选路 | ListHealthy 取第一个 + 熔断 |

## 3. 数据模型

```text
users
  id (UUID PK)
  tg_user_id (UNIQUE NOT NULL)
  username, first_name, last_name, language_code
  is_bot, is_premium, …（From 可得字段，可空）
  last_seen_at, created_at, updated_at

sessions
  id (PK)
  user_id → users.id
  chat_id
  case_id, status, current_input_index
  input_keys_json, draft_json
  created_at, updated_at
  索引：(chat_id) 查活跃；user_id

tasks
  id (PK)
  session_id → sessions.id   -- 必填；不冗余 user/chat
  case_id
  status
  instance_id（派发后）
  prompt_id
  input_prefix
  outputs_json, error_code, error_message
  created_at, updated_at
  索引：instance_id, status, session_id

comfy_instances
  id (PK, 稳定字符串如 gpu-1)
  base_url, enabled
  capabilities_json（可选）
  created_at, updated_at
```

### 3.1 生命周期

- **Session**：collecting → confirming → submitted | exited。填表结束；**不包含** Task 执行阶段。  
- **Task**：确认时创建并写入 `session_id`；pending → queued → running → 终态。  
- 通知回图：`Task` → join `Session.chat_id`（及 User）。  
- Session 行长期保留；禁止因「活跃结束」物理删除导致 Task 断链。

### 3.2 User upsert

TG 消息/回调查路径：从 `From` 取资料，按 `tg_user_id` upsert，刷新 `last_seen_at` 与可变字段；返回内部 `user_id` 供 Session 使用。

## 4. HTTP API（bot :8080）

| 方法 | 路径 | 数据源 |
|---|---|---|
| CRUD | `/api/v1/comfy-instances` | DB |
| GET | `/api/v1/comfy-instances/{id}/system` | Comfy `GET /system_stats` |
| GET | `/api/v1/comfy-instances/{id}/queue` | Comfy `GET /queue`（或等价只读） |
| GET | `/api/v1/comfy-instances/{id}/tasks` | DB Task `WHERE instance_id=?`（分页/状态过滤） |

- system/queue 不可达：明确错误或 `reachable=false`，禁止伪造空成功。  
- Mock：返回标明 `mock: true` 的占位。  
- 不用 Comfy `/history` 充当业务任务史。  
- 本期无鉴权：README 注明本机/内网。

## 5. 运行时接线

1. **实例仓储 + Registry 适配**：DB 为真相源；写后刷新进程内客户端 map；启动用 `comfy_instances` 或单 `comfyui_base_url` upsert。  
2. **健康**：对 enabled 真实实例周期请求 `/system_stats`；失败剔出 ListHealthy。  
3. **Orchestrator**：健康 ∩ 熔断允许上 **round-robin**；无实例则保持 pending。  
4. **Actuator**：按 dispatch `InstanceID` 取客户端；执行进度经 status 事件由 Orchestrator 写回 **Task**（prompt_id / status / outputs / error）。  
5. **去掉 Ledger**：删除 `MemoryLedger` / `LocalRun` 作为执行真相源；Orchestrator 对账/查询执行态 MUST 读 Task 仓储，MUST NOT 依赖独立 Ledger。`GetRun` 若保留，仅为 Task 的只读适配。  
6. **Comfy Client** 扩展：`SystemStats(ctx)`、`Queue(ctx)`（Mock 实现）。  
7. **ConfirmRun**：确保 Session 已绑定 `user_id`；创建 Task 时写 `session_id`；`ListMyTasks` 经 Session.chat_id / user 关联查询。  
8. **main**：AutoMigrate 新表；替换 Memory Task/Session 仓储为 GORM；不再注入 Ledger。

## 6. 测试策略

- User upsert 幂等与字段刷新  
- Session 持久化与按 chat 活跃查询  
- ConfirmRun → Task.session_id；join 得 chat_id  
- Instance CRUD、客户端池刷新  
- system/queue Mock 与不可达  
- ListByInstance 不含无 instance 的 pending  
- Round-robin 与不健康过滤  
- 无 Ledger：对账/GetRun 仅依赖 Task  
 

## 7. Make / 文档

- Makefile：`build` / `test` / `run` / `run-mock`  
- README：架构、种子、CRUD、system/queue/tasks curl、Mock、无鉴权警示  
- `configs/bot.example.yaml` 注释  

## 8. Spec Patch 清单

- 更新 `comfy-instance-pool`：system/queue 分路由  
- 新增 `user-directory`、`session-persistence`、`task-persistence`  
- 同步 change 内 proposal / open design / tasks  
