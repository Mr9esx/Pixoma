# Comet Design Handoff

- Change: case-comfy-notify-path
- Phase: design
- Mode: compact
- Context hash: 43b04b22a3dc64ac2fef60c93bbfc4628b993dd84e76e6c6c6738131e00b33f8

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## docs/openspec/changes/case-comfy-notify-path/proposal.md

- Source: docs/openspec/changes/case-comfy-notify-path/proposal.md
- Lines: 1-30
- SHA256: 612bb4517336ba6b2b8d1d6701cbf2d1939c28066605bb4c81813f0352fe1672

```md
## Why

一期骨架虽已有 Case→ConfirmRun→Mock→通知冒烟，但执行面仍用空 `StaticWorkflows`，不加载 Case `bindings.workflow`、不注入已物化输入；TG 只收文本，协议声明的 `image` 输入无法采集。结果是「开始 Case → 输入 → Comfy 执行 → 取产物 → 完成 → 通知」在真实语义与真实 Comfy 下走不通，图片输入更是断链。

## What Changes

- Actuator 按 Task→Case 加载 workflow 模板，按 `bindings.inputs` 注入文本与图片（图片先入 Blob，再按绑定提交给 Comfy；Mock 与真实 HTTP 共用同一注入路径）
- 保持 `comfy_mock` / `COMFY_MOCK` 一键开关；Mock 开时端到端成功，关且 Comfy 可达时走真实 Submit/Wait
- Case 协议与种子：明确支持「文本 + 图片」混合输入的 Case（含校验、绑定与至少一个可跑种子）
- TG：当前字段为 `image` 时接收 Photo（及必要的 document/图片文件），写入 Session Draft 的 Blob；`number`/`boolean` 按字段类型解析；发图文件名不含路径分隔符
- 补齐接线与回归：集成冒烟覆盖「文本+图片 → 执行 → 通知」；映射缺失/空图在调用 Comfy 前失败上报

## Capabilities

### New Capabilities

（无）

### Modified Capabilities

- `workflow-protocol`: 强化文本+图片混合 Case 的协议/校验与绑定约定（image 输入物化后可被执行面消费）
- `comfyui-executor`: 从 Case 快照加载图并注入文本/图片；Mock 与真实客户端共用；禁止空 stub 冒充成功路径
- `channel-tg`: 采集图片输入、按类型解析标量、可靠投递产物图
- `dialog-session`: Session 草稿支持图片 Blob 值进入后续 ConfirmRun 物化

## Impact

- 代码：`actuator`（CaseSnapshot/注入）、`comfyui`（必要时图片上传）、`main` 接线、`channel/tg`、Case 种子 JSON、集成测试
- 配置：`comfy_mock`、`comfyui_base_url`、真实 Comfy 可用的 workflow fixture/种子
- 非目标：Session/Task 持久化、视频输入采集、多实例调度、管理后台

```

## docs/openspec/changes/case-comfy-notify-path/design.md

- Source: docs/openspec/changes/case-comfy-notify-path/design.md
- Lines: 1-52
- SHA256: 53d9566a8405a14c8f046f3fa09816409c1c95a78885af0bf8e8c9e9aa233be7

```md
## Context

参见 `proposal.md` 的 Why。当前仓库已有 Catalog（SQLite）、ConfirmRun 物化输入到 Blob、Orchestrator 同步内存队列、Comfy `NewClient(Mock|HTTP)`，但 `main` 将 Actuator `Workflows` 设为 `StaticWorkflows{}`，不读 Case、不注入；TG 仅注册文本 Update。规格层面一期已写明注入与 image 类型，实现未闭环。

## Goals / Non-Goals

**Goals（设计层）**

- 单一执行路径：Task → Case 快照 → 加载 staged 输入 → 注入 → Client.Submit/Wait → Blob 产物 → status
- 文本与图片注入共用绑定模型；图片在客户端侧完成 upload（真实）或等价引用（Mock）
- TG 按 Case 字段类型分流文本/图片 Update

**Non-Goals（设计层）**

- Session/Task 持久化与进程崩溃恢复
- 视频采集与复杂多图相册编辑
- 更换队列实现或引入分布式调度

## Decisions

1. **CaseSnapshot 提供方替代 StaticWorkflows 默认空图**  
   - 选择：`CaseSnapshot{Tasks, Cases, Blob}` 实现 `WorkflowForTask`，深拷贝 `bindings.workflow` 后注入。  
   - 备选：ConfirmRun 时把图塞进 StaticWorkflows map — 拒绝，因与 Case 版本/重试脱节。

2. **图片注入：真实 Comfy 先 Upload 再写节点字段；Mock 接受占位文件名**  
   - 选择：在执行面注入阶段，对 `image` 类型调用 Client 扩展或旁路 `UploadImage`（若现有 Client 无此能力则最小扩展接口）；Mock 返回稳定假名。  
   - 备选：只把本地路径塞进图 — 真实 Comfy 不可用。

3. **TG 媒体：RegisterHandlers 增加 Photo/Document 匹配，按当前字段类型提交 Blob Draft**  
   - 选择：Adapter 查当前 Case 字段类型；`image` 才收下媒体。  
   - 标量：`number`/`boolean` 在 Adapter 解析后再 `SubmitInput`。

4. **种子 Case**  
   - 至少一个「文本+图片」混合 Case（mock 可用 Stub 图 + 绑定）；另保留可切真实 Comfy 的 workflow fixture（或文档化的最小 API graph），由 `comfy_mock: false` 验收。

5. **发图文件名**  
   - `filepath.Base(ref.Key)`（或等价），避免 Telegram 拒收。

## Risks / Trade-offs

- [真实 Comfy 节点/模型环境差异] → 提供可配置种子 graph + mock 默认；真实路径以「绑定注入 + HTTP 提交成功」为验收，不强绑定特定 checkpoint 名  
- [图片 Document MIME 多样] → 先支持 Photo + 常见 image/* Document；其余明确拒绝  
- [Client 接口扩展] → 保持 Mock 同步演进（`comfy_mock` 规则）

## Migration Plan

- 部署后默认 `comfy_mock: true` 行为向好（注入后仍出 mock 图）；关 mock 需本机 Comfy 与种子 graph  
- 回滚：可临时切回 StaticWorkflows，但本 change 验收不依赖该回滚路径

## Open Questions

- 真实 Comfy 种子 graph 以仓库内 fixture 为准，还是运行时从外部路径加载？（实现时优先仓库内最小 fixture，外部路径可作为后续增强）

```

## docs/openspec/changes/case-comfy-notify-path/tasks.md

- Source: docs/openspec/changes/case-comfy-notify-path/tasks.md
- Lines: 1-28
- SHA256: 4b4b2ea1e3a49f2bb8bbb4ccadf1106776d4d5b41d52104c3e745004a90ae551

```md
## 1. Case 快照与注入

- [ ] 1.1 实现 CaseSnapshot：按 Task 加载 Case，深拷贝 `bindings.workflow`，从 InputPrefix 读取 staged 文本/数值/布尔/图片元数据
- [ ] 1.2 实现按 `bindings.inputs` 注入；必填缺失、绑定缺失、空 workflow、节点不存在时在 Submit 前失败并上报 status
- [ ] 1.3 为文本注入与映射失败编写单元测试（先红后绿）

## 2. Comfy 客户端与图片

- [ ] 2.1 扩展 Client（Mock + HTTP）以支持图片上传/引用；Mock 返回稳定假名并保持 `comfy_mock` 开关
- [ ] 2.2 注入路径对 `image` 字段走上传后再写节点；补充 Mock/HTTP 选型与上传相关测试
- [ ] 2.3 在 `main` 将 Worker.Workflows 接到 CaseSnapshot（Tasks/Cases/Blob），移除默认空 StaticWorkflows 成功主路径

## 3. 协议种子与校验

- [ ] 3.1 增加或更新「文本 + 图片」混合 Case 种子（含 bindings）；校验拒绝「图片字段仅文本」
- [ ] 3.2 提供可关 mock 验收的最小真实 workflow fixture（或文档化仓库内 graph），与种子绑定一致

## 4. TG 采集与投递

- [ ] 4.1 RegisterHandlers 支持 Photo（及常见 image Document）；当前字段为 image 时写入 Blob Draft
- [ ] 4.2 当前字段为 number/boolean 时解析标量；image 字段收到普通文本时提示而非误写入
- [ ] 4.3 SendPhoto 使用无路径分隔符的安全文件名；补适配器测试

## 5. 端到端验收

- [ ] 5.1 更新/新增集成冒烟：ConfirmRun → 注入可观测 → Mock 成功 → notify（含图片输入场景）
- [ ] 5.2 验证 `comfy_mock: true` 主路径与 `comfy_mock: false` 在可达 Comfy（或 HTTP fixture）下 Submit/Wait 行为符合规格
- [ ] 5.3 全量相关包 `go test` 通过，并确认 Mock 与真实路径代码同步演进

```

## docs/openspec/changes/case-comfy-notify-path/specs/channel-tg/spec.md

- Source: docs/openspec/changes/case-comfy-notify-path/specs/channel-tg/spec.md
- Lines: 1-36
- SHA256: 60e51f0609ca7e9e12562c3538053704d490f8e6d2a6e294a94bb6043b759c67

```md
## ADDED Requirements

### Requirement: 当前输入为图片时采集 Telegram 媒体
当会话当前待填字段类型为 `image` 时，适配器 MUST 接受用户发送的 Photo（以及可识别的图片 Document），下载并写入 Blob，再以 Blob Draft 提交给应用层。此时 MUST NOT 把任意纯文本当作该图片字段的合法值（引导文案与跳过规则除外）。

#### Scenario: 用户发送照片填入参考图
- **WHEN** 当前 Case 字段为必填 `image`，用户发送一张 Photo
- **THEN** 适配器将该图物化为 Blob 并推进会话到下一输入或确认态

#### Scenario: 图片字段收到无关文本时提示
- **WHEN** 当前字段为 `image` 且用户发送非命令普通文本
- **THEN** 适配器提示需要发送图片，而不把该文本写入图片 Draft

### Requirement: 按字段类型解析标量文本
当当前字段类型为 `number` 或 `boolean` 时，适配器 MUST 将用户文本解析为对应 Draft 类型后再提交；解析失败 MUST 提示用户重试，不得以错误类型进入 ConfirmRun 校验。

#### Scenario: 数字种子解析成功
- **WHEN** 当前字段为 `number`，用户发送合法整数字符串
- **THEN** 草稿以数值形式保存且后续校验可通过

## MODIFIED Requirements

### Requirement: TG 适配器将应用 DTO 渲染为 Bot API 消息
系统 MUST 提供 Telegram 适配器：接收 Bot Update，调用渠道无关的应用用例，并将菜单/Case 列表/会话提示/错误/结果渲染为 Telegram 支持的消息形态（文本、Photo、InlineKeyboard 等）。应用层 MUST NOT 依赖 Telegram SDK 类型。发送 Photo 时文件名 MUST 为无路径分隔符的安全基名，避免因 blob key 含 `/` 导致投递失败。

#### Scenario: Case 列表以按钮呈现
- **WHEN** 用户进入某分类的 Case 列表
- **THEN** 适配器发送包含 Case 入口的 InlineKeyboard（可分页）

#### Scenario: 完成后发送图片结果
- **WHEN** 适配器收到 succeeded 的用户通知且输出含 image BlobRef
- **THEN** 适配器向对应用户发送图片（或等价媒体消息）

#### Scenario: 产物 Photo 使用安全文件名
- **WHEN** BlobRef.Key 含目录前缀（如 `outputs/task/0_out.png`）
- **THEN** 发往 Telegram 的上传文件名仅为基名（如 `0_out.png`）

```

## docs/openspec/changes/case-comfy-notify-path/specs/comfyui-executor/spec.md

- Source: docs/openspec/changes/case-comfy-notify-path/specs/comfyui-executor/spec.md
- Lines: 1-35
- SHA256: e901f1e0ac65c17b5eb7de7f33151822f7132e408788aaf6b16f7af2ea96747d

```md
## MODIFIED Requirements

### Requirement: Actuator 作为执行面调用 ComfyUI
系统 MUST 提供 Actuator 模块：消费指向本实例的 dispatch 命令，按 Case 快照中的绑定将输入注入工作流图，调用配置的 ComfyUI 客户端（Mock 或真实 HTTP，由配置开关选择），并等待完成或失败。Actuator MUST NOT 使用与 Case 无关的空 stub 图作为成功主路径的默认行为；MUST NOT 直接更新全局 Task 表的终态；MUST 通过 status 事件向 Orchestrator 上报。

#### Scenario: 收到 dispatch 后执行文生图
- **WHEN** Actuator 收到合法 dispatch 且 ComfyUI 客户端可用，且 Case 含有效 workflow 与注入后的图
- **THEN** 系统向 ComfyUI 提交工作流，并上报至少一次 running 与一次终态 status（succeeded 或 failed）

#### Scenario: ComfyUI 不可达时上报失败 status
- **WHEN** ComfyUI 实例不可达或返回连接错误（真实 HTTP 模式）
- **THEN** Actuator 上报 failed status（含可诊断信息），且不假装成功

#### Scenario: Mock 开关开启时仍完成主路径
- **WHEN** `comfy_mock`（或等价环境变量）为真
- **THEN** 同一注入与 status 路径可成功完成并产生可投递图片产物

### Requirement: 按绑定映射注入输入
系统 MUST 依据 case 中声明的节点/字段注入映射，将逻辑 input 键映射到 ComfyUI 工作流图中的具体节点入参。文本 MUST 注入为对应字段值；图片 MUST 基于已物化 Blob，经客户端要求的上传/引用步骤后注入。缺少映射、映射目标不存在、或必填物化输入缺失时，系统 MUST 在调用 ComfyUI 前失败并上报 failed status。

#### Scenario: 文本注入到指定节点
- **WHEN** case 将 `prompt` 映射到某节点的文本字段，且 staged 输入含该文本
- **THEN** 提交给 ComfyUI 的工作流中该节点字段等于已物化输入中的 prompt 值

#### Scenario: 图片注入到指定节点
- **WHEN** case 将 `reference`（image）映射到某节点图片字段，且 staged 输入含对应 Blob
- **THEN** 提交前完成上传或等价引用，且工作流中该字段指向可用的图片输入

#### Scenario: 映射缺失导致拒绝执行
- **WHEN** 某必填 input 没有有效的 ComfyUI 注入映射
- **THEN** 不调用 ComfyUI，并上报指出映射问题的 failed status

#### Scenario: 空 workflow 拒绝执行
- **WHEN** Case 快照中 `bindings.workflow` 为空或不含可提交图
- **THEN** 不调用 ComfyUI，并上报失败 status

```

## docs/openspec/changes/case-comfy-notify-path/specs/dialog-session/spec.md

- Source: docs/openspec/changes/case-comfy-notify-path/specs/dialog-session/spec.md
- Lines: 1-12
- SHA256: 8005dd32b2dad6b778addd9231f641cfcbfaf54b05916a56cd7bbddacff62ec9

```md
## ADDED Requirements

### Requirement: Session 草稿支持图片 Blob 值
对话会话 MUST 允许将某个 input 键的 Draft 存为媒体 Blob 引用（与文本/数值/布尔并存）。跳过规则对允许跳过的非必填图片字段仍然有效；必填图片缺少 Blob 时不得进入可成功 Confirm 的状态。

#### Scenario: 提交图片草稿后索引前进
- **WHEN** 会话处于 collecting，当前键类型语义为图片，且提交带 Blob 的 Draft
- **THEN** 该键被记录，会话前进到下一输入或 confirming

#### Scenario: 跳过可选图片字段
- **WHEN** 当前键为非必填且允许跳过的图片字段，用户选择跳过
- **THEN** 会话不要求该 Blob，并前进到下一输入或 confirming

```

## docs/openspec/changes/case-comfy-notify-path/specs/workflow-protocol/spec.md

- Source: docs/openspec/changes/case-comfy-notify-path/specs/workflow-protocol/spec.md
- Lines: 1-33
- SHA256: 96d3524f531065239f62794b46d3d0911794dd1b1e729896734ed87d1b6c2f77

```md
## MODIFIED Requirements

### Requirement: Input schema 支持多类型采集与可选跳过
系统 MUST 将 input 定义为有序数组。每个 input 项 MUST 包含：字段键、类型、是否必填、描述、可选 preview，以及在非必填时可被跳过的标记。本阶段类型 MUST 至少支持：`string`（文本）、`image`、`video`、`number`、`boolean`、`enum`。混合 Case（同时包含 `string` 与 `image`）MUST 合法；校验 MUST 要求必填 `image` 携带可解析的媒体引用（Blob），不得仅用文本冒充图片。

#### Scenario: 必填文本输入不可跳过
- **WHEN** text2img case 的 prompt 输入标记为必填
- **THEN** 协议将该输入视为不可跳过，缺少值时校验失败

#### Scenario: 非必填输入允许跳过
- **WHEN** case 中某输入标记为非必填且允许跳过
- **THEN** 调用方可不提供该输入值，校验在其余必填项满足时仍可通过

#### Scenario: 文本加图片混合 Case 通过校验
- **WHEN** case 的 input schema 同时定义必填 `string`（如 prompt）与必填 `image`（如 reference），且提交值分别为文本与媒体 Blob
- **THEN** 协议校验通过，二者均可供执行面物化与注入

#### Scenario: 必填图片仅有文本时校验失败
- **WHEN** 必填 `image` 输入只收到文本而无 Blob
- **THEN** 校验失败并指出该字段需要媒体

#### Scenario: img2img 需要图片与可选文本
- **WHEN** case 的 input schema 依次定义 image（必填）与 string（可选）
- **THEN** 协议要求先满足图片输入，文本可缺省或跳过

## ADDED Requirements

### Requirement: 图片输入绑定可指向 Comfy 节点入参
系统 MUST 允许 `bindings.inputs` 将逻辑 `image` 键映射到工作流节点字段；绑定缺失时执行面 MUST 在提交 Comfy 前失败（行为由执行面规格约束，协议 MUST 允许声明此类绑定）。

#### Scenario: 参考图绑定声明完整
- **WHEN** case 为 `reference`（image）声明 `node_id` 与 `field_path`
- **THEN** 协议接受该绑定作为可注册 Case 的一部分

```
