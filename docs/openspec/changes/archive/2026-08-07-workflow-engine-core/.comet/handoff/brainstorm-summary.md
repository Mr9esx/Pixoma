# Brainstorm Summary

- Change: workflow-engine-core
- Date: 2026-08-07
- Status: **CONFIRMED** — 用户确认架构冻结并恢复 Comet Design Doc

## Confirmed Technical Approach

- 领域模块可组装：按 **DDD 限界上下文** Catalog / Conversation / Runtime / ChannelTG / Platform
- Monorepo：`apps/bot`（本期）；预留 `apps/admin-api` + `web/admin`；Admin 复用 Catalog/Runtime，禁止依赖 TG
- 每 BC：`domain` / `application` / `infrastructure`；跨 BC 用例在 `packaging/botapp`
- 部署：模块组合优先；可 all-in-one wire，亦可按模块拆进程
- Bot（渠道）| Orchestrator（任务控制面）| Actuator@ComfyUI（执行面）
- Task 终态仅 Orchestrator 写库；Actuator 只发 status + 本地 ledger + ExecutionQuery 对账
- ConfirmRun 后：task.created → dispatch → status → notify.user（同进程走 Queue Port/Memory）
- 双触发：task.created + SchedulePending；status 丢失靠超时扫库 + Query 对账
- 风暴防护：限流、退避+jitter、熔断、分池、redispatch 令牌桶、notify 去重
- 协议校验：JSON Schema + 媒体/OSS 扩展
- TG：go-telegram/bot；Session 锁；温和取消；生成中可开新 Case
- DB：GORM+SQLite→MySQL；Blob/Queue 端口化

## Key Trade-offs and Risks

- Orchestrator 汇聚点易成重试风暴 → 强制配额与熔断
- status 终态与对账双路径 → 统一 applyStatus 幂等入口
- Open 阶段曾写「不做 TG」→ 架构讨论后一期含 TG Adapter（已确认）

## Testing Strategy

- 协议/JSON Schema 单元测试
- Session 锁与状态迁移单测
- Task 状态机与 applyStatus 幂等单测
- Orchestrator：调度、对账、熔断、限流单测（假 Queue/假 Actuator Query）
- Actuator：注入与 status 发布（假 ComfyUI）
- 集成：Memory Queue all-in-one 走通 text2img；TG 可用 mock

## Spec Patches

- 更新 proposal 范围：纳入 TG Adapter / Session / Orchestrator 异步模型
- workflow-protocol：JSON Schema 校验
- comfyui-executor：改为 Actuator 语义（status 上报，不独占写 Task）
- 新增 dialog-session、task-orchestrator、channel-tg 能力 specs
- 补充边界：温和取消、风暴防护、对账
