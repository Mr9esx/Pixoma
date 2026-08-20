## 1. 数据模型与迁移

- [x] 1.1 新增 `topics` 表（key 唯一、name、enabled、时间戳）与 GORM 模型，纳入 AutoMigrate
- [x] 1.2 `edges` 增加 `subscribe_topics_json` 列；读取时空值按 `["default"]` 处理
- [x] 1.3 `tasks` 增加 `dispatch_topic` 与 `attempts` 列；复用 `edge_id`/`lease_until` 表示持有节点与租约
- [x] 1.4 Case 协议模型与 doc_json 增加可选 `routing`（rules: [{when, topic}]），旧文档无该字段兼容读写
- [x] 1.5 控制面启动幂等种入默认 Topic `default`（已存在跳过，删除被拒绝）

## 2. 条件协议

- [x] 2.1 定义 `AttributeDescriptor`（key、JSON Schema、UI 标签、上下文来源）与 provider 注册表（按 user/case/input 命名空间）
- [x] 2.2 内置 provider：`user.is_premium`、`case.category`、`case.tags`（含取值函数与 schema）
- [x] 2.3 规则 JSON 解析与校验：叶子 {field, op, value}、and/or 组合；未知字段/非法 op/类型不符给出可定位错误
- [x] 2.4 求值引擎：eq/ne/in/gt/gte/lt/lte/exists，缺失上下文按 false（exists 除外）
- [x] 2.5 单元测试：合法/非法规则、组合求值、新 provider 注册后不改引擎即可用（协议扩展性测试）

## 3. Topic 领域与 Admin API

- [x] 3.1 Topic 仓储与 service：CRUD、启用/禁用、default 不可删
- [x] 3.2 `GET/POST /api/v1/topics`、`GET/PUT/DELETE /api/v1/topics/{key}` 及校验（key 命名、禁用态引用）
- [x] 3.3 节点订阅绑定：`PATCH /api/v1/edges/{id}` 支持 `subscribe_topics`，列表/详情返回有效订阅集合
- [x] 3.4 Case 路由配置读写：case-admin-api 创建/更新/详情载荷支持 `routing`，保存时校验 Topic 存在且启用、条件协议合法
- [x] 3.5 条件目录 API `GET /api/v1/routing/attributes` 返回属性 key/schema/UI 标签

## 4. 调度改造

- [x] 4.1 Orchestrator 调度流程改为：读 Case 路由 → 求值首个命中规则 → 确定 `dispatch_topic`（无命中 = default）
- [x] 4.2 调度不再预绑定 edge_id：Task 置为 `queued` 且携带 `dispatch_topic`，prep job 包（job_ref）语义保持
- [x] 4.3 SchedulePending/事件触发双通道沿用，按 Topic 可领取态推进

## 5. 原子领取

- [x] 5.1 `GET /agent/v1/jobs/claim` 改为按节点订阅 Topic 集合扫描可领取任务（BREAKING，edge_id 用于读绑定）
- [x] 5.2 原子抢占：条件 UPDATE（SQLite 事务 / MySQL/Postgres RETURNING 等价语义）保证同一任务只被一台领取，写入 edge_id + lease_until
- [x] 5.3 长轮询重扫与空返回；并发竞争落败返回空结果
- [x] 5.4 并发互斥契约测试：两节点同时 claim 同一 Topic 单任务，恰好一成功一空

## 6. 失败重回 Topic 与超时回收

- [x] 6.1 failed 有界重试：attempts < max（默认 3）时任务回 `queued`（清持有者、attempts+1、可选退避），超限收敛最终 failed 并记录原因
- [x] 6.2 ReconcileStale 增强：租约过期且无终态的任务重新可领取或按重试上限收口；心跳续约保持
- [x] 6.3 复用 StormGuard 限流/退避/熔断与分池，防止回收与重试风暴
- [x] 6.4 单元测试：失败重回、宕机回收、超限收口、重复 status 不重复副作用

## 7. 端到端与兼容

- [x] 7.1 allinone/Mock 主路径：默认 Topic 下任务端到端成功
- [x] 7.2 split 主路径：Edge 配置订阅多个 Topic，按订阅领取并执行成功
- [x] 7.3 旧数据兼容：空 `dispatch_topic` 按 default 处理；未配置订阅的旧 Edge 按 default 领取
- [x] 7.4 文档同步：data-model、runtime、task-data-walkthrough、README（Topic、订阅、claim 语义）
- [x] 7.5 全量 `go test ./...` 与关键冒烟通过
