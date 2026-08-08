# 架构文档索引

从这里读完整项目架构（不只是表结构）。

| 文档 | 内容 |
|---|---|
| [overview.md](./overview.md) | **入口**：系统上下文、进程逻辑视图、仓库布局、配置、依赖摘要 |
| [bounded-contexts.md](./bounded-contexts.md) | DDD 限界上下文、包职责、依赖规则、可拆分组合 |
| [runtime.md](./runtime.md) | ConfirmRun→调度→执行→通知；事件 topics；Task 状态机；观测 API |
| [data-model.md](./data-model.md) | 领域字段、SQLite 表结构、表关系 ER、实例关系 ER |
| [diagrams/system.html](./diagrams/system.html) | 系统拓扑可视化（浏览器打开） |

运维命令与 Mock 开关见仓库根 [README.md](../../README.md)。
