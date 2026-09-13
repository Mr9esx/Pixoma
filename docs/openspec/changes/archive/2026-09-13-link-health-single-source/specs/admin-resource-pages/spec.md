## ADDED Requirements

### Requirement: 四类资源健康来自唯一组装器
工作流、消息平台、任务队列、计算节点的列表圆点与详情「状态与关联」MUST 使用链路健康组装器返回的结论与引用。MUST NOT 再由各页面自行拼接多份列表来发明绿黄。健康未返回或请求失败时 MUST NOT 把圆点显示为正常。

#### Scenario: 唯一入口不可用则工作流列表为黄
- **WHEN** 某工作流只挂在一个已不可用的消息平台上，运维打开工作流列表
- **THEN** 该工作流的列表圆点为存在问题

#### Scenario: 详情与列表同源
- **WHEN** 运维从该工作流列表进入详情
- **THEN** 详情健康区块报告存在问题，并指出不可用的那一跳

#### Scenario: 查询失败不标绿
- **WHEN** 健康查询失败
- **THEN** 列表圆点不显示为正常

### Requirement: 进入消息平台列表即异步探测
打开消息平台列表路由时，前端 MUST 发起一次异步探测踢脚。列表与详情 MUST 立刻用已存 last_check / 适配器状态与健康查询渲染。MUST NOT 为等待探测而挡住骨架。MUST NOT 对每个通道依次同步 POST `/check`。

#### Scenario: 进页立刻出列表
- **WHEN** 运维打开消息平台列表
- **THEN** 列表请求不依赖探测踢脚完成；踢脚在后台进行

#### Scenario: 探测结束后详情与健康一起刷新
- **WHEN** 进页踢脚已被接受，且某已启用通道的 last_check 随后变成 network
- **THEN** 详情标题 pill 与「状态与关联」都反映无法连接；MUST NOT 一边显示无法连接、一边显示链路正常

#### Scenario: last_check 为 network 时详情不得写链路正常
- **WHEN** 通道 last_check 为 network，标题 pill 显示无法连接 Telegram
- **THEN** 「状态与关联」MUST NOT 渲染链路正常

### Requirement: 拓扑使用同一份组装结果
工作台配置拓扑与详情拓扑弹窗 MUST 使用组装器返回的节点、连线与健康。图上某节点的绿黄 MUST 与该资源详情健康一致。MUST NOT 在前端用另一套规则给图节点上色。

#### Scenario: 图上工作流与详情同色
- **WHEN** 某工作流详情健康为存在问题
- **THEN** 工作台拓扑与该工作流焦点弹窗上对应节点均为黄色
