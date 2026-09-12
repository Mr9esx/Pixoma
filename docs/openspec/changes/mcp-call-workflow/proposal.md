## Why

路线图要 MCP 调用工作流。今天没有 MCP server；真正跑 Case 只走 Telegram 对话里的 ConfirmRun，管理端还禁止代跑。客户端需要连上**已在跑的 Pixoma 主进程**，列出已有工作流、提交输入、拿到成品（至少图）。

## What Changes

- 主进程 HTTP 提供 MCP：**Streamable HTTP**（`/mcp`）与旧版 **SSE**（`/sse`）。JSON-RPC 生命周期完整；声明 tools / resources / prompts / logging / completions / elicitation。
- 新增 **无 IM 填表会话** 的执行门面：已有 Case + 输入 → Task，发起成功即返回 `task_id`；凭本用户任务列表或 id 查询，工具内不等出图。
- 新增消息平台类型 `mcp`：不收 IM、不启 Bot。用户走 `channel_users`（允许/付费/拒绝）。每用户一把后台签发的 Bearer。
- 管理端 ConfirmRun 禁令保持：MCP 不是 admin-api 上的代跑入口。
- `/mcp`、`/sse` 始终校验 Bearer（含本机），不认管理员 cookie，不写 yaml。

### 非目标

- 不通过 MCP 创建或修改 Case。
- 不声明 sampling、不做 MCP Apps。
- 不把主进程 stdin 当作 MCP 传输。
- 不替代管理端 CRUD。
- 不绕过 `comfy_mock` 开关。
- 不把 MCP 登记到公网 registry、不做 OAuth 授权码。
- MCP 平台不收消息、不启 IM 适配器。

## Capabilities

### New Capabilities

- `mcp-call-workflow`：主进程 MCP 传输与协议面、无填表会话执行、产物回传、MCP 用户与每用户 Bearer。

### Modified Capabilities

- `task-admin-api`：明确 MCP 执行入口与「admin 不得 ConfirmRun」并存，互不替代。
- `channel-management`：平台类型 `mcp`，无 IM 凭证与适配器。
- `user-admin-api`：允许在 MCP 平台上管理端创建用户并签发 token。

## Impact

- `internal/mcp` 挂到现有 `http.Server`；`botapp.RunCase`。
- `channels` 增加 `mcp`；工厂对 mcp 不装配 IM 适配器。
- 复用 `channel_users`、Task 创建与 `task.created`、blob。
- 每用户 token 密文表；MCP 渠道详情生成/复制/轮换。
- 架构：`overview.md`、`runtime.md`、`data-model.md`、`bounded-contexts.md`（若平台枚举在此）。
- Mock：`comfy_mock` 下后续查询仍须能读到图。
