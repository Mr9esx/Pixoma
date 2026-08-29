# Comet Design Handoff

- Change: dashboard-workbench-refactor
- Phase: design
- Mode: compact
- Context hash: 1b31f3bb975c4594a414181a150ff96992bfd2b8b346703c75822220fa0c0814

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## docs/openspec/changes/dashboard-workbench-refactor/proposal.md

- Source: docs/openspec/changes/dashboard-workbench-refactor/proposal.md
- Lines: 1-37
- SHA256: b87a4078432a0fb3d71dceb3c35d4b121eb8928148692a8859e58e824e6f0ad2

```md
## Why

现有 `/` 仪表盘是纵向排列的信息页，顶部单独展示标题与说明，且各分区（实时状态 / 任务效能 / 业务分析）彼此独立、缺乏主次与"当前需要关注"的聚合入口。运维打开首页时先看到的是描述文字而非数据，无法一眼得知集群健康、任务负载与待处理异常，操作效率与信息密度都不足。

本次将把该页改造成"工作台"：移除顶部独立标题/说明，换成一条全宽的欢迎卡（含当前用户昵称与快捷入口），其下按"左侧数据大盘 / 右侧关注信息"双栏组织，让健康状态、负载热点、任务趋势与待办异常在同一视图中即可浏览与跳转。

## What Changes

- 移除 `/` 页顶部独立的 `title` 与 `desc`（`dashboard.title` / `dashboard.fullAccuracyNote` 区块）。
- 新增全宽欢迎卡，基于 `@kokonutui/mouse-effect-card`：展示当前登录用户昵称（`/setup/me` 的 `nickname || username`）与快捷入口（新建工作流、添加/管理节点）。
- 页面主体改为左右双栏布局：左栏为数据大盘，右栏为"需要关注的信息"。
- 数据大盘新增/重排以下区块：
  - 全宽 `kibo-ui/contribution-graph` 展示每天执行任务量。
  - 三个卡片：节点情况（总 / 生效中）、平均负载、负载最高节点 Top5。
  - "工作流使用热度 Top" 与 "任务耗时" 左右两列。
  - "任务状态分布" 与 "错误 Top5" 左右两列。
- 右栏"需要关注的信息"接入真实数据：离线节点 / Comfy 未运行节点、未启用的工作流、近段时间失败任务 Top。
- 保留全局时间范围选择（近 7 / 30 / 90 天 + 自定义起止日期），contribution-graph 与统计卡片随范围联动刷新；节点在线/未启用等"实时"信息不受时间范围影响。
- 去掉旧的"集群实时负载图"（RealtimeStatusSection 中的 fleet bar chart 与"每节点任务量"卡），由"平均负载 / 负载最高节点 Top5"卡片替代，避免重复。

> 说明：原 `Dashboard 中等总览` 需求中的"分区展示实时与区间数据"与"全局时间范围"等展示性要求被新工作台视图取代；数据请求仍全部来自 `/api/v1/stats/*`、`/edges`、`/cases` 等 admin-api，不允许直连数据库或引入前端 mock 作为验收路径。

## Capabilities

### New Capabilities
- `admin-dashboard-workbench`: 后台工作台页的信息架构与展示需求，包括欢迎卡、数据大盘（任务热度图/节点情况/平均负载/负载 Top5/工作流热度 Top/任务耗时/状态分布/错误 Top5）与右侧关注信息区，以及全局时间范围联动与 admin-api 数据来源约束。

### Modified Capabilities
- `admin-resource-pages`: 原 `Dashboard 中等总览` 需求中的"数字卡片与简单状态/占比分布"及"分区展示实时与区间数据"将被新工作台视图覆盖；保留数据非样本全量口径与 admin-api 通信约束。

## Impact

- **前端**：`web/admin/src/features/dashboard/*`（新增 `workbench-*` 分区组件，调整/移除 `realtime-status-section`、`task-stats-section`、`case-analysis-section` 的部分卡片）、`web/admin/src/routes/_app/index.tsx`（渲染入口）、`web/admin/src/lib/i18n/locales/{zh,en}.json`（新增工作台文案）。
- **组件**：新增 `@kokonutui/mouse-effect-card` 与 `kibo-ui/contribution-graph` 注册表组件（写入 `web/admin/src/components/` 对应目录），遵循项目 shadcn 语义令牌与设计体系。
- **数据**：复用现有 `/api/v1/stats/tasks/daily`、`/stats/tasks/errors`、`/stats/tasks/edges`、`/stats/cases/top`、`/stats/fleet`、`/edges`、`/edges/presence`、`/cases`、`/setup/me`；无后端聚合逻辑变更。
- **测试**：调整 `dashboard` 相关契约/组件测试以匹配新布局；不新增截图类或浏览器截图测试脚手架。
- **行为兼容**：`/` 路由与菜单不变；不引入破坏性 API 变更（无 `**BREAKING**`）。

```

## docs/openspec/changes/dashboard-workbench-refactor/design.md

- Source: docs/openspec/changes/dashboard-workbench-refactor/design.md
- Lines: 1-75
- SHA256: 7e71da763dec2a000a7b77625e9493c7a31e84c1dff49077bce03f8b0c59f1b0

```md
## Context

现有 `/` 页由 `web/admin/src/features/dashboard/dashboard-page.tsx` 驱动，纵向堆叠「实时状态 / 任务效能 / 业务分析」三个 section，数据来自 `/api/v1/stats/*`、`/edges`、`/cases`。全部统计接口与前端类型（`lib/api/stats.ts`、`lib/api/types.ts`）已齐备，无后端改动需求。本次仅重构前端信息架构与组件布局，遵循 `pixoma-design-system`（中性灰语义令牌、表面无投影、accent 每屏至多两次、图表 chart-1…5、失败隔离、中英 i18n 默认中文）。

## Goals / Non-Goals

**Goals:**
- 移除页面顶部独立 `title` / `desc` 区块。
- 全宽欢迎卡（kokonutui mouse-effect-card）承载用户昵称与快捷入口。
- 左栏数据大盘（贡献图 + 三张概览卡 + 两组左右布局）+ 右栏关注信息。
- 复用现有 admin-api 接口，保持全量口径，所有统计卡失败隔离。
- 保留全局时间范围控件并联动区间型区块；实时/节点状态不随时间变化。

**Non-Goals:**
- 不修改后端聚合逻辑或新增接口。
- 不改动侧边栏、路由体系、`/` 路由入口。
- 不引入前端 mock 作为交付路径。
- 不新增截图/浏览器截图类测试脚手架。

## Decisions

### 1. 组件获取方式
两个外部组件都从注册表落地为仓库自有文件，而非运行期依赖：
- `@kokonutui/mouse-effect-card`：`https://kokonutui.com/r/mouse-effect-card.json`，写入 `web/admin/src/components/kokonutui/mouse-effect-card.tsx`。依赖占位（card/button 本仓库已有），`color` 直接用现成语义令牌（已有 `spotlight-cards.tsx` 同目录先例）。
- `kibo-ui/contribution-graph`：`https://kibo-ui.com/r/contribution-graph.json`，写入 `web/admin/src/components/kibo-ui/contribution-graph/index.tsx`。依赖 `date-fns`（本项目已有）。
- 备选：可用 `pnpm dlx shadcn@latest add ...` 走 components.json 注册表。但考虑这两个 item 的 target 目录与现有组件目录（`components/kokonutui`）一致性，直接按注册表返回内容落文件更可控。两者都遵循项目约定。

### 2. 布局结构
根容器改为两段：
```
div @container
├── MouseEffectCard（全宽欢迎卡，含昵称 + 2 个快捷入口）
└── div grid lg:grid-cols-[minmax(0,1fr)_360px] gap-4
    ├── 左栏：space-y-4
    │   ├── ContributionGraph（全宽，任务量日热度）
    │   ├── grid sm:grid-cols-3 三卡（节点情况 / 平均负载 / 负载 Top5）
    │   ├── grid lg:grid-cols-2（工作流使用热度 Top | 任务耗时）
    │   └── grid lg:grid-cols-2（任务状态分布 | 错误 Top5）
    └── 右栏：attention-panel（离线/未运行节点、未启用工作流、失败任务 Top）
```
- 用 CSS grid + `minmax(0,1fr)` 保证窄屏不溢出；窄屏优先降级为单列（右侧关注信息落到下方）。
- 欢迎卡用 `@kokonutui/mouse-effect-card`，但需覆盖其内置 `max-w-md`、固定 `h-[400px]` 与默认品牌文案，改为全宽、内容自适应，仅保留鼠标点阵动效与「表面无投影」的 card border 分层。

### 3. 数据与联动
- `ContribGraph` 数据来自 `listTaskDailyStats(range)`，将 `days[].processed` 映射为 `{ date, count, level }`（level 由 count 相对分位数生成 0–4）。
- 三张概览卡：节点情况用 `listEdges`（总数/启用数）；平均负载与负载 Top5 用 `listFleetStats`（`avg_cpu/mem/gpu`、`nodes` 按 `cpu_usage_percent` 降序取 top5）。`/stats/fleet` 无 `from/to`，属实时口径，不随时间范围变化。
- 「工作流使用热度 Top」用 `listTaskCaseTopStats({...range, limit:10})`；「任务耗时」用 `listTaskDailyStats(range)` 的 `avg_queue_ms` / `avg_exec_ms` / `avg_duration_ms`；「任务状态分布」用 `daily.summary` 的 `succeeded/failed/cancelled`；「错误 Top5」用 `listTaskErrorStats({...range, limit:5})`。
- 右侧关注信息：离线/未运行节点用 `listEdges` + `listPresence` 交叉；未启用工作流用 `listCases({enabled:false})`；失败任务 Top 用 `listTaskErrorStats`。右栏整体随全局范围（失败 Top 联动），节点/工作流状态保持实时。
- 时间范围选择器复用现有 `TaskRangePicker`（7/30/90 + 自定义），state 提至工作台根组件。

### 4. 失败隔离
保持现有模式：每个区块独立 `useQuery`，独立 `isLoading/isError`，用现有 `LoadingSkeleton` / `ErrorBanner`（`components/feedback/*`）。任一 API 失败仅该区块进入错误/空态。

### 5. i18n
工作台文案走 `zh.json` / `en.json`，默认中文；移动端/窄屏降级说明、卡片标题、关注信息空态均为 i18n key。文案遵循 `voice-profile.md`（动作起头、无禁用词、无感叹号/emoji）。

### 6. 旧区块取舍
- 移除旧 `realtime-status-section` 的「集群实时负载图」bar chart、算力池汇总卡、每节点任务量卡，避免与新增「平均负载 / 负载 Top5」重复。
- 保留 `realtime-status-section` 中与「节点情况（总/生效）」相关的现有查询与卡逻辑，迁移到新概览卡。

## Risks / Trade-offs

- [kokonutui mouse-effect-card 默认样式与设计体系冲突：硬编码 `zinc`/`bg-white`、`max-w-md`、固定高度] → 落文件后覆盖为语义令牌（border 分层、无投影、全宽、内容自适应），并去掉默认品牌文案/CTA。
- [两个组件依赖注册表，若 shadcn registry 版本漂移影响 target 结构] → 以当前拉取内容落文件，锁定版本；实现阶段用 `pnpm` 校验依赖。
- [贡献图 `level` 需手动分档，与 `/stats/tasks/daily` 返回值无直接映射] → 在纯函数中按 count 四分位生成 level，并配单测。
- [贡献图窄屏可能横向溢出] → 外层 `@container` + `overflow-x-auto`，并提供 time range 快捷项。
- [右栏关注信息与左栏失败 Top 数据源重叠] → 二者用同一 query（`listTaskErrorStats`）或独立 query 但共用 queryKey，避免重复请求。实现阶段确认。

## Migration Plan

纯前端重构，无数据迁移。部署为一次 UI 发布；回滚即恢复 `dashboard-page.tsx` 旧布局，无状态字段变更。

## Open Questions

- 无。用户已确认 1A / 2AB / 3ABC / 4A，旧集群负载图按「去掉」处理（最终审视可改回）。

```

## docs/openspec/changes/dashboard-workbench-refactor/tasks.md

- Source: docs/openspec/changes/dashboard-workbench-refactor/tasks.md
- Lines: 1-40
- SHA256: 88e0f861b6bf8198b3467de8b67f31f89977954f032e2c284fb8f47fc099959e

```md
## 1. 组件与基础设施

- [ ] 1.1 将 `@kokonutui/mouse-effect-card` 落地到 `web/admin/src/components/kokonutui/mouse-effect-card.tsx`，改为使用语义令牌、全宽、内容自适应，去掉默认品牌文案/CTA 与固定高度，保留鼠标点阵动效。
- [ ] 1.2 将 `kibo-ui/contribution-graph` 落地到 `web/admin/src/components/kibo-ui/contribution-graph/index.tsx`，确认 `date-fns` 依赖可用。
- [ ] 1.3 为 contribution-graph 编写 `dailyToActivity`（`listTaskDailyStats` days → `{date,count,level}`，count 四分位分档 0–4）纯函数及单测。
- [ ] 1.4 新增工作台 i18n key（zh/en）：欢迎卡标题、快捷入口、数据大盘卡片标题、右栏关注信息标题与空态、窄屏说明。

## 2. 欢迎卡与根布局

- [ ] 2.1 重构 `web/admin/src/routes/_app/index.tsx` / `dashboard-page.tsx`：移除顶部 `title`/`desc` 区块，改为全宽欢迎卡 + 双栏容器（`grid lg:grid-cols-[minmax(0,1fr)_360px]`，窄屏降级单列）。
- [ ] 2.2 欢迎卡接入 `/setup/me`（`fetchCurrentUser`），展示 `nickname || username`；提供「新建工作流」「添加/管理节点」两个快捷入口并正确跳转。
- [ ] 2.3 将时间范围 state（复用 `TaskRangePicker`）上提到工作台根，作为区间型区块的统一驱动。

## 3. 数据大盘左栏

- [ ] 3.1 新增 `workbench-contribution-section`：全宽 contribution-graph 展示每日任务量，随全局范围刷新。
- [ ] 3.2 新增节点概览卡（总数/启用数，来自 `listEdges`）。
- [ ] 3.3 新增平均负载卡（`listFleetStats` 的 avg_cpu/mem/gpu）。
- [ ] 3.4 新增负载最高节点 Top5 卡（`listFleetStats.nodes` 按 cpu 降序取 top5）。
- [ ] 3.5 新增「工作流使用热度 Top + 任务耗时」左右布局（`listTaskCaseTopStats` limit 10 / `listTaskDailyStats` 的 avg_queue/exec/duration）。
- [ ] 3.6 新增「任务状态分布 + 错误 Top5」左右布局（`daily.summary` / `listTaskErrorStats` limit 5）。
- [ ] 3.7 每个区块保持独立 `useQuery` 与 `LoadingSkeleton`/`ErrorBanner` 失败隔离。

## 4. 右栏关注信息

- [ ] 4.1 新增 `workbench-attention-section`：右侧展示离线节点 / Comfy 未运行节点（`listEdges` + `listPresence` 交叉）。
- [ ] 4.2 展示未启用工作流（`listCases({ enabled: false })`），可跳转工作流列表。
- [ ] 4.3 展示失败任务 Top（`listTaskErrorStats`，与左栏错误 Top5 共用 query 或 queryKey 避免重复请求）。

## 5. 旧区块清理与样式

- [ ] 5.1 移除旧 `realtime-status-section` 的「集群实时负载图」bar chart、算力池汇总卡、每节点任务量卡；保留节点/在线相关逻辑并迁移到新概览卡。
- [ ] 5.2 按 `pixoma-design-system` 核对：只使用语义令牌与 `color-mix`、表面无投影、accent 每屏至多两次、图表用 chart-1…5、卡片 border 分层。
- [ ] 5.3 确保文案走 i18n（默认中文），无禁用词/感叹号/emoji。

## 6. 测试与验收

- [ ] 6.1 调整/新增 `dashboard` 相关组件与契约测试，覆盖新布局、贡献图映射、失败隔离、时间范围联动。
- [ ] 6.2 运行该前端项目的 `lint`、`build`、`test`，确保通过。
- [ ] 6.3 按 `pixoma-design-system` DESIGN.md 第 11 节 10 条验收清单过查工作台页。

```

## docs/openspec/changes/dashboard-workbench-refactor/specs/admin-dashboard-workbench/spec.md

- Source: docs/openspec/changes/dashboard-workbench-refactor/specs/admin-dashboard-workbench/spec.md
- Lines: 1-100
- SHA256: 4228ccba7516af0b23402932b6353c2a425eaa4c2691f712f586ee169a1952e5

[TRUNCATED]

```md
## Purpose
将后台首页从信息罗列式仪表盘升级为工作台视图，通过欢迎卡、数据大盘与右侧关注信息双栏，让运维在同屏了解集群健康、负载热点、任务趋势与待办异常，并能直接跳转到对应管理入口。

## ADDED Requirements

### Requirement: 工作台欢迎卡
系统 MUST 在工作台顶部展示一条全宽欢迎卡（基于 `@kokonutui/mouse-effect-card`），展示当前登录用户的昵称或用户名，并提供新建工作流、添加/管理节点两个快捷入口。欢迎卡 MUST NOT 复现独立的 `dashboard.title` / `dashboard.fullAccuracyNote` 标题说明区块。

#### Scenario: 展示当前用户昵称
- **WHEN** 运维登录后打开工作台首页
- **THEN** 欢迎卡展示 `/setup/me` 返回的 `nickname || username`

#### Scenario: 快捷入口可跳转
- **WHEN** 运维点击欢迎卡上的「新建工作流」或「添加/管理节点」
- **THEN** 跳转到对应的工作流新建页或节点管理页

### Requirement: 工作台双栏布局
系统 MUST 以左右双栏组织工作台主体：左栏为数据大盘，右栏为「需要关注的信息」。左栏内各统计区块 MUST 可独立加载与失败隔离，任一依赖 API 失败时仅该区块进入错误/空态，其它区块仍可展示。

#### Scenario: 打开工作台看到双栏
- **WHEN** 运维打开工作台首页
- **THEN** 页面呈现左栏数据大盘与右栏关注信息两栏

#### Scenario: 单个区块失败隔离
- **WHEN** 工作台某一统计依赖的 API 请求失败
- **THEN** 仅对应区块进入错误/空态，其它区块照常渲染

### Requirement: 每日任务量贡献图
在数据大盘顶部 MUST 全宽展示 `kibo-ui/contribution-graph`，以日历热力方式呈现所选时间范围内每天执行的任务量；数据 MUST 来自 `/api/v1/stats/tasks/daily`，为全量按天聚合。

#### Scenario: 展示每日任务量
- **WHEN** 运维打开工作台且统计接口可用
- **THEN** 贡献图按天展示任务量，无数据日期以空态填充

#### Scenario: 随全局范围刷新
- **WHEN** 运维切换近 7 / 30 / 90 天或自定义起止日期
- **THEN** 贡献图按所选范围重新查询并刷新

### Requirement: 节点概览卡片
数据大盘 MUST 提供节点情况卡片（总节点数、生效中节点数）、平均负载卡片、负载最高节点 Top5 卡片。负载数据 MUST 来自 `/stats/fleet`，节点总数与生效数 MUST 来自 `/edges`。平均负载与负载 Top5 MUST 按所选时间范围与实时口径一致展示。

#### Scenario: 展示节点总与生效中
- **WHEN** 运维打开工作台
- **THEN** 节点卡片展示节点总数与生效中节点数

#### Scenario: 展示平均负载
- **WHEN** 运维打开工作台且 fleet 数据可用
- **THEN** 平均负载卡片展示平均 CPU / 内存 / GPU 利用率

#### Scenario: 展示负载最高节点 Top5
- **WHEN** 运维打开工作台
- **THEN** 负载 Top5 卡片按负载从高到低列出最多 5 个节点

### Requirement: 工作流与任务效能区块
数据大盘 MUST 提供「工作流使用热度 Top」与「任务耗时」左右两列，且提供「任务状态分布」与「错误 Top5」左右两列。工作流热度 MUST 来自 `/stats/cases/top`，任务耗时 MUST 来自 `/stats/tasks/daily`，错误码 MUST 来自 `/stats/tasks/errors`（Top5）。

#### Scenario: 展示工作流使用热度 Top
- **WHEN** 运维打开工作台
- **THEN** 按热度从高到低展示最多 10 个工作流及其任务量与平均耗时

#### Scenario: 展示任务耗时
- **WHEN** 运维打开工作台
- **THEN** 展示所选范围内的任务耗时分布（排队 / 执行 / 平均耗时）

#### Scenario: 展示任务状态分布
- **WHEN** 运维打开工作台
- **THEN** 展示成功 / 失败 / 取消的状态分布

#### Scenario: 展示错误 Top5
- **WHEN** 运维打开工作台
- **THEN** 按错误码数量从高到低列出最多 5 个错误码

### Requirement: 右侧关注信息区
系统 MUST 在工作台右栏展示「需要关注的信息」，接入真实数据：离线节点 / Comfy 未运行节点、未启用的工作流、近段时间失败任务 Top。数据 MUST 来自 `/edges`、`/edges/presence`、`/cases` 与 `/stats/tasks/errors`。

#### Scenario: 展示离线节点
- **WHEN** 运维打开工作台且存在离线或 Comfy 未运行的节点
- **THEN** 右栏列出对应节点及其异常状态

#### Scenario: 展示未启用工作流

```

Full source: docs/openspec/changes/dashboard-workbench-refactor/specs/admin-dashboard-workbench/spec.md

## docs/openspec/changes/dashboard-workbench-refactor/specs/admin-resource-pages/spec.md

- Source: docs/openspec/changes/dashboard-workbench-refactor/specs/admin-resource-pages/spec.md
- Lines: 1-28
- SHA256: cb645f79c2cff2f97d54ea2447c98c8e055bc228d075a425afbcecd8901ad925

```md
## MODIFIED Requirements

### Requirement: Dashboard 中等总览
控制台 MUST 提供 Dashboard 页（工作台视图）：以欢迎卡 + 左栏数据大盘 / 右栏关注信息的方式组织；任务相关统计 MUST 来自专用统计接口（`/api/v1/stats/tasks/*`），为全量按天聚合，不再受列表接口 limit 样本限制；实例与 Case 汇总仍可来自既有列表类接口。该视图的详细展示需求见 `admin-dashboard-workbench` capability，此处仅保留全量口径、admin-api 通信与失败隔离约束。

#### Scenario: Dashboard 展示聚合信息
- **WHEN** 用户打开 Dashboard 且 admin-api 可用
- **THEN** 页面展示工作台视图（欢迎卡、节点/负载/工作流热度/任务耗时/状态分布/错误 Top5 等区块），网络请求指向 admin-api

#### Scenario: Dashboard 卡片失败隔离
- **WHEN** 某一汇总依赖的 API 请求失败
- **THEN** 仅对应卡片或区块进入错误/空态，其它区块仍可展示

#### Scenario: 任务统计不受样本限制
- **WHEN** 所选日期范围内实际任务数超过 200
- **THEN** 任务统计卡展示真实全量计数，而非列表样本计数

#### Scenario: 日期范围选择
- **WHEN** 运维选择起止日期或快捷范围（如近 7 / 30 / 90 天）
- **THEN** 区间型统计区块按所选范围重新查询并展示

#### Scenario: 分区展示实时与区间数据
- **WHEN** 运维打开 Dashboard
- **THEN** 页面按「数据大盘 / 关注信息」展示；实时状态区（节点启用、在线状态）不随时间范围变化，区间型统计区统一跟随顶部时间范围控件刷新

#### Scenario: 全局时间范围
- **WHEN** 运维在 Dashboard 顶部切换近 7 / 30 / 90 天或选择起止日期
- **THEN** 所有区间型统计区块按同一范围重新查询

```
