## Purpose

让 MCP 客户端连上已在跑的 Pixoma 主进程，列出并调用已启用工作流，拿到成品（至少图片），不必经过 Telegram 或管理端代跑。

## ADDED Requirements

### Requirement: 主进程 MCP 传输
系统 MUST 在 Pixoma 主进程的 HTTP 服务上提供 MCP：**Streamable HTTP**（`/mcp`）与规格中的旧版 **HTTP+SSE**（`/sse`）。JSON-RPC 生命周期 MUST 支持 initialize、ping、标准错误与通知。MUST NOT 把主进程标准输入当作 MCP 传输。MUST NOT 提供创建或修改 Case 的工具。MUST NOT 把该服务登记到公网 MCP registry。

#### Scenario: Streamable HTTP 可列出工作流
- **WHEN** 客户端对主进程 `/mcp` 完成 initialize 后调用 tools/list
- **THEN** 返回工具列表，其中含列出与调用已启用工作流的工具，不含创建或 PATCH Case 的工具

#### Scenario: 旧版 SSE 可握手
- **WHEN** 客户端按旧 HTTP+SSE 连接 `/sse` 并完成 initialize
- **THEN** 连接成功且可调用 tools/list

#### Scenario: 主进程 stdin 不是 MCP
- **WHEN** Pixoma 作为守护进程运行
- **THEN** 写入进程标准输入 MUST NOT 被解释为 MCP JSON-RPC

### Requirement: 声明的 capability 做实
`initialize` 的 capabilities MUST 包含 tools、resources、prompts、logging、completions、elicitation，且对应方法可调用。MUST NOT 声明 sampling。缺必填输入时，支持 elicitation 的客户端 MUST 能被要求补字段；不支持时 tools/call MUST 返回可理解校验错误且不创建成功终态任务。

#### Scenario: 产物可作为 resource 读取
- **WHEN** 任务已成功且输出含图片
- **THEN** 客户端能通过 resource URI 或 get_task 取得该图

#### Scenario: 无创建工具
- **WHEN** 客户端查看工具列表
- **THEN** 不存在用于创建或 PATCH Case 的工具

### Requirement: 无会话调用并取回产物
系统 MUST 接受工作流 id 与符合 schema 的输入，创建 Task 后立刻返回 `task_id`，MUST NOT 在发起工具内等待任务终态。调用方 MUST 能列出**当前 Bearer 对应用户**的任务，并凭 id 读取状态与终态产物（至少图片字节或可打开的资源）。MUST NOT 要求存在 Telegram 会话。`comfy_mock` 为真时 MUST 仍能把任务跑到出图，供后续查询读到。

#### Scenario: 发起成功即返回 task_id
- **WHEN** 权限为允许的 MCP 用户对已启用工作流提交合法输入
- **THEN** 调用在 Task 已创建并入队后返回 `task_id`，不必等到 succeeded/failed，且无需任何 IM 投递

#### Scenario: 可列出并查询本用户任务
- **WHEN** 客户端调用 list_tasks 或 get_task
- **THEN** 仅返回该 Bearer 对应用户的任务；已成功的任务可取到产物（resource 或字节）；MUST NOT 返回其他 MCP 用户的任务

#### Scenario: 缺必填输入被拒
- **WHEN** 客户端省略必填输入且未完成 elicitation
- **THEN** 不创建成功终态任务，并返回可理解的校验错误

#### Scenario: 非允许权限不建任务
- **WHEN** 该 MCP 用户权限为付费或拒绝
- **THEN** 不创建任务，并返回可理解的权限错误

### Requirement: MCP 调用方身份
MCP 调用 MUST 使用所属 MCP 消息平台上的 `channel_users` 作为归属。Task 的 ChatID MUST 属于该 MCP 渠道，MUST NOT 复用 Telegram chat id。终态 MUST 能被 MCP 读取，MUST NOT 仅依赖 Telegram notify。IM notify 对 MCP 渠道 MUST 跳过。

#### Scenario: 无 TG 通道也能完成
- **WHEN** 环境中没有启用的 Telegram 消息平台，但存在启用的 MCP 消息平台与允许用户
- **THEN** MCP 调用仍能创建 Task 并在成功后可供查询产物

### Requirement: 每用户一把 Bearer
系统 MUST 在 MCP 消息平台详情提供：生成用户（显示名）、复制该用户密钥与客户端配置、查看详情、重新生成该用户密钥、删除该用户。凭据 MUST 以 AES-GCM 密文按用户存储，MUST NOT 以 yaml 或 `platform_settings` 单把总密钥为主路径。明文 MUST 仅通过管理 API 返回给 admin/operator；viewer MUST NOT 获得明文。重新生成 MUST 立刻使该用户旧密钥失效，MUST NOT 影响其他用户。删除 MUST 立刻使该用户密钥失效并移出名单，MUST NOT 通过该入口删除非 MCP 消息平台用户。MUST NOT 做 OAuth 授权码。

#### Scenario: 无用户或无匹配 Bearer 被拒
- **WHEN** 没有启用的 MCP 平台、没有用户、或请求未带匹配 Bearer
- **THEN** 对 `/mcp` 与 `/sse` 的请求被拒绝，且不执行工作流

#### Scenario: 生成用户后可复制
- **WHEN** 管理员在 MCP 消息平台上生成用户
- **THEN** 管理 API 返回明文 token；admin/operator 之后仍可再取该用户明文；后台可复制客户端配置（当前 origin + `/mcp` + Bearer）

#### Scenario: 删除用户后凭据失效
- **WHEN** 管理员删除某 MCP 用户
- **THEN** 该用户从渠道名单消失，其 Bearer 立刻被拒绝

### Requirement: MCP 传输始终校验 Bearer
`/mcp` 与 `/sse` MUST 将 Bearer 解析为唯一 MCP 用户：所属渠道 platform 为 `mcp` 且已启用。包括监听回环时。MUST NOT 用管理员 cookie 替代 Bearer。Bearer 缺失、错误、用户不存在或渠道停用时 MUST 拒绝且不执行工作流。

#### Scenario: 无 Bearer 被拒
- **WHEN** 请求未带有效 MCP Bearer（即使来自 127.0.0.1，或带有管理员 cookie）
- **THEN** MCP HTTP 请求被拒绝，且不执行工作流

#### Scenario: 正确 Bearer 且无管理员 cookie
- **WHEN** 客户端不带管理员会话、但携带某启用 MCP 用户的有效 Bearer 访问 `/mcp`
- **THEN** initialize 不被 setup Gate 以未登录为由拒绝，且后续工具以该用户身份执行
