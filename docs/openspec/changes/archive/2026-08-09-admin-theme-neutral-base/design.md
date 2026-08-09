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
