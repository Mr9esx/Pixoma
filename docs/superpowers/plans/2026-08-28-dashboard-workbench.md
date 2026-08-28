---
change: dashboard-workbench-refactor
design-doc: docs/superpowers/specs/2026-08-28-dashboard-workbench-design.md
base-ref: 0b6756c6d1b7c6f9fe4ef03e0482e8a91feb93e8
---

# Dashboard Workbench Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把 `/` 仪表盘重构为工作台，移除顶部标题/说明，改造成欢迎卡 + 左数据大盘 / 右关注信息双栏，并保留全局时间范围联动。

**Architecture:** 纯前端重构 `web/admin`。新增全宽欢迎卡（kokonutui mouse-effect-card 点阵动效 + 用户昵称 + 两个快捷入口）与数据大盘左栏（kibo-ui 贡献图 + 三张概览卡 + 两组双列图表），右侧为分区关注信息列表。全局时间范围 state 上提到工作台根，区间型区块经 `queryKeys.stats.*` 联动；实时/节点状态固定 queryKey 不受范围影响。所有区块独立 `useQuery` 失败隔离，遵循 `pixoma-design-system`。

**Tech Stack:** pnpm + Vite + TanStack Router + shadcn/ui + Tailwind v4 + recharts + motion + i18next（zh/en，默认中文） + vitest。

## Global Constraints

- 后台前端遵循 `pixoma-design-system`：只用 `web/admin/src/styles/theme.css` 语义令牌，禁止新增裸 hex；派生色用 `color-mix(in oklch, …)`；表面无投影（`shadow-none`），浮层才允许 shadow；accent 每屏至多两次；图表用 `chart-1…5` + `ChartContainer`。
- 组件优先复用 `web/admin/src/components/ui/*` 已装 shadcn 组件；新组件用项目注册表落地到 `components/kokonutui` / `components/kibo-ui`。
- 文案走 i18n（zh/en），默认中文；action-first，禁禁用词（请/烦请/前往/进行/完成/实施/温馨提示）、无感叹号、无 emoji、不以"用户"当称呼。
- 禁止新增截图类或浏览器截图测试脚手架。
- TDD：纯函数与组件/契约测试先写失败再实现。
- 运行校验：`pnpm lint`、`pnpm build`、`pnpm test` 需通过（在 `web/admin` 内执行）。

---

### Task 1: 落地 kibo-ui contribution-graph 组件

**Files:**
- Create: `web/admin/src/components/kibo-ui/contribution-graph/index.tsx`
- Test: `web/admin/src/features/dashboard/daily-to-activity.test.ts`

**Interfaces:**
- Consumes: `@/lib/api/types` 的 `TaskDailyStat`；`date-fns`。
- Produces: `ContributionGraph`、`ContributionGraphCalendar`、`ContributionGraphLegend`（从注册表落地）；`dailyToActivity(days: TaskDailyStat[]): Activity[]`（Activity = `{ date: string; count: number; level: number }`）。

- [x] **Step 1: 拉取注册表组件**

用 curl 拉取 `https://kibo-ui.com/r/contribution-graph.json` 的 `files[0].content`，按其 `target` 写入 `web/admin/src/components/kibo-ui/contribution-graph/index.tsx`。确认 `cn` 来自 `@/lib/utils`，`date-fns` 已在依赖。若 target 结构与本描述不符，按返回 target 为准并记录。

- [x] **Step 2: 写 dailyToActivity 失败测试**

创建 `web/admin/src/features/dashboard/daily-to-activity.ts` 前先写 `daily-to-activity.test.ts`：

```ts
import { describe, expect, it } from 'vitest'
import { dailyToActivity } from './daily-to-activity'
import type { TaskDailyStat } from '@/lib/api/types'

function day(date: string, processed: number): TaskDailyStat {
  return { date, processed, succeeded: 0, failed: 0, cancelled: 0, avg_duration_ms: null, avg_queue_ms: null, avg_exec_ms: null }
}

describe('dailyToActivity', () => {
  it('maps processed to count and level, generating 0-4 levels', () => {
    const days = [day('2026-08-01', 0), day('2026-08-02', 30), day('2026-08-03', 60), day('2026-08-04', 90), day('2026-08-05', 120)]
    const acts = dailyToActivity(days)
    expect(acts).toHaveLength(5)
    expect(acts[0]).toMatchObject({ date: '2026-08-01', count: 0, level: 0 })
    expect(acts[1].level).toBeGreaterThan(0)
    expect(acts[4].level).toBe(4)
  })
  it('returns empty array for empty input', () => {
    expect(dailyToActivity([])).toEqual([])
  })
  it('handles null avg fields and zero-only input', () => {
    const acts = dailyToActivity([day('2026-08-01', 0), day('2026-08-02', 0)])
    expect(acts.every((a) => a.level === 0)).toBe(true)
  })
})
```

- [x] **Step 3: 运行测试确认失败**

Run: `cd web/admin && pnpm vitest run src/features/dashboard/daily-to-activity.test.ts`
Expected: FAIL（"dailyToActivity" 未定义）。

- [x] **Step 4: 实现 dailyToActivity**

创建 `web/admin/src/features/dashboard/daily-to-activity.ts`：

```ts
import type { TaskDailyStat } from '@/lib/api/types'

export type Activity = { date: string; count: number; level: number }

export function dailyToActivity(days: TaskDailyStat[]): Activity[] {
  if (days.length === 0) return []
  const max = Math.max(...days.map((d) => d.processed))
  return days.map((d) => {
    const count = d.processed
    const level = max <= 0 ? 0 : Math.min(4, Math.round((count / max) * 4))
    return { date: d.date, count, level }
  })
}
```

- [x] **Step 5: 运行测试确认通过**

Run: `cd web/admin && pnpm vitest run src/features/dashboard/daily-to-activity.test.ts`
Expected: PASS。

- [x] **Step 6: 提交**

```bash
git add web/admin/src/components/kibo-ui/contribution-graph/index.tsx web/admin/src/features/dashboard/daily-to-activity.ts web/admin/src/features/dashboard/daily-to-activity.test.ts
git commit -m "feat: add kibo-ui contribution graph and daily activity mapper"
```

---

### Task 2: 落地 kokonutui mouse-effect-card 并适配设计体系

**Files:**
- Create: `web/admin/src/components/kokonutui/mouse-effect-card.tsx`

**Interfaces:**
- Consumes: `motion`、`@/components/ui/card`、`@/components/ui/button`、`@/lib/utils`。
- Produces: `MouseEffectCard`（`{ className?, title, subtitle?, children, primaryCtaText?, primaryCtaUrl?, secondaryCtaText?, secondaryCtaUrl?, footerText?, ... }`，default export）。

- [x] **Step 1: 拉取注册表组件**

用 curl 拉取 `https://kokonutui.com/r/mouse-effect-card.json` 的 `files[0].content`，按其 `target` 写入 `web/admin/src/components/kokonutui/mouse-effect-card.tsx`。

- [x] **Step 2: 覆盖样式为语义令牌**

在落文件基础上：
- 外层 `Card` 去掉 `max-w-md` 与 `p-0`，改为 `w-full`；保留 `overflow-hidden rounded-2xl border`，去掉 `shadow-none` 之外的投影（本身无 shadow）。
- 去掉默认品牌文案默认值（`title="Acme"`、`subtitle="Build interfaces..."`、`topText="Case Study"`、`primaryCtaText="Get Started"`、`footerText="We do it all"` 等），改为可空、由使用方传入。
- 去掉 `CardContent` 的固定 `h-[400px]`，改为自适应（`min-h` 由使用方定）。
- `dots` 的 `bg-zinc-400 dark:bg-zinc-600` 改为 `bg-muted-foreground/30` 之类的语义色；`bg-white`/`bg-zinc-950` 光晕改为 `color-mix(in oklch, var(--foreground) 8%, transparent)`。
- 保留 `motion`、`ResizeObserver`、鼠标/键盘交互与 `ariaLabel`。

- [x] **Step 3: 为适配写契约测试并加入 vitest include**

新增 `web/admin/src/components/kokonutui/mouse-effect-card.contract.test.ts`（读源码断言：去默认品牌文案、语义令牌、全宽自适应、保留动效），并加入 `vitest.config.ts` include。运行 `pnpm vitest run src/components/kokonutui/mouse-effect-card.contract.test.ts` 通过。

- [x] **Step 4: 提交**

```bash
git add web/admin/src/components/kokonutui/mouse-effect-card.tsx
git commit -m "feat: add kokonutui mouse-effect card with semantic tokens"
```

---

### Task 3: 新增工作台 i18n 文案

**Files:**
- Modify: `web/admin/src/lib/i18n/locales/zh.json`
- Modify: `web/admin/src/lib/i18n/locales/en.json`

**Interfaces:**
- Produces: `dashboard.workbench.*` 命名空间，供各组件 `useTranslation()` 使用。

- [x] **Step 1: 在 zh.json 的 `dashboard` 下新增 workbench key**

```json
{
  "welcomeTitle": "欢迎回来，{{name}}",
  "welcomeSubtitle": "今天想从哪开始",
  "quickCreateCase": "新建工作流",
  "quickManageEdges": "添加/管理节点",
  "taskHeatTitle": "每日任务量",
  "nodeOverviewTitle": "节点情况",
  "nodeTotal": "总",
  "nodeEnabled": "生效中",
  "avgLoadTitle": "平均负载",
  "topLoadTitle": "负载最高节点 Top5",
  "workflowTopTitle": "工作流使用热度 Top",
  "taskDurationTitle": "任务耗时",
  "statusDistributionTitle": "任务状态分布",
  "errorTopTitle": "错误 Top5",
  "attentionTitle": "需要关注",
  "nodeAbnormalTitle": "节点异常",
  "disabledWorkflowTitle": "未启用工作流",
  "failedTaskTopTitle": "失败任务 Top",
  "allHealthy": "全部正常",
  "allEnabled": "已全部启用",
  "noFailedTasks": "暂无失败任务"
}
```

- [x] **Step 2: 在 en.json 的 `dashboard` 下补对应英文 key**（无禁用词、action-first）。

- [x] **Step 3: 验证 i18n 键对齐（zh/en 21/21）且通过 copy-quality/locale 测试**

Run: `cd web/admin && node -e "const z=require('./src/lib/i18n/locales/zh.json').dashboard, e=require('./src/lib/i18n/locales/en.json').dashboard; const zk=Object.keys(z).filter(k=>/^workbench|welcome|quick|taskHeat|nodeOverview|avgLoad|topLoad|workflowTop|taskDuration|statusDistribution|errorTop|attention|nodeAbnormal|disabledWorkflow|failedTaskTop|allHealthy|allEnabled|noFailedTasks/.test(k)); const miss=zk.filter(k=>!(k in e)); console.log('missing en keys:', miss)"`
Expected: `missing en keys: []`。

- [ ] **Step 4: 提交（注：i18n 文件含用户既有未提交改动，提交边界在 Task 9 统一处理，暂不 commit）**

```bash
git add web/admin/src/lib/i18n/locales/zh.json web/admin/src/lib/i18n/locales/en.json
git commit -m "feat: add workbench i18n copy"
```

---

### Task 4: 重构工作台根组件与路由入口

**Files:**
- Modify: `web/admin/src/features/dashboard/dashboard-page.tsx`
- Modify: `web/admin/src/routes/_app/index.tsx`
- Test: `web/admin/src/features/dashboard/workbench-page.test.tsx`

**Interfaces:**
- Consumes: `WorkbenchWelcomeCard`、`WorkbenchDataBoard`、`WorkbenchAttention`（后续任务产出）、`TaskRangePicker`、`fetchCurrentUser`。
- Produces: `DashboardPage`（数据渲染根），接收/持有全局 `range` state。

- [x] **Step 1: 重构 DashboardPage**

将 `dashboard-page.tsx` 改为：
- `const [range, setRange] = useState<StatsRange>({ from: daysAgo(29), to: daysAgo(0) })`（复用现有 `TaskRangePicker` / `date-range`）。
- 顶部门 `MouseEffectCard`（全宽）包裹欢迎区；其下 `grid lg:grid-cols-[minmax(0,1fr)_360px] gap-4`，左 `WorkbenchDataBoard range={range}`，右 `WorkbenchAttention range={range}`。
- 移除原 `dashboard.title` / `dashboard.fullAccuracyNote` 的 `<h1><p>` 区块。

- [x] **Step 2: 写失败测试**

`workbench-page.test.tsx` 渲染 `DashboardPage`，断言：无 `dashboard.title` 文本；存在欢迎卡区域测试 id `workbench-welcome`；存在双栏容器 `workbench-grid` 与左/右 test id。若 `workbench-welcome` / `workbench-grid` 尚未实现，先以占位 test id 为准并在后续任务实现。测试用 vitest browser + `@testing-library/react`（如已有）。若项目用其它测试栈，遵循现有 dashboard contract test 风格。

- [x] **Step 3: 更新路由入口**

`routes/_app/index.tsx` 保持 `component: DashboardPage` 不变（确认已有）。

- [x] **Step 4: 运行测试）

Run: `cd web/admin && pnpm vitest run src/features/dashboard/workbench-page.test.tsx`

- [x] **Step 5: 提交**

```bash
git add web/admin/src/features/dashboard/dashboard-page.tsx web/admin/src/features/dashboard/workbench-page.test.tsx web/admin/src/routes/_app/index.tsx
git commit -m "feat: rework dashboard into workbench root layout"
```

---

### Task 5: 实现欢迎卡

**Files:**
- Create: `web/admin/src/features/dashboard/workbench-welcome-card.tsx`
- Test: `web/admin/src/features/dashboard/workbench-welcome-card.test.tsx`

**Interfaces:**
- Consumes: `MouseEffectCard`、`fetchCurrentUser`（`/setup/me`）、`useTranslation`、`@/components/ui/button`。
- Produces: `WorkbenchWelcomeCard`。

- [x] **Step 1: 写失败测试**

`workbench-welcome-card.test.tsx`：mock `fetchCurrentUser` 返回 `{ username: 'ops', nickname: '运维' }`，断言渲染 `欢迎回来，运维`（i18n `welcomeTitle` 用 name 插值），存在两个快捷入口按钮测试 id `workbench-quick-create-case` / `workbench-quick-manage-edges`。

- [x] **Step 2: 运行确认失败**

Run: `cd web/admin && pnpm vitest run src/features/dashboard/workbench-welcome-card.test.tsx`
Expected: FAIL。

- [x] **Step 3: 实现**

```tsx
export function WorkbenchWelcomeCard() {
  const { t } = useTranslation()
  const { data: user, isLoading } = useQuery({ queryKey: ['me'], queryFn: fetchCurrentUser })
  const name = user?.nickname || user?.username || ''
  return (
    <MouseEffectCard data-testid="workbench-welcome" className="w-full" title={t('dashboard.workbench.welcomeTitle', { name })} subtitle={t('dashboard.workbench.welcomeSubtitle')}>
      <div className="flex items-center gap-3">
        <Button data-testid="workbench-quick-create-case" asChild><Link to="/cases">{t('dashboard.workbench.quickCreateCase')}</Link></Button>
        <Button data-testid="workbench-quick-manage-edges" asChild variant="outline"><Link to="/edges">{t('dashboard.workbench.quickManageEdges')}</Link></Button>
      </div>
    </MouseEffectCard>
  )
}
```

（若 `MouseEffectCard` 通过 `children` 承载右侧按钮，则按钮放 children；否则用其 props。以落文件后的实际 API 为准，保证测试 id。）

- [x] **Step 4: 运行测试通过**

Run: `cd web/admin && pnpm vitest run src/features/dashboard/workbench-welcome-card.test.tsx`
Expected: PASS。

- [x] **Step 5: 提交**

```bash
git add web/admin/src/features/dashboard/workbench-welcome-card.tsx web/admin/src/features/dashboard/workbench-welcome-card.test.tsx
git commit -m "feat: add workbench welcome card"
```

---

### Task 6: 实现数据大盘左栏（贡献图 + 三概览卡 + 双列图表）

**Files:**
- Create: `web/admin/src/features/dashboard/workbench-data-board.tsx`
- Create: `web/admin/src/features/dashboard/workbench-overview-cards.tsx`（节点情况 / 平均负载 / 负载 Top5）
- Create: `web/admin/src/features/dashboard/workbench-chart-pairs.tsx`（工作流热度+任务耗时 / 状态分布+错误Top5）
- Test: `web/admin/src/features/dashboard/workbench-data-board.test.tsx`

**Interfaces:**
- Consumes: `dailyToActivity`、`listTaskDailyStats`、`listTaskEdgeStats`、`listFleetStats`、`listTaskCaseTopStats`、`listTaskErrorStats`、`listEdges`、`queryKeys`、`TaskRangePicker` 的 `StatsRange`。
- Produces: `WorkbenchDataBoard({ range })`，含 `WorkbenchContribution`、`WorkbenchOverviewCards`、`WorkbenchChartPairs` 子组件。

- [x] **Step 1: 写失败测试（组件级）**

`workbench-data-board.test.tsx`：mock 各 API，断言渲染以下 test id：`workbench-contribution`、`workbench-node-overview`、`workbench-avg-load`、`workbench-top-load`、`workbench-workflow-top`、`workbench-task-duration`、`workbench-status-distribution`、`workbench-error-top`。mock 一个 API 返回 reject，断言该区块显示 `ErrorBanner` 且其它区块仍渲染。

- [x] **Step 2: 运行确认失败**

Run: `cd web/admin && pnpm vitest run src/features/dashboard/workbench-data-board.test.tsx`

- [x] **Step 3: 实现 WorkbenchContribution**

`useQuery({ queryKey: queryKeys.stats.tasksDaily(range.from, range.to), queryFn: () => listTaskDailyStats(range) })`，`const acts = dailyToActivity(days)`，渲染 `<ContributionGraph data={acts}><ContributionGraphCalendar>...</ContributionGraphCalendar><ContributionGraphLegend/></ContributionGraph>`，外层 `overflow-x-auto`。

- [x] **Step 4: 实现 WorkbenchOverviewCards**

- 节点情况：`listEdges` → `enabled/total` + 进度条。
- 平均负载：`listFleetStats` → `avg_cpu_usage_percent`、`avg_mem_usage_percent`、`avg_gpu_usage_percent`（null 显示 `—`）。
- 负载 Top5：`listFleetStats.nodes` 按 `cpu_usage_percent` 降序取 5，紧凑列表 + 进度条（`topByUsage` 纯函数可在 `daily-to-activity.ts` 或独立文件，含单测）。若需单测，任务内同步写 `top-by-usage.test.ts`。

- [x] **Step 5: 实现 WorkbenchChartPairs**

- 工作流热度 Top：`listTaskCaseTopStats({ ...range, limit: 10 })`，`recharts` Bar。
- 任务耗时：`listTaskDailyStats(range)`，`ComposedChart`（queue/exec/duration）。
- 状态分布：`daily.summary`，`Pie` donut。
- 错误 Top5：`listTaskErrorStats({ ...range, limit: 5 })`，分级列表或 donut。
- 全部用 chart-1…5 + `ChartContainer`。

- [x] **Step 6: 运行测试通过**

Run: `cd web/admin && pnpm vitest run src/features/dashboard/workbench-data-board.test.tsx`
Expected: PASS。

- [x] **Step 7: 提交**

```bash
git add web/admin/src/features/dashboard/workbench-data-board.tsx web/admin/src/features/dashboard/workbench-overview-cards.tsx web/admin/src/features/dashboard/workbench-chart-pairs.tsx web/admin/src/features/dashboard/workbench-data-board.test.tsx
git commit -m "feat: add workbench data board"
```

---

### Task 7: 实现右栏关注信息

**Files:**
- Create: `web/admin/src/features/dashboard/workbench-attention.tsx`
- Test: `web/admin/src/features/dashboard/workbench-attention.test.tsx`

**Interfaces:**
- Consumes: `listEdges`、`listPresence`、`listCases`、`listTaskErrorStats`、`queryKeys`、`StatsRange`、`useTranslation`。
- Produces: `WorkbenchAttention({ range })`。

- [x] **Step 1: 写失败测试**

`workbench-attention.test.tsx`：mock `listEdges`/`listPresence` 使一个节点 `edge_online=false`；mock `listCases` 返回一个 enabled:false；mock `listTaskErrorStats` 返回 items。断言：`workbench-node-abnormal` 节含该节点；`workbench-disabled-workflow` 节含工作流名；`workbench-failed-task` 节含错误码。空态：全部节点在线、无未启用工作流时显示 `workbench-all-healthy` / `workbench-all-enabled` 空态文案。

- [x] **Step 2: 运行确认失败**

Run: `cd web/admin && pnpm vitest run src/features/dashboard/workbench-attention.test.tsx`

- [x] **Step 3: 实现**

- 节点异常：`listEdges` + `listPresence` 交叉，`!p.edge_online || !p.comfy_running` 计入；列表项可点击跳 edges。
- 未启用工作流：`listCases({ enabled: false })`；可点击跳 `cases` 列表。
- 失败任务 Top：复用 `listTaskErrorStats({ ...range, limit: 5 })`（queryKey 与左栏错误 Top5 一致），展示错误码 + 数量，可点击跳 `tasks`。
- 分区列表 + 状态点/徽标；空态文案用 i18n。

- [x] **Step 4: 运行测试通过）

Run: `cd web/admin && pnpm vitest run src/features/dashboard/workbench-attention.test.tsx`
Expected: PASS。

- [x] **Step 5: 提交**

```bash
git add web/admin/src/features/dashboard/workbench-attention.tsx web/admin/src/features/dashboard/workbench-attention.test.tsx
git commit -m "feat: add workbench attention panel"
```

---

### Task 8: 清理旧 dashboard 区块

**Files:**
- Delete: `web/admin/src/features/dashboard/realtime-status-section.tsx`（或保留但移除集群负载图/算力池/每节点任务量卡）
- Modify: `web/admin/src/features/dashboard/dashboard-page.tsx`（若已替换则不重复）
- Modify: `web/admin/src/features/dashboard/task-stats-section.tsx`、`case-analysis-section.tsx`（若已由新组件替代则删除不再引用）

**Interfaces:**
- Produces: 工作台页面不再引用已删除区块。

- [x] **Step 1: 删除/精简旧区块**

删除 `realtime-status-section.tsx`、`task-stats-section.tsx`、`case-analysis-section.tsx`（或移除其中被替代的图表卡），确保 `dashboard-page.tsx` 与新组件不再 import 它们。

- [x] **Step 2: 清理无用依赖与测试**

移除对已删区块的测试文件引用（如 `task-stats.contract.test.ts`、`case-analysis` 相关），保留 `aggregate.ts` 等仍被使用的工具。

- [x] **Step 3: 提交**

```bash
git add -A web/admin/src/features/dashboard
git commit -m "refactor: remove legacy dashboard sections"
```

---

### Task 9: 全量校验与设计验收

**Files:**
- 无新增（只运行校验）

- [x] **Step 1: lint（workbench 文件通过；项目级 19 errors 均为用户未提交重构引起，非本 change）**

Run: `cd web/admin && pnpm lint`
Expected: 通过（如失败修复）。

- [ ] **Step 2: build（阻塞：项目整体 build 失败，error 全部来自用户未提交改动：app-sidebar/field-cards/sessions-list-panel/tasks-list-panel，本 change 文件无 TS error。需用户确认处理边界后再推进。**

Run: `cd web/admin && pnpm build`
Expected: 通过。

- [x] **Step 3: test（workbench 相关测试全部通过；唯一失败 shell-layout.contract.test 因用户未完成语言切换器重构 ENOENT，非本 change）**

Run: `cd web/admin && pnpm test`
Expected: 通过。

- [x] **Step 4: 设计体系验收（语义令牌、无投影、chart 语义色、i18n 中英、action-first 已核对，workbench 文件满足）**

按 `pixoma-design-system` DESIGN.md 第 11 节 10 条核对：语义令牌、无投影、accent≤2 次、chart 语义色、i18n 中英默认中文、action-first 无禁用词。任何不符就地修复。

- [ ] **Step 5: 提交剩余变更（i18n 文件含用户既有未提交改动，待用户确认提交边界）**

```bash
git add -A
git commit -m "chore: verify workbench and run design acceptance"
```

---

## 调整记录（用户 4 点反馈）

- [x] 1. 图表卡改为参考 Sales Overview 的 summary 风格：标题 + 关键指标数字 + 图表；任务耗时改为渐变面积图。
- [x] 2. 欢迎卡文字左对齐、垂直居中（MouseEffectCard 增加 `align` prop）。
- [x] 3. 去除右栏真实关注信息，改为占位卡（信息待补充）；时间控件不再置于顶部。
- [x] 4. 每日任务量贡献图改为展示本年度（1/1-今天），不受时间控件；节点情况/平均负载/负载 Top5 保持实时；时间控件移至数据大盘底部，仅驱动区间图表。

相关实现与测试已通过（workbench 契约测试 24/24），源码已提交（commit `20496be` 及后续）。
