# 工作流引擎核心 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Language:** zh-CN（Comet artifact language）

---
change: workflow-engine-core
design-doc: docs/superpowers/specs/2026-08-07-workflow-engine-core-design.md
base-ref: f1a5fa828da331c7e75ac4ca23f03d5a57969c23
---

> `base-ref` 为 main 初始提交。实现在分支 `feat/workflow-engine-core`。

**Goal:** 落地可组装的工作流核心：协议/注册、Session 锁、异步 Task、Orchestrator（独占写终态+对账+风暴防护）、Actuator、TG Adapter；Queue/Blob 端口一期用 Memory/LocalFS。

**Architecture:** DDD 限界上下文（Catalog / Conversation / Runtime / ChannelTG / Platform）；Monorepo 预留 `apps/admin-api` + `web/admin`。ConfirmRun 后 `task.created → dispatch → status → notify.user`。

**Tech Stack:** Go, GORM+SQLite, JSON Schema, go-telegram/bot, chi, slog, testify

---

## 文件结构（将创建）

按 DDD + Monorepo。详见  
`docs/openspec/changes/workflow-engine-core/.comet/handoff/project-layout.md`。

```text
apps/bot/cmd/comfyui-bot/main.go
apps/admin-api/README.md
web/admin/README.md
internal/sharedkernel/
internal/catalog/{domain,application,infrastructure}
internal/conversation/{domain,application,infrastructure}
internal/runtime/{domain,application/orchestrator,infrastructure/...}
internal/channel/tg/
internal/platform/{queue,blob,notify,instance}/
internal/packaging/botapp/
internal/identity/.gitkeep
configs/ data/ docs/ go.mod
```

---

### Task 1: 脚手架与端口（DDD 树）

**Files:**
- Create: `go.mod`, `apps/bot/cmd/comfyui-bot/main.go`
- Create: `apps/admin-api/README.md`, `web/admin/README.md`, `internal/identity/.gitkeep`
- Create: `internal/sharedkernel/*.go`
- Create: `internal/platform/queue/{port.go,memory/...}`, `internal/platform/blob/{port.go,localfs/...}`
- Create: bot config/db wire 于 `apps/bot` 或 `internal/platform`

- [x] 1.1 初始化 module、config、slog、chi `/healthz`、GORM SQLite 打开
- [x] 1.2 实现 queue Publisher/Subscriber（Memory）并单测 Publish→Subscribe
- [x] 1.3 实现 blob Put/Get（LocalFS）并单测
- [x] 1.4 `git init`（若尚未）+ 提交脚手架；更新本 plan `base-ref`
- [x] 1.5 GORM + glebarez SQLite + AutoMigrate

### Task 2: Protocol + Case Registry

**Files:**
- Create: `internal/catalog/domain/{document.go,repository.go}`
- Create: `internal/catalog/infrastructure/validation/`
- Create: `internal/catalog/infrastructure/persistence/`

- [x] 2.1 Case 文档模型 + JSON Schema 校验 + 媒体钩子；单测合法/非法/enum/skip
- [x] 2.2 GORM 仓储：Save/Get/List/Disable；集成测试（sqlite memory）
- [x] 2.3 提交

### Task 3: Session + Task 领域

**Files:**
- Create: `internal/domain/session/{session.go,service.go,service_test.go}`
- Create: `internal/domain/task/{task.go,repository.go,task_test.go}`

- [ ] 3.1 Session：Start/Submit/Skip/Exit/锁冲突；单测
- [ ] 3.2 Task：状态迁移方法 + 温和取消规则；单测非法边
- [ ] 3.3 提交

### Task 4: App ConfirmRun

**Files:**
- Create: `internal/app/facade.go`, `internal/app/confirm_run.go`, `internal/app/confirm_run_test.go`

- [ ] 4.1 Facade：菜单/Case/Session 用例委托
- [ ] 4.2 ConfirmRun：校验→物化 Blob→Create pending→Clear session→Publish `task.created`
- [ ] 4.3 单测（假 blob/queue/repos）
- [ ] 4.4 提交

### Task 5: Orchestrator

**Files:**
- Create: `internal/orchestrator/{service.go,apply_status.go,reconcile.go,storm.go,*_test.go}`
- Create: `internal/port/instance/registry.go`, `internal/port/notify/notify.go`

- [ ] 5.1 OnTaskCreated + SchedulePending；dispatch；MarkQueued
- [ ] 5.2 applyStatus 幂等 + notify.user
- [ ] 5.3 ReconcileStale + ExecutionQuery 补写
- [ ] 5.4 RateLimiter/Backoff/CircuitBreaker 接入调度与对账
- [ ] 5.5 RequestCancel（pending/queued）
- [ ] 5.6 单测后提交

### Task 6: Actuator + ComfyUI

**Files:**
- Create: `internal/comfyui/client.go`, `internal/actuator/{worker.go,ledger.go,query.go,*_test.go}`

- [ ] 6.1 Comfy client 接口 + mock 实现
- [ ] 6.2 HandleDispatch：注入、ledger、status running/终态
- [ ] 6.3 ExecutionQuery + status 重发路径
- [ ] 6.4 单测后提交

### Task 7: TG Adapter 与组装

**Files:**
- Create: `internal/adapter/tg/{bot.go,render.go,notify_handler.go}`
- Modify: `cmd/comfyui-tgbot/main.go`（wire all-in-one）
- Create: `configs/config.example.yaml`, `README.md`, 种子 Case JSON

- [ ] 7.1 Update 路由到 Facade；菜单/列表/锁拦截渲染
- [ ] 7.2 订阅 `notify.user` 发图/去重
- [ ] 7.3 main 组装：Memory queue 订阅 orchestrator/actuator/tg
- [ ] 7.4 README：配置、种子 Case、本地跑通说明
- [ ] 7.5 冒烟测试（mock Comfy）后提交

### Task 8: 验收对齐

- [ ] 8.1 对照 `tasks.md` 与 specs 勾验清单
- [ ] 8.2 确认：无强取消、无扣费、Actuator 不写 Task 终态、Orchestrator 有限流

---

## 执行注意

- 同进程也必须走 `queue.Port`，禁止 app 直接调 actuator。
- 所有 Task 终态写入只经 `orchestrator.applyStatus`。
- 实现语言与注释可用英文；**计划/Comet 产物保持 zh-CN**。
