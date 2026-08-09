## Purpose

为五类管理资源提供页面信息架构与基础管理交互，并通过 admin-api 完成真实数据操作。

## ADDED Requirements

### Requirement: 资源页 Master–Detail 布局
实例、Case、Task、User、Session 管理页 MUST 采用左右分栏 Master–Detail：左侧列表、右侧详情或表单；未选中时右栏为空态；窄屏允许降级为顺序整页。

#### Scenario: 选中列表项刷新右栏
- **WHEN** 运维在某一资源页左侧列表选中一项
- **THEN** 右侧展示该资源详情或可编辑表单，且无需离开当前资源页进入无关整页壳

### Requirement: 实例管理页
控制台 MUST 提供实例列表/详情或表单页，支持基础 CRUD 与观测入口（对接 admin-api 实例接口）。

#### Scenario: 从 UI 完成实例列表与创建
- **WHEN** 运维在实例页查看列表并提交合法新建
- **THEN** UI 展示更新后的列表且数据来自 admin-api

### Requirement: Case 管理页
控制台 MUST 提供 Case 列表与编辑/创建/启用禁用相关页面交互；创建与编辑 MUST 以结构化多段表单为主路径（非整页 JSON 编辑器作为唯一入口）。

#### Scenario: 从 UI 禁用 Case
- **WHEN** 运维在 Case 页对某 Case 执行禁用
- **THEN** UI 反映禁用状态且请求发往 admin-api

#### Scenario: 结构化编辑 Case
- **WHEN** 运维打开 Case 创建或编辑页
- **THEN** 页面以分段表单展示基础信息、输入输出与绑定等字段，提交后数据经 admin-api 持久化

### Requirement: User / Session / Task 运维页
控制台 MUST 提供 User、Session、Task 的列表与详情页；Task 页 MUST 支持取消等已由 API 提供的运维动作。

#### Scenario: 查看用户与会话详情
- **WHEN** 运维从列表进入某 User 或 Session 详情
- **THEN** 页面展示 admin-api 返回的关键字段

#### Scenario: 从 UI 取消任务
- **WHEN** 运维在 Task 详情对可取消任务执行取消
- **THEN** UI 反映取消结果或明确错误

### Requirement: Dashboard 中等总览
控制台 MUST 提供 Dashboard 页：数字卡片与简单状态/占比分布；数据 MUST 来自 admin-api 既有列表类接口的前端聚合（本期不新增统计专用端点）。

#### Scenario: Dashboard 展示聚合信息
- **WHEN** 用户打开 Dashboard 且 admin-api 可用
- **THEN** 页面展示至少实例与 Task（或 Case）相关的汇总卡片或分布，且网络请求指向 admin-api

#### Scenario: Dashboard 卡片失败隔离
- **WHEN** 某一汇总依赖的 API 请求失败
- **THEN** 仅对应卡片或区块进入错误/空态，其它区块仍可展示

### Requirement: 仅通过 admin-api 通信
前端 MUST 仅通过配置的 admin-api 基址访问管理数据，MUST NOT 直连数据库或 bot 管理残留路径作为正式方案，MUST NOT 依赖独立前端 mock 层作为交付验收路径。

#### Scenario: API 基址可配置
- **WHEN** 开发者配置 admin-api 基址并打开资源页
- **THEN** 网络请求指向该基址下的管理 API
