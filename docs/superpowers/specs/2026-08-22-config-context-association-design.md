---
comet_change: config-context-association
role: technical-design
canonical_spec: openspec
---

# config-context-association 深度技术设计（一版）

## 1. 目标

管理控制台配置链路（菜单 → 工作流 → Topic → 节点）上下文关联，一版落地：Case 编辑页「处理流程」路由配置 + 四模块关联面板（上游/下游引用 + 就绪状态 + 跳转）+ 菜单挂载就绪摘要。

## 2. 关联数据来源（前端组合，不做新端点）

- Case → Topic：`case.routing.rules[].topic`（已落库）。
- Topic → 节点：`edges.subscribe_topics` 过滤 + `listPresence` 在线判定。
- Case → 菜单：`getCaseMenuPlacements(caseId)`（既有接口）。
- Topic → Case：后端既有 `CountCaseRefs`（doc_json LIKE）返回数量；一版用数量 + 前端能拿到的 routing 反查（列表接口按 `routing` 字段组合）。
- 节点 → Case：`node.subscribe_topics` → 该 topic 被哪些 case.routing 引用（前端组合 cases 列表）。

## 3. 组件与页面改动

- 新增 `ContextLinks`（features/config-context/context-links.tsx）：shadcn Card + 分组行（上游/下游/就绪），项为 Badge（就绪/警告）+ Link 跳转 + 一句人话说明。
- Case 详情页：新增「处理流程」节（`TaskFlowEditor` 编辑 routing，保存 `patchCase(id, { routing })`）与关联面板（路由 Topic / 执行节点 / 菜单挂载）。
- Topic 详情页：关联面板（路由到的 Case + 订阅节点 + 就绪状态）。
- 节点详情页：关联面板（订阅 Topic + 可达 Case 摘要）。
- MenuCardEditor：挂载 open_case 时显示该 Case 就绪摘要（规则完整 + 目标 Topic 有在线订阅者）。

## 4. 文案与样式

- 全部走 i18n zh/en；文案只说结论（如「2 台在线节点可执行」），不写解释性废话。
- 组件复用 shadcn `Card` / `Badge` / `Link` / `Button`，沿用 neutral 主题变量。

## 5. 主键盘树页

一版暂缓：`/tg-menu` 树形页独立落地（复用菜单 API 与 MenuCardEditor 动作编辑），作为下一步。

## 6. 测试

合同测试锁定 ContextLinks 结构、Case「处理流程」节与保存载荷、就绪徽标文案；`pnpm tsc -b` + `pnpm vitest run` 全绿。
