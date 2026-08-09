## Context

参见 `proposal.md`。`web/admin` 现为占位；基座选用 satnaing/shadcn-admin（Vite + React + TanStack Router + shadcn）。后端分属 foundation 与 resources-api（均已归档可用）。

## Goals / Non-Goals

**Goals:**
- 可运行脚手架与菜单驱动导航
- 默认 Dashboard（中等：卡片 + 简单分布）+ 五类资源页的基础信息架构与交互
- 对接真实 admin-api；中英双语齐全 + 语言切换
- 包管理器 pnpm

**Non-Goals:**
- 登录鉴权；实时监控大屏/告警；TG/Menu 落库管理；独立前端 mock；组件库重写；前端直连 DB

## Decisions

1. **落位 `web/admin`**，以 shadcn-admin 为起点裁剪 demo 页，保留布局/侧栏模式；**pnpm**。  
2. **菜单配置化**：Dashboard → 实例 → Case → Task → User → Session；默认路由 `/` = Dashboard。  
3. **数据层**：TanStack Query + `lib/api` 指向 `VITE_ADMIN_API_BASE`；**不留**前端 mock。  
4. **页面深度**：五资源列表/详情统一 **Master–Detail（方案 B）**；实例/Case 右栏偏完整表单（Case 结构化多段）；User/Session 只读；Task 右栏可取消；Dashboard 用 list API 前端聚合。备选 A–E 见 `docs/superpowers/specs/2026-08-09-admin-list-detail-layout-options.md`。  
5. **无鉴权**：不保留 Clerk 强制登录；移除或旁路模板鉴权挡板。  
6. **i18n**：默认中文，本期中英两套齐全（壳子 + Dashboard + 五资源）+ 顶栏切换。

## Risks / Trade-offs

- [基座裁剪成本] → 锁定版本、只删鉴权与无关 demo。  
- [Dashboard 计数受 list limit 影响] → UI/文档标明样本口径；统计 API 另开。  
- [Case 表单工作量大] → workflow/schema 允许受控 JSON 兜底。  
- [无 mock 联调门槛] → README 写清先起 admin-api。

## Migration Plan

1. 初始化前端并跑通空壳 + i18n + Dashboard。  
2. 菜单 + 五页骨架。  
3. 逐资源对接 API。  
4. 文档：安装、环境变量、与 admin-api 联调。

## Open Questions

- （已关闭）包管理器：pnpm  
- （已关闭）开发期 mock：不留  
- （已关闭）TG/Menu：本期不做  
