# Pixoma AI Studio 实施计划

> 执行方式：当前分支 `feat-studio` 顺序实施；每个行为先写失败测试，再写最小实现，阶段完成后回归并提交。

## 目标与验收口径

- 后台新增唯一入口「创作 Studio」，进入独立三栏工作台。
- 单 Agent 支持自然语言、模型调用、Skill、MCP、工作流调用、审批与后台执行。
- Session 自动保存全部消息、运行、事件、审批、资产与 Trace。
- Flow 是可编辑的 Session SOP / 资产路线，不是 Trace；默认按对话产出顺序追加。
- Session 资产可手动创建、上传、由模型生成、由工作流生成，并可保存至资产库。
- 模型配置支持 OpenAI Responses、OpenAI Chat Compatible、Anthropic Messages Compatible。
- 前端复用 Pixoma 设计系统与现有 shadcn/ui；Chat 行为使用 assistant-ui；Flow 使用 React Flow。
- 内置 Mock 工作流跑通「用户对话 → 执行工作流 → 生成资产 → Flow 展示」闭环。
- 线上 OpenAI 兼容模型仅通过运行时环境变量做验收，密钥不进入代码、日志、快照或提交。

## Task 1：Studio 领域模型与持久化

- [x] 先写领域规则测试：Session、Message、Run、Event、Approval、Asset、AssetVersion、FlowNode、FlowEdge 的状态和归属约束。
- [x] 定义独立的 `internal/studio/domain`，不复用旧聊天 Session 聚合。
- [x] 先写 SQLite/GORM 仓储失败测试，覆盖创建会话、追加消息、幂等事件、资产版本、Flow 排序和账号隔离。
- [x] 实现 GORM Rows、事务仓储、分页查询与唯一索引。
- [x] 注册所有迁移模型。
- [x] 运行 `go test ./internal/studio/...`。

## Task 2：应用服务与后台运行状态机

- [x] 先写用例测试：发送消息自动建会话、首条输入生成标题、后台 Run 生命周期、取消、重试和断线后回读。
- [x] 实现 SessionService、AssetService、FlowService、RunService。
- [x] 建立后台 Runner 与可恢复事件流；HTTP 请求结束不取消运行。
- [x] 实现三档权限策略：请求批准、帮我批准、完全访问。
- [x] 实现审批挂起/通过/拒绝与可恢复执行。
- [x] 运行并发、状态迁移与恢复测试。

## Task 3：Mock Agent 与 Mock 工作流闭环

- [x] 先写端到端服务测试，输入漫画需求后应产生助手回复、工作流 Run、输入资产、输出图片资产、Flow 节点与 Trace。
- [x] 实现可注入的 Agent/Workflow 接口。
- [x] 实现确定性 Mock Agent、Mock「分镜生成」工作流与本地 SVG/图片产物。
- [x] 验证资产内容可通过现有 Blob Store 读取。
- [x] 验证后台运行时切出页面不影响完成。

## Task 4：模型、Skill、MCP 与工作流能力注册

- [ ] 先写模型配置仓储与密钥加密测试。
- [ ] 实现模型 CRUD、连接测试、默认模型、Agent 可用开关、思考配置与能力元数据。
- [ ] 实现 OpenAI Responses、OpenAI Chat Compatible、Anthropic Messages Compatible 适配器。
- [ ] 引入 Eino，基于官方 ChatModelAgent、HITL 与 Skill Middleware 组装单 Agent。
- [ ] 实现 DB 内联 Skill CRUD、启停与提示词注入，不引入依赖图或脚本执行。
- [ ] 实现 MCP Streamable HTTP 连接器 CRUD、探测、工具发现和调用授权。
- [ ] 给 CaseDocument 增加 `agent_callable`，复用现有名称、说明、输入输出定义形成工具描述。
- [ ] 实现统一 Capability Registry，供 Agent 按启用状态装载 Skill、MCP 工具和工作流。

## Task 5：Studio HTTP API 与 AG-UI 事件桥

- [ ] 先写 Handler 测试，覆盖账号隔离、校验错误、游标分页、幂等键、SSE 断线续传和审批权限。
- [ ] 实现 Session、Message、Run、Approval、Asset、Library、Flow、Trace API。
- [ ] 实现模型、Skill、MCP 配置 API 与工作流 Agent 可用开关 API。
- [ ] 实现 Eino 事件到 AG-UI 事件的薄桥接，并提供 `Last-Event-ID`/游标续传。
- [ ] 接入 adminhost 与认证上下文。
- [ ] 输出 OpenAPI/接口数据结构文档并校验示例。

## Task 6：前端依赖、数据层与路由壳

- [ ] 通过项目 pnpm 与 shadcn CLI 核对/安装缺失组件，不引入第二套通用 UI 库。
- [ ] 安装并验证 assistant-ui 官方 AG-UI adapter；继续使用已安装的 `@xyflow/react`。
- [ ] 先写 API 客户端、Query Key、断线重连与本地草稿状态测试。
- [ ] 菜单新增唯一「创作 Studio」入口。
- [ ] 新增 Studio 独立布局与路由：对话、资产库、AI 设置。
- [ ] 完成路由级错误边界、加载骨架与权限状态。

## Task 7：生产级 Studio 工作台 UI

- [ ] 先写关键交互测试：新建对话、历史切换、发送消息、模型切换、权限切换、审批、后台运行回连。
- [ ] 实现左栏：返回、新对话、搜索/历史、资产库、AI 设置、用户信息。
- [ ] 实现中栏：标题、消息流、工具/审批卡片、附件、Composer 内模型/思考/权限选择。
- [ ] 用 assistant-ui 管理消息运行时，但视觉完全复用 Pixoma 组件与语义令牌。
- [ ] 实现空、加载、流式、停止、失败、重试、断线、后台运行和只读历史状态。
- [ ] 完成键盘操作、焦点管理、ARIA、缩放和窄屏布局。

## Task 8：Flow、资产与 Trace UI

- [ ] 先写 Flow 节点增删改、拖拽排序/定位、连线、保存与失败回滚测试。
- [ ] 用 React Flow 实现可编辑资产路线，默认按产出顺序布局，支持阶段/计划/操作/资产节点。
- [ ] 实现右栏 Flow/资产 Tab 与独立 Trace 抽屉。
- [ ] 实现 Session 资产预览、上传、创建文本/Markdown、版本、保存到资产库与作为输入引用。
- [ ] 实现资产库文件夹、搜索、筛选、详情、来源会话关联和用户上传来源。
- [ ] 完成图片、文本、Markdown、JSON、通用文件的安全预览和下载。

## Task 9：AI 设置 UI

- [ ] 先写模型、Skill、MCP、Agent 能力设置交互测试。
- [ ] 模型页：协议、Base URL、密钥、模型、能力、Agent 可用、默认模型、思考与连接测试。
- [ ] Skill 页：内联内容创建、编辑、预览、启停与测试，不展示依赖配置。
- [ ] MCP 页：Streamable HTTP 地址、系统凭证、连接测试、工具清单与启停。
- [ ] 工作流页：仅增加「可供 Agent 使用」开关，其余信息复用工作流定义。
- [ ] 敏感字段默认掩码，前端永不回显完整密钥。

## Task 10：集成、验收与交付

- [ ] 后端全量：`go test ./...`。
- [ ] 前端：`pnpm test`、`pnpm lint`、`pnpm format:check`、`pnpm build`。
- [ ] 用 Mock 工作流跑通真实浏览器交互闭环，检查刷新恢复与后台执行。
- [ ] 通过环境变量使用临时 Ark 凭据跑线上 OpenAI Compatible 冒烟测试，确保输出和流式事件正常；测试后清理临时配置。
- [ ] 按 Pixoma DESIGN 第 11 节逐项验收视觉、文案、响应式和可访问性。
- [ ] 检查无裸色值、无 TODO/console、无失效按钮、无生产 Mock 数据、无密钥泄露。
- [ ] 更新 PRD/技术设计中的实现偏差、部署配置和运维说明。
- [ ] 使用标准代码审查与安全审查处理问题。
- [ ] 检查 `git diff`，确保不包含用户原有 `apps/pixoma/internal/webembed/dist/index.html` 改动。
