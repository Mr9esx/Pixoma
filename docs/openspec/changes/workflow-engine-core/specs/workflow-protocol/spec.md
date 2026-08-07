## ADDED Requirements

### Requirement: Case 协议定义可执行的工作流契约
系统 MUST 提供 Case/Workflow 协议，用于描述一个可注册、可校验、可执行的工作流 case。每个 case MUST 至少包含：稳定标识、显示名称、描述、可选 preview 资源引用、价格数值字段、一个或多个类别标签、有序的 input schema 列表、有序的 output schema 列表，以及到 ComfyUI 工作流定义的绑定信息（含节点注入映射）。

#### Scenario: 注册前协议字段齐全
- **WHEN** 调用方提交一个完整的 text2img case 定义
- **THEN** 系统接受该定义，并保留其元数据、input/output schema 与 ComfyUI 绑定信息以供后续查询与执行

#### Scenario: 缺少必填协议字段被拒绝
- **WHEN** 调用方提交缺少标识、input schema 或 ComfyUI 绑定的 case 定义
- **THEN** 系统拒绝该定义并返回可定位缺失字段的错误

### Requirement: Input schema 支持多类型采集与可选跳过
系统 MUST 将 input 定义为有序数组。每个 input 项 MUST 包含：字段键、类型、是否必填、描述、可选 preview，以及在非必填时可被跳过的标记。本阶段类型 MUST 至少支持：`string`（文本）、`image`、`video`、`number`、`boolean`、`enum`。后续 TG 对话可按该数组顺序逐项引导；本阶段仅保证协议与校验语义成立。

#### Scenario: 必填文本输入不可跳过
- **WHEN** text2img case 的 prompt 输入标记为必填
- **THEN** 协议将该输入视为不可跳过，缺少值时校验失败

#### Scenario: 非必填输入允许跳过
- **WHEN** case 中某输入标记为非必填且允许跳过
- **THEN** 调用方可不提供该输入值，校验在其余必填项满足时仍可通过

#### Scenario: img2img 需要图片与可选文本
- **WHEN** case 的 input schema 依次定义 image（必填）与 string（可选）
- **THEN** 协议要求先满足图片输入，文本可缺省或跳过

### Requirement: Output schema 支持图片、文本与文件扩展
系统 MUST 将 output 定义为有序数组。每个 output 项 MUST 包含：字段键、类型、描述。类型 MUST 至少支持：`image`、`text`、`file`。视频等二进制产物 MAY 以 `file`（或带媒体提示的 file 元数据）表示，以便调用方按 schema 回收结果。

#### Scenario: 文生图产出图片
- **WHEN** text2img case 的 output schema 声明一个 image 输出
- **THEN** 执行成功后结果中必须能按该字段键取回图片内容或可访问引用

#### Scenario: 文生视频产出文件
- **WHEN** text2video case 的 output schema 声明一个 file 输出（视频）
- **THEN** 执行成功后结果中必须能按该字段键取回对应文件内容或可访问引用

### Requirement: 输入校验遵循 schema 的表单规则
系统 MUST 使用 **JSON Schema**（或与其语义等价的引擎）对提交的输入集合执行校验，规则至少覆盖：必填、类型匹配、`number` 的最小/最大（若配置）、`string` 的最小/最大长度（若配置）、`enum` 的枚举集合约束。对 `image`/`video` 等媒体输入，系统 MUST 在 Schema 校验之外提供扩展钩子校验引用可解析性与允许的 MIME（若配置）。校验失败 MUST 返回按字段聚合的错误信息，且不得创建执行 Task 或向 ComfyUI 提交工作流。

#### Scenario: 类型不匹配导致失败
- **WHEN** schema 要求 image，但调用方提供了纯文本值
- **THEN** 校验失败并指出该字段类型错误，不触发执行

#### Scenario: 全部输入合法则通过
- **WHEN** 调用方按 schema 提供全部必填输入且类型与约束均满足
- **THEN** 校验通过，输入可进入 ConfirmRun 物化与后续编排

#### Scenario: JSON Schema 引擎拒绝非法枚举
- **WHEN** enum 字段取值不在 schema 枚举集合内
- **THEN** 校验失败且不创建 Task

### Requirement: 价格字段仅为协议数值元数据
系统 MUST 在 case 上提供数值型价格/费用字段以供展示或后续积分系统读取。本阶段 MUST NOT 实现扣费、余额校验或支付流程。

#### Scenario: 查询 case 可见价格数值
- **WHEN** 调用方读取已注册 case
- **THEN** 响应包含协议中的价格数值字段，且不产生任何扣费副作用

### Requirement: 场景类别可扩展而不锁死实现集合
系统 MUST 允许 case 使用类别标签表达场景（例如 text2img、text2video、img2img、img2video、videoedit、imgedit，以及后续可能的 upscale、inpaint、remove-bg 等）。协议 MUST NOT 将可运行能力硬编码为固定枚举实现集合；具体能力由该 case 的 input/output schema 与 ComfyUI 绑定决定。

#### Scenario: 使用已知标签注册 case
- **WHEN** 调用方以 `text2img` 标签注册 case
- **THEN** 系统保存该标签，并仍以 schema 为准决定输入输出行为

#### Scenario: 使用新标签扩展场景
- **WHEN** 调用方以尚未内置示例的标签（如 `upscale`）注册合法 case
- **THEN** 系统接受该 case，不因标签不在预置列表示例中而拒绝
