# Brainstorm Summary

- Change: comfy-multi-instance
- Date: 2026-08-08

## 确认的技术方案

### 持久化

- 保留 `catalog_cases.doc_json`（含 bindings.workflow）；**不拆** bindings
- **`users`**：内部 UUID PK；唯一 `tg_user_id`；TG From 可得字段 + upsert / `last_seen_at`
- **`sessions`**：`user_id` + `chat_id`；行长期保留
- **`tasks`**：必填 `session_id`；不冗余 user/chat；含 instance_id/prompt_id/状态/产物
- **`comfy_instances`**：CRUD
- **去掉 Ledger**：执行/对账只信 Task；删除 MemoryLedger

### 观测与调度

- CRUD + `system` / `queue` / `tasks` 分接口
- 健康 + round-robin；`comfy_mock` 占位
- Makefile + README

### 生命周期

- Session = 填表；Task = 工单；Task→Session→User

## 关键取舍与风险

- 范围含 User/Session/Task + 多实例 + 去 Ledger
- 回图 join Session；Session 不可随意删

## 测试策略

- 仓储/API；ConfirmRun session_id；无 Ledger 对账；Mock system/queue；选路

## Spec Patch

- task-persistence / comfyui-executor：Task 为执行态唯一真相源
- Design Doc / tasks / plan 已同步
