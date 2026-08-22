## 1. 依赖与入口

- [x] 1.1 引入 `@formity/react`；spike 验证三步 flow（form/variables/condition/return）嵌入画布与 `CaseForm` 自定义组件、RHF 校验后 `next()` 模式（参照 formity.app/examples）；受阻则记录 `setup-wizard` 回退方案
- [x] 1.2 侧栏在 Dashboard 后新增「快速配置」项（icon Zap）与落地页 `/quick-config`（支持 `?caseId=` 深链）
- [x] 1.3 Formity flow 步骤状态：`useFormity`（history/jump/params）与本地会话恢复；zh/en i18n 文案成对（menu 文案已成对；向导内文案待全量 i18n）

## 2. 落地页与向导壳

- [x] 2.1 落地页：继续上次配置卡片（本地会话 caseId + 步骤）、新建工作流入口、已有 Case 搜索列表（复用 `CaseListPanel`）
- [x] 2.2 向导壳：进度「Step X of 3」+ 步骤指示器（已完成可回跳、未完成锁定）、每步全屏切换不堆叠、顶栏摘要 chips
- [x] 2.3 底部导航：上一步 / 下一步（保存并继续），Step 3 为「完成并查看摘要」

## 3. Step 1 工作流配置页

- [x] 3.1 模式由落地页入口决定（Formity 变量 `mode` 传入），Step 1 不再提供模式切换；顶部仅展示当前模式标签与「返回落地页更换」出口
- [x] 3.2 新建：复用 `CaseForm` 创建流程（`WorkflowImportSection` 导入 → `FieldCards` 字段 → `BasicsSection`），**未导入合法 JSON 不得前进**
- [x] 3.3 调整已有：`CaseListPanel` 选择后加载基础信息到表单编辑，可重新导入 JSON 替换 workflow
- [x] 3.4 收集载荷：`collectOnly` 模式校验后把 Case 草稿写入共享状态，不落库；完成页统一提交

## 4. Step 2 处理流程页

- [x] 4.1 `TaskFlowCanvas` 配置模式嵌入 Formity 步骤屏幕（工具栏：添加规则/新建 Topic/选择 Topic/新建节点/选择节点）
- [x] 4.2 行内新建 Topic（key slug + `[a-z0-9-]` 校验）与计算节点（精简 create-edge-wizard）即时落库
- [x] 4.3 收集 routing：保存前校验并定位错误到分支；落库统一在完成页 `PATCH /cases/{id}` 写 `routing`
- [x] 4.4 状态提示（ready / bound-offline / unbound）与离线弱化，供完成页就绪计算使用

## 5. Step 3 投放页

- [x] 5.1 展示已有投放（复用 `MenuPlacementsSection` 数据源，MVP 自绘列表）
- [x] 5.2 渠道卡片：启用渠道可配置（锁定 `open_workflow` + direct/list），未启用置灰
- [x] 5.3 入队待提交菜单（渠道 + 标签 + 模式），完成页统一读菜单 → 追加 → `PUT` 全量写回

## 6. 完成页：就绪清单与发布门禁

- [x] 6.1 就绪计算（G1/G2/G3）与三态渲染（就绪/警告/缺口）
- [x] 6.2 发布按钮门禁：缺口禁用并列出原因，点击缺口项回跳对应步骤；全绿可用
- [x] 6.3 发布操作：`POST /cases/{id}/enable`；已发布状态展示

## 7. 保存与恢复

- [x] 7.1 前三步只收集草稿、可自由回跳；本地会话记录完整草稿（caseDraft/routing/pendingEntries）
- [x] 7.2 刷新/重进恢复：Formity 状态 + 本地会话（caseId + 步骤下标）`jump` 到已保存步骤并标记完成状态（MVP：恢复草稿后从第一步进入）

## 8. 质量与联调

- [x] 8.1 Contract 测试：落地页三入口、Formity 步骤一次只显示一步、进度与回跳门控、未导入阻止前进、就绪清单与发布门禁、离线警告、刷新恢复、i18n 成对
- [x] 8.2 单测：就绪计算（G1/G2/G3）、菜单按钮追加的 PUT 载荷、Formity flow 分支（新建/已有）、key slug
- [x] 8.3 `pnpm build` + `pnpm test` 通过；bundle 观察 `@formity/react` 体积
- [x] 8.4 端到端冒烟：导入工作流建 Case → Step 2 画布配置规则/Topic/节点 → 投放渠道 → 完成页全绿 → 发布 → 回读一致
