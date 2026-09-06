## 1. 组装器与不变式

- [x] 1.1 先写活路不变式测试：唯一入口不可用 → 工作流不得 ok；双入口有活路 → 工作流 ok、坏入口在引用里 warn；唯一节点不可用 → 工作流与队列不得 ok；输入缺失 → pending 不得 ok
- [x] 1.2 实现 `packaging/linkhealth` 拼图与走路径；平台起点由组装器合成，页面不得另持定义
- [x] 1.3 运行信号组装器读不到时标 pending；查询热路径不发起外部探测
- [x] 1.4 `GET /api/v1/link-health` 挂到控制面，鉴权与既有管理 API 相同

## 2. 页面只渲染

- [x] 2.1 工作流 / 平台 / 队列 / 节点列表圆点与详情健康改吃查询；删除运行时 `*References` / `listHealthTone` 发明绿黄
- [x] 2.2 工作台拓扑与详情弹窗用查询的 nodes/edges/health
- [x] 2.3 合同测试：三处同色；查询失败不标绿；不再 per-case 拼菜单算健康

## 3. 架构文档

- [x] 3.1 更新 `docs/architecture/bounded-contexts.md`、`data-model.md`、`overview.md`
- [x] 3.2 将 `diagrams/link-health.html` 放到 `docs/architecture/diagrams/` 并在索引挂上

## 4. 验收

- [x] 4.1 不变式测试与前端合同全绿；`go test` 与 admin vitest / tsc

审查（standard）接受项：
- 平台是否可用跟适配器启停走；Telegram 探测只在打开通道详情时自动打，不挡列表绿黄。GET /link-health 热路径仍不探测。
- 无会话拒绝依赖既有 admin gate，与其它 `/api/v1` 相同；handler 单测不重复套 gate。
