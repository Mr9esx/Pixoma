---
comet_change: dashboard-workbench-refactor
role: technical-design
canonical_spec: openspec
---

# Dashboard Workbench 深度技术设计

## Context

现有 `/` 页由 `web/admin/src/features/dashboard/dashboard-page.tsx` 驱动，纵向堆叠「实时状态 / 任务效能 / 业务分析」三个 section，顶部单独展示 `dashboard.title` 与 `dashboard.fullAccuracyNote`。数据源已齐备：`/api/v1/stats/tasks/daily`、`/stats/tasks/errors`、`/stats/tasks/edges`、`/stats/cases/top`、`/stats/fleet`、`/edges`、`/edges/presence`、`/cases`、`/setup/me`，前端类型见 `lib/api/types.ts`。本次仅重构前端信息架构与组件布局，不新增后端接口。视觉遵循 `pixoma-design-system`：中性灰语义令牌、表面无投影、accent 每屏至多两次、图表 chart-1…5、失败隔离、中英 i18n 默认中文。

## Goals / Non-Goals

**Goals:**
- 移除页面顶部独立 `title` / `desc` 区块。
- 全宽欢迎卡（kokonutui mouse-effect-card 点阵动效），欢迎语取当前登录用户昵称，含两个快捷入口。
- 左栏数据大盘：全宽贡献图 + 三张概览卡 + 两组双列图表。
- 右栏欢迎卡：欢迎语 + 新建工作流 / 添加节点快捷入口（左对齐、垂直居中）。
- 全局时间范围控件驱动区间型区块；实时/节点状态不随时间变化。
- 复用现有 admin-api，保持全量口径，所有区块失败隔离。

**Non-Goals:**
- 不修改后端聚合逻辑或新增接口。
- 不改动侧边栏、路由体系、`/` 路由入口。
- 不引入前端 mock 作为交付路径。
- 不新增截图/浏览器截图类测试脚手架。

## 布局架构

```
/ 工作台页
└── div @container/content flex-1
    └── div grid lg:grid-cols-[minmax(0,1fr)_360px] gap-4
        ├── 主机栏 space-y-4
        │   ├── WorkbenchContribution（kibo-ui 贡献图，全宽，全年）
        │   ├── grid sm:grid-cols-3
        │   │     ├── 节点情况（总 / 生效中）
        │   │     ├── 平均负载（avg_cpu/mem/gpu）
        │   │     └── 负载最高节点 Top5
        │   ├── grid lg:grid-cols-2（工作流使用热度 Top | 任务耗时）
        │   ├── grid lg:grid-cols-2（任务状态分布 | 错误 Top5）
        │   └── TaskRangePicker（数据大盘底部，仅驱动区间图表）
        └── 右栏 WorkbenchWelcomeCard（mouse-effect-card 点阵动效，左对齐、垂直居中）
```

- 根容器复用现有 `contentRegionClassName`（`@container/content`），保证窄屏降级单列。
- 双栏用 CSS grid：`lg:grid-cols-[minmax(0,1fr)_360px]`；右栏固定 360px，窄屏回退为单列（右栏落到底部）。右栏为欢迎卡。
- 所有区块独立 `useQuery`，统一 `LoadingSkeleton` / `ErrorBanner` 失败隔离。

## 数据流与联动

| 区块 | 数据源 | 是否随全局范围联动 |
|---|---|---|
| 欢迎卡昵称 | `/setup/me`（`fetchCurrentUser`） | 否（实时） |
| 贡献图 | `/stats/tasks/daily`（本年度 1/1-今天）→ `days[].processed` → `{ date, count, level }` | 否（全年） |
| 节点情况卡 | `/edges`（总数/启用数） | 否（实时） |
| 平均负载卡 | `/stats/fleet`（avg_cpu/mem/gpu） | 否（实时） |
| 负载 Top5 卡 | `/stats/fleet`（`nodes[]` 按 cpu 降序取 5） | 否（实时） |
| 工作流热度 Top | `/stats/cases/top`（limit 10） | 是 |
| 任务耗时 | `/stats/tasks/daily`（avg_queue/exec/duration） | 是 |
| 任务状态分布 | `/stats/tasks/daily`（summary） | 是 |
| 错误 Top5 | `/stats/tasks/errors`（limit 5） | 是 |
| 右栏欢迎卡 | `/setup/me`（昵称/用户名） | 否（实时） |

- 全局 `range` state 复用 `TaskRangePicker`（近 7/30/90 天 + 自定义），由工作台根持有，传给 `WorkbenchDataBoard`，控件渲染在数据大盘底部。
- 仅区间型图表（工作流热度、任务耗时、状态分布、错误 Top5）用 `queryKeys.stats.*` 派生缓存键并随 `range` 刷新；贡献图与节点概览卡使用固定/全年 queryKey，不受范围影响。
- 卡片采用 summary 样式：`CardHeader` 内标题 + 关键指标数字，图表用 `chart-1…5` 语义色；任务耗时图用渐变填充 `AreaChart`。

## 关键组件设计

### 1. WorkbenchWelcomeCard
- 基于落文件的 `kokonutui/mouse-effect-card.tsx`，覆盖：
  - 外层 `card`：去掉内置 `max-w-md`、`h-[400px]`、`bg-white` 光晕、默认品牌文案与促销 CTA；用 `border` 分层 + `shadow-none`（设计体系无投影）。
  - 点阵动效保留（`motion`），作为背景层，`dots` 颜色改用语义令牌（`muted-foreground` / `color-mix`）。
  - 内容（作为 `children` 自定义）：标题 "欢迎回来，{nickname || username}"（无多余副标题），下方 `Button` 快捷入口（新建工作流 → case 新建页；添加/管理节点 → edges 页）；整体左对齐、垂直居中。
  - 保持 `ariaLabel`、键盘可达（`onKeyDown` 方向键移动点阵焦点）。

### 2. WorkbenchContribution
- 数据：`listTaskDailyStats(range)` → `days`。
- 映射纯函数 `dailyToActivity(days)`：输出 `Activity[]`，`count = processed`，`level` 按 count 相对量分档（0–4）。分档建议按 min/max 归一化后等分，或用分位数；纯函数便于单测。
- 渲染：`<ContributionGraph><ContributionGraphCalendar>{...}</ContributionGraphCalendar><ContributionGraphLegend /></ContributionGraph>`，外层 `overflow-x-auto` 防窄屏溢出。
- `labels.totalCount` 用中文 i18n 覆盖默认英文。

### 3. 三张概览卡（节点情况 / 平均负载 / 负载 Top5）
- 节点情况：`listEdges`，卡片显示 `enabled / total` + 进度条（保留现有 `realtime-edges-card` 逻辑）。
- 平均负载：`listFleetStats`，显示 `avg_cpu_usage_percent`、`avg_mem_usage_percent`、`avg_gpu_usage_percent`（缺失显示 `—`）。
- 负载 Top5：`listFleetStats.nodes` 按 `cpu_usage_percent` 降序取 5，紧凑列表 + 进度条，标注节点 id 与负载。

### 4. 双列图表组
- 工作流使用热度 Top（`recharts` Bar / 分级列表，来自 `listTaskCaseTopStats`） | 任务耗时（`ComposedChart`，queue/exec/duration，来自 daily）。
- 任务状态分布（`Pie` donut，summary） | 错误 Top5（分级列表或 donut，`listTaskErrorStats` limit 5）。
- 一律用 chart-1…5 语义色；数据必须填充编码，单卡片失败隔离。

### 5. 右侧欢迎卡（右栏）
- 右栏渲染 `WorkbenchWelcomeCard`：欢迎语（`/setup/me` 的 `nickname || username`）+ 两个快捷入口（新建工作流 → `/cases`、添加/管理节点 → `/edges`）。
- 暂不接入关注信息数据；后续若需补充关注信息，再扩展右栏卡片内容。

## 失败隔离与空态

- 每个区块独立 `useQuery`，独立 `isLoading` / `isError`；非加载/错误时展示空态文案（i18n）。
- 任一区块 API 失败仅该区块显示 `ErrorBanner`（含重试），其它区块照常渲染。
- 不因某区块失败而整页熔断。

## i18n 与文案

- 新增 key 全部写入 `zh.json` / `en.json`（默认中文）。涉及：欢迎卡标题、快捷入口、数据大盘卡片标题、贡献图全年说明、任务量单位。
- 语气遵循 `voice-profile.md`：action-first、无禁用词（请/烦请/前往/进行/完成/实施/温馨提示）、无感叹号、无 emoji。

## 样式与设计体系约束

- 只用 `colors_and_type.css` / `build/theme.css` 语义令牌，禁止新增裸 hex；派生色用 `color-mix(in oklch, …)`。
- 表面无投影（`shadow-none`），仅对话框/dropdown/popover 等浮层保留 `shadow-md` 以上。
- accent（陶土橙）每屏至多出现两次；作小号文字/点缀用深档 `--accent-text`。
- 图表用 `chart-1…5` 语义色 + `ChartContainer`（recharts）。
- 组件形状优先复用 shadcn/ui 已装组件（`components/ui/*`）。

## 测试策略

- 纯函数单测：
  - `dailyToActivity`：count → level 分档边界（空数组、零值、最大/最小）。
  - `topByUsage`：fleet nodes 按 cpu 降序取 top5 稳定性。
  - `pickAttention`：节点交叉（在线/Comfy 未运行）、未启用工作流、失败 Top 解析。
- 组件/契约测试：
  - 工作台根渲染欢迎卡 + 双栏。
  - 贡献图映射与联动刷新。
  - 右栏欢迎卡（标题 + 快捷入口，无多余副标题）。
  - 失败隔离（单区块 API 失败不影响其它）。
- 全量校验：`pnpm lint`、`pnpm build`、`pnpm test` 通过；`pixoma-design-system` DESIGN.md 第 11 节 10 条验收。
- 不使用截图类测试脚手架（遵循项目"模型不支持截图"约定）。

## Risks / Trade-offs

- [kokonutui 组件硬编码样式冲突] → 落文件后覆盖为语义令牌、无投影、全宽、内容自适应；保留动效但弱化背景色对比。
- [贡献图 `level` 需手动分档，与 daily 返回值无直接映射] → 纯函数四分位分档 + 单测。
- [欢迎卡右侧窄列下标题/按钮可能换行] → children 用 `flex flex-wrap` 自适应，标题左对齐、内容垂直居中。
- [贡献图窄屏横向溢出] → 外层 `@container` + `overflow-x-auto`。
- [去掉旧集群负载图] → 由"平均负载 + 负载 Top5"覆盖负载维度（已与用户确认）。

## Migration Plan

纯前端重构，无数据迁移或后端变更。部署为一次 UI 发布；回滚即恢复 `dashboard-page.tsx` 旧布局。

## Open Questions

- 无。三个设计决策（贡献图全年、欢迎卡保留点阵动效、右侧欢迎卡替代占位卡）已与用户确认。
