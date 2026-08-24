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
