## Context

见 `proposal.md` 的 Why。用户聊天必须自带 MCP 连接器；skill 不再写鉴权教程。现有失败文案（`unauthorized`、`user access denied`）对 Agent 没有下一步。

## Goals / Non-Goals

**Goals:**

- Skill 写调用链 + 错误对照；缺工具 = 提醒去配连接器。
- 把 MCP HTTP 401/403 与 tool 错误改成「给 AI 的处置说明」。

**Non-Goals:**

- 不改工具集合与异步语义。
- 不在 skill 里教 mint / 粘贴 Bearer。

## Decisions

### 1. 鉴权从略，错误加重

- **选择**：Skill 开篇写前置：聊天需支持 MCP 连接器。找不到工具 → 只提醒去配连接器。401/403/权限/缺字段由**接口文案 + skill 表**共同指导。
- **理由**：连接器是用户聊天的事；Agent 真正卡住的是「没工具」和「短错误读不懂」。
- **备选**：Skill 详写复制配置、粘 token（用户否定，也不安全）。

### 2. 错误文案写在 MCP 边界，映射内部错误

- **选择**：`RequireBearer` 的 401/403 正文、`toolErr` 对 `ErrAccessDenied` / 未找到 / 停用 / 缺字段做面向 Agent 的句子。内部 sentinel 可保留，对外替换。
- **理由**：Agent 只看见 HTTP/tool 文本。
- **备选**：只改 skill、不改接口（用户要求接口文案本身可指导）。

### 3. 包路径与提示词不变

- **选择**：`docs/skills/pixoma-mcp/`，公网 `https://pixoma.miaoplus.com/skills/pixoma-mcp/SKILL.md`。提示词只负责下载安装 + 点明需要 MCP 连接器。
- **理由**：与已确认的分发方式一致。

## Risks / Trade-offs

- [各聊天连接器 UI 不同] 文案写「MCP / 连接器设置」，不写死某一个产品菜单名。
- [公网 URL 未同步] 文档同时给仓库相对路径。

## Migration Plan

改错误文案需同步现有断言「unauthorized」等测试。回滚恢复短词 + 删除 skill。不改表。无需更新架构文档（不改拓扑与主链路）。
