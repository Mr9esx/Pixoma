---
change: link-health-single-source
design-doc: docs/superpowers/specs/2026-09-06-link-health-single-source-design.md
base-ref: 2adfdaaea9d2cfbf6d73a5e4668f772ca760b59c
---

# 链路健康唯一主人 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans. Language: zh-CN

**Goal:** 配置链路绿黄只由控制面组装器计算；后台只渲染；删掉各页自算逻辑，不留降级。

**Architecture:** `internal/packaging/linkhealth` 对只读快照拼图并走完整活路；`GET /api/v1/link-health` 输出 nodes/edges。探测/适配器必须组装器能读到（通道上次探测随通道持久化，作为输入可见性，不是产品标题）。前端删除 `*References` / `listHealthTone` / 多路拼图。

**Tech Stack:** Go（标准 library 测试）、chi admin-api、React Query admin。

## Global Constraints

- 不保留旧健康函数作为 fallback；查询失败只显示错误/未就绪，不得本地再算一遍。
- 查询热路径不得 POST 外部探测。
- 活路不变式测试，禁止「通道 error → case warn」当唯一验收。
- 文案无解释性说明句；状态规范绿/黄 + pending 不得当绿。
- 删除旧逻辑，不写兼容双路径。

---

### Task 1: 组装器不变式

**Files:**
- Create: `internal/packaging/linkhealth/graph.go`
- Create: `internal/packaging/linkhealth/graph_test.go`

**Interfaces:**
- Produces: `Assemble(Snapshot) Graph`；`NodeID(kind, id) string`；health `ok|warn|pending`

- [x] **Step 1: 写失败测试** 四场景：唯一入口不可用；双入口有活路；唯一节点不可用；从未探测 → pending 不得 ok
- [x] **Step 2: 跑测确认失败**
- [x] **Step 3: 实现 Assemble**
- [x] **Step 4: 测试通过**
- [x] **Step 5: Commit** `feat(linkhealth): assemble path-based config health`

### Task 2: 探测对组装器可见 + HTTP

**Files:**
- Modify channel domain/persistence/`CheckReachability` 写入上次探测
- Create: `internal/httpapi/linkhealth/handler.go` + test
- Modify: `internal/httpapi/adminhost/server.go`、`apps/pixoma/cmd/pixoma/main.go`、livedemo

- [x] **Step 1: 测试 check 后 List 能读到 kind；GET /link-health 200；无会话由既有管理门闩拒绝（与其它 /api/v1 一致）；handler 不调用 CheckTelegram**
- [x] **Step 2: 实现落库与装配 Snapshot → Assemble**
- [x] **Step 3: 测试通过并提交** `feat(linkhealth): expose GET /api/v1/link-health`

### Task 3: 前端只渲染并删除旧逻辑

**Files:**
- Create `web/admin/src/lib/api/link-health.ts`
- Modify 四类列表/详情、topology、query-keys
- Delete `references.ts`、`list-health.ts` 及运行时调用；类型迁到 `features/link-health/types.ts`
- 合同测试改为断言 queryKeys.linkHealth 且源码不含 `caseReferences(`

- [x] **Step 1: 合同测试先失败**
- [x] **Step 2: 改吃 API，删除旧文件**
- [x] **Step 3: vitest + tsc 通过并提交** `feat(admin): render link-health from single query`

### Task 4: 架构文档

**Files:** `docs/architecture/bounded-contexts.md`、`data-model.md`、`overview.md`、`docs/architecture/diagrams/link-health.html`、`docs/architecture/README.md`

- [x] 同步读模型与图，提交 `docs: document link-health assembler`
