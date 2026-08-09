---
comet_change: admin-web-console
role: technical-design
canonical_spec: openspec
archived-with: 2026-08-09-admin-web-console
status: final
---

# admin-web-console 技术设计

## 1. 目标与边界

在 `web/admin` 落地可运行管理控制台：裁剪 shadcn-admin 基座、配置化侧栏、中英 i18n、中等 Dashboard，以及实例 / Case / Task / User / Session 页面，**仅**通过 `apps/admin-api` 完成基础管理。

**非目标：** 鉴权登录；TG / Menu 落库管理；实时监控大屏与告警；独立前端 mock 层；前端直连 DB 或 bot 管理残留路径；组件库重写。

OpenSpec 能力（canonical）：`admin-web-shell`、`admin-resource-pages`。

## 2. 架构

```
浏览器 web/admin (Vite + React)
  布局 / 侧栏菜单配置 / i18n
  路由 (TanStack Router)
        │
        ▼
  lib/api + TanStack Query
  VITE_ADMIN_API_BASE
        │
        ▼
apps/admin-api  /api/v1/*
  comfy-instances | cases | tasks | users | sessions
```

### 2.1 基座与裁剪

- 起点：satnaing/shadcn-admin（Vite、React、TanStack Router、shadcn/ui、主题）。
- 保留：侧栏 + 顶栏布局、明暗主题（若基座已有）、通用 UI  primitives。
- 移除：Clerk / 强制登录、与本期无关的 demo 业务页与菜单。
- 包管理器：**pnpm**（与上游模板一致）；`web/admin/README.md` 写清安装与联调。

### 2.2 菜单与路由

集中菜单配置（id / titleKey / path / icon），顺序固定：

1. Dashboard → `/`（默认）
2. 实例 → `/instances`
3. Case → `/cases`
4. Task → `/tasks`
5. User → `/users`
6. Session → `/sessions`

未知路径重定向到 `/`。五类资源页采用 **Master–Detail（方案 B）**：同一路由前缀下左列表右详情（如 `/cases` + 选中 id，或 `/cases/$id` 且列表保持挂载）；窄屏可降级为先列表后整页详情。Dashboard 仍为整页，不适用分栏。

备选布局 A–E 全文见：`docs/superpowers/specs/2026-08-09-admin-list-detail-layout-options.md`（后续 change 可再选）。

### 2.3 数据层

- 单一 API 客户端：`baseURL = import.meta.env.VITE_ADMIN_API_BASE`，JSON fetch，错误体尽量透出后端 `message`。
- TanStack Query：列表/详情 query；变更 mutation 后 invalidate 相关 query。
- **不**引入 MSW / 假数据开关；开发与验收一律打真 admin-api。

### 2.4 i18n

- 默认语言：`zh`；完整 `en` 资源（壳子、Dashboard、五资源页文案）。
- 顶栏语言切换；偏好可写入 `localStorage`（版本化 key，如 `admin-locale:v1`）。
- 技术选型：基座若已有 i18n 则沿用；否则轻量 `i18next` / 等价方案，不自研框架。

## 3. 页面设计

### 3.0 列表 / 详情布局（本期 = B）

全站五资源统一 **左右分栏 Master–Detail**：

| 区域 | 行为 |
|---|---|
| 左栏 | 可筛选列表；选中行高亮；展示 id/状态/关键列 |
| 右栏 | 当前选中项的详情或表单；可滚动；Case 为多段结构化表单 |
| 空态 | 未选中时右栏提示「选择一项」；新建时右栏直接出空表单 |
| 窄屏 | 可先全宽列表，点进后再全宽详情（降级），桌面默认同屏 |

不采用本期：A 整页跳转、C 卡片抽屉、D 高密度行内主操作台、E 文档目录（均已入库方案库，供后用）。

### 3.1 Dashboard（中等）

- 数字卡片：实例总数/启用数、Case 启用数、Task 按状态计数等——**用现有 list API 前端聚合**，本期不加统计端点。
- 简单可视化：Task 状态分布、实例启用占比（条形/简单图即可）。
- 卡片失败隔离：单块错误/空态，不拖垮整页。
- 卡片可导航到对应资源列表。

### 3.2 实例

- 左列表 + 右详情/表单：`base_url`、`enabled`、`capabilities` 等对齐 admin-api。
- 右栏可含观测入口：system / queue / instance tasks——只读展示 API 返回。

### 3.3 Case（结构化表单）

右栏主路径为多段可滚动表单，不对运维甩整页 JSON 编辑器：

| 段 | 字段 |
|---|---|
| 基础 | id、name、description、preview、price、tags、menu_key、categories、enabled |
| 输入/输出定义 | `inputs[]` / `outputs[]` 行编辑 |
| 绑定 | `bindings.inputs` / `bindings.outputs` |
| Workflow | `bindings.workflow`：优先结构化控件；图结构复杂时允许**受控 JSON 区**作为该段兜底 |
| Input Schema | `input_schema`：分区编辑或受控 JSON |

列表支持与 API 对齐的过滤（`enabled`、`menu_key`、`q` 等）。动作：创建、更新、enable、disable。

### 3.4 Task / User / Session

- 均为左列表 + 右详情（方案 B）。
- Task：右栏可取消时 `POST .../cancel`。
- User / Session：右栏**只读**。

### 3.5 通用 UX

加载骨架、空态、错误条/toast；变更成功明确反馈；文案走 i18n。

## 4. 错误与联调

| 情况 | 行为 |
|---|---|
| API 非 2xx | 展示后端错误文案（有则），否则通用失败文案（双语） |
| 网络/CORS | 提示检查 `VITE_ADMIN_API_BASE` 与 admin-api |
| Dashboard 单卡片失败 | 仅该卡片错误态 |

联调：先起 admin-api，再 `pnpm install && pnpm dev`。CORS 依赖 admin-api 既有配置。

## 5. 测试策略

- 手测为主：壳子导航、Dashboard、五资源主路径、中英切换、无鉴权可达。
- 可单测：菜单顺序、locale 持久化、Dashboard 聚合纯函数。
- 交付门槛：请求仅指向 admin-api；无 TG/Menu 管理入口；无前端 mock 开关。

## 6. 风险

- Case 表单深度 vs 工期：分段交付，workflow/schema 兜底 JSON 须在 UI 标明「高级/原始」。
- 列表无 total 时 Dashboard 计数受 `limit` 影响：文档与 UI 注明「基于当前拉取样本/上限」或拉足够大的 limit；若不准，后续可单开统计 API，不在本期强依赖。
- 基座升级与本地裁剪冲突：锁定依赖版本，少改 primitives。

## 7. Spec Patch 摘要

已回写 OpenSpec：放宽 i18n 与中等 Dashboard；补默认首页与双语验收；包管理器/mock 决策写入 `design.md`。
