# TG 菜单编辑器前端 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把消息平台菜单编辑器改成所见即所得：左侧 TG 手机模拟（实时、可点击），右侧按能力参数 schema 渲染配置表单，保存前校验并显示未保存提示。

**Architecture:** 纯函数层（`menu-simulation.ts` 复刻后端布局逻辑、`menu-validate.ts` 保存校验）与组件层分离；编辑器状态仍持有 `MenuNode[]` 树，PUT 全量保存不变。能力参数表单由 `params_schema`（含 `x-admin.widget` 提示）驱动，open_case 不再特判。

**Tech Stack:** React 19 + TypeScript + Vite + TanStack Router/Query + Tailwind v4 + radix-ui（Select/Dialog/Switch）+ vitest（contract test 读源码断言）。

## Global Constraints

- 测试命令：`cd web/admin && pnpm test`；类型 `pnpm exec tsc -b`；格式 `pnpm exec prettier --write <files>`；lint `pnpm exec eslint <files>`。
- 新测试文件必须加入 `web/admin/vitest.config.ts` 的 include 列表（现有约定是显式列出）。
- 用户文案禁用内部术语（capability/params/schema/placeholder 等），统一「功能/参数/按钮/说明」等大白话。
- 所有文案走 i18n，zh/en 成对。
- 运行时行为与后端 `BuildKeyboardLayout` 保持一致（含「第一个根按钮的 columns 决定整键盘列数」这一既有语义）。
- 编辑器不再使用 `placeholder_text` / `reply` 字段配置（后端读取时已转换为能力）；`MenuNode` 类型字段保留以兼容旧响应。

---

### Task F1: 编辑器 i18n 文案

**Files:**
- Modify: `web/admin/src/features/channels/channel-menu-editor.contract.test.ts`（追加 key 断言）
- Modify: `web/admin/src/lib/i18n/locales/zh.json`、`en.json`

**Interfaces:**
- Produces: `channelMenu.*` 新增 keys（`userView`、`currentEditing`、`buttonLabel`、`buttonAction`、`actionShowChildren`、`actionOpenWorkflow`、`actionReplyText`、`actionReplyMedia`、`paramSelectWorkflows`、`paramReplyText`、`paramReplyImages`、`mainKeyboard`、`columnsPerRow`、`maxRootHint`、`oneLevelHint`、`unsavedCount`、`invalidMainKeyboard`、`invalidMissingAction`、`invalidOpenCaseEmpty`、`invalidNameEmpty`、`simNote` 等）

- [ ] **Step 1: 写失败测试**

在 `channel-menu-editor.contract.test.ts` 追加：

```ts
const NEW_KEYS = [
  'userView',
  'currentEditing',
  'buttonLabel',
  'buttonAction',
  'actionShowChildren',
  'actionOpenWorkflow',
  'actionReplyText',
  'actionReplyMedia',
  'paramSelectWorkflows',
  'paramReplyText',
  'paramReplyImages',
  'mainKeyboard',
  'columnsPerRow',
  'maxRootHint',
  'oneLevelHint',
  'unsavedCount',
  'invalidMainKeyboard',
  'invalidMissingAction',
  'invalidOpenCaseEmpty',
  'invalidNameEmpty',
  'simNote',
] as const

describe('wysiwyg menu editor i18n', () => {
  it('zh and en define every editor key', () => {
    const zh = JSON.parse(read(ZH)) as { channelMenu: Record<string, string> }
    const en = JSON.parse(read(EN)) as { channelMenu: Record<string, string> }
    for (const key of NEW_KEYS) {
      expect(zh.channelMenu[key], `zh missing channelMenu.${key}`).toBeTruthy()
      expect(en.channelMenu[key], `en missing channelMenu.${key}`).toBeTruthy()
    }
  })
})
```

- [ ] **Step 2: 运行确认失败**

Run: `cd web/admin && pnpm test src/features/channels/channel-menu-editor.contract.test.ts`
Expected: FAIL（key 缺失）。

- [ ] **Step 3: 添加文案**

zh.json `channelMenu` 段追加：

```json
    "userView": "用户视角（Telegram）",
    "currentEditing": "当前编辑",
    "buttonLabel": "按钮上显示的名字",
    "buttonAction": "这个按钮做什么",
    "actionShowChildren": "显示子按钮",
    "actionOpenWorkflow": "打开工作流",
    "actionReplyText": "提示文字",
    "actionReplyMedia": "回复图文",
    "paramSelectWorkflows": "选择工作流",
    "paramReplyText": "回复文字",
    "paramReplyImages": "图片链接（每行一个）",
    "mainKeyboard": "主键盘整体",
    "columnsPerRow": "每行按钮数",
    "maxRootHint": "主键盘建议不超过 6 个按钮",
    "oneLevelHint": "模拟里只支持一层子按钮（Telegram 限制）",
    "unsavedCount": "未保存：{{count}} 处",
    "invalidMainKeyboard": "主键盘按钮不能超过 6 个",
    "invalidMissingAction": "这个按钮还没选功能",
    "invalidOpenCaseEmpty": "至少选择一个工作流",
    "invalidNameEmpty": "按钮名不能为空",
    "simNote": "模拟到这一步，真实流程以保存后为准"
```

en.json 对应英文（`userView`=“User view (Telegram)”、`buttonAction`=“What does this button do?”、`actionOpenWorkflow`=“Open workflow”、`unsavedCount`=“{{count}} unsaved changes” 等）。

- [ ] **Step 4: 运行确认通过**

Run: `cd web/admin && pnpm test src/features/channels/channel-menu-editor.contract.test.ts`
Expected: PASS。

- [ ] **Step 5: 格式化并提交**

```bash
cd web/admin
pnpm exec prettier --write src/features/channels/channel-menu-editor.contract.test.ts src/lib/i18n/locales/zh.json src/lib/i18n/locales/en.json
cd /Users/mr9esx/Documents/Pixoma
git add web/admin/src/features/channels/channel-menu-editor.contract.test.ts
git commit -m "test(admin): wysiwyg menu editor i18n contract"
```

> locale 文件若混有你在途改动，与上次一样不提交，只提交测试；结尾统一归档。

---

### Task F2: capabilities API 类型 + params_schema

**Files:**
- Modify: `web/admin/src/lib/api/channels.ts`（`CapabilityBrief`）
- Modify: `web/admin/src/lib/api/channels.test.ts` 或 `channel-menu.test.ts`

**Interfaces:**
- Produces: `CapabilityBrief` 增加 `params_schema: Record<string, unknown>`；`listCapabilities` 返回结构不变

- [ ] **Step 1: 写失败测试**

在 `web/admin/src/lib/api/channel-menu.test.ts` 或 `channels.test.ts` 追加：

```ts
it('capabilities carry params_schema', async () => {
  const url = 'http://127.0.0.1:8081/api/v1/channels/capabilities'
  mockFetchOnce(url, [
    {
      id: 'open_case',
      display_name: '打开工作流',
      params_schema: {
        type: 'object',
        properties: {
          case_ids: { type: 'array', items: { type: 'string' } },
        },
      },
    },
  ])
  const caps = await listCapabilities()
  expect(caps[0].params_schema.properties.case_ids.type).toBe('array')
})
```

（按现有 `channels.test.ts` 的 mock 方式照抄。）

- [ ] **Step 2: 运行确认失败**

Run: `cd web/admin && pnpm test src/lib/api/`
Expected: 编译失败（`params_schema` 不存在）。

- [ ] **Step 3: 实现**

```ts
export type CapabilityBrief = {
  id: string
  display_name: string
  params_schema: Record<string, unknown>
}
```

- [ ] **Step 4: 运行确认通过 + 提交**

```bash
cd web/admin && pnpm test src/lib/api/ && pnpm exec tsc -b
cd /Users/mr9esx/Documents/Pixoma
git add web/admin/src/lib/api/channels.ts web/admin/src/lib/api/channels.test.ts
git commit -m "feat(admin): capabilities expose params schema"
```

---

### Task F3: menu-simulation 纯函数（TG 模拟）

**Files:**
- Create: `web/admin/src/features/channels/lib/menu-simulation.ts`
- Create: `web/admin/src/features/channels/lib/menu-simulation.test.ts`
- Modify: `web/admin/vitest.config.ts`（include 追加测试文件）

**Interfaces:**
- Consumes: `MenuNode`（`@/lib/api/channel-menu`）
- Produces:
  - `export type SimButton = { label: string; kind: 'capability' | 'group'; capabilityId?: string }`
  - `export type SimGroup = { id: string; title: string; intro: string; buttons: SimButton[] }`
  - `export type TgSimulation = { keyboard: string[][]; groups: Record<string, SimGroup> }`
  - `export function rootColumnsOf(items: MenuNode[]): number`（第一个启用根项的 `render_override.columns`，1-8，默认 2）
  - `export function buildTgSimulation(items: MenuNode[]): TgSimulation`

- [ ] **Step 1: 写失败测试**

```ts
import { describe, expect, it } from 'vitest'
import type { MenuNode } from '@/lib/api/channel-menu'
import { buildTgSimulation, rootColumnsOf } from './menu-simulation'

function node(partial: Partial<MenuNode>): MenuNode {
  return { id: 'x', label: 'x', order: 0, enabled: true, ...partial }
}

describe('menu simulation', () => {
  it('builds keyboard rows from enabled root items with columns', () => {
    const items: MenuNode[] = [
      node({ id: 'a', label: 'A', order: 0, render_override: { columns: 2 } }),
      node({ id: 'b', label: 'B', order: 1 }),
      node({ id: 'c', label: 'C', order: 2 }),
      node({ id: 'off', label: 'OFF', order: 3, enabled: false }),
    ]
    expect(rootColumnsOf(items)).toBe(2)
    expect(buildTgSimulation(items).keyboard).toEqual([
      ['A', 'B'],
      ['C'],
    ])
  })

  it('renders one-level groups with intro and child buttons', () => {
    const items: MenuNode[] = [
      node({
        id: 'tools',
        label: '工具',
        order: 0,
        intro_text: '小工具们',
        children: [
          node({ id: 'edit', label: 'AI 修图', order: 0, capability_id: 'open_case', params: { case_ids: ['c1'] } }),
          node({ id: 'matting', label: '抠图', order: 1, capability_id: 'open_case', params: { case_ids: ['c2'] } }),
        ],
      }),
    ]
    const sim = buildTgSimulation(items)
    expect(sim.groups['tools']).toEqual({
      id: 'tools',
      title: '工具',
      intro: '小工具们',
      buttons: [
        { label: 'AI 修图', kind: 'capability', capabilityId: 'open_case' },
        { label: '抠图', kind: 'capability', capabilityId: 'open_case' },
      ],
    })
  })

  it('filters disabled children from group buttons', () => {
    const items: MenuNode[] = [
      node({
        id: 'g',
        label: 'G',
        order: 0,
        children: [
          node({ id: 'on', label: 'ON', order: 0, capability_id: 'reply_text', params: { text: 'x' } }),
          node({ id: 'off', label: 'OFF', order: 1, enabled: false, capability_id: 'reply_text', params: { text: 'x' } }),
        ],
      }),
    ]
    expect(buildTgSimulation(items).groups['g'].buttons.map((b) => b.label)).toEqual(['ON'])
  })
})
```

- [ ] **Step 2: 运行确认失败**

Run: `cd web/admin && pnpm test src/features/channels/lib/menu-simulation.test.ts`
Expected: FAIL（模块不存在）。

- [ ] **Step 3: 实现 menu-simulation.ts**

```ts
import type { MenuNode } from '@/lib/api/channel-menu'

export type SimButton = {
  label: string
  kind: 'capability' | 'group'
  capabilityId?: string
}

export type SimGroup = {
  id: string
  title: string
  intro: string
  buttons: SimButton[]
}

export type TgSimulation = {
  keyboard: string[][]
  groups: Record<string, SimGroup>
}

export function rootColumnsOf(items: MenuNode[]): number {
  for (const it of items) {
    if (!it.enabled) continue
    const v = it.render_override?.columns
    if (typeof v === 'number' && v >= 1 && v <= 8) return Math.trunc(v)
  }
  return 2
}

export function buildTgSimulation(items: MenuNode[]): TgSimulation {
  const enabledRoots = items.filter((it) => it.enabled)
  const cols = rootColumnsOf(items)
  const keyboard: string[][] = []
  let row: string[] = []
  for (const it of enabledRoots) {
    row.push(it.label)
    if (row.length >= cols) {
      keyboard.push(row)
      row = []
    }
  }
  if (row.length > 0) keyboard.push(row)

  const groups: Record<string, SimGroup> = {}
  const walk = (nodes: MenuNode[]) => {
    for (const n of nodes) {
      if (n.children?.length) {
        groups[n.id] = {
          id: n.id,
          title: n.label,
          intro: n.intro_text ?? n.label,
          buttons: n.children
            .filter((c) => c.enabled)
            .map((c) => ({
              label: c.label,
              kind: c.capability_id ? 'capability' : 'group',
              capabilityId: c.capability_id,
            })),
        }
        walk(n.children)
      }
    }
  }
  walk(items)
  return { keyboard, groups }
}
```

- [ ] **Step 4: 运行确认通过 + 提交**

```bash
cd web/admin && pnpm test src/features/channels/lib/menu-simulation.test.ts && pnpm exec tsc -b
pnpm exec prettier --write src/features/channels/lib
cd /Users/mr9esx/Documents/Pixoma
git add web/admin/vitest.config.ts web/admin/src/features/channels/lib
git commit -m "feat(admin): tg menu simulation pure functions"
```

---

### Task F4: 能力参数表单（schema → 控件）

**Files:**
- Create: `web/admin/src/features/channels/params-form.tsx`
- Modify: `web/admin/src/features/channels/channel-menu-editor.contract.test.ts`（追加断言）

**Interfaces:**
- Consumes: `CapabilityBrief.params_schema`、`CaseRecord`（工作流选择）
- Produces:
  - `export function ParamsForm(props: { schema: Record<string, unknown>; value: Record<string, unknown>; onChange: (next: Record<string, unknown>) => void; caseOptions?: { id: string; name: string }[]; disabled?: boolean }): React.JSX.Element`
  - 控件映射：`string` → Input；`x-admin.widget === 'text'` → Textarea；`number` → Input number；`boolean` → Switch；`array of string` 且 `x-admin.widget === 'workflow_picker'` → 工作流多选（checkbox 列表，`caseOptions` 提供）；其它数组 → 每行一个的 Textarea
  - `data-testid='capability-params-form'`

- [ ] **Step 1: 追加失败测试（contract）**

```ts
const PARAMS_FORM = join(here, 'params-form.tsx')

describe('params form', () => {
  it('is schema driven with workflow picker widget', () => {
    const source = read(PARAMS_FORM)
    expect(source).toContain("data-testid='capability-params-form'")
    expect(source).toContain('workflow_picker')
    expect(source).toContain('x-admin')
    expect(source).toContain('caseOptions')
  })
})
```

- [ ] **Step 2: 运行确认失败**

Run: `cd web/admin && pnpm test src/features/channels/channel-menu-editor.contract.test.ts`
Expected: FAIL。

- [ ] **Step 3: 实现 params-form.tsx**

```tsx
import { useTranslation } from 'react-i18next'
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'

type Props = {
  schema: Record<string, unknown>
  value: Record<string, unknown>
  onChange: (next: Record<string, unknown>) => void
  caseOptions?: { id: string; name: string }[]
  disabled?: boolean
}

function xAdminWidget(schema: Record<string, unknown>): string | undefined {
  const ext = schema['x-admin']
  if (ext && typeof ext === 'object') {
    const widget = (ext as Record<string, unknown>)['widget']
    if (typeof widget === 'string') return widget
  }
  return undefined
}

export function ParamsForm({ schema, value, onChange, caseOptions, disabled }: Props) {
  const { t } = useTranslation()
  const properties = (schema['properties'] ?? {}) as Record<string, Record<string, unknown>>
  const required = Array.isArray(schema['required']) ? (schema['required'] as string[]) : []

  function patch(key: string, next: unknown) {
    onChange({ ...value, [key]: next })
  }

  return (
    <div data-testid='capability-params-form' className='space-y-3'>
      {Object.entries(properties).map(([key, propSchema]) => {
        const widget = xAdminWidget(propSchema)
        const label = key
        const isRequired = required.includes(key)
        const current = value[key]

        if (widget === 'workflow_picker') {
          const selected = Array.isArray(current) ? (current as string[]) : []
          return (
            <div key={key} className='space-y-1.5'>
              <Label>{t('channelMenu.paramSelectWorkflows')}</Label>
              <div className='max-h-48 space-y-1.5 overflow-auto rounded-md border p-2'>
                {(caseOptions ?? []).map((c) => (
                  <label key={c.id} className='flex items-center gap-2 text-sm'>
                    <Checkbox
                      checked={selected.includes(c.id)}
                      onCheckedChange={(v) =>
                        patch(
                          key,
                          v === true
                            ? [...selected, c.id]
                            : selected.filter((id) => id !== c.id),
                        )
                      }
                      disabled={disabled}
                    />
                    <span>{c.name}</span>
                  </label>
                ))}
              </div>
            </div>
          )
        }

        if (widget === 'text') {
          return (
            <div key={key} className='space-y-1.5'>
              <Label>
                {label}
                {isRequired ? ' *' : ''}
              </Label>
              <Textarea
                value={typeof current === 'string' ? current : ''}
                onChange={(e) => patch(key, e.target.value)}
                disabled={disabled}
                rows={3}
              />
            </div>
          )
        }

        if (propSchema['type'] === 'number') {
          return (
            <div key={key} className='space-y-1.5'>
              <Label>{label}</Label>
              <Input
                type='number'
                value={typeof current === 'number' ? current : ''}
                onChange={(e) => patch(key, Number(e.target.value) || 0)}
                disabled={disabled}
              />
            </div>
          )
        }

        if (propSchema['type'] === 'boolean') {
          return (
            <div key={key} className='flex items-center justify-between gap-3 rounded-md border px-3 py-2'>
              <Label>{label}</Label>
              <Switch
                checked={current === true}
                onCheckedChange={(v) => patch(key, v)}
                disabled={disabled}
              />
            </div>
          )
        }

        if (propSchema['type'] === 'array') {
          const items = Array.isArray(current) ? (current as string[]) : []
          return (
            <div key={key} className='space-y-1.5'>
              <Label>{label}</Label>
              <Textarea
                value={items.join('\n')}
                onChange={(e) =>
                  patch(
                    key,
                    e.target.value.split('\n').map((s) => s.trim()).filter(Boolean),
                  )
                }
                disabled={disabled}
                rows={3}
              />
            </div>
          )
        }

        return (
          <div key={key} className='space-y-1.5'>
            <Label>{label}</Label>
            <Input
              value={typeof current === 'string' ? current : ''}
              onChange={(e) => patch(key, e.target.value)}
              disabled={disabled}
            />
          </div>
        )
      })}
    </div>
  )
}
```

- [ ] **Step 4: 运行确认通过 + 提交**

```bash
cd web/admin && pnpm test src/features/channels/channel-menu-editor.contract.test.ts && pnpm exec tsc -b
pnpm exec prettier --write src/features/channels/params-form.tsx
cd /Users/mr9esx/Documents/Pixoma
git add web/admin/src/features/channels/params-form.tsx web/admin/src/features/channels/channel-menu-editor.contract.test.ts
git commit -m "feat(admin): schema driven capability params form"
```

---

### Task F5: 编辑器双栏重构（手机模拟 + 配置面板）

**Files:**
- Modify: `web/admin/src/features/channels/channel-menu-editor.tsx`（重构渲染）
- Create: `web/admin/src/features/channels/phone-simulation.tsx`
- Create: `web/admin/src/features/channels/node-config-panel.tsx`
- Modify: `web/admin/src/features/channels/channel-menu-editor.contract.test.ts`（重写断言）

**Interfaces:**
- Consumes: `buildTgSimulation` / `ParamsForm` / `CapabilityBrief` / 树操作函数（保留 `findNode` / `updateNodeInTree` / `removeNodeFromTree` / `addChildToTree`）
- Produces:
  - `export function PhoneSimulation(props: { items: MenuNode[]; selectedId: string | null; onSelect: (id: string) => void }): React.JSX.Element`——渲染键盘 + 选中分组/能力的第一步消息；`data-testid='phone-simulation'`
  - `export function NodeConfigPanel(props: { node: MenuNode; isRoot: boolean; capabilities: CapabilityBrief[]; caseOptions: { id: string; name: string }[]; onUpdate: (patch: Partial<MenuNode>) => void; onAddChild: () => void; onRemove: () => void }): React.JSX.Element`——按钮名、功能选择（分组特殊项 + 能力列表）、schema 参数表单、本层说明、子按钮列表、主键盘列数（仅根）、高级折叠；`data-testid='node-config-panel'`

- [ ] **Step 1: 重写失败测试（contract）**

替换 `channel-menu-editor.contract.test.ts` 的 `editor uses expandable tree...` 测试为：

```ts
it('editor is wysiwyg: phone simulation plus config panel', () => {
  const source = read(EDITOR)
  expect(source).toContain('PhoneSimulation')
  expect(source).toContain('NodeConfigPanel')
  expect(source).toContain('buildTgSimulation')
  expect(source).toContain('params_schema')
  expect(source).not.toContain('visibleTreeRows')
  expect(source).not.toContain('expandedIds')
})

it('phone simulation renders keyboard and group message', () => {
  const source = read(join(here, 'phone-simulation.tsx'))
  expect(source).toContain("data-testid='phone-simulation'")
  expect(source).toContain('buildTgSimulation')
  expect(source).toContain('onSelect')
})

it('config panel offers group and capability actions with schema form', () => {
  const source = read(join(here, 'node-config-panel.tsx'))
  expect(source).toContain("data-testid='node-config-panel'")
  expect(source).toContain('buttonAction')
  expect(source).toContain('ParamsForm')
  expect(source).toContain('actionShowChildren')
})
```

- [ ] **Step 2: 运行确认失败**

Run: `cd web/admin && pnpm test src/features/channels/channel-menu-editor.contract.test.ts`
Expected: FAIL。

- [ ] **Step 3: 实现 phone-simulation.tsx**

```tsx
import { useTranslation } from 'react-i18next'
import type { MenuNode } from '@/lib/api/channel-menu'
import { buildTgSimulation } from './lib/menu-simulation'
import { cn } from '@/lib/utils'

type Props = {
  items: MenuNode[]
  selectedId: string | null
  onSelect: (id: string) => void
}

export function PhoneSimulation({ items, selectedId, onSelect }: Props) {
  const { t } = useTranslation()
  const sim = buildTgSimulation(items)
  const selectedGroup = selectedId ? sim.groups[selectedId] : undefined

  return (
    <div data-testid='phone-simulation' className='flex flex-col gap-3'>
      <div className='rounded-xl border bg-background p-3'>
        <div className='space-y-2 text-sm'>
          {selectedGroup ? (
            <>
              <p className='font-medium'>{selectedGroup.title}</p>
              <p className='text-muted-foreground'>{selectedGroup.intro}</p>
              <div className='flex flex-wrap gap-2 pt-1'>
                {selectedGroup.buttons.map((b) => (
                  <span
                    key={b.label}
                    className='rounded-md border border-cyan-500/40 bg-cyan-500/10 px-3 py-1.5 text-xs'
                  >
                    {b.label}
                  </span>
                ))}
              </div>
              <p className='pt-1 text-xs text-muted-foreground'>‹ 返回主菜单</p>
            </>
          ) : (
            <p className='text-muted-foreground'>欢迎使用，请选择功能 👇</p>
          )}
        </div>
        <div className='mt-3 border-t pt-3'>
          {sim.keyboard.map((row, i) => (
            <div key={i} className='mb-2 flex gap-2 last:mb-0'>
              {row.map((label) => {
                const item = items.find((it) => it.label === label)
                return (
                  <button
                    key={label}
                    type='button'
                    onClick={() => item && onSelect(item.id)}
                    className={cn(
                      'flex-1 rounded-md border px-2 py-2 text-sm',
                      item?.id === selectedId
                        ? 'border-primary bg-primary/10 font-medium'
                        : 'bg-muted/40 hover:bg-muted',
                    )}
                  >
                    {label}
                  </button>
                )
              })}
            </div>
          ))}
        </div>
      </div>
      <p className='text-xs text-muted-foreground'>
        {t('channelMenu.oneLevelHint')} · {t('channelMenu.simNote')}
      </p>
    </div>
  )
}
```

- [ ] **Step 4: 实现 node-config-panel.tsx（核心）**

```tsx
import { useTranslation } from 'react-i18next'
import type { MenuNode } from '@/lib/api/channel-menu'
import type { CapabilityBrief } from '@/lib/api/channels'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { ParamsForm } from './params-form'

type Props = {
  node: MenuNode
  isRoot: boolean
  capabilities: CapabilityBrief[]
  caseOptions: { id: string; name: string }[]
  onUpdate: (patch: Partial<MenuNode>) => void
  onAddChild: () => void
  onRemove: () => void
}

export function NodeConfigPanel({
  node,
  isRoot,
  capabilities,
  caseOptions,
  onUpdate,
  onAddChild,
  onRemove,
}: Props) {
  const { t } = useTranslation()
  const isGroup = (node.children?.length ?? 0) > 0
  const capability = capabilities.find((c) => c.id === node.capability_id)
  const actionValue = isGroup ? 'group' : (node.capability_id ?? 'none')

  function onActionChange(value: string) {
    if (value === 'group') {
      onUpdate({ capability_id: undefined, params: undefined })
    } else if (value === 'none') {
      onUpdate({ capability_id: undefined, params: undefined })
    } else {
      const cap = capabilities.find((c) => c.id === value)
      const next: Partial<MenuNode> = { capability_id: value }
      if (value === 'open_case' && !node.params?.case_ids) {
        next.params = { ...(node.params ?? {}), case_ids: [] }
      }
      next.params = cap ? { ...(node.params ?? {}), ...(cap.params_schema['x-default'] as Record<string, unknown> | undefined) } : node.params
      onUpdate(next)
    }
  }

  return (
    <div data-testid='node-config-panel' className='space-y-4'>
      <div className='flex items-start justify-between gap-3'>
        <div>
          <div className='text-xs text-muted-foreground'>{t('channelMenu.currentEditing')}</div>
          <h2 className='text-lg font-semibold'>{node.label.trim() || t('channelMenu.untitled')}</h2>
        </div>
        <Button type='button' variant='ghost' size='sm' onClick={onRemove}>
          {t('channelMenu.removeItem')}
        </Button>
      </div>

      <div className='space-y-1.5'>
        <Label>{t('channelMenu.buttonLabel')}</Label>
        <Input
          value={node.label}
          onChange={(e) => onUpdate({ label: e.target.value })}
          autoComplete='off'
        />
      </div>

      <div className='space-y-1.5'>
        <Label>{t('channelMenu.buttonAction')}</Label>
        <Select value={actionValue} onValueChange={onActionChange}>
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {!isGroup ? (
              <SelectItem value='group'>{t('channelMenu.actionShowChildren')}</SelectItem>
            ) : null}
            <SelectItem value='none'>{t('channelMenu.capabilityNone')}</SelectItem>
            {capabilities.map((c) => (
              <SelectItem key={c.id} value={c.id}>
                {c.display_name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {!isGroup && capability ? (
        <ParamsForm
          schema={capability.params_schema}
          value={(node.params as Record<string, unknown>) ?? {}}
          onChange={(params) => onUpdate({ params: { ...(node.params ?? {}), ...params } })}
          caseOptions={caseOptions}
        />
      ) : null}

      {isGroup ? (
        <div className='space-y-1.5'>
          <Label>{t('channelMenu.fieldIntro')}</Label>
          <Textarea
            value={node.intro_text ?? ''}
            onChange={(e) => onUpdate({ intro_text: e.target.value })}
            rows={3}
          />
          <div className='rounded-md border p-2'>
            {(node.children ?? []).map((child) => (
              <div key={child.id} className='px-2 py-1.5 text-sm'>
                {child.label.trim() || t('channelMenu.untitled')}
              </div>
            ))}
            <Button type='button' variant='ghost' size='sm' onClick={onAddChild}>
              {t('channelMenu.addChild')}
            </Button>
          </div>
        </div>
      ) : null}

      {isRoot ? (
        <div className='border-t pt-3'>
          <div className='space-y-1.5'>
            <Label>{t('channelMenu.mainKeyboard')}</Label>
            <Input
              type='number'
              min={1}
              max={8}
              value={(node.render_override?.columns as number | undefined) ?? 2}
              onChange={(e) =>
                onUpdate({
                  render_override: {
                    ...(node.render_override ?? {}),
                    columns: Number(e.target.value) || 2,
                  },
                })
              }
            />
            <p className='text-xs text-muted-foreground'>{t('channelMenu.columnsPerRow')}</p>
          </div>
        </div>
      ) : null}
    </div>
  )
}
```

> 说明：`actionValue` 对分组项显示「显示子按钮」；「无」= 未选功能（保存时校验拦截）。`x-default` 是 schema 里可选的默认值扩展（本任务不强制实现；`open_case` 初始化空 `case_ids` 已覆盖）。

- [ ] **Step 5: 重构 channel-menu-editor.tsx 渲染**

把 `MasterDetailShell` 列表/详情替换为双栏布局：

```tsx
<div className='grid min-h-0 flex-1 gap-3 md:grid-cols-[minmax(0,1fr)_400px]'>
  <div className='min-h-0 overflow-auto rounded-md border p-4'>
    <PhoneSimulation items={items} selectedId={selectedId} onSelect={setSelectedId} />
  </div>
  <div className='min-h-0 overflow-auto rounded-md border p-4'>
    {selected && selectedId ? (
      <NodeConfigPanel
        node={selected}
        isRoot={isRootItem}
        capabilities={capsQuery.data ?? []}
        caseOptions={casesQuery.data ?? []}
        onUpdate={(patch) => updateNode(selectedId, patch)}
        onAddChild={() => addChildItem(selectedId)}
        onRemove={removeSelected}
      />
    ) : (
      <EmptyState message={t('common.selectItem')} />
    )}
  </div>
</div>
```

删除 `visibleTreeRows` / `expandedIds` / `ancestorIds` 及展开逻辑；保留 `findNode` / `updateNodeInTree` / `removeNodeFromTree` / `addChildToTree` / `normalizeNode`。未保存计数与保存校验由 Task F6 接入（本任务不实现）。

- [ ] **Step 6: 运行确认通过**

```bash
cd web/admin
pnpm test src/features/channels/channel-menu-editor.contract.test.ts
pnpm exec tsc -b
pnpm exec eslint src/features/channels/channel-menu-editor.tsx src/features/channels/phone-simulation.tsx src/features/channels/node-config-panel.tsx
```
Expected: PASS / 零错误。

- [ ] **Step 7: 格式化并提交**

```bash
cd web/admin
pnpm exec prettier --write src/features/channels
cd /Users/mr9esx/Documents/Pixoma
git add web/admin/src/features/channels/channel-menu-editor.tsx web/admin/src/features/channels/phone-simulation.tsx web/admin/src/features/channels/node-config-panel.tsx web/admin/src/features/channels/channel-menu-editor.contract.test.ts
git commit -m "feat(admin): wysiwyg menu editor with phone simulation"
```

---

### Task F6: 保存校验 + 未保存提示

**Files:**
- Create: `web/admin/src/features/channels/lib/menu-validate.ts`
- Create: `web/admin/src/features/channels/lib/menu-validate.test.ts`
- Modify: `web/admin/vitest.config.ts`
- Modify: `web/admin/src/features/channels/channel-menu-editor.tsx`（接入校验与红条）

**Interfaces:**
- Produces:
  - `export type MenuValidation = { ok: boolean; errors: { id: string; message: string }[] }`
  - `export function validateMenu(items: MenuNode[]): MenuValidation`
  - `export function countUnsaved(items: MenuNode[], baseline: MenuNode[]): number`

- [ ] **Step 1: 写失败测试**

```ts
import { describe, expect, it } from 'vitest'
import type { MenuNode } from '@/lib/api/channel-menu'
import { countUnsaved, validateMenu } from './menu-validate'

function node(partial: Partial<MenuNode>): MenuNode {
  return { id: 'x', label: 'x', order: 0, enabled: true, ...partial }
}

describe('menu validate', () => {
  it('rejects more than 6 root buttons', () => {
    const items = Array.from({ length: 7 }, (_, i) =>
      node({ id: `r${i}`, label: `R${i}`, order: i, capability_id: 'reply_text', params: { text: 'x' } }),
    )
    expect(validateMenu(items).ok).toBe(false)
  })

  it('rejects leaf without capability and open_case without cases', () => {
    const items: MenuNode[] = [
      node({ id: 'a', label: 'A', order: 0 }),
      node({ id: 'b', label: 'B', order: 1, capability_id: 'open_case', params: { case_ids: [] } }),
    ]
    const errors = validateMenu(items).errors
    expect(errors.some((e) => e.id === 'a')).toBe(true)
    expect(errors.some((e) => e.id === 'b')).toBe(true)
  })

  it('rejects nested groups', () => {
    const items: MenuNode[] = [
      node({
        id: 'g',
        label: 'G',
        order: 0,
        children: [
          node({ id: 'g2', label: 'G2', order: 0, children: [node({ id: 'x', label: 'X', order: 0, capability_id: 'reply_text', params: { text: 'x' } })] }),
        ],
      }),
    ]
    expect(validateMenu(items).ok).toBe(false)
  })

  it('counts unsaved changes against baseline', () => {
    const baseline: MenuNode[] = [node({ id: 'a', label: 'A', order: 0, capability_id: 'reply_text', params: { text: 'x' } })]
    const changed: MenuNode[] = [node({ id: 'a', label: 'A2', order: 0, capability_id: 'reply_text', params: { text: 'x' } })]
    expect(countUnsaved(changed, baseline)).toBe(1)
  })
})
```

- [ ] **Step 2: 运行确认失败**

Run: `cd web/admin && pnpm test src/features/channels/lib/menu-validate.test.ts`
Expected: FAIL。

- [ ] **Step 3: 实现 menu-validate.ts**

```ts
import type { MenuNode } from '@/lib/api/channel-menu'

export type MenuValidation = {
  ok: boolean
  errors: { id: string; message: string }[]
}

function isOpenCaseEmpty(node: MenuNode): boolean {
  if (node.capability_id !== 'open_case') return false
  const ids = node.params?.case_ids
  return !Array.isArray(ids) || ids.length === 0
}

function validateNode(node: MenuNode, isRoot: boolean, errors: { id: string; message: string }[]) {
  if (!node.label.trim()) errors.push({ id: node.id, message: 'name' })
  if (isRoot) {
    if (!node.enabled) return
  }
  const hasChildren = (node.children?.length ?? 0) > 0
  if (!hasChildren && !node.capability_id) errors.push({ id: node.id, message: 'action' })
  if (isOpenCaseEmpty(node)) errors.push({ id: node.id, message: 'open_case' })
  if (hasChildren) {
    for (const child of node.children ?? []) {
      if ((child.children?.length ?? 0) > 0) errors.push({ id: child.id, message: 'nested' })
    }
  }
  for (const child of node.children ?? []) validateNode(child, false, errors)
}

export function validateMenu(items: MenuNode[]): MenuValidation {
  const errors: { id: string; message: string }[] = []
  if (items.filter((it) => it.enabled).length > 6) {
    errors.push({ id: '_root', message: 'root_count' })
  }
  for (const it of items) validateNode(it, true, errors)
  return { ok: errors.length === 0, errors }
}

function diffCount(a: unknown, b: unknown): number {
  return JSON.stringify(a) === JSON.stringify(b) ? 0 : 1
}

export function countUnsaved(items: MenuNode[], baseline: MenuNode[]): number {
  const byId = (list: MenuNode[]) => new Map(list.map((n) => [n.id, n]))
  const cur = byId(items)
  const base = byId(baseline)
  const ids = new Set([...cur.keys(), ...base.keys()])
  let count = 0
  for (const id of ids) {
    const a = cur.get(id)
    const b = base.get(id)
    if (!a || !b) {
      count += 1
      continue
    }
    count += diffCount(a.label, b.label)
    count += diffCount(a.capability_id, b.capability_id)
    count += diffCount(a.params, b.params)
    count += diffCount(a.intro_text, b.intro_text)
    count += diffCount(a.children, b.children)
  }
  return count
}
```

- [ ] **Step 4: 接入编辑器**

`channel-menu-editor.tsx`：`const validation = useMemo(() => validateMenu(items), [items])`；顶栏红条（`!validation.ok` 时显示首条错误对应的 `t('channelMenu.invalid*')`）；`saveMutation` 前先 `if (!validation.ok) return`；保存按钮 disabled 规则不变。未保存计数 `const dirtyCount = countUnsaved(items, initialItems)`，保存成功或加载后重置 `initialItems`。

- [ ] **Step 5: 运行确认通过 + 提交**

```bash
cd web/admin
pnpm test src/features/channels
pnpm exec tsc -b
pnpm exec prettier --write src/features/channels
cd /Users/mr9esx/Documents/Pixoma
git add web/admin/vitest.config.ts web/admin/src/features/channels/lib web/admin/src/features/channels/channel-menu-editor.tsx
git commit -m "feat(admin): menu editor save validation and unsaved hint"
```

---

### Task F7: 全量验证

**Files:** 无新增。

- [ ] **Step 1: 全量检查**

```bash
cd /Users/mr9esx/Documents/Pixoma/web/admin
pnpm test
pnpm exec tsc -b
pnpm exec eslint src/features/channels src/features/cases
pnpm exec prettier --check src/features/channels src/features/cases
```
Expected: 全部通过（react-refresh 既有 warning 除外）。

- [ ] **Step 2: 提交遗留改动**

```bash
cd /Users/mr9esx/Documents/Pixoma
git add -A web/admin
git commit -m "chore(admin): finalize tg menu editor frontend"
```
