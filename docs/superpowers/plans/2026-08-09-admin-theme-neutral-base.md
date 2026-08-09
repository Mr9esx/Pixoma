---
change: admin-theme-neutral-base
design-doc: docs/superpowers/specs/2026-08-09-admin-theme-neutral-base-design.md
base-ref: 2d2fdd33bbef674e3519a2ef547591fc34ffb34e
archived-with: 2026-08-09-admin-theme-neutral-base
---

# admin-theme-neutral-base Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 `web/admin` 亮/暗主题基色从 slate 换成 neutral，消除暗色冷蓝屏。

**Architecture:** 手工对齐 shadcn 官方 neutral OKLCH 语义 token；更新 `components.json`；保留现有 sidebar var 映射与 ThemeProvider；补合同测试锁定 `baseColor` 与暗色 background chroma≈0。

**Tech Stack:** CSS 变量（OKLCH）、shadcn/ui、Vitest 合同测试、pnpm

## Global Constraints

- 产物与提交说明语言：zh-CN
- 不改 ThemeProvider / 布局 / 字体 / 圆角体系
- 不依赖 `shadcn migrate base-color`（当前 CLI 无此迁移）
- chart 色板可保留色相；主表面必须中性灰
- Canonical spec：`docs/openspec/changes/admin-theme-neutral-base/specs/admin-web-shell/spec.md`

## 文件地图

| 文件 | 职责 |
|---|---|
| `web/admin/components.json` | `baseColor: neutral` |
| `web/admin/src/styles/theme.css` | `:root` / `.dark` 语义色 → 官方 neutral；保留 sidebar var 映射 |
| `web/admin/src/lib/show-submitted-data.tsx` | `bg-slate-950` → 语义色 |
| `web/admin/src/styles/theme-neutral.contract.test.ts`（新建）或并入现有 shell/theme 合同测试 | 断言 baseColor + 暗色 background |

---

## Task 1: 合同测试先红

**Files:**
- Create or extend: `web/admin/src/styles/theme-neutral.contract.test.ts`（或现有 `shell-layout.contract.test.ts` 旁新建更贴切）
- Read: `web/admin/components.json`, `web/admin/src/styles/theme.css`

- [x] **Step 1.1** 写失败测试：读 `components.json`，断言 `tailwind.baseColor === "neutral"`
- [x] **Step 1.2** 写失败测试：从 `theme.css` 的 `.dark` 块解析 `--background`，断言为 `oklch(0.145 0 0)`（或 chroma 通道为 0）
- [x] **Step 1.3** 运行 `pnpm --dir web/admin exec vitest run src/styles/theme-neutral.contract.test.ts`（路径以实际为准），确认失败
- [x] **Step 1.4** Commit：`test: 锁定 admin neutral 基色合同`

## Task 2: 迁移基色与 token

**Files:**
- Modify: `web/admin/components.json`
- Modify: `web/admin/src/styles/theme.css`

- [x] **Step 2.1** `baseColor` 改为 `"neutral"`
- [x] **Step 2.2** 按官方 Default Theme CSS（neutral）替换 `:root` / `.dark` 语义色；保留 `--radius`、字体 `@theme`、`--sidebar: var(--background)` 等映射
- [x] **Step 2.3** 重跑合同测试，确认通过
- [x] **Step 2.4** Commit：`feat: admin 主题基色改为 neutral`

## Task 3: 清理硬编码并回归

**Files:**
- Modify: `web/admin/src/lib/show-submitted-data.tsx`
- Verify: `rg '\bslate-[0-9]' web/admin`

- [x] **Step 3.1** `bg-slate-950` → `bg-muted`（或更合适的语义色）
- [x] **Step 3.2** 确认无其它 `slate-*` 色阶硬编码
- [x] **Step 3.3** 运行相关 vitest（合同 + 既有 theme/shell）；必要时目视 light/dark
- [x] **Step 3.4** Commit：`fix: 去掉 admin 残留 slate 硬编码`
- [x] **Step 3.5** 勾选 `docs/openspec/changes/admin-theme-neutral-base/tasks.md` 对应项
