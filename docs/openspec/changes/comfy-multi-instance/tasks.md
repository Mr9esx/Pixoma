## 1. User / Session / Task 持久化

- [x] 1.1 新增 `users` 表与仓储；TG 路径按 tg_user_id upsert（含 From 全量可得字段与 last_seen_at）
- [x] 1.2 新增 `sessions` 表与 GORM 仓储；替换 Memory；含 user_id + chat_id；活跃查询；长期保留
- [x] 1.3 新增 `tasks` 表与 GORM 仓储；替换 Memory；必填 session_id；支持 ListByInstance / 经 Session 列「我的任务」
- [x] 1.4 ConfirmRun 与通知路径：创建 Task 写 session_id；回图 join Session.chat_id；补测试
- [x] 1.5 去掉 Actuator Ledger：对账/GetRun 只读 Task；删除 MemoryLedger 接线与依赖

## 2. 实例池与观测

- [x] 2.1 `comfy_instances` 表 + Repository CRUD；启动种子 upsert；刷新客户端池
- [x] 2.2 HTTP `/api/v1/comfy-instances` CRUD
- [x] 2.3 Comfy Client：`SystemStats` / `Queue`（Mock 占位）
- [x] 2.4 `GET .../{id}/system`、`.../queue`、`.../tasks`；不可达明确错误；补测试

## 3. 健康、选路与执行

- [x] 3.1 enabled 实例周期健康探测；ListHealthy 过滤
- [x] 3.2 Orchestrator round-robin；无可用实例不投递
- [x] 3.3 Actuator 按 InstanceID 选用客户端；`main` 接线 AutoMigrate

## 4. Make 与文档

- [x] 4.1 Makefile：build / test / run / run-mock
- [x] 4.2 README：架构、种子、CRUD、system/queue/tasks curl、Mock、无鉴权警示
- [x] 4.3 `configs/bot.example.yaml` 注释更新

## 5. 验收

- [x] 5.1 相关 `go test` 通过
- [x] 5.2 按 README 可管理实例、查 system/queue、查实例任务；重启后 Session/Task/User 仍在
