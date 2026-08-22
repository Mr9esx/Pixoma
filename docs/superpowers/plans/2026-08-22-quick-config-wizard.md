---
change: quick-config-wizard
design-doc: docs/superpowers/specs/2026-08-21-quick-config-design.md
base-ref: ed3d2c38dc1f8f998d1384dd24868fc9fcf7517f
---

# quick-config-wizard 剩余任务实施计划

> **For agentic workers:** 使用 executing-plans 或 subagent-driven-development 实施。步骤用 `- [ ]`。

**Goal:** 完成 quick-config-wizard 剩余 4 个任务：Formity 步骤状态与 i18n、行内新建 Topic/节点、刷新恢复、端到端冒烟。

**Global Constraints:**
- 沿用现有 Formity flow（`quick-config-flow.tsx`）与 `lib/session.ts` 会话结构，不重写架构。
- i18n 必须 zh/en 成对；合同测试同步更新。
- `go build`/`go test`/`pnpm tsc -b`/`pnpm vitest run` 全绿。

## Task 1.3: Formity 步骤状态 + i18n

**Files:**
- Modify: `web/admin/src/features/quick-config/quick-config-flow.tsx`
- Modify: `web/admin/src/features/quick-config/lib/session.ts`
- Modify: `web/admin/src/features/quick-config/{step1-workflow,step2-processing,step3-channels,wizard-chrome}.tsx`
- Modify: `web/admin/src/lib/i18n/locales/{zh,en}.json`
- Modify: `web/admin/src/features/quick-config/quick-config.contract.test.ts`

- [x] 1.3.1 `saveQuickConfigSession` 保存当前 `step`（在 flow 内通过 `useFormity` 的 `currentIndex`/history 获取），不再硬编码 1
- [x] 1.3.2 刷新恢复时按 `session.step` 用 `jump` 回到已保存步骤（MVP：恢复后从第一步进入，保留 step 字段供后续精确跳转）
- [x] 1.3.3 向导内全部可见文案（按钮、校验错误、步骤标题、chips）改为 i18n key，zh/en 成对
- [x] 1.3.4 合同测试断言 i18n key 成对与步骤持久化字段

## Task 4.2: 行内新建 Topic 与计算节点

**Files:**
- Modify: `web/admin/src/features/task-flow/task-flow-canvas.tsx`（或 step2 工具栏）
- Modify: `web/admin/src/lib/api/topics.ts`、`web/admin/src/lib/api/edges.ts`
- Modify: `web/admin/src/features/quick-config/quick-config.contract.test.ts`

- [x] 4.2.1 「新建 Topic」行内表单：key slug 校验（`^[a-z0-9]+(-[a-z0-9]+)*$`，≤64），创建成功即时刷新 topics 查询
- [x] 4.2.2 「新建计算节点」精简向导（复用 create-edge-wizard 精简版）：name/capabilities/description，创建成功即时刷新 edges 查询
- [x] 4.2.3 合同测试：slug 校验、即时落库调用与刷新

## Task 7.2: 刷新/重进恢复

**Files:**
- Modify: `web/admin/src/features/quick-config/quick-config-page.tsx`
- Modify: `web/admin/src/features/quick-config/quick-config-flow.tsx`
- Modify: `web/admin/src/features/quick-config/quick-config.contract.test.ts`

- [x] 7.2.1 恢复时使用 `session.step` 初始化 Formity（`initialStep`/`jump`），MVP 至少恢复草稿并从第一步进入
- [x] 7.2.2 恢复后步骤指示器标记已保存步骤为完成（wizard-chrome 完成态）
- [x] 7.2.3 合同测试：刷新恢复场景（已有断言补全实现）

## Task 8.4: 端到端冒烟

**Files:**
- 验证为主（`web/admin` dev + pixoma/admin-api 后端）

- [x] 8.4.1 本地起 pixoma（或 admin-api）+ Vite，Mock 数据下走通：导入工作流建 Case → Step2 画布配置规则/Topic/节点 → 投放渠道 → 完成页全绿 → 发布 → 回读一致
- [x] 8.4.2 记录冒烟证据到验证报告
