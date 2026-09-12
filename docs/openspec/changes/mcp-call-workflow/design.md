## Context

见 `proposal.md` 的 Why。Case CRUD 与校验已在 admin-api；执行链是 Session → ConfirmRun → Task → Edge → notify。MCP 需要跳过 IM 填表 Session，但仍走同一条 Task/Orchestrator，以便 `comfy_mock` 与调度不变。MCP 挂在主进程。调用方是 MCP 消息平台上的 `channel_users`，每用户一把 Bearer。

## Goals / Non-Goals

**Goals:**

- 主进程 Streamable HTTP + 旧 SSE 可调用已有工作流。
- MCP 作为消息平台；用户走允许/付费/拒绝；每用户一把后台签发的 Bearer。
- 与 admin 代跑禁令并存。

**Non-Goals:**

- 不在本期做 sampling / MCP Apps。
- 不把主进程 stdin 当 MCP。
- 不设计工作流可视化编辑。
- 不做 OAuth 授权码。
- MCP 平台不收 IM、不启 Bot。

## Decisions

### 1. 执行门面挂在 botapp 旁，不经 admin HTTP

- **选择**：内部 `RunCase(ctx, caseID, inputs, actor)`：校验 schema、写 blob、创建 Task，等价 ConfirmRun 成功之后的路径。MCP 工具调它后立刻返回 `task_id`；产物由 `list_tasks` / `get_task` / resource 再读。
- **理由**：admin-api 明确禁止 ConfirmRun；复用 HTTP 会把 AI 客户端绑到管理员 cookie。
- **备选**：MCP 假装成 TG 用户走 Session——还要假 chat id，notify 仍找 TG。

### 2. 发起即返回，不阻塞等出图

- **选择**：`run_workflow` 在 Task 创建并发布 `task.created` 后立刻返回 `task_id`（及 pending 等当前状态）。不等 succeeded/failed。查询用 `list_tasks`、`get_task`；图用 resource / `get_task` 再取。
- **理由**：Comfy 可能数分钟；MCP 工具阻塞会卡住对话。发起成功即可，调用方自己查。
- **备选**：工具内等到终态（已否定）。

### 3. 身份：MCP 消息平台上的 channel_users

- **选择**：`platform=mcp` 的消息平台实例；用户 `(channel_id, external_user_id)` 与 TG 用户同一张 `channel_users` 表。Bearer 解析到该用户。`run_workflow` 与 TG 相同：仅 **允许** 可跑；付费/拒绝不建任务。ChatID 用该 MCP 渠道地址，notify 对 platform=mcp 跳过。
- **理由**：用户页能管权限；多人各一把钥匙。
- **备选**：固定 `mcp:local` 一把总密钥（已否定）。绑到某个 TG 用户（身份搅在一起）。

### 4. 传输挂在主进程 HTTP

- **选择**：现有 `http.Server` 上 `/mcp`（Streamable HTTP）与 `/sse`（旧 SSE）。
- **理由**：MCP 与 Orchestrator 同进程；SSE 是用户明确要求。
- **备选**：仅 stdio 子进程——用户否定。

### 5. 每用户一把密钥，在 MCP 渠道详情签发

- **选择**：消息平台创建 MCP（无 Bot Token）。详情里生成用户（显示名）→ 建用户（默认允许）+ mint Bearer。密文按用户存表，不写 yaml、不进 `platform_settings` 总密钥。`/mcp` 始终校验 Bearer（含本机）。管理员 cookie 不够。停用渠道 → 该渠道用户全部 401。
- **理由**：对齐节点 token 的生成/复制/轮换，且能按人作废。
- **备选**：设置页一把总密钥（已否定）。OAuth（本期不做）。

## Risks / Trade-offs

- [查询负担] 发起不带图 → 客户端必须再 list/get。
- [媒体输入] MCP 传入图片需写入 blob → 路径或 base64，限制大小。
- [暴露面] 默认 `:8080` → 无匹配 Bearer 则 401；本机也不免检。
- [与飞书 change 并行] `ValidPlatform` 需加入 `mcp`；MCP 渠道 MUST NOT 走 TG 工厂。

## Migration Plan

- 无存量 MCP。回滚：停挂 MCP 路由、去掉 MCP 平台类型；IM 不受影响。

## Open Questions

无。
