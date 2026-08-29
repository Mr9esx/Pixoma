# 改菜单弹层 · 可写能力地图 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans（本项目默认 build_mode）配合 superpowers:test-driven-development 逐任务实施。步骤用 `- [ ]` 勾选跟踪。

**Goal:** 把「改菜单」弹层换成与详情只读地图同构的可写能力地图（左主键盘、右点下去、底未挂上），清掉大纲/卡片库/双路按钮编辑/关窗丢草稿等交互债。

**Architecture:** 草稿与保存仍在 `MenuEditorModal`。纯函数（脏检测、列数夹紧、空白键/按钮）进 `menu-flow.ts`。抽出 `MenuMapLayout` 给只读地图与可写地图共用壳。可写交互在 `menu-writable-map.tsx`：格子改键名、点下去顶行改动作、打开卡片时在消息上改正文/媒体/按钮字。`CardPicker` + `buildNewCardDraft` 就地新建卡片。关窗未保存用已有 `AlertDialog`。

**Tech Stack:** React/TS · TanStack Query · shadcn Dialog / AlertDialog / Button / Input / Select / Textarea · vitest node 契约 + 纯函数 · i18n zh/en · Pixoma 语义令牌。

---
change: channel-menu-editor-refactor
design-doc: docs/superpowers/specs/2026-08-29-menu-editor-writable-map-design.md
read-map: docs/superpowers/specs/2026-08-29-channel-menu-capability-map-design.md
---

## Global Constraints

- 只改 `web/admin/src/` 前端；不改菜单/卡片 API、不改六种 `Action`、不改 bot 运行时。
- 文案成对写入 `zh.json` / `en.json`；中文走 Pixoma Voice（不用请/您/感叹号）。界面不写「芯片」。
- 取色只用语义令牌与 `color-mix`；表面无投影；弹层主 CTA 只有「保存」。
- 触屏命中 `min-h-11`（44px）；`< 920px` 上下叠、不横滑。
- 不画系统「‹ 返回」；弹层不挂 `OutlinePane` / `CardLibraryPane` / `PhoneSimulation` / `NewCardDialog`。
- 未挂上口径必须走 `orphanCards`（含按钮上的 `open_card`）。
- 选中未挂上卡片时不渲染动作行。
- vitest `environment: 'node'`；新测试加入 `web/admin/vitest.config.ts` 的 `include`。
- 不提交 git（用户未要求）。每任务末跳过 commit。

## File map

| 文件 | 职责 |
|---|---|
| `lib/menu-flow.ts` | 已有人话/trail/未挂上；新增脏检测、列数、空白键/按钮 |
| `menu-map-layout.tsx` | 只读/可写共用：顶栏槽 + 键盘栏 + 点下去栏 + 未挂上 |
| `menu-capability-map.tsx` | 只读；改用 layout |
| `menu-writable-map.tsx` | 可写地图（键名、动作行、消息编辑、加键） |
| `menu-editor-modal.tsx` | 草稿、保存、关窗确认；挂 writable map |
| `card-picker.tsx` | 保持已有/新建入口；新建由父级就地完成 |
| `locales/zh.json` `en.json` | 新键 |
| `menu-editor.contract.test.ts` | 弹层契约改写 |
| 删除（无引用后） | `outline-pane.tsx`、`card-library-pane.tsx`、`phone-simulation.tsx`、`new-card-dialog.tsx`、`item-editor.tsx`、`card-editor.tsx`、`button-editor.tsx` 及对应只测这些文件的契约 |

---

### Task 1: 草稿脏检测 / 列数 / 空白键按钮

**Files:**
- Modify: `web/admin/src/features/menu/lib/menu-flow.ts`
- Modify: `web/admin/src/features/menu/lib/menu-flow.test.ts`

**Interfaces:**
- Consumes: `Menu`, `MenuItem`, `Card`, `CardButton`, `Action` from `@/lib/api/channel-menu`
- Produces:
  ```ts
  export function clampMenuColumns(n: number): number
  export function isMenuDraftDirty(
    saved: { menu: Menu; cards: Card[] },
    draft: { menu: Menu; cards: Card[] }
  ): boolean
  export function createBlankMenuItem(id: string): MenuItem
  export function createBlankCardButton(id: string): CardButton
  ```

- [ ] **Step 1.1: 写失败测试**

在 `menu-flow.test.ts` 追加：

```ts
import {
  clampMenuColumns,
  createBlankCardButton,
  createBlankMenuItem,
  isMenuDraftDirty,
} from './menu-flow'

it('clampMenuColumns keeps 1–8, defaults 2', () => {
  expect(clampMenuColumns(1)).toBe(1)
  expect(clampMenuColumns(8)).toBe(8)
  expect(clampMenuColumns(0)).toBe(2)
  expect(clampMenuColumns(9)).toBe(2)
  expect(clampMenuColumns(3.7)).toBe(3)
})

it('isMenuDraftDirty is false when equal, true when columns or a label changes', () => {
  const menu: Menu = {
    id: 'ch',
    name: '',
    columns: 2,
    items: [{ id: 'i1', label: 'A', action: { type: 'send_text', text: 'x' } }],
  }
  const cards: Card[] = []
  expect(isMenuDraftDirty({ menu, cards }, { menu, cards })).toBe(false)
  expect(
    isMenuDraftDirty(
      { menu, cards },
      { menu: { ...menu, columns: 3 }, cards }
    )
  ).toBe(true)
  expect(
    isMenuDraftDirty(
      { menu, cards },
      {
        menu: {
          ...menu,
          items: [{ ...menu.items[0], label: 'B' }],
        },
        cards,
      }
    )
  ).toBe(true)
})

it('createBlankMenuItem defaults to empty send_text; createBlankCardButton same', () => {
  const item = createBlankMenuItem('mi-1')
  expect(item).toEqual({
    id: 'mi-1',
    label: '',
    action: { type: 'send_text', text: '' },
  })
  const btn = createBlankCardButton('b-1')
  expect(btn).toEqual({
    id: 'b-1',
    label: '',
    action: { type: 'send_text', text: '' },
  })
})
```

- [ ] **Step 1.2: 跑测试确认失败**

Run: `cd web/admin && pnpm exec vitest run src/features/menu/lib/menu-flow.test.ts`

Expected: FAIL（`clampMenuColumns` / `isMenuDraftDirty` / `createBlankMenuItem` 未导出）

- [ ] **Step 1.3: 最小实现**

在 `menu-flow.ts` 增加（`JSON.stringify` 比较前对 `media`/`buttons` 缺省成 `[]`）：

```ts
export function clampMenuColumns(n: number): number {
  const x = Math.floor(n)
  return x >= 1 && x <= 8 ? x : 2
}

export function createBlankMenuItem(id: string): MenuItem {
  return { id, label: '', action: { type: 'send_text', text: '' } }
}

export function createBlankCardButton(id: string): CardButton {
  return { id, label: '', action: { type: 'send_text', text: '' } }
}

function stableMenu(menu: Menu): Menu {
  return {
    ...menu,
    columns: clampMenuColumns(menu.columns),
    items: menu.items.map((it) => ({ ...it })),
  }
}

function stableCards(cards: Card[]): Card[] {
  return cards.map((c) => ({
    ...c,
    media: c.media ?? [],
    buttons: c.buttons ?? [],
  }))
}

export function isMenuDraftDirty(
  saved: { menu: Menu; cards: Card[] },
  draft: { menu: Menu; cards: Card[] }
): boolean {
  return (
    JSON.stringify({
      menu: stableMenu(saved.menu),
      cards: stableCards(saved.cards),
    }) !==
    JSON.stringify({
      menu: stableMenu(draft.menu),
      cards: stableCards(draft.cards),
    })
  )
}
```

补 `MenuItem` / `CardButton` 的 import（若文件尚未从 channel-menu 导入这两型）。

- [ ] **Step 1.4: 跑测试确认通过**

Run: `cd web/admin && pnpm exec vitest run src/features/menu/lib/menu-flow.test.ts`

Expected: PASS

---

### Task 2: i18n 键

**Files:**
- Modify: `web/admin/src/lib/i18n/locales/zh.json`
- Modify: `web/admin/src/lib/i18n/locales/en.json`
- Modify: `web/admin/src/features/menu/menu-editor.contract.test.ts`（`V2_KEYS`）

**Interfaces:**
- Produces keys under `menu.*`:
  - `columnCount` 列 / Columns
  - `addKey` + 加键 / Add key
  - `actionRow` 动作 / Action
  - `unsaved` 未保存 / Unsaved
  - `discardEdits` 丢弃这次修改？ / Discard these edits?
  - `discard` 丢弃 / Discard
  - `addCardButton` 已有「加一个按钮」；可继续用。芯片加号用 `addCardButton` 或保留短「+」不单独建键——按钮旁加号用 `+` 字符即可，不加新键。

- [ ] **Step 2.1: 写失败测试**

在 `V2_KEYS` 数组追加：`'columnCount', 'addKey', 'actionRow', 'unsaved', 'discardEdits', 'discard'`。

追加：

```ts
it('writable map chrome copy', () => {
  const zh = JSON.parse(readFileSync(ZH, 'utf8')) as { menu: Record<string, string> }
  expect(zh.menu.columnCount).toBe('列')
  expect(zh.menu.addKey).toBe('+ 加键')
  expect(zh.menu.actionRow).toBe('动作')
  expect(zh.menu.unsaved).toBe('未保存')
  expect(zh.menu.discardEdits).toBe('丢弃这次修改？')
  expect(zh.menu.discard).toBe('丢弃')
  expect(zh.menu.editMenu).toBe('改菜单')
})
```

- [ ] **Step 2.2: 跑测试确认失败**

Run: `cd web/admin && pnpm exec vitest run src/features/menu/menu-editor.contract.test.ts`

Expected: FAIL（缺键）

- [ ] **Step 2.3: 写入 json**

`zh.json` `menu` 对象（放在 `editMenu` 附近）：

```json
"columnCount": "列",
"addKey": "+ 加键",
"actionRow": "动作",
"unsaved": "未保存",
"discardEdits": "丢弃这次修改？",
"discard": "丢弃"
```

`en.json` 对应：

```json
"columnCount": "Columns",
"addKey": "+ Add key",
"actionRow": "Action",
"unsaved": "Unsaved",
"discardEdits": "Discard these edits?",
"discard": "Discard"
```

标题用已有 `editMenu`「改菜单」，不要再用「编辑菜单」当 DialogTitle。

- [ ] **Step 2.4: 跑测试确认通过**

Run: `cd web/admin && pnpm exec vitest run src/features/menu/menu-editor.contract.test.ts src/lib/i18n/copy-quality.test.ts`

Expected: PASS

---

### Task 3: 弹层契约改成可写地图（先红）

**Files:**
- Modify: `web/admin/src/features/menu/menu-editor.contract.test.ts` 的 `describe('menu edit mode')` 以及依赖已删文件的 describe（先改断言；实现在后续任务变绿）

**Interfaces:**
- 弹层必须：`menu-editor-modal`、`save-menu`、`map-keyboard`、`map-path`、`add-menu-item` 或改为 `add-key`（本计划统一 `data-testid='add-key'`）、`menu-columns`、`action-type-select`（有选中时）、`validateMenuConfig`、`putMenu`、`onSaved`
- 弹层禁止：`tab-outline`、`tab-library`、`phone-simulation`、`new-card` 顶栏、`OutlinePane`、`CardLibraryPane`、`PhoneSimulation`、`NewCardDialog`、`ItemEditor`、`CardEditor`、`editor-pane`

- [ ] **Step 3.1: 重写失败测试**

把 `describe('menu edit mode')` 整段换成：

```ts
describe('menu edit mode (可写能力地图弹层)', () => {
  it('is a writable map: keyboard + path + save, no outline/library/phone', () => {
    const source = readFileSync(MENU_EDITOR_MODAL, 'utf8')
    expect(source).toContain("data-testid='menu-editor-modal'")
    expect(source).toContain("data-testid='save-menu'")
    expect(source).toContain("data-testid='add-key'")
    expect(source).toContain("data-testid='menu-columns'")
    expect(source).toContain("data-testid='map-keyboard'")
    expect(source).toContain("data-testid='map-path'")
    expect(source).toContain('validateMenuConfig')
    expect(source).toContain('putMenu')
    expect(source).toContain('onSaved')
    expect(source).toContain('isMenuDraftDirty')
    expect(source).toContain('discardEdits')
    expect(source).not.toContain("data-testid='tab-outline'")
    expect(source).not.toContain("data-testid='tab-library'")
    expect(source).not.toContain("data-testid='phone-simulation'")
    expect(source).not.toContain("data-testid='new-card'")
    expect(source).not.toContain("data-testid='editor-pane'")
    expect(source).not.toContain('OutlinePane')
    expect(source).not.toContain('CardLibraryPane')
    expect(source).not.toContain('PhoneSimulation')
    expect(source).not.toContain('NewCardDialog')
    expect(source).not.toContain('ItemEditor')
    expect(source).not.toContain('CardEditor')
    expect(source).not.toContain('ButtonEditor')
  })
})
```

`describe('phone simulation')` / `outline pane` / `card library pane` / `item editor` / `button editor` / `card editor` / `new card dialog`：**先留着**（文件还在）。Task 8 删文件时再改成「这些文件不存在」或从 vitest include 移除并删 describe。

- [ ] **Step 3.2: 跑测试确认失败**

Run: `cd web/admin && pnpm exec vitest run src/features/menu/menu-editor.contract.test.ts`

Expected: FAIL（弹层仍有 `tab-outline` / `new-card`，没有 `add-key` / `menu-columns`）

不要在本任务改 `menu-editor-modal.tsx`。

---

### Task 4: 抽出 `MenuMapLayout`

**Files:**
- Create: `web/admin/src/features/menu/menu-map-layout.tsx`
- Modify: `web/admin/src/features/menu/menu-capability-map.tsx`
- Modify: `web/admin/src/features/menu/menu-capability-map.contract.test.ts`（断言仍有 `menu-capability-map` / `map-keyboard` / `map-path` / `edit-menu`）

**Interfaces:**
- Consumes: `ReactNode`
- Produces:
  ```tsx
  export function MenuMapLayout(props: {
    toolbar: React.ReactNode
    keyboard: React.ReactNode
    path: React.ReactNode
    orphans?: React.ReactNode
    testId?: string
  }): JSX.Element
  ```

壳 class：外层 `overflow-hidden rounded-[8px] border border-border bg-card`；两栏 `grid min-h-[420px] grid-cols-1 min-[920px]:grid-cols-2`；左栏 `p-4 min-[920px]:border-r border-border`。弹层里可把 `min-h` 改小，用可选 `className`：

```tsx
export function MenuMapLayout(props: {
  toolbar: React.ReactNode
  keyboard: React.ReactNode
  path: React.ReactNode
  orphans?: React.ReactNode
  testId?: string
  className?: string
}): JSX.Element
```

- [ ] **Step 4.1: 写失败测试**

在 `menu-capability-map.contract.test.ts` 追加：

```ts
it('uses MenuMapLayout', () => {
  const src = readFileSync(join(here, 'menu-capability-map.tsx'), 'utf8')
  expect(src).toContain('MenuMapLayout')
})
```

- [ ] **Step 4.2: 跑测试确认失败**

Run: `cd web/admin && pnpm exec vitest run src/features/menu/menu-capability-map.contract.test.ts`

Expected: FAIL（没有 `MenuMapLayout`）

- [ ] **Step 4.3: 抽出并改只读地图**

`menu-map-layout.tsx`：

```tsx
import { cn } from '@/lib/utils'

export function MenuMapLayout({
  toolbar,
  keyboard,
  path,
  orphans,
  testId,
  className,
}: {
  toolbar: React.ReactNode
  keyboard: React.ReactNode
  path: React.ReactNode
  orphans?: React.ReactNode
  testId?: string
  className?: string
}) {
  return (
    <div
      data-testid={testId}
      className={cn(
        'overflow-hidden rounded-[8px] border border-border bg-card',
        className
      )}
    >
      <div className='flex items-center justify-between gap-3 border-b border-border px-4 py-3'>
        {toolbar}
      </div>
      <div className='grid min-h-[420px] grid-cols-1 min-[920px]:grid-cols-2'>
        <div className='min-w-0 border-border p-4 min-[920px]:border-r'>
          {keyboard}
        </div>
        <div className='min-w-0 p-4' data-testid='map-path'>
          {path}
        </div>
      </div>
      {orphans}
    </div>
  )
}
```

`MenuCapabilityMap` 用该壳包住现有 toolbar/键盘/点下去/未挂上；`data-testid='menu-capability-map'` 放在 layout 的 `testId`。`map-keyboard` 仍在键盘 children 上。

- [ ] **Step 4.4: 跑测试确认通过**

Run: `cd web/admin && pnpm exec vitest run src/features/menu/menu-capability-map.contract.test.ts`

Expected: PASS（只读行为字符串契约不回退）

---

### Task 5: `MenuWritableMap` + 接进弹层（契约变绿的主体）

**Files:**
- Create: `web/admin/src/features/menu/menu-writable-map.tsx`
- Modify: `web/admin/src/features/menu/menu-editor-modal.tsx`
- Test: 沿用 Task 3 契约（本任务必须使其 PASS）
- Create: `web/admin/src/features/menu/menu-writable-map.contract.test.ts`
- Modify: `web/admin/vitest.config.ts` include 该文件

**Interfaces:**
- Consumes: `Menu`, `Card[]`, `WorkflowRef[]`, `MapTrailStep[]`, `clampMenuColumns`, `createBlankMenuItem`, `createBlankCardButton`, `orphanCards`, trail helpers, `ActionForm`, `CardPicker`
- Produces:
  ```tsx
  export function MenuWritableMap(props: {
    menu: Menu
    cards: Card[]
    workflows: WorkflowRef[]
    trail: MapTrailStep[]
    onTrailChange: (next: MapTrailStep[]) => void
    onMenuChange: (next: Menu) => void
    onCardsChange: (next: Card[]) => void
    onAddKey: () => void
  }): JSX.Element
  ```

弹层继续持有 `save` / `putMenu` / 关窗。`uid` 可留在 modal，加键时 `createBlankMenuItem(uid('mi'))`。

- [ ] **Step 5.1: 写失败测试（writable 契约）**

`menu-writable-map.contract.test.ts`：

```ts
import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const here = dirname(fileURLToPath(import.meta.url))
const src = () => readFileSync(join(here, 'menu-writable-map.tsx'), 'utf8')

describe('MenuWritableMap', () => {
  it('edits key labels on the grid and actions only in the path pane', () => {
    const s = src()
    expect(s).toContain("data-testid='map-keyboard'")
    expect(s).toContain("data-testid='add-key'")
    expect(s).toContain("data-testid='map-key-label'")
    expect(s).toContain('ActionForm')
    expect(s).toContain("t('menu.actionRow')")
    expect(s).toContain('trailFromItem')
    expect(s).toContain('pushTrailButton')
    expect(s).toContain('kind === \'orphan\'')
  })
})
```

- [ ] **Step 5.2: 跑测试确认失败**

Run: `cd web/admin && pnpm exec vitest run src/features/menu/menu-writable-map.contract.test.ts`

Expected: FAIL（文件不存在）

- [ ] **Step 5.3: 实现可写地图（键盘 + 动作行 + 非卡片 payload）**

要点（必须全部做到）：

1. `MenuMapLayout`，`testId` 可省略（testid 在 modal 根上）。键盘栏标题 `t('menu.mainKeyboard')`。
2. 每键：外层 `button` 或可点容器 `data-testid='map-key'` 选中 `onTrailChange(trailFromItem(it))`。键名 `<input data-testid='map-key-label'>`，`onChange` 只改 `label`，`stopPropagation` 避免误选。第二行人话只读，用既有 `actionOutcomeLabel` + `t('menu.'+key)`。
3. `+ 加键`：`data-testid='add-key'`，`onAddKey()`。
4. `menu.columns` 不在本组件改——列数控件在 modal 顶栏。本组件用 `clampMenuColumns(menu.columns)` 排格子。
5. 点下去：栏名 `t('menu.mapPath')`。有 trail 时面包屑与只读地图相同。
6. **动作行：** `last.kind !== 'orphan'` 时渲染 `<p>{t('menu.actionRow')}</p>` + `ActionForm`（`action={last.action}`）。`onChange`：若 `kind==='item'` 更新 `menu.items`；若 `kind==='btn'` 更新所属卡的该按钮。`onCreateNewCard` 由父级传入或内部调用 `onCardsChange` + `buildNewCardDraft`（Task 6 补名称输入；本任务可先 `buildNewCardDraft({ name: t('menu.untitled'), text: '' })` 并 `onPick` 其 id）。
7. **orphan：** 不渲染 `ActionForm` / `actionRow`。下面直接编辑该卡（Task 6 消息体；本任务至少显示卡片名 input）。
8. 非 `open_card` 的 payload：复用 `ActionForm` 已有字段（它已按 type 渲染）。因此动作行用完整 `ActionForm` 即可覆盖工作流/文本/媒体/URL。`open_card` 时 `ActionForm` 含 `CardPicker`；**卡片消息体不要再挂第二份 ActionForm**（消息编辑在 Task 6）。
9. 选中键时提供删除：`data-testid='delete-menu-item'`，从 `items` 去掉并 `onTrailChange([])`。
10. 断引用：格子人话用 destructive class（与只读地图相同 `actionIsBroken`）。

`ActionForm` 的 `open_card` 会显示 CardPicker。卡片消息（图/正文/按钮）是额外一块，只在 `last.action.type==='open_card'` 且卡片存在时渲染——Task 6。本任务可先只渲染 ActionForm。

顶栏列数 + 保存 + dirty 在 modal：

```tsx
// menu-editor-modal 顶栏（示意）
<DialogTitle>{t('menu.editMenu')}</DialogTitle>
<Label htmlFor='menu-columns'>{t('menu.columnCount')}</Label>
<Input
  id='menu-columns'
  data-testid='menu-columns'
  type='number'
  min={1}
  max={8}
  value={clampMenuColumns(baseMenu.columns)}
  onChange={(e) =>
    setMenuDraft({
      ...baseMenu,
      columns: clampMenuColumns(Number(e.target.value)),
    })
  }
/>
{isMenuDraftDirty({ menu: savedMenu, cards: savedCards }, { menu: baseMenu, cards: baseCards })
  ? t('menu.unsaved')
  : null}
<Button data-testid='save-menu' onClick={() => void save()}>{t('common.save')}</Button>
```

去掉顶栏 `add-menu-item`、`new-card`、Tabs、ItemEditor、CardEditor、NewCardDialog。body 只挂 `MenuWritableMap`。草稿：`menuDraft ?? savedMenu` 与现在相同。`addKey`：`createBlankMenuItem` + 选中 `trailFromItem`。

关窗确认可先不做（Task 7）。本任务至少让 Task 3 契约里除 `discardEdits` 外的断言变绿——**把 `discardEdits` 留在 Task 7**。若 Task 3 已要求 `discardEdits`，本任务在 modal 里先 `import` 并放一个隐藏/未接好的字符串引用不够；应在本任务 `t('menu.discardEdits')` 写进 AlertDialog（可先 `open={false}` 的死代码？禁止。要么 Task 3 把 discard 挪到 Task 7，要么本任务做完整 AlertDialog。

**决定：Task 3 契约已含 `discardEdits`。本任务必须接 AlertDialog，逻辑在 Task 7 写完整也行，但字符串必须出现。** 最小：关窗 `handleOpenChange` 若 dirty 则 `setDiscardOpen(true)` 否则关。AlertDialog 用 `t('menu.discardEdits')` / `t('menu.discard')` / `t('common.cancel')`。

参考 `channel-detail-panel.tsx` 的 AlertDialog 用法（受控 `open` + `onOpenChange`，不要 Trigger 包保存按钮）。

```tsx
const [discardOpen, setDiscardOpen] = useState(false)

function handleOpenChange(next: boolean) {
  if (!next && isMenuDraftDirty(...)) {
    setDiscardOpen(true)
    return
  }
  if (!next) resetDrafts()
  onOpenChange(next)
}
```

- [ ] **Step 5.4: 跑测试确认通过**

Run:

```
cd web/admin && pnpm exec vitest run \
  src/features/menu/menu-editor.contract.test.ts \
  src/features/menu/menu-writable-map.contract.test.ts \
  src/features/menu/menu-capability-map.contract.test.ts
```

Expected: PASS

若 `menu-writable-map` 未进 include，先改 `vitest.config.ts`。

---

### Task 6: 卡片消息可写 + 按钮往下走 + 就地新建

**Files:**
- Modify: `web/admin/src/features/menu/menu-writable-map.tsx`
- Modify: `web/admin/src/features/menu/menu-writable-map.contract.test.ts`
- Modify: `web/admin/src/features/menu/card-picker.tsx`（仅当需要 `data-testid`；逻辑可留在 writable map）

**Interfaces:**
- 打开卡片且卡存在：消息框 `data-testid='edit-card'`：`card-name`、正文 textarea、媒体行（URL + kind select：image/video/animation）、按钮列表
- 按钮：`data-testid='edit-card-button'`，label input，只读人话，`onClick` 容器推进 `pushTrailButton`（label input `stopPropagation`）
- `data-testid='add-button'` 追加 `createBlankCardButton`
- 点进按钮后：动作行改该按钮；`data-testid='delete-button'` 可删
- 新建：`ActionForm`/`CardPicker` 的 `onCreateNew` 展开 `data-testid='new-card-name'` + `data-testid='new-card-confirm'`，调用 `validateNewCardDraft` / `buildNewCardDraft`，挂到当前 `open_card.card_id`，**不**渲染 `NewCardDialog`

- [ ] **Step 6.1: 写失败测试**

在 `menu-writable-map.contract.test.ts` 追加：

```ts
it('edits open_card as a message and pushes trail on button click', () => {
  const s = src()
  expect(s).toContain("data-testid='edit-card'")
  expect(s).toContain("data-testid='add-button'")
  expect(s).toContain("data-testid='edit-card-button'")
  expect(s).toContain('pushTrailButton')
  expect(s).toContain('buildNewCardDraft')
  expect(s).toContain("data-testid='new-card-name'")
  expect(s).toContain("data-testid='new-card-confirm'")
  expect(s).not.toContain('NewCardDialog')
  expect(s).not.toContain('ButtonEditor')
})
```

- [ ] **Step 6.2: 跑测试确认失败**

Run: `cd web/admin && pnpm exec vitest run src/features/menu/menu-writable-map.contract.test.ts`

Expected: FAIL（缺 `edit-card`）

- [ ] **Step 6.3: 实现消息编辑**

找当前卡：

```ts
function cardFromStep(step: MapTrailStep, cards: Card[]): Card | undefined {
  if (step.kind === 'orphan') return cards.find((c) => c.id === step.id)
  if (step.action.type === 'open_card' && step.action.card_id) {
    return cards.find((c) => c.id === step.action.card_id)
  }
  return undefined
}
```

当 `last.kind==='orphan'` 或（`last.action.type==='open_card'` 且找到卡）时渲染消息。媒体每行：`url` input + `Select` kind `image|video|animation`。缩略图规则与只读地图 `MediaThumbs` 相同（可把 `MediaThumbs` 抽到 `menu-map-layout.tsx` 或 `media-thumbs.tsx` 以免复制；若抽取，只读地图改为 import）。

按钮人话只读。点按钮（非输入）：`onTrailChange(pushTrailButton(trail, b))`。

`open_card` 且卡缺失：显示 `t('menu.mapCardMissing')`，动作行（非 orphan）仍可换卡。

新建卡片 UI：本地 `useState` `creatingCard`。`CardPicker onCreateNew={() => setCreatingCard(true)}`。确认：

```ts
const v = validateNewCardDraft({ name, text: '' })
if (!v.ok) return
const card = buildNewCardDraft({ name, text: '' })
onCardsChange([...cards, card])
// 把当前 item 或 button 的 action.card_id 设为 card.id
setCreatingCard(false)
```

- [ ] **Step 6.4: 跑测试确认通过**

Run: `cd web/admin && pnpm exec vitest run src/features/menu/menu-writable-map.contract.test.ts src/features/menu/menu-editor.contract.test.ts`

Expected: PASS

---

### Task 7: 未挂上 / 删除卡片 / 关窗确认（补全）

**Files:**
- Modify: `web/admin/src/features/menu/menu-writable-map.tsx`
- Modify: `web/admin/src/features/menu/menu-editor-modal.tsx`
- Modify: `web/admin/src/features/menu/menu-writable-map.contract.test.ts`

**Interfaces:**
- 未挂上：`data-testid='map-orphans'`，`orphanCards(menu, cards)`，点了 `trailFromOrphan(c)`
- 正在改卡时 `data-testid='delete-card'`
- 关窗 dirty → AlertDialog；确定 `resetDrafts` + `onOpenChange(false)`；取消留在弹层

- [ ] **Step 7.1: 写失败测试**

```ts
it('lists orphans with map-orphans and can delete a card', () => {
  const s = src()
  expect(s).toContain("data-testid='map-orphans'")
  expect(s).toContain('orphanCards')
  expect(s).toContain('trailFromOrphan')
  expect(s).toContain("data-testid='delete-card'")
})
```

modal 契约已要求 `discardEdits`。再在 `menu-editor.contract.test.ts` 的 edit mode 追加：

```ts
expect(source).toContain('AlertDialog')
expect(source).toContain("t('menu.discard')")
```

- [ ] **Step 7.2: 跑测试确认失败**

Run: `cd web/admin && pnpm exec vitest run src/features/menu/menu-writable-map.contract.test.ts src/features/menu/menu-editor.contract.test.ts`

Expected: 若未挂上未接，FAIL

- [ ] **Step 7.3: 实现**

未挂上块与只读地图相同结构，虚线边框。删除卡片：`onCardsChange(cards.filter(...))`，清空 trail。不要在未挂上自动删卡。

AlertDialogFooter：`AlertDialogCancel` → `t('common.cancel')`；`AlertDialogAction` → `t('menu.discard')`。

- [ ] **Step 7.4: 跑测试确认通过**

Run: 同上。Expected: PASS

---

### Task 8: 删除死代码并收契约

**Files:**
- Delete（确认 `web/admin/src` 无其它 import）：
  - `outline-pane.tsx` `outline-pane.test.ts`
  - `card-library-pane.tsx` `card-library-pane.test.ts`
  - `phone-simulation.tsx`
  - `new-card-dialog.tsx`（保留 `card-draft.ts` + `new-card-dialog.test.ts` 对 draft 的测试——把 test 文件改 import 为 `./card-draft`，或改名为 `card-draft.test.ts`）
  - `item-editor.tsx`
  - `card-editor.tsx` `card-editor.test.ts`
  - `button-editor.tsx`
- Modify: `menu-editor.contract.test.ts` 删除针对已删文件的 describe（phone / outline / library / item / button / card editor / new card dialog UI）
- Modify: `vitest.config.ts` 去掉已删测试路径；若改名为 `card-draft.test.ts` 则加入 include
- `ActionForm` / `CardPicker` / `action-templates` **保留**

- [ ] **Step 8.1: 写失败测试**

在 `menu-editor.contract.test.ts` 追加：

```ts
it('dead editor panes are gone', () => {
  const files = [
    'outline-pane.tsx',
    'card-library-pane.tsx',
    'phone-simulation.tsx',
    'new-card-dialog.tsx',
    'item-editor.tsx',
    'card-editor.tsx',
    'button-editor.tsx',
  ]
  for (const f of files) {
    expect(existsSync(join(here, f))).toBe(false)
  }
})
```

需要 `import { existsSync } from 'node:fs'`。

- [ ] **Step 8.2: 跑测试确认失败**

Run: `cd web/admin && pnpm exec vitest run src/features/menu/menu-editor.contract.test.ts`

Expected: FAIL（文件仍存在）

- [ ] **Step 8.3: 删除文件、改测试、改 include**

`new-card-dialog.test.ts`：若仍测 draft，改为 `from './card-draft'`，文件名可保留以免大范围改 include，或 rename。**不要**留下对 `NewCardDialog` 组件的契约。

Grep `web/admin/src` 确认无残留 import。

- [ ] **Step 8.4: 跑测试确认通过**

Run: `cd web/admin && pnpm exec vitest run src/features/menu/menu-editor.contract.test.ts src/features/menu/new-card-dialog.test.ts`

Expected: PASS（或改名后的 draft 测试 PASS）

---

### Task 9: 回归

**Files:** 无新文件。

- [ ] **Step 9.1: tsc**

Run: `cd web/admin && pnpm exec tsc --noEmit`

Expected: 无错误

- [ ] **Step 9.2: menu + i18n + channel-layout vitest**

Run:

```
cd web/admin && pnpm exec vitest run \
  src/features/menu \
  src/lib/i18n/copy-quality.test.ts \
  src/lib/i18n/locale.test.ts \
  src/features/channels/channel-layout.contract.test.ts
```

Expected: 上述全部 PASS。若全量 vitest 里 `workbench-welcome-card` 高度失败，与本需求无关，不修。

- [ ] **Step 9.3: 手验清单（实现者在浏览器过一遍）**

- 详情只读地图仍可点键看点下去；「改菜单」打开弹层
- 格子改键名；第二行人话点不了
- 点下去顶行可把键从发文字改成打开卡片，选已有/新建
- 卡片按钮改字；点按钮往下走；顶行动作改成打开另一张卡
- 列数 1–8 格子重排
- + 加键；保存后详情地图刷新
- 未保存关窗出现「丢弃这次修改？」
- 未挂上口径与详情一致；点口袋无动作行

---

## Spec coverage（自检）

| Spec 节 | 任务 |
|---|---|
| 可写地图同构、无大纲/库/手机 | 3, 4, 5, 8 |
| 格子只改键名；动作仅点下去顶行 | 5 |
| 按钮与键同一套动作、可再打开卡片 | 5, 6 |
| 未挂上无动作行、口径 orphanCards | 1 已有 orphanCards，7 |
| 列数、加键、保存唯一主 CTA | 1, 2, 5 |
| 就地新建、无 NewCardDialog | 6, 8 |
| 关窗确认、未保存文案 | 2, 5, 7 |
| 六种动作字段 | ActionForm 复用，5–6 |
| 媒体 URL+kind | 6 |
| 删除死代码 | 8 |
| 只读地图不回退 | 4, 9 |
| 不改 API / 运行时 | Global Constraints |

无 TBD。类型名全程：`clampMenuColumns` / `isMenuDraftDirty` / `createBlankMenuItem` / `createBlankCardButton` / `MenuMapLayout` / `MenuWritableMap`。
