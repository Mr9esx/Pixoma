---
comet_change: mcp-call-workflow
role: technical-design
canonical_spec: openspec
archived-with: 2026-09-12-mcp-call-workflow
status: final
---

# MCP 调用已有工作流 深度技术设计

Canonical 行为见 OpenSpec delta。高层见 change 内 `design.md`。

## 1. 目标

Pixoma 主进程对外提供完整 MCP 传输，让任意 MCP 客户端列出并调用已启用工作流、取回成品图。不经过 Telegram，也不走管理端 ConfirmRun。

## 2. 现状

- 无 MCP 实现、无依赖。
- 执行链：`botapp.ConfirmRun` 读 Session（须 `confirming`）→ 校验 → stage blob → `Tasks.Create` → `TopicTaskCreated`。
- HTTP 已在主进程：`http_addr`（默认 `:8080`）。`setup.Gate` 只拦 `/api/`；`/healthz`、`/agent/`、非 `/api/` 路径放行。
- 管理 API 明确禁止代用户 ConfirmRun。

## 3. 模块切分

```text
internal/mcp/                 JSON-RPC、传输、Bearer→用户、工具/资源
internal/httpapi/channels     MCP 平台创建；详情生成用户 / 轮换 token
internal/packaging/botapp     RunCase（无填表 Session）
apps/pixoma/.../application   挂 /mcp /sse；mcp 平台不装配 IM 适配器
web/admin                   消息平台：MCP 类型 + 用户密钥
```

不新建写侧限界上下文。Case 仍只由 admin CRUD 写。

## 4. 传输

挂在**已有** `http.Server`，不另开端口。

| 路径 | 传输 | 谁用 |
|---|---|---|
| `POST/GET /mcp` | 现行 Streamable HTTP（POST 消息，可选 SSE 流） | 新客户端 |
| `/sse` + 配套 POST | 规格中的旧 HTTP+SSE | 尚未改 Streamable 的客户端 |

不把 MCP 接到主进程 stdin。守护进程的 stdin 不是 JSON-RPC 通道。

实现优先官方 `github.com/modelcontextprotocol/go-sdk`；若其 Streamable HTTP / 旧 SSE 不齐，改用能覆盖这两条传输的 Go 库，协议消息不得手搓一套不兼容方言。

## 5. 协议 capability

`initialize` 声明：

- `tools`（含 listChanged：Case 启停后可通知）
- `resources`（Case 文档、任务产物）
- `prompts`
- `logging`
- `completions`
- `elicitation`

**不声明** `sampling`、MCP Apps。客户端问未声明方法时按 JSON-RPC `method not found` 返回。

工具（均不得创建/修改 Case）：

- `list_workflows`
- `get_workflow`（入参 schema）
- `run_workflow`（创建 Task 后立刻返回 `task_id`，不等终态）
- `list_tasks`（当前 Bearer 用户的任务）
- `get_task`（仅本用户；成功则可取产物）

资源 URI：

- `pixoma://case/{id}` 入参与说明
- `pixoma://task/{id}/output/{n}` 产物图

缺必填：先 elicitation；客户端不支持 elicitation 时 `tools/call` 返回可理解的校验错误，不创建成功终态 Task。

## 6. 执行数据流

```text
MCP tools/call run_workflow
        │
        ▼
botapp.RunCase(caseID, inputs, actor)
  先查 channel_users.Access，非 always_allowed 则拒绝
  校验 Document + 已启用
  图片入参写入 blob
  Tasks.Create(pending)  （归属该 MCP 用户；写 submitted 会话；ChatID 为该 MCP 渠道，不得假造 TG chat）
  Publish TopicTaskCreated
        │
        ▼
立刻返回 task_id（pending）；工具结束，不阻塞
        │
        ▼
同进程 Orchestrator → Edge → Comfy（后台跑；comfy_mock 可走通）
        │
        ▼
之后：list_tasks / get_task / resources/read
成功：产物图（resource 或 blob）
失败：get_task 带任务错误信息
```

`Task.ChatID` 为该 MCP 渠道上的地址。IM notify：渠道 `platform=mcp` 则跳过。MCP 只读库和 blob。`list_tasks` / `get_task` 不得跨用户。

`RunCase` 与 `ConfirmRun` 共享 stage + create + publish；ConfirmRun 继续走 Session。禁止把 RunCase 接到 `POST /api/v1/tasks`。

## 7. 身份与鉴权

### 7.1 消息平台与用户

MCP 是消息平台类型 `mcp`，与 Telegram 并列出现在「消息平台」。

- 创建：名称；**无** Bot Token。不启动 IM 适配器，探测不打外部 IM。
- 用户：`channel_users` + `channel_user_external_identities`（`channel_id` = 该 MCP 渠道，`external_user_id` 系统生成）。
- 新增用户只能后台做（没有入站消息）。显示名写入 `username`。新用户 Access = `always_allowed`（刚发钥匙）。之后在用户页改允许/付费/拒绝；**付费/拒绝与 TG 一样不能跑工作流**。
- 停用该渠道：其上所有 Bearer 401。可有多条 MCP 渠道；token 全局唯一。

### 7.2 每用户 token

存在独立表（如 `mcp_user_tokens`）：`user_id`、`token_hash`（查找）、`token_cipher`（AES-GCM，与节点 token 同套密钥）。不写 yaml，不进 `platform_settings` 总密钥。

MCP 渠道详情：

| 动作 | 行为 |
|---|---|
| 生成用户 | 填显示名 → 建用户 + mint；响应带明文 |
| 复制 | 该用户密钥全文 + 客户端 JSON（origin + `/mcp` + Bearer） |
| 重新生成 | 只作废这一人的旧密钥 |
| 详情 | 用名单上的字段看该用户，不调运营用户详情接口 |
| 删除 | 删用户与 token；旧 Bearer 立刻 401 |

- GET 明文：admin/operator；viewer 无明文。
- 轮换 / 删除：admin/operator；viewer 403。
- 界面：`SecretInput`、复制、详情弹层、`ConfirmDialog`（「重新生成 MCP token？」「旧 token 立刻作废。」／「删除「{{name}}」？」「Token 立刻作废。」）。无说明性文案。

API（示意）：

```text
POST /api/v1/channels/{id}/mcp-users     { "name": "我的 Cursor" }
     → 201 { user, token }                 // 渠道必须是 mcp
DELETE /api/v1/channels/{id}/mcp-users/{userId}
     → { "deleted": true }                 // 渠道必须是 mcp；用户必须属于该渠道
GET  /api/v1/users/{id}/mcp-token
     → { "token": "<plain>" }             // admin/operator；viewer 无 token
POST /api/v1/users/{id}/mcp-token/rotate
     → { "token": "<plain>" }
```

Mint 对齐 `edge.MintToken`。

### 7.3 `/mcp` 怎么验

`/mcp`、`/sse` 不在 `/api/` 下，Gate 不拦。

1. 取 Bearer；缺失 → 401。
2. 用 hash 找到 token 行 → 用户 → 渠道。渠道不是已启用的 `mcp` → 401。
3. `subtle.ConstantTimeCompare` 核对明文。
4. **本机同样要 Bearer**。管理员 cookie 不能代替。
5. 通过后 JSON-RPC 以该用户执行。`run_workflow` 再查 Access，非允许则不建任务。

不做公网发现、OAuth、公网 registry。

## 8. 测试

- `RunCase`：合法输入 + 允许用户 → 创建 Task；缺必填 / 非允许不建任务。
- 协议：initialize 声明的 capability 与上表一致；`tools/list` 无 create/patch Case。
- 传输：Streamable HTTP 与旧 SSE 各一条握手 + `tools/list`。
- 无启用 Telegram 通道时，启用 MCP + 允许用户仍能创建 Task。
- HTTP：无用户 / 错 Bearer / 仅 cookie / 渠道停用 → `/mcp` 401；正确 Bearer 无 cookie → initialize 成功。
- 管理：MCP 平台无 IM 凭证可建；生成用户写入 token 表；rotate 后旧 Bearer 401、其他用户不受影响；viewer 无明文。
- 权限：拒绝/付费用户 Bearer 仍可握手，但 `run_workflow` 不建任务。
- `GET/POST /api/v1/tasks` 仍无 ConfirmRun。

## 9. 风险

- 默认绑所有网卡 → 无匹配 Bearer 则拒绝；本机也不免检。
- 工厂若仍无视 platform 会把 MCP 渠道当 TG 去连 → 装配时 mcp MUST 跳过适配器。
- 发起不带图 → `list_tasks` + `get_task` 必须按用户隔离。
- GET token 对 admin 回说明文；viewer 必须剥掉。
- 入站图片：本机路径或 base64，限制大小。
- SDK 传输覆盖不全 → 集成测试锁传输行为。

## 10. 架构文档（实现期）

`docs/architecture/overview.md`、`runtime.md`、`data-model.md`：MCP HTTP 入口；平台类型 `mcp`；`mcp_user_tokens`；notify 对 mcp 渠道跳过。
