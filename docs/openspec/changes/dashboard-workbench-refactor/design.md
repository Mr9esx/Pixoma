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
