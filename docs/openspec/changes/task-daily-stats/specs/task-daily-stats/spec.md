## Purpose

为管理端 Dashboard 提供全量、按天聚合的任务统计能力：每日已处理任务数、成功率、错误码分布与每节点负载，数据来自专门的统计表与统计接口，不受列表接口样本数量限制。

## ADDED Requirements

### Requirement: 任务按天统计持久化
系统 MUST 在任务进入终态（succeeded / failed / cancelled）时，按完成时间（completed_at）归天更新当日统计；同一任务的重复终态处理 MUST 幂等，不得重复累加计数。

#### Scenario: 任务完成写入当日统计
- **WHEN** 任务进入 succeeded / failed / cancelled 终态
- **THEN** 对应日期（completed_at 所在天）的 processed 计数与对应状态计数均增加，且 processed = succeeded + failed + cancelled

#### Scenario: 重复终态处理不重复计数
- **WHEN** 同一任务的终态被重复应用（如重试、对账）
- **THEN** 该日统计计数不重复累加

#### Scenario: 取消计入已处理
- **WHEN** 任务被取消并进入 cancelled 终态
- **THEN** 对应日期 processed 与 cancelled 计数均增加

### Requirement: 每日统计查询接口
系统 MUST 提供 `GET /api/v1/stats/tasks/daily` 管理接口，支持 `from` / `to` 日期范围参数，返回范围内逐日 processed / succeeded / failed / cancelled 与平均耗时，并附范围汇总（成功率等）。

#### Scenario: 按范围返回每日数据
- **WHEN** 客户端请求 daily 接口并携带合法的 from / to 日期范围
- **THEN** 返回范围内逐日统计，按日期升序排列，并附范围成功率汇总

#### Scenario: 无数据返回空
- **WHEN** 请求范围内没有任何终态任务
- **THEN** 返回空序列与零值汇总，HTTP 200 而非错误

#### Scenario: 非法日期范围被拒绝
- **WHEN** from 晚于 to，或日期格式非法
- **THEN** 接口返回 400 与明确错误信息

#### Scenario: 空白天返回零计数
- **WHEN** 请求范围内某一天没有任何终态任务
- **THEN** 该天仍出现在逐日序列中，计数为 0，时间轴连续

#### Scenario: 成功率口径
- **WHEN** 计算范围汇总中的成功率
- **THEN** success_rate 等于 succeeded / (succeeded + failed)，且成功与失败均为 0 时返回空值

### Requirement: 错误码与每节点负载统计
系统 MUST 提供错误码 Top-N 与每节点已处理任务数统计，供 Dashboard 展示。

#### Scenario: 错误码 Top-N
- **WHEN** 客户端请求错误码统计接口并携带 from / to 与 limit
- **THEN** 返回按错误码聚合计数降序的前 N 条（无错误码的任务不占位）

#### Scenario: 每节点负载
- **WHEN** 客户端请求每节点统计接口并携带 from / to
- **THEN** 返回范围内每个节点的已处理任务数与占比，按计数降序

#### Scenario: 仅统计终态任务
- **WHEN** 范围内同时存在排队、运行中的任务与终态任务
- **THEN** 每节点负载仅按终态任务（succeeded / failed / cancelled）的 edge_id 计数，非终态任务不计入

#### Scenario: 空数据降级
- **WHEN** 范围内无失败任务或无派发节点数据
- **THEN** 对应统计返回空序列，前端展示空态而非报错

### Requirement: 统计口径
每日统计 MUST 以 completed_at 归天，processed 定义为 succeeded + failed + cancelled；历史任务 MUST 支持通过一次性 backfill 补齐统计表。

#### Scenario: 历史数据补齐
- **WHEN** 统计表刚上线或存在历史终态任务
- **THEN** 执行 backfill 后，历史每天均能按上述口径查出统计

#### Scenario: 口径一致
- **WHEN** 同一时间范围内分别查询 daily 汇总与逐日数据
- **THEN** 汇总计数等于各日计数之和
