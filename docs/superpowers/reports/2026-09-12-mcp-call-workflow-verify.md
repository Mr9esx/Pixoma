# mcp-call-workflow 验证报告

- Change: mcp-call-workflow
- Date: 2026-09-12
- Mode: full
- Language: zh-CN
- Base: `ef01f1f7e7f06d084e1b7bec12b5fe26a5cf5e45` … `HEAD`

## 规模

任务 10、delta 4 个 capability、提交区间 133 文件 → full。

## Completeness

| 项 | 结果 |
|---|---|
| tasks.md 10/10 `[x]` | PASS |
| Superpowers plan 步骤勾选 | PASS |
| `botapp.RunCase` | PASS `internal/packaging/botapp/run_case.go` |
| `/mcp` `/sse` | PASS `internal/mcp/http.go` |
| 平台 `mcp`、无适配器、notify 跳过 | PASS `assembler.go` / `telegram.go` / `notify.go` |
| 每用户 Bearer + 后台生成/复制/轮换/删除 | PASS `mcp_users.go` + `mcp-users-section.tsx` |
| 架构文档 | PASS `overview.md` / `runtime.md` / `data-model.md` |

## Correctness（规格场景）

| 场景 | 证据 |
|---|---|
| Streamable 列出工具、无 create/patch Case | `TestMCP_StreamableHTTP_InitializeAndListTools` |
| 旧 SSE 可握手 | `TestMCP_LegacySSE_Initialize` |
| initialize 含 tools/resources/prompts/logging/completions/elicitation，无 sampling | 同上；`server.go` 未设 Sampling |
| 发起即返回 task_id | `TestMCP_RunWorkflow_ReturnsTaskID` |
| 仅本用户任务 | `TestMCP_ListTasks_OnlyOwnUser` |
| 缺必填不建任务 | `TestMCP_RunWorkflow_MissingRequired` |
| 付费用户不建任务 | `TestMCP_RunWorkflow_PaidUserDoesNotCreateTask` |
| 成功任务可读图 | `TestMCP_GetTask_ReadsSucceededImage` |
| 无 Bearer / cookie / 错 token / 停用渠道 → 401 | `TestMCP_RequiresBearer` |
| Streamable 会话绑用户 | `TestMCP_StreamableSessionRejectsOtherBearer` |
| SSE 会话绑用户 | `TestMCP_LegacySSESessionRejectsOtherBearer` |
| 生成/轮换/删除用户 | `TestMCPUser_MintAndRotate` |
| mint 失败回滚 | `TestMCPUser_MintFailureRollsBackUser` |
| 复制客户端配置 | `mcpClientConfigJSON` + 契约 `mcpCopyClient` / `mcpServers` / `Bearer` |
| MCP 非 admin ConfirmRun | `RunCase` 在 botapp，未挂 admin tasks ConfirmRun |
| IM 用户无管理创建入口 | 仅 `POST /channels/{id}/mcp-users` |

命令（本轮）：

- `go build ./...` exit 0
- `go test ./...` exit 0
- `go test ./internal/mcp/ ./internal/httpapi/channels/ ./internal/packaging/botapp/ ./internal/channels/application/ -count=1` 全部 ok

## Coherence

与 `design.md` / Design Doc 一致：执行门面在 botapp、发起即返回、`platform=mcp` 走 `channel_users`、每用户 Bearer、挂主进程 `/mcp`+`/sse`、不经 admin ConfirmRun。

Build 阶段 `review_mode: standard` 已审实现；Important 已修（会话绑用户、16MiB body、mint 回滚、复制配置、读图测试）。Verify 未复审未变 diff。

## 缺口

### SUGGESTION

- `comfy_mock` 出图在 MCP 测试里用「建任务 → 标 succeeded → 塞 blob → get_task/ReadResource」锁住读路径，没有从 `/mcp` 再跑一遍完整 Orchestrator。主链仍走既有 mock。
- 主进程 stdin 不是 MCP：架构上 MCP 只挂 HTTP，无 stdin 传输测试。
- 全量 `pnpm test` 另有一条既有契约（工作台 TopologyCard 顺序）与本 change 无关，未纳入。

### 审查已接受（非阻塞）

- 删渠道暂不级联清 token
- viewer 详情无明文
- Create/Delete MCP 用户本身不查角色（走管理端鉴权）

## 未在浏览器点过

本环境无后台浏览器自动化。MCP 用户表、复制配置、删除依赖契约测试 + 你本地打开消息平台详情看一眼。

## 结论

PASS。无 CRITICAL / IMPORTANT。可归档。
