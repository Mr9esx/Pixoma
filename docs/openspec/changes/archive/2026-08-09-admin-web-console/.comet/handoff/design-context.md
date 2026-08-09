# Comet Design Handoff

- Change: admin-web-console
- Phase: design
- Mode: compact
- Context hash: 03cb8b822020e8f2797ffd1a487447b65f9b46500be3710fba80ceca22ddc681

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## docs/openspec/changes/admin-web-console/proposal.md

- Source: docs/openspec/changes/admin-web-console/proposal.md
- Lines: 1-29
- SHA256: ba4e84cdf2f4900b8b2ec8c3321cbd679afcd14c90aa4bfece641a293fe25824

```md
## Why

后端管理 API 就绪后，仍缺少基于 shadcn-admin 的管理后台前端：无菜单信息架构、无资源页面，运维只能靠 curl。需要在 `web/admin` 落地控制台骨架、菜单模块与各资源页面，并只通过 admin-api 完成基础管理操作。

## What Changes

- 以 satnaing/shadcn-admin（React + Vite + shadcn/ui + TanStack Router）为基座初始化 `web/admin`（pnpm）
- 实现后台**菜单模块**与默认 **Dashboard**（侧栏信息架构、路由注册、与资源页映射）
- 设计并实现资源页面：实例、Case、User、Session、Task（列表/详情/表单等基础管理交互）；Case 以结构化表单为主
- 对接 admin-api（含 foundation 实例 API 与 resources API）；交付以真 API 为准，不留独立前端 mock
- 中英双语齐全 + 语言切换；本期不做登录鉴权页；前端不直连 DB；不做 TG/Menu 落库管理

## Capabilities

### New Capabilities

- `admin-web-shell`: shadcn-admin 脚手架、布局、主题与菜单模块
- `admin-resource-pages`: 五类资源页面的信息架构、交互与对接 admin-api 的基础管理能力

### Modified Capabilities

- （无）

## Impact

- 代码：`web/admin/**`（从占位 README 变为可运行 SPA）
- 依赖：`admin-api-foundation`（必需）；全资源页依赖 `admin-resources-api`
- 工具链：Node + pnpm
- 非目标：Clerk/完整鉴权、实时大屏/告警、TG/Menu 落库、前端 mock、向公网发布、Bot UI

```

## docs/openspec/changes/admin-web-console/design.md

- Source: docs/openspec/changes/admin-web-console/design.md
- Lines: 1-43
- SHA256: eb58367a095afa4fe378c9df101de8db9243ec57631e0a38a0f193b0d8b62b6b

```md
## Context

参见 `proposal.md`。`web/admin` 现为占位；基座选用 satnaing/shadcn-admin（Vite + React + TanStack Router + shadcn）。后端分属 foundation 与 resources-api（均已归档可用）。

## Goals / Non-Goals

**Goals:**
- 可运行脚手架与菜单驱动导航
- 默认 Dashboard（中等：卡片 + 简单分布）+ 五类资源页的基础信息架构与交互
- 对接真实 admin-api；中英双语齐全 + 语言切换
- 包管理器 pnpm

**Non-Goals:**
- 登录鉴权；实时监控大屏/告警；TG/Menu 落库管理；独立前端 mock；组件库重写；前端直连 DB

## Decisions

1. **落位 `web/admin`**，以 shadcn-admin 为起点裁剪 demo 页，保留布局/侧栏模式；**pnpm**。  
2. **菜单配置化**：Dashboard → 实例 → Case → Task → User → Session；默认路由 `/` = Dashboard。  
3. **数据层**：TanStack Query + `lib/api` 指向 `VITE_ADMIN_API_BASE`；**不留**前端 mock。  
4. **页面深度**：实例/Case 偏完整表单（Case 结构化多段）；User/Session 只读；Task 列表+取消；Dashboard 用 list API 前端聚合。  
5. **无鉴权**：不保留 Clerk 强制登录；移除或旁路模板鉴权挡板。  
6. **i18n**：默认中文，本期中英两套齐全（壳子 + Dashboard + 五资源）+ 顶栏切换。

## Risks / Trade-offs

- [基座裁剪成本] → 锁定版本、只删鉴权与无关 demo。  
- [Dashboard 计数受 list limit 影响] → UI/文档标明样本口径；统计 API 另开。  
- [Case 表单工作量大] → workflow/schema 允许受控 JSON 兜底。  
- [无 mock 联调门槛] → README 写清先起 admin-api。

## Migration Plan

1. 初始化前端并跑通空壳 + i18n + Dashboard。  
2. 菜单 + 五页骨架。  
3. 逐资源对接 API。  
4. 文档：安装、环境变量、与 admin-api 联调。

## Open Questions

- （已关闭）包管理器：pnpm  
- （已关闭）开发期 mock：不留  
- （已关闭）TG/Menu：本期不做  

```

## docs/openspec/changes/admin-web-console/tasks.md

- Source: docs/openspec/changes/admin-web-console/tasks.md
- Lines: 1-26
- SHA256: db9f2358750fb9e7d0eb916e7f4364b7c1a681a4b80a92dbd37637dc29bc8009

```md
## 1. 脚手架与壳

- [ ] 1.1 在 `web/admin` 基于 shadcn-admin 初始化可运行项目并写清安装/启动文档
- [ ] 1.2 移除或旁路模板鉴权挡板，使无登录可进入布局
- [ ] 1.3 配置 admin-api 基址环境变量与 API 客户端封装

## 2. 菜单、i18n 与 Dashboard

- [ ] 2.1 实现集中式菜单配置（Dashboard → 实例 → Case → Task → User → Session）
- [ ] 2.2 侧栏渲染菜单并与 TanStack Router 路由绑定；默认路由为 Dashboard
- [ ] 2.3 清理无关 demo 菜单项与鉴权挡板，保留管理布局与主题能力
- [ ] 2.4 中英 i18n（默认中文）+ 语言切换
- [ ] 2.5 Dashboard 中等总览（卡片 + 简单分布，list API 前端聚合）

## 3. 资源页面

- [ ] 3.1 实例管理页：列表/创建或编辑/观测入口，对接 foundation API
- [ ] 3.2 Case 管理页：列表/编辑/创建/禁用，对接 resources API
- [ ] 3.3 User、Session 列表与详情页（只读为主）
- [ ] 3.4 Task 列表/详情与取消操作
- [ ] 3.5 统一空态、加载态与错误提示

## 4. 联调验收

- [ ] 4.1 与 admin-api 联调：菜单可达全部资源页并完成基础管理操作
- [ ] 4.2 确认请求仅指向 admin-api，文档说明联调步骤

```

## docs/openspec/changes/admin-web-console/specs/admin-resource-pages/spec.md

- Source: docs/openspec/changes/admin-web-console/specs/admin-resource-pages/spec.md
- Lines: 1-52
- SHA256: c5fc6cd1a0bd817ba1403fb623bdac565907a33374c50db72cfebeb8f3968c18

```md
## Purpose

为五类管理资源提供页面信息架构与基础管理交互，并通过 admin-api 完成真实数据操作。

## ADDED Requirements

### Requirement: 实例管理页
控制台 MUST 提供实例列表/详情或表单页，支持基础 CRUD 与观测入口（对接 admin-api 实例接口）。

#### Scenario: 从 UI 完成实例列表与创建
- **WHEN** 运维在实例页查看列表并提交合法新建
- **THEN** UI 展示更新后的列表且数据来自 admin-api

### Requirement: Case 管理页
控制台 MUST 提供 Case 列表与编辑/创建/启用禁用相关页面交互；创建与编辑 MUST 以结构化多段表单为主路径（非整页 JSON 编辑器作为唯一入口）。

#### Scenario: 从 UI 禁用 Case
- **WHEN** 运维在 Case 页对某 Case 执行禁用
- **THEN** UI 反映禁用状态且请求发往 admin-api

#### Scenario: 结构化编辑 Case
- **WHEN** 运维打开 Case 创建或编辑页
- **THEN** 页面以分段表单展示基础信息、输入输出与绑定等字段，提交后数据经 admin-api 持久化

### Requirement: User / Session / Task 运维页
控制台 MUST 提供 User、Session、Task 的列表与详情页；Task 页 MUST 支持取消等已由 API 提供的运维动作。

#### Scenario: 查看用户与会话详情
- **WHEN** 运维从列表进入某 User 或 Session 详情
- **THEN** 页面展示 admin-api 返回的关键字段

#### Scenario: 从 UI 取消任务
- **WHEN** 运维在 Task 详情对可取消任务执行取消
- **THEN** UI 反映取消结果或明确错误

### Requirement: Dashboard 中等总览
控制台 MUST 提供 Dashboard 页：数字卡片与简单状态/占比分布；数据 MUST 来自 admin-api 既有列表类接口的前端聚合（本期不新增统计专用端点）。

#### Scenario: Dashboard 展示聚合信息
- **WHEN** 用户打开 Dashboard 且 admin-api 可用
- **THEN** 页面展示至少实例与 Task（或 Case）相关的汇总卡片或分布，且网络请求指向 admin-api

#### Scenario: Dashboard 卡片失败隔离
- **WHEN** 某一汇总依赖的 API 请求失败
- **THEN** 仅对应卡片或区块进入错误/空态，其它区块仍可展示

### Requirement: 仅通过 admin-api 通信
前端 MUST 仅通过配置的 admin-api 基址访问管理数据，MUST NOT 直连数据库或 bot 管理残留路径作为正式方案，MUST NOT 依赖独立前端 mock 层作为交付验收路径。

#### Scenario: API 基址可配置
- **WHEN** 开发者配置 admin-api 基址并打开资源页
- **THEN** 网络请求指向该基址下的管理 API

```

## docs/openspec/changes/admin-web-console/specs/admin-web-shell/spec.md

- Source: docs/openspec/changes/admin-web-console/specs/admin-web-shell/spec.md
- Lines: 1-48
- SHA256: 7d63efeb37f04e4732e1ffd1fb8bc8ff5a1db51426a41bc02fd9f6a4d1c2e686

```md
## Purpose

基于 shadcn-admin 提供管理后台应用壳：布局、主题、可配置侧栏菜单、默认 Dashboard 与中英 i18n，作为所有资源页的导航入口。

## ADDED Requirements

### Requirement: 可运行的管理前端应用
系统 MUST 在 `web/admin` 提供可本地启动的 React 管理前端，技术栈基于 shadcn/ui 与 shadcn-admin 基座，包管理器为 pnpm。

#### Scenario: 本地启动控制台
- **WHEN** 开发者按文档使用 pnpm 安装依赖并启动开发服务器，且已配置可到达的 admin-api
- **THEN** 浏览器可打开管理控制台壳页面

### Requirement: 菜单模块
系统 MUST 提供后台菜单模块，将菜单项映射到路由，并在侧栏展示 Dashboard 与全部一期资源入口；菜单顺序 MUST 为 Dashboard、实例、Case、Task、User、Session。

#### Scenario: 侧栏展示菜单
- **WHEN** 用户打开控制台
- **THEN** 侧栏可见 Dashboard、实例、Case、Task、User、Session 菜单项，且顺序如上

#### Scenario: 点击菜单进入对应路由
- **WHEN** 用户点击某一菜单项
- **THEN** 导航到该菜单绑定的页面路由

### Requirement: 默认进入 Dashboard
系统 MUST 将应用默认路由指向 Dashboard 页。

#### Scenario: 打开根路径
- **WHEN** 用户访问控制台根路径
- **THEN** 内容区展示 Dashboard（而非强制登录页或其它资源页）

### Requirement: 布局与基础主题
系统 MUST 提供含侧栏与顶栏的管理布局，并支持基座自带的基础主题切换能力（若基座提供明暗主题）。

#### Scenario: 布局稳定包裹页面
- **WHEN** 用户在资源页之间切换
- **THEN** 侧栏/顶栏布局保持，内容区切换为目标页

### Requirement: 中英 i18n
系统 MUST 提供中英两套完整界面文案（壳子、Dashboard、一期资源页），默认语言为中文，并 MUST 提供语言切换能力。

#### Scenario: 默认中文
- **WHEN** 用户首次打开控制台且无已保存语言偏好
- **THEN** 界面以中文展示

#### Scenario: 切换到英文
- **WHEN** 用户将语言切换为英文
- **THEN** 壳子与当前页可见文案切换为英文，且偏好在刷新后仍生效

```
