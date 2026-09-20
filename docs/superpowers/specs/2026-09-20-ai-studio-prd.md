---
status: ready-for-pre-build-review
product: Pixoma Studio
scope: 单 Agent 创作工作区与个人资产库
created: 2026-09-20
updated: 2026-09-21
---

# Pixoma Studio PRD

## 1. 产品定义

Pixoma Studio 是 Pixoma 后台中的独立 AI 创作工作区。用户通过自然语言与单个 Agent 协作，Agent 可读取 Session 上下文，调用模型、Pixoma 工作流、声明式 Skill 和远程 MCP，并把有复用价值的内容沉淀为资产。

Studio 的核心不是“聊天版工作流启动器”，而是完整的创作闭环：

```text
提出目标 → 组织上下文 → Agent 规划与执行 → 生成资产
       → 在可编辑 Flow 中组织创作路径 → 继续迭代
       → 保存到个人资产库 → 在新 Session 中复用
```

### 1.1 核心对象

- Session：一次持续创作的工作区。
- Agent Run：一条用户消息触发的一次 Agent 执行。
- Asset：可独立查看、复用和版本化的内容对象。
- Session Flow：用户可编辑的 SOP 与资产 Road。
- Trace：不可变的执行事实，用于排障、恢复和审计。

Flow 与 Trace 必须分开：Flow 是用户可调整的创作结构；Trace 是系统实际发生过的模型与 Tool 事件。

## 2. 目标用户与数据边界

第一期面向已登录的 Pixoma 后台用户。

- 创作者：通过对话完成多步骤内容创作。
- 运营人员：生成、迭代和复用图文及媒体资产。
- 工作流设计者：让已有 Pixoma 工作流可以被 Agent 调用。
- 系统管理员：维护模型、Agent、Skill、MCP 和全局策略。

第一期数据边界：

- Session、消息、Flow、Trace 和个人资产库仅创建者可见。
- 不做 Session 分享、团队资产、多人编辑和评论。
- 模型、Agent 设置、Skill 和 MCP 由系统管理员统一配置。
- 普通用户不能读取密钥或修改系统级 AI 能力。

## 3. 产品价值链

```text
创作意图与参考资产
  ↓
Session 上下文组织
  ↓
单 Agent 理解、规划与能力选择
  ↓
模型 / 工作流 / Skill / MCP / Asset Tool
  ↓
审批、后台执行、断线恢复与错误处理
  ↓
文档 / 图片 / 音频 / 视频 / 文件资产
  ↓
可编辑 SOP 与资产 Road
  ↓
个人资产库与跨 Session 复用
```

Pixoma 的价值不止来自单次回复，而是让创作过程产生可组织、可追溯、可继续使用的资产。

## 4. 信息架构

### 4.1 后台菜单

现有后台只新增一个入口：

```text
创作 Studio
```

不新增 Studio Group，也不在普通后台菜单分别放置“AI 创作”和“资产库”。

### 4.2 Studio Shell

进入 `/studio` 后切换到独立 Shell。

左栏固定包含：

- 返回后台。
- 新对话。
- 资产库。
- 对话搜索与历史。
- 当前用户信息。

选择 Session 后，左栏右侧为：

```text
Chat 中栏 | 创作 Flow / Session 资产右栏
```

选择资产库后，左栏保持不变，右侧全部内容区域变为：

```text
文件夹 | 资产列表或网格 | 资产详情
```

### 4.3 AI 设置

系统级配置继续位于现有后台：

```text
设置 / AI 能力
├─ 模型连接
├─ Agent 设置
├─ Skills
└─ MCP 连接器
```

## 5. Session 与 Chat

### 5.1 自动保存

Session 必须自动保存：

- 用户与 Agent 的全部消息。
- 模型输入输出及实际模型快照。
- Thinking/Reasoning（仅保存提供方明确返回的内容）。
- Tool Call 参数摘要和结果摘要。
- 工作流任务、审批、错误、停止和恢复。
- 资产引用、固定版本和 Flow 变化。

这些数据同时服务历史恢复与 Trace，不依赖浏览器内存。

### 5.2 标题

- 第一条用户消息保存后异步生成 10–20 个中文字符的标题。
- 标题生成不得阻塞主 Run。
- 标题模型不可用时，使用首条消息的确定性截断结果。
- 用户可以点击标题手动修改。

### 5.3 单 Session 执行约束

- 一个 Session 同时只允许一个活跃 Run。
- Run 运行中，Composer 显示停止按钮，不发送新消息。
- 等待审批或补充输入时，Composer 恢复可操作。
- 用户可以离开页面、切换 Session 或关闭浏览器，Run 继续执行。
- 用户需要并行创作时新建另一个 Session。

### 5.4 历史消息

本期不支持：

- 编辑已发送消息。
- 一键重新生成。
- 对话分支。

修改、重试和换方案通过新消息完成。失败的操作可以从 Flow 节点发起新的执行，原记录保留。

### 5.5 Composer

Composer 工具栏包含：

```text
[添加资产] [Skill] [模型] [权限模式] [发送 / 停止]
```

- 模型与权限模式只影响下一次 Run。
- 选择后保存为当前 Session 偏好。
- Header 不放模型和权限选择器。
- 主模型只负责 Agent 对话；图片生成模型和工作流内部模型由能力适配层选择。

## 6. 单 Agent

### 6.1 本期边界

- 全站只有一个 Studio Agent。
- 不提供 Agent 列表、角色编排、多 Agent 或 SubAgent。
- 每次 Run 固定 Agent 配置版本。
- 领域模型不提前建设多 Agent 关系。

### 6.2 系统提示词

系统提示词拆成两层：

1. Pixoma 内置运行契约：资产引用、Tool 调用、审批、Trace 和安全边界，管理员不能编辑。
2. 管理员 Agent 指令：身份、业务原则、回答风格和工作偏好，可编辑和测试。

每次保存 Agent 设置自动生成版本；正在执行的 Run 不受影响。

### 6.3 Thinking / Reasoning

后台配置 `reasoning_visibility`：

- `hidden`：不展示。
- `collapsed`：展示并默认折叠。
- `expanded`：默认展开。

默认 `collapsed`。只展示模型 API 明确返回的 Thinking/Reasoning，不生成或伪造思考过程。加密推理状态可以保存并回传提供方，但不能解密展示。

## 7. 模型连接

### 7.1 协议

本期支持：

- OpenAI Responses API。
- OpenAI Chat Completions 兼容协议。
- Anthropic Messages API 兼容协议。

模型可以连接云端或本地服务。

### 7.2 连接与模型

一个连接可以配置多个模型。模型记录：

- 模型 ID 与显示名称。
- 文本、视觉输入、Tool Calling、图片生成和 Reasoning 能力。
- 是否启用。
- 是否允许 Agent 使用。
- 默认 Reasoning Effort。
- 最近能力测试结果。

Agent 可用条件：

```text
enabled && agent_enabled && connection_healthy
```

Session Run 必须保存连接、协议、模型 ID 和能力快照，不保存明文密钥。

## 8. Asset

### 8.1 定义

Asset 是在 Session 中产生、引入或编辑，具有独立身份、可查看、可复用、可版本化的内容对象。

属于资产：

- 图片、视频、音频、文档和通用文件。
- Markdown、纯文本、JSON、分镜脚本等结构化内容。
- 用户手动创建或上传的内容。
- 模型直接生成的媒体。
- 工作流明确声明为产物的文本或文件。
- 从资产库引入 Session 的固定版本。

不属于资产：

- 普通 Chat 消息。
- 生图提示词和普通字符串参数。
- Seed、尺寸、步数等标量参数。
- Tool 状态、日志、错误和临时推理结果。

非资产输入输出仍保存在 Run 和 Trace 中。

### 8.2 版本

- AssetVersion 不可变。
- 编辑文本资产创建新版本。
- Session 和 Flow 节点固定实际使用的版本 ID。
- 历史 Run 不随资产后续编辑变化。

### 8.3 Session 资产上下文

- Agent 始终能看到 Session 资产索引：ID、名称、类型、版本、来源和摘要。
- 本轮显式引用的资产加载实际内容。
- Agent 可通过 `read_asset` 按需读取其他 Session 资产。
- 大文本分段读取，不反复注入整份内容。
- 图片仅在模型支持视觉且本轮需要时发送。

### 8.4 资产库

保存到资产库不是复制文件，而是创建个人资产库引用：

- Session 继续固定原版本。
- 资产库指向同一个 AssetVersion。
- 后续编辑生成新版本，不覆盖历史 Session。
- 从资产库引入 Session 时固定所选版本。
- 直接上传或创建的资产来源显示“用户创建”，不要求关联 Session。

## 9. Session Flow

### 9.1 定义

Session Flow 是可编辑的创作 SOP 与资产 Road，不是对话 Trace。

漫画 Session 可以表现为：

```text
01 立住角色
├─ 角色设定.md
├─ 参考图.png
└─ 三视图生成 → 三视图.png

02 排好分镜
├─ 故事.md
└─ 分镜工作流 → 分镜脚本 / 分镜图

03 选模型出稿
└─ 分镜 → 图片生成 → 成稿
```

### 9.2 节点

本期节点类型：

- 阶段：SOP 编号、标题、说明和顺序。
- 计划：尚未绑定具体能力的待办步骤。
- 操作：模型生成、工作流或有产物意义的 MCP 调用。
- 资产：固定到某个 AssetVersion。

普通聊天、审批、Thinking 和日志不会自动成为 Flow 节点。

### 9.3 连线

阶段顺序使用 `sort_order`，不使用连线。阶段内部连线只表示真实数据关系：

- `input`：资产作为操作输入。
- `output`：操作产生资产。
- `reference`：资产之间的参考关系。

### 9.4 编辑

用户可以：

- 新增、重命名、删除、排序和调整阶段。
- 新增计划节点。
- 将 Session 资产添加到阶段。
- 在阶段间移动节点。
- 创建、删除和重新连接关系。
- 选择当前采用的资产版本。
- 从操作节点发起新的执行。
- 撤销、重做和整理布局。

删除 Flow 节点只移除画布关系，不删除底层资产或 Trace。

### 9.5 Agent 修改规则

Agent 使用语义命令更新 Flow，不直接生成坐标：

- `create_stage`
- `create_plan`
- `add_operation`
- `attach_asset`
- `connect_nodes`
- `set_node_status`

用户手动调整优先。Agent 不得在用户未要求时删除、重命名或重排用户创建的阶段；新增节点只整理新增区域，不重排整张画布。

本期不做 SOP 模板库和“保存为模板”。

## 10. Trace

Trace 与 Flow 独立，通过 Chat Header 或操作节点打开抽屉。

Trace 记录：

- Run 生命周期。
- Prompt 与上下文版本摘要。
- 模型、协议、Token、耗时和结束原因。
- Thinking/Reasoning（提供方明确返回时）。
- Tool Call、工作流任务和 MCP 调用。
- 审批、重试、错误和恢复。
- 资产输入、输出及固定版本。

密钥永远不进入 Trace；敏感 Tool 字段按定义脱敏。

## 11. Skill

Skill 是可复用的指令、知识与操作方法，不是另一套工作流系统。

本期 Skill 包含：

- `SKILL.md`
- `references/`
- `assets/`

不包含：

- 任意脚本执行。
- Skill 依赖关系表。
- 与具体工作流或 MCP 的强绑定。
- 单独权限矩阵。

运行时采用渐进式加载：模型默认只看到名称和描述，命中或显式选择后才加载完整内容。每次 Run 固定实际 Skill 版本。

## 12. MCP

### 12.1 本期能力

- 仅支持远程 Streamable HTTP。
- 管理员配置系统级共享凭据。
- 支持 Tool、Resource 和 Prompt 发现。
- Tool 由模型选择调用。
- Resource 被选择后进入 Session 上下文，不自动注入全部内容。
- Prompt 由用户或 Agent 显式使用，不能覆盖上层系统规则。

### 12.2 预留但不开放

- `stdio` Transport。
- 用户级 OAuth 凭据。

底层保留 Transport 和凭据归属类型，但界面不可创建未实现类型。

## 13. 工作流接入

现有工作流是唯一事实来源。名称、描述、输入、输出、字段说明、类型和 JSON Schema 全部复用当前 CaseDocument。

本期只新增：

```text
agent_callable: boolean
```

进入 Agent Tool Catalog 的条件：

```text
enabled && agent_callable
```

不新增面向模型的名称、描述、示例、资产类型、风险、费用或超时配置。运行适配层从现有工作流定义生成 Tool Schema 和资产映射。

## 14. 权限与审批

Composer 提供 Codex 式三档模式：

| 模式 | 行为 |
|---|---|
| 请求批准 | 命中审批规则后由用户批准或拒绝 |
| 帮我批准 | 命中审批规则后由 Reviewer 模型审核；失败或高风险时交给用户 |
| 完全访问 | 在用户权限和系统策略允许范围内不显示审批请求 |

规则：

- 新 Session 默认“请求批准”。
- 用户选择保存为当前 Session 偏好。
- “完全访问”首次启用显示风险确认。
- 系统级 Deny、RBAC、连接器禁用、跨用户隔离和 Schema 校验不能被绕过。
- 审批状态持久化；刷新或离开页面后仍可继续。

## 15. 后台运行与通知

- 浏览器断开不取消 Run。
- 回到 Session 后恢复 Snapshot，并从最后事件序号继续订阅。
- Studio 历史显示运行中、等待审批、完成未查看和失败状态。
- 用户停留在 Pixoma 后台时显示完成或失败 Toast。
- Studio 菜单显示未查看完成与待审批数量。
- 本期不做系统通知、邮件或浏览器推送。

## 16. 内置 Tool

- `list_session_assets`
- `read_asset`
- `create_text_asset`
- `update_text_asset`
- `generate_image`
- `edit_session_flow`
- `search_capabilities`
- `load_skill`
- 动态 Workflow Tools
- 动态 MCP Tools

“保存到资产库”不作为 Agent Tool，由用户显式点击完成。

## 17. 使用场景与覆盖判断

### 17.1 通用用户旅程

| 阶段 | 用户行为 | Studio 提供的能力 | 用户可见结果 |
|---|---|---|---|
| 开始 | 新建 Session，用自然语言说明目标 | 自动保存首条消息、生成标题、建立空资产索引 | 可持续恢复的创作空间 |
| 准备 | 上传、创建或从资产库引入参考资料 | 资产版本固定、按需读取、Composer 引用 | 明确的上下文与输入版本 |
| 规划 | 与 Agent 讨论方案并确认步骤 | 单 Agent、Skill、能力检索、计划节点 | 初始创作 Road |
| 执行 | 让 Agent 生成内容或调用工作流、MCP | 三档批准、后台 Run、失败恢复 | 文档、图片等 Session 资产 |
| 整理 | 调整阶段、节点、关系和采用版本 | 可编辑 Flow，用户调整优先 | 符合个人方法的 SOP 与资产 Road |
| 迭代 | 换模型、修改资产、重新执行局部步骤 | 不可变 AssetVersion、操作节点再执行 | 保留历史的派生版本 |
| 沉淀 | 把选定资产保存到资产库 | LibraryRef 复用同一 AssetVersion | 可跨 Session 使用的个人资产 |
| 回看 | 恢复历史对话或排查执行问题 | Session 快照、Trace、模型和配置快照 | 可继续创作、可解释执行 |

### 17.2 典型场景

| 场景 | 旅程 | 本期覆盖 |
|---|---|---|
| 单篇漫画 | 故事 → 角色 → 分镜 → 逐格出稿 → 资产沉淀 | 完整覆盖核心链路 |
| 电商商品图 | 引用商品图 → 场景探索 → 批量工作流 → 选稿 | 覆盖；批处理依赖工作流 |
| 短视频前期 | 脚本 → 镜头 → 参考图 → 视频工作流 | 覆盖资产与执行；无专业时间线 |
| 品牌营销 | 品牌资料 → 文案 → 海报 → MCP 发布 | 覆盖；外部写入受审批控制 |
| 研究整理 | MCP 获取资料 → 笔记 → Markdown → 报告 | 覆盖；无独立向量知识库 |
| 资产再创作 | 资产库引用 → 新 Session → 派生版本 | 完整覆盖单用户复用 |

### 17.3 漫画场景能力核对

以“单篇漫画”为端到端验收样例：

1. 用户描述故事方向，Agent 生成故事设定 Markdown Asset。
2. 用户修改故事设定，系统创建新 AssetVersion，旧版本继续可追溯。
3. Agent 创建“立住角色、排好分镜、选模型出稿”三个阶段。
4. Agent 直接调用生图模型生成三视图，图片成为 Asset，不要求经过工作流。
5. Agent 把故事设定和三视图作为固定版本输入，调用分镜工作流。
6. 用户离开 Studio，工作流继续执行；返回后从服务端快照恢复进度。
7. 分镜脚本和分镜图加入 Session 资产与 Flow；Trace 保留真实调用记录。
8. 用户拖动、增删和重连 Flow 节点，不改变资产和 Trace。
9. 用户把采用的角色图与分镜保存到资产库，在后续 Session 复用。

本期能力可以完成该闭环。明确不覆盖的是多人协作、SOP 模板复用、专业漫画排版编辑和逐格图像编辑；这些缺口不会阻断“从构思到分镜资产”的核心旅程。

## 18. 资源护栏

本期不建设金额计费、余额或按用户金额配额。提供：

- 最大 ReAct 步数。
- Run 超时。
- 每用户并发限制。
- 模型 Token、图片和工作流调用量记录。
- 用量和费用估算。
- 管理员可配置的调用次数与并发限制。

## 19. 本期范围

### 19.1 包含

- 单入口 Studio Shell。
- Chat、可编辑 Flow、Session 资产和个人资产库。
- 单 Agent、Eino ReAct 与 AG-UI。
- OpenAI Responses、Chat Completions 兼容和 Anthropic Messages。
- 模型、Agent、Skill 和 MCP 后台设置。
- 声明式 Skill。
- 远程 MCP Client。
- 工作流 Tool 适配。
- 三档审批。
- 后台运行、断线恢复和 Trace。
- 资产版本、来源和固定引用。

### 19.2 不包含

- 多 Agent、SubAgent。
- SOP 模板库。
- 对话编辑、重新生成和分支。
- 团队协作与共享资产。
- 用户级 MCP OAuth。
- `stdio MCP`。
- Skill 脚本。
- 专业图片、音频或视频编辑器。
- 大规模 RAG 与资产语义检索。
- 系统、邮件和浏览器推送。
- Session 归档和删除。
- 金额计费和余额系统。

## 20. 验收标准

- 后台菜单只有一个「创作 Studio」入口。
- Studio 左栏可在 Session 与资产库之间切换。
- 模型和权限模式位于 Composer，不在 Header。
- 首条消息自动保存并生成标题。
- 同一 Session 只有一个活跃 Run。
- 页面关闭后 Run 继续，回来后不重复执行。
- 审批刷新后仍可继续。
- 同一 Tool Call 不因重连执行两次。
- 模型、工作流和 MCP 产物统一进入资产体系。
- 文本资产编辑创建新版本。
- 资产库引用固定版本，不复制 Blob。
- Flow 可编辑且不是 Trace。
- Flow 重排不修改 Trace。
- 删除 Flow 节点不删除资产或执行记录。
- 用户只能访问自己的 Session 和资产。
- 密钥不进入消息、Trace 和前端响应。
- 模型、Skill、Prompt、权限和资产版本可从历史 Run 快照追溯。

## 21. 开发前技术 Spike

正式建设业务功能前验证：

1. Eino 事件到 AG-UI 标准事件的流式映射，以及 Interrupt/Resume。
2. assistant-ui AG-UI Runtime 加薄重连层后，能否恢复服务端后台 Run。
3. 现有工作流 Task 能否通过外部任务引用唤醒等待中的 Agent Run。

Spike 失败时先调整技术方案，不在完整页面开发后补救。

## 22. 开工门槛

开始完整业务开发前必须同时满足：

- 本 PRD、UX、技术设计和原型完成一次联合评审。
- 三项 Spike 有可运行结果和结论记录。
- 数据库迁移、API 契约、AG-UI 事件契约冻结第一版。
- 明确本期默认模型、标题模型、审核模型及不可用时的降级策略。
- 选定 1 个现有工作流作为首个端到端接入样例。
- 确认远程 MCP 测试服务和一套无敏感数据的测试凭据。
- 漫画验收样例能覆盖直接模型产物、工作流产物、后台恢复、批准、Flow 编辑和保存到资产库。
