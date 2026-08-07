# 验证报告：workflow-engine-core

- Change: `workflow-engine-core`
- Date: 2026-08-07
- Mode: **full**（tasks=30，delta specs=6，changed files≈60）
- Branch: `feat/workflow-engine-core`
- Language: zh-CN

## 结论

**PASS** — 一期可组装工作流核心已落地；Memory all-in-one + mock Comfy + TG 主路径可跑通。无 CRITICAL / IMPORTANT 阻断项。

## 检查项

| # | 检查 | 结果 | 证据 |
|---|------|------|------|
| 1 | tasks.md 全部 `[x]` | PASS | `docs/openspec/changes/workflow-engine-core/tasks.md` |
| 2 | Implementation Plan 全部 `[x]` | PASS | `docs/superpowers/plans/2026-08-07-workflow-engine-core.md` |
| 3 | 构建与测试 | PASS | `go test ./... && go build ./apps/bot/cmd/comfyui-bot`（已 `record-check`） |
| 4 | 对齐 Design Doc 高层决策 | PASS | DDD BC、事件链、Orchestrator 独占终态、温和取消、Memory/LocalFS |
| 5 | 对齐 proposal 目标 | PASS | 协议/注册/Session/异步 Task/Orchestrator/Actuator/TG；无扣费/强取消 |
| 6 | Spec 场景覆盖（能力级） | PASS | 见下表；由单测 + 集成冒烟 + 真机 TG mock 覆盖主场景 |
| 7 | 安全扫描 | PASS | Token 仅 `.env`（gitignore）；仓库无硬编码 Bot Token |
| 8 | Standard 评审（正确性/安全/边界） | PASS | 见「评审摘要」 |

## Spec 能力覆盖摘要

| Capability | 覆盖情况 |
|------------|----------|
| workflow-protocol | JSON Schema + 媒体钩子单测 |
| workflow-registry | GORM Save/Get/List/Disable 集成测 |
| dialog-session | Start/锁/Exit/Confirm 解锁单测 + TG 冲突文案 |
| task-orchestrator | 调度/applyStatus 幂等/对账/熔断/取消单测；主进程定时 Schedule+Reconcile |
| comfyui-executor | Mock Actuator ledger/status/QueryAdapter；**真 HTTP Comfy 未接（一期 mock 可接受）** |
| channel-tg | 菜单/预览/填表/确认/发图；notify 去重 |

## 验收边界确认

- 无强取消（running 不可 cancel）— 单测覆盖
- 无扣费 — 价格仅为元数据
- Actuator 不写 Task 终态 — 只发 status；Orchestrator `applyStatus` 写库
- Orchestrator 有限流/熔断/分池 — `StormGuard` + `NextBackoff`

## 已知非阻断偏差（WARNING）

1. ComfyUI 为 Mock PNG，非真实 HTTP client（proposal 允许 mock 冒烟）
2. Session/Task 运行时仓储仍为 Memory；Catalog 为 GORM（可后续统一持久化）
3. 视频/充值/签到/个人中心为菜单占位
4. TG ReplyKeyboard 需消息下发刷新（产品约束，已提供 `/menu` 与多入口刷新）

## 评审摘要（standard）

- 正确性：ConfirmRun→事件→终态→notify 链路一致；同步 Memory 队列下确认提示顺序已修正
- 安全：无密钥入库；Blob 路径有 cleanKey
- 边界：Session 锁冲突三按钮、终态 notify 去重、对账走同一 `applyStatus`

## 命令证据

```text
go test ./...
go build -o /tmp/comfyui-bot ./apps/bot/cmd/comfyui-bot
# comet state record-check … exit=0
```
