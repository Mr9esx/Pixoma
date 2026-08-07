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
