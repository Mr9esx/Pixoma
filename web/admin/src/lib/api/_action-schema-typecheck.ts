// Type-level 编译期检查：v2 Action schema 必须收紧。
// 当 Action 类型包含 workflow_ids 或 mode 时，下面三行会编译失败（type 错），
// build 阶段（tsc -p tsconfig.app.json --noEmit）会捕获。
//
// 触发 build:cd web/admin && pnpm exec tsc -p tsconfig.app.json --noEmit
import type { Action } from './channel-menu'

// 必须: Action 有 workflow_id（单数字段）
type _HasWorkflowId = 'workflow_id' extends keyof Action ? true : never
const _hasWorkflowId: _HasWorkflowId = true
void _hasWorkflowId

// 必须: Action 没有 workflow_ids（旧 schema 残留会触发 type 错）
type _NoWorkflowIds = 'workflow_ids' extends keyof Action ? never : true
const _noWorkflowIds: _NoWorkflowIds = true
void _noWorkflowIds

// 必须: Action 没有 mode 字段
type _NoMode = 'mode' extends keyof Action ? never : true
const _noMode: _NoMode = true
void _noMode
