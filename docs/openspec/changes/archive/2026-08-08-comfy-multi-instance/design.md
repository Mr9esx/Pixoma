## Context

单实例静态注册 + 内存 Task/Session 不足以支撑多机 Comfy 与运维观测；亦无法重启恢复填表/工单，且无 TG 用户档案。

## Goals / Non-Goals

**Goals**

- 实例落库 CRUD、健康 + round-robin、客户端池
- system / queue 分接口；按实例 DB tasks
- User / Session / Task 落库；Task→Session→User
- Makefile / README

**Non-Goals**

- 拆 catalog_cases bindings
- Comfy `/history` 当业务史；清队列/interrupt
- 管理后台、强 RBAC、多机 Worker
- 独立 Actuator Ledger（执行态只信 Task）

## Decisions

1. 同进程多 HTTP 客户端  
2. DB 为实例真相源；YAML/单 URL 种子 upsert  
3. User：内部 UUID + 唯一 `tg_user_id`；From 全量可得字段 + upsert  
4. Session：`user_id` + `chat_id`；长期保留  
5. Task：只挂 `session_id`；不冗余 user/chat  
6. 观测：`/system`、`/queue`、`/tasks` 三分  
7. 健康用 system_stats；选路 round-robin  
8. **去掉 Ledger**：执行/对账只信持久化 Task  

## Risks / Trade-offs

- [范围大] → 设计已确认同 change 交付  
- [通知需 join] → ConfirmRun/通知路径统一经 Session  
- [Session 删除] → 禁止断链物理删  

## Migration Plan

- AutoMigrate 新表；Memory 仓储换 GORM  
- 空库实例种子；旧内存数据不迁移  

## Open Questions

（无阻塞）
