# Pixoma 引导式部署（大爆炸）Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 交付零配置 `pixoma` + 向导落库 + Edge 长轮询派发，默认路径去掉 Redis/queue/allinone-split。

**Architecture:** 控制面一体进程托管调度、管理 API、（发布）静态后台与 Agent API；任务经 DB 可领取态由 `pixoma-edge-agent` 长轮询 claim；本机自动拉起 Edge；业务库支持 SQLite/MySQL/Postgres。

**Tech Stack:** Go、GORM、现有 bot/admin/edge 包、web/admin（Vite）embed、火山/S3 blob 适配器（远程）。

## Global Constraints

- Language: zh-CN 产物与用户文档
- Change: `pixoma-guided-deploy`；Design Doc: `docs/superpowers/specs/2026-08-12-pixoma-guided-deploy-design.md`
- 方案 A 大爆炸：默认路径不保留 Redis 跨进程双轨
- Edge 共享 Agent Token；本机自动拉起 Edge；配置重启生效
- Comfy mock 开关必须保持端到端可用
- TDD：每任务先红后绿（若用户选 tdd）

---
change: pixoma-guided-deploy
design-doc: docs/superpowers/specs/2026-08-12-pixoma-guided-deploy-design.md
base-ref: 3a0518564cee9bf9d9305caf6c44c190717af8d2
---

## 文件地图（预期）

| 区域 | 路径 |
|---|---|
| 引导态 | `internal/platform/bootstrap/` |
| Settings | `internal/platform/settings/` |
| Agent API | `internal/httpapi/agent/` 或 `apps/...` |
| Task lease | `internal/runtime/...` Task 模型与 repo |
| pixoma 入口 | `apps/pixoma/cmd/pixoma/` |
| Edge | `apps/edge-agent` → 产品名 `pixoma-edge-agent` |
| 向导 UI | `web/admin` setup/login routes |
| 文档 | `README.md`、`docs/architecture/*` |

---

### Task 1: Bootstrap + 管理员门闩

**Files:**
- Create: `internal/platform/bootstrap/*.go`
- Modify: admin HTTP 鉴权中间件 / 门闩

- [x] **Step 1: 写失败测试** — 未初始化时业务 API 拒绝；bootstrap 可写默认管理员
- [x] **Step 2: 跑测试确认失败**
- [x] **Step 3: 最小实现** bootstrap 库、默认账密、initialized 标志
- [x] **Step 4: 跑测试通过**
- [x] **Step 5: Commit**

### Task 2: Task 可领取态 + lease + Agent claim API

**Files:**
- Modify: `internal/runtime/domain`、`persistence/gorm_task.go`
- Create: Agent claim/heartbeat/status handlers
- Modify: Orchestrator 调度：写 `job_ref` + queued，不再 Publish Redis

- [x] **Step 1: 写失败测试** — claim 原子性、lease 过期可再领、错 Token 401
- [x] **Step 2: 跑测试确认失败**
- [x] **Step 3: 最小实现** 字段迁移、claim API、调度改可领取
- [x] **Step 4: 跑测试通过**
- [x] **Step 5: Commit**

### Task 3: pixoma-edge-agent 拉取循环

**Files:**
- Modify: `apps/edge-agent`（或新 cmd 名）
- Remove/停用：默认 Redis subscribe 派发路径

- [x] **Step 1: 写失败测试** — mock 控制面 claim→执行→status（可用 httptest）
- [x] **Step 2: 跑测试确认失败**
- [x] **Step 3: 实现** 长轮询客户端 + Comfy mock/真机 + blob
- [x] **Step 4: 跑测试通过**
- [x] **Step 5: Commit**

### Task 4: pixoma 一体入口 + 本机自动拉起 Edge

**Files:**
- Create: `apps/pixoma/cmd/pixoma`
- Embed 或组装 admin routes；spawn child edge

- [x] **Step 1: 写失败测试/冒烟脚本契约** — 启动日志含 URL 与默认账密（可测 bootstrap 钩子）
- [x] **Step 2: 实现** 零配置启动、挂 Agent+管理 API、本机 spawn Edge
- [x] **Step 3: 本机 localfs + mock 手工/集成冒烟
- [x] **Step 4: Commit**

### Task 5: Settings + 三库 + 向导 API/UI

**Files:**
- Create: settings 持久化；向导 API
- Modify: `web/admin` 登录/向导流
- GORM dialector：sqlite/mysql/postgres

- [ ] **Step 1: 写失败测试** — 远程+localfs 拒绝；SQLite 向导保存可读回
- [ ] **Step 2: 实现** 向导步骤、加密存密钥、finalize+重启提示
- [ ] **Step 3: MySQL/Postgres 连通+migrate 测试（tag 或 testcontainers）
- [ ] **Step 4: 发布 embed 前端构建接入
- [ ] **Step 5: Commit**

### Task 6: 拆除默认 Redis/queue 用户路径 + 文档

**Files:**
- Modify: `botconfig.ValidateRuntimeDrivers` 叙事改为本机/远程
- Modify: README、`docs/architecture/runtime.md`、overview
- 删除或隔离默认 Redis 装配

- [ ] **Step 1: 更新校验与装配** 默认路径无 Redis
- [ ] **Step 2: 文档与 BREAKING 说明
- [ ] **Step 3: 全量相关测试 + 本机 E2E 清单勾选
- [ ] **Step 4: Commit**

## 验收清单（对齐 Design Doc）

- [ ] 空目录 `pixoma` → 日志 URL+账密
- [ ] 向导本机 SQLite+localfs → 自动 Edge mock → 任务成功
- [ ] 远程+localfs 失败
- [ ] 错 Token 401
- [ ] lease 过期可再领
- [ ] `COMFY_MOCK` 主路径仍通
