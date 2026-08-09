---
change: admin-shell-sidebar-footer
design-doc: docs/superpowers/specs/2026-08-09-admin-shell-sidebar-footer-design.md
base-ref: 6e0d1afba44e9ffabcfc621da960b691a0b21ab7
---

# admin-shell-sidebar-footer Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 `web/admin` 管理壳收成 Pixoma 品牌：去掉内容区壳级顶栏，语言/主题迁入侧栏底栏，侧栏品牌显示 Pixoma。

**Architecture:** 主布局仍在 `routes/_app.tsx`（`SidebarProvider` + `AppSidebar` + `SidebarInset`）；`AppSidebar` 在 `SidebarFooter` 挂载 `LanguageSwitcher` 与 `ThemeSwitch`；`LanguageSwitcher` 改为与 `ThemeSwitch` 一致的图标触发 + `DropdownMenu`；`AppTitle` / `Logo` 只改文案；**不修改** `MENU_ITEMS` 与菜单 map 逻辑。

**Tech Stack:** Vite、React、TypeScript、TanStack Router、shadcn/ui Sidebar、`lucide-react`、`react-i18next`、Vitest（合约测试读源码）

**依据：**
- Design Doc：`docs/superpowers/specs/2026-08-09-admin-shell-sidebar-footer-design.md`
- OpenSpec：`docs/openspec/changes/admin-shell-sidebar-footer/{proposal,design,tasks}.md`
- Delta spec：`docs/openspec/changes/admin-shell-sidebar-footer/specs/admin-web-shell/spec.md`

## Global Constraints

- 产物语言：zh-CN（计划与用户可见文案默认中文）
- 包管理器：**pnpm only**（`cd web/admin && pnpm …`）
- **禁止** 修改 `web/admin/src/config/menu.ts` 与 `AppSidebar` 内 `MENU_ITEMS.map` 块（降低与 `tg-menu-config` 合并冲突）
- **禁止** 新增 E2E / 视觉回归套件
- **禁止** 重绘 Logo SVG 路径；仅改 `<title>` / 可访问名称与 `AppTitle` 文案
- 内容区 **MUST NOT** 保留仅承载语言/主题的壳级 `<header>`
- 语言/主题 **唯一壳级挂载点**：`AppSidebar` 的 `SidebarFooter`
- 侧栏收起（icon）模式：底栏控件仍须可点；`DropdownMenu` 使用 portal（与 `ThemeSwitch` 一致，`modal={false}`）
- `data-testid='language-switcher'` 保留在新组件根节点
- 默认语言 `zh`；偏好 key：`admin-locale:v1`（沿用现有 `setStoredLocale`）
- 合并 `tg-menu-config` 时：先合对方菜单 diff，本 change 只加 Footer / Title / 删顶栏
- 本 change 不改 admin-api、不改架构拓扑；**不必** 更新 `docs/architecture/`

---

## 文件结构（本 change 触及）

| 路径 | 变更 |
|---|---|
| `web/admin/src/routes/_app.tsx` | 删除 `SidebarInset` 内壳级 `<header>` 及相关 import |
| `web/admin/src/components/layout/app-sidebar.tsx` | 增加 `SidebarFooter`，挂 `LanguageSwitcher` + `ThemeSwitch` |
| `web/admin/src/components/layout/app-title.tsx` | 主标题 `Pixoma`；移除副标题行 |
| `web/admin/src/assets/logo.tsx` | `<title>` 改为 Pixoma |
| `web/admin/src/components/layout/language-switcher.tsx` | 双文字按钮 → Globe 图标 + 下拉 |
| `web/admin/src/components/layout/shell-layout.contract.test.ts` | **新建** 壳布局合约测试 |
| `web/admin/vitest.config.ts` | `include` 追加新合约测试路径 |

**只读对照、本 change 不为主路径：**
- `web/admin/src/components/layout/authenticated-layout.tsx`（已无顶栏；确认无语言/主题 import 即可）
- `web/admin/src/components/theme-switch.tsx`（参考实现，通常不改）

---

### Task 1: 壳布局合约测试（先红后绿）

**Files:**
- Create: `web/admin/src/components/layout/shell-layout.contract.test.ts`
- Modify: `web/admin/vitest.config.ts`（`include` 数组）

**Interfaces:**
- Produces: 合约测试文件名 `shell-layout.contract.test.ts`，断言壳级顶栏移除、Footer 挂载、Pixoma 品牌、无模板文案

- [ ] **Step 1: 将新测试加入 Vitest 白名单**

在 `web/admin/vitest.config.ts` 的 `include` 数组末尾追加：

```ts
'src/components/layout/shell-layout.contract.test.ts',
```

- [ ] **Step 2: 写失败的合约测试**

创建 `web/admin/src/components/layout/shell-layout.contract.test.ts`：

```ts
import { existsSync, readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const here = dirname(fileURLToPath(import.meta.url))
const srcRoot = join(here, '../..')

const APP_LAYOUT = join(srcRoot, 'routes/_app.tsx')
const APP_SIDEBAR = join(here, 'app-sidebar.tsx')
const APP_TITLE = join(here, 'app-title.tsx')
const LOGO = join(srcRoot, 'assets/logo.tsx')
const LANG_SWITCHER = join(here, 'language-switcher.tsx')

function read(path: string) {
  return readFileSync(path, 'utf8')
}

describe('admin shell layout (sidebar footer + no content header)', () => {
  it('_app.tsx has no shell-level header with language/theme', () => {
    const source = read(APP_LAYOUT)
    expect(source).not.toMatch(/<header[^>]*>[\s\S]*LanguageSwitcher/)
    expect(source).not.toMatch(/<header[^>]*>[\s\S]*ThemeSwitch/)
    expect(source).not.toContain('import { LanguageSwitcher }')
    expect(source).not.toContain('import { ThemeSwitch }')
  })

  it('AppSidebar mounts tools in SidebarFooter', () => {
    const source = read(APP_SIDEBAR)
    expect(source).toContain('SidebarFooter')
    expect(source).toContain('LanguageSwitcher')
    expect(source).toContain('ThemeSwitch')
  })

  it('AppTitle shows Pixoma without template subtitle', () => {
    const source = read(APP_TITLE)
    expect(source).toContain('Pixoma')
    expect(source).not.toContain('Shadcn-Admin')
    expect(source).not.toContain('Vite + ShadcnUI')
  })

  it('Logo accessible name is Pixoma', () => {
    const source = read(LOGO)
    expect(source).toContain('<title>Pixoma</title>')
    expect(source).not.toContain('Shadcn-Admin')
  })

  it('LanguageSwitcher uses dropdown trigger (not dual text buttons)', () => {
    const source = read(LANG_SWITCHER)
    expect(source).toContain('DropdownMenu')
    expect(source).toContain('data-testid=\'language-switcher\'')
    expect(source).not.toMatch(/size='sm'[\s\S]*lang\.zh[\s\S]*size='sm'[\s\S]*lang\.en/)
  })
})
```

- [ ] **Step 3: 运行测试确认失败**

```bash
cd web/admin
pnpm test src/components/layout/shell-layout.contract.test.ts
```

Expected: FAIL（`AppTitle` 仍含 Shadcn-Admin、`_app.tsx` 仍有顶栏、`AppSidebar` 无 Footer 等）

- [ ] **Step 4: Commit 测试骨架（可选，实现前单独提交）**

```bash
git add web/admin/vitest.config.ts web/admin/src/components/layout/shell-layout.contract.test.ts
git commit -m "test(admin): add shell layout contract tests for sidebar footer change"
```

---

### Task 2: 侧栏品牌 — AppTitle 与 Logo

**Files:**
- Modify: `web/admin/src/components/layout/app-title.tsx:28-29`
- Modify: `web/admin/src/assets/logo.tsx:20`

**Interfaces:**
- Consumes: 无
- Produces: 侧栏品牌区显示 `Pixoma`；SVG `<title>Pixoma</title>`

- [ ] **Step 1: 修改 AppTitle 主标题并移除副标题**

`web/admin/src/components/layout/app-title.tsx` 将 Link 内两行改为单行：

```tsx
<span className='truncate font-bold'>Pixoma</span>
```

删除原第二行：

```tsx
<span className='truncate text-xs'>Vite + ShadcnUI</span>
```

保留 `ToggleSidebar` 与 `Link to='/'` 行为不变。

- [ ] **Step 2: 修改 Logo 可访问名称**

`web/admin/src/assets/logo.tsx`：

```tsx
<title>Pixoma</title>
```

（`id='shadcn-admin-logo'` 本期可保留，避免无关 DOM 选择器回归；仅改 title 文本。）

- [ ] **Step 3: 运行合约测试（部分通过）**

```bash
cd web/admin
pnpm test src/components/layout/shell-layout.contract.test.ts
```

Expected: `AppTitle` / `Logo` 相关用例 PASS；Footer / 顶栏 / LanguageSwitcher 仍 FAIL

- [ ] **Step 4: Commit**

```bash
git add web/admin/src/components/layout/app-title.tsx web/admin/src/assets/logo.tsx
git commit -m "feat(admin): brand sidebar title as Pixoma"
```

---

### Task 3: LanguageSwitcher — 图标 + 下拉

**Files:**
- Modify: `web/admin/src/components/layout/language-switcher.tsx`（整文件重写）

**Interfaces:**
- Consumes: `setStoredLocale`, `AppLocale` from `@/lib/i18n`；`t` from `react-i18next`；`i18n.changeLanguage`
- Produces: `export function LanguageSwitcher()` — 根节点带 `data-testid='language-switcher'`；Globe 图标触发；下拉项 `t('lang.zh')` / `t('lang.en')`

- [ ] **Step 1: 按 ThemeSwitch 模式实现下拉**

完整替换 `language-switcher.tsx`：

```tsx
import { Check, Globe } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { setStoredLocale, type AppLocale } from '@/lib/i18n'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'

export function LanguageSwitcher() {
  const { i18n, t } = useTranslation()
  const current: AppLocale = i18n.language.startsWith('en') ? 'en' : 'zh'

  function switchTo(locale: AppLocale) {
    void i18n.changeLanguage(locale)
    setStoredLocale(locale)
  }

  return (
    <div data-testid='language-switcher'>
      <DropdownMenu modal={false}>
        <DropdownMenuTrigger asChild>
          <Button
            variant='ghost'
            size='icon'
            className='scale-95 rounded-full'
            type='button'
          >
            <Globe className='size-[1.2rem]' />
            <span className='sr-only'>{t('lang.switch')}</span>
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align='end'>
          <DropdownMenuItem onClick={() => switchTo('zh')}>
            {t('lang.zh')}
            <Check
              size={14}
              className={cn('ms-auto', current !== 'zh' && 'hidden')}
            />
          </DropdownMenuItem>
          <DropdownMenuItem onClick={() => switchTo('en')}>
            {t('lang.en')}
            <Check
              size={14}
              className={cn('ms-auto', current !== 'en' && 'hidden')}
            />
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  )
}
```

- [ ] **Step 2: 补充 i18n key `lang.switch`（若 locale 文件尚无）**

检查 `web/admin/src/lib/i18n/locales/zh.json` 与 `en.json` 的 `lang` 段。若无 `switch`，追加：

`zh.json`：

```json
"switch": "切换语言"
```

`en.json`：

```json
"switch": "Switch language"
```

若已有等价 key（如 `common.language`），可复用并在 Step 1 改用该 key，**勿** 留未定义 key。

- [ ] **Step 3: 运行合约测试**

```bash
cd web/admin
pnpm test src/components/layout/shell-layout.contract.test.ts
```

Expected: LanguageSwitcher 用例 PASS

- [ ] **Step 4: Commit**

```bash
git add web/admin/src/components/layout/language-switcher.tsx web/admin/src/lib/i18n/locales/zh.json web/admin/src/lib/i18n/locales/en.json
git commit -m "feat(admin): language switcher icon dropdown in sidebar style"
```

---

### Task 4: AppSidebar — SidebarFooter 挂载工具

**Files:**
- Modify: `web/admin/src/components/layout/app-sidebar.tsx`

**Interfaces:**
- Consumes: `LanguageSwitcher`, `ThemeSwitch`；`SidebarFooter` from `@/components/ui/sidebar`
- Produces: Footer 内横向 `LanguageSwitcher` + `ThemeSwitch`；**不改动** `MENU_ITEMS.map` 块

- [ ] **Step 1: 增加 import**

```tsx
import { LanguageSwitcher } from './language-switcher'
import { ThemeSwitch } from '@/components/theme-switch'
```

在 sidebar import 中追加 `SidebarFooter`：

```tsx
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  // ...existing imports
} from '@/components/ui/sidebar'
```

- [ ] **Step 2: 在 SidebarContent 之后、SidebarRail 之前插入 Footer**

```tsx
      </SidebarContent>
      <SidebarFooter>
        <div
          className='flex items-center gap-2 px-1 group-data-[collapsible=icon]:justify-center'
        >
          <LanguageSwitcher />
          <ThemeSwitch />
        </div>
      </SidebarFooter>
      <SidebarRail />
```

说明：
- `group-data-[collapsible=icon]:justify-center` 使收起态两图标居中
- 不包裹 Tooltip：与 `ThemeSwitch` 一致依赖 `sr-only`；若手动验收发现收起态难发现，再对两按钮外包 `Tooltip`（非本 Task 默认范围）

- [ ] **Step 3: 确认未改动 MENU_ITEMS**

```bash
cd web/admin
git diff src/components/layout/app-sidebar.tsx | grep -E 'MENU_ITEMS|config/menu' || true
```

Expected: 无对 `MENU_ITEMS` / `menu.ts` 的 diff 行

- [ ] **Step 4: 运行合约测试**

```bash
pnpm test src/components/layout/shell-layout.contract.test.ts
```

Expected: `AppSidebar mounts tools in SidebarFooter` PASS

- [ ] **Step 5: Commit**

```bash
git add web/admin/src/components/layout/app-sidebar.tsx
git commit -m "feat(admin): mount language and theme switches in sidebar footer"
```

---

### Task 5: 移除内容区壳级顶栏

**Files:**
- Modify: `web/admin/src/routes/_app.tsx`

**Interfaces:**
- Consumes: 语言/主题已由 `AppSidebar` Footer 提供
- Produces: `SidebarInset` 内直接 `<div className='flex-1 p-4'><Outlet /></div>`，无壳级 `<header>`

- [ ] **Step 1: 删除顶栏与无用 import**

从 `_app.tsx` 移除：

```tsx
import { LanguageSwitcher } from '@/components/layout/language-switcher'
import { ThemeSwitch } from '@/components/theme-switch'
```

删除整个：

```tsx
            <header className='flex h-14 items-center gap-2 border-b px-4'>
              <div className='ms-auto flex items-center gap-2'>
                <LanguageSwitcher />
                <ThemeSwitch />
              </div>
            </header>
```

保留 `SidebarInset` 与 `Outlet` 包裹：

```tsx
          <SidebarInset
            className={cn(
              '@container/content',
              'has-data-[layout=fixed]:h-svh',
              'peer-data-[variant=inset]:has-data-[layout=fixed]:h-[calc(100svh-(var(--spacing)*4))]',
            )}
          >
            <div className='flex-1 p-4'>
              <Outlet />
            </div>
          </SidebarInset>
```

- [ ] **Step 2: 确认 authenticated-layout 无重复挂载**

```bash
grep -n 'LanguageSwitcher\|ThemeSwitch\|<header' web/admin/src/components/layout/authenticated-layout.tsx || true
```

Expected: 无匹配（当前文件已无顶栏；若 grep 有命中则按同样规则清理，但通常无需改）

- [ ] **Step 3: 运行全部合约测试**

```bash
cd web/admin
pnpm test src/components/layout/shell-layout.contract.test.ts
pnpm test
```

Expected: 全部 PASS；无回归

- [ ] **Step 4: Commit**

```bash
git add web/admin/src/routes/_app.tsx
git commit -m "feat(admin): remove content shell header; tools live in sidebar footer"
```

---

### Task 6: 手动验收与 lint

**Files:**
- 无代码变更（仅验证）

**Interfaces:**
- Consumes: Task 1–5 全部完成

- [ ] **Step 1: 本地启动**

```bash
# 仓库根或 apps/admin-api 按 README 起 API（若 Dashboard 需要数据）
cd web/admin
pnpm dev
```

浏览器打开控制台默认地址（通常 `http://localhost:5173`）。

- [ ] **Step 2: 桌面展开侧栏（OpenSpec 4.1）**

对照检查：
- 侧栏顶部显示 **Pixoma**，无「Shadcn-Admin」「Vite + ShadcnUI」
- 内容区顶部 **无** 横条顶栏，页面从 padding 内直接开始
- 侧栏底栏可见 Globe 与主题图标；切换中文/English 后菜单文案变化
- 切换 Light/Dark/System 后主题变化

- [ ] **Step 3: 侧栏收起 icon 模式（OpenSpec 4.2）**

点击 `AppTitle` 旁折叠触发（桌面 `Menu` 图标）收起侧栏：
- 底栏两图标仍可见、可点
- 语言下拉可选 zh/en；主题下拉可选三种模式
- 下拉不被侧栏裁切（portal 到 body）

- [ ] **Step 4: 移动端抽屉**

窄屏或 DevTools 设备模式：
- 通过 `AppTitle` 区域打开侧栏 sheet
- 底栏工具可用
- 内容区仍无壳级顶栏

- [ ] **Step 5: 刷新持久化**

切换为 English 后刷新页面 → 仍为 English（`admin-locale:v1`）。

- [ ] **Step 6: lint**

```bash
cd web/admin
pnpm lint
pnpm format:check
```

Expected: 无 error

---

## 并行 change 合并备忘

若与 `tg-menu-config` 同改 `app-sidebar.tsx`：

1. 先合并对方的 `MENU_ITEMS` / 菜单项 diff
2. 在本分支 Footer 块上解决冲突：保留对方菜单 + 本 change 的 `SidebarFooter` 块
3. **禁止** 在冲突解决时顺带改菜单顺序或条目

---

## Self-Review（计划自检）

| Spec / tasks 条目 | 对应 Task |
|---|---|
| 1.1 AppTitle Pixoma、无副标题 | Task 2 |
| 1.2 Logo / 可访问名 Pixoma | Task 2 |
| 2.1 SidebarFooter + 两控件 | Task 4 |
| 2.2 收起态可图标操作 | Task 4（布局 class）+ Task 6 Step 3 |
| 3.1 移除 `_app` 壳顶栏 | Task 5 |
| 3.2 内容区不再 import 语言/主题 | Task 5 |
| 4.1–4.3 手动与单测 | Task 1 + Task 6 |
| 不改 MENU_ITEMS | Task 4 Step 3 |
| LanguageSwitcher 图标+下拉 | Task 3 |
| OpenSpec：无内容顶栏、底栏工具、Pixoma 品牌 | Task 2–6 |

占位扫描：无 TBD / TODO /「适当处理」类步骤。

---

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-08-09-admin-shell-sidebar-footer.md`. Two execution options:

**1. Subagent-Driven (recommended)** — 每 Task 派生子 agent，Task 间人工/代理复核

**2. Inline Execution** — 本 session 用 executing-plans 按 Task 批量执行并设检查点

Which approach?
