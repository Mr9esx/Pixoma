## Why

当前 bot 只连单一 Comfy、Task/Session 仅内存、不存 TG 用户。需要：同进程多 Comfy（落库 CRUD + 健康选路）、分接口观测 system/queue、按实例查本系统 Task，以及 User/Session/Task 持久化与关联，支撑重启可恢复与运维查询。

## What Changes

- `comfy_instances` 落库 + CRUD；健康探测；客户端池；round-robin
- `GET .../system`、`GET .../queue`、`GET .../tasks`
- **User** 表（内部 UUID + 唯一 tg_user_id，资料 upsert）
- **Session** 落库（user_id + chat_id；长期保留）
- **Task** 落库（只挂 session_id；经 Session join 用户/聊天）
- Makefile + README；保留 `comfy_mock`
- 不拆 catalog_cases bindings；不清队列/interrupt；无管理后台
- **去掉 Actuator Ledger**；执行态只信 Task

## Capabilities

### New Capabilities

- `comfy-instance-pool`：实例 CRUD、健康、客户端池、system/queue/tasks 观测
- `user-directory`：TG 用户持久化与 upsert
- `session-persistence`：Session 落库与 User/Chat 关联
- `task-persistence`：Task 落库与 session_id 关联

### Modified Capabilities

- `task-orchestrator`：round-robin 选路
- `comfyui-executor`：按 InstanceID 选用客户端

## Impact

- 代码：仓储、TG upsert、ConfirmRun 关联、API、选路、main 接线
- 文档：Makefile、README
- 非目标：拆 doc_json bindings、强鉴权、多机 Worker、Comfy history 当业务史
