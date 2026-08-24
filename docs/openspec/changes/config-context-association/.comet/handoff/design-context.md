# Comet Design Handoff

- Change: config-context-association
- Phase: design
- Mode: compact
- Context hash: c87881928447b68d560f95e32345b87a62771195cf04c10e2f5c0e31e8f3614f

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## docs/openspec/changes/config-context-association/proposal.md

- Source: docs/openspec/changes/config-context-association/proposal.md
- Lines: 1-24
- SHA256: 603c22e7bb67696b90cb5d8307e57c289116722526362a2e322b60ba5467ddde

```md
## Why

管理控制台的配置链路（菜单 → 工作流 → Topic → 节点）分散在独立页面：Case 的 routing 规则只能通过「快速配置」Step2 或原型页编辑，普通 Case 编辑页无法配置；四模块详情页之间缺少上下文关联与就绪反馈，用户需要跨页拼接"我改这里，上下游谁受影响"，心智负担高；主键盘树管理页（文件夹/挂载 Case/placeholder/reply_media）需求已存在于 spec 但未实现。

## What Changes

- Case 编辑页新增「处理流程」节：复用 `TaskFlowEditor` 配置 routing（规则 → Topic），保存时 `PATCH /cases/{id}` 写 routing；未导入合法 workflow 时仍可编辑路由（规则独立于 workflow 图）。
- 四模块详情页统一「关联上下文」面板：Case ↔ Topic ↔ 节点 ↔ 菜单 双向引用 + 就绪状态（Topic 是否有在线订阅节点、Case 规则是否完整、菜单挂载的 Case 是否可执行）+ 点击跳转。
- 菜单：`MenuCardEditor` 挂载 Case 时展示该 Case 的路由就绪与可执行节点摘要；恢复主键盘树管理页（`/tg-menu`：文件夹/子项/挂载 Case/placeholder/reply_media 树编辑，保存写菜单 API）。
- 跨模块数据：前端组合查询（`cases.routing` → `topics` → `edges.subscribe_topics` → 菜单挂载）；如组合成本过高，补充后端聚合端点（如 `GET /api/v1/stats/case-context/{id}`）。

## Capabilities

### New Capabilities
- `config-context-association`: 管理控制台配置模块间的上下文关联（关联面板、就绪状态、跨模块跳转）、Case 路由编辑入口与主键盘树管理页。

### Modified Capabilities
- `admin-resource-pages`: Case 编辑页新增「处理流程」路由配置节；四模块详情页新增关联上下文面板；主键盘树管理页补齐实现。

## Impact

- 前端：`web/admin` 的 cases / topics / edges / channels / menu 详情页与编辑表单、新增 `/tg-menu` 树管理页、`TaskFlowEditor` 复用、关联面板组件、i18n、合同测试。
- 后端：视聚合成本决定是否新增 Case 上下文聚合端点。
- 文档：管理配置指引补充关联链路说明。

```

## docs/openspec/changes/config-context-association/design.md

- Source: docs/openspec/changes/config-context-association/design.md
- Lines: 1-51
- SHA256: 51cee194b6aa3e3850c5ffc6aa90028b1796e7c803f68d65131581df02bb6529

```md
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

```

## docs/openspec/changes/config-context-association/tasks.md

- Source: docs/openspec/changes/config-context-association/tasks.md
- Lines: 1-26
- SHA256: aa5848f31d405cbecae062d0932ee30f4543d95578498ae7dbdae39d840f1535

```md
## 1. Case 路由编辑

- [ ] 1.1 Case 编辑/详情页新增「处理流程」节：嵌入 `TaskFlowEditor`，本地维护 routing 草稿
- [ ] 1.2 保存：`PATCH /cases/{id}` 写 routing；未导入 workflow 也可编辑；保存后回读一致
- [ ] 1.3 合同测试：处理流程节、保存载荷、未导入 workflow 可编辑

## 2. 关联上下文面板

- [ ] 2.1 统一 `ContextLinks` 组件（上游/下游引用 + 就绪状态 + 跳转），复用四模块详情页
- [ ] 2.2 Case 详情：路由 Topic + 订阅节点 + 菜单挂载
- [ ] 2.3 Topic 详情：路由 Case（复用 CountCaseRefs）+ 订阅节点 + 就绪状态
- [ ] 2.4 节点详情：订阅 Topic + 关联 Case 摘要
- [ ] 2.5 菜单挂载摘要：MenuCardEditor 挂载 Case 显示 routing 就绪与可执行节点
- [ ] 2.6 合同测试：关联面板结构、就绪徽标、跳转

## 3. 主键盘树管理页

- [ ] 3.1 新增 `/tg-menu` 路由与树形编辑页（文件夹/子项/挂载 Case/placeholder/reply_media）
- [ ] 3.2 保存走菜单 API；失败展示错误；刷新后树保持
- [ ] 3.3 合同测试：树编辑、保存成功/失败

## 4. 验证与文档

- [ ] 4.1 `pnpm tsc -b` + `pnpm vitest run`；`go build ./...` + `go test ./...`
- [ ] 4.2 冒烟：Case 保存 routing → Topic 详情关联 → 菜单挂载摘要一致
- [ ] 4.3 管理配置指引补充关联链路说明

```

## docs/openspec/changes/config-context-association/specs/admin-resource-pages/spec.md

- Source: docs/openspec/changes/config-context-association/specs/admin-resource-pages/spec.md
- Lines: 1-23
- SHA256: 50bdda51a284cd68b6fceb35742b3843cbec4286370a6da343d2f180bc2f0d3a

```md
## ADDED Requirements

### Requirement: Case 编辑页可配置处理流程
Case 编辑/详情页 MUST 提供「处理流程」配置节，复用任务分流编辑器编辑 routing 规则（条件 → Topic），并支持保存到 Case（`PATCH /cases/{id}`）。

#### Scenario: 编辑并保存路由规则
- **WHEN** 运维在 Case 编辑页打开「处理流程」并完成规则配置
- **THEN** 规则保存在该 Case 上，保存后回读一致

#### Scenario: 未导入 workflow 也可编辑路由
- **WHEN** Case 尚未导入合法 workflow JSON
- **THEN** 路由编辑仍可用（规则与 workflow 图相互独立），保存仅写 routing

### Requirement: 主键盘树管理页
控制台 MUST 提供主键盘树管理页：编辑树形菜单项（文件夹、子项、挂载 Case、placeholder、reply_media），保存调用菜单 API；失败展示错误且不假装成功。

#### Scenario: 编辑文件夹并挂载 Case 后保存
- **WHEN** 运维配置文件夹及其 Case 关联并保存成功
- **THEN** 页面提示成功，刷新后树与挂载仍在

#### Scenario: 保存失败展示错误
- **WHEN** 菜单 API 返回校验或网络错误
- **THEN** 页面展示错误信息，不进入「已保存」误导态

```

## docs/openspec/changes/config-context-association/specs/config-context-association/spec.md

- Source: docs/openspec/changes/config-context-association/specs/config-context-association/spec.md
- Lines: 1-24
- SHA256: a256f63c0eb404602c2ea9e50b6ec68cf9862c645e761a4b2c43e01d01ffbcb5

```md
## Purpose

为管理控制台配置链路（菜单 → 工作流 → Topic → 节点）提供跨模块上下文关联：统一的关联面板展示上下游引用、就绪状态与跳转，降低配置时的心智负担。

## ADDED Requirements

### Requirement: 四模块关联上下文面板
Case / Topic / 节点 / 菜单详情页 MUST 提供统一的关联上下文面板，展示上下游引用、就绪状态并提供跳转。

#### Scenario: Case 详情展示关联链路
- **WHEN** 运维打开 Case 详情
- **THEN** 可看到该 Case 路由到的 Topic、订阅这些 Topic 的节点、以及挂载该 Case 的菜单入口；任一关联项可点击跳转

#### Scenario: Topic 详情展示关联链路
- **WHEN** 运维打开 Topic 详情
- **THEN** 可看到路由到该 Topic 的 Case 与订阅该 Topic 的节点，并显示是否就绪（存在在线订阅节点）

#### Scenario: 节点详情展示关联摘要
- **WHEN** 运维打开节点详情
- **THEN** 可看到该节点订阅的 Topic 与经由这些 Topic 可达的 Case 摘要

#### Scenario: 菜单挂载展示就绪摘要
- **WHEN** 运维在菜单编辑中挂载一个 Case
- **THEN** 可看到该 Case 的路由就绪状态与可执行节点摘要（规则完整且目标 Topic 有在线订阅节点）

```
