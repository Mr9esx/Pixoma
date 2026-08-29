## MODIFIED Requirements

### Requirement: Dashboard 中等总览
控制台 MUST 提供 Dashboard 页（工作台视图）：以欢迎卡 + 左栏数据大盘 / 右栏关注信息的方式组织；任务相关统计 MUST 来自专用统计接口（`/api/v1/stats/tasks/*`），为全量按天聚合，不再受列表接口 limit 样本限制；实例与 Case 汇总仍可来自既有列表类接口。该视图的详细展示需求见 `admin-dashboard-workbench` capability，此处仅保留全量口径、admin-api 通信与失败隔离约束。

#### Scenario: Dashboard 展示聚合信息
- **WHEN** 用户打开 Dashboard 且 admin-api 可用
- **THEN** 页面展示工作台视图（欢迎卡、节点/负载/工作流热度/任务耗时/状态分布/错误 Top5 等区块），网络请求指向 admin-api

#### Scenario: Dashboard 卡片失败隔离
- **WHEN** 某一汇总依赖的 API 请求失败
- **THEN** 仅对应卡片或区块进入错误/空态，其它区块仍可展示

#### Scenario: 任务统计不受样本限制
- **WHEN** 所选日期范围内实际任务数超过 200
- **THEN** 任务统计卡展示真实全量计数，而非列表样本计数

#### Scenario: 日期范围选择
- **WHEN** 运维选择起止日期或快捷范围（如近 7 / 30 / 90 天）
- **THEN** 区间型统计区块按所选范围重新查询并展示

#### Scenario: 分区展示实时与区间数据
- **WHEN** 运维打开 Dashboard
- **THEN** 页面按「数据大盘 / 关注信息」展示；实时状态区（节点启用、在线状态）不随时间范围变化，区间型统计区统一跟随顶部时间范围控件刷新

#### Scenario: 全局时间范围
- **WHEN** 运维在 Dashboard 顶部切换近 7 / 30 / 90 天或选择起止日期
- **THEN** 所有区间型统计区块按同一范围重新查询
