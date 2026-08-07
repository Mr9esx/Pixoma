## Context

空仓库起步，技术栈为 Golang。产品最终形态是可部署的 TG Bot 服务，终端用户通过对话调用 ComfyUI（文生图、图像/视频编辑等）。本期只交付工作流核心：协议、基于 ORM 的注册仓储、单实例 ComfyUI 执行器。TG 对话、积分扣费、ComfyUI 调度均明确延后，但协议需预留对话与展示所需元数据（preview、描述、价格数值、可跳过输入等）。

## Goals / Non-Goals

**Goals:**

- 定义可扩展的 Case/Workflow 协议（input/output schema、校验、元数据、类别标签、ComfyUI 绑定）
- 用 ORM + SQLite 注册/查询/更新/停用 case，模型可迁移到 MySQL
- 单实例 ComfyUI：注入输入、执行、按 output schema 回收 image/text/file
- 覆盖已知场景标签示例，并以 schema 驱动扩展，而不是写死 case 类型实现

**Non-Goals:**

- TG 对话状态机与 Bot 部署平台
- 真实积分/支付
- ComfyUI 多实例调度
- 管理后台 UI

## Decisions

### D1. 模块划分

采用清晰分层：

```
protocol/     # 协议模型、schema、校验
registry/     # ORM 实体、仓储、迁移
comfyui/      # HTTP 客户端、注入、轮询、产物回收
app/api 或 cmd/ # 最小调用入口（HTTP 或 CLI，实现阶段二选一偏 HTTP 便于联调）
```

协议与执行解耦：registry 存协议文档 + 绑定；executor 只消费「已校验输入 + case 快照」。

### D2. ORM 选型：GORM

- **选择**：GORM + SQLite 驱动，配置层可切换 `mysql` DSN。
- **理由**：生态成熟、迁移工具够用、后续换 MySQL 成本低。
- **备选**：ent（更强类型，样板更多）、sqlx（偏手写 SQL）。本期优先交付速度与可切换性，选 GORM。

### D3. 协议载体

- **选择**：领域用 Go struct；持久化可将完整 case 文档存 JSON 列，并用关键列索引 `id`、状态、类别标签。
- **理由**：schema 会演进，整文档 JSON 降低改表频率；标签可用关联表或 JSON 查询策略（SQLite 下关联表更稳）。
- **备选**：每个 schema 字段拆列——过早结构化，迁移重。

### D4. 类别标签 vs 固定枚举

- **选择**：`tags/categories` 为字符串集合；预置文档示例含 text2img、text2video、img2img、img2video、videoedit、imgedit，并可扩展 upscale、inpaint、remove-bg 等。
- **理由**：真实差异在 input/output schema 与工作流图，不在类型名。

### D5. Input / Output 类型基线

- Input：`string` | `image` | `video` | `number` | `boolean` | `enum`
- Output：`image` | `text` | `file`（视频走 file，可附 `media_type`）
- 每个 input：`key`、`type`、`required`、`skip_allowed`（仅当非必填）、`description`、`preview`、校验约束

### D6. ComfyUI 绑定模型

- Case 保存：工作流模板（API prompt graph）+ `input_bindings`（logical key → node_id + field_path）+ `output_bindings`（logical key → 产物定位方式，如 node/output index 或 filename 前缀规则）。
- 执行时：深拷贝模板 → 注入 → `/prompt` 提交 → 轮询 history/等待 → 拉产物。

### D7. 价格字段

- 仅 `price`（数值，单位由上层约定，本期不解释货币/积分换算）。
- 不调用任何扣费接口。

### D8. 本期入口形态

- 提供最小 HTTP API 或内部 Service 接口用于注册、查询、校验、执行；不实现 TG webhook。
- 具体路由在实现任务中细化，设计层要求「可被后续 Bot 层直接调用的应用服务接口」。

## Risks / Trade-offs

- **[Risk] ComfyUI 工作流图与节点字段因版本/自定义节点而异** → 绑定外置到 case 定义；提供清晰映射错误；用 1～2 个样例工作流做集成测试。
- **[Risk] 大文件（视频）内存压力** → 产物落本地临时目录/对象路径，结果返回引用；避免无缓冲超大文件。
- **[Risk] SQLite 与 MySQL 方言差异** → 只用 GORM 可移植特性；避免原始 SQLite JSON 特有 SQL。
- **[Risk] 协议过早绑死 TG 交互** → 协议只表达数据与校验语义；对话编排留给后续模块。
- **[Trade-off] JSON 整文档存储 vs 强列结构** → 灵活优先，列表过滤靠索引列/标签表补偿。

## Migration Plan

1. 引入 Go module、配置、GORM 自动迁移。
2. 落地协议校验与样例 case（至少 text2img、一个含 image 输入的 case）。
3. 接通单实例 ComfyUI 执行与产物回收。
4. 后续切 MySQL：改 DSN + driver，跑迁移；不改协议字段语义。
5. 回滚：停用新版本二进制；SQLite 文件可备份恢复。

## Open Questions

- 最小对外入口最终用 HTTP 还是仅 Go API（库）+ CLI：建议 HTTP，待实现前可按联调需要微调，不阻设计。
- 图片/视频输入是传本地路径、URL 还是 multipart 字节：建议 Service 层统一为「已物化的本地文件引用或字节」，由上层适配 TG/HTTP。
- `price` 的小数精度与单位命名（`price` vs `credit_cost`）：本期用 `price` 数值，单位文档说明即可。
