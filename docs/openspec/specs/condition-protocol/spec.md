# condition-protocol Specification

## Purpose
定义可扩展的投放条件协议：属性提供方注册、声明式 JSON 规则结构与求值语义、以及由 JSON Schema 驱动的校验与表单描述，使新增条件类型无需修改任务分流引擎代码。
## Requirements
### Requirement: 条件规则为声明式 JSON
系统 MUST 以声明式 JSON 表示投放条件。单个条件 MUST 为 `{field, op, value}`；复杂条件 MUST 支持 `{and: [...]}` / `{or: [...]}` 组合。`field` MUST 引用已注册属性提供方提供的属性 key；`op` MUST 为协议内置运算（至少 `eq`、`ne`、`in`、`gt`、`gte`、`lt`、`lte`、`exists`），`value` 的类型 MUST 与该属性 schema 定义一致。

#### Scenario: 合法规则可求值
- **WHEN** 规则为 `{"and":[{"field":"user.is_premium","op":"eq","value":true},{"field":"case.category","op":"in","value":["image","video"]}]}`
- **THEN** 引擎按上下文（用户属性 + Case 属性）求值并返回布尔结果

#### Scenario: 未知字段或非法运算被拒绝
- **WHEN** 规则引用未注册字段或未定义运算
- **THEN** 规则校验失败并返回可定位错误（字段/运算名），不得静默视为 false

### Requirement: 属性提供方注册与 schema 驱动
系统 MUST 提供属性提供方注册机制：每个属性 MUST 声明稳定 key、JSON Schema（类型/枚举/描述/UI 标签）、可选的上下文来源（user/case/input）。新增条件类型 MUST 通过注册 provider + schema 完成，MUST NOT 修改引擎求值代码；管理端条件表单 MUST 由属性 schema 自动渲染（输入控件、可选项、校验）。

#### Scenario: 新属性无需改引擎
- **WHEN** 注册新属性 `user.region`（提供 schema 与取值函数）
- **THEN** 无需改动引擎代码即可在规则中使用并求值，管理端自动出现对应条件配置控件

#### Scenario: 上下文缺失按 false 处理
- **WHEN** 求值时某属性对应上下文数据缺失（如用户无 is_premium 记录）
- **THEN** 该条件求值为 false（`exists` 运算除外），不报错

### Requirement: 条件求值上下文
系统 MUST 为求值提供结构化上下文，至少包含：用户属性命名空间（本期示例 `user.is_premium`）、Case 属性命名空间（`case.category`、`case.tags`），并预留输入属性命名空间；上下文字段由 provider 声明。

#### Scenario: 按用户与 Case 属性路由
- **WHEN** 上下文为 user.is_premium=true、case.category=image
- **THEN** 规则 `user.is_premium eq true` 与 `case.category in [image, video]` 均命中

#### Scenario: provider 求值错误不被吞掉
- **WHEN** 求值某叶子条件时属性 provider 返回错误（区别于属性缺失）
- **THEN** 求值返回错误，不得静默按 false 处理

