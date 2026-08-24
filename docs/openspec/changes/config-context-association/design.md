## Context

参见 proposal.md - Why：配置链路分散，Case routing 无独立入口，主键盘树页缺失，四模块无跨页上下文。前端已有 `TaskFlowEditor`（顶栏校验 + 画布 + Topic 池）、`MenuCardEditor`、topic 详情绑定节点、case 菜单挂载 API（`getCaseMenuPlacements`）等可复用资产。

## Goals / Non-Goals

**Goals:**
- Case 编辑页嵌入 `TaskFlowEditor` 配置 routing 并保存。
- 四模块详情页统一「关联上下文」面板（上游引用 + 下游影响 + 就绪状态 + 跳转）。
- 菜单挂载展示 Case 就绪摘要；恢复 `/tg-menu` 主键盘树管理页。

**Non-Goals:**
- 不做全局配置链路总览图（方案 B，后续可单独 change）。
- 不改快速配置向导本身；不重构 Topic/节点 CRUD 语义。

## Decisions

### 1. Case routing 编辑：复用 TaskFlowEditor

Case 编辑页新增「处理流程」节：直接复用 `TaskFlowEditor`（真实 topics/attributes/edges/presence），本地维护 routing 草稿，保存时 `PATCH /cases/{id}` 提交 routing；Case 已发布/已存在时仍可编辑并保存（规则与 workflow 独立）。

### 2. 关联上下文面板：统一组件 + 前端组合查询

新建 `ContextLinks` 组件（或 `association-panel`），四模块详情页共用：

- Case 详情：`topics = case.routing 引用的 Topic`；`nodes = edges.filter(subscribe_topics ∩ topics)`；`menus = getCaseMenuPlacements(id)`。
- Topic 详情：`cases = 后端 CountCaseRefs（既有 doc_json LIKE 查询）+ 前端展示`；`nodes = edges.filter(subscribe_topics.includes(topic))`；就绪 = 有在线节点。
- 节点详情：`topics = node.subscribe_topics`；`cases = topics 聚合后查 case.routing 引用`（前端组合或后端聚合）。
- 菜单（MenuCardEditor）：挂载 open_case 时读该 Case 的 routing + topics + edges，显示就绪摘要。

若组合查询导致 N+1 或体验差，补 `GET /api/v1/stats/case-context/{id}`（返回 routing 相关 topics/nodes/menu 摘要）与 topic-context；首版倾向前端组合（数据量小），聚合端点作为风险缓解。

### 3. 主键盘树管理页

新增 `/tg-menu` 路由与页面：树形编辑（文件夹/子项/挂载 Case/placeholder/reply_media），读写既有菜单 API（`GET/PUT /api/v1/channels/{id}/menu` 或主键盘 API），沿用 `MenuCardEditor` 的动作编辑能力；保存失败展示错误。

## Risks / Trade-offs

- 前端组合查询在数据量大时变慢 → 必要时补后端聚合端点（决策 2 已预留）。
- 主键盘树页与"按渠道卡片"模型并存 → 明确主键盘树是全局视角、渠道卡片是按渠道视角，同一真相源（菜单 API）。
- Case routing 与已有发布状态冲突 → 保存时保持幂等 PATCH，不改发布门禁。

## Migration Plan

1. 先落地 Case「处理流程」节与关联面板（复用 TaskFlowEditor）。
2. 再落地菜单就绪摘要与主键盘树页。
3. 全量验证与冒烟（Case 保存 routing → Topic 详情关联 → 菜单挂载摘要）。

## Open Questions

无阻塞项。是否新增后端聚合端点视组合查询实测而定。
