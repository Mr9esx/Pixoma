## Why

管理后台要把「一个工作流从创建到真正被用户使用」配通，需要横跨 Case 编辑、任务分流画布（React Flow）、Topic/计算节点管理、渠道菜单四个页面，操作链路长且管理员不知道「还差什么」。需要一个菜单级的「快速配置」入口：用 **Formity 的 flow 步骤机制**（参照其官方 examples 形态）编排三步向导——库一次渲染一个步骤屏幕，配合完成页就绪清单与发布门禁，让不熟悉内部结构的运营能明确知道下一步该做什么。

## What Changes

- 管理后台侧栏新增「快速配置」入口（`admin-web-shell`），落地页 `/quick-config`：继续上次配置 / 新建工作流 / 选择已有工作流；**落地页的选择即向导模式**（`mode` 变量带入流程），Step 1 不再重复询问新建/已有。
- **三步由 Formity flow 步骤机制驱动**（参照 Formity examples：一次只渲染当前步骤屏幕、进度指示「Step X of 3」、`next`/`back`/`jump` 切换、已完成步骤可回跳；不做独立路由页、不堆叠在同一页面）：
  - **Step 1 工作流配置**：按落地页选择的模式直接呈现——新建（复用 `CaseForm`，**强制导入合法 JSON**，杜绝空壳 Case）或调整已有（Case 已选中并加载到同一表单编辑）。
  - **Step 2 处理流程**：独立页面内嵌 `TaskFlowCanvas` 配置模式，行内新建/选择 Topic 与计算节点，保存 `routing` 与节点订阅。
  - **Step 3 投放**：独立页面复用 `MenuPlacementsSection` 与 `action-form`（锁定 `open_workflow`），配置渠道菜单入口。
  - **完成页**：就绪清单（工作流已导入 / 处理流程已配置 / 至少一个投放）+ **发布门禁**（发布 = 启用 Case），缺口项可回跳对应步骤补齐。
- 引入 **`@formity/react`** 作为向导编排引擎（form/variables/condition/return + 步骤屏幕切换），步骤组件自行校验与落库（官方推荐模式）。
- 分步保存与恢复：前三步只收集草稿，**完成页统一提交**（Case → routing → 菜单）；Formity 状态 + 本地会话保存完整草稿，刷新/重进可恢复；落地页「继续上次配置」一键恢复。

## Capabilities

### New Capabilities

- `quick-config-wizard`: 快速配置多步向导——落地页、三步独立步骤页（工作流配置/处理流程/投放）、完成页就绪清单与发布门禁、Formity 编排与分步保存恢复。

### Modified Capabilities

- `admin-web-shell`: 侧栏菜单新增「快速配置」项（含路由与 zh/en i18n 文案）；不改既有菜单项与页面布局。

## Impact

- 前端：`web/admin` 新增 `features/quick-config`（落地页、三步步骤页、完成页、就绪计算、Formity flow）；路由 `/quick-config` 与步骤寻址参数；侧栏 `config/menu.ts` 与 i18n 更新；依赖新增 `@formity/react`（`@xyflow/react` 已存在）。
- 消费 API（不改后端契约）：`/api/v1/cases`（list/create/patch/get）、`/api/v1/topics`（CRUD）、`/api/v1/edges`（list/create/patch、`subscribe_topics`）、`/api/v1/routing/attributes`、`/api/v1/channels`（list）、`/api/v1/channels/{id}/menu`（PUT）、`/api/v1/cases/{id}/menu-placements`。
- 前置依赖：`task-flow-editor` change 提供 `TaskFlowCanvas` 配置模式；本 change 在其上做向导编排。
- 非目标：不改后端调度/求值；不新增后端草稿态或模板 API；不做 ComfyUI workflow 画布编辑（完整 Case 编辑器保留）；不做独立路由页或单页堆叠式工作台（步骤由 Formity flow 管理）。
