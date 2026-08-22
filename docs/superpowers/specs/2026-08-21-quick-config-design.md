---
comet_change: quick-config-wizard
role: technical-design
canonical_spec: openspec
archived-with: 2026-08-22-quick-config-wizard
status: final
---

# 快速配置向导（quick-config-wizard）：Design Doc

> OpenSpec 上游：`docs/openspec/changes/quick-config-wizard/`（proposal / design / delta specs / tasks）。

## 背景与问题

后台要把「一个工作流从创建到被用户使用」配通，需要横跨 Case 编辑、任务分流画布、Topic/节点管理、渠道菜单四个页面，链路长且管理员不知道「还差什么」。目标：菜单级「快速配置」入口，用 Formity 编排三步向导，配合完成页就绪清单与发布门禁，杜绝空壳 Case。

## 已确认决策

| 项 | 决策 |
|---|---|
| 步骤形态 | **Formity flow 步骤机制**（`form` 元素 + `next`/`back`/`jump`），一次只渲染一个步骤屏幕，进度「Step X of 3」，不做独立路由页、不堆叠 |
| 入口 | 侧栏 Dashboard 后新增「快速配置」（/quick-config）；落地页：继续上次配置 / 新建 / 选择已有；**落地页选择即向导模式**（Step 1 不再询问新建/已有） |
| Step 1 | 复用 `CaseForm`（新建强制导入合法 JSON，`redirectAfterSave=false` + `hideActions` + `formId` 触发提交；调整已有加载到同一表单） |
| Step 2 | 内嵌 `TaskFlowCanvas`，实时拉取 topics/attributes/edges/presence，保存 `PATCH /cases/{id}` 写 `routing` |
| Step 3 | 复用菜单 API：读 `GET /channels/{id}/menu` → `addWorkflowMenuEntry` 追加 `open_workflow` 按钮 → `PUT` 全量写回；展示已有投放 |
| 完成页 | 就绪清单 G1（workflow/inputs/input_schema）/G2（规则完整 + Topic 绑定启用节点，离线=warn）/G3（≥1 投放）；发布 = `POST /cases/{id}/enable` |
| 恢复 | 本地会话 `pixoma:quick-config`（caseId/mode/step/updatedAt），落地页「继续上次配置」 |
| 后端缺口 | Case 创建要求客户端自带 id → 修复为 id=0 自动分配（validation 移除 id 必填、repo 回写自动 id 并同步 DocJSON） |

## 就绪模型

- G1：`bindings.workflow` 非空、`inputs` 非空、`input_schema` 存在。
- G2：`rules.length >= 1`、每条规则已连 Topic、每个使用中的 Topic 绑定 ≥1 启用节点；全部离线 = warn（不阻塞发布）。
- G3：`menu-placements` 非空。

## 实现清单（第一版已交付）

- `features/quick-config/`：types、wizard-chrome（进度 + 摘要 chips + 底部导航）、step1/2/3、done-screen、quick-config-flow（Formity flow）、quick-config-page（落地页）、lib（readiness/session/menu-payload + 单测）。
- `lib/api/`：types 增加 `RoutingConfig`/`CaseRecord.routing`；cases 增加 `enableCase`/`disableCase`；新增 routing（attributes 目录）。
- `config/menu.ts` + menu.test：快速配置入口；i18n zh/en 成对。
- `vitest.config.ts`：纳入 quick-config 测试；CaseForm 增加向导内嵌 props。

## 风险与后续

- Formity 1.x/3.x 运行时行为仅以类型与示例验证，待联调冒烟确认；受阻回退 `setup-wizard` 步骤状态机。
- 画布内「行内新建 Topic/节点」工具栏为后续增量（当前通过画布编辑器与节点订阅 API 配置）。
- 就绪计算在完成页读取 topics/edges/presence，数据加载完成前按钮禁用。
- `task-flow-canvas` 分包体积大（1.6MB），后续按需拆包。
