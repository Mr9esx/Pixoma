# mcp-caller-skill Specification

## Purpose
给支持 MCP 连接器的聊天里的 Agent 一份可下载 skill，并让 MCP 失败文案足够指导 AI 排除「没配连接器 / 没权限 / 缺字段」等问题。
## Requirements
### Requirement: 一句提示词即可安装 skill
仓库文档 MUST 提供一句完整提示词，其中含 skill 包的公开 URL（`https://pixoma.miaoplus.com` 下的稳定路径）。提示词 MUST NOT 包含 MCP Bearer、token 或密钥。提示词 MUST 写明：用户的聊天产品需要能配置 MCP 连接器。

#### Scenario: 用户只贴提示词
- **WHEN** 用户把文档中的安装提示词交给 Agent，且本机尚未安装该 skill
- **THEN** Agent 从该 URL 取得 skill 包并写入本机 skills 目录，且对话中不出现 Bearer

#### Scenario: 提示词不含密钥
- **WHEN** 读者打开仓库文档中的安装提示词
- **THEN** 文本中没有 token、Bearer 或可当作密钥的占位串

### Requirement: skill 教调用顺序，鉴权从略
skill 包 MUST 说明：`list_workflows` → `get_workflow` → `run_workflow` → 轮询 `get_task` → 读 `pixoma://task/{id}/output/{n}`。`run_workflow` MUST 被描述为立刻返回 `task_id`。skill MUST NOT 展开教如何签发或粘贴 Bearer。skill MUST 假定用户聊天支持 MCP 连接器。

#### Scenario: 工具已在则按序调用
- **WHEN** Agent 能看到 Pixoma 的 MCP tools，且用户要跑已启用工作流
- **THEN** Agent 按 list → get_workflow → run → 轮询 get_task 取产物，不等待 run 一次返回成品

#### Scenario: 禁区
- **WHEN** 用户要求通过 MCP 创建或修改 Case
- **THEN** Agent 拒绝，只使用列出、查看、发起与查询

### Requirement: 找不到工具时提醒用户去配 MCP
当 Agent 看不到 Pixoma MCP tools（`list_workflows` 等）时，skill MUST 要求 Agent 明确告诉用户：当前聊天还没有配置 Pixoma MCP 连接器，需要在该聊天的 MCP / 连接器设置里加上本实例后再试。MUST NOT 向用户索要 token 贴进对话。

#### Scenario: 未配置连接器
- **WHEN** 已安装 skill 的 Agent 在工具列表中找不到 Pixoma MCP tools
- **THEN** Agent 提醒用户去聊天的 MCP 连接器里配置 Pixoma，并说明配好之前无法跑工作流

### Requirement: MCP 失败文案必须指导 AI
对 `/mcp`、`/sse` 的 HTTP 401/403，以及 tools/call 的 `isError` 文本，系统 MUST 返回可执行的指导（下一步做什么），MUST NOT 只返回无步骤的短词（例如单独的 `unauthorized`）。指导 MUST 覆盖至少：未授权或渠道不可用、用户权限不允许跑、工作流不存在或已停用、缺必填输入、任务不属于当前用户。文案面向 Agent，可请用户去连接器或管理端改配置，MUST NOT 要求把 Bearer 发给 Agent。

#### Scenario: HTTP 401 或 403
- **WHEN** 请求未带有效 Bearer、用户不存在、或 MCP 渠道未启用
- **THEN** 响应体说明：检查聊天里的 Pixoma MCP 连接器是否指向本实例且仍有效；不要向用户索要 token；配好后重试

#### Scenario: 权限不允许跑
- **WHEN** 用户权限为付费或拒绝而调用 `run_workflow`
- **THEN** 不创建任务，错误文案说明该 MCP 用户当前不能跑工作流，请管理员在 MCP 渠道用户里改为允许后再试

#### Scenario: 缺必填且未补全
- **WHEN** `run_workflow` 省略必填且 elicitation 未完成
- **THEN** 错误文案列出缺哪些字段，并提示先 `get_workflow` 再带齐 `inputs` 重试

### Requirement: skill 提供错误对照步骤
skill 包 MUST 为至少下列失败给出 Agent 步骤：找不到工具、HTTP 401/403、权限不允许、缺必填、工作流停用或不存在、任务不属于当前用户或查询失败。步骤 MUST 与接口返回文案一致，先读错误正文再按表执行。

#### Scenario: Agent 读到 403
- **WHEN** 调用 MCP 得到 403（或等价未授权）且 skill 已加载
- **THEN** Agent 按 skill 错误表与响应正文，引导用户检查 MCP 连接器，不索要 token，不改走管理端代跑

### Requirement: 本仓库开发技能树不装该包
该 skill 包 MUST 作为产品附件分发，MUST NOT 安装到本仓库开发 Agent 的常驻 skills 目录。

#### Scenario: 仓库内位置
- **WHEN** 开发者查看本仓库开发用 skills 目录
- **THEN** 调用方 skill 不在其中；源文件在文档约定的产品附件路径

