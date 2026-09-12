---
comet_change: mcp-caller-skill
role: technical-design
canonical_spec: openspec
archived-with: 2026-09-12-mcp-caller-skill
status: final
---

# MCP 调用方 Skill 深度技术设计

Canonical 行为见 OpenSpec delta。高层见 change 内 `design.md`。用户已确认方案 A。

## 1. 目标

调用方 Agent 能按 tool 链跑工作流；聊天自带 MCP 连接器。失败时接口正文与 skill 错误表给出下一步。不教贴 token。

## 2. 现状

- 5 个 tools、2 个 resource 模板、1 个 prompt；Bearer 中间件。
- HTTP 失败正文多为 `unauthorized`。`toolErr` 直接 `err.Error()`（如 `user access denied`）。
- 后台已有「复制配置」；本期 skill 不展开教它。

## 3. 模块

```text
internal/mcp/auth.go     401/403 响应正文
internal/mcp/tools.go    toolErr 按类型换指导句
docs/skills/pixoma-mcp/  产品 skill（不进 .agents/skills）
README / docs/install    一句安装提示词
```

不改 `RunCase` 主链路；内部 sentinel 可保留，只在 MCP 边界换对外句子。

## 4. HTTP 状态与正文

保持现有状态码，只改 body（纯文本即可）：

| 情况 | 状态 | 正文要点（面向 Agent） |
|---|---|---|
| 无/错 Bearer | 401 | 检查聊天的 Pixoma MCP 连接器是否指向本实例且凭据仍有效；不要向用户要 token；配好后重试 |
| 渠道停用 / 非 mcp 平台 / 用户不存在 | 401 或现有码 | 同上，并说明连接器对应的 MCP 渠道或用户可能已停用 |
| 会话身份不匹配（现测 403） | 403 | 连接器身份与会话不一致；让用户重连该连接器后重试，不要索要 token |

句子固定，测试用子串断言（如「连接器」「不要」+ token 相关禁止语）。

## 5. Tool 错误映射

`toolErr` 在写出 `IsError` 前 `errors.Is` / 关键词映射：

| 内部 | 对外要点 |
|---|---|
| `botapp.ErrAccessDenied` | 该 MCP 用户当前不能跑工作流；请管理员在 MCP 渠道用户里改为允许后再试 |
| Case 不存在 / `ErrDisabled` | 工作流不存在或已停用；先 `list_workflows` 换已启用的 id |
| 缺必填 | 列出缺的 key；先 `get_workflow` 再带齐 `inputs` |
| 任务不可见 / 空 task_id | 只能查当前连接器用户的任务；核对 `task_id` 或先 `list_tasks` |
| 其它 | 保留原错误，前缀一句「读完本句后按 skill 错误表处理」仅当否则会丢失信息时才加；默认透传即可 |

不引入 `code` 字段。

## 6. Skill 包

路径：`docs/skills/pixoma-mcp/SKILL.md`。

结构：

1. frontmatter：`name: pixoma-mcp`；`description` 仅 Use when（跑 Pixoma 工作流 / 出图 / 查自己的任务 / 找不到 MCP 工具）。
2. 前置：聊天必须能配 MCP 连接器。
3. 调用链：list → get_workflow → run → poll get_task → resource。
4. 错误表：无工具；401/403；权限；缺字段；停用/不存在；任务不可见。步骤与第 4、5 节句子同义。
5. 禁区：不创建 Case、不索要 token、不走管理端代跑。

## 7. 提示词

文档一句，含 `https://pixoma.miaoplus.com/skills/pixoma-mcp/SKILL.md`，并写「聊天需支持 MCP 连接器」。无密钥。另注仓库相对路径。

## 8. 测试

- `auth_test`：401/403 状态不变，body 含连接器指导、不含要求把 token 发给模型。
- `tools_test`：付费用户、停用、缺字段、他人任务的 `IsError` 文本含对应下一步。
- 可选：skill 文件存在、含错误表标题、description 无流程摘要；可用现有文案/路径合同测试或短文件断言。

## 9. 风险

- 连接器 UI 各异：只写「MCP / 连接器」。
- 公网未挂文件：文档给仓库路径。
- 不改架构文档（无拓扑/主链路变化）。
