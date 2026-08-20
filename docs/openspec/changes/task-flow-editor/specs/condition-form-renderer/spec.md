## Purpose

按条件属性目录（`/api/v1/routing/attributes`）自动渲染条件配置表单：字段类型、可选项、说明与 UI 标签均由 schema 驱动，新增属性无需修改前端代码。

## ADDED Requirements

### Requirement: 属性目录驱动条件表单
条件配置表单 MUST 从条件目录 API 拉取属性描述（key、JSON Schema、UI 标签）并按 schema 渲染输入控件（文本/数字/布尔/枚举）与校验；未知或未注册属性 MUST 在表单中不可选。

#### Scenario: 按 schema 渲染枚举
- **WHEN** 属性 `case.category` 的 schema 声明枚举 `["image","video","audio"]`
- **THEN** 表单渲染为可选项控件，仅允许枚举值

#### Scenario: 新增属性自动出现
- **WHEN** 后端新增注册属性 `user.region` 并出现在条件目录 API
- **THEN** 前端无需改动即可在条件表单中选择该属性

### Requirement: 规则组合编辑与回读
表单 MUST 支持 `and`/`or` 组合编辑（可嵌套），并能将组合规则渲染回表单；保存时 MUST 输出符合条件协议的 JSON 规则。

#### Scenario: 组合规则回读
- **WHEN** 打开条件为 `and(user.is_premium, case.category in [image])` 的已有规则
- **THEN** 表单展示对应组合结构并可编辑保存，输出 JSON 与原规则等价
