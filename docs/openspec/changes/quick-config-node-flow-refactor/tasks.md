## 1. 会话版本化与旧版清空

- [x] 1.1 `lib/session.ts` 会话载荷增加 `schemaVersion`（值 2），保存路径写入该标记
- [x] 1.2 向导入口（`QuickConfigPage` 的 `loadQuickConfigSession` 与恢复路径）检测旧版/缺失版本会话时清除并视为无会话
- [x] 1.3 测试：旧版会话不恢复、可重新从第一步开始（vitest）

## 2. 共享状态与 Flow 屏序重构

- [x] 2.1 `types.ts` 的 `WizardShared` 增加运行节点草稿 `selectedEdgeId`（与后续给共享的 default 绑定标记）
- [x] 2.2 `quick-config-flow.tsx` 屏序置换：工作流 → 运行节点 → 特殊规则 → 投放 → 完成页；移除原 Step2 画布固定屏
- [x] 2.3 `wizard-chrome` 步骤标签按新四步映射（zh/en）
- [x] 2.4 合同测试：新屏序、一次只渲染当前屏、已完成步骤可回跳

## 3. 运行节点步骤（Step2）

- [x] 3.1 新增 `step2-node.tsx`：`listEdges()` 可选中列表 + presence 在线态；无节点时可打开复用 `CreateEdgeWizard`
- [x] 3.2 新建成功后回填 `selectedEdgeId`；本步无任何写请求
- [x] 3.3 合约测试：选择已有节点、新建节点回填、不产生订阅/写请求

## 4. 特殊规则分支（Step3）

- [x] 4.1 新增 `step3-rules.tsx`：「不需要 → 默认路由（default）」/「需要 → 跳转既有独立规则编辑页」
- [x] 4.2 Default 分支标记待写 `default` 订阅（写入 shared 草稿）；规则分支先把工作流落库取 caseId 并标记 `handledBy='editor'`
- [x] 4.3 合约测试：两路分支选择与持久化标记

## 5. 完成页就绪与提交

- [x] 5.1 `lib/readiness.ts` 就绪项改为「工作流已导入 / 运行节点已选 / 处理流程就绪 / 至少一个投放」，三态展示
- [x] 5.2 完成页（`done-screen`）按依赖顺序提交：先 Case（工作流），再对运行节点 `patchEdge` 追加 `default`（幂等合并），再投放菜单；发布启用 Case
- [x] 5.3 缺口/警告处理：节点未选阻塞并回跳；节点离线为警告（可发布）；`patchEdge` 失败保留草稿可重试
- [x] 5.4 合约测试：四就绪项、default 订阅提交、发布成功/失败

## 6. i18n 与文档

- [x] 6.1 `zh.json` / `en.json` 新增步骤标题与分支文案
- [x] 6.2 管理配置指引补充四步向导与旧会话清空说明
- [x] 6.3 `pnpm tsc -b` + `pnpm vitest run`；`go build ./...` + `go test ./...`（无回归）
