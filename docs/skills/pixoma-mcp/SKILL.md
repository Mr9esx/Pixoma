---
name: pixoma-mcp
description: "Use when the user wants to run a Pixoma workflow, generate an image through Pixoma, check their own Pixoma tasks, or Pixoma MCP tools are missing from the chat."
---

# Pixoma MCP

## 前置

用户的聊天产品必须能配置 MCP 连接器。本 skill 不教如何配置 token。

## 调用顺序

能看到 Pixoma tools 时按这次序：

1. `list_workflows` 列出已启用工作流
2. `get_workflow` 看必填 `inputs`
3. `run_workflow`：立刻返回 `task_id`，不要等它一次吐成品
4. 轮询 `get_task`
5. 读 `pixoma://task/{id}/output/{n}`

只使用列出、查看、发起与查询。用户要求通过 MCP 创建或修改 Case 时拒绝。

## 错误对照

先读接口正文，再按本表做。步骤与接口固定句同义。

| 情况 | Agent 步骤 |
|---|---|
| 工具列表里没有 `list_workflows` 等 | 明确告诉用户：当前聊天还没有配置 Pixoma MCP 连接器，要在该聊天的 MCP / 连接器设置里加上本实例后再试。配好之前无法跑工作流。不要向用户索要 token。 |
| HTTP 401，或正文说渠道/用户可能已停用 | 检查聊天的 Pixoma MCP 连接器是否指向本实例且凭据仍有效。对应的 MCP 渠道或用户可能已停用。不要向用户索要 token。配好后重试。 |
| HTTP 403，或连接器身份与会话不一致 | 连接器身份与会话不一致。让用户重连该连接器后重试。不要向用户索要 token。不要改走管理端代跑。 |
| 不能跑工作流 / MCP 渠道用户 | 该 MCP 用户当前不能跑工作流。请管理员在 MCP 渠道用户里改为允许后再试。 |
| 缺少必填 / `inputs` | 列出缺的字段。先 `get_workflow` 再带齐 `inputs` 重试。 |
| 工作流不存在或已停用 | 先 `list_workflows` 换已启用的 id。 |
| 任务不可见 / 查失败 | 只能查当前连接器用户的任务。核对 `task_id` 或先 `list_tasks`。 |

## 禁区

- 不创建或修改 Case
- 不向用户索要 token，不把 Bearer 贴进对话
- 不改走管理端代跑工作流
