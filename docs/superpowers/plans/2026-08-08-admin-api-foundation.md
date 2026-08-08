---
change: admin-api-foundation
design-doc: docs/superpowers/specs/2026-08-08-admin-api-foundation-design.md
base-ref: 5b8ca1cb37a00497ab3e50c03a11a9483a900f86
---

---
change: admin-api-foundation
design-doc: docs/superpowers/specs/2026-08-08-admin-api-foundation-design.md
base-ref: 5b8ca1cb37a00497ab3e50c03a11a9483a900f86
---

# admin-api-foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 落地可独立启动的 admin-api，抽出与 bot 共用的开库/迁移/种子启动包，把实例管理 HTTP 从 bot 迁到 admin-api，并让 bot 在探活周期内从 DB 刷新机器名单。

**Architecture:** 新建 `internal/platform/appboot`（或等价）承载 OpenDB+AutoMigrate+可选实例种子；`apps/admin-api` 组合根挂 chi、CORS、health、复用 `comfyinstances.Handler`；bot 卸管理路由，探活循环内 `Pool.Refresh`；两边同一 `database_dsn`，无新表。

**Tech Stack:** Go、chi、GORM/sqlite、既有 `internal/httpapi/comfyinstances`、YAML 配置

## Global Constraints

- 产物语言：zh-CN
- 无鉴权；文档必须警示勿对公网暴露
- admin-api 禁止依赖 `channel/tg`
- 本期不新建/不改 `comfy_instances` 表结构
- bot 与 admin-api 必须共用同一 database_dsn
- Comfy Mock 开关与既有 `comfy_mock` / `COMFY_MOCK` 约定对齐（观测路径）
- 遵循仓库 golang skills；改动范围限于本 change

---

### Task 1: 公共启动包 appboot

**Files:**
- Create: `internal/platform/appboot/boot.go`（及必要的 options 类型）
- Create: `internal/platform/appboot/boot_test.go`
- Modify（按需）: `apps/bot/cmd/comfyui-bot/main.go` 改为调用 appboot（可在本任务末尾或 Task 3 完成；本任务至少提供可测 API）

**Interfaces:**
- Produces: 类似 `appboot.Open(ctx, opts) (*gorm.DB, error)` 与 `appboot.Migrate(db, models...)` / `Bootstrap(opts) (db, cleanup, error)`；opts 含 DSN、是否 migrate 实例相关模型、可选实例种子列表

- [x] **Step 1: 写失败测试** — 内存/临时 sqlite DSN 下 Bootstrap 后 `comfy_instances` 可写入/读出（或 migrate 后表存在）
- [x] **Step 2: 跑测试确认失败**
- [x] **Step 3: 实现最小 appboot**
- [x] **Step 4: 测试通过**
- [x] **Step 5: Commit** — `feat(appboot): extract shared DB bootstrap for admin-api and bot`

---

### Task 2: admin-api 进程骨架（health + CORS + 配置）

**Files:**
- Create: `apps/admin-api/cmd/admin-api/main.go`
- Create: `internal/platform/adminconfig/config.go`（或与 botconfig 共享薄封装；优先独立 adminconfig 避免拖入 TG 字段）
- Create: `configs/admin-api.example.yaml`
- Modify: `apps/admin-api/README.md`、`Makefile`（若已有 run 目标则增加 `run-admin-api`）

**Interfaces:**
- Consumes: appboot
- Produces: 可监听 `http_addr`（默认 `:8081`）、`GET /healthz` 返回 ok、CORS 中间件允许本地开发源

- [x] **Step 1: 写失败测试或 smoke** — 配置 Load 默认端口；router 对 `/healthz` 返回 200（可用 httptest）
- [x] **Step 2: 实现配置 + main 接线（开库 migrate，先不挂实例路由也可，但建议同任务挂空路由组）**
- [x] **Step 3: 测试/本地启动通过**
- [x] **Step 4: Commit** — `feat(admin-api): add host process with health and CORS`

---

### Task 3: 挂载实例 API 并卸下 bot 管理路由

**Files:**
- Modify: `apps/admin-api/cmd/admin-api/main.go` — Mount `/api/v1/comfy-instances`
- Modify: `apps/bot/cmd/comfyui-bot/main.go` — 移除 comfyinstances Mount；改用 appboot；保留 `/healthz` 若需要
- Modify: `README.md` — 实例管理 curl 改指向 admin-api
- Test: 复用/扩展 `internal/httpapi/comfyinstances/handler_test.go`（语义不变）

**Interfaces:**
- Consumes: `comfyinstances.Handler`、Pool、Repo、Tasks、Mock 开关
- Produces: admin-api 上与迁出前一致的 CRUD/观测行为；bot 不再提供管理路径

- [x] **Step 1: 确认现有 handler 测试仍绿**
- [x] **Step 2: admin-api 挂载 Handler + Pool.Refresh 写路径**
- [x] **Step 3: bot 删除管理路由挂载**
- [x] **Step 4: 更新 README / admin-api README（无鉴权、同库、新端口）**
- [x] **Step 5: Commit** — `feat(admin-api): migrate comfy-instances HTTP off bot`

---

### Task 4: bot 探活周期内 Refresh 名单

**Files:**
- Modify: `apps/bot/cmd/comfyui-bot/main.go` — Probe 循环内先 `pool.Refresh` 再 `pool.Probe`（或等价顺序，保证从 DB 重建后再探活）
- Create/Modify: `internal/platform/instance` 相关测试，或 bot 级可测辅助函数（避免只能手工等 30s）

**Interfaces:**
- Consumes: `Pool.Refresh`、`Pool.Probe`、`health_probe_interval`
- Produces: 管理端写库后，一个探活间隔内 bot 调度视图可见（规格场景）

- [x] **Step 1: 写失败测试** — 模拟库中新增启用实例后调用 Refresh，Pool 可见新客户端/列表
- [x] **Step 2: 实现循环内 Refresh（若 Refresh 已存在，任务重点是接线与回归测试）**
- [x] **Step 3: 测试通过**
- [x] **Step 4: Commit** — `fix(bot): refresh instance pool each health probe tick`

---

### Task 5: 端到端验收与文档收尾

**Files:**
- Modify: `README.md`、`apps/admin-api/README.md`、`configs/admin-api.example.yaml`（核对 DSN 示例与 bot 一致写法）
- Modify: `docs/openspec/changes/admin-api-foundation/tasks.md` — 勾选已完成项

- [ ] **Step 1: 手工或脚本** — 启动 admin-api，curl 列表/创建；打 bot 旧管理路径应失败
- [ ] **Step 2: 确认 mock 下观测仍可用**
- [ ] **Step 3: 勾选 tasks.md 对应项**
- [ ] **Step 4: Commit** — `docs(admin-api): document admin-api foundation rollout`

---

## 执行注意

- 每个 Task 保持可独立审查；不要把 resources-api / 前端范围带进来
- TDD：业务逻辑与 Refresh/Bootstrap 优先红绿；纯文档步骤可 direct
- 提交信息用英文 conventional 前缀即可，与仓库近期风格一致
