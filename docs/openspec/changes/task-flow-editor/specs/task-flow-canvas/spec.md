## Purpose

基于 React Flow 的任务分流画布：以有向图展示并编辑「Case 起始 → 条件判定 → Topic 目标」的单步分流语义，负责校验与序列化，产出/消费 `topic-routing` 的 Case `routing` 配置。

## ADDED Requirements

### Requirement: 画布结构与编辑语义
系统 MUST 提供 React Flow 画布展示任务分流：Case 起始节点 → 条件节点（分支）→ Topic 目标节点；编辑模式 MUST 支持新增/删除条件分支、选择目标 Topic 与调整分支顺序；只读模式 MUST 隐藏编辑能力。画布语义 MUST 表达「首个命中即投」与「无命中回退默认 Topic」。

#### Scenario: 查看已有路由
- **WHEN** 打开含两条规则（条件1→Topic A、条件2→Topic B）的 Case
- **THEN** 画布显示 Case 节点、两个条件分支与两个 Topic 目标节点，且默认 Topic 回退可见

#### Scenario: 新增条件分支
- **WHEN** 在编辑模式新增一个条件分支并选择目标 Topic
- **THEN** 画布出现新分支，保存后生成对应路由规则

### Requirement: 序列化与回读
画布保存时 MUST 按分支顺序序列化为 Case `routing` 载荷（有序 `rules`，每条含 `when` 与 `topic`）；从 API 回读 Case 路由配置时 MUST 能渲染回画布。未配置路由的 Case MUST 在画布上仅显示默认 Topic 回退语义。

#### Scenario: 保存生成路由载荷
- **WHEN** 用户编辑画布后保存
- **THEN** 提交的 Case `routing.rules` 顺序与画布分支顺序一致，且每条含合法条件与目标 Topic key

#### Scenario: 空配置回读
- **WHEN** 打开无路由配置的 Case
- **THEN** 画布显示默认 Topic 回退语义，不报错

### Requirement: 画布校验
画布保存前 MUST 校验：每个条件分支具有完整合法条件与已启用目标 Topic；引用不存在或禁用 Topic 时 MUST 阻止保存并给出可读错误（定位到对应分支）。

#### Scenario: 引用禁用 Topic 阻止保存
- **WHEN** 某分支目标 Topic 已被禁用
- **THEN** 保存被阻止，错误定位到该分支
