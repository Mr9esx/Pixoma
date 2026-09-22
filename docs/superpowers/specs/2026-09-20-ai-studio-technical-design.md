---
status: ready-for-pre-build-review
product: Pixoma Studio
role: technical-design
created: 2026-09-20
updated: 2026-09-21
---

# Pixoma Studio 技术设计

## 1. 设计原则

1. Studio 是独立限界上下文，不复用现有 Bot Session、Conversation 或 MCP Server 的业务模型。
2. 只通过稳定 Port 接入现有工作流、Blob、Crypto 和后台用户能力。
3. Eino 负责 Agent 推理循环；Pixoma 负责 Session、资产、Flow、权限、后台执行和持久化。
4. AG-UI 是 Agent 与前端的事件协议，不是数据库和领域模型。
5. Session Flow 与 Run Trace 分表、分服务、分权限语义。
6. 所有可能跨连接和跨进程的状态必须持久化，不能依赖浏览器或 goroutine 生命周期。

## 2. 总体架构

```text
Web Admin
├─ Studio Shell
├─ assistant-ui + @assistant-ui/react-ag-ui
├─ @xyflow/react
└─ AI 设置页
        │
        │ REST + AG-UI/SSE
        ▼
HTTP Boundary
├─ Studio REST API
├─ AG-UI Run Endpoint
├─ SSE Replay / Attach Endpoint
└─ AI Admin API
        │
        ▼
Studio Application
├─ Session Service
├─ Asset Service
├─ Flow Service
├─ Run Coordinator
├─ Approval Service
└─ AI Config Service
        │
        ▼
Agent Runtime
├─ Context Compiler
├─ AgentEngine Port
├─ Capability Registry
├─ Policy Engine
├─ Checkpoint / Resume
└─ AG-UI Bridge
        │
        ▼
Adapters
├─ Eino ADK
├─ Model Providers
├─ Workflow Runner
├─ MCP Client
├─ Skill Backend
└─ Asset / Flow Tools
        │
        ▼
Infrastructure
├─ GORM Database
├─ Blob Storage
├─ Secret Crypto
├─ Background Worker
└─ Event Notifier
```

## 3. Go 模块边界

建议新增：

```text
internal/studio/
├─ domain/
│  ├─ session/
│  ├─ run/
│  ├─ asset/
│  ├─ flow/
│  └─ config/
├─ application/
│  ├─ session/
│  ├─ run/
│  ├─ asset/
│  ├─ flow/
│  ├─ approval/
│  └─ admin/
├─ infrastructure/
│  ├─ persistence/
│  ├─ eino/
│  ├─ agui/
│  ├─ models/
│  ├─ mcpclient/
│  ├─ skills/
│  └─ workflow/
└─ worker/

internal/httpapi/studio/
internal/httpapi/aiconfig/
```

不复用 `internal/sessions`：该模块服务现有 Bot/工作流输入收集，生命周期与 Studio Session 不同。

不反向依赖 `internal/mcp`：现有模块继续作为 Pixoma 对外 MCP Server；Studio 使用独立 MCP Client Adapter。

允许复用的平台能力：

- `internal/platform/blob`
- `internal/platform/crypto`
- `internal/platform/db`
- 后台身份与 RBAC 中间件
- 现有 Task/Case Application Port

## 4. 核心 Port

```go
type AgentEngine interface {
    Start(ctx context.Context, input EngineInput) (EventStream, error)
    Resume(ctx context.Context, input ResumeInput) (EventStream, error)
}

type ChatModel interface {
    Generate(ctx context.Context, req ModelRequest) (ModelResponse, error)
    Stream(ctx context.Context, req ModelRequest) (ModelStream, error)
}

type WorkflowRunner interface {
    Describe(ctx context.Context, caseID uint64) (WorkflowDefinition, error)
    Start(ctx context.Context, cmd WorkflowRunCommand) (WorkflowRunRef, error)
    Get(ctx context.Context, ref WorkflowRunRef) (WorkflowRunStatus, error)
    Cancel(ctx context.Context, ref WorkflowRunRef) error
}

type MCPClient interface {
    Discover(ctx context.Context, connection MCPConnection) (MCPCatalog, error)
    CallTool(ctx context.Context, call MCPToolCall) (MCPToolResult, error)
    ReadResource(ctx context.Context, ref MCPResourceRef) (MCPResource, error)
    GetPrompt(ctx context.Context, ref MCPPromptRef) (MCPPrompt, error)
}

type SkillBackend interface {
    ListSummaries(ctx context.Context) ([]SkillSummary, error)
    LoadVersion(ctx context.Context, ref SkillVersionRef) (SkillPackage, error)
}

type EventSink interface {
    Append(ctx context.Context, event RunEvent) error
}
```

领域层和 Application 层不能出现 Eino、AG-UI、OpenAI、Anthropic 或 MCP SDK 类型。

## 5. 前端架构

### 5.1 Chat

使用：

- `@assistant-ui/react`
- `@assistant-ui/react-ag-ui`
- `@ag-ui/client`

assistant-ui 官方 AG-UI Runtime 负责消息、文本流、Tool Call、Reasoning 和 Interrupt 的 UI 状态。Pixoma 提供 Attachment、History 和错误处理 Adapter。

不依赖实验性的 Thread List Adapter。Session 历史、切换和路由由 Pixoma + TanStack Query/Router 管理。

### 5.2 后台 Run 重连

AG-UI `HttpAgent` 当前没有完整的 HTTP SSE 游标重连能力。实现薄的 `PixomaHttpAgent`：

- 初次 Run 仍发送标准 `RunAgentInput`。
- 记录服务端 `run_id` 和最新 `sequence`。
- 页面恢复时调用 Attach Endpoint。
- 按 `after_sequence` 重放，再继续实时订阅。
- 事件载荷保持标准 AG-UI Schema。

这只是 Transport 扩展，不创建 Pixoma 私有 Chat 协议。

### 5.3 Flow

使用 `@xyflow/react` v12+：

- 自定义 Stage、Plan、Operation 和 Asset 节点。
- 受控 Nodes/Edges。
- NodeTypes/EdgeTypes 使用稳定引用。
- 节点内交互元素使用 `nodrag`，滚动区使用 `nowheel`。
- 多 Handle 使用稳定唯一 ID。
- 不用 `display:none` 隐藏 Handle。

Flow 客户端状态分为：

- Durable：节点业务数据、坐标、尺寸、边和视口，保存到服务端。
- Ephemeral：拖动中、选中、弹层和 Hover，只保留本地。
- Computed：React Flow 测量尺寸，不保存。

本期不引入 ELK 等大布局依赖。采用确定性布局：阶段自上而下排列，阶段内部按资产 → 操作 → 资产布局。用户手动移动后标记 `user_positioned`。

Undo/Redo 在拖动开始、删除、连线和程序化修改前保存快照，不记录每个 Drag Move。

### 5.4 UI 组件

- 基础组件只从 `web/admin/src/components/ui` 引用，不在 Studio 目录复制 Button、Form、Dialog、Select、Toast 等实现。
- 现有组件覆盖不了时，先用 `pnpm dlx shadcn@latest add` 加入同一组件目录，再按 Pixoma 设计系统校正；不安装第二套通用 UI 库。
- Studio Shell 复用 Sidebar、Button、ScrollArea、Avatar、Tooltip 和现有后台布局模式。
- Chat 使用 assistant-ui primitives 和 AG-UI Runtime；Button、附件外壳、菜单、审批卡片、错误提示和 Composer 装饰层使用 Pixoma 组件与令牌。
- Chat 与 Flow 的分栏使用 shadcn Resizable；项目尚未安装时通过 shadcn CLI 添加。
- Trace 使用 Sheet；危险确认使用 AlertDialog；详情与创建使用 Dialog。
- 模型、Skill、Session 资产和资产库选择使用 Command + Popover。
- 运行反馈使用 Progress、Skeleton、Alert、Empty 和 Sonner。
- 设置页复用现有 Settings Shell、Field/Form、Input、Select、Switch、Table 和 Tabs。
- Flow 使用现有 `@xyflow/react`，自定义节点内部使用 Card、Badge、Button、Tooltip 和 DropdownMenu。
- 图标统一使用项目现有 `lucide-react`，不使用 emoji 或另一套图标库。
- 所有样式只使用 Pixoma 语义令牌；表面无投影，浮层才允许投影。

assistant-ui 作为 Chat 行为层，不拥有最终视觉：

- 不直接引入与 Pixoma 冲突的 assistant-ui 默认主题。
- 用 Pixoma 类名和语义令牌实现 Message、Composer、Attachment、Tool Call、Reasoning 和 Interrupt 的所有状态。
- assistant-ui 升级不得改变平台全局 Button、Input、Popover 或字体规则。

### 5.5 生产级前端约束

本期前端按完整产品功能交付，不设置“先做能跑、再补 UI”的阶段。

- 所有数据请求具备 loading、empty、error、retry、stale 和 permission denied 处理。
- 所有 Mutation 具备 pending、幂等防重、成功反馈、失败恢复和必要的乐观更新回滚。
- Session、消息、资产和 Flow 的大列表采用分页、增量加载或虚拟化，不能因数据增长明显阻塞主线程。
- Chat、Flow 和设置页按路由或功能切分；重型编辑器和 Flow 画布按需加载。
- 切换 Session 时取消过期请求，防止旧响应覆盖新页面。
- 不在生产代码保留 mock、静态假数据、未接线控件、TODO UI、`console.log` 或吞错分支。
- TypeScript 类型、API Schema 和运行时校验保持一致，不用 `any` 绕过核心领域数据。
- 文案走现有 i18n，默认中文，并符合 Pixoma Voice。

## 6. Eino Runtime

### 6.1 选择

本期使用 Eino ADK `ChatModelAgent`，实现 ReAct 语义，不采用预制 DeepAgent。

原因：

- 单 Agent 足够覆盖当前需求。
- 可复用 Eino Tool、Middleware、Interrupt/Resume 和 Checkpoint。
- Session、资产和权限继续由 Pixoma 控制。

审批和补充输入复用 Eino ADK Human-in-the-Loop 的 Interrupt/Resume 语义；Pixoma 负责把 Checkpoint、审批记录和恢复令牌持久化，并转换为 AG-UI Interrupt Outcome。

Eino 通过 `AgentEngine` Port 隔离，便于升级或替换。

### 6.2 Middleware 顺序

```text
Session Context
→ Prompt Compiler
→ Skill Loader
→ Capability Discovery
→ Policy / Approval
→ Trace
→ Checkpoint
→ Asset / Flow Projection
```

### 6.3 单 Run 生命周期

```text
保存用户消息
→ 创建 queued Run
→ Worker Claim
→ 编译上下文
→ Eino ReAct
→ 模型或 Tool Event
→ 持久化 Message / Step / Event / Checkpoint
→ 转换为 AG-UI Event
→ 更新 Asset / Flow
→ completed / waiting / failed / cancelled
```

## 7. Model Adapter

### 7.1 协议适配

- OpenAI Responses：使用 `instructions`、Item 和原生 Tool Call。
- OpenAI Chat Completions 兼容：使用 System/User/Assistant/Tool Messages。
- Anthropic Messages：系统提示词放顶层 `system`，不能伪造 `system` role message。

统一内部请求表达：

- System Instructions。
- 多模态 Messages。
- Tool Definitions。
- Reasoning Config。
- Output Constraints。
- Usage / Finish Reason。

### 7.2 能力判定

模型可用能力取以下交集：

```text
管理员声明 ∩ 能力测试成功 ∩ 协议适配器支持
```

未验证 Tool Calling 的模型不能作为主 Agent 模型。图片生成是独立能力，不根据模型名称猜测。

### 7.3 密钥

- 使用现有 Crypto 基础设施加密。
- 前端只收到掩码与 `has_secret`。
- 更新时空值表示保持原密钥，显式动作才清除。
- Run 快照不保存密钥和完整认证 Header。

## 8. Prompt 与上下文编译

### 8.1 层级

每次 Run 按以下顺序构建：

1. Pixoma 内置运行契约。
2. 当前 Agent 配置版本。
3. 权限模式和 Tool Policy。
4. 显式选择或自动命中的 Skill。
5. Session 历史摘要。
6. 最近消息。
7. 当前 Flow 语义摘要。
8. Session 资产索引。
9. 本轮引用资产内容。
10. 当前可用 Tool Schema。
11. 当前用户消息。

优先级：

```text
平台运行契约
> 管理员 Agent 指令
> 权限和安全策略
> Skill 指令
> 用户请求
> MCP Resource / 外部文件内容
```

Skill、MCP、资产和外部页面均作为不可信内容封装，不能生成高优先级指令。

### 8.2 上下文预算

- 每个 Agent 模型必须配置 `context_window_tokens`、`max_input_tokens` 和 `max_output_tokens`；Agent 模型缺失或非法时拒绝保存/运行，不根据模型名猜测限制。
- 会话对话预算为 `floor((min(max_input_tokens, context_window_tokens - max_output_tokens) - reserved_tokens) × 0.9)`；`reserved_tokens` 包含 System Prompt 和 Tool Schema。
- 本地 Token 估算是保守启发式，不承诺与提供方 tokenizer 一致；提供方返回上下文过长错误时，Agent 会强制缩短消息并重试一次。
- 保留当前消息、最近消息、未完成约束和 Flow 摘要。
- 旧消息生成版本化 Session Summary。
- 压缩按“旧读/查工具结果清理 → LLM 摘要 → 最近轮次窗口 → 保留当前消息的硬截断”降级，Tool 消息必须与对应的 Assistant Tool Call 成对存在。
- Trace 不进入模型上下文。
- 资产索引常驻，内容按需读取。
- 大文本按段读取。
- 图片仅在模型支持视觉且本轮需要时注入。
- Context Snapshot 保存版本 ID、Hash 和摘要，不复制全部大内容。

## 9. Tool Registry

### 9.1 内置 Tool

- `list_session_assets`
- `read_asset`
- `create_text_asset`
- `update_text_asset`
- `generate_image`
- `edit_session_flow`
- `search_capabilities`
- `load_skill`

“保存到资产库”不是 Agent Tool。

### 9.2 工作流 Tool

工作流 Tool Definition 从现有 Case 构建：

- 名称：根据 Case ID 生成稳定内部名称。
- 描述：复用 `CaseDocument.Name` 和 `Description`。
- Input Schema：复用 `InputSchema`。
- 字段说明：复用 `Inputs.Description`。
- 输入资产类型：从 `Inputs.Type` 推导。
- 输出资产类型：从 `Outputs.Type` 和 `MediaType` 推导。

Case 进入 Tool Catalog 的条件：

```text
case.Enabled && case.AgentCallable
```

不向模型暴露 Comfy Workflow JSON 和 Binding 内部结构。

工作流较多时，先通过 `search_capabilities` 搜索候选，再按需加载 Tool Schema。

### 9.3 Tool 幂等

- 每个 Tool Step 使用稳定 `idempotency_key`。
- 恢复时先查询 Step 状态，完成的 Tool 不重复执行。
- 只读、声明幂等的 Tool 可自动重试。
- 外部写 Tool 默认不自动重试。

## 10. Skill

### 10.1 Runtime 选择

复用 Eino ADK 官方 Skill Middleware，不自行实现 Skill 发现与加载循环。Pixoma 实现其 `Backend` 接口，将数据库中的 SkillVersion 映射为 `List` 与 `Get`。

本期只使用 Inline 模式：

- 不提供 `AgentHub` 与 `ModelHub`。
- 不接受 `context: fork` 或 `fork_with_context`。
- 不接受 Skill 内的 Agent 或模型覆盖。
- 不向 Skill 暴露文件执行器、Shell 或脚本 Tool。
- Skill 仍通过同一个 ChatModelAgent 和统一 Tool Registry 执行。

这能复用 Eino 官方渐进式加载能力，同时保持单 Agent 和轻量 Skill 边界。

### 10.2 数据与包

Skill 元数据保存在数据库，版本包存数据库文本与 Blob：

```text
SKILL.md
references/
assets/
```

不支持 `scripts/`。不建立 `skill_dependencies`。

### 10.3 渐进式加载

1. 初始 Context 只注入启用 Skill 的名称和描述。
2. 用户显式选择或模型匹配后调用 `load_skill`。
3. Runtime 加载固定版本的完整 `SKILL.md`。
4. References 和 Assets 按需读取。
5. Run Snapshot 记录实际 SkillVersion ID。

Skill 中的能力名称只能作为检索提示，不能绕过 Tool Registry 和 Policy Engine。

## 11. MCP Client

### 11.1 Transport

本期实现 `streamable_http`。领域类型预留 `stdio`，但：

- 不实现进程执行器。
- Admin API 拒绝创建该类型。
- UI 不展示该选项。

### 11.2 凭据

本期支持系统级：

- None。
- Bearer Token。
- Custom Headers。
- OAuth 2.1 系统授权。

数据模型预留 `credential_owner_type=user`，本期 API 拒绝用户级授权。

### 11.3 能力语义

- Tools：模型控制，调用前经过 Policy Engine。
- Resources：应用或用户选择后作为不可信上下文读取。
- Prompts：显式使用，不作为系统提示词覆盖层。

发现结果保存为快照，并支持 Tool/Resource/Prompt 单项启停。MCP Tool 使用连接器命名空间，避免名称冲突。

### 11.4 网络安全

- 仅允许 HTTP/HTTPS。
- 限制重定向次数。
- 默认拒绝云元数据地址和 Link-local 地址。
- 私网和本地地址需要管理员显式允许。
- DNS 解析与实际连接目标都做检查，避免 DNS Rebinding。
- 请求与响应设大小、超时和并发限制。

## 12. 审批与策略

### 12.1 模式

```go
type ApprovalMode string

const (
    ApprovalAsk       ApprovalMode = "ask"
    ApprovalAutoReview ApprovalMode = "auto_review"
    ApprovalFullAccess ApprovalMode = "full_access"
)
```

Policy Engine 输出：

```text
allow
deny
require_approval
auto_review
```

### 12.2 恢复

需要审批时：

1. 保存 Run Step。
2. 保存 Eino Checkpoint。
3. 创建 Approval。
4. 发出 AG-UI Interrupt。
5. Run 进入 `waiting_approval`。
6. 用户或 Reviewer 决策。
7. 从同一 Checkpoint 恢复。
8. 使用原 Step 幂等键执行 Tool。

Auto Review 使用独立 Reviewer 模型。模型不可用、超时、解析失败或策略不确定时失败关闭。

Full Access 跳过审批提示，但不能绕过 Deny、RBAC、Schema、连接器禁用和所有权检查。

## 13. Session Flow

### 13.1 与 Trace 分离

- Flow：用户可编辑的创作状态。
- Trace：不可变的运行事实。
- Operation Node 可选保存 `source_run_step_id`，只用于跳转。
- 删除或重排 Flow 不修改 Run、Step 和 Event。

### 13.2 Agent 命令

`edit_session_flow` 只接受语义命令，不接受绝对坐标：

```json
{
  "operations": [
    {"type": "create_stage", "title": "排好分镜"},
    {"type": "attach_asset", "asset_version_id": "...", "stage_id": "..."},
    {"type": "connect_nodes", "edge_type": "input", "source": "...", "target": "..."}
  ]
}
```

服务端创建默认位置。前端用户移动使用 Flow REST API 写入坐标和 `user_positioned=true`。

### 13.3 Revision

- `studio_session_flows.revision` 单调递增。
- PATCH 请求携带 `base_revision`。
- 服务端事务内校验并应用批量操作。
- 冲突返回最新 Revision 和 Snapshot。
- 本期不做多人合并；同一用户多 Tab 使用后写失败提示。

## 14. Asset

### 14.1 身份与版本

- Asset 表示内容身份。
- AssetVersion 表示不可变内容版本。
- 文本可内联保存，媒体与大文件使用 BlobRef。
- `content_hash` 用于完整性与可选去重，不作为用户可见身份。

### 14.2 Session 与 Library

- `studio_session_assets` 固定 Session 使用的 AssetVersion。
- `studio_library_assets` 指向 Asset 与当前版本。
- 保存到资产库只新增 LibraryRef，不复制 Blob。
- Flow Asset Node 固定 AssetVersion ID。

### 14.3 上传安全

- 复用现有 `media_max_bytes` 限制，并允许 AI 配置进一步收紧。
- 校验声明 MIME、探测 MIME 和扩展名。
- 可执行文件不作为可预览资产。
- 预览使用受控 Content-Type 和 Content-Disposition。
- 为后续病毒扫描预留状态，不在本期承诺扫描服务。

## 15. 数据库

所有 JSON 字段在 GORM 层使用文本存储和显式序列化，保持现有数据库驱动兼容。

### 15.1 Session 与 Run

`studio_sessions`

| 字段 | 说明 |
|---|---|
| `id` | UUID/ULID |
| `owner_user_id` | ConsoleUser ID |
| `title` | 标题 |
| `preferred_model_profile_id` | Session 主模型偏好 |
| `approval_mode` | 三档权限模式 |
| `active_run_id` | 活跃 Run，可空 |
| `last_message_at` | 历史排序 |
| `unread_result_count` | 未查看结果数 |
| `version` | CAS 版本 |
| `created_at` / `updated_at` | 时间 |

`studio_messages`

| 字段 | 说明 |
|---|---|
| `id` | Message ID |
| `session_id` | Session |
| `run_id` | 所属 Run，可空 |
| `role` | user / assistant / tool / reasoning |
| `content_json` | 结构化 Parts |
| `status` | streaming / complete / error / cancelled |
| `sequence` | Session 内顺序 |
| `model_snapshot_json` | 可空，不含密钥 |
| `created_at` / `completed_at` | 时间 |

`studio_runs`

| 字段 | 说明 |
|---|---|
| `id` | Run ID |
| `session_id` | Session |
| `user_message_id` | 触发消息 |
| `status` | queued / running / waiting_* / completed / failed / cancelled |
| `agent_config_version_id` | 固定 Agent 配置 |
| `model_snapshot_json` | 固定模型与能力 |
| `approval_mode` | 本 Run 权限模式 |
| `reasoning_visibility` | 本 Run 展示模式 |
| `context_snapshot_json` | 上下文版本与 Hash |
| `lease_owner` / `lease_until` | Worker 租约 |
| `attempt` | 恢复次数 |
| `last_sequence` | 最新事件序号 |
| `error_code` / `error_message` | 失败摘要 |
| `started_at` / `completed_at` / `created_at` | 时间 |

`studio_run_steps`

- `id`、`run_id`、`parent_step_id`。
- `kind`、`capability_name`、`status`。
- `input_json`、`output_json`。
- `idempotency_key`、`external_task_id`。
- `started_at`、`completed_at`。

`studio_run_events`

- `id` 单调主键。
- `session_id`、`run_id`、`sequence`。
- `event_type`、`payload_json`、`created_at`。
- `(run_id, sequence)` 唯一索引。

`studio_run_checkpoints`

- `run_id`、`checkpoint_version`、`checkpoint_data`、`created_at`。

`studio_approvals`

- `id`、`run_id`、`step_id`、`status`。
- `request_json`、`reviewer_type`、`review_model_snapshot_json`。
- `decision_json`、`decided_by`、`expires_at`、时间。

`studio_session_summaries`

- `id`、`session_id`、`through_message_sequence`。
- `summary`、`model_snapshot_json`、`created_at`。

### 15.2 Asset 与 Library

`studio_assets`

- `id`、`owner_user_id`、`kind`、`name`。
- `source_type`、`source_ref`、时间。

`studio_asset_versions`

- `id`、`asset_id`、`version`。
- `media_type`、`blob_ref`、`text_content`。
- `metadata_json`、`content_hash`、`created_by`、`created_at`。
- `(asset_id, version)` 唯一。

`studio_session_assets`

- `session_id`、`asset_id`、`asset_version_id`、`source`、`added_at`。

`studio_library_folders`

- `id`、`owner_user_id`、`parent_id`、`name`、`sort_order`、时间。
- `(owner_user_id, parent_id, name)` 唯一。

`studio_library_assets`

- `id`、`owner_user_id`、`folder_id`、`asset_id`、`current_version_id`、时间。

### 15.3 Flow

`studio_session_flows`

- `session_id` 主键。
- `revision`。
- `viewport_json`。
- `updated_at`。

`studio_flow_nodes`

- `id`、`session_id`、`type`、`parent_id`。
- `title`、`description`、`status`、`sort_order`。
- `position_x`、`position_y`、`width`、`height`。
- `asset_version_id`。
- `operation_type`、`operation_ref`、`source_run_step_id`。
- `user_positioned`、`created_by`、时间。

`studio_flow_edges`

- `id`、`session_id`、`type`。
- `source_node_id`、`target_node_id`。
- `source_handle`、`target_handle`、`created_at`。

### 15.4 AI 配置

`ai_model_connections`

- 协议、Base URL、加密凭据、自定义 Header、健康状态和时间。

`ai_model_profiles`

- 连接、模型 ID、显示名称、能力 JSON。
- `enabled`、`agent_enabled`、默认 Reasoning Effort、测试结果。

`ai_agent_config_versions`

- 版本、管理员指令、各用途模型、权限设置、Reasoning 和运行限制。
- `active`、创建者和时间。

`ai_skills`

- 名称、描述、启停、当前版本和时间。

`ai_skill_versions`

- Skill、版本、`SKILL.md`、Package Manifest、创建者和时间。

`ai_mcp_connections`

- Transport、Endpoint、认证密文、健康状态、启停和时间。

`ai_mcp_items`

- 连接、类型、远端名称、Schema/Metadata 快照。
- 启停和调用策略。

现有 `cases` 增加：

```text
agent_callable boolean not null default false
```

该字段属于 Case 可用性，不放进 CaseDocument。

## 16. API

### 16.1 Session 与 Run

```text
POST   /api/v1/studio/sessions
GET    /api/v1/studio/sessions
GET    /api/v1/studio/sessions/{id}
PATCH  /api/v1/studio/sessions/{id}
GET    /api/v1/studio/sessions/{id}/snapshot

POST   /api/v1/studio/agent
GET    /api/v1/studio/runs/{id}/events?after_sequence=
POST   /api/v1/studio/runs/{id}/cancel
GET    /api/v1/studio/runs/{id}/trace
```

`/studio/agent` 接收标准 AG-UI `RunAgentInput`：

- `threadId` 对应 Session ID。
- `runId` 用作幂等 ID。
- `messages` 只接受本轮新增用户消息。
- `state` 接受已验证的模型、权限模式、资产引用和 Skill 选择。
- `resume` 用于 Interrupt 恢复。

后端按 Session ID 从数据库重建历史，不信任客户端上传的 System/Assistant 历史。

### 16.2 Flow

```text
GET   /api/v1/studio/sessions/{id}/flow
PATCH /api/v1/studio/sessions/{id}/flow
```

PATCH 使用 `base_revision + operations[]`，事务内原子执行。

### 16.3 资产

```text
GET  /api/v1/studio/sessions/{id}/assets
POST /api/v1/studio/sessions/{id}/assets/text
POST /api/v1/studio/sessions/{id}/assets/upload
GET  /api/v1/studio/assets/{id}
GET  /api/v1/studio/assets/{id}/versions/{versionId}
POST /api/v1/studio/assets/{id}/versions
POST /api/v1/studio/assets/{id}/save-to-library

GET   /api/v1/studio/library/folders
POST  /api/v1/studio/library/folders
PATCH /api/v1/studio/library/folders/{id}
GET   /api/v1/studio/library/assets
POST  /api/v1/studio/library/assets
GET   /api/v1/studio/library/assets/{id}
PATCH /api/v1/studio/library/assets/{id}
POST  /api/v1/studio/sessions/{id}/library-assets/{assetId}/attach
```

### 16.4 AI 设置

```text
/api/v1/ai/model-connections
/api/v1/ai/model-profiles
/api/v1/ai/agent-config
/api/v1/ai/skills
/api/v1/ai/mcp-connections
```

关键动作：

```text
POST /model-connections/{id}/test
POST /model-connections/{id}/discover-models
POST /model-profiles/{id}/test-capabilities
POST /agent-config/test
POST /skills/{id}/test
POST /skills/{id}/publish
POST /mcp-connections/{id}/test
POST /mcp-connections/{id}/discover
PATCH /mcp-connections/{id}/items/{itemId}
PATCH /api/v1/cases/{id}/agent-access
```

所有 Mutating API 支持 Idempotency-Key 或领域版本校验。

## 17. AG-UI 映射

| Pixoma 事件 | AG-UI Event |
|---|---|
| Run 开始 | `RUN_STARTED` |
| Assistant 文本 | `TEXT_MESSAGE_*` |
| Thinking | `THINKING_*` |
| Reasoning | `REASONING_*` |
| Tool 调用 | `TOOL_CALL_*` |
| Tool 结果 | `TOOL_CALL_RESULT` |
| Flow/资产变化 | `STATE_SNAPSHOT` / `STATE_DELTA` |
| 审批 | `RUN_FINISHED` + Interrupt Outcome |
| 完成 | `RUN_FINISHED` |
| 失败 | `RUN_ERROR` |
| 取消 | `RUN_CANCELLED` |

State 只包含轻量引用：

```json
{
  "flow_revision": 18,
  "flow_patch": [],
  "asset_refs": [],
  "active_run": {},
  "pending_approval": null
}
```

AG-UI 不携带 Blob、完整 Trace、密钥和大文本正文。

## 18. 后台 Worker 与恢复

### 18.1 Claim

- Worker 通过 Run 状态、Session `active_run_id`、CAS Version 和 Lease 领取。
- 同一 Run 只有一个有效 Lease Owner。
- Lease 过期后其他 Worker 可以恢复。
- 所有 Step 在执行副作用前持久化幂等键。

### 18.2 外部工作流

```text
Tool Step created
→ WorkflowRunner.Start
→ 保存 external_task_id
→ Run waiting_external
→ 释放 Worker Lease
→ Poll/Callback 发现 Task 终态
→ Run 重新 queued
→ 读取 Checkpoint Resume
```

不得用单个 goroutine 等待数分钟工作流。

### 18.3 Event Replay

- Domain 状态和 Run Event 在同一事务更新。
- In-process Notifier 只负责唤醒订阅者，不是事实来源。
- SSE 首先查询 `sequence > after_sequence`，再订阅新事件。
- 客户端按 `(run_id, sequence)` 去重。
- 发现序号缺口时重新拉取 Session Snapshot。

## 19. 停止、失败与重试

- 模型网络错误可在未发生副作用前自动重试。
- 只读幂等 Tool 可以自动重试。
- 外部写 Tool 默认不自动重试。
- 停止 Run 后取消模型流和可取消 Tool。
- 外部工作流不能立即取消时进入 `cancel_requested`。
- 取消后到达的产物仍保存，但不自动成为 Flow 有效输出。
- Run 失败不删除已生成资产。
- Chat 显示可理解错误；Trace 保存分类、原始错误摘要和恢复动作。

## 20. 标题与总结

- 标题任务与主 Run 分开执行，不占用 Session 活跃 Run。
- 优先使用配置的标题模型，未配置则复用主模型。
- 失败时使用确定性截断，不重试到阻塞用户。
- Session Summary 使用配置的总结模型，未配置则复用主模型。
- Summary 固定 `through_message_sequence`，不覆盖旧版本。

## 21. 权限与安全

- 所有 Studio 查询强制 `owner_user_id = current_console_user`。
- AI 配置接口要求系统管理员角色。
- Tool 参数通过 JSON Schema 校验。
- 模型、MCP、Skill、工作流在每次调用前检查启用状态。
- 外部内容按 Prompt Injection 风险处理。
- Trace 与日志执行字段级脱敏和大小限制。
- Prompt、文件、Tool 参数和结果默认视为敏感数据，不导出到非配置的遥测系统。
- Session 权限模式不能提升后台用户 RBAC。

## 22. 用量与资源护栏

- Agent Config 保存最大 ReAct 步数、Run 超时、每用户并发和最大并行 Tool 数。
- 记录 Input/Output Token、Cache Token、Reasoning Token（提供方支持时）、图片数和工作流调用次数。
- 费用为估算值，模型价格缺失时显示“不可估算”，不按零费用处理。
- 本期不实现余额扣减和金额硬配额。

## 23. 测试策略

### 23.1 后端单元与契约

- OpenAI Responses、Chat Completions 和 Anthropic Messages 请求转换。
- Anthropic 顶层 `system`。
- 能力声明与测试结果交集。
- Context 优先级、摘要和资产按需加载。
- Workflow CaseDocument 到 Tool Schema 转换。
- Skill 渐进加载和版本固定。
- MCP 名称空间、Schema、凭据和策略。
- 三档审批及 Auto Review 失败关闭。
- Tool 幂等、Checkpoint 和 Resume。
- AssetVersion 不可变与 LibraryRef。
- Flow Revision、语义操作和 Trace 隔离。
- RBAC、Owner Scope、密钥脱敏和 SSRF 防护。

### 23.2 AG-UI 契约

- 使用官方 Schema 样例验证 Go 序列化结果。
- 文本、Thinking、Reasoning、Tool、State、Interrupt 和错误顺序。
- `RUN_FINISHED` Interrupt Outcome。
- SSE 中断、重连、重复事件和 Sequence 缺口。
- assistant-ui History 恢复审批元数据。

### 23.3 Worker 集成

- 浏览器断开后继续执行。
- 进程重启、Lease 过期和另一个 Worker 恢复。
- 工作流 Task 完成后重新入队。
- 同一 Tool 不因恢复执行两次。
- Cancel 与延迟到达的外部结果。

### 23.4 前端

- 单入口 Studio 路由。
- Session 与资产库模式切换。
- Composer 模型和权限选择。
- assistant-ui AG-UI 消息与审批渲染。
- Flow 拖动、连线、Revision 冲突、Undo/Redo。
- 删除节点不删除资产。
- 文本资产新版本。
- Trace Drawer。
- 设置四页的权限、健康状态和密钥掩码。
- 现有 Pixoma shadcn/ui 组件复用检查；不得出现重复基础组件。
- assistant-ui 默认样式隔离与 Pixoma 主题一致性。
- 亮色、暗色、920px 断点和无页面级横向滚动。
- loading、empty、error、permission denied、offline、reconnecting、stale 和 retry 状态。
- hover、focus、active、selected、disabled、dragging 和 reduced-motion。
- 键盘导航、焦点管理、读屏名称、表单错误关联和 WCAG 2.1 AA 对比度。
- 长文本、多附件、大量消息、大量资产和大型 Flow 的性能与布局压力测试。

前端合并门槛：

- TypeScript、ESLint、单元测试、组件测试和生产构建全部通过。
- 不存在未接线按钮、演示数据、临时 TODO、控制台错误和未处理 Promise rejection。
- 对照 UX 第 16 节逐项验收，本期范围内不得以“后续补齐”关闭缺口。

## 24. 开发前 Spike

### Spike A：Eino → AG-UI

验收：

- 文本和 Tool 流事件顺序正确。
- Interrupt 保存 Checkpoint。
- Resume 不新建业务 Run、不重复 Tool。

### Spike B：assistant-ui 后台恢复

验收：

- 关闭页面时服务端 Run 继续。
- 重新打开后先恢复 History/Snapshot，再从 Sequence 续接。
- 不重复渲染 Message 和 Tool Call。

### Spike C：现有工作流恢复

验收：

- Agent Run 保存现有 Task ID 后释放 Worker。
- Task 成功、失败、取消都能恢复 Agent。
- 输出映射为 AssetVersion，并建立 Flow Input/Output Edge。

Spike 未通过前不进入大规模页面和业务开发。

## 25. 技术依据

- [Eino ADK ChatModelAgent](https://www.cloudwego.io/docs/eino/core_modules/eino_adk/agent_implementation/chat_model/)
- [Eino ADK Human-in-the-Loop](https://www.cloudwego.io/docs/eino/core_modules/eino_adk/agent_hitl/)
- [Eino ADK Skill Middleware](https://www.cloudwego.io/docs/eino/core_modules/eino_adk/eino_adk_chatmodelagentmiddleware/middleware_skill/)
- [Eino Tool 组件](https://github.com/cloudwego/eino/blob/main/components/tool/doc.go)
- [AG-UI 协议](https://github.com/ag-ui-protocol/ag-ui/blob/main/docs/ag_ui.md)
- [assistant-ui AG-UI Runtime](https://www.assistant-ui.com/docs/runtimes/ag-ui/overview)
- [assistant-ui AG-UI Runtime Options](https://www.assistant-ui.com/docs/runtimes/ag-ui/runtime-options)
- [AG-UI HttpAgent 重连缺口](https://github.com/ag-ui-protocol/ag-ui/issues/1852)
- [AG-UI HttpAgent 当前实现](https://github.com/ag-ui-protocol/ag-ui/blob/main/sdks/typescript/packages/client/src/agent/http.ts)
- [Eino AG-UI 适配提案](https://github.com/cloudwego/eino-ext/issues/881)
- [OpenAI Agent 审批与安全](https://learn.chatgpt.com/docs/agent-approvals-security?translationFallback=zh-Hans)
- [Anthropic Messages API](https://platform.claude.com/docs/en/api/messages/create)
- [MCP Tools](https://modelcontextprotocol.io/specification/2025-06-18/server/tools)
- [MCP Resources](https://modelcontextprotocol.io/specification/2025-06-18/server/resources)
- [MCP Prompts](https://modelcontextprotocol.io/specification/2025-06-18/server/prompts)
