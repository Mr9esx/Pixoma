# Brainstorm Summary

- Change: admin-web-console
- Date: 2026-08-09

## 确认的技术方案

方案 1：裁剪 satnaing/shadcn-admin 进 `web/admin` + 薄 `lib/api`（fetch + TanStack Query）对接 `VITE_ADMIN_API_BASE`。

- 包管理器：pnpm
- 无鉴权；去掉 Clerk/登录挡板/无关 demo
- 默认首页 Dashboard（中等深度：卡片 + 简单分布/趋势；列表 API 前端聚合；无新统计端点）
- 侧栏：Dashboard → 实例 → Case → Task → User → Session
- Case：尽量全结构化多段表单；User/Session 只读；Task 可取消；实例 CRUD + 观测入口
- 默认中文 + 中英双语齐全（壳子 + Dashboard + 五资源）+ 语言切换
- 不留独立前端 mock；不直连 DB；不依赖 channel/tg
- TG / Menu 落库管理本期不做
- **列表/详情布局：本期统一方案 B（左右分栏 Master–Detail）**；A–E 全文入库见 `docs/superpowers/specs/2026-08-09-admin-list-detail-layout-options.md` 与 `.comet/handoff/list-detail-layout-options.md`

## 关键取舍与风险

- 基座裁剪成本 vs 从零搭壳：选裁剪，换速度
- Dashboard「中等」与原 non-goal「复杂可视化大盘」冲突 → Spec Patch 放宽为允许中等 Dashboard，仍禁止实时大屏/告警
- Case 结构化表单工作量大；workflow 图仍可能需受控 JSON 区作为兜底
- 无 mock：本地必须起 admin-api，联调门槛略高但交付更实
- 方案 B：Case 长表单在右栏需可滚动；窄屏降级以免不可用

## 测试策略

- 本地：admin-api + `pnpm install && pnpm dev`（README 写清）手测主路径
- 前端：关键纯函数（菜单配置、i18n key、Dashboard 聚合）可单测；页面以联调验收为主
- 验收：默认进 Dashboard；侧栏顺序；中英切换；五资源基础管理；请求只打 admin-api

## Spec Patch

- `design.md` / proposal 非目标：取消「不做完整 i18n」；改写「复杂可视化大盘」为允许中等 Dashboard
- `admin-web-shell`：补 Dashboard 默认路由、侧栏入口、中英 i18n 与语言切换场景
- `admin-resource-pages`：补 Case 结构化表单主路径、双语页面文案覆盖（如需）
- Open Questions：包管理器定为 pnpm；不留前端 mock
