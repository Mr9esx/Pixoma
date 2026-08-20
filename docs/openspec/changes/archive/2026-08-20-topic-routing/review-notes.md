# Review Notes — topic-routing（build 阶段审查）

- **日期**：2026-08-20
- **范围**：`3f2cc676..HEAD`（19 个实现提交，Go 后端 Topic 调度）
- **审查方式**：`review_mode: standard` 要求请求代码审查；子代理派发通道在本环境连续 3 次无法送达任务载荷（agents 均回复"未收到任务"），经用户确认"继续"后，由主会话对完整 diff 执行同等的正确性/安全/边界审查。**偏离说明**：未使用 `requesting-code-review` 的子代理模板，审查覆盖同一检查面并修复发现项。

## Strengths

- 原子领取沿用既有"事务内 SELECT + 条件 UPDATE + RowsAffected 守卫"模式扩展为 Topic 集合；SQLite 单写者串行、Postgres/MySQL 行锁 + status 守卫，同一任务最多一台消费（并发契约测试覆盖）。
- 状态机边界清晰：`PrepareForTopic` 仅 pending→queued 且不绑定节点；`RequeueAfterFailure` 有界（3 次重试 5s/15s/45s）；租约回收不烧 attempts；终态通知幂等。
- 条件协议为纯领域包，provider 注册 + JSON Schema 驱动，求值错误向上传播（不吞、不误投默认池）；新增属性有扩展性契约测试。
- 默认 Topic 幂等种子、`default` 不可删除/禁用；空 `dispatch_topic` 查询层按默认处理，兼容旧数据。

## Issues

### Critical

无。

### Important

- `CountCaseRefs` / `CountEdgeRefs` 的 LIKE 模式直接拼接 URL 传入的 key，`%`/`_` 可放大匹配。**已修复**：`escapeLike` 转义 `\`、`%`、`_`（apps/pixoma/cmd/pixoma/main.go）。
- `PUT /api/v1/topics/default` 可把默认 Topic 禁用，导致无规则命中任务失去兜底。**已修复**：禁用 `default` 返回 409（internal/httpapi/topics/handler.go），并补测试。

### Minor（接受，记录理由）

- `dispatch_topic IN (...) OR dispatch_topic=''` 的空值兜底在滚动升级窗口内允许任意订阅节点领取旧行：迁移会在启动时把存量 queued 归 `default`，窗口期很短，接受。
- `requeue_at`/`lease_until` 零值在 MySQL 严格模式可能写入 `0001-01-01`：与既有 `lease_until` 写法一致，本项目既有行为，接受。
- claim 长轮询每拍调用 `RequeueExpiredLeases`：既有行为未变，且有 StormGuard 保护调度/回收池，接受。
- 复合索引未建（`status`、`dispatch_topic` 各自索引）：当前数据量下可接受，规模增长时再补。

## Verdict

Ready to proceed（发现项已修复，全量 `go test ./...`、`go build ./...`、关键包 `-race` 通过）。
