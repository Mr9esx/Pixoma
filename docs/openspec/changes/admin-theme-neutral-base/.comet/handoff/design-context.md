# Comet Design Handoff

- Change: admin-theme-neutral-base
- Phase: design
- Mode: full
- Context hash: 1f8e6f1e10fed885792213e96f02de00cfaf77050e6ad5acb9d03a2c9b981fb5

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## docs/openspec/changes/admin-theme-neutral-base/proposal.md

- Source: docs/openspec/changes/admin-theme-neutral-base/proposal.md
- Lines: 1-27
- SHA256: 222686e5907aa6669ca1165900494d0a63b50b27253573cdf2ea2dcb800d5431

```md
## Why

管理后台暗色模式当前使用 shadcn **slate** 基色，OKLCH 色相约 265° 且带明显饱和度，大面积背景/卡片/muted 叠在一起会显「冷蓝屏」。需要换成真中性灰（**neutral**），让日夜主题都干净、不发蓝，符合专业后台观感。

## What Changes

- 将 `web/admin` 的 shadcn `baseColor` 从 `slate` 切换为 `neutral`
- 同步更新亮色（`:root`）与暗色（`.dark`）CSS 语义 token（background、foreground、card、primary、muted、sidebar 等）
- 审计并清理会与新基色打架的硬编码 `slate-*` 工具类（若存在且影响观感）
- 保持现有明暗主题切换、cookie 持久化与命令面板主题命令行为不变

## Capabilities

### New Capabilities

- （无）

### Modified Capabilities

- `admin-web-shell`: 「布局与基础主题」从未指定基色，收紧为亮/暗共用 **neutral** 中性基色；暗色 MUST 不以蓝灰（slate）为主调

## Impact

- 代码：`web/admin/src/styles/theme.css`、`web/admin/components.json`；可能波及少量硬编码色类的组件
- API / 后端：无
- 依赖：可选用 shadcn `migrate base-color` CLI，或以官方 neutral token 手工对齐
- 用户可见：亮暗模式整体灰阶观感变化；功能行为不变

```

## docs/openspec/changes/admin-theme-neutral-base/design.md

- Source: docs/openspec/changes/admin-theme-neutral-base/design.md
- Lines: 1-55
- SHA256: 3c7c0ba2d6fe18d2835c771e4a01ea0cf79e6fdb13e28864eadc7ffff8697e3e

```md
## Context

当前 `web/admin` 使用 shadcn `baseColor: slate`，主题变量集中在 `src/styles/theme.css`（`:root` / `.dark`）。暗色背景等 token 色相约 265°、chroma 明显，导致「冷蓝屏」。主题切换由既有 `ThemeProvider`（cookie + `.dark` class）负责，本变更不改切换机制。动机见 `proposal.md`。

## Goals / Non-Goals

**Goals:**

- 亮色与暗色成对切换到官方 **neutral** 语义 token
- `components.json` 的 `baseColor` 与 CSS 对齐，避免后续 `shadcn add` 再拉回 slate
- 清理与观感冲突的硬编码 `slate-*`（仅影响表面色的用法）

**Non-Goals:**

- 不引入自定义品牌色板 / 多主题皮肤
- 不改 zinc / stone；不单独定制 chart 色板（除非随官方 neutral 预设一并替换）
- 不改 ThemeProvider、命令面板主题命令、字体或布局结构
- 不做全站视觉 redesign（圆角、阴影、动效等）

## Decisions

### 1. 基色选 Neutral（非 Zinc）

- **选择**：`neutral`（真灰，chroma≈0）
- **理由**：用户目标是去掉蓝感；zinc 仍带轻微冷色，对「太蓝」去得不够干净
- **备选**：zinc（更接近多数 shadcn 模板）、仅降低 slate chroma（改动小但长期易被 CLI 覆盖）

### 2. 迁移方式优先官方 token 对齐

- **选择**：以 shadcn 官方 neutral 的 `:root` / `.dark` 变量为准更新 `theme.css`，并设置 `components.json` → `baseColor: "neutral"`
- **可选加速**：若本机 CLI 可用，可用 `shadcn migrate base-color --from slate --to neutral`（或等价命令）改 CSS + config；不可用则手工粘贴官方值
- **保留**：现有 `--radius`、字体 `@theme`、sidebar 与 semantic token 的映射关系（`--sidebar: var(--background)` 等项目约定可保留，只要底层色已是 neutral）

### 3. 硬编码色类：按需审计，不全面替换 Tailwind 调色板

- **选择**：检索 `slate-*` 等与表面/背景相关的硬编码；语义上应走 token 的改为 `background` / `muted` / `border` 等；状态色（如 `emerald-*` 成功态）保留
- **理由**：全面改 Tailwind 默认色阶收益低；问题根源在 CSS 变量

### 4. 验证方式

- 本地切 light/dark，目视主背景与卡片无蓝灰主调
- 既有 theme 相关合同测试（若有）保持通过；必要时补一条「baseColor=neutral」或 token 抽样断言

## Risks / Trade-offs

- [Risk] 手工粘贴 token 与官方版本漂移 → Mitigation：优先 CLI migrate；手工时对照当前 shadcn registry neutral 文档
- [Risk] 局部 `slate-*` / `bg-slate-950` 残留造成色块不一致 → Mitigation：实施时 ripgrep 审计并改关键路径
- [Trade-off] 全站灰阶观感会变；接受为预期结果，不做逐页像素对齐

## Migration Plan

1. 更新 `components.json` 与 `theme.css`
2. 审计硬编码色类
3. 本地 light/dark 冒烟
4. 回滚：恢复 `baseColor: slate` 与原 `theme.css` 即可（纯前端、无数据迁移）

```

## docs/openspec/changes/admin-theme-neutral-base/tasks.md

- Source: docs/openspec/changes/admin-theme-neutral-base/tasks.md
- Lines: 1-13
- SHA256: 48fc002551f616c616b2c2e940d74c302b90a6b58bd1d44885bad0b3ec15c309

```md
## 1. 基色迁移

- [ ] 1.1 将 `web/admin/components.json` 的 `tailwind.baseColor` 从 `slate` 改为 `neutral`
- [ ] 1.2 用 shadcn `migrate base-color`（若可用）或对照官方 neutral 预设，更新 `web/admin/src/styles/theme.css` 的 `:root` 与 `.dark` 语义 token；保留项目约定的 `--radius`、字体与 sidebar 变量映射策略

## 2. 硬编码色审计

- [ ] 2.1 检索 `web/admin` 中影响表面观感的 `slate-*`（及明显冲突的硬编码背景色），改为语义 token 类或可接受的中性写法；保留状态色（如 emerald）与无关装饰色

## 3. 验证

- [ ] 3.1 本地切换 light / dark / system，确认主背景与卡片不呈蓝灰主调，主题切换与偏好持久化仍正常
- [ ] 3.2 运行相关前端测试（至少 theme / shell 相关合同测试），必要时补充 `baseColor=neutral` 或 token 抽样断言

```

## docs/openspec/changes/admin-theme-neutral-base/specs/admin-web-shell/spec.md

- Source: docs/openspec/changes/admin-theme-neutral-base/specs/admin-web-shell/spec.md
- Lines: 1-20
- SHA256: 1e4073f30aad78d8c6afa789d9168dd3074c97c6894b2c2f26a77eab025dda77

```md
## ADDED Requirements

### Requirement: 中性基色主题
系统 MUST 以 shadcn **neutral** 作为管理控制台亮色与暗色共用的基色（`baseColor`），并通过 CSS 语义变量驱动背景、前景、卡片、主色、muted、边框、侧栏等相关 token。暗色模式 MUST 不以蓝灰（slate）为主调；亮色与暗色 MUST 使用同一套基色，避免日夜切换出现「一套灰、一套蓝」的不一致。

#### Scenario: 暗色不发蓝
- **WHEN** 用户将主题切换为暗色（或系统偏好解析为暗色）并打开任意壳页
- **THEN** 页面主背景与主要表面色呈现中性灰，不以明显蓝灰为主调

#### Scenario: 亮暗共用同一基色
- **WHEN** 用户在亮色与暗色之间切换
- **THEN** 两侧均基于 neutral 语义 token，且主题切换能力（含偏好持久化）仍可用

#### Scenario: 基座配置与 token 对齐
- **WHEN** 开发者查看 `web/admin` 的 shadcn 基座配置与主题 CSS
- **THEN** `baseColor` 为 `neutral`，且亮/暗 CSS 变量与该基色一致

#### Scenario: 主表面中性不影响 chart 色相
- **WHEN** 主题已切换为 neutral，且页面使用 chart 语义色（`chart-1`…`chart-5`）
- **THEN** 主背景与主要表面仍为中性灰；chart 色板 MAY 保留色相（不要求无色）

```
