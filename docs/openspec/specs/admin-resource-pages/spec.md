# admin-resource-pages Specification

## Purpose
为五类管理资源提供页面信息架构与基础管理交互，并通过 admin-api 完成真实数据操作。
## Requirements
### Requirement: 资源页 Master–Detail 布局
实例、Case、Task、User、Session 管理页 MUST 采用左右分栏 Master–Detail：左侧列表、右侧详情或表单；未选中时右栏为空态；窄屏允许降级为顺序整页。

#### Scenario: 选中列表项刷新右栏
- **WHEN** 运维在某一资源页左侧列表选中一项
- **THEN** 右侧展示该资源详情或可编辑表单，且无需离开当前资源页进入无关整页壳

### Requirement: 左栏紧凑筛选
带列表筛选的 Master–Detail 左栏 MUST 遵循紧凑筛选：可搜索字段合并为单一搜索框（无字段 Label）；封闭枚举 MUST 使用横向可滑 Segment，MUST NOT 用「Label + 全宽 Select」作为默认形态。细则见 `docs/frontend/admin-list-filters.md`。

#### Scenario: Case 左栏筛选
- **WHEN** 运维打开 Case 列表
- **THEN** 可见单一搜索（覆盖 id / 名称）与启用状态横向 Segment

#### Scenario: Task / Session 状态筛选
- **WHEN** 运维在 Task 或 Session 左栏切换状态
- **THEN** 状态以横向可滑 Segment 呈现，且列表区域仍为左栏主要可视高度

#### Scenario: User 左栏搜索
- **WHEN** 运维在 User 左栏输入搜索词
- **THEN** 单一搜索框可覆盖用户标识类字段（含 tg_user_id），且无独立 tg_user_id 输入框

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
控制台 MUST 提供 Dashboard 页：数字卡片与简单状态/占比分布；任务相关统计 MUST 来自专用统计接口（`/api/v1/stats/tasks/*`），为全量按天聚合，不再受列表接口 limit 样本限制；实例与 Case 汇总仍可来自既有列表类接口。

#### Scenario: Dashboard 展示聚合信息
- **WHEN** 用户打开 Dashboard 且 admin-api 可用
- **THEN** 页面展示实例与 Case 汇总卡片，以及任务统计区块（每日处理任务数柱状图、成功率、错误码 Top-N、每节点负载），网络请求指向 admin-api

#### Scenario: Dashboard 卡片失败隔离
- **WHEN** 某一汇总依赖的 API 请求失败
- **THEN** 仅对应卡片或区块进入错误/空态，其它区块仍可展示

#### Scenario: 任务统计不受样本限制
- **WHEN** 所选日期范围内实际任务数超过 200
- **THEN** 每日柱状图与统计卡展示真实全量计数，而非列表样本计数

#### Scenario: 日期范围选择
- **WHEN** 运维选择起止日期或快捷范围（如近 7 / 30 / 90 天）
- **THEN** 每日处理任务数柱状图按所选范围重新查询并展示

#### Scenario: 分区展示实时与区间数据
- **WHEN** 运维打开 Dashboard
- **THEN** 页面按「实时状态 / 任务效能 / 业务分析」分区展示；实时状态区（节点启用、在线状态、算力池、集群实时负载）不随时间范围变化，任务效能与业务分析区统一跟随顶部时间范围控件刷新

#### Scenario: 全局时间范围
- **WHEN** 运维在 Dashboard 顶部切换近 7 / 30 / 90 天或选择起止日期
- **THEN** 任务效能与业务分析区的全部图表按同一范围重新查询

### Requirement: 仅通过 admin-api 通信
前端 MUST 仅通过配置的 admin-api 基址访问管理数据，MUST NOT 直连数据库或 bot 管理残留路径作为正式方案，MUST NOT 依赖独立前端 mock 层作为交付验收路径。

#### Scenario: API 基址可配置
- **WHEN** 开发者配置 admin-api 基址并打开资源页
- **THEN** 网络请求指向该基址下的管理 API

### Requirement: 主键盘管理页（树）
控制台 MUST 提供「主键盘」管理页：编辑树形菜单项（含文件夹、子项、挂载 Case、placeholder、reply_media）。保存 MUST 调用 admin-api 树形 Menu 接口；失败时 MUST 展示错误且不假装成功。本期 MUST NOT 要求浏览器本地上传图片。

#### Scenario: 编辑文件夹并挂载 Case 后保存
- **WHEN** 运维配置文件夹及其 Case 关联并保存成功
- **THEN** 页面提示成功，刷新后树与挂载仍在

#### Scenario: 保存失败展示错误
- **WHEN** admin-api 返回校验或网络错误
- **THEN** 页面展示错误信息，不进入「已保存」误导态

### Requirement: Case 详情展示菜单挂载
Case 详情 MUST 展示该 Case 出现在主键盘中的路径列表（只读）；数据 MUST 来自 admin-api。

#### Scenario: 已挂载 Case 可见路径
- **WHEN** 运维打开已挂到某文件夹的 Case 详情
- **THEN** 页面展示至少一条可读的主键盘路径

### Requirement: 请求仅指向 admin-api
主键盘页与 Case 挂载展示的请求 MUST 仅使用 `VITE_ADMIN_API_BASE`；MUST NOT 引入前端 mock 作为验收路径。

#### Scenario: Network 指向 admin-api
- **WHEN** 运维加载主键盘页或带挂载信息的 Case 详情
- **THEN** 浏览器请求前缀为配置的 admin-api 基址

