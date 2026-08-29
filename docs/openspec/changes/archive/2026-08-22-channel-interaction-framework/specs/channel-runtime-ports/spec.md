## MODIFIED Requirements

### Requirement: 消息平台端口契约
系统 MUST 定义消息平台运行时端口，至少覆盖：入站事件（消息/回调查询/媒体/按钮点击）、出站消息（文本/菜单/选项列表/媒体）、媒体桥（上传/下载/转存）与身份映射（外部用户 → 内部用户）。入站事件中的动作 MUST 为能力调用形态（capability_id + params），MUST NOT 是具体业务专属动作。应用层 MUST 仅依赖端口类型，MUST NOT 依赖具体平台 SDK。

#### Scenario: 应用层不感知平台
- **WHEN** 新增一个平台适配器且不改动业务能力应用层
- **THEN** 应用层代码不出现该平台 SDK 类型

## ADDED Requirements

### Requirement: 交互动作泛化为能力调用
系统 MUST 将端口层的交互动作由工作流专属（如打开 Case、确认）泛化为能力调用（capability_id + params + 消息平台账户上下文）；适配器把 UI 事件翻译为能力调用，业务结果以消息平台无关结构返回。

#### Scenario: 事件翻译为能力调用
- **WHEN** 适配器收到按钮点击事件
- **THEN** 输出包含 capability_id 与 params 的能力调用，而非业务专属动作
