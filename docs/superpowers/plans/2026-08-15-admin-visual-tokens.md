---
design-doc: docs/superpowers/specs/2026-08-15-admin-visual-tokens-design.md
---

# admin-visual-tokens Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 去掉 `web/admin` 贴在页面上的盒子投影，改成细线框；配色仍是现有 neutral 纯灰。

**Architecture:** 不改 `theme.css`。只改组件 class：表面 `shadow-sm` / `shadow-xs` 去掉，浮层（dialog / dropdown / popover / sheet 等）保留。用合同测试锁卡片、inset 画布、按钮，并覆盖表单控件与筛选箭头。

**Tech Stack:** React、Tailwind 工具类、Vitest 合同测试、pnpm、OpenSpec `admin-web-shell`

## Global Constraints

- 产物与提交说明语言：zh-CN
- 不改 `web/admin/src/styles/theme.css`、不改 `components.json` 的 `baseColor: "neutral"`
- 不改 ThemeProvider、布局排法、字体、圆角 `--radius`、Master–Detail
- 不要全局 `* { box-shadow: none }`
- 浮层投影必须保留：`dialog`、`alert-dialog`、`dropdown-menu`、`popover`、`select` 弹出层、`sheet`、批量操作条、加载条
- sidebar **floating** 变体上的 `shadow-sm` 保留；只去掉 **inset** `SidebarInset` 的 `shadow-sm`
- Canonical spec：`docs/openspec/specs/admin-web-shell/spec.md`

## 文件地图

| 文件 | 职责 |
|---|---|
| `web/admin/src/styles/theme-neutral.contract.test.ts` | 保留既有 neutral 色值断言；新增表面无投影抽样 |
| `web/admin/src/components/ui/card.tsx` | 卡片去 `shadow-sm` |
| `web/admin/src/components/ui/sidebar.tsx` | `SidebarInset` inset 变体去 `shadow-sm` |
| `web/admin/src/components/ui/button.tsx` | default / destructive / outline / secondary 去 `shadow-xs` |
| `web/admin/src/components/ui/input.tsx` 等表单控件 | 去 `shadow-xs` |
| `web/admin/src/components/password-input.tsx` | 密码框去 `shadow-xs` |
| `web/admin/src/components/filters/filter-segment.tsx` | 左右箭头去 `shadow-sm` |
| `docs/openspec/specs/admin-web-shell/spec.md` | 增补「表面无投影」需求；基色条款不动 |

---

### Task 1: 表面阴影合同测试先红

**Files:**
- Modify: `web/admin/src/styles/theme-neutral.contract.test.ts`
- Test: 同文件（Vitest 已 include 该路径，不必改 `vitest.config.ts`）

**Interfaces:**
- Consumes: 现有 `read()`、`adminRoot`、`SRC_ROOT`（该文件内 `here` 的上两级是 `web/admin`，`src` 在 `join(here, '..')` 若从 styles 出发；当前文件已用 `join(here, '../..')` 作为 `adminRoot`）
- Produces: 三个新用例名：`Card surface has no drop shadow`、`SidebarInset inset variant has no drop shadow`、`button default variant has no shadow-xs`

- [ ] **Step 1.1** 在 `theme-neutral.contract.test.ts` 末尾追加辅助函数与 `describe`（不要改已有三个 neutral 用例）：

```ts
function extractFunction(source: string, name: string): string {
  const start = source.indexOf(`function ${name}(`)
  expect(start, `expected function ${name}(`).toBeGreaterThanOrEqual(0)
  const next = source.indexOf('\nfunction ', start + 1)
  return next === -1 ? source.slice(start) : source.slice(start, next)
}

describe('admin theme surface has no drop shadow', () => {
  it('Card surface has no drop shadow', () => {
    const src = read(join(SRC_ROOT, 'components/ui/card.tsx'))
    const cardFn = extractFunction(src, 'Card')
    expect(cardFn).not.toMatch(/\bshadow-sm\b/)
    expect(cardFn).toMatch(/\bborder\b/)
  })

  it('SidebarInset inset variant has no drop shadow', () => {
    const src = read(join(SRC_ROOT, 'components/ui/sidebar.tsx'))
    const insetFn = extractFunction(src, 'SidebarInset')
    expect(insetFn).not.toMatch(/\bshadow-sm\b/)
    expect(insetFn).toMatch(/rounded-xl/)
  })

  it('button default variant has no shadow-xs', () => {
    const src = read(join(SRC_ROOT, 'components/ui/button.tsx'))
    const match = src.match(/default:\s*\n\s*'([^']+)'/)
    expect(match, 'expected default variant string').toBeTruthy()
    expect(match![1]).not.toMatch(/\bshadow-xs\b/)
  })
})
```

注意：`SRC_ROOT` 已是 `join(adminRoot, 'src')`。`extractFunction(src, 'Card')` 会在遇到 `function CardHeader` 处截断，这是预期。

- [ ] **Step 1.2** 跑测试，确认新用例失败、旧用例仍过：

Run: `pnpm --dir web/admin exec vitest run src/styles/theme-neutral.contract.test.ts`

Expected: 既有 `admin theme neutral base color` 三个用例 PASS；新 describe 三个用例 FAIL（`card.tsx` 仍含 `shadow-sm`，`SidebarInset` 仍含 `shadow-sm`，button default 仍含 `shadow-xs`）。

- [ ] **Step 1.3** Commit

```bash
git add web/admin/src/styles/theme-neutral.contract.test.ts
git commit -m "$(cat <<'EOF'
test: 锁定 admin 表面组件无投影

EOF
)"
```

---

### Task 2: 去掉卡片、内容画布、按钮表面投影

**Files:**
- Modify: `web/admin/src/components/ui/card.tsx`
- Modify: `web/admin/src/components/ui/sidebar.tsx`（仅 `SidebarInset` 那一行）
- Modify: `web/admin/src/components/ui/button.tsx`

**Interfaces:**
- Consumes: Task 1 的三个用例
- Produces: 卡片/inset/按钮表面无 `shadow-sm`/`shadow-xs`；`group-data-[variant=floating]:shadow-sm` 仍在 `sidebar.tsx` 其它函数里

- [ ] **Step 2.1** `card.tsx` 中 `Card` 的 class 从：

```
'flex flex-col gap-6 rounded-xl border bg-card py-6 text-card-foreground shadow-sm',
```

改为：

```
'flex flex-col gap-6 rounded-xl border bg-card py-6 text-card-foreground shadow-none',
```

- [ ] **Step 2.2** `sidebar.tsx` 的 `SidebarInset` class 从：

```
'md:peer-data-[variant=inset]:m-2 md:peer-data-[variant=inset]:ms-0 md:peer-data-[variant=inset]:rounded-xl md:peer-data-[variant=inset]:shadow-sm md:peer-data-[variant=inset]:peer-data-[state=collapsed]:ms-2',
```

改为：

```
'md:peer-data-[variant=inset]:m-2 md:peer-data-[variant=inset]:ms-0 md:peer-data-[variant=inset]:rounded-xl md:peer-data-[variant=inset]:peer-data-[state=collapsed]:ms-2',
```

不要动 `group-data-[variant=floating]:shadow-sm` 那一行。

- [ ] **Step 2.3** `button.tsx` 四个变体去掉 `shadow-xs `（含后面空格）：

```
default:
  'bg-primary text-primary-foreground hover:bg-primary/90',
destructive:
  'bg-destructive text-white hover:bg-destructive/90 focus-visible:ring-destructive/20 dark:focus-visible:ring-destructive/40 dark:bg-destructive/60',
outline:
  'border bg-background hover:bg-accent hover:text-accent-foreground dark:bg-input/30 dark:border-input dark:hover:bg-input/50',
secondary:
  'bg-secondary text-secondary-foreground hover:bg-secondary/80',
```

ghost / link 本来就没有 `shadow-xs`，不要改。

- [ ] **Step 2.4** 重跑合同测试：

Run: `pnpm --dir web/admin exec vitest run src/styles/theme-neutral.contract.test.ts`

Expected: 全部 PASS（含 Task 1 三个新用例 + 三个旧色值用例）。

- [ ] **Step 2.5** Commit

```bash
git add web/admin/src/components/ui/card.tsx web/admin/src/components/ui/sidebar.tsx web/admin/src/components/ui/button.tsx
git commit -m "$(cat <<'EOF'
fix: 去掉 admin 卡片、内容区和按钮的表面阴影

EOF
)"
```

---

### Task 3: 表单控件与筛选箭头去投影

**Files:**
- Modify: `web/admin/src/styles/theme-neutral.contract.test.ts`
- Modify: `web/admin/src/components/ui/input.tsx`
- Modify: `web/admin/src/components/ui/textarea.tsx`
- Modify: `web/admin/src/components/ui/select.tsx`（仅 `SelectTrigger`）
- Modify: `web/admin/src/components/ui/checkbox.tsx`
- Modify: `web/admin/src/components/ui/switch.tsx`
- Modify: `web/admin/src/components/ui/radio-group.tsx`（仅 `RadioGroupItem`）
- Modify: `web/admin/src/components/ui/input-otp.tsx`
- Modify: `web/admin/src/components/ui/calendar.tsx`（仅 `dropdown_root`）
- Modify: `web/admin/src/components/password-input.tsx`
- Modify: `web/admin/src/components/filters/filter-segment.tsx`

**Interfaces:**
- Consumes: Task 1 的 `extractFunction`
- Produces: 上列表面控件无 `shadow-xs` / 箭头无 `shadow-sm`；`SelectContent` 的 `shadow-md` 仍在

- [ ] **Step 3.1** 在同一 `describe('admin theme surface has no drop shadow')` 里追加用例（先红）：

```ts
  it('form controls have no shadow-xs', () => {
    const files: [string, string][] = [
      ['components/ui/input.tsx', 'Input'],
      ['components/ui/textarea.tsx', 'Textarea'],
      ['components/ui/select.tsx', 'SelectTrigger'],
      ['components/ui/checkbox.tsx', 'Checkbox'],
      ['components/ui/switch.tsx', 'Switch'],
      ['components/ui/radio-group.tsx', 'RadioGroupItem'],
    ]
    for (const [rel, fn] of files) {
      const body = extractFunction(read(join(SRC_ROOT, rel)), fn)
      expect(body, rel).not.toMatch(/\bshadow-xs\b/)
    }
    expect(read(join(SRC_ROOT, 'components/ui/input-otp.tsx'))).not.toMatch(
      /\bshadow-xs\b/,
    )
    expect(read(join(SRC_ROOT, 'components/ui/calendar.tsx'))).not.toMatch(
      /\bshadow-xs\b/,
    )
    expect(read(join(SRC_ROOT, 'components/password-input.tsx'))).not.toMatch(
      /\bshadow-xs\b/,
    )
  })

  it('filter-segment arrows have no shadow-sm', () => {
    const src = read(join(SRC_ROOT, 'components/filters/filter-segment.tsx'))
    expect(src).not.toMatch(/\bshadow-sm\b/)
  })

  it('select popover still has elevation', () => {
    const content = extractFunction(
      read(join(SRC_ROOT, 'components/ui/select.tsx')),
      'SelectContent',
    )
    expect(content).toMatch(/\bshadow-md\b/)
  })
```

- [ ] **Step 3.2** 跑测试，确认这两个新失败用例红、Task 1/2 用例仍绿：

Run: `pnpm --dir web/admin exec vitest run src/styles/theme-neutral.contract.test.ts`

Expected: `form controls have no shadow-xs` 与 `filter-segment arrows have no shadow-sm` FAIL；其余 PASS（含 `select popover still has elevation`，因 `SelectContent` 已有 `shadow-md`）。

- [ ] **Step 3.3** 从下列字符串中删除 `shadow-xs `（保留其余 class；`transition-[color,box-shadow]` 可留，那是过渡属性名不是投影）：

`input.tsx`：

```
'flex h-9 w-full min-w-0 rounded-md border border-input bg-transparent px-3 py-1 text-base transition-[color,box-shadow] outline-none selection:bg-primary selection:text-primary-foreground file:inline-flex file:h-7 file:border-0 file:bg-transparent file:text-sm file:font-medium file:text-foreground placeholder:text-muted-foreground disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-50 md:text-sm dark:bg-input/30',
```

`textarea.tsx`：把 `text-base shadow-xs transition-[color,box-shadow]` 改成 `text-base transition-[color,box-shadow]`。

`select.tsx` `SelectTrigger`：把 `whitespace-nowrap shadow-xs transition-[color,box-shadow]` 改成 `whitespace-nowrap transition-[color,box-shadow]`。不要改 `SelectContent` 的 `shadow-md`。

`checkbox.tsx`：`border-input shadow-xs transition-shadow` → `border-input transition-shadow`。

`switch.tsx`：`border-transparent shadow-xs transition-all` → `border-transparent transition-all`。

`radio-group.tsx` `RadioGroupItem`：`text-primary shadow-xs transition-[color,box-shadow]` → `text-primary transition-[color,box-shadow]`。

`input-otp.tsx` slot：`text-sm shadow-xs transition-all` → `text-sm transition-all`。

`calendar.tsx` `dropdown_root`：`border-input shadow-xs has-focus:border-ring` → `border-input has-focus:border-ring`。

`password-input.tsx`：`py-1 text-sm shadow-xs transition-colors` → `py-1 text-sm transition-colors`。

`filter-segment.tsx` 左右箭头两处：`rounded-full border shadow-sm backdrop-blur-sm` → `rounded-full border backdrop-blur-sm`。

- [ ] **Step 3.4** 重跑合同测试：

Run: `pnpm --dir web/admin exec vitest run src/styles/theme-neutral.contract.test.ts`

Expected: 全部 PASS。

- [ ] **Step 3.5** 确认浮层没被误伤：

Run: `rg -n 'shadow-(sm|xs|md|lg|xl)' web/admin/src/components/ui/dialog.tsx web/admin/src/components/ui/dropdown-menu.tsx web/admin/src/components/ui/popover.tsx web/admin/src/components/ui/sheet.tsx web/admin/src/components/ui/alert-dialog.tsx`

Expected: 这些文件仍含 `shadow-md` 或 `shadow-lg`。

- [ ] **Step 3.6** Commit

```bash
git add \
  web/admin/src/styles/theme-neutral.contract.test.ts \
  web/admin/src/components/ui/input.tsx \
  web/admin/src/components/ui/textarea.tsx \
  web/admin/src/components/ui/select.tsx \
  web/admin/src/components/ui/checkbox.tsx \
  web/admin/src/components/ui/switch.tsx \
  web/admin/src/components/ui/radio-group.tsx \
  web/admin/src/components/ui/input-otp.tsx \
  web/admin/src/components/ui/calendar.tsx \
  web/admin/src/components/password-input.tsx \
  web/admin/src/components/filters/filter-segment.tsx
git commit -m "$(cat <<'EOF'
fix: 去掉 admin 表单控件和筛选箭头的表面阴影

EOF
)"
```

---

### Task 4: OpenSpec 与回归

**Files:**
- Modify: `docs/openspec/specs/admin-web-shell/spec.md`
- Verify: `web/admin` 既有合同测试

**Interfaces:**
- Consumes: 已落地的表面无投影行为
- Produces: canonical spec 增加「表面无投影」；「中性基色主题」原文不动

- [ ] **Step 4.1** 在 `docs/openspec/specs/admin-web-shell/spec.md` 的「中性基色主题」整节之后、「未初始化进入向导而非业务壳」之前，插入：

```md
### Requirement: 表面无投影
系统 MUST 让贴在页面上的卡片、inset 内容画布、按钮与表单控件使用细线框而非 drop shadow。对话框、下拉菜单、popover、sheet 等浮层 MAY 保留投影。

#### Scenario: 卡片与内容区无表面阴影
- **WHEN** 用户打开 Dashboard 或任意壳页
- **THEN** 卡片与 inset 内容画布不以 drop shadow 垫高，而以边框区分层次

#### Scenario: 浮层仍有投影
- **WHEN** 用户打开下拉菜单或对话框
- **THEN** 该浮层仍可使用投影与页面背景区分
```

不要改「中性基色主题」四条 scenario。不要改 `docs/architecture/`。

- [ ] **Step 4.2** 跑 admin 合同测试：

Run: `pnpm --dir web/admin test`

Expected: 全部 PASS。`theme.css` 与 `components.json` 的 `baseColor` 相对本 change 无 diff。

- [ ] **Step 4.3** Commit

```bash
git add docs/openspec/specs/admin-web-shell/spec.md
git commit -m "$(cat <<'EOF'
docs: admin 壳增加表面无投影约定

EOF
)"
```

---

## Spec coverage

| Spec 条目 | 任务 |
|---|---|
| 不改配色 / `baseColor` / `theme.css` | Global Constraints + Task 4.2 验无 diff |
| 卡片、inset、按钮去表面阴影 | Task 1–2 |
| 表单控件、filter-segment 箭头去阴影 | Task 3 |
| 浮层保留投影 | Task 3.1 `select popover` 用例 + Task 3.5 rg |
| 不改 Master–Detail / 布局 variant | 无对应实现步骤（故意） |
| OpenSpec 基色条款不动、补表面无投影 | Task 4 |
| 既有 neutral 合同仍过 | Task 1.2 / 2.4 / 3.4 / 4.2 |
