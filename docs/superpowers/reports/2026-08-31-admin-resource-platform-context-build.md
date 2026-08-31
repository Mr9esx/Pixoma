# Admin Resource Platform Context — Build Report

## Verification

| Command | Result |
| --- | --- |
| `go test ./...` | PASS |
| `pnpm --dir web/admin exec vitest run src/lib/api/users.test.ts src/lib/api/tasks.test.ts src/lib/api/sessions.test.ts src/features/tasks/list-panel.contract.test.ts src/features/sessions/list-panel.contract.test.ts src/features/users/list-panel.contract.test.ts` | PASS |
| `pnpm --dir web/admin run build` | PASS |
| `pnpm --dir web/admin exec eslint src/features/tasks/list-panel.tsx src/features/tasks/detail-panel.tsx src/features/sessions/list-panel.tsx src/features/sessions/detail-panel.tsx src/features/users/list-panel.tsx src/features/users/detail-panel.tsx src/lib/api/types.ts src/lib/api/users.ts` | PASS，0 errors / 6 warnings |
| `comet classic openspec -- validate admin-resource-platform-context --type change --strict` | PASS |

## Review

- 使用独立的 Codex review 子进程审查 `4eb7e8ce096340490d757d017c3f7882dc1bcec6..HEAD`。
- 发现生产装配中 Session handler 缺少平台名称读取器，会显示空 `channel_name`；已在 `5f6c448` 修复。
- 复核确认 Task admin projection 使用 LEFT JOIN，孤儿 Task 保留主行并返回空平台/用户上下文。
- 复核确认 User API 与前端已移除 `tg_user_id`；搜索使用 `q`，精确定位可继续使用 `channel_id` / `external_user_id`。
- ESLint 的 6 个 warning 来自既有 fast-refresh export、TanStack Table memoization 和 User actions 依赖提示；本次无 error，风险接受。

## Scope Handling

- 工作区中来自其他 landing 任务的未提交改动保持未处理。
- 本 change 提交只精确暂存当前任务相关文件。

## Core Acceptance Check

- Task 列表返回并展示消息平台、关联用户、关联 Session；用户和 Session 可跳转详情。
- Session 列表和详情展示消息平台。
- User 列表和详情展示消息平台与平台无关外部 ID；不再展示 Telegram 专用字段。
