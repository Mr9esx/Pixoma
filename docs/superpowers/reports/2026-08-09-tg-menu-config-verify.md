# tg-menu-config 验证报告

- **日期**: 2026-08-09
- **Change**: `tg-menu-config`
- **分支**: `feature/20260809/admin-web-console`（`isolation: current`）
- **verify_mode**: `full`
- **language**: zh-CN
- **base-ref**: `6e0d1afba44e9ffabcfc621da960b691a0b21ab7`

## 结论

**PASS** — 实现与 proposal / design / delta specs / tasks 对齐；自动化回归通过；无 CRITICAL / IMPORTANT 未处理项。

## 规模与审查

| 项 | 结果 |
|---|---|
| Scale | full（15 tasks / 5 capabilities / ≫8 files） |
| Build 阶段 code review（standard） | 已完成；Important 已修；Minor 已记录接受理由 |
| Verify 增量 review | Build 之后主要为 docs/校验收紧；依赖边界与校验行为已用测试覆盖，无新增 CRITICAL |

## 检查项（完整验证）

| # | 检查 | 结果 |
|---|---|---|
| 1 | tasks.md 全部 `[x]` | OK |
| 2 | 符合 `design.md` 决策（单文档 Menu、动作模型、包边界、刷新读库、控制台入口） | OK；控制台采用单页编辑（design 允许） |
| 3 | 符合 Design Doc `docs/superpowers/specs/2026-08-09-tg-menu-config-design.md` | OK |
| 4 | 能力场景：持久化/种子/校验/reply_media/Bot 动作/admin GET-PUT/侧栏页 | OK（单测 + handler/channel 测试） |
| 5 | proposal 目标满足（落库、Bot 驱动、admin-api、web、非多租户） | OK |
| 6 | delta spec ↔ design 无矛盾 | OK |
| 7 | Design Doc 可定位 | OK |

## 自动化证据（本轮新鲜执行）

```text
go test ./internal/tgmenu/... ./internal/httpapi/tgmenu/... ./internal/channel/tg/... ./apps/admin-api/... -count=1
→ PASS

go build -o /dev/null ./apps/bot/cmd/comfyui-bot ./apps/admin-api/cmd/admin-api
→ PASS

cd web/admin && pnpm exec tsc -p tsconfig.json --noEmit
→ PASS

go list -deps ./apps/admin-api/cmd/admin-api ./internal/httpapi/tgmenu | rg channel/tg
→ 无匹配（admin-api / httpapi/tgmenu 不依赖 channel/tg）
```

## 安全与边界

- 无新增鉴权；与既有 admin-api 一致，README 已警示仅内网。
- 图片仅 http(s) URL；无上传。
- 空菜单 / 全禁用 / 重复 id / 重复 label 拒绝写入；Case 仓储瞬时错误不再伪装为 400。

## 手工联调（留档）

真机 TG 键盘与 `reply_media` 出图需本地 `make run-all` + Bot Token，本轮未跑。建议归档前抽查：

1. 控制台「TG 菜单」见种子六项并可保存  
2. `/start` 键盘文案随配置更新  
3. `reply_media` / `open_case` 点击行为符合配置  
4. Network 仅打 `VITE_ADMIN_API_BASE`

## Dirty worktree 说明（未纳入本 change）

工作区仍有与本 change 无关的未提交改动（如 `Makefile`/`README` 入口调整、`admin-shell-sidebar-footer`、部分 `web/admin` 壳层文件、`configs/admin-api.yaml`）。验证以已提交的 `tg-menu-config` 范围为准，未修改或丢弃这些文件。

## 接受的 Minor（自 build review）

- 种子 `btn-help` 为 `placeholder`（与计划一致；`/help` 仍保留帮助文案）
- `MainMenuRows` 保留作夹具
- 暂不加菜单 TTL 缓存
