# TG 菜单/卡片重构 · 前端（分层链式编辑器）Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 全新所见即所得编辑器：左边 TG 手机模拟（主键盘 + 卡片流 + 返回），右边沿链分层编辑（⓪ 菜单 → ① 菜单项 → ② 卡片 → ③ 按钮 → ④ 卡片…无限）；卡片就地创建/复用；校验只在保存时。

**Architecture:** 纯函数层（`menu-flow.ts` 模拟与校验）与组件层分离；新编辑器 `MenuCardEditor.tsx` 持有 `Menu + Card[]` 草稿，PUT 全量保存；`ActionForm` 按动作类型渲染参数表单；`CardPicker` 就地新建或选已有卡片。

**Tech Stack:** React 19 + TS + Vite + TanStack Query/Router + Tailwind + radix-ui + vitest（contract test 读源码）。

## Global Constraints

- 测试命令：`cd web/admin && pnpm test`；类型 `pnpm exec tsc -b`；格式/质量 `pnpm exec prettier --write` / `pnpm exec eslint`。
- 新测试文件必须加入 `web/admin/vitest.config.ts` include 列表。
- 用户文案禁止内部术语（Menu/MenuItem/CardButton/Action 等），统一「菜单项/卡片/按钮/动作」大白话。
- 主键盘无数量上限；校验只在保存时；返回按钮自动生成（不配置）。
- 完全替换旧编辑器：旧 `channel-menu-editor.tsx` / `phone-simulation.tsx` / `node-config-panel.tsx` / `params-form.tsx` / `menu-simulation.ts` / `menu-validate.ts` 与旧 API 类型在本计划删除。

---

### Task F1: 纯函数（buildTgFlow / validateMenuConfig）

**Files:**
- Create: `web/admin/src/features/menu/lib/menu-flow.ts`、`menu-flow.test.ts`
- Modify: `web/admin/vitest.config.ts`

**Interfaces:**
- Consumes: `Menu` / `MenuItem` / `Card` / `CardButton` / `Action` 类型（Task F3 定义，先在此定义临时类型或直接定义在 `menu-flow.ts` 并由 F3 复用）
- Produces:
  - `export type ActionType = 'open_card' | 'open_workflow' | 'send_text' | 'send_media' | 'open_url' | 'copy_text' | 'placeholder'`
  - `export type Action = { type: ActionType; card_id?: string; workflow_ids?: string[]; mode?: 'list' | 'direct'; text?: string; media?: { kind: string; url: string; caption?: string }[]; url?: string }`
  - `export type MenuItem = { id: string; label: string; action: Action }`
  - `export type Menu = { id: string; name: string; columns: number; items: MenuItem[] }`
  - `export type CardButton = { id: string; label: string; action: Action }`
  - `export type Card = { id: string; name: string; media: { kind: string; url: string; caption?: string }[]; text: string; buttons: CardButton[] }`
  - `export function buildTgFlow(menu: Menu, cards: Card[]): { keyboard: string[][]; cardById: Map<string, Card> }`
  - `export function validateMenuConfig(menu: Menu, cards: Card[]): { ok: boolean; errors: { path: string; message: string }[] }`

- [ ] **Step 1: 写失败测试**

```ts
import { describe, expect, it } from 'vitest'
import { buildTgFlow, validateMenuConfig, type Action, type Card, type Menu } from './menu-flow'

const placeholder: Action = { type: 'placeholder' }

describe('menu flow', () => {
  it('builds keyboard rows without a 6-button cap', () => {
    const menu: Menu = {
      id: 'm', name: '主', columns: 2,
      items: Array.from({ length: 7 }, (_, i) => ({ id: `i${i}`, label: `B${i}`, action: placeholder })),
    }
    const { keyboard } = buildTgFlow(menu, [])
    expect(keyboard.flat().length).toBe(7)
    expect(keyboard[0]).toEqual(['B0', 'B1'])
  })

  it('collects cards reachable from menu and buttons', () => {
    const cardA: Card = { id: 'a', name: 'A', media: [], text: 'A', buttons: [{ id: 'b1', label: '去B', action: { type: 'open_card', card_id: 'b' } }] }
    const cardB: Card = { id: 'b', name: 'B', media: [], text: 'B', buttons: [] }
    const menu: Menu = { id: 'm', name: '主', columns: 2, items: [{ id: 'mi', label: '入口', action: { type: 'open_card', card_id: 'a' } }] }
    const flow = buildTgFlow(menu, [cardA, cardB])
    expect(flow.cardById.get('a')).toEqual(cardA)
    expect(flow.cardById.get('b')).toEqual(cardB)
  })

  it('validates only on save: missing label, action target, workflow ids', () => {
    const menu: Menu = {
      id: 'm', name: '主', columns: 2,
      items: [
        { id: 'x', label: '', action: placeholder },
        { id: 'y', label: 'Y', action: { type: 'open_card' } },
        { id: 'z', label: 'Z', action: { type: 'open_workflow', workflow_ids: [] } },
      ],
    }
    const errors = validateMenuConfig(menu, []).errors
    expect(errors.length).toBeGreaterThanOrEqual(3)
  })
})
```

- [ ] **Step 2: 运行确认失败**

Run: `cd web/admin && pnpm test src/features/menu/lib/menu-flow.test.ts`
Expected: FAIL（模块不存在）。

- [ ] **Step 3: 实现 menu-flow.ts**

```ts
export type ActionType = 'open_card' | 'open_workflow' | 'send_text' | 'send_media' | 'open_url' | 'copy_text' | 'placeholder'
export type Action = { type: ActionType; card_id?: string; workflow_ids?: string[]; mode?: 'list' | 'direct'; text?: string; media?: { kind: string; url: string; caption?: string }[]; url?: string }
export type MenuItem = { id: string; label: string; action: Action }
export type Menu = { id: string; name: string; columns: number; items: MenuItem[] }
export type CardButton = { id: string; label: string; action: Action }
export type Card = { id: string; name: string; media: { kind: string; url: string; caption?: string }[]; text: string; buttons: CardButton[] }

export function buildTgFlow(menu: Menu, cards: Card[]) {
  const cols = menu.columns >= 1 && menu.columns <= 8 ? menu.columns : 2
  const keyboard: string[][] = []
  let row: string[] = []
  for (const it of menu.items) {
    row.push(it.label)
    if (row.length >= cols) { keyboard.push(row); row = [] }
  }
  if (row.length > 0) keyboard.push(row)
  const cardById = new Map(cards.map((c) => [c.id, c]))
  return { keyboard, cardById }
}

export function validateMenuConfig(menu: Menu, cards: Card[]) {
  const errors: { path: string; message: string }[] = []
  menu.items.forEach((it, i) => {
    const path = `menu.items[${i}]`
    if (!it.label.trim()) errors.push({ path, message: 'label' })
    validateAction(it.action, path, cards, errors)
  })
  for (const card of cards) {
    card.buttons.forEach((b, i) => {
      const path = `card:${card.id}.buttons[${i}]`
      if (!b.label.trim()) errors.push({ path, message: 'label' })
      validateAction(b.action, path, cards, errors)
    })
  }
  return { ok: errors.length === 0, errors }
}

function validateAction(a: Action, path: string, cards: Card[], errors: { path: string; message: string }[]) {
  if (a.type === 'open_card' && !a.card_id) errors.push({ path, message: 'card' })
  if (a.type === 'open_card' && a.card_id && !cards.some((c) => c.id === a.card_id)) errors.push({ path, message: 'card_missing' })
  if (a.type === 'open_workflow' && (!a.workflow_ids || a.workflow_ids.length === 0)) errors.push({ path, message: 'workflow' })
  if (a.type === 'open_url' && !/^https?:\/\//.test(a.url ?? '')) errors.push({ path, message: 'url' })
  if (a.type === 'send_media' && (!a.media || a.media.length === 0)) errors.push({ path, message: 'media' })
}
```

- [ ] **Step 4: 运行确认通过 + 提交**

```bash
cd web/admin && pnpm test src/features/menu/lib/menu-flow.test.ts && pnpm exec tsc -b
pnpm exec prettier --write src/features/menu/lib web/admin/vitest.config.ts
cd /Users/mr9esx/Documents/Pixoma
git add web/admin/vitest.config.ts web/admin/src/features/menu/lib
git commit -m "feat(admin): menu/card flow simulation and validation"
```

---

### Task F2: i18n 文案

**Files:**
- Modify: `web/admin/src/features/menu/menu-editor.contract.test.ts`（新建，含 key 断言）
- Modify: `web/admin/src/lib/i18n/locales/zh.json` / `en.json`
- Modify: `web/admin/vitest.config.ts`（include 新 contract 文件）

**Interfaces:**
- Produces: `menu.*` keys：`mainKeyboard`、`columnsPerRow`、`addMenuItem`、`menuItemLabel`、`menuItemAction`、`cardName`、`cardMedia`、`cardText`、`cardButtons`、`addCardButton`、`buttonLabel`、`buttonAction`、`actionOpenCard`、`actionOpenWorkflow`、`actionSendText`、`actionSendMedia`、`actionOpenUrl`、`actionCopyText`、`actionPlaceholder`、`targetCard`、`newCard`、`pickExistingCard`、`workflowList`、`workflowMode`、`backTo`、`cardList`、`saveValidation`、`errLabel`、`errCard`、`errWorkflow`、`errUrl`、`errMedia`

- [ ] **Step 1: 写失败测试（contract）**

`menu-editor.contract.test.ts`：

```ts
const NEW_KEYS = ['mainKeyboard','columnsPerRow','addMenuItem','menuItemLabel','menuItemAction','cardName','cardMedia','cardText','cardButtons','addCardButton','buttonLabel','buttonAction','actionOpenCard','actionOpenWorkflow','actionSendText','actionSendMedia','actionOpenUrl','actionCopyText','actionPlaceholder','targetCard','newCard','pickExistingCard','workflowList','backTo','cardList','saveValidation','errLabel','errCard','errWorkflow','errUrl','errMedia'] as const

describe('menu editor i18n', () => {
  it('zh and en define every key', () => {
    const zh = JSON.parse(readFileSync(ZH, 'utf8')) as { menu: Record<string, string> }
    const en = JSON.parse(readFileSync(EN, 'utf8')) as { menu: Record<string, string> }
    for (const key of NEW_KEYS) {
      expect(zh.menu[key], `zh missing menu.${key}`).toBeTruthy()
      expect(en.menu[key], `en missing menu.${key}`).toBeTruthy()
    }
  })
})
```

- [ ] **Step 2: 运行确认失败 → Step 3: 添加文案 → Step 4: 通过**

（流程同前几轮：先跑失败，再补 zh/en `menu` 段，再跑通过。）

- [ ] **Step 5: 提交**

```bash
cd /Users/mr9esx/Documents/Pixoma
git add web/admin/vitest.config.ts web/admin/src/features/menu/menu-editor.contract.test.ts
git commit -m "test(admin): menu editor i18n contract"
```

> locale 文件混有你在途改动时不提交，结尾统一归档。

---

### Task F3: API client（menu / cards）

**Files:**
- Modify: `web/admin/src/lib/api/channel-menu.ts`（替换为 `Menu` / `Card` 类型与函数）
- Modify: `web/admin/src/lib/api/channel-menu.test.ts`

**Interfaces:**
- Produces:
  - `export type Action / MenuItem / Menu / CardButton / Card`（与 F1 同构）
  - `export function getMenu(channelId): Promise<Menu>`
  - `export function putMenu(channelId, menu: Menu): Promise<Menu>`
  - `export function listCards(channelId, q?): Promise<Card[]>`
  - `export function createCard(channelId, card: Card): Promise<Card>`
  - `export function updateCard(channelId, card: Card): Promise<Card>`
  - `export function deleteCard(channelId, cardId): Promise<void>`
  - `export function getCardReferences(channelId, cardId): Promise<string[]>`

- [ ] **Step 1: 写失败测试**

```ts
it('getMenu GETs the channel menu', async () => {
  const fetchMock = vi.fn().mockResolvedValue(
    new Response(
      JSON.stringify({ id: 'm', name: '主', columns: 2, items: [] }),
      { status: 200, headers: { 'Content-Type': 'application/json' } },
    ),
  )
  vi.stubGlobal('fetch', fetchMock)
  const menu = await getMenu('ch1')
  expect(menu.name).toBe('主')
  expect(fetchMock).toHaveBeenCalledWith(
    'http://127.0.0.1:8081/api/v1/channels/ch1/menu',
    expect.anything(),
  )
})

it('deleteCard sends DELETE', async () => {
  const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }))
  vi.stubGlobal('fetch', fetchMock)
  await deleteCard('ch1', 'c1')
  expect(fetchMock).toHaveBeenCalledWith(
    'http://127.0.0.1:8081/api/v1/channels/ch1/cards/c1',
    expect.objectContaining({ method: 'DELETE' }),
  )
})
```

- [ ] **Step 2: 运行确认失败 → Step 3: 实现 client → Step 4: 通过 + 提交**

```bash
cd /Users/mr9esx/Documents/Pixoma
git add web/admin/src/lib/api/channel-menu.ts web/admin/src/lib/api/channel-menu.test.ts
git commit -m "feat(admin): menu and cards api client"
```

---

### Task F4: ActionForm（动作参数表单）

**Files:**
- Create: `web/admin/src/features/menu/action-form.tsx`
- Modify: `web/admin/src/features/menu/menu-editor.contract.test.ts`（追加断言）

**Interfaces:**
- Produces:
  - `export function ActionForm(props: { action: Action; cards: Card[]; workflows: { id: string; name: string }[]; onChange: (next: Action) => void; disabled?: boolean }): React.JSX.Element`
  - 类型选择（下拉）+ 按类型渲染参数：`open_card` → CardPicker（新建/选已有）；`open_workflow` → 工作流多选 + 模式；`send_text`/`copy_text` → Textarea；`send_media` → 媒体 URL 列表；`open_url` → URL 输入；`placeholder` → 无
  - `data-testid='action-form'`

- [ ] **Step 1: 追加失败测试（contract）**

```ts
const ACTION_FORM = join(here, 'action-form.tsx')

it('action form offers all tg-native action types', () => {
  const source = readFileSync(ACTION_FORM, 'utf8')
  expect(source).toContain("data-testid='action-form'")
  for (const key of ['open_card','open_workflow','send_text','send_media','open_url','copy_text','placeholder']) {
    expect(source).toContain(`'${key}'`)
  }
})
```

- [ ] **Step 2: 运行确认失败 → Step 3: 实现（下拉 + 条件参数表单）**

```tsx
export function ActionForm({ action, cards, workflows, onChange, disabled }: Props) {
  const { t } = useTranslation()
  const types: ActionType[] = ['open_card','open_workflow','send_text','send_media','open_url','copy_text','placeholder']
  return (
    <div data-testid='action-form' className='space-y-3'>
      <Select value={action.type} onValueChange={(type) => onChange({ type: type as ActionType })} disabled={disabled}>
        <SelectTrigger><SelectValue /></SelectTrigger>
        <SelectContent>
          {types.map((type) => (
            <SelectItem key={type} value={type}>{t(`menu.action${cap(type)}`)}</SelectItem>
          ))}
        </SelectContent>
      </Select>
      {action.type === 'open_card' ? (
        <CardPicker cards={cards} value={action.card_id} onPick={(cardId) => onChange({ ...action, card_id: cardId })} onCreateNew={() => { /* 由父级就地新建 */ }} />
      ) : null}
      {action.type === 'open_workflow' ? (
        <div className='max-h-44 space-y-1.5 overflow-auto rounded-md border p-2'>
          {workflows.map((w) => (
            <label key={w.id} className='flex items-center gap-2 text-sm'>
              <Checkbox
                checked={(action.workflow_ids ?? []).includes(w.id)}
                onCheckedChange={(v) => onChange({ ...action, workflow_ids: v ? [...(action.workflow_ids ?? []), w.id] : (action.workflow_ids ?? []).filter((id) => id !== w.id) })}
                disabled={disabled}
              />
              <span>{w.name}</span>
            </label>
          ))}
        </div>
      ) : null}
      {action.type === 'send_text' || action.type === 'copy_text' ? (
        <Textarea value={action.text ?? ''} onChange={(e) => onChange({ ...action, text: e.target.value })} rows={3} disabled={disabled} />
      ) : null}
      {action.type === 'send_media' ? (
        <Textarea
          value={(action.media ?? []).map((m) => m.url).join('\n')}
          onChange={(e) => onChange({ ...action, media: e.target.value.split('\n').map((url) => ({ kind: 'image', url: url.trim() })).filter((m) => m.url) })}
          rows={3}
          disabled={disabled}
        />
      ) : null}
      {action.type === 'open_url' ? (
        <Input value={action.url ?? ''} onChange={(e) => onChange({ ...action, url: e.target.value })} disabled={disabled} />
      ) : null}
    </div>
  )
}
```

（`cap` 为「首字母大写」小工具，或直接写 switch 映射 i18n key。）

- [ ] **Step 4: 通过 + 提交**

```bash
cd /Users/mr9esx/Documents/Pixoma
git add web/admin/src/features/menu/action-form.tsx web/admin/src/features/menu/menu-editor.contract.test.ts
git commit -m "feat(admin): action param forms"
```

---

### Task F5: PhoneSimulation（完整手机模拟）

**Files:**
- Create: `web/admin/src/features/menu/phone-simulation.tsx`
- Modify: `web/admin/src/features/menu/menu-editor.contract.test.ts`

**Interfaces:**
- Produces:
  - `export function PhoneSimulation(props: { menu: Menu; cards: Card[]; path: string[]; onOpenCard: (cardId: string) => void; onBack: () => void; onSelectMenuItem: (item: MenuItem) => void }): React.JSX.Element`
  - 渲染主键盘（columns 自由）+ 当前路径的卡片流（媒体/文字/按钮 + 自动「‹ 返回」）；点击卡片按钮触发 `onOpenCard`；点击返回触发 `onBack`
  - `data-testid='phone-simulation'`

- [ ] **Step 1: 追加失败测试（contract）**：断言含 `buildTgFlow`、`onOpenCard`、`onBack`、`data-testid='phone-simulation'`
- [ ] **Step 2: 失败 → Step 3: 实现 → Step 4: 通过 + 提交**

```bash
cd /Users/mr9esx/Documents/Pixoma
git add web/admin/src/features/menu/phone-simulation.tsx web/admin/src/features/menu/menu-editor.contract.test.ts
git commit -m "feat(admin): phone simulation for menu/card flows"
```

---

### Task F6: 分层编辑器（MenuCardEditor + CardPicker + CardListPanel）

**Files:**
- Create: `web/admin/src/features/menu/menu-card-editor.tsx`
- Create: `web/admin/src/features/menu/card-picker.tsx`
- Create: `web/admin/src/features/menu/card-list-panel.tsx`
- Modify: `web/admin/src/routes/_app/channels/$id.tsx`（菜单 Tab 改用新编辑器）
- Modify: `web/admin/src/features/menu/menu-editor.contract.test.ts`

**Interfaces:**
- Produces:
  - `export function MenuCardEditor(props: { channelId: string }): React.JSX.Element`——数据：`menu` 草稿 + `cards` 草稿；分层渲染：⓪ 菜单（列数 + 菜单项列表）→ ① 选中菜单项（label + ActionForm）→ ② 目标卡片（媒体/文字/按钮，就地）→ ③ 按钮（label + ActionForm）→ ④ 目标卡片……（递归）；面包屑 `path: string[]` 可点击回跳；`data-testid='menu-card-editor'`
  - `export function CardPicker(props: { cards: Card[]; value?: string; onPick: (cardId: string) => void; onCreateNew: () => void }): React.JSX.Element`——已有卡片下拉 + 「新建一张（就地编辑）」；`data-testid='card-picker'`
  - `export function CardListPanel(props: { cards: Card[]; onSelect: (card: Card) => void; onDelete: (card: Card) => void }): React.JSX.Element`——搜索 + 列表 + 引用删除提示；`data-testid='card-list-panel'`
  - 保存：`putMenu(channelId, menu)` + 全量 `cards` 落库；校验只在保存时（`validateMenuConfig`），错误红条定位 path

- [ ] **Step 1: 追加失败测试（contract）**

```ts
it('editor renders layered chain and card picker', () => {
  const source = readFileSync(join(here, 'menu-card-editor.tsx'), 'utf8')
  expect(source).toContain("data-testid='menu-card-editor'")
  expect(source).toContain('ActionForm')
  expect(source).toContain('CardPicker')
  expect(source).toContain('buildTgFlow')
  expect(source).toContain('validateMenuConfig')
  expect(source).not.toContain('capability_id')
})
```

- [ ] **Step 2: 失败 → Step 3: 实现三个组件 + 路由接入 → Step 4: tsc/eslint/测试通过 → Step 5: 提交**

```bash
cd /Users/mr9esx/Documents/Pixoma
git add web/admin/src/features/menu web/admin/src/routes/_app/channels/\$id.tsx web/admin/src/features/menu/menu-editor.contract.test.ts
git commit -m "feat(admin): layered menu/card editor"
```

---

### Task F7: 保存校验接入 + 全量验证

**Files:**
- Modify: `web/admin/src/features/menu/menu-card-editor.tsx`（校验红条 + 保存拦截 + 未保存计数）

- [ ] **Step 1: 接入**：`validation = useMemo(() => validateMenuConfig(menu, cards), [menu, cards])`；保存按钮 onClick 前 `if (!validation.ok) return`；红条显示首条错误（`t('menu.errLabel')` 等按 message 映射）；`dirtyCount` 对比查询基线。
- [ ] **Step 2: 全量检查**

```bash
cd /Users/mr9esx/Documents/Pixoma/web/admin
pnpm test
pnpm exec tsc -b
pnpm exec eslint src/features/menu src/features/channels
pnpm exec prettier --check src/features/menu
```

- [ ] **Step 3: 提交遗留改动**

```bash
cd /Users/mr9esx/Documents/Pixoma
git add -A web/admin
git commit -m "chore(admin): finalize menu/card editor"
```

---

### Task F8: 删除旧编辑器实现

**Files:**
- Delete: `web/admin/src/features/channels/channel-menu-editor.tsx`、`phone-simulation.tsx`、`node-config-panel.tsx`、`params-form.tsx`、`lib/menu-simulation.ts`、`lib/menu-validate.ts`、`menu-preview.tsx`
- Modify: `web/admin/src/routes/_app/channels/$id.tsx`（预览 Tab 移除或指向手机模拟）
- Modify: `web/admin/vitest.config.ts`（移除旧测试条目）、`web/admin/src/features/channels/channel-menu-editor.contract.test.ts`（删除）

- [ ] **Step 1: 验证旧文件无引用**

```bash
cd web/admin && rg -l "channel-menu-editor|phone-simulation|node-config-panel|params-form|menu-simulation|menu-validate|ChannelMenuPreview" src | grep -v "features/menu"
```

- [ ] **Step 2: 删除文件 + 清理配置 + 全量检查**（`pnpm test` / `tsc`）
- [ ] **Step 3: 提交**

```bash
cd /Users/mr9esx/Documents/Pixoma
git add -A web/admin
git commit -m "refactor(admin): remove legacy menu editor"
```
