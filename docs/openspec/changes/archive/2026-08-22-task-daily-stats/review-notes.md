# task-daily-stats 代码评审记录

## 评审方式

`review_mode: standard`。已按 comet-build review gate 加载 `requesting-code-review` 技能并尝试派发 reviewer subagent；本会话 subagent 通道无法接收任务（两次派发均返回待命），降级为主会话内联只读评审（`git diff 05b221f..HEAD`），结论记录于此。

## 评审结论

**Strengths**

- 三张窄表按维度拆分，查询走主键/索引，读路径 O(天数)，不再全表扫描任务。
- orchestrator 写路径用「旧状态 ≠ 新状态且为终态」守卫，重复/对账事件不会双计；零值 `completed_at` 回退事件时间。
- daily 接口零填充 + 成功率 `succeeded/(succeeded+failed)`（分母 0 为 null）、from/to 校验（400）均有单测覆盖。
- backfill 以任务表为权威源按天重算绝对值覆盖，幂等可重跑。
- 前端沿用既有 shadcn chart 体系与 Calendar range，查询 key 含日期范围，失败按卡片隔离。

**Issues**

- Important（已修复）：`ListErrors` / `ListEdges` 使用 `AS count` + `ORDER BY count`，别名与 SQL 函数名冲突在 MySQL 下有歧义；改为 `AS total` + `ORDER BY total` 并手工映射。
- Minor（接受）：管理端统计接口与现有 admin-api 一致无鉴权，属既有部署约束（README 已警示仅本机/内网）。
- Minor（接受）：任务完成时间等于创建时间时 `avg_duration_ms` 为 0 而非 null（口径为毫秒取整，可接受）。
- Minor（接受）：实施计划中 handler 测试先以注释占位再补真实断言，最终实现已含完整断言（handler_test.go）。

**Assessment: Ready to proceed**（Important 发现已修复并回归；Minor 记录接受理由与影响范围）
