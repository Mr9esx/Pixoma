# Verification Report: admin-resource-platform-context

## Summary

| 维度 | 状态 |
| --- | --- |
| Completeness | 17/17 tasks 完成；4 个 capability delta 全部覆盖 |
| Correctness | 15/15 scenarios 已核对；关键行为有后端与前端契约测试 |
| Coherence | API 投影、用户信息展示与资源页结构均符合 design 决策 |

**结论：所有检查通过，可以进入归档确认。**

## Verification Checks

### Completeness

- `tasks.md` 中 15 项任务全部勾选，无未完成任务。
- `proposal.md` 声明的 Task、Session、User 三类资源上下文增强均已在实现中落地。
- 4 个 modified capabilities 均存在对应 delta spec：
  - `task-admin-api`
  - `session-admin-api`
  - `user-admin-api`
  - `admin-resource-pages`

### Correctness

- Task API 使用 `tasks.session_id = sessions.id` 关联 Session，再补齐 `channel_id`、`channel_name`、`user_id` 和用户资料投影；孤行任务保留主记录并返回空上下文。实现见 `internal/runtime/infrastructure/persistence/gorm_task_admin.go`。
- Task 后端测试覆盖列表、详情、平台过滤、用户上下文和孤行任务。见 `internal/httpapi/tasks/handler_test.go`。
- Session API 列表与详情都返回 `channel_id`，详情补充 `channel_name`；后端测试覆盖平台过滤和详情展示。见 `internal/httpapi/sessions/handler.go`、`internal/httpapi/sessions/handler_test.go`。
- 2026-08-31 复核调整后，Session API 列表和详情补充嵌套 `user` 投影，前端用「关联用户」替代裸 User ID，并支持跳转用户详情。实现见 `internal/conversation/infrastructure/persistence/gorm_session_admin.go`。
- Task、Session 和 User 的消息平台均显示 `channel_name`，不再把 `channel_id` 作为可读展示值；缺失时显示空态。
- Task、Session 和 User 的操作列均固定在右侧，并显示「操作」列名；用户列表将「使用权限」从隐藏菜单改为独立可见按钮。
- Task、Session 和 User 的时间字段统一通过 `web/admin/src/lib/format.ts` 按浏览器本地时区格式化；零值时间显示为空态。
- 2026-09-01 复核调整后，Task 详情仅对 pending / queued 显示取消按钮；Task、Session 和 User 详情 Modal 共用 504px 宽度与同一内容内边距规则，与计算节点新建 Modal 对齐。
- User API 返回内部用户 ID、`channel_id`、`external_user_id` 和用户资料；查询支持平台无关 `q`、`channel_id` 和 `external_user_id`，且 DTO 与契约中不再暴露 `tg_user_id`。见 `internal/httpapi/users/handler.go`、`internal/identity/infrastructure/persistence/gorm_user.go`。
- User 后端测试同时断言 `channel_id` / `external_user_id` 存在和 `tg_user_id` 不存在。见 `internal/httpapi/users/handler_test.go`。
- 前端 Task 列表展示消息平台、关联用户和 Session，并提供用户与 Session 详情跳转；详情页同步展示上下文。见 `web/admin/src/features/tasks/list-panel.tsx`、`web/admin/src/features/tasks/detail-panel.tsx`。
- 前端 Session 列表与详情展示消息平台。见 `web/admin/src/features/sessions/list-panel.tsx`、`web/admin/src/features/sessions/detail-panel.tsx`。
- 前端 User 列表使用「消息平台」和统一「用户信息」列，优先展示用户名或姓名，并以次级文本展示通用外部用户标识；前端契约测试禁止 `tg_user_id` 回流。见 `web/admin/src/features/users/list-panel.tsx`、`web/admin/src/features/users/list-panel.contract.test.ts`。
- 中英文 i18n 同步新增平台、用户信息和外部用户标识文案，并为空值保留空态展示。

### Coherence

- 实现采用 design.md 中选定的只读关联投影，没有新增冗余平台列，也没有数据库迁移。
- Task 一次列表查询完成平台与用户上下文投影，符合避免前端逐行补查的目标。
- Session 直接暴露既有持久化平台字段；User 抽象为通用外部身份，符合多消息平台模型。
- 三个资源页沿用 DataTable / Master–Detail 模式，列名与值使用 i18n，字段缺失显示 `—`，未将平台硬编码为 Telegram。

## Validation Evidence

| 命令 | 结果 |
| --- | --- |
| `go test ./...` | PASS |
| 受影响 Vitest 契约测试（7 个文件，21 个测试） | PASS |
| `pnpm --dir web/admin run build` | PASS |
| `openspec validate admin-resource-platform-context --strict` | PASS |
| 受影响前端文件 ESLint | PASS（0 errors） |

## Issues

### CRITICAL

无。

### WARNING

- 无阻塞问题。
- 受影响前端文件有 6 个既有 lint warning，主要来自 Fast refresh、TanStack Table 兼容性提示和既有的 hook dependency 提示；未新增 lint error，不影响本变更验收。
- 全量 ESLint 中存在 `web/admin/src/features/quick-config/quick-config-flow.tsx` 的 duplicate import error；该文件不在本 change 的 diff 范围内，属于既有独立问题，不在本次 verify 中修复。

### SUGGESTION

- 后续可在独立技术债任务中统一处理 Admin 资源列表的 TanStack Table lint 提示。

## Final Assessment

本 change 满足 proposal、delta specs、design 和 tasks 的验收要求。实现、测试与文档没有发现漂移；验证通过，可等待用户明确确认后归档。
