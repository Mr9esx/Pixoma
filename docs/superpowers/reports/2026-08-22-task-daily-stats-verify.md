# 验证报告：task-daily-stats

- 日期：2026-08-22
- 变更：`05b221f..HEAD`（28 个文件，+3599/−103）
- 验证模式：full（任务 24、能力 2、变更文件 28）

## Summary

| 维度 | 状态 |
|---|---|
| Completeness | 24/24 任务完成；2 个能力规格需求全部实现 |
| Correctness | 全部需求有实现与测试映射；关键场景均有覆盖 |
| Coherence | 与 design.md / Design Doc 决策一致；无 spec 漂移 |

## 检查项

1. **tasks.md 全部完成**：24/24 `[x]`，无未勾选任务。
2. **实现符合 design.md 高层决策**：三张窄表 + orchestrator 终态写路径（old≠new 守卫 + 零值 completed_at 回退）、`STATS_TIMEZONE`（默认 Asia/Shanghai）、API from/to 零填充与成功率口径、backfill 绝对值覆盖、保留清理——均已实现。
3. **实现符合 Design Doc**（`docs/superpowers/specs/2026-08-21-task-daily-stats-design.md`）：数据模型、写路径、读路径、保留/backfill、前端范围选择与图表均按文档落地。
4. **能力规格场景**：
   - 按天统计持久化 / 重复终态不双计 / 取消计入：`taskstats` 仓储 + orchestrator 测试覆盖；
   - daily 零填充 / 空数据 200 / 非法范围 400 / 成功率公式与 null：handler 测试覆盖；
   - 错误码 Top-N / 每节点负载 + total / 仅终态任务 / 空数据降级：仓储与 handler 测试覆盖；
   - 历史数据补齐（backfill 幂等）：命令实现 + 临时库端到端冒烟验证；
   - Dashboard 聚合信息 / 失败隔离 / 不受样本限制 / 日期范围选择：前端组件、合同测试与 i18n 覆盖。
5. **proposal.md 目标满足**：Dashboard 任务统计改为全量按天聚合，不再受 limit=200 样本限制；统计接口与统计表落地。
6. **delta spec 与 Design Doc 无矛盾**：Spec Patch（空白天零填充、成功率口径、仅终态任务）已同步写入 Design Doc 决策与实现。
7. **Design Doc 可定位**：`docs/superpowers/specs/2026-08-21-task-daily-stats-design.md` 存在且与当前 change 关联（frontmatter `comet_change: task-daily-stats`）。

## 验证证据

- `go build ./...`：exit 0
- `go test ./...`：exit 0，无 FAIL
- `pnpm -C web/admin tsc -b`：exit 0
- `pnpm -C web/admin vitest run`：51 文件 / 235 测试全部通过
- 端到端冒烟：临时库 seed 4 条终态任务 → backfill → `/api/v1/stats/tasks/daily`（08-21 processed=4、success_rate=0.6667）、`/errors`（timeout=1）、`/edges`（gpu-1=2、gpu-2=1、total=3）、非法范围 400，全部符合预期
- 记录于 `.comet.yaml`：`verify` command-check exit=0

## 工作区归因说明

`git status` 中其余未提交改动（topic-routing 归档删除、task-flow-editor / quick-config / channel / cases 相关文件等）属于其它 active change，与本 change 无关；按 dirty-worktree 协议保留不处理，未纳入本 change 验证范围。

## 代码评审（去重说明）

build 阶段已按 `review_mode: standard` 完成最终代码审查并记录于 `review-notes.md`（Important 发现——聚合别名 `count` 与 SQL 函数名歧义——已修复；Minor 项已记录接受理由）。verify 阶段聚焦 build 之后新增改动（backfill 零值归天修复、别名修复、产物提交），经内联复核无 CRITICAL / IMPORTANT 问题。

## Issues

无 CRITICAL。无 WARNING。

SUGGESTION（记录，不阻塞）：
- 管理端统计接口与既有 admin-api 一致无鉴权，属既有部署约束，README 已警示仅本机/内网。

## Final Assessment

全部检查通过，无关键问题。Ready for archive。

## 补充：运行期崩溃修复（verify-fail → build → 复验）

归档确认期间用户反馈 Dashboard 运行期崩溃：`TypeError: Cannot read properties of undefined (reading 'success_rate')`（`task-stats-section.tsx`）。

**根因**：开发前端（Vite 代理 → `127.0.0.1:8082`）连接的是改动前构建的 `pixoma` 旧二进制；旧二进制没有 `/api/v1/stats/*` 路由，`webembed` 的 SPA fallback 对未匹配路径返回 index.html（HTTP 200），`apiFetch` 把 HTML 字符串当作数据返回，`daily.data?.summary.success_rate` 在 `summary` 缺失时抛异常。

**修复**：
- 前端：新增 `task-stats-parse.ts`（`pickDays` / `pickSuccessRate` / `pickErrorItems` / `pickEdgeItems`）对畸形/非 JSON 载荷防御性解析，组件不再因缺字段崩溃，降级为空态；配套 5 个单测。
- 后端：`webembed.Handler` 对 `/api/*` 未匹配路径返回 JSON 404（不再回退 SPA index.html），避免旧/未知 API 路径伪装成 200 HTML；配套测试。

**复验**：`verify-fail`（第 1 次）→ build 修复提交 → `go build ./...`、`go test ./...`、`pnpm tsc -b`、`pnpm vitest run`（52 文件 / 240 测试）全部通过；build 与 verify command-check 均已重新记录 exit=0。
