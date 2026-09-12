## Why

Pixoma 已通过 MCP tools 提供跑工作流的能力，但调用方 Agent 不知道调用顺序，也读不懂现在的短错误（如 `unauthorized`、`user access denied`）。用户的 LM 聊天必须自己支持 MCP 连接器；skill 不负责教鉴权，但找不到工具和接口失败时，必须能指导 AI 和下一步。

## What Changes

- 新增调用方 skill 包：调用顺序、异步轮询、禁区；**前置假定**聊天产品已能配 MCP 连接器。
- 文档一句安装提示词，指向 `https://pixoma.miaoplus.com` 上的 skill 包；不含密钥。
- Skill **少写鉴权/贴 token**。找不到 Pixoma MCP 工具时，提醒用户：聊天还没配 MCP 连接器。
- MCP **失败返回的文案**（HTTP 401/403 与 tool `isError`）MUST 写清 AI 该怎么处理（查连接器、查用户权限、补字段等），不得只丢 `unauthorized`。
- Skill 内附错误对照：每种常见失败对应步骤。

### 非目标

- 不改工具名、资源 URI、发起即返回 `task_id` 的语义。
- 不在对话里收集或转发 Bearer；不把 skill 写成「如何签发 token」教程。
- 不在管理端加「复制 Skill」，不改落地页区块。
- 不把该包装进本仓库开发 Agent 常驻技能树。
- 不声明 sampling，不新增 Case 写入工具。

## Capabilities

### New Capabilities

- `mcp-caller-skill`：调用方 skill、安装提示词、缺工具提醒，以及 MCP 失败文案对 AI 的指导契约。

### Modified Capabilities

## Impact

- `docs/skills/pixoma-mcp/SKILL.md` 与文档提示词。
- `internal/mcp`（及必要时 `RunCase` 映射）：401/403 与 tool 错误文案改为对 AI 可执行。
- 不改任务执行主链路、表结构、渠道签发流程。
