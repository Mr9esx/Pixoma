---
status: draft-for-review
product: Pixoma Studio
role: technical-design
date: 2026-09-20
---

# Pixoma Studio 技术设计

## 1. 总体架构

```text
React Admin
├── assistant-ui + react-ag-ui
├── Studio Shell
├── React Flow
└── Asset Library UI
        │ AG-UI / REST
        ▼
Agent HTTP Boundary
├── AG-UI Run Endpoint
├── Session API
├── Asset API
└── Model / Tool Catalog API
        │
        ▼
Agent Application
├── Session Service
├── Run Service
├── Context Builder
├── Policy Engine
├── Asset Service
└── Trace Projector
        │
        ▼
Eino Runtime
├── ReAct Agent
├── Model Adapter Registry
└── Tool Registry
    ├── Workflow Tool Adapter
    ├── Model Generation Adapter
    ├── Skill Adapter
    ├── MCP Client Adapter
    └── Asset Tool Adapter
```

Eino 负责推理循环；AG-UI 负责 Agent 与前端通信；Pixoma 负责 Session、Run、权限、资产、Trace 和持久化。

## 2. Go 模块

建议新增：

```text
internal/agent/
├── domain/
│   ├── session.go
│   ├── message.go
│   ├── run.go
│   ├── trace.go
│   ├── model_connection.go
│   └── tool.go
├── application/
│   ├── session_service.go
│   ├── run_service.go
│   ├── context_builder.go
│   ├── policy.go
│   └── title_service.go
├── infrastructure/
│   ├── eino/
│   ├── agui/
│   ├── models/
│   ├── persistence/
│   └── tools/
└── protocol/
    ├── dto.go
    └── state.go

internal/assets/
├── domain/
├── application/
└── infrastructure/

internal/httpapi/studio/
```

现有 `internal/mcp` 继续作为 Pixoma 对外 MCP Server。Studio 的 MCP 能力使用独立 MCP Client Adapter，不反向依赖该 Server 包。

## 3. 领域接口

```go
type AgentRuntime interface {
    Run(ctx context.Context, input RunInput, sink EventSink) error
    Resume(ctx context.Context, input ResumeInput, sink EventSink) error
    Cancel(ctx context.Context, runID RunID) error
}

type ModelRegistry interface {
    ListAvailable(ctx context.Context, actor Actor) ([]ModelProfile, error)
    Resolve(ctx context.Context, connectionID string) (ModelAdapter, error)
}

type ToolRegistry interface {
    List(ctx context.Context, scope ToolScope) ([]ToolDefinition, error)
    Resolve(ctx context.Context, name string) (Tool, error)
}

type WorkflowExecutor interface {
    Describe(ctx context.Context, workflowID string) (WorkflowDefinition, error)
    Start(ctx context.Context, cmd WorkflowRunCommand) (WorkflowRunRef, error)
    Get(ctx context.Context, ref WorkflowRunRef) (WorkflowRunStatus, error)
    Cancel(ctx context.Context, ref WorkflowRunRef) error
}

type AssetService interface {
    Create(ctx context.Context, cmd CreateAssetCommand) (*AssetVersion, error)
    CreateVersion(ctx context.Context, cmd CreateVersionCommand) (*AssetVersion, error)
    AttachToSession(ctx context.Context, sessionID SessionID, ref AssetVersionRef, role AssetRole) error
    SaveToLibrary(ctx context.Context, ref AssetVersionRef, folderIDs []FolderID) error
}
```

Workflow Tool 只依赖 `WorkflowExecutor`，不直接依赖 `botapp.Facade`。适配器可以复用现有 Task 调度和 Blob 能力。

## 4. 核心数据结构

```go
type Session struct {
    ID                SessionID
    OwnerUserID       string
    Title             string
    ModelConnectionID string
    Status            SessionStatus
    CreatedAt         time.Time
    UpdatedAt         time.Time
}

type Run struct {
    ID             RunID
    SessionID      SessionID
    TriggerMessage MessageID
    Status         RunStatus
    IdempotencyKey string
    ModelSnapshot  ModelSnapshot
    LastSequence   int64
    StartedAt      time.Time
    CompletedAt    *time.Time
    Error          *RunError
}

type TraceEvent struct {
    ID        int64
    RunID     RunID
    Sequence  int64
    Type      string
    Payload   json.RawMessage
    CreatedAt time.Time
}

type Asset struct {
    ID              AssetID
    OwnerUserID     string
    OriginSessionID *SessionID
    Visibility      AssetVisibility
    Kind            AssetKind
    Source          AssetSource
    CurrentVersion int
    CreatedAt       time.Time
    UpdatedAt       time.Time
}

type AssetVersion struct {
    AssetID       AssetID
    Version       int
    Name          string
    MIME          string
    Size          int64
    BlobRef       string
    TextContent   *string
    SourceRunID   *RunID
    SourceStepID  *string
    CreatedBy     string
    CreatedAt     time.Time
}

type ToolDefinition struct {
    Name          string
    Kind          ToolKind
    Description   string
    InputSchema   json.RawMessage
    OutputSchema  json.RawMessage
    Risk          RiskLevel
    PolicyKey     string
    AssetMappings []AssetMapping
}

type RunStep struct {
    ID            StepID
    RunID         RunID
    ParentStepID  *StepID
    Type          StepType
    Name          string
    Status        StepStatus
    InputSummary  json.RawMessage
    OutputSummary json.RawMessage
}
```

`OriginSessionID` 表达资产从哪里产生，`Visibility` 表达当前可从哪里访问。Session 内生成的资产初始为 `session` 可见；保存到资产库后变为 `library` 可见，但来源 Session 和版本血缘不变。资产库资产引入 Session 时创建 `agent_session_assets` 固定版本引用，不复制 Asset。

## 5. 数据库表

### 5.1 Agent

`agent_sessions`

| 字段 | 说明 |
|---|---|
| `id` | Session ID |
| `owner_user_id` | 后台用户 ID |
| `title` | AI 生成或备用标题 |
| `model_connection_id` | 当前模型连接 |
| `status` | idle / running / waiting_approval / failed |
| `created_at` / `updated_at` | 时间 |

`agent_messages`

| 字段 | 说明 |
|---|---|
| `id` | Message ID |
| `session_id` | Session ID |
| `run_id` | 可空，所属 Run |
| `role` | user / assistant / system / tool |
| `content_json` | 完整多模态消息 |
| `created_at` | 时间 |

`agent_runs`

| 字段 | 说明 |
|---|---|
| `id` | Run ID |
| `session_id` | Session ID |
| `trigger_message_id` | 触发消息 |
| `status` | queued / running / waiting_approval / waiting_external / succeeded / failed / cancelled |
| `model_snapshot_json` | 连接与模型快照，不含密钥 |
| `idempotency_key` | 客户端重试去重键，同一 Session 内唯一 |
| `worker_id` / `lease_expires_at` | 后台执行租约 |
| `last_sequence` | 已持久化事件的最新序号 |
| `version` | 状态 CAS 版本 |
| `error_json` | 错误信息 |
| `started_at` / `completed_at` | 时间 |

`agent_run_steps`

- Run 内 Flow 节点的查询投影，不作为执行真相源。
- 字段包括 `id`、`run_id`、`parent_step_id`、`step_type`、`name`、`status`、`input_summary_json`、`output_summary_json`、`started_at`、`completed_at`。
- `step_type` 支持 message / model / tool / workflow / approval / asset。

`agent_tool_calls`

- Tool 调用的结构化索引，字段包括 Tool 类型、名称、参数摘要、结果摘要、风险等级、审批 ID、外部任务 ID、状态和耗时。
- 完整原始事件仍进入 `agent_trace_events`；索引表用于 Flow、筛选和运营查询。

`agent_trace_events`

| 字段 | 说明 |
|---|---|
| `id` | 单调主键 |
| `run_id` | Run ID |
| `sequence` | Run 内顺序，唯一索引 |
| `event_type` | AG-UI 或 Pixoma 投影事件类型 |
| `payload_json` | 原始事件数据 |
| `created_at` | 时间 |

`agent_event_outbox`

- 与领域状态、Trace Event 在同一事务写入。
- Dispatcher 按 `run_id + sequence` 发布 AG-UI 事件，记录投递状态与重试次数。
- 客户端按 Sequence 去重；投递可以至少一次，状态变化必须幂等。

`agent_approvals`

| 字段 | 说明 |
|---|---|
| `id` | Approval ID |
| `run_id` | Run ID |
| `tool_call_id` | Tool Call ID |
| `policy_key` | 权限策略键 |
| `request_json` | 展示信息 |
| `decision` | pending / allow_once / allow_session / deny |
| `decided_by` / `decided_at` | 处理人和时间 |

### 5.2 模型与 Tool

`agent_model_connections`

- 名称、provider、base URL、模型能力、密钥密文、是否默认、是否启用。
- 密钥使用现有加密基础设施。

`agent_session_permissions`

- Session 内已允许的策略键与范围。

Skill 与 MCP 的配置表按后续连接器设计细化；Tool Registry 对调用方提供统一定义。

### 5.3 资产

`assets`

- ID、所有者、来源 Session、可见范围、类型、来源、当前版本、时间。
- `visibility` 为 session / library；来源 Session 与可见范围不能互相替代。

`asset_versions`

- `(asset_id, version)` 联合主键。
- 名称、MIME、大小、BlobRef、可选文本内容、来源 Run/Step、创建者和时间。

`agent_session_assets`

- Session、Asset、固定版本、角色、是否固定上下文、关联时间。

`asset_lineage_edges`

- `from_asset_id/version`、`to_asset_id/version`、关系类型、Run、Step。

`asset_library_entries`

- Asset、保存人、保存时间；表示资产进入资产库。

`asset_folders`

- ID、父文件夹、名称、所有者和时间。

`asset_folder_memberships`

- Folder 与 Asset 多对多关系。

### 5.4 约束、索引与事务

- `agent_trace_events(run_id, sequence)` 唯一，确保事件可按 Run 有序重放。
- `agent_runs(session_id, idempotency_key)` 唯一，避免发送重试创建重复 Run。
- `agent_runs(status, lease_expires_at)` 建 Worker 领取索引。
- `agent_messages(session_id, created_at, id)`、`agent_run_steps(run_id, started_at)` 建历史查询索引。
- `asset_versions(asset_id, version)` 唯一；新版本号在事务内分配。
- `agent_session_assets(session_id, asset_id, asset_version, role)` 唯一，引用必须指向已存在版本。
- `asset_folders(owner_user_id, parent_id, name)` 唯一，文件夹成员关系删除不级联删除 Asset。
- Run 状态、Trace Event 和 Outbox 必须在同一事务提交；Blob 上传先写临时对象，元数据提交后再确认，失败对象由清理任务回收。

## 6. AG-UI 映射

| Pixoma 事实 | AG-UI |
|---|---|
| Run 开始/结束/失败 | `RUN_STARTED` / `RUN_FINISHED` / `RUN_ERROR` |
| Agent 流式回答 | `TEXT_MESSAGE_*` |
| Tool 调用 | `TOOL_CALL_*` |
| Tool 结果 | `TOOL_CALL_RESULT` |
| Flow、资产与 Session 状态 | `STATE_SNAPSHOT` / `STATE_DELTA` |
| 运行进度 | `ACTIVITY_SNAPSHOT` / `ACTIVITY_DELTA` |
| 审批 | Interrupt + Resume |
| 特殊扩展 | 命名空间 `CUSTOM` 事件 |

协议边界以 AG-UI 官方事件 Schema 为准。Go 侧 SDK 只放在 `infrastructure/agui`，即使更换社区 SDK 或改为自行序列化，也不影响领域层、Eino Runtime 和前端事件语义。必须增加官方 Schema 样例的兼容性测试，避免 SDK 版本差异变成 Pixoma 私有协议。

### 6.1 AG-UI 使用边界与风险

| 风险 | 影响 | 设计处理 |
|---|---|---|
| Go SDK 成熟度低于 TypeScript SDK | 事件字段或升级节奏可能不一致 | 领域层不依赖 SDK；用官方 Schema 做契约测试 |
| 标准 Run 流偏向连接内流式交互 | Pixoma 的工作流可能运行数分钟并跨页面 | Run 生命周期服务端化；重连仍发送标准事件，只扩展 cursor 传输 |
| `STATE_DELTA` 不是数据库 | 断线或前端重载不能只依赖内存状态 | 服务端保存 Snapshot、Trace 和最新 Sequence |
| 大媒体进入事件流 | SSE 体积、内存和重放成本失控 | AG-UI 只传资产元数据、预览 URL 与固定版本引用 |
| assistant-ui 与右侧 Flow 状态各自维护 | Chat 与 Flow 可能出现不同口径 | 两者订阅同一服务端 State；Flow 不从 Chat DOM 推断状态 |
| Eino Interrupt 与 AG-UI Resume 语义不同 | 审批后可能错误地新建 Run | 保存 Eino checkpoint 和 resume token，由同一 Run 恢复 |

因此采用“AG-UI-first，但不是 AG-UI-only”：对话、Tool、状态、活动和中断使用标准事件；Session 列表、资产管理、Trace 查询和后台任务控制继续使用普通领域 API。

建议 AG-UI State：

```json
{
  "session": {"id": "...", "title": "...", "model": "..."},
  "run": {"id": "...", "status": "running"},
  "flow": {"nodes": [], "edges": []},
  "assets": [],
  "pendingApprovals": []
}
```

State 只包含资产元数据和引用，不包含大文件正文。敏感模型参数和密钥不进入事件流。

## 7. HTTP 接口

### 7.1 Agent 与 Session

```text
POST /api/v1/studio/sessions
GET  /api/v1/studio/sessions
GET  /api/v1/studio/sessions/{id}
POST /api/v1/studio/sessions/{id}/runs        AG-UI Run
POST /api/v1/studio/runs/{id}/resume          审批或补充输入后恢复
POST /api/v1/studio/runs/{id}/cancel
GET  /api/v1/studio/runs/{id}/events?after=   AG-UI 事件重连
GET  /api/v1/studio/runs/{id}/trace
GET  /api/v1/studio/runs/{id}/state
```

`runs` 端点接收 AG-UI Run 输入并创建服务端 Run；初次连接与重连均输出标准 AG-UI 事件。`events` 只补充后台运行所需的 cursor 传输能力，不引入新的事件格式。Session 列表、资产与 Trace 查询继续使用普通 JSON API。

### 7.2 资产

```text
GET  /api/v1/assets
POST /api/v1/assets
GET  /api/v1/assets/{id}
POST /api/v1/assets/{id}/versions
POST /api/v1/assets/{id}/library
GET  /api/v1/asset-folders
POST /api/v1/asset-folders
PATCH /api/v1/asset-folders/{id}
POST /api/v1/asset-folders/{id}/assets
DELETE /api/v1/asset-folders/{id}/assets/{assetId}
```

本期不提供 Session 归档和删除 API。

## 8. 后台运行模型

用户请求创建 Run 后，HTTP 连接只负责订阅，不拥有执行生命周期：

```text
Create Run
→ 持久化 queued
→ 后台 Worker 领取
→ Eino ReAct 执行
→ Trace/Event 持久化
→ AG-UI 广播
→ 浏览器断开不取消
```

同一 Run 只允许一个执行持有者。使用 lease 或数据库 CAS 防止重复执行。客户端重连时：

1. 读取 Run 状态与最新 State Snapshot。
2. 从已知 Sequence 后订阅增量事件。
3. 检测缺口时重新获取 Snapshot。

Worker 在执行外部 Workflow Tool 时只持久化外部 Task 引用，不用 goroutine 等待整个任务。轮询或回调处理器把 Task 状态转换为新的 Trace Event；Run 进入 `waiting_external`，Task 完成后重新入队恢复 Eino。这样进程重启不会丢失等待中的工作流。

## 9. Trace 与可观测性

Trace 至少记录：

- Run 生命周期。
- 模型连接与模型标识。
- 模型调用耗时、Token 用量与结束原因。
- Tool Call 名称、参数摘要、结果摘要和耗时。
- 工作流 Task ID 和状态。
- 审批请求与决策。
- 资产输入、输出和版本。
- 错误分类和恢复动作。

不记录明文密钥，不向前端展示原始 Chain of Thought。敏感 Tool 参数按定义做字段级脱敏。

## 10. 上下文构建

Context Builder 按以下顺序构建模型输入：

1. 系统规则与 Agent 角色。
2. Tool 定义与权限。
3. Session 摘要。
4. 最近消息。
5. 固定的 Session Asset。
6. 本次消息 Asset。
7. 必要的历史 Tool 结果摘要。

大资产只注入文本摘要、元数据或可读取引用。上下文超限时先压缩历史，不静默删除本次输入资产。

## 11. 安全

- 模型密钥只在服务端解密。
- MCP 凭据按连接器隔离。
- Tool 参数必须通过 Schema 校验。
- Asset 读取按所有者和后台角色授权。
- 外部 URL、MCP 返回和用户文件视为不可信内容。
- 高风险 Tool 由 Policy Engine 中断，不依赖模型自行判断。
- AG-UI State 和 Trace Payload 做大小限制和字段脱敏。

## 12. 测试

### 12.1 后端

- Session 首消息、标题异步生成和备用标题。
- Run 持久化、后台执行和浏览器断开继续运行。
- AG-UI 文本、Tool、State、Activity、Interrupt 事件顺序。
- 重连 Snapshot + Sequence 增量恢复。
- 审批一次、Session 允许和拒绝。
- Workflow Tool 任务关联与状态同步。
- 资产版本、Session 引用、保存到资产库和血缘。
- 模型切换与 Run 模型快照。
- Trace 脱敏和完整性。
- AG-UI 官方 Schema 样例与 Go 序列化结果的契约测试。
- Worker lease 过期、进程重启和外部 Workflow Task 完成后的恢复。

### 12.2 前端

- Studio Shell 三栏和窄屏切换。
- Session 历史与自动恢复。
- assistant-ui AG-UI Runtime 消息、Tool 和审批渲染。
- Flow 与资产 Tab 的双向联动。
- 文本资产编辑与版本保存。
- 页面切换后后台 Run 状态恢复。
- 资产库文件夹、筛选和关联 Session。

## 13. 演进边界

- 内部领域模型不依赖 assistant-ui。
- AG-UI 适配集中在 `infrastructure/agui`。
- Eino 适配集中在 `infrastructure/eino`。
- Workflow Tool 通过端口复用执行能力，不复用 MCP Server 或 Bot 对话逻辑。
- 资产服务可独立于 Agent 被其他后台模块复用。

## 14. 技术依据

- [Eino ReAct Agent](https://github.com/cloudwego/eino/blob/main/flow/agent/react/react.go)
- [Eino 项目说明](https://github.com/cloudwego/eino/blob/main/README.md)
- [AG-UI 事件](https://github.com/ag-ui-protocol/ag-ui/blob/main/docs/concepts/events.mdx)
- [AG-UI 状态同步](https://github.com/ag-ui-protocol/ag-ui/blob/main/docs/concepts/state.mdx)
- [AG-UI Go SDK](https://github.com/ag-ui-protocol/ag-ui/blob/main/docs/sdk/go/overview.mdx)
- [assistant-ui AG-UI Runtime](https://www.assistant-ui.com/docs/runtimes/ag-ui/overview)
