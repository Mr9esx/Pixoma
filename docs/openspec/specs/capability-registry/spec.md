# capability-registry Specification

## Purpose
能力注册表把「业务能做什么」从渠道和菜单中抽象出来：每个能力声明其展示名、参数 schema、执行入口与各渠道的渲染方式；新增业务模块只需注册能力，无需改动适配器或菜单模型。
## Requirements
### Requirement: 能力声明
系统 MUST 提供能力注册表，每个能力至少声明：能力 id（kebab-case）、展示名、参数 schema、执行 handler 标识、各渠道渲染声明。注册表 MUST 支持运行时查询与按渠道筛选，供菜单配置和适配器使用。

#### Scenario: 注册新能力
- **WHEN** 新增一个能力（如 checkin）并完成注册
- **THEN** 无需改动适配器或菜单模型，即可在菜单配置中引用并触发

#### Scenario: 查询能力清单
- **WHEN** 管理台或适配器请求能力清单
- **THEN** 返回已注册能力的 id、展示名、参数 schema 与渠道渲染声明

### Requirement: 参数 schema
能力 MUST 以 **JSON Schema** 声明其参数（draft-07 子集），管理台据此渲染参数表单、适配器据此校验事件载荷；非法参数 MUST 被拒绝并返回可理解错误。

#### Scenario: 按 schema 校验参数
- **WHEN** 菜单项或事件携带不符合 schema 的参数
- **THEN** 系统拒绝并返回参数错误说明

### Requirement: 渠道渲染声明
能力 MUST 在能力定义内按渠道提供默认渲染声明（入口形态：主键盘直达 / 消息按钮 / 分组；展示配置如每行按钮数）；菜单项 MAY 通过渲染覆盖（render override）微调指定渠道的配置，覆盖语义为按 key 合并。渲染声明按渠道隔离；未声明某渠道渲染的能力在该渠道不展示。

#### Scenario: 按渠道展示能力入口
- **WHEN** 同一能力在 TG 与后续飞书渠道消费
- **THEN** 各自按本渠道的渲染声明展示，无需改写能力定义

### Requirement: 内置能力
系统 MUST 内置 `open_case` 能力（执行现有 Case 工作流：预览/开始/填表/确认/结果）；`open_case` 参数 MUST 支持 case 选择上下文（如 back 引用），以承载文件夹/返回语义。

#### Scenario: open_case 走既有工作流
- **WHEN** 用户点击 open_case 能力入口
- **THEN** 进入既有 Case 预览与填表工作流，行为等价保留

