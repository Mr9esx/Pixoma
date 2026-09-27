# GitHub Skill 对话安装设计

## 目标

Pixoma Studio Agent 支持用户在对话中提交公开 GitHub Skill 链接并安装。安装结果保存到当前账户的 Skills，安装完成后可供后续对话选择。

## 方案

在 Studio Agent 中增加 `install_skill` 工具。工具仅处理公开 GitHub 上的 Skill 文件或目录链接。输入支持指向 `SKILL.md` 的 `blob` 链接，以及包含 `SKILL.md` 的 `tree` 目录链接。

工具直接读取 `raw.githubusercontent.com` 上的 `SKILL.md`，使用 Goldmark 解析主说明并读取其中明确引用的 Markdown 和 TXT 参考文件，保留每个文件的相对路径，并将它们组织为现有 Skill 的 `Prompt` 内容。安装时从 `SKILL.md` 的 YAML metadata 读取名称和描述，检查必需字段后，通过现有账户能力配置服务创建已启用的 Skill。

安装内容仅作为 Agent 的创作指导文本。工具不执行脚本，也不调用 Skill 包中的命令。安装工具只在用户明确要求安装时调用。

## 数据流

1. 用户在 Pixoma Studio 对话中提交 GitHub Skill 链接并要求安装。
2. Agent 调用 `install_skill`，并传入该链接。
3. 工具验证链接格式和 GitHub 来源，读取 `SKILL.md`，并读取其中明确引用的 Markdown/TXT 文件。
4. 工具检查 `SKILL.md` 的 metadata，整理主说明和参考文件内容。
5. 工具使用当前 Run 的账户 ID 创建已启用的 Skill，并返回安装结果。
6. 后续 Run 获取账户中已启用的 Skill；用户可在聊天栏选择新 Skill，Agent 通过现有 `load_skill` 工具读取说明。

## 错误处理

链接不受支持、GitHub 内容不可读取、`SKILL.md` 缺少必需字段、metadata 无效或 Skill 保存失败时，工具返回对应错误，不创建 Skill。下载只允许访问公开 GitHub 原始文本地址，避免将用户提供的链接作为任意网络请求目标。

## 验证

- 检查支持的 GitHub `blob` 与 `tree` 链接解析结果。
- 检查 `SKILL.md` metadata 和缺失字段的处理。
- 检查主说明明确引用的参考文件以原相对路径组织到 `Prompt` 中。
- 检查 Skill 归属当前账户、启用状态和后续 Run 的读取结果。
- 检查不支持来源和 GitHub 读取错误时不会创建 Skill。

## 已确认范围

- 来源限定为公开 GitHub。
- 安装范围为当前账户。
- 安装结果供后续对话选择。
- Skill 主说明明确引用的文本参考文件随主说明一起保存；远端脚本不执行。
