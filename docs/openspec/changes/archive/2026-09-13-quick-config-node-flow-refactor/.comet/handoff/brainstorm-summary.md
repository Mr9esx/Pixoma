# Brainstorm Summary

- Change: quick-config-node-flow-refactor
- Date: 2026-08-28

## 确认的技术方案

- 向导四步 + 完成页：工作流编辑 → 运行节点 → 特殊规则 → 投放 → 完成页发布。
- 入口不变「从一个工作流开始」；投放与完成页发布门禁保留。
- 节点可直接绑定 Topic（`EdgeRecord.subscribe_topics` / `effective_topics`，`patchEdge` 支持 `subscribe_topics`）。
- **Default 分支 = 对选定运行节点执行 `PATCH /edges/{id} {subscribe_topics:[...+default]}` 绑定 default，任务回退 default 后由订阅 default 节点竞争消费（非独占）。**
- **取消 Case 级归属字段（assignedEdgeId）设计**：不改 Case 模型/DTO/接口，不动数据面调度。
- **TaskFlowEditor 本期不接入**；「需要特殊规则」保留入口但跳转既有独立规则编辑页，向导内不重建编辑器。
- 旧会话清空（BREAKING），会话载荷增加 schemaVersion。
- 完成页四项就绪：工作流已导入、运行节点已选且订阅就绪、处理流程就绪、至少一个投放。

## 关键取舍与风险

- 节点自绑定 default，竞争消费（非独占），复用现有能力、后端零新增。
- 规则定制从向导移出到独立编辑页，本期向导只覆盖 default 主路径。

## 测试策略

- 前端合同测试（quick-config）：新屏序、运行节点步骤、default 分支、完成页就绪、会话清空。
- 后端 go test：无回归。
- 冒烟：默认路由完整链路；需要规则跳转独立编辑页。

## Spec Patch

- `quick-config-node-assignment`：改为「运行节点选择」+「Default 分支绑定与派发」（去 Case 归属字段）。
- `quick-config-wizard`：处理流程步骤改为默认路由 + 跳转独立规则编辑页；新增「运行节点步骤」；完成页四就绪改为运行节点口径。
