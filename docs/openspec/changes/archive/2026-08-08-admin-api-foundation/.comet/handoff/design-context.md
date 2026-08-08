# Comet Design Handoff

- Change: admin-api-foundation
- Phase: design
- Mode: compact
- Context hash: fc3d7c2d7c4803acd5325e3bc776b7c991eeb959ab9606ef2195c27e045f913e

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## docs/openspec/changes/admin-api-foundation/proposal.md

- Source: docs/openspec/changes/admin-api-foundation/proposal.md
- Lines: 1-29
- SHA256: 5e7f2b9f1b5cccafd65b57875b30852f39ee617c692fb299ad6cd62bca45e23b

```md
## Why

管理能力目前临时挂在 bot 同进程 HTTP 上，且 `apps/admin-api` 仅有占位目录，无法作为独立管理入口。需要先落地可运行的 admin-api 骨架，并把已有实例管理接口从 bot 迁出，为后续资源 API 与 `web/admin` 提供稳定后端边界。

## What Changes

- 新建可独立启动的 `apps/admin-api`（配置、wire、health、CORS；本期无鉴权）
- 将 bot 上的 `/api/v1/comfy-instances*` 管理/观测 HTTP **迁移**到 admin-api（复用现有 handler/应用能力，禁止依赖 `channel/tg`）
- **BREAKING**：bot 进程不再挂载管理 CRUD/观测路由；运维改连 admin-api
- 明确 admin-api 与 bot 共用同一持久化（DB/配置约定），实例池刷新语义与现有行为对齐
- 文档与示例配置补充 admin-api 启动与 curl 用法

## Capabilities

### New Capabilities

- `admin-api-host`: 独立 admin-api 进程的启动、健康检查、CORS、无鉴权管理入口与 bot 职责边界
- `comfy-instance-admin-api`: Comfy 实例在 admin-api 上的 CRUD 与 system/queue/tasks 观测 HTTP

### Modified Capabilities

- （无）主规格中实例管理行为不改语义，仅变更承载进程；若后续发现既有 spec 写死“挂在 bot”，再补 delta

## Impact

- 代码：`apps/admin-api/`、`apps/bot/cmd/comfyui-bot`（卸下管理路由）、`internal/httpapi/comfyinstances`（挂载点迁移）、`configs/`（admin-api 配置）
- API：管理客户端改连 admin-api 基址；bot HTTP 仅保留非管理用途（如 `/healthz`，若仍需要）
- 依赖：与 bot 共享 DB / 实例仓储；不引入鉴权依赖
- 后续 change：`admin-resources-api`、`admin-web-console` 依赖本 change

```

## docs/openspec/changes/admin-api-foundation/design.md

- Source: docs/openspec/changes/admin-api-foundation/design.md
- Lines: 1-58
- SHA256: 050afe6088e818975d7cff1350e117db406c8fe6ab017b4009b44a1811df2261

```md
## Context

参见 `proposal.md`。现状：`apps/admin-api` 仅 README 占位；`internal/httpapi/comfyinstances` 已在 `apps/bot` 挂载；`web/admin` 约定只连 admin-api。本 change 只做宿主与实例 API 迁移，其他资源与前端属批次内后续 change。

## Goals / Non-Goals

**Goals:**
- 可独立运行的 admin-api 组合根（配置、DB、chi 路由、CORS、health）
- 复用现有实例仓储/Pool 观测能力，handlers 改由 admin-api 挂载
- bot 卸下管理路由，职责回到对话与编排

**Non-Goals:**
- Case/User/Session/Task 管理 API（`admin-resources-api`）
- 前端菜单与页面（`admin-web-console`）
- 鉴权、RBAC、多租户
- 改变实例调度/健康探测的核心算法（仅保证管理写路径后池可刷新的既有约定）

## Decisions

1. **独立进程而非 bot 子路由**  
   - 选择：新建 `apps/admin-api/cmd/admin-api`。  
   - 相对：继续挂在 bot。  
   - 理由：与既有 monorepo 规划一致，避免管理面与 TG 通道耦合，便于后续只暴露 admin-api。

2. **Handler 包留在 `internal/httpapi`，换挂载点**  
   - 选择：迁移挂载，尽量少改 handler 语义。  
   - 相对：复制一份 admin 专用 handler。  
   - 理由：降低双份协议漂移；admin-api 与 bot 都可依赖同一 httpapi（bot 本期不再 Mount）。

3. **共用同一 DB/配置源约定**  
   - 选择：admin-api 读取与 bot 相同的持久化（如同一 sqlite/DSN），实例变更对 bot 池可见需明确刷新策略（启动加载 + 写后刷新，或依赖既有 Probe/Reload）。  
   - 相对：独立管理库。  
   - 理由：一期运维简单；避免双写。

4. **CORS 宽松联调默认**  
   - 选择：开发配置允许本地前端源；生产收紧留给后续。  
   - 理由：本期无鉴权 + 本地联调优先。

5. **无鉴权**  
   - 选择：明确文档警示，不实现 token。  
   - 理由：用户确认本期不做鉴权。

## Risks / Trade-offs

- [无鉴权公网暴露] → README/配置注释强制警示；默认绑定本机。  
- [bot 与 admin-api 双进程写实例导致池陈旧] → 写路径触发池刷新或文档要求重启/刷新；Design Doc 阶段细化。  
- [BREAKING 运维脚本仍打 bot 端口] → README 与变更说明给出新基址与迁移步骤。

## Migration Plan

1. 落地 admin-api 并可 curl 实例 API。  
2. bot 移除管理路由并发布说明。  
3. 更新本地脚本/README 指向 admin-api。  
4. 回滚：临时恢复 bot Mount（不推荐并行长期双挂）。

## Open Questions

- 实例写后跨进程通知 bot 池刷新的最小机制（共享文件信号 / 仅 Probe 周期 / 管理 API 触发）——不改变规格可观测行为，实现期选定。

```

## docs/openspec/changes/admin-api-foundation/tasks.md

- Source: docs/openspec/changes/admin-api-foundation/tasks.md
- Lines: 1-19
- SHA256: 9a89b85421fddd5a0daa324312bfa814604d13312709bcb231c5b6ba809d24ce

```md
## 1. Admin API 骨架

- [ ] 1.1 创建 `apps/admin-api/cmd/admin-api` 入口与配置加载（含 `configs/admin-api.example.yaml`）
- [ ] 1.2 Wire DB 与实例仓储/Pool（与 bot 共用持久化约定）
- [ ] 1.3 挂载 `/healthz`、CORS 中间件与 chi 路由骨架
- [ ] 1.4 补充 Makefile/README：启动方式与无鉴权公网警示

## 2. 实例管理 API 迁移

- [ ] 2.1 在 admin-api 挂载 `/api/v1/comfy-instances*`（复用 `internal/httpapi/comfyinstances`）
- [ ] 2.2 验证 CRUD + system/queue/tasks 观测与迁出前语义一致（含 mock 标记）
- [ ] 2.3 选定并实现实例写后对 bot 侧池可见的最小策略（Probe/刷新/文档约定之一）
- [ ] 2.4 从 `apps/bot` 移除管理 CRUD/观测路由挂载并更新相关说明

## 3. 验收与回归

- [ ] 3.1 独立启动 admin-api，用 curl 走通实例列表/创建/更新/观测
- [ ] 3.2 确认 bot 原管理路径不再提供管理 API
- [ ] 3.3 更新根 README「实例管理与观测」指向 admin-api

```

## docs/openspec/changes/admin-api-foundation/specs/admin-api-host/spec.md

- Source: docs/openspec/changes/admin-api-foundation/specs/admin-api-host/spec.md
- Lines: 1-33
- SHA256: f2227d0317274717282670f54b814b2260fcfda6500774945a677ca8013a513e

```md
## Purpose

定义独立 admin-api 进程作为无鉴权管理入口的宿主职责，以及与 bot 进程在管理 HTTP 上的边界。

## ADDED Requirements

### Requirement: 独立 admin-api 进程可启动并提供健康检查
系统 MUST 提供可独立启动的 admin-api 服务进程，并暴露健康检查端点以供运维探测。

#### Scenario: 健康检查成功
- **WHEN** 运维对 admin-api 发起健康检查请求
- **THEN** 服务返回成功响应，表明进程已就绪

### Requirement: 管理 HTTP 无鉴权（本期）
本期 admin-api 的管理 HTTP MUST 在无鉴权模式下可用，以便本地与可信内网联调；文档 MUST 警示不得对公网暴露。

#### Scenario: 无鉴权访问管理接口
- **WHEN** 客户端在未提供凭证的情况下调用 admin-api 管理接口
- **THEN** 请求不得因“缺少鉴权”被拒绝（仍可因业务校验失败被拒绝）

### Requirement: CORS 允许管理前端联调
admin-api MUST 支持 CORS，使 `web/admin` 开发源可在浏览器中调用其管理 API。

#### Scenario: 浏览器预检或跨域请求
- **WHEN** 管理前端所在源向 admin-api 发起跨域请求（含预检）
- **THEN** 服务返回允许该联调场景的 CORS 响应头

### Requirement: bot 不再承载管理 CRUD/观测路由
bot 进程 MUST NOT 再挂载已迁移的管理 CRUD/观测 HTTP 路由；该类能力仅由 admin-api 提供。

#### Scenario: bot 上管理路径不可用
- **WHEN** 客户端对 bot 的原管理路径（如 `/api/v1/comfy-instances`）发起请求
- **THEN** 该路径不再提供原管理 API 行为（例如 404 或不存在对应路由）

```

## docs/openspec/changes/admin-api-foundation/specs/comfy-instance-admin-api/spec.md

- Source: docs/openspec/changes/admin-api-foundation/specs/comfy-instance-admin-api/spec.md
- Lines: 1-62
- SHA256: 57e1b29d2daaf08e4742ef300e124438ac72146d1100d6401e95249c1f12d4f7

```md
## Purpose

在独立 admin-api 上提供 Comfy 实例的管理 CRUD 与 system/queue/tasks 观测 HTTP，语义与迁出前对齐。

## ADDED Requirements

### Requirement: 实例列表与详情
系统 MUST 通过 admin-api 提供 Comfy 实例的列表与按 ID 查询能力。

#### Scenario: 列出实例
- **WHEN** 客户端请求实例列表
- **THEN** 返回当前持久化中的实例集合（含启用状态等关键字段）

#### Scenario: 获取单个实例
- **WHEN** 客户端请求已存在实例的详情
- **THEN** 返回该实例记录

#### Scenario: 实例不存在
- **WHEN** 客户端请求不存在的实例 ID
- **THEN** 返回表示未找到的错误响应

### Requirement: 实例创建与更新
系统 MUST 允许通过 admin-api 创建与更新实例（含 base URL、启用状态、能力等字段），变更 MUST 持久化。

#### Scenario: 创建实例
- **WHEN** 客户端提交合法的新实例载荷
- **THEN** 实例被持久化并可在后续列表/详情中查到

#### Scenario: 更新实例
- **WHEN** 客户端提交已存在实例的合法更新
- **THEN** 持久化记录反映更新后的字段

### Requirement: 实例删除或停用
系统 MUST 支持通过 admin-api 删除或停用实例，使该实例不再作为可用管理目标（停用语义须在响应/列表中可观察）。

#### Scenario: 停用或删除后不可再作为健康调度目标
- **WHEN** 运维通过 admin-api 停用或删除某实例
- **THEN** 后续列表/详情反映该变更，且不得再将该实例表现为可用启用实例（除非再次启用）

### Requirement: 实例观测 system/queue/tasks
系统 MUST 在 admin-api 上按实例提供 system、queue 与本系统 tasks 观测端点，行为与迁出前 bot 挂载时对齐（含 mock 模式下的可观察标记，若启用 mock）。

#### Scenario: 查询 system
- **WHEN** 客户端请求某已存在实例的 system 观测
- **THEN** 返回该实例的 system 观测载荷或明确的错误

#### Scenario: 查询 queue
- **WHEN** 客户端请求某已存在实例的 queue 观测
- **THEN** 返回该实例的 queue 观测载荷或明确的错误

#### Scenario: 按实例列出本系统 tasks
- **WHEN** 客户端请求某已存在实例关联的本系统 tasks
- **THEN** 返回与该实例关联的任务列表（可为空）

### Requirement: 管理端变更对 bot 调度池最终可见
当 admin-api 与 bot 共用同一持久化时，经 admin-api 持久化的实例变更 MUST 在 bot 的约定刷新周期内对 bot 调度用实例视图可见。系统 MUST NOT 要求该可见性为瞬时；刷新周期与 bot 健康探活间隔对齐（可配置）。

#### Scenario: 一个探活周期内 bot 可见新名单
- **WHEN** 运维通过 admin-api 创建或更新某启用实例并写入持久化
- **AND** bot 与 admin-api 使用同一数据库
- **AND** 已等待至多一个 bot 健康探活间隔
- **THEN** bot 的调度用实例视图 MUST 能观察到该变更（例如新启用实例可进入健康候选，或停用实例不再表现为可用启用实例）

```
