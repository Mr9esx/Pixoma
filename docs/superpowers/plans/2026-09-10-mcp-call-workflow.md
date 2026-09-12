---
change: mcp-call-workflow
design-doc: docs/superpowers/specs/2026-09-10-mcp-call-workflow-design.md
base-ref: ef01f1f7e7f06d084e1b7bec12b5fe26a5cf5e45
---

# MCP 调用已有工作流 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans or superpowers:subagent-driven-development. Steps use checkbox (`- [x]`) syntax.

**Goal:** 主进程 HTTP 提供 Streamable HTTP 与旧 SSE，MCP 客户端能列出并调用已启用工作流，在 `comfy_mock` 下拿到图。

**Architecture:** `botapp.RunCase` 与 `ConfirmRun` 共享 stage/create/`task.created`，不经填表 Session。`internal/mcp` 挂 `/mcp` `/sse`。消息平台 `mcp` + `channel_users`；每用户 Bearer。notify 对 platform=mcp 跳过。

**Tech Stack:** Go、现有 botapp/Task/blob、官方或等价 MCP Go SDK（须覆盖 Streamable HTTP + 旧 SSE）。

## Global Constraints

- 产物语言 zh-CN；界面无说明性文案。
- 不创建/修改 Case；不声明 sampling；不占用主进程 stdin。
- 后台签发每用户 MCP Bearer；`/mcp` 始终校验；不要求管理员 cookie；不写 yaml。
- `comfy_mock: true` 时后续查询须能读到图。
- 架构改动同步 `docs/architecture/overview.md`、`runtime.md`、`data-model.md`。

## File map

- Create: `internal/packaging/botapp/run_case.go`、`run_case_test.go`
- Create: `internal/mcp/`（server、tools、resources、auth、http 挂载）
- Create: `mcp_user_tokens` 持久化
- Modify: `internal/channels`（`PlatformMCP`；工厂跳过 IM 适配器；notify 按 platform=mcp 跳过）
- Modify: `internal/httpapi/channels`、`internal/httpapi/users`（生成用户 / token）
- Modify: `internal/packaging/botapp/facade.go`（RunCase）
- Modify: `apps/pixoma/internal/application/app.go`（挂路由）
- Modify: `web/admin` 创建 MCP 平台 + 渠道详情用户密钥（不走设置页总密钥）
- Modify: `docs/architecture/overview.md`、`runtime.md`、`data-model.md`

---

### Task 1: RunCase 无会话执行门面

**Files:**
- Create: `internal/packaging/botapp/run_case.go`
- Modify: `internal/packaging/botapp/facade.go`
- Test: `internal/packaging/botapp/run_case_test.go`（可复用 `confirm_run_test.go` 的 memCases / blob / publisher）

**Interfaces:**
- Consumes: `Facade.Cases`、`Validator`、`Tasks`、`Blob`、`Publisher`、`NewTaskID`、`Now`；`catalogdomain.InputValue`
- Produces:

```go
type RunCaseCmd struct {
    CaseID sharedkernel.CaseID
    Inputs []catalogdomain.InputValue
    Actor ChatID // FormatChatID({ChannelID: mcpChannelID, ExternalChatID: user.ExternalUserID})
    UserID string // channel_users.id；Access 非 always_allowed 则拒绝
}
type ConfirmRunResult struct { TaskID sharedkernel.TaskID; Status sharedkernel.TaskStatus }
func (f *Facade) RunCase(ctx context.Context, cmd RunCaseCmd) (*ConfirmRunResult, error)
```

- SessionID：写入一条 `submitted` 会话（UserID=该 MCP 用户），不走填表采集。
- 禁用 Case → `catalogdomain.ErrDisabled`；校验失败不 Create Task。

- [x] **Step 1: 写失败测试** `TestRunCase_CreatesPendingAndPublishes`：内存 Case 已启用、合法文本、允许用户；断言 Task pending、ChatID 为该 MCP 渠道、发布 `TopicTaskCreated`。`TestRunCase_RejectsMissingRequired`。`TestRunCase_RejectsDeniedUser`。`TestRunCase_WorksWithoutTelegram`。

- [x] **Step 2: 跑测试确认失败**

```bash
go test ./internal/packaging/botapp/ -count=1 -run 'TestRunCase_'
```

Expected: FAIL 找不到 `RunCase`

- [x] **Step 3: 实现** 从 `ConfirmRun` 抽出 `stageInputs` + Create + Publish；`RunCase` 跳过 Session 读写。

- [x] **Step 4: 测试通过** 同上命令 Expected: PASS。`TestConfirmRun_*` 仍绿。

---

### Task 2: MCP 服务器挂主进程（传输 + tools/resources）

**Files:**
- Create: `internal/mcp/server.go`、`http.go`、`tools.go`、`resources.go`、`http_test.go`
- Modify: `apps/pixoma/internal/application/app.go`（`r.Mount("/mcp", ...)`、`r.Mount("/sse", ...)` 在 Gate 之后即可，因路径非 `/api/`）
- Test: 用 `httptest.Server` 打 initialize + tools/list；两条传输各一条。

**Interfaces:**
- Consumes: `RunCase`、`Cases.List/Get`、`Tasks.Get`、`blob.Store`
- Produces: HTTP handler；tools `list_workflows` `get_workflow` `run_workflow` `list_tasks` `get_task`
- Resource URI：`pixoma://case/{id}`、`pixoma://task/{id}/output/{n}`
- initialize capabilities：tools、resources、prompts、logging、completions、elicitation；**不**含 sampling
- SDK：优先 `github.com/modelcontextprotocol/go-sdk`；若无旧 SSE，用能同时提供 Streamable HTTP + SSE 的库，禁止自创 JSON-RPC 方言。
- `run_workflow`：调用 `RunCase` 后立刻返回 `task_id`。`list_tasks` 走当前用户 ChatID。`get_task` 校验归属。

- [x] **Step 1: 写失败测试** `TestMCP_StreamableHTTP_InitializeAndListTools`、`TestMCP_LegacySSE_Initialize`：断言 tools 名称集合、无 create/patch case。

- [x] **Step 2:** `go test ./internal/mcp/ -count=1` Expected: FAIL 无包或 404

- [x] **Step 3: 实现** server + 挂载；prompts 至少一条 `run_workflow` 模板；elicitation：缺必填且客户端声明 elicitation 时发 elicitation，否则 tool error。

- [x] **Step 4:** 测试 PASS。再跑 `go test ./internal/packaging/botapp/` 防回归。

---

### Task 3: MCP 平台、每用户 token、传输鉴权、notify 跳过

**Files:**
- Create: `internal/mcp/auth.go`、`auth_test.go`
- Modify: `internal/channels/domain/channel.go`（`PlatformMCP`）；工厂对 mcp 不装配适配器
- Modify: `internal/httpapi/channels`（创建 mcp 无 IM 凭证；`POST/DELETE .../mcp-users`）
- Modify: `internal/httpapi/users`（GET/rotate mcp-token）
- Modify: `internal/channels/application/notify.go`（渠道 platform=mcp 则跳过）
- Modify: `web/admin` 创建表单可选 MCP；渠道详情生成/复制/详情/重新生成/删除
- Test: 无 Bearer / 错 token / 仅 cookie / 渠道停用 → 401；rotate 只作废该用户；拒绝用户 run 不建任务。

- [x] **Step 1: 写失败测试** `TestMCP_RequiresBearer`、`TestMCPUser_MintAndRotate`、`TestNotify_SkipsMCPPlatform`、`TestCreateMCPChannel_NoIMAdapter`。

- [x] **Step 2:** `go test ./internal/mcp/ ./internal/httpapi/channels/ ./internal/httpapi/users/ ./internal/channels/application/ -count=1 -run 'TestMCP|TestNotify_SkipsMCP|TestCreateMCP'`

- [x] **Step 3: 实现** 平台类型、token 表、管理 API、Bearer 中间件、渠道详情 UI；notify 跳过。

- [x] **Step 4:** 测试 PASS。补 `comfy_mock`：RunCase 后任务标 succeeded 并放 blob，`get_task` 能读。

---

### Task 4: 架构文档

**Files:**
- Modify: `docs/architecture/overview.md`、`docs/architecture/runtime.md`、`docs/architecture/data-model.md`
- 外部入口增加 MCP（主进程 `/mcp` `/sse`）；平台 `mcp`；表 `mcp_user_tokens`；notify 对 mcp 渠道跳过。

- [x] **Step 1: 改文档** 与 overview 拓扑一致，不写长教程。
- [x] **Step 2:** 无对应单测；人工确认三文件提到 MCP 凭据且不把 stdin 写成传输。

## Spec coverage

- 主进程传输 / SSE 握手 → Task 2
- capability 声明与无创建工具 → Task 2
- 无会话出图 / 无 TG → Task 1 + 3
- MCP 平台 / 每用户 token / 始终 Bearer / 渠道详情 → Task 3
- admin 无 ConfirmRun → 不改 task admin HTTP（已有 spec）；实现不得新增 ConfirmRun 路由
