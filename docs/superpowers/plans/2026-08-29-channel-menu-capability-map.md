# 渠道详情 · 菜单能力地图 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans（本项目默认 build_mode）配合 superpowers:test-driven-development 逐任务实施。步骤用 `- [ ]` 勾选跟踪。

**Goal:** 把渠道详情菜单主视图换成只读能力地图（主键盘 → 点下去 → 未挂上），「改菜单」打开既有 `MenuEditorModal`。

**Architecture:** 纯函数放进 `menu-flow.ts`（人话结果、断引用、可达卡片、未挂上、工作流入口计数、路径 trail）。新组件 `MenuCapabilityMap` 只负责展示与点选。`MenuCardEditor` 变成宿主：拉数 + 地图 + 弹层。详情页去掉区块 hint 和外层卡片套卡片。

**Tech Stack:** React/TS · TanStack Query · shadcn `Button` · vitest（node 环境：纯函数单测 + 源码契约）· i18n zh/en · Pixoma 语义令牌。

---
change: channel-menu-editor-refactor
design-doc: docs/superpowers/specs/2026-08-29-channel-menu-capability-map-design.md
prototype: docs/superpowers/prototypes/2026-08-29-channel-menu-capability-map.html
---

## Global Constraints

- 只改 `web/admin/src/` 前端；不改菜单/卡片 API、不改 bot 运行时。
- 文案成对写入 `zh.json` / `en.json`；中文走 Pixoma Voice（不用请/您/感叹号/说明性 hint）。
- 取色只用语义令牌与 `color-mix`；表面无投影；主 CTA 只有「改菜单」。
- 触屏键 `min-height: 44px`；`< 920px` 上下叠、不横滑。
- 不画系统「‹ 返回」；主视图不放 `PhoneSimulation`、不放大纲/卡片库 tab。
- vitest `environment: 'node'`，新测试必须加入 `web/admin/vitest.config.ts` 的 `include`。
- 不提交 git（本项目 guard；用户未要求提交）。每任务末跳过 commit。

---

### Task 1: 地图纯函数（人话 / 断引用 / 可达 / 未挂上 / 计数）

**Files:**
- Modify: `web/admin/src/features/menu/lib/menu-flow.ts`
- Modify: `web/admin/src/features/menu/lib/menu-flow.test.ts`
- Consumes: `Action` / `Card` / `Menu`（`@/lib/api/channel-menu`）；`WorkflowRef`（`../node-view`）
- Produces:
  ```ts
  export type MapOutcomeKey =
    | 'mapOpenCard'
    | 'mapStartWorkflow'
    | 'mapSendText'
    | 'mapSendMedia'
    | 'mapOpenUrl'
    | 'mapCopyText'
    | 'mapCardMissing'
    | 'mapWorkflowMissing'

  export type MapOutcome = { key: MapOutcomeKey; name?: string }

  export function actionOutcomeLabel(
    action: Action,
    cards: Card[],
    workflows: WorkflowRef[]
  ): MapOutcome

  export function actionIsBroken(
    action: Action,
    cards: Card[],
    workflows: WorkflowRef[]
  ): boolean

  export function reachableCardIds(menu: Menu, cards: Card[]): Set<string>
  export function orphanCards(menu: Menu, cards: Card[]): Card[]
  export function workflowEntryCount(menu: Menu, cards: Card[]): number
  export function mediaKindLabelKey(
    kind: string
  ): 'mapThumbImage' | 'mapThumbVideo' | 'mapThumbAnimation'
  ```

- [ ] **Step 1.1: 写失败测试**

在 `menu-flow.test.ts` 追加（保留现有用例）。fixtures：

```ts
import type { WorkflowRef } from '../node-view'
import {
  actionIsBroken,
  actionOutcomeLabel,
  mediaKindLabelKey,
  orphanCards,
  reachableCardIds,
  workflowEntryCount,
} from './menu-flow'

const workflows: WorkflowRef[] = [
  { id: 1, name: '写实人像' },
  { id: 2, name: '动漫风' },
]

const style: Card = {
  id: 'style',
  name: '风格选择',
  media: [],
  text: '选',
  buttons: [
    { id: 'b1', label: '写实', action: { type: 'open_workflow', workflow_id: '1' } },
    { id: 'b2', label: '说明', action: { type: 'open_card', card_id: 'guide' } },
  ],
}
const guide: Card = {
  id: 'guide',
  name: '风格说明',
  media: [],
  text: '说明',
  buttons: [],
}
const welcome: Card = {
  id: 'welcome',
  name: '欢迎卡',
  media: [],
  text: '未挂',
  buttons: [{ id: 'w1', label: '去生成', action: { type: 'open_card', card_id: 'style' } }],
}
const cycleA: Card = {
  id: 'ca',
  name: 'A',
  media: [],
  text: 'A',
  buttons: [{ id: 'x', label: '去B', action: { type: 'open_card', card_id: 'cb' } }],
}
const cycleB: Card = {
  id: 'cb',
  name: 'B',
  media: [],
  text: 'B',
  buttons: [{ id: 'y', label: '去A', action: { type: 'open_card', card_id: 'ca' } }],
}

it('actionOutcomeLabel 六种动作与断引用', () => {
  expect(actionOutcomeLabel({ type: 'open_card', card_id: 'style' }, [style], workflows)).toEqual({
    key: 'mapOpenCard',
    name: '风格选择',
  })
  expect(actionOutcomeLabel({ type: 'open_card', card_id: 'gone' }, [style], workflows)).toEqual({
    key: 'mapCardMissing',
  })
  expect(actionOutcomeLabel({ type: 'open_workflow', workflow_id: '1' }, [], workflows)).toEqual({
    key: 'mapStartWorkflow',
    name: '写实人像',
  })
  expect(actionOutcomeLabel({ type: 'open_workflow', workflow_id: 'gone' }, [], workflows)).toEqual({
    key: 'mapWorkflowMissing',
  })
  expect(actionOutcomeLabel({ type: 'send_text', text: 'hi' }, [], [])).toEqual({ key: 'mapSendText' })
  expect(actionOutcomeLabel({ type: 'send_media', media: [] }, [], [])).toEqual({ key: 'mapSendMedia' })
  expect(actionOutcomeLabel({ type: 'open_url', url: 'https://x' }, [], [])).toEqual({ key: 'mapOpenUrl' })
  expect(actionOutcomeLabel({ type: 'copy_text', text: 'CODE' }, [], [])).toEqual({ key: 'mapCopyText' })
})

it('actionIsBroken 只在卡片/工作流缺失时为 true', () => {
  expect(actionIsBroken({ type: 'open_card', card_id: 'gone' }, [style], workflows)).toBe(true)
  expect(actionIsBroken({ type: 'open_workflow', workflow_id: 'gone' }, [], workflows)).toBe(true)
  expect(actionIsBroken({ type: 'open_card', card_id: 'style' }, [style], workflows)).toBe(false)
  expect(actionIsBroken({ type: 'send_text', text: 'hi' }, [], [])).toBe(false)
})

it('reachableCardIds 从主键盘沿 open_card 走，循环不死', () => {
  const menu: Menu = {
    id: 'm',
    name: '主',
    columns: 2,
    items: [{ id: 'i1', label: '图', action: { type: 'open_card', card_id: 'style' } }],
  }
  const ids = reachableCardIds(menu, [style, guide, welcome])
  expect([...ids].sort()).toEqual(['guide', 'style'])
  const cyc: Menu = {
    id: 'm',
    name: '主',
    columns: 2,
    items: [{ id: 'i', label: 'A', action: { type: 'open_card', card_id: 'ca' } }],
  }
  expect(reachableCardIds(cyc, [cycleA, cycleB]).size).toBe(2)
})

it('orphanCards 是键盘走不到的卡片', () => {
  const menu: Menu = {
    id: 'm',
    name: '主',
    columns: 2,
    items: [{ id: 'i1', label: '图', action: { type: 'open_card', card_id: 'style' } }],
  }
  expect(orphanCards(menu, [style, guide, welcome]).map((c) => c.id)).toEqual(['welcome'])
})

it('workflowEntryCount 去重且计入断引用与未挂上卡片上的入口', () => {
  const menu: Menu = {
    id: 'm',
    name: '主',
    columns: 2,
    items: [
      { id: 'i1', label: '直出', action: { type: 'open_workflow', workflow_id: '1' } },
      { id: 'i2', label: '会员', action: { type: 'open_workflow', workflow_id: 'gone' } },
    ],
  }
  // style 上还有 workflow 1（与键盘重复）和 guide 无工作流；welcome 无工作流
  expect(workflowEntryCount(menu, [style, welcome])).toBe(2)
})

it('mediaKindLabelKey', () => {
  expect(mediaKindLabelKey('image')).toBe('mapThumbImage')
  expect(mediaKindLabelKey('video')).toBe('mapThumbVideo')
  expect(mediaKindLabelKey('animation')).toBe('mapThumbAnimation')
})
```

- [ ] **Step 1.2: 跑测试确认失败**

Run: `cd web/admin && pnpm exec vitest run src/features/menu/lib/menu-flow.test.ts`

Expected: FAIL（`actionOutcomeLabel` is not a function / not exported）

- [ ] **Step 1.3: 最小实现**

在 `menu-flow.ts` 增加：

```ts
import type { WorkflowRef } from '../node-view'

export type MapOutcomeKey =
  | 'mapOpenCard'
  | 'mapStartWorkflow'
  | 'mapSendText'
  | 'mapSendMedia'
  | 'mapOpenUrl'
  | 'mapCopyText'
  | 'mapCardMissing'
  | 'mapWorkflowMissing'

export type MapOutcome = { key: MapOutcomeKey; name?: string }

function cardById(cards: Card[], id: string | undefined): Card | undefined {
  return id ? cards.find((c) => c.id === id) : undefined
}

function workflowById(
  workflows: WorkflowRef[],
  id: string | undefined
): WorkflowRef | undefined {
  return id ? workflows.find((w) => String(w.id) === id) : undefined
}

export function actionOutcomeLabel(
  action: Action,
  cards: Card[],
  workflows: WorkflowRef[]
): MapOutcome {
  switch (action.type) {
    case 'open_card': {
      const c = cardById(cards, action.card_id)
      return c ? { key: 'mapOpenCard', name: c.name } : { key: 'mapCardMissing' }
    }
    case 'open_workflow': {
      const w = workflowById(workflows, action.workflow_id)
      return w
        ? { key: 'mapStartWorkflow', name: w.name }
        : { key: 'mapWorkflowMissing' }
    }
    case 'send_text':
      return { key: 'mapSendText' }
    case 'send_media':
      return { key: 'mapSendMedia' }
    case 'open_url':
      return { key: 'mapOpenUrl' }
    case 'copy_text':
      return { key: 'mapCopyText' }
    default:
      return { key: 'mapCardMissing' }
  }
}

export function actionIsBroken(
  action: Action,
  cards: Card[],
  workflows: WorkflowRef[]
): boolean {
  const key = actionOutcomeLabel(action, cards, workflows).key
  return key === 'mapCardMissing' || key === 'mapWorkflowMissing'
}

export function reachableCardIds(menu: Menu, cards: Card[]): Set<string> {
  const byId = new Map(cards.map((c) => [c.id, c]))
  const seen = new Set<string>()
  const walk = (action: Action) => {
    if (action.type !== 'open_card' || !action.card_id) return
    if (seen.has(action.card_id)) return
    const card = byId.get(action.card_id)
    if (!card) return
    seen.add(card.id)
    for (const b of card.buttons) walk(b.action)
  }
  for (const it of menu.items) walk(it.action)
  return seen
}

export function orphanCards(menu: Menu, cards: Card[]): Card[] {
  const reached = reachableCardIds(menu, cards)
  return cards.filter((c) => !reached.has(c.id))
}

function collectWorkflowIds(action: Action, cards: Card[], seen: Set<string>, walking: Set<string>) {
  if (action.type === 'open_workflow' && action.workflow_id) {
    seen.add(action.workflow_id)
  }
  if (action.type === 'open_card' && action.card_id && !walking.has(action.card_id)) {
    walking.add(action.card_id)
    const card = cards.find((c) => c.id === action.card_id)
    if (card) for (const b of card.buttons) collectWorkflowIds(b.action, cards, seen, walking)
  }
}

export function workflowEntryCount(menu: Menu, cards: Card[]): number {
  const seen = new Set<string>()
  const walking = new Set<string>()
  for (const it of menu.items) collectWorkflowIds(it.action, cards, seen, walking)
  for (const card of cards) {
    for (const b of card.buttons) collectWorkflowIds(b.action, cards, seen, walking)
  }
  return seen.size
}

export function mediaKindLabelKey(
  kind: string
): 'mapThumbImage' | 'mapThumbVideo' | 'mapThumbAnimation' {
  if (kind === 'video') return 'mapThumbVideo'
  if (kind === 'animation') return 'mapThumbAnimation'
  return 'mapThumbImage'
}
```

注意：`workflowEntryCount` 必须扫 **全部卡片按钮**（含未挂上），断引用 id 也计入 `seen`。

- [ ] **Step 1.4: 跑测试确认通过**

Run: `cd web/admin && pnpm exec vitest run src/features/menu/lib/menu-flow.test.ts`

Expected: PASS

---

### Task 2: 路径 trail 纯函数

**Files:**
- Modify: `web/admin/src/features/menu/lib/menu-flow.ts`
- Modify: `web/admin/src/features/menu/lib/menu-flow.test.ts`
- Consumes: Task 1 的类型；`MenuItem` / `CardButton`
- Produces:
  ```ts
  export type MapTrailStep = {
    kind: 'item' | 'orphan' | 'btn'
    id: string
    label: string
    action: Action
  }

  export function trailFromItem(item: MenuItem): MapTrailStep[]
  export function trailFromOrphan(card: Card): MapTrailStep[]
  export function pushTrailButton(
    trail: MapTrailStep[],
    button: CardButton
  ): MapTrailStep[]
  export function sliceTrail(
    trail: MapTrailStep[],
    indexInclusive: number
  ): MapTrailStep[]
  ```

- [ ] **Step 2.1: 写失败测试**

```ts
it('trailFromItem / pushTrailButton / sliceTrail', () => {
  const item: MenuItem = {
    id: 'i1',
    label: '图片生成',
    action: { type: 'open_card', card_id: 'style' },
  }
  const t0 = trailFromItem(item)
  expect(t0).toEqual([
    { kind: 'item', id: 'i1', label: '图片生成', action: item.action },
  ])
  const btn: CardButton = {
    id: 'b1',
    label: '写实',
    action: { type: 'open_workflow', workflow_id: '1' },
  }
  const t1 = pushTrailButton(t0, btn)
  expect(t1).toHaveLength(2)
  expect(t1[1]).toEqual({
    kind: 'btn',
    id: 'b1',
    label: '写实',
    action: btn.action,
  })
  expect(sliceTrail(t1, 0)).toEqual(t0)
})

it('trailFromOrphan 起点是 orphan + open_card 自己', () => {
  const t = trailFromOrphan(welcome)
  expect(t).toEqual([
    {
      kind: 'orphan',
      id: 'welcome',
      label: '欢迎卡',
      action: { type: 'open_card', card_id: 'welcome' },
    },
  ])
})
```

- [ ] **Step 2.2: 跑测试确认失败**

Run: `cd web/admin && pnpm exec vitest run src/features/menu/lib/menu-flow.test.ts`

Expected: FAIL（`trailFromItem` is not a function）

- [ ] **Step 2.3: 最小实现**

```ts
export type MapTrailStep = {
  kind: 'item' | 'orphan' | 'btn'
  id: string
  label: string
  action: Action
}

export function trailFromItem(item: MenuItem): MapTrailStep[] {
  return [{ kind: 'item', id: item.id, label: item.label, action: item.action }]
}

export function trailFromOrphan(card: Card): MapTrailStep[] {
  return [
    {
      kind: 'orphan',
      id: card.id,
      label: card.name,
      action: { type: 'open_card', card_id: card.id },
    },
  ]
}

export function pushTrailButton(
  trail: MapTrailStep[],
  button: CardButton
): MapTrailStep[] {
  return trail.concat([
    { kind: 'btn', id: button.id, label: button.label, action: button.action },
  ])
}

export function sliceTrail(
  trail: MapTrailStep[],
  indexInclusive: number
): MapTrailStep[] {
  return trail.slice(0, indexInclusive + 1)
}
```

- [ ] **Step 2.4: 跑测试确认通过**

Run: `cd web/admin && pnpm exec vitest run src/features/menu/lib/menu-flow.test.ts`

Expected: PASS

---

### Task 3: i18n 文案

**Files:**
- Modify: `web/admin/src/lib/i18n/locales/zh.json`
- Modify: `web/admin/src/lib/i18n/locales/en.json`
- Modify: `web/admin/src/features/menu/menu-editor.contract.test.ts`（`V2_KEYS` 追加地图键；`editMenu` 已在列表里）
- Modify: `web/admin/src/features/channels/channel-layout.contract.test.ts`

**Produces:** 下列键必须中英成对。中文用下面原文（不要改成更「完整」的说明句）。

| 键 | zh | en |
|---|---|---|
| `menu.mapPath` | 点下去 | Next |
| `menu.mapEmpty` | 还没有键 | No keys |
| `menu.mapOrphans` | 未挂上 · {{n}} | Unlinked · {{n}} |
| `menu.mapCountKeys` | {{n}} 键 | {{n}} keys |
| `menu.mapCountCards` | {{n}} 卡片 | {{n}} cards |
| `menu.mapCountWorkflows` | {{n}} 工作流入口 | {{n}} workflow entries |
| `menu.mapOpenCard` | 打开「{{name}}」 | Open "{{name}}" |
| `menu.mapStartWorkflow` | 开始「{{name}}」 | Start "{{name}}" |
| `menu.mapSendText` | 发这段话 | Send text |
| `menu.mapSendMedia` | 发图 | Send media |
| `menu.mapOpenUrl` | 打开链接 | Open link |
| `menu.mapCopyText` | 复制这段字 | Copy text |
| `menu.mapCardMissing` | 卡片不存在 | Card missing |
| `menu.mapWorkflowMissing` | 工作流不存在 | Workflow missing |
| `menu.mapPayloadWorkflow` | 开始工作流 | Start workflow |
| `menu.mapPayloadText` | 发这段话 | Send text |
| `menu.mapPayloadCopy` | 复制这段字 | Copy text |
| `menu.mapPayloadMedia` | 发图 | Send media |
| `menu.mapPayloadUrl` | 打开链接 | Open link |
| `menu.mapThumbImage` | 图 | Image |
| `menu.mapThumbVideo` | 视频 | Video |
| `menu.mapThumbAnimation` | 动画 | Animation |
| `menu.mapCrumbOrphan` | 未挂上 | Unlinked |
| `menu.mapLoadFailed` | 菜单读取失败 | Menu failed to load |
| `menu.editMenu` | 改菜单 | Edit menu |
| `channels.tabMenu` | 菜单 | Menu |

删除 `channels.tabMenuHint`（zh + en）。`menu.mainKeyboard` 已是「主键盘」，复用，不要新建同义键。

- [ ] **Step 3.1: 写失败测试**

`menu-editor.contract.test.ts` 的 `V2_KEYS` 末尾追加：

```ts
  'mapPath',
  'mapEmpty',
  'mapOrphans',
  'mapCountKeys',
  'mapCountCards',
  'mapCountWorkflows',
  'mapOpenCard',
  'mapStartWorkflow',
  'mapSendText',
  'mapSendMedia',
  'mapOpenUrl',
  'mapCopyText',
  'mapCardMissing',
  'mapWorkflowMissing',
  'mapPayloadWorkflow',
  'mapPayloadText',
  'mapPayloadCopy',
  'mapPayloadMedia',
  'mapPayloadUrl',
  'mapThumbImage',
  'mapThumbVideo',
  'mapThumbAnimation',
  'mapCrumbOrphan',
  'mapLoadFailed',
```

新增断言 `editMenu` 中文：

```ts
it('editMenu is 改菜单', () => {
  const zh = JSON.parse(readFileSync(ZH, 'utf8')) as { menu: Record<string, string> }
  expect(zh.menu.editMenu).toBe('改菜单')
})
```

`channel-layout.contract.test.ts` 追加：

```ts
it('menu section has no hint and title is 菜单', () => {
  const detail = readFileSync(join(here, 'channel-detail-panel.tsx'), 'utf8')
  const zh = JSON.parse(
    readFileSync(join(here, '../../lib/i18n/locales/zh.json'), 'utf8')
  ) as { channels: Record<string, string> }
  expect(zh.channels.tabMenu).toBe('菜单')
  expect(zh.channels.tabMenuHint).toBeUndefined()
  expect(detail).not.toMatch(/tabMenuHint/)
})
```

（此步 detail 还没改，`tabMenuHint` 断言会失败，先只加 json 相关；**本步只改 json + V2_KEYS + editMenu 文案测试**。channel-layout 那条放到 Task 6。）

本任务只改 json 和 `V2_KEYS` + `editMenu is 改菜单`。

- [ ] **Step 3.2: 跑测试确认失败**

Run: `cd web/admin && pnpm exec vitest run src/features/menu/menu-editor.contract.test.ts`

Expected: FAIL missing `menu.mapPath` 等

- [ ] **Step 3.3: 写入 zh.json / en.json，改 `editMenu` 与 `channels.tabMenu`，删除 `tabMenuHint`**

按上表逐键写入 `menu` 对象（放在 `editMenu` 附近）。`channels.tabMenu` 改为「菜单」/ 已有 `"Menu"`。删除两个 locale 里的 `tabMenuHint` 行。

- [ ] **Step 3.4: 跑测试确认通过**

Run: `cd web/admin && pnpm exec vitest run src/features/menu/menu-editor.contract.test.ts src/lib/i18n/copy-quality.test.ts`

Expected: PASS（新中文不含请/您/感叹号）

---

### Task 4: `MenuCapabilityMap` 组件

**Files:**
- Create: `web/admin/src/features/menu/menu-capability-map.tsx`
- Create: `web/admin/src/features/menu/menu-capability-map.contract.test.ts`
- Modify: `web/admin/vitest.config.ts`（`include` 加上该 contract 文件）
- Consumes: Task 1–2 函数；Task 3 i18n 键；shadcn `Button`
- Produces: `<MenuCapabilityMap menu cards workflows onEdit />`，`data-testid='menu-capability-map'`

**交互（必须与纯函数一致，不要在组件里重写可达算法）：**

- `useState<MapTrailStep[]>([])`
- 点主键盘键 → `setTrail(trailFromItem(item))`
- 点未挂上芯片 → `setTrail(trailFromOrphan(card))`
- 点卡片按钮 → `setTrail((t) => pushTrailButton(t, button))`
- 点面包屑 index i → `setTrail((t) => sliceTrail(t, i))`
- `columns`：`menu.columns >= 1 && menu.columns <= 8 ? menu.columns : 2`
- 未挂上：`orphanCards(menu, cards)` 长度为 0 则不渲染口袋
- 空 `menu.items`：键盘区文案 `t('menu.mapEmpty')`，不渲染键
- 系统返回按钮：源码里不得出现 `‹ 返回` 或 `menu.backTo`

路径面板：`trail` 为空只渲染栏名 `menu.mapPath`。最后一步 `action`：

- `open_card` 且卡片存在 → 消息框：`name` 12px muted；`media` 缩略图（有 `url` 用 `<img>`，否则色块 + `t('menu.'+mediaKindLabelKey(kind))`）；正文；按钮 `min-height: 36px`
- 其他类型 → `payload-k` 用对应 `mapPayload*`；工作流名 / 文本 / URL / 媒体缩略图；断引用用 destructive 边框 + `mapCardMissing` / `mapWorkflowMissing`

顶栏：左 `t('channels.tabMenu')`（「菜单」）+ 三个 `tabular-nums` 计数（`items.length` / `cards.length` / `workflowEntryCount`）→ 右 `<Button data-testid='edit-menu' onClick={onEdit}>{t('menu.editMenu')}</Button>`

布局 class（不要改语义）：

- 外壳：`overflow-hidden rounded-[8px] border border-border bg-card`（无 shadow）
- 两栏：`grid min-h-[420px] md:grid-cols-2 max-md:grid-cols-1`；左栏 `border-r` 仅 `md:`
- 键：`grid gap-2` + `gridTemplateColumns: repeat(cols, minmax(0,1fr))`；键 `min-h-11`（44px）`border` 选中 `border-foreground bg-muted`；断引用 `border-destructive` + outcome `text-destructive`
- 920px：靠 `md:` 断点（Tailwind md=768。**规格是 920px。**）用任意值：`min-[920px]:grid-cols-2` 与 `min-[920px]:border-r`，不要用 `md:`。

- [ ] **Step 4.1: 写失败契约测试** `menu-capability-map.contract.test.ts`

```ts
import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const src = readFileSync(
  join(dirname(fileURLToPath(import.meta.url)), 'menu-capability-map.tsx'),
  'utf8'
)

describe('MenuCapabilityMap', () => {
  it('is read-only map with edit CTA and trail helpers', () => {
    expect(src).toContain("data-testid='menu-capability-map'")
    expect(src).toContain("data-testid='map-keyboard'")
    expect(src).toContain("data-testid='map-path'")
    expect(src).toContain("data-testid='map-key'")
    expect(src).toContain("data-testid='edit-menu'")
    expect(src).toContain('trailFromItem')
    expect(src).toContain('trailFromOrphan')
    expect(src).toContain('pushTrailButton')
    expect(src).toContain('sliceTrail')
    expect(src).toContain('orphanCards')
    expect(src).toContain('workflowEntryCount')
    expect(src).toContain('actionOutcomeLabel')
    expect(src).toContain('actionIsBroken')
    expect(src).toContain('min-[920px]:grid-cols-2')
    expect(src).toContain('min-h-11')
    expect(src).not.toContain('PhoneSimulation')
    expect(src).not.toContain('tab-outline')
    expect(src).not.toContain('‹ 返回')
    expect(src).not.toContain('shadow-')
    expect(src).not.toContain('ActionForm')
  })
})
```

- [ ] **Step 4.2: 跑测试确认失败**

先把该文件路径写入 `vitest.config.ts` 的 `include`。

Run: `cd web/admin && pnpm exec vitest run src/features/menu/menu-capability-map.contract.test.ts`

Expected: FAIL（ENOENT `menu-capability-map.tsx`）

- [ ] **Step 4.3: 实现组件**

对照原型 `docs/superpowers/prototypes/2026-08-29-channel-menu-capability-map.html`。完整骨架：

```tsx
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import type { Action, Card, Menu } from '@/lib/api/channel-menu'
import {
  actionIsBroken,
  actionOutcomeLabel,
  mediaKindLabelKey,
  orphanCards,
  pushTrailButton,
  sliceTrail,
  trailFromItem,
  trailFromOrphan,
  workflowEntryCount,
  type MapTrailStep,
} from './lib/menu-flow'
import type { WorkflowRef } from './node-view'

function labelOf(
  t: (k: string, o?: { name?: string; n?: number }) => string,
  action: Action,
  cards: Card[],
  workflows: WorkflowRef[]
) {
  const o = actionOutcomeLabel(action, cards, workflows)
  return t(`menu.${o.key}`, { name: o.name ?? '' })
}

export function MenuCapabilityMap({
  menu,
  cards,
  workflows,
  onEdit,
}: {
  menu: Menu
  cards: Card[]
  workflows: WorkflowRef[]
  onEdit: () => void
}) {
  const { t } = useTranslation()
  const [trail, setTrail] = useState<MapTrailStep[]>([])
  const cols = menu.columns >= 1 && menu.columns <= 8 ? menu.columns : 2
  const orphans = orphanCards(menu, cards)
  const last = trail[trail.length - 1]
  const selectedItemId = trail[0]?.kind === 'item' ? trail[0].id : null
  const selectedOrphanId = trail[0]?.kind === 'orphan' ? trail[0].id : null

  return (
    <div
      data-testid='menu-capability-map'
      className='overflow-hidden rounded-[8px] border border-border bg-card'
    >
      <div className='flex items-center justify-between gap-3 border-b border-border px-4 py-3'>
        <div className='flex min-w-0 flex-wrap items-center gap-3'>
          <h2 className='text-[15px] font-semibold'>{t('channels.tabMenu')}</h2>
          <div className='flex gap-3.5 font-mono text-xs tabular-nums text-muted-foreground'>
            <span>
              <b className='font-semibold text-foreground'>{menu.items.length}</b>{' '}
              {t('menu.mapCountKeys', { n: menu.items.length }).replace(/^\d+\s*/, '')}
            </span>
            {/* 不要用 replace。改成三个独立短键更干净则同时改 i18n：见下方计数。 */}
          </div>
        </div>
        <Button type='button' size='sm' data-testid='edit-menu' onClick={onEdit}>
          {t('menu.editMenu')}
        </Button>
      </div>
      {/* 左 map-keyboard / 右 map-path：min-[920px]:grid-cols-2；键 data-testid='map-key' */}
    </div>
  )
}
```

计数不要 `replace`。Task 3 的 `mapCountKeys` 已是 `{{n}} 键`。顶栏写成：

```tsx
<span className='font-mono text-xs tabular-nums'>
  {t('menu.mapCountKeys', { n: menu.items.length })}
</span>
```

i18n 里数字用 `{{n}}`，调用 `t('menu.mapCountKeys', { n: menu.items.length })`。若 i18next 默认插值是 `{{n}}` 即可。三个计数同理。

人话：`labelOf(t, action, cards, workflows)`。断引用键加 `actionIsBroken` class。

路径：`trail.length === 0` 时 `<div data-testid='map-path'><p>{t('menu.mapPath')}</p></div>`。有 trail 时面包屑根是 `trail[0].kind === 'orphan' ? t('menu.mapCrumbOrphan') : t('menu.mainKeyboard')`，中间节 `onClick={() => setTrail((cur) => sliceTrail(cur, i))}`。

最后一步 `open_card` 且 `cards.find(c => c.id === last.action.card_id)` 有值：渲染消息框与按钮，`onClick={() => setTrail((cur) => pushTrailButton(cur, b))}`。否则按 `last.action.type` 渲染 payload（`mapPayload*`）。

空键盘：`menu.items.length === 0` 时 `t('menu.mapEmpty')`。

未挂上：`orphans.length > 0` 才渲染 `data-testid='map-orphans'`，标题 `t('menu.mapOrphans', { n: orphans.length })`。

不要把 `‹ 返回` 画进去。不要 `accent-brand` / `text-accent-text` / `shadow-*`。

- [ ] **Step 4.4: 跑测试确认通过**

Run: `cd web/admin && pnpm exec vitest run src/features/menu/menu-capability-map.contract.test.ts src/features/menu/lib/menu-flow.test.ts`

Expected: PASS

---

### Task 5: 宿主 `MenuCardEditor` 换成地图

**Files:**
- Modify: `web/admin/src/features/menu/menu-card-editor.tsx`
- Modify: `web/admin/src/features/menu/menu-editor.contract.test.ts` 的 `menu view mode` describe

**当前主视图（要删）：** `Tabs` 大纲/卡片库 + `PhoneSimulation`。

**目标主视图：**

```tsx
return (
  <div data-testid='menu-card-editor'>
    <MenuCapabilityMap
      menu={menu}
      cards={cards}
      workflows={workflows}
      onEdit={() => setEditorOpen(true)}
    />
    <MenuEditorModal
      open={editorOpen}
      onOpenChange={setEditorOpen}
      channelId={channelId}
      savedMenu={menu}
      savedCards={cards}
      onSaved={() => {
        /* 不在宿主保留 selectedNode；地图内部 trail 随卸载/key 处理 */
      }}
    />
  </div>
)
```

保存后要清地图路径：给 `MenuCapabilityMap` 加 `key={menu.id + cards.map(c => c.id).join()}` **不够**（保存同 id）。改为 `mapEpoch`：`onSaved` 里 `setMapEpoch((n) => n + 1)`，`<MenuCapabilityMap key={mapEpoch} ... />`。

加载 / 失败：失败文案改 `t('menu.mapLoadFailed')`，不要 `common.errorGeneric`。加载仍用 `LoadingSkeleton`。

删除宿主里对 `OutlinePane` / `CardLibraryPane` / `PhoneSimulation` / `leftTab` / `selectedNode` 的引用。

- [ ] **Step 5.1: 改契约测试（先失败）**

把 `describe('menu view mode ...')` 整段换成：

```ts
describe('menu view mode (主视图只读能力地图)', () => {
  it('hosts MenuCapabilityMap and MenuEditorModal only', () => {
    const source = readFileSync(MENU_EDITOR, 'utf8')
    expect(source).toContain("data-testid='menu-card-editor'")
    expect(source).toContain('MenuCapabilityMap')
    expect(source).toContain('MenuEditorModal')
    expect(source).toContain('mapLoadFailed')
    expect(source).not.toContain('PhoneSimulation')
    expect(source).not.toContain("data-testid='left-pane'")
    expect(source).not.toContain("data-testid='preview-pane'")
    expect(source).not.toContain("data-testid='tab-outline'")
    expect(source).not.toContain("data-testid='tab-library'")
    expect(source).not.toContain('OutlinePane')
    expect(source).not.toContain('CardLibraryPane')
    expect(source).not.toContain("data-testid='add-menu-item'")
    expect(source).not.toContain("data-testid='save-menu'")
  })
})
```

弹层那条 `menu edit mode` **保持不变**（大纲 tab 仍在 modal）。

- [ ] **Step 5.2: 跑测试确认失败**

Run: `cd web/admin && pnpm exec vitest run src/features/menu/menu-editor.contract.test.ts`

Expected: FAIL（仍含 `PhoneSimulation` / `tab-outline`）

- [ ] **Step 5.3: 改 `menu-card-editor.tsx`**

按上面目标结构改。保留 `useQuery` 三条（menu / cards / cases）。`workflows` 映射不变。

- [ ] **Step 5.4: 跑测试确认通过**

Run: `cd web/admin && pnpm exec vitest run src/features/menu/menu-editor.contract.test.ts src/features/menu/menu-capability-map.contract.test.ts`

Expected: PASS

---

### Task 6: 详情页去掉 hint 与双层卡片

**Files:**
- Modify: `web/admin/src/features/channels/channel-detail-panel.tsx`
- Modify: `web/admin/src/features/channels/channel-layout.contract.test.ts`

菜单区块改为：

```tsx
<section id='channel-menu-section' className='flex flex-col gap-4'>
  <MenuCardEditor channelId={id} />
</section>
```

删除该 section 的 `SectionHead` 和外层 `kit.cardWrap`。文案 section 不动。

- [ ] **Step 6.1: 写失败测试**

```ts
it('menu section has no hint and no extra card wrap', () => {
  const detail = readFileSync(join(here, 'channel-detail-panel.tsx'), 'utf8')
  const zh = JSON.parse(
    readFileSync(join(here, '../../lib/i18n/locales/zh.json'), 'utf8')
  ) as { channels: Record<string, string> }
  expect(zh.channels.tabMenu).toBe('菜单')
  expect(zh.channels.tabMenuHint).toBeUndefined()
  expect(detail).not.toMatch(/tabMenuHint/)
  expect(detail).toMatch(/<section id='channel-menu-section'[\s\S]*?<MenuCardEditor/)
  const menuChunk = detail.split("id='channel-menu-section'")[1].split("id='channel-text-section'")[0]
  expect(menuChunk).not.toMatch(/SectionHead/)
  expect(menuChunk).not.toMatch(/kit\.cardWrap/)
})
```

保留原有 `#channel-menu-section` 在 LinkHealth 之前的断言。

- [ ] **Step 6.2: 跑测试确认失败**

Run: `cd web/admin && pnpm exec vitest run src/features/channels/channel-layout.contract.test.ts`

Expected: FAIL（仍有 `tabMenuHint` / `SectionHead`）

- [ ] **Step 6.3: 改 detail panel**

- [ ] **Step 6.4: 跑相关测试**

Run: `cd web/admin && pnpm exec vitest run src/features/channels/channel-layout.contract.test.ts src/features/menu/menu-editor.contract.test.ts src/lib/i18n/copy-quality.test.ts`

Expected: PASS

---

### Task 7: 全量前端回归

- [ ] **Step 7.1: 跑 admin vitest 全量**

Run: `cd web/admin && pnpm exec vitest run`

Expected: PASS

- [ ] **Step 7.2: 类型检查**

Run: `cd web/admin && pnpm exec tsc --noEmit`

Expected: 无错误

---

## Spec coverage

| Spec 节 | 任务 |
|---|---|
| §1 只读地图 + 改菜单弹层 | 4, 5 |
| §3 六种动作人话 / 断引用 / 未挂上定义 | 1 |
| §4 版面、920px、44px、不画返回 | 4 |
| §5 加载/失败/空/保存清路径 | 5 |
| §6 文案表 | 3 |
| §7 纯函数 + `MenuCapabilityMap` + 宿主 | 1, 2, 4, 5 |
| §8 测试 | 1, 2, 4, 5, 6, 7 |
| §9 设计系统 | 4（无 shadow / 无 accent / 单 CTA） |
| 详情挂载点 / 去 hint | 6 |
| 原型 HTML | 对照 Task 4，不改运行时 |

## 不在本计划

- `MenuEditorModal` 内部大纲/表单
- PhoneSimulation（主视图移除后，弹层若未用则保持现状）
- 架构文档 / 后端
