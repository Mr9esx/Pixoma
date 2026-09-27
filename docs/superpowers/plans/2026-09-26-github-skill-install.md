# GitHub Skill 对话安装实施计划

> **For agentic workers:** 按任务逐项实施；本次由当前会话连续执行。

**Goal:** 允许 Pixoma Studio Agent 在用户明确要求时，从公开 GitHub 链接安装 Skill 到当前账户。

**Architecture:** 新增 GitHub Skill 包读取器，验证 GitHub 文件或目录链接，从 `raw.githubusercontent.com` 读取 `SKILL.md` 与其中明确引用的 Markdown/TXT 参考文件，再把相对路径和正文写入现有 `Prompt`。新增 `install_skill` Eino 工具，通过已有能力配置服务保存已启用 Skill；后续 Run 沿用现有账户快照和 `load_skill` 流程。

**Tech Stack:** Go、Eino tools、`net/http`、Goldmark、`gopkg.in/yaml.v3`、GORM SQLite。

## 全局要求

- Skill 仅保存到本次 Run 所属账户。
- 下载来源限定为公开 GitHub 仓库。
- 仅处理 `SKILL.md` 明确引用的 `.md` 和 `.txt` 文件，不执行下载内容。
- 不修改工作区内已有的其他未提交改动。

---

### 任务 1：读取并整理 GitHub Skill 包

**文件：**
- 新建：`internal/studio/infrastructure/einoagent/github_skill.go`
- 新建：`internal/studio/infrastructure/einoagent/github_skill_test.go`

**接口：**
- 产出 `parseGitHubSkillURL(rawURL string) (githubSkillSource, error)`，解析 `github.com/{owner}/{repo}/blob/{ref}/{path}/SKILL.md` 与 `github.com/{owner}/{repo}/tree/{ref}/{path}`。
- 产出 `fetchGitHubSkill(ctx context.Context, source githubSkillSource) (githubSkillPackage, error)`，读取目标目录里的 Skill 文件。
- 产出 `githubSkillPackage`，字段包含 Skill 名称、描述、合并后的 Prompt。

- [x] 为 `blob`、`tree`、错误主机名、错误路径编写纯函数测试。
- [x] 为 YAML metadata 缺失、无效，以及参考文件相对路径保留编写纯函数测试；使用 Goldmark 解析 Markdown 节点与链接。
- [x] 使用 `yaml.v3` 解析 `SKILL.md` 的 metadata；仅读取主说明明确引用的 `.md`、`.txt` 文件，并在 Prompt 中按相对路径组织正文。
- [x] 通过 `raw.githubusercontent.com` 读取仓库文件；拒绝非 GitHub 来源及不受支持的路径。
- [x] 执行 `go test ./internal/studio/infrastructure/einoagent -run 'Test(ParseGitHubSkillURL|GitHubSkillPackage)'`。

### 任务 2：添加对话安装工具并保存账户 Skill

**文件：**
- 新建：`internal/studio/infrastructure/einoagent/install_skill_tool.go`
- 修改：`internal/studio/infrastructure/einoagent/engine.go`
- 修改：`internal/studio/application/capability_config.go`
- 修改：`apps/pixoma/internal/application/app.go`
- 新建：`internal/studio/infrastructure/einoagent/install_skill_tool_test.go`

**接口：**
- 在 application 层新增 `SkillCreator` 接口，方法签名为 `CreateSkill(context.Context, CreateSkillInput) (*SkillView, error)`。
- 在 `Engine` 增加 `SkillCreator studioapp.SkillCreator` 字段；生产组装时传入 `studioCapabilityService`。
- 新增 `newInstallSkillTool(accountID string, creator studioapp.SkillCreator, sink studioapp.AgentSink)`，返回可调用的 Eino 工具。

- [x] 新增工具说明和参数 schema，要求输入 `github_url`。
- [x] 工具调用 `fetchGitHubSkill`，使用当前 Run 的账户 ID、Skill metadata 和合并 Prompt 调用 `CreateSkill`，并将 `Enabled` 设为 `true`。
- [x] 按现有 `load_skill` 工具的事件格式写入工具开始、参数、结果和结束事件；调用错误时返回 Eino 工具错误。
- [x] 在 `Engine.resolveTools` 仅于 `SkillCreator` 已配置时注册安装工具。
- [x] 检查工具公开名称、说明和必需的 `github_url` 参数。
- [x] 用环境变量控制真实 GitHub 集成验证；经公开 H3 链接调用 `install_skill`，使用实际 `CapabilityConfigService` 与持久层保存并读取 Skill，并检查工具事件顺序。
- [x] 执行安装器定向测试，以及 `go test ./internal/studio/application ./internal/studio/infrastructure/persistence ./apps/pixoma/internal/application`。

### 任务 3：端到端验证与文档复核

**文件：**
- 验证：`docs/superpowers/specs/2026-09-26-github-skill-install-design.md`
- 验证：`internal/studio/infrastructure/einoagent/engine_prompt_test.go`

- [x] 确认 H3 公开 GitHub 链接的真实网络安装会保存名称、描述、`SKILL.md`、`references/base-en.txt` 和 `references/ref-en.txt` 内容。
- [x] 确认 Agent Run 结束后，Studio 刷新 Skills 查询，使后续对话选择栏读取新 Skill。
- [x] 执行前端 Skills 刷新定向测试、安装器定向测试及账户保存相关 Go 包测试。
- [x] 检查 `git diff --check` 和功能相关文件 diff，确认安装内容没有被执行，账户 ID 来自 Run。
