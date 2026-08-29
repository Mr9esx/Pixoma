---
change: link-health-visibility
design-doc: docs/superpowers/specs/2026-08-24-link-health-visibility-design.md
base-ref: cbdc9fb
archived-with: link-health-visibility
---

# 链路健康提示与下一步行动 实施计划（Phase 1）

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 Case / Topic / 计算节点（Edge）三页提供统一的依赖健康提示：结论（能用/不能用）→ 断点（原因 + 配置/运行分类）→ 下一步（去哪修 + **怎么处理**），并展示引用列表。目标是用人话回答「这个实体现在能不能用、为什么、接下来做什么、具体怎么修」。

**Architecture:** 沿用 config-context-association 的「前端组合、不新增后端端点」模式：页面既有查询（`listCases`/`listEdges`/`listPresence`/`getCaseMenuPlacements`）经纯函数 `features/link-health/lib/references.ts` 聚合为引用列表与健康结论，断点携带 `fix`（config/runtime）分类、`action`（去向 + 文案 key）与 `guide`（人话处理步骤）；`LinkHealthSection` 渲染结论横幅、断点（原因 + 分类徽标 + 行动按钮 + 处理指引）、引用列表。**一期不做 React Flow 全链路图（用户决策，二期评估）。**

**Tech Stack:** React 19、TypeScript、@tanstack/react-query、i18next（zh/en）、Vite、Vitest（node 环境 + 文本合同测试）。

## Global Constraints

- 不新增后端端点；一期全部前端组合（与 config-context-association 设计一致）。
- 每个断点必须携带 `fix: 'config' | 'runtime'` 分类、`action: { to, key }` 行动入口与 `guide` 处理步骤（i18n key）；用户看到断点时必须同时看到「下一步做什么」和「怎么处理」。
- 所有用户可见文案走 i18n zh/en 成对（新命名空间 `linkHealth`），合同测试强制成对。
- 组件样式遵循既有约定，不自定义大 padding 卡片：区块头用 `SectionHead`（`@/features/edges/observation-panel`，标题 + 虚线 + 一行 hint）；断点列表用 `rounded-md border` + `divide-y` 紧凑行（`px-3 py-2.5`）；引用列表用表格化列表（`rounded-md border bg-muted/20` + 表头计数 + 每页 10 条分页：上一页/下一页 + 页码，名称 + 状态标签）；Header 告警几何对齐 `ErrorBanner`（`rounded-md border px-4 py-3 text-sm`，琥珀色表示异常）。
- 新增测试文件必须登记到 `web/admin/vitest.config.ts` 的 `include` 列表。
- 验证命令：`pnpm tsc -b` 与 `pnpm vitest run`（或单文件 `pnpm vitest run <file>`）全绿后才可提交。
- 工作区存在未提交的 case 删除守卫改动（`internal/…`、`web/admin/src/features/cases/detail-panel.tsx` 等）：提交时只 `git add` 本任务文件，不夹带无关改动。
- 提交信息遵循仓库习惯：`feat(link-health): …`。
- 一期不引入 `@xyflow/react` 相关代码；全链路图（React Flow）在 Phase 2 独立 plan 评估。

---

## Task 1: 引用与健康纯函数库（含断点分类与行动）

**Files:**
- Create: `web/admin/src/features/link-health/lib/references.ts`
- Create: `web/admin/src/features/link-health/lib/references.test.ts`
- Modify: `web/admin/vitest.config.ts`（`include` 追加 `src/features/link-health/lib/references.test.ts`）

**Interfaces:**
- Produces: `type HealthState = 'ok' | 'warn' | 'bad'`
- Produces: `interface ReferenceItem { id: string; name: string; state: HealthState; to: string }`
- Produces: `interface HealthBreakpoint { stage: 'entry' | 'workflow' | 'topic' | 'node'; fix: 'config' | 'runtime'; key: string; params?: Record<string, string>; action: { to: string; key: string }; guide: string }`（`guide` 为 i18n key，人话处理步骤）
- Produces: `interface EntityHealth { state: HealthState; breakpoints: HealthBreakpoint[] }`
- Produces: `interface EdgeLike { id: string; name: string; enabled: boolean; subscribe_topics?: string[]; effective_topics?: string[] }`（`EdgeRecord` 满足该形状）
- Produces: `interface CaseLike { id: number; name: string; routing?: RoutingConfig }`（`CaseRecord` 满足该形状）
- Produces: `function caseRoutingTopics(routing: RoutingConfig | undefined): string[]`（无 rules 或无 topic 时回退 `['default']`，去重保序）
- Produces: `function edgeTopics(edge: EdgeLike): string[]`（`effective_topics` 非空优先，否则 `subscribe_topics`，否则 `[]`）
- Produces: `function edgeIsReady(edge: EdgeLike, presence: EdgePresence[]): boolean`（`enabled && edge_online && comfy_running`）
- Produces: `function topicReferences(key: string, input: HealthInput): { cases: ReferenceItem[]; edges: ReferenceItem[]; health: EntityHealth }`
- Produces: `function edgeReferences(id: string, input: HealthInput): { topics: ReferenceItem[]; cases: ReferenceItem[]; health: EntityHealth }`
- Produces: `function caseReferences(caseId: number, input: HealthInput): { menuEntries: ReferenceItem[]; topics: ReferenceItem[]; health: EntityHealth }`
- Produces: `interface HealthInput { cases: CaseLike[]; edges: EdgeLike[]; presence: EdgePresence[]; placements?: MenuPlacement[] }`

断点分类约定：
- `fix: 'config'` = 改配置可修（没绑 Topic、没挂入口、没配路由）——修完立即生效。
- `fix: 'runtime'` = 环境问题（节点停用/离线）——修完需等在线。

- [ ] **Step 1: 写失败测试**（`references.test.ts`），并登记到 `vitest.config.ts`

```ts
import { describe, expect, it } from 'vitest'
import type { EdgePresence, EdgeRecord } from '@/features/task-flow/types'
import type { MenuPlacement } from '@/lib/api/channel-menu'
import type { CaseRecord } from '@/lib/api/types'
import {
  caseReferences,
  caseRoutingTopics,
  edgeReferences,
  edgeIsReady,
  topicReferences,
} from './references'

const caseA = {
  id: 1,
  name: '案例 A',
  routing: { rules: [{ when: { field: 'x', op: 'eq', value: 1 }, topic: 't1' }] },
} as unknown as CaseRecord
const caseDefault = { id: 2, name: '案例 B', routing: undefined } as unknown as CaseRecord
const edgeT1: EdgeRecord = {
  id: 'node-1',
  name: '节点 1',
  enabled: true,
  subscribe_topics: ['t1'],
  effective_topics: ['t1'],
}
const edgeDefault: EdgeRecord = {
  id: 'node-2',
  name: '节点 2',
  enabled: true,
  subscribe_topics: ['default'],
  effective_topics: [],
}
const presence: EdgePresence[] = [{ id: 'node-1', edge_online: true, comfy_running: true }]
const placement: MenuPlacement = {
  channel_id: 'c1',
  channel_name: '消息平台 1',
  item_id: 'm1',
  kind: 'open_case',
  path: [{ id: 'm1', label: '入口' }],
}

describe('caseRoutingTopics', () => {
  it('无 routing 时回退默认 Topic', () => {
    expect(caseRoutingTopics(undefined)).toEqual(['default'])
  })
  it('返回去重后的规则 Topic 列表', () => {
    expect(
      caseRoutingTopics({
        rules: [
          { when: { field: 'a', op: 'eq', value: 1 }, topic: 't1' },
          { when: { field: 'b', op: 'eq', value: 2 }, topic: 't1' },
        ],
      }),
    ).toEqual(['t1'])
  })
})

describe('edgeIsReady', () => {
  it('启用且在线且 Comfy 运行时为就绪', () => {
    expect(edgeIsReady(edgeT1, presence)).toBe(true)
  })
  it('停用节点不算就绪', () => {
    expect(edgeIsReady({ ...edgeT1, enabled: false }, presence)).toBe(false)
  })
})

describe('topicReferences', () => {
  const input = { cases: [caseA, caseDefault], edges: [edgeT1, edgeDefault], presence }
  it('t1 被 Case A 引用且节点在线 → ok', () => {
    const refs = topicReferences('t1', input)
    expect(refs.cases.map((c) => c.id)).toEqual(['1'])
    expect(refs.edges.map((e) => e.id)).toEqual(['node-1'])
    expect(refs.health.state).toBe('ok')
    expect(refs.health.breakpoints).toEqual([])
  })
  it('default 只有离线节点 → warn、断点 runtime、带行动', () => {
    const refs = topicReferences('default', input)
    expect(refs.cases.map((c) => c.id)).toEqual(['2'])
    expect(refs.health.state).toBe('warn')
    const bp = refs.health.breakpoints.find((b) => b.stage === 'node')
    expect(bp?.fix).toBe('runtime')
    expect(bp?.action.to).toBe('/edges')
    expect(bp?.action.key).toBe('linkHealth.actionManageNodes')
    expect(bp?.guide).toBe('linkHealth.guideSubscribersOffline')
  })
  it('未被引用的 Topic → warn、断点 config、带行动', () => {
    const refs = topicReferences('t-ghost', input)
    expect(refs.health.state).toBe('warn')
    const bp = refs.health.breakpoints[0]
    expect(bp?.stage).toBe('workflow')
    expect(bp?.fix).toBe('config')
    expect(bp?.action.key).toBe('linkHealth.actionConfigureRouting')
  })
})

describe('edgeReferences', () => {
  const input = { cases: [caseA, caseDefault], edges: [edgeT1, edgeDefault], presence }
  it('node-1 被 t1 绑定并可到达 Case A → ok', () => {
    const refs = edgeReferences('node-1', input)
    expect(refs.topics.map((t) => t.id)).toEqual(['t1'])
    expect(refs.cases.map((c) => c.id)).toEqual(['1'])
    expect(refs.health.state).toBe('ok')
  })
  it('node-2 离线 → warn、断点 runtime、行动指向节点', () => {
    const refs = edgeReferences('node-2', input)
    expect(refs.health.state).toBe('warn')
    const bp = refs.health.breakpoints.find((b) => b.key === 'linkHealth.edgeNotReady')
    expect(bp?.fix).toBe('runtime')
    expect(bp?.action.to).toBe('/edges/node-2')
  })
})

describe('caseReferences', () => {
  const input = {
    cases: [caseA, caseDefault],
    edges: [edgeT1, edgeDefault],
    presence,
    placements: [placement],
  }
  it('无入口时 warn、断点 config、行动去添加入口', () => {
    const refs = caseReferences(1, { ...input, placements: [] })
    expect(refs.menuEntries).toEqual([])
    expect(refs.health.state).toBe('warn')
    const bp = refs.health.breakpoints.find((b) => b.key === 'linkHealth.noMenuEntry')
    expect(bp?.fix).toBe('config')
    expect(bp?.action.key).toBe('linkHealth.actionAddEntry')
    expect(bp?.guide).toBe('linkHealth.guideNoMenuEntry')
  })
  it('有入口但节点离线 → warn、断点 runtime', () => {
    const refs = caseReferences(2, input)
    expect(refs.menuEntries).toHaveLength(1)
    expect(refs.topics.map((t) => t.id)).toEqual(['default'])
    expect(refs.health.state).toBe('warn')
    expect(refs.health.breakpoints.some((b) => b.stage === 'node' && b.fix === 'runtime')).toBe(true)
  })
})
```

- [ ] **Step 2: 运行确认失败**

Run: `pnpm vitest run src/features/link-health/lib/references.test.ts`
Expected: FAIL（模块不存在 / 断言失败）

- [ ] **Step 3: 实现 `references.ts`**

```ts
import type { EdgePresence, RoutingConfig } from '@/features/task-flow/types'
import type { MenuPlacement } from '@/lib/api/channel-menu'

export type HealthState = 'ok' | 'warn' | 'bad'

export interface ReferenceItem {
  id: string
  name: string
  state: HealthState
  to: string
}

export interface HealthBreakpoint {
  stage: 'entry' | 'workflow' | 'topic' | 'node'
  fix: 'config' | 'runtime'
  key: string
  params?: Record<string, string>
  action: { to: string; key: string }
}

export interface EntityHealth {
  state: HealthState
  breakpoints: HealthBreakpoint[]
}

export interface EdgeLike {
  id: string
  name: string
  enabled: boolean
  subscribe_topics?: string[]
  effective_topics?: string[]
}

export interface CaseLike {
  id: number
  name: string
  routing?: RoutingConfig
}

export interface HealthInput {
  cases: CaseLike[]
  edges: EdgeLike[]
  presence: EdgePresence[]
  placements?: MenuPlacement[]
}

export function caseRoutingTopics(routing: RoutingConfig | undefined): string[] {
  const keys = (routing?.rules ?? [])
    .map((r) => r.topic)
    .filter((x): x is string => Boolean(x))
  return keys.length > 0 ? Array.from(new Set(keys)) : ['default']
}

export function edgeTopics(edge: EdgeLike): string[] {
  const keys =
    edge.effective_topics && edge.effective_topics.length > 0
      ? edge.effective_topics
      : (edge.subscribe_topics ?? [])
  return keys
}

export function edgeIsReady(edge: EdgeLike, presence: EdgePresence[]): boolean {
  if (!edge.enabled) return false
  const row = presence.find((p) => p.id === edge.id)
  return Boolean(row && row.edge_online && row.comfy_running)
}

const ok = (id: string, name: string, to: string): ReferenceItem => ({ id, name, state: 'ok', to })
const warn = (id: string, name: string, to: string): ReferenceItem => ({ id, name, state: 'warn', to })

export function topicReferences(key: string, input: HealthInput) {
  const cases = input.cases.filter((c) => caseRoutingTopics(c.routing).includes(key))
  const edges = input.edges.filter((e) => edgeTopics(e).includes(key))
  const ready = edges.filter((e) => edgeIsReady(e, input.presence))
  const breakpoints: HealthBreakpoint[] = []
  if (cases.length === 0) {
    breakpoints.push({
      stage: 'workflow',
      fix: 'config',
      key: 'linkHealth.noCaseRoutes',
      action: { to: '/cases', key: 'linkHealth.actionConfigureRouting' },
      guide: 'linkHealth.guideNoCaseRoutes',
    })
  }
  if (edges.length === 0) {
    breakpoints.push({
      stage: 'node',
      fix: 'config',
      key: 'linkHealth.noEdgeSubscribers',
      action: { to: '/edges', key: 'linkHealth.actionBindTopic' },
      guide: 'linkHealth.guideNoEdgeSubscribers',
    })
  } else if (ready.length === 0) {
    breakpoints.push({
      stage: 'node',
      fix: 'runtime',
      key: 'linkHealth.subscribersOffline',
      action: { to: '/edges', key: 'linkHealth.actionManageNodes' },
      guide: 'linkHealth.guideSubscribersOffline',
    })
  }
  return {
    cases: cases.map((c) => ok(String(c.id), c.name, `/cases/${c.id}`)),
    edges: edges.map((e) =>
      edgeIsReady(e, input.presence) ? ok(e.id, e.name, `/edges/${e.id}`) : warn(e.id, e.name, `/edges/${e.id}`),
    ),
    health: { state: breakpoints.length > 0 ? 'warn' : 'ok', breakpoints },
  }
}

export function edgeReferences(id: string, input: HealthInput) {
  const edge = input.edges.find((e) => e.id === id)
  if (!edge) {
    return {
      topics: [],
      cases: [],
      health: {
        state: 'bad' as HealthState,
        breakpoints: [{
          stage: 'node',
          fix: 'config',
          key: 'linkHealth.edgeMissing',
          action: { to: '/edges', key: 'linkHealth.actionManageNodes' },
          guide: 'linkHealth.guideEdgeMissing',
        }],
      },
    }
  }
  const topics = edgeTopics(edge)
  const cases = input.cases.filter((c) =>
    caseRoutingTopics(c.routing).some((k) => topics.includes(k)),
  )
  const breakpoints: HealthBreakpoint[] = []
  if (topics.length === 0) {
    breakpoints.push({
      stage: 'topic',
      fix: 'config',
      key: 'linkHealth.noTopicBinding',
      action: { to: `/edges/${id}`, key: 'linkHealth.actionEditNode' },
      guide: 'linkHealth.guideNoTopicBinding',
    })
  }
  if (cases.length === 0) {
    breakpoints.push({
      stage: 'workflow',
      fix: 'config',
      key: 'linkHealth.noCaseReachable',
      action: { to: '/cases', key: 'linkHealth.actionConfigureRouting' },
      guide: 'linkHealth.guideNoCaseReachable',
    })
  }
  if (!edgeIsReady(edge, input.presence)) {
    breakpoints.push({
      stage: 'node',
      fix: 'runtime',
      key: 'linkHealth.edgeNotReady',
      action: { to: `/edges/${id}`, key: 'linkHealth.actionCheckNode' },
      guide: 'linkHealth.guideEdgeNotReady',
    })
  }
  return {
    topics: topics.map((k) => ok(k, k, `/topics/${encodeURIComponent(k)}`)),
    cases: cases.map((c) => ok(String(c.id), c.name, `/cases/${c.id}`)),
    health: { state: breakpoints.length > 0 ? 'warn' : 'ok', breakpoints },
  }
}

export function caseReferences(caseId: number, input: HealthInput) {
  const record = input.cases.find((c) => c.id === caseId)
  if (!record) {
    return {
      menuEntries: [],
      topics: [],
      health: {
        state: 'bad' as HealthState,
        breakpoints: [{
          stage: 'workflow',
          fix: 'config',
          key: 'linkHealth.caseMissing',
          action: { to: '/cases', key: 'linkHealth.actionConfigureRouting' },
          guide: 'linkHealth.guideCaseMissing',
        }],
      },
    }
  }
  const placements = input.placements ?? []
  const topics = caseRoutingTopics(record.routing)
  const breakpoints: HealthBreakpoint[] = []
  if (placements.length === 0) {
    breakpoints.push({
      stage: 'entry',
      fix: 'config',
      key: 'linkHealth.noMenuEntry',
      action: { to: '/channels', key: 'linkHealth.actionAddEntry' },
      guide: 'linkHealth.guideNoMenuEntry',
    })
  }
  const topicItems = topics.map((key) => {
    const edges = input.edges.filter((e) => edgeTopics(e).includes(key))
    const ready = edges.filter((e) => edgeIsReady(e, input.presence))
    if (ready.length === 0) {
      breakpoints.push({
        stage: 'node',
        fix: 'runtime',
        key: 'linkHealth.topicNoReadyNode',
        params: { topic: key },
        action: { to: '/edges', key: 'linkHealth.actionManageNodes' },
        guide: 'linkHealth.guideTopicNoReadyNode',
      })
    }
    return ready.length > 0
      ? ok(key, key, `/topics/${encodeURIComponent(key)}`)
      : warn(key, key, `/topics/${encodeURIComponent(key)}`)
  })
  return {
    menuEntries: placements.map((p) =>
      ok(`${p.channel_id}:${p.item_id}`, p.channel_name ?? p.channel_id, `/channels/${encodeURIComponent(p.channel_id)}`),
    ),
    topics: topicItems,
    health: { state: breakpoints.length > 0 ? 'warn' : 'ok', breakpoints },
  }
}
```

- [ ] **Step 4: 运行确认通过**

Run: `pnpm vitest run src/features/link-health/lib/references.test.ts`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add web/admin/src/features/link-health/lib/references.ts web/admin/src/features/link-health/lib/references.test.ts web/admin/vitest.config.ts
git commit -m "feat(link-health): add reference and health pure functions with breakpoint actions"
```

---

## Task 2: LinkHealthSection + i18n + 合同测试

**Files:**
- Create: `web/admin/src/features/link-health/link-health-alert.tsx`
- Create: `web/admin/src/features/link-health/link-health-section.tsx`
- Create: `web/admin/src/features/link-health/link-health.contract.test.ts`
- Modify: `web/admin/vitest.config.ts`（`include` 追加 `src/features/link-health/link-health.contract.test.ts`）
- Modify: `web/admin/src/lib/i18n/locales/zh.json`、`en.json`（新增 `linkHealth` 命名空间）

**Interfaces:**
- Consumes: Task 1 的 `EntityHealth/ReferenceItem/HealthBreakpoint`
- Produces: `function LinkHealthAlert(props: { name: string; health: EntityHealth; anchorTo: string }): JSX.Element`（Header 告警：仅 `health.state !== 'ok'` 时渲染，含锚点跳转到下方健康区块）
- Produces: `function LinkHealthSection(props: { title: string; health: EntityHealth; upstream: { title: string; items: ReferenceItem[] }; downstream: { title: string; items: ReferenceItem[] } }): JSX.Element`

- [ ] **Step 1: 写失败合同测试**（`link-health.contract.test.ts`），并登记到 `vitest.config.ts`

```ts
import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const here = dirname(fileURLToPath(import.meta.url))

function read(rel: string) {
  return readFileSync(join(here, rel), 'utf8')
}

describe('link health visibility', () => {
  it('LinkHealthAlert 在 Header 展示告警摘要（仅异常时）', () => {
    const alert = read('link-health-alert.tsx')
    expect(alert).toContain("data-testid='link-health-alert'")
    expect(alert).toContain("role='alert'")
    expect(alert).toContain('linkHealth.alertTitle')
    expect(alert).toContain('linkHealth.alertSummary')
    expect(alert).toContain('linkHealth.alertViewDetails')
    expect(alert).toContain('anchorTo')
    expect(alert).toContain("health.state === 'ok'")
  })

  it('LinkHealthSection 提供断点（分类 + 行动 + 指引）与引用列表', () => {
    const section = read('link-health-section.tsx')
    expect(section).toContain("data-testid='link-health-section'")
    expect(section).toContain("data-testid='link-health-breakpoints'")
    expect(section).toContain('SectionHead')
    expect(section).toContain('linkHealth.sectionHint')
    expect(section).toContain('divide-y')
    expect(section).toContain('linkHealth.fixConfig')
    expect(section).toContain('linkHealth.fixRuntime')
    expect(section).toContain('b.action.to')
    expect(section).toContain('b.guide')
    expect(section).toContain('linkHealth.howToHandle')
    expect(section).toContain("data-testid='link-health-reference-list'")
  })

  it('i18n linkHealth 命名空间 zh/en 成对', () => {
    const zh = JSON.parse(read('../../lib/i18n/locales/zh.json'))
    const en = JSON.parse(read('../../lib/i18n/locales/en.json'))
    for (const k of Object.keys(zh.linkHealth)) {
      expect(en.linkHealth[k]).toBeTruthy()
    }
  })
})
```

- [ ] **Step 2: 运行确认失败**

Run: `pnpm vitest run src/features/link-health/link-health.contract.test.ts`
Expected: FAIL（文件不存在 / 断言不满足）

- [ ] **Step 3: 实现 `link-health-alert.tsx`**

```tsx
import { useTranslation } from 'react-i18next'
import { AlertCircle } from 'lucide-react'
import type { EntityHealth } from './lib/references'

export type LinkHealthAlertProps = {
  name: string
  health: EntityHealth
  /** 页面内锚点，指向下方健康区块，例如 '#link-health-section'。 */
  anchorTo: string
}

export function LinkHealthAlert({ name, health, anchorTo }: LinkHealthAlertProps) {
  const { t } = useTranslation()
  if (health.state === 'ok') return null
  const n = health.breakpoints.length
  return (
    <div
      role='alert'
      data-testid='link-health-alert'
      className='flex flex-wrap items-center justify-between gap-3 rounded-md border border-amber-500/40 bg-amber-500/10 px-4 py-3 text-sm text-amber-700 dark:border-amber-400/20 dark:bg-amber-400/10 dark:text-amber-300'
    >
      <div className='flex min-w-0 flex-wrap items-center gap-2'>
        <AlertCircle className='size-4 shrink-0' aria-hidden='true' />
        <span className='font-medium'>{t('linkHealth.alertTitle', { name })}</span>
        <span className='text-amber-700/70 dark:text-amber-300/70'>
          {t('linkHealth.alertSummary', { n })}
        </span>
      </div>
      <a href={anchorTo} className='font-medium underline underline-offset-2'>
        {t('linkHealth.alertViewDetails')} ↓
      </a>
    </div>
  )
}
```

- [ ] **Step 4: 实现 `link-health-section.tsx`**

```tsx
import { useState } from 'react'
import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { CircleCheck } from 'lucide-react'
import { cn } from '@/lib/utils'
import { SectionHead } from '@/features/edges/observation-panel'
import { Button } from '@/components/ui/button'
import type { EntityHealth, ReferenceItem } from './lib/references'

const stateClass: Record<string, string> = {
  ok: 'border-emerald-600/20 bg-emerald-50 text-emerald-700 dark:border-emerald-400/20 dark:bg-emerald-900/30 dark:text-emerald-400',
  warn: 'border-amber-500/40 bg-amber-500/10 text-amber-700 dark:border-amber-400/20 dark:bg-amber-400/10 dark:text-amber-300',
  bad: 'border-red-600/20 bg-red-50 text-red-700 dark:border-red-400/20 dark:bg-red-900/30 dark:text-red-400',
}

const PAGE_SIZE = 10

export type LinkHealthSectionProps = {
  title: string
  health: EntityHealth
  upstream: { title: string; items: ReferenceItem[] }
  downstream: { title: string; items: ReferenceItem[] }
}

export function LinkHealthSection({
  title,
  health,
  upstream,
  downstream,
}: LinkHealthSectionProps) {
  const { t } = useTranslation()
  const blocked = health.breakpoints.length > 0
  return (
    <section
      id='link-health-section'
      data-testid='link-health-section'
      className='space-y-3'
    >
      <SectionHead title={title} hint={t('linkHealth.sectionHint')} />
      {health.state === 'ok' ? (
        <div
          data-testid='link-health-ok'
          className='flex items-center gap-2 rounded-md border border-emerald-600/20 bg-emerald-50 px-3 py-2.5 text-sm text-emerald-700 dark:border-emerald-400/20 dark:bg-emerald-900/30 dark:text-emerald-400'
        >
          <CircleCheck className='size-4 shrink-0' aria-hidden='true' />
          <span className='font-medium'>{t('linkHealth.stateOk')}</span>
          <span className='text-emerald-700/70 dark:text-emerald-400/70'>
            {t('linkHealth.stateOkDetail')}
          </span>
        </div>
      ) : null}
      {blocked ? (
        <div
          className='overflow-hidden rounded-md border'
          data-testid='link-health-breakpoints'
        >
          <ul className='divide-y'>
            {health.breakpoints.map((b, i) => (
              <li key={`${b.stage}-${i}`} className='space-y-1 px-3 py-2.5'>
                <div className='flex flex-wrap items-center gap-2'>
                  <span className='text-sm text-amber-700 dark:text-amber-300'>
                    {t(b.key, b.params)}
                  </span>
                  <span
                    className={cn(
                      'rounded-sm px-1.5 py-0.5 text-[11px]',
                      b.fix === 'config'
                        ? 'bg-sky-500/10 text-sky-700 dark:text-sky-300'
                        : 'bg-rose-500/10 text-rose-700 dark:text-rose-300',
                    )}
                  >
                    {b.fix === 'config'
                      ? t('linkHealth.fixConfig')
                      : t('linkHealth.fixRuntime')}
                  </span>
                  <Link
                    to={b.action.to}
                    className='ml-auto text-sm font-medium text-foreground underline underline-offset-2'
                  >
                    {t(b.action.key)} →
                  </Link>
                </div>
                <p
                  className='text-sm text-muted-foreground'
                  data-testid='link-health-guide'
                >
                  {t('linkHealth.howToHandle')}：{t(b.guide)}
                </p>
              </li>
            ))}
          </ul>
        </div>
      ) : null}
      <div className='grid gap-3 sm:grid-cols-2'>
        <ReferenceList title={upstream.title} items={upstream.items} />
        <ReferenceList title={downstream.title} items={downstream.items} />
      </div>
    </section>
  )
}

function ReferenceList({
  title,
  items,
}: {
  title: string
  items: ReferenceItem[]
}) {
  const { t } = useTranslation()
  const [page, setPage] = useState(1)
  const totalPages = Math.max(1, Math.ceil(items.length / PAGE_SIZE))
  const safePage = Math.min(page, totalPages)
  const start = (safePage - 1) * PAGE_SIZE
  const pageItems = items.slice(start, start + PAGE_SIZE)
  return (
    <div
      className='overflow-hidden rounded-md border bg-muted/20'
      data-testid='link-health-reference-list'
    >
      <div className='flex items-center justify-between border-b px-3 py-2'>
        <p className='text-xs font-medium text-muted-foreground'>{title}</p>
        <span className='text-xs text-muted-foreground'>{items.length}</span>
      </div>
      {items.length === 0 ? (
        <p className='px-3 py-2 text-sm text-muted-foreground'>
          {t('linkHealth.none')}
        </p>
      ) : (
        <>
          <ul className='divide-y'>
            {pageItems.map((item) => (
              <li
                key={`${item.id}:${item.name}`}
                className='flex items-center gap-2 px-3 py-2'
              >
                <Link
                  to={item.to}
                  className={cn(
                    'min-w-0 truncate text-sm font-medium text-foreground hover:underline',
                  )}
                >
                  {item.name}
                </Link>
                <span
                  className={cn(
                    'ml-auto shrink-0 rounded-md border px-1.5 py-0.5 text-[11px]',
                    stateClass[item.state],
                  )}
                >
                  {item.state === 'ok'
                    ? t('linkHealth.stateReady')
                    : item.state === 'warn'
                      ? t('linkHealth.stateWarn')
                      : t('linkHealth.stateBad')}
                </span>
              </li>
            ))}
          </ul>
          {totalPages > 1 ? (
            <div className='flex items-center justify-between border-t px-3 py-2'>
              <span className='text-xs text-muted-foreground'>
                {t('linkHealth.page', { page: safePage, total: totalPages })}
              </span>
              <div className='flex gap-2'>
                <Button
                  type='button'
                  variant='outline'
                  size='sm'
                  disabled={safePage <= 1}
                  onClick={() => setPage(safePage - 1)}
                >
                  {t('linkHealth.prev')}
                </Button>
                <Button
                  type='button'
                  variant='outline'
                  size='sm'
                  disabled={safePage >= totalPages}
                  onClick={() => setPage(safePage + 1)}
                >
                  {t('linkHealth.next')}
                </Button>
              </div>
            </div>
          ) : null}
        </>
      )}
    </div>
  )
}
```

- [ ] **Step 5: 补充 i18n（zh.json / en.json 各加一段）**

zh.json：

```json
"linkHealth": {
  "title": "状态与关联",
  "sectionHint": "检查入口、Topic 与节点的依赖是否就绪，并给出下一步处理指引",
  "stateOk": "链路正常",
  "stateOkDetail": "入口、Topic、节点均就绪",
  "stateReady": "就绪",
  "stateWarn": "异常",
  "stateBad": "缺失",
  "prev": "上一页",
  "next": "下一页",
  "page": "第 {{page}} / {{total}} 页",
  "alertTitle": "{{name}} 当前不可用",
  "alertSummary": "有 {{n}} 处问题需要处理",
  "alertViewDetails": "查看处理指引",
  "breakpoints": "断点",
  "none": "无",
  "howToHandle": "怎么处理",
  "fixConfig": "配置问题",
  "fixRuntime": "运行问题",
  "actionAddEntry": "去添加入口",
  "actionManageNodes": "去管理节点",
  "actionConfigureRouting": "去配置路由",
  "actionBindTopic": "去订阅 Topic",
  "actionEditNode": "去编辑节点",
  "actionCheckNode": "检查节点",
  "guideNoMenuEntry": "在消息平台菜单编辑器中添加「打开工作流」入口，选择该工作流，保存并发布。",
  "guideTopicNoReadyNode": "确认该 Topic 已有计算节点订阅且在线；在计算节点列表启用计算节点，或检查 agent 心跳与 Comfy 服务。",
  "guideNoCaseRoutes": "在该工作流的处理流程中添加路由规则，把条件指向该 Topic，保存后重新检查。",
  "guideNoEdgeSubscribers": "在计算节点详情中勾选该 Topic 的订阅并保存；没有可订阅的计算节点时先新增计算节点。",
  "guideSubscribersOffline": "检查订阅计算节点的 edge-agent 是否在运行、控制面网络是否可达、Comfy 服务是否正常。",
  "guideNoTopicBinding": "在计算节点编辑弹窗中勾选至少一个 Topic 订阅并保存。",
  "guideNoCaseReachable": "在该工作流的处理流程中配置路由规则，让任务经订阅的 Topic 派给该节点。",
  "guideEdgeNotReady": "先确认计算节点已启用；已启用仍离线时，检查 edge-agent 进程、网络与 Comfy 服务，等待心跳恢复。",
  "guideCaseMissing": "该工作流已不存在，去工作流列表重新创建或选择其他实体。",
  "guideEdgeMissing": "该节点已不存在，去节点列表重新创建或选择其他实体。",
  "noMenuEntry": "没有消息平台菜单入口，用户不可达",
  "noCaseRoutes": "没有工作流路由到该 Topic",
  "noTopicBinding": "该计算节点没有订阅任何 Topic",
  "noCaseReachable": "没有工作流把任务派给该节点",
  "edgeNotReady": "计算节点未就绪（停用或离线）",
  "topicNoReadyNode": "Topic {{topic}} 没有在线计算节点",
  "noEdgeSubscribers": "没有计算节点订阅该 Topic",
  "subscribersOffline": "订阅计算节点全部离线",
  "caseMissing": "工作流不存在",
  "edgeMissing": "计算节点不存在",
  "executedWorkflows": "处理的工作流",
  "usedWorkflows": "使用的工作流",
  "subscribedTopics": "订阅 Topic",
  "boundNodes": "绑定计算节点",
  "relatedEntries": "关联入口",
  "routeTopics": "路由 Topic"
}
```

en.json：

```json
"linkHealth": {
  "title": "Status & relations",
  "sectionHint": "Checks whether entries, topics, and nodes are ready, with next-step guidance",
  "stateOk": "All ready",
  "stateOkDetail": "Entries, topics, and nodes are ready",
  "stateReady": "Ready",
  "stateWarn": "Needs attention",
  "stateBad": "Missing",
  "prev": "Previous",
  "next": "Next",
  "page": "Page {{page}} of {{total}}",
  "alertTitle": "{{name}} is currently unavailable",
  "alertSummary": "{{n}} issue(s) need attention",
  "alertViewDetails": "View fix guide",
  "breakpoints": "Breakpoints",
  "none": "None",
  "howToHandle": "How to fix",
  "fixConfig": "Configuration",
  "fixRuntime": "Runtime",
  "actionAddEntry": "Add entry",
  "actionManageNodes": "Manage nodes",
  "actionConfigureRouting": "Configure routing",
  "actionBindTopic": "Subscribe topic",
  "actionEditNode": "Edit node",
  "actionCheckNode": "Check node",
  "guideNoMenuEntry": "In the channel menu editor, add an open-workflow entry that selects this workflow, save, and publish.",
  "guideTopicNoReadyNode": "Make sure this topic has a subscribing compute node that is online; enable the node in the compute node list, or check agent heartbeat and the Comfy service.",
  "guideNoCaseRoutes": "Add a routing rule in this workflow's processing flow that points a condition to this topic, save, then re-check.",
  "guideNoEdgeSubscribers": "Select this topic in the compute node's subscription settings and save; create a compute node first if none are available.",
  "guideSubscribersOffline": "Check that subscribing compute nodes' edge-agent is running, the control plane is reachable, and the Comfy service is healthy.",
  "guideNoTopicBinding": "Select at least one topic subscription in the compute node edit dialog and save.",
  "guideNoCaseReachable": "Configure routing rules in this workflow's processing flow so tasks are dispatched to this node through subscribed topics.",
  "guideEdgeNotReady": "First confirm the compute node is enabled; if it is still offline, check the edge-agent process, network, and Comfy service, then wait for heartbeat to recover.",
  "guideCaseMissing": "This workflow no longer exists — recreate it in the workflow list or select another entity.",
  "guideEdgeMissing": "This node no longer exists — recreate it in the node list or select another entity.",
  "noMenuEntry": "No menu entry — users cannot reach this case",
  "noCaseRoutes": "No workflow routes to this topic",
  "noTopicBinding": "This compute node subscribes to no topics",
  "noCaseReachable": "No workflow routes tasks to this node",
  "edgeNotReady": "Compute node is not ready (disabled or offline)",
  "topicNoReadyNode": "Topic {{topic}} has no online compute node",
  "noEdgeSubscribers": "No compute node subscribes to this topic",
  "subscribersOffline": "All subscribing compute nodes are offline",
  "caseMissing": "Workflow not found",
  "edgeMissing": "Compute node not found",
  "executedWorkflows": "Workflows it processes",
  "usedWorkflows": "Workflows that use it",
  "subscribedTopics": "Subscribed topics",
  "boundNodes": "Bound compute nodes",
  "relatedEntries": "Related entries",
  "routeTopics": "Routed topics"
}
```

- [ ] **Step 6: 运行确认通过**

Run: `pnpm vitest run src/features/link-health/link-health.contract.test.ts`
Expected: PASS

- [ ] **Step 7: 提交**

```bash
git add web/admin/src/features/link-health web/admin/vitest.config.ts web/admin/src/lib/i18n/locales/zh.json web/admin/src/lib/i18n/locales/en.json
git commit -m "feat(link-health): add health section with breakpoint actions"
```

---

## Task 3: Edge 详情接入绑定提示与行动

**Files:**
- Modify: `web/admin/src/features/edges/detail-panel.tsx`（在 `presenceQuery` 之后加 `casesQuery`，在观测面板之后渲染 `LinkHealthSection`）
- Modify: `web/admin/src/features/link-health/link-health.contract.test.ts`（追加 Edge 详情断言）

**Interfaces:**
- Consumes: Task 2 的 `LinkHealthSection`；Task 1 的 `edgeReferences`、`EdgeLike`
- Consumes: 既有 `getEdge`（返回 `ComfyEdge`，含 `subscribe_topics/effective_topics`）、`listPresence`、`listCases`

- [ ] **Step 1: 追加失败合同测试**

在 `link-health.contract.test.ts` 追加：

```ts
  it('Edge 详情接入绑定提示与行动', () => {
    const panel = read('../edges/detail-panel.tsx')
    expect(panel).toContain('LinkHealthSection')
    expect(panel).toContain('LinkHealthAlert')
    expect(panel).toContain('edgeReferences')
    expect(panel).toContain('listCases')
  })
```

Run: `pnpm vitest run src/features/link-health/link-health.contract.test.ts`
Expected: FAIL

- [ ] **Step 2: 修改 `detail-panel.tsx`**

新增 imports：

```tsx
import { useMemo } from 'react'
import { listCases } from '@/lib/api/cases'
import { LinkHealthAlert } from '@/features/link-health/link-health-alert'
import { LinkHealthSection } from '@/features/link-health/link-health-section'
import { edgeReferences } from '@/features/link-health/lib/references'
```

在 `presenceQuery` 声明之后新增查询与派生数据（`edge` 为该文件既有 `detailQuery.data`）：

```tsx
  const casesQuery = useQuery({
    queryKey: queryKeys.cases.all,
    queryFn: () => listCases(),
  })

  const edgeLike = {
    id: edge.id,
    name: edge.name,
    enabled: edge.enabled,
    subscribe_topics: edge.subscribe_topics ?? [],
    effective_topics: edge.effective_topics ?? [],
  }
  const linkInput = {
    cases: casesQuery.data ?? [],
    edges: [edgeLike],
    presence: presenceQuery.data ?? [],
  }
  const edgeRefs = useMemo(() => edgeReferences(id, linkInput), [id, linkInput])
```

> 注意：`edge` 在 `detailQuery.data` 判空之后才可用，因此上述派生放现有 `const edge = detailQuery.data` 之后。

在**详情头部**（标题 `h2` 与状态标签之后、规格区之前）渲染 Header 告警：

```tsx
      <LinkHealthAlert
        name={edge.name}
        health={edgeRefs.health}
        anchorTo='#link-health-section'
      />
```

在 `</section>` 收尾前（观测面板之后）渲染健康检查明细：

```tsx
      <LinkHealthSection
        title={t('linkHealth.title')}
        health={edgeRefs.health}
        upstream={{ title: t('linkHealth.executedWorkflows'), items: edgeRefs.cases }}
        downstream={{ title: t('linkHealth.subscribedTopics'), items: edgeRefs.topics }}
      />
```

- [ ] **Step 3: 运行确认通过**

Run: `pnpm vitest run src/features/link-health/link-health.contract.test.ts && pnpm tsc -b`
Expected: PASS

- [ ] **Step 4: 提交**

```bash
git add web/admin/src/features/edges/detail-panel.tsx web/admin/src/features/link-health/link-health.contract.test.ts
git commit -m "feat(link-health): edge detail binding hint and next actions"
```

---

## Task 4: Topic 详情接入引用列表与行动

**Files:**
- Modify: `web/admin/src/features/topics/topic-detail-panel.tsx`（新增 `presenceQuery`，在既有 `ConfigChain` 之后渲染 `LinkHealthSection`）
- Modify: `web/admin/src/features/link-health/link-health.contract.test.ts`（追加 Topic 详情断言）

**Interfaces:**
- Consumes: Task 2 的 `LinkHealthSection`；Task 1 的 `topicReferences`、`EdgeLike`
- Consumes: 既有 `listCases`（`casesQuery`）、`listEdges`（`edgesQuery`）、`listPresence`（新增）

- [ ] **Step 1: 追加失败合同测试**

```ts
  it('Topic 详情接入引用列表与行动', () => {
    const panel = read('../topics/topic-detail-panel.tsx')
    expect(panel).toContain('LinkHealthSection')
    expect(panel).toContain('LinkHealthAlert')
    expect(panel).toContain('topicReferences')
    expect(panel).toContain('listPresence')
  })
```

Run: `pnpm vitest run src/features/link-health/link-health.contract.test.ts`
Expected: FAIL

- [ ] **Step 2: 修改 `topic-detail-panel.tsx`**

新增 imports：

```tsx
import { useMemo } from 'react'
import { listPresence } from '@/lib/api/edges'
import { LinkHealthAlert } from '@/features/link-health/link-health-alert'
import { LinkHealthSection } from '@/features/link-health/link-health-section'
import { topicReferences } from '@/features/link-health/lib/references'
```

新增查询与派生（`topicKey` 为既有 props；`edgesQuery`/`casesQuery` 为既有查询）：

```tsx
  const presenceQuery = useQuery({
    queryKey: queryKeys.edges.presence,
    queryFn: listPresence,
  })

  const linkInput = {
    cases: casesQuery.data ?? [],
    edges: edgesQuery.data ?? [],
    presence: presenceQuery.data ?? [],
  }
  const topicRefs = useMemo(
    () => topicReferences(topicKey, linkInput),
    [topicKey, linkInput],
  )
```

在**详情头部**（标题与状态标签之后）渲染 Header 告警：

```tsx
      <LinkHealthAlert
        name={name || topicKey}
        health={topicRefs.health}
        anchorTo='#link-health-section'
      />
```

在 `TaskFlowEditor` 之后渲染健康检查明细（工作流页 `ConfigChain` 已移除，由本区块承担）：

```tsx
      <LinkHealthSection
        title={t('linkHealth.title')}
        health={topicRefs.health}
        upstream={{ title: t('linkHealth.usedWorkflows'), items: topicRefs.cases }}
        downstream={{ title: t('linkHealth.boundNodes'), items: topicRefs.edges }}
      />
```

- [ ] **Step 3: 运行确认通过**

Run: `pnpm vitest run src/features/link-health/link-health.contract.test.ts && pnpm tsc -b`
Expected: PASS

- [ ] **Step 4: 提交**

```bash
git add web/admin/src/features/topics/topic-detail-panel.tsx web/admin/src/features/link-health/link-health.contract.test.ts
git commit -m "feat(link-health): topic detail references and next actions"
```

---

## Task 5: Case 详情接入可达性提示与行动

**Files:**
- Modify: `web/admin/src/features/config-context/case-context-section.tsx`（复用既有 `topicsQuery/edgesQuery/presenceQuery/placementsQuery`；`ConfigChain` 已移除，在 `TaskFlowEditor` 之后渲染 `LinkHealthSection`）

> 该文件内的 `TaskFlowEditor` 同步改为：默认 `preview`（只读画布、隐藏顶栏与 Topic 池），右上角「编辑处理流程」按钮打开全屏 `Dialog` 编辑（含保存按钮）。
- Modify: `web/admin/src/features/link-health/link-health.contract.test.ts`（追加 Case 详情断言）

**Interfaces:**
- Consumes: Task 2 的 `LinkHealthSection`；Task 1 的 `caseReferences`
- Consumes: 既有 `CaseContextSection` 内的 `record.id`（number）、`placementsQuery.data`（`MenuPlacement[]`）

- [ ] **Step 1: 追加失败合同测试**

```ts
  it('Case 详情接入可达性提示与行动', () => {
    const section = read('../config-context/case-context-section.tsx')
    expect(section).toContain('LinkHealthSection')
    expect(section).toContain('LinkHealthAlert')
    expect(section).toContain('caseReferences')
    expect(section).toContain('linkHealth.title')
  })
```

Run: `pnpm vitest run src/features/link-health/link-health.contract.test.ts`
Expected: FAIL

- [ ] **Step 2: 修改 `case-context-section.tsx`**

> 文件位于 `features/config-context/`（Case 详情「处理流程」区块），不是 `features/cases/sections/`。

新增 imports：

```tsx
import { useMemo } from 'react'
import { LinkHealthAlert } from '@/features/link-health/link-health-alert'
import { LinkHealthSection } from '@/features/link-health/link-health-section'
import { caseReferences } from '@/features/link-health/lib/references'
```

在既有派生（`chain` 之前或之后）新增：

```tsx
  const linkInput = {
    cases: [record],
    edges,
    presence: presenceQuery.data ?? [],
    placements: placementsQuery.data ?? [],
  }
  const caseRefs = useMemo(
    () => caseReferences(record.id, linkInput),
    [record.id, linkInput],
  )
```

> `edges` 为该文件既有由 `edgesQuery.data` 映射成的 `EdgeRecord[]` 变量；`presenceQuery.data` 原样可用。

在 **CaseContextSection 顶部**（页面头部下方第一屏）渲染 Header 告警：

```tsx
      <LinkHealthAlert
        name={record.name}
        health={caseRefs.health}
        anchorTo='#link-health-section'
      />
```

> Case 的页面标题在 `detail-panel.tsx`，而健康数据在本组件内；为避免重复查询，Alert 放在本组件顶部（仍属第一屏）。如需严格置顶到 `detail-panel`，可抽共享 hook（Phase 2 优化）。

在 `TaskFlowEditor` 之后渲染健康检查明细（`ConfigChain` 已移除）：

```tsx
      <LinkHealthSection
        title={t('linkHealth.title')}
        health={caseRefs.health}
        upstream={{ title: t('linkHealth.relatedEntries'), items: caseRefs.menuEntries }}
        downstream={{ title: t('linkHealth.routeTopics'), items: caseRefs.topics }}
      />
```

- [ ] **Step 3: 运行确认通过**

Run: `pnpm vitest run src/features/link-health/link-health.contract.test.ts && pnpm tsc -b`
Expected: PASS

- [ ] **Step 4: 全量验证**

Run: `pnpm tsc -b && pnpm vitest run`
Expected: 全绿（含既有 config-context 合同测试）

- [ ] **Step 5: 提交（只 add 本任务文件，不夹带 case 删除守卫的未提交改动）**

```bash
git add web/admin/src/features/config-context/case-context-section.tsx web/admin/src/features/link-health/link-health.contract.test.ts
git commit -m "feat(link-health): case detail reachability and next actions"
```

---

## Phase 2 / Phase 3（独立 plan，本 plan 不含）

按 writing-plans 的 Scope Check，以下内容拆为独立 plan，开工前再按本格式补齐 TDD 任务：

**Phase 2 — `2026-08-24-link-health-entry-workflow.md`（建议）**
- **React Flow 全链路图（暂缓，用户决策）**：`buildLinkGraph`/`LinkGraph` 只读链路图（入口→Case→Topic→节点，点击节点切换视角）；届时按 react-flow skill 用确定性分层坐标 + `fitView`/`Controls`，不引入 task-flow 编辑器布局 hook。
- **内联一键修复**：复用既有编辑表单的 mutation（如节点详情内直接启用/停用、Topic 订阅保存、Case 路由保存），在断点旁直接提供「一键修复」按钮，减少跳转；先做高频三个（启用节点 / 绑定 Topic / 配置路由）。
- 入口视角：入口发布/生效状态 + 入口 → Case 引用列表（复用 `getMenu`/`getCaseMenuPlacements`）。
- 处理流程升级：`workflow-graph-preview.tsx` 可视化 + 流程与 Topic 绑定一致性检查。
- 变更影响分析：停用 Edge / 解绑 Topic / 下线入口前，用 `edgeReferences/topicReferences/caseReferences` 预览影响面。

**Phase 3 — `2026-08-24-link-health-ops.md`（建议）**
- 孤儿实体统计面板（无入口 Case / 无引用 Topic / 无绑定节点 / 无引用入口）。
- 定时巡检与告警（复用 `internal/platform/notify`）。
- 健康状态沿链路传播汇总、流程版本管理、全局搜索。

---

## Self-Review

- **Spec 覆盖**：设计文档缺口 1-3 由 Task 1（纯函数 + 断点分类/行动）、Task 2（区块）、Task 3-5（三页接入）覆盖；React Flow 全链路图按用户决策移至 Phase 2，不在本 plan 冒充占位。
- **占位扫描**：无 TBD/TODO；页面接入处引用既有变量并给出替代说明（`edges`），未留“后续再补”。
- **类型一致性**：`HealthBreakpoint` 的 `fix/action/guide` 在 Task 1 定义并被 Task 2 渲染使用；`edgeReferences/caseReferences/topicReferences` 返回结构与 `LinkHealthSection` props 一致；三页调用与 `LinkHealthSection` 签名一致；`guide*` i18n key 与 zh/en 两个 locale 成对。
