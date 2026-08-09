# Brainstorm Summary

- Change: admin-theme-neutral-base
- Date: 2026-08-09

## 确认的技术方案

方案 A：手工将 `theme.css` 的 `:root` / `.dark` 语义色对齐官方 neutral OKLCH；`components.json` → `baseColor: "neutral"`；保留 `--sidebar: var(--background)` 等项目映射；将唯一硬编码 `bg-slate-950` 改为语义色。不依赖 shadcn `migrate base-color`（当前 CLI 无此迁移）。

## 关键取舍与风险

- CLI 无 base-color 迁移 → 手工贴官方 token，对照 docs 校验
- 保留 sidebar var 映射，避免侧栏视觉二次扰动
- chart-* 可保留官方彩色；主表面去蓝与 chart 解耦

## 测试策略

合同测试断言 `baseColor=neutral` 与暗色 `--background` chroma≈0；手工 light/dark 目视；既有 theme/shell 测试保持通过。

## Spec Patch

在 `admin-web-shell` delta 增加边界场景：主表面中性灰不要求 chart 色板无色相。
