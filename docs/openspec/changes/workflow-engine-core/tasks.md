## 1. 工程与端口

- [x] 1.1 初始化 Go module、`cmd` wire、config/slog、chi health
- [x] 1.2 实现 `port/queue` + Memory 适配器
- [x] 1.3 实现 `port/blob` + LocalFS 适配器
- [x] 1.4 实现 `port/notify`、`port/instance`（一期单实例注册表）
- [x] 1.5 GORM + SQLite 连接与 AutoMigrate 骨架

## 2. 协议与注册

- [x] 2.1 Case 文档模型与 JSON Schema 校验 + 媒体钩子
- [x] 2.2 Case Registry 仓储（CRUD、按 tag/菜单过滤、停用）
- [x] 2.3 协议与仓储单元/集成测试

## 3. Session 与 Task 领域

- [x] 3.1 Dialog Session 状态机与按 chat_id 锁
- [x] 3.2 Task 聚合与合法迁移（含温和取消）
- [x] 3.3 领域单测（锁、非法边、幂等前提）

## 4. App 与 ConfirmRun

- [x] 4.1 app.Facade：菜单/Case/Session/ConfirmRun/ListTasks
- [x] 4.2 ConfirmRun：校验、物化 Blob、建 pending、发 `task.created`、解锁 Session
- [ ] 4.3 DeliverNotify 用例入口（供 TG 适配）

## 5. Orchestrator

- [x] 5.1 OnTaskCreated + SchedulePending 双触发调度
- [x] 5.2 applyStatus 幂等写库 + 发 notify.user
- [x] 5.3 超时对账 + ExecutionQuery
- [x] 5.4 风暴防护：限流、退避+jitter、熔断、分池、redispatch 配额
- [x] 5.5 温和取消（pending/queued）
- [x] 5.6 Orchestrator 单测（假 Queue/Query）

## 6. Actuator 与 ComfyUI

- [x] 6.1 ComfyUI HTTP client（submit/wait/fetch）
- [x] 6.2 HandleDispatch：注入、执行、ledger、发 status
- [x] 6.3 ExecutionQuery 与 status 重发
- [x] 6.4 假 ComfyUI 单测 + 可选真机联调

## 7. TG Adapter

- [ ] 7.1 go-telegram/bot 接入与 Update 路由
- [ ] 7.2 菜单/分类/Case/会话/锁拦截渲染
- [ ] 7.3 消费 notify.user 发结果（去重）
- [ ] 7.4 样例 Case 种子与 README 跑通说明

## 8. 验收

- [x] 8.1 Memory all-in-one 冒烟：text2img 路径（可 mock Comfy）
- [ ] 8.2 对照 specs 清单勾验；确认无强取消/扣费/Actuator 直写终态
