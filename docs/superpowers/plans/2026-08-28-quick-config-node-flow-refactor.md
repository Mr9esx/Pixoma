# 快速配置四步向导重构 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans（本项目默认 build_mode）配合 superpowers:test-driven-development 逐任务实施。步骤用 `- [ ]` 勾选跟踪。

**Goal:** 把快速配置重构为「工作流编辑 → 运行节点 → 特殊规则 → 投放 → 完成页」，Default 分支把运行节点绑定 default、规则分支跳既有独立页，旧会话清空。

**Architecture:** 前端 `web/admin` 内重构（后端零新增）。基于 Formity 4 屏替换：[`quick-config-flow.tsx`](web/admin/src/features/quick-config/quick-config-flow.tsx) 屏序调整，新增 `step2-node.tsx` 与 `step3-rules.tsx`，`done-screen.tsx` 提交阶段对运行节点执行 `patchEdge` 追加 default；`session.ts` 加 schemaVersion 清空旧会话。

**Tech Stack:** React/TS · TanStack Query · Formity · shadcn/ui · vitest（合同测试用文件断言）；Go 后端不改。

---
change: quick-config-node-flow-refactor
design-doc: docs/superpowers/specs/2026-08-28-quick-config-node-flow-refactor-design.md
base-ref: 7632b5c0ddd0531a52b508da6531755169935a0a
---

## Global Constraints
- 全部代码在 `web/admin/src/`，TypeScript strict；组件优先复用既有 shadcn/ui 与 `features/edges`、`features/task-flow` 已装组件。
- 新增/改动文案必须成对写入 `web/admin/src/lib/i18n/locales/zh.json` 与 `en.json`。
- **后端零改动**：不做 Case 归属字段；只用已有 `patchEdge(id, {subscribe_topics})`、`listEdges`、`listPresence`。
- **不在向导内使用 `TaskFlowEditor`/`TaskFlowCanvas`**；规则分支走跳转既有独立规则编辑页。
- 不提交 git（除非计划步骤显式要求；本项目 guard 管理）。

---

### Task 1: 会话版本化与旧版清空

**Files:**
- Modify: `web/admin/src/features/quick-config/lib/session.ts`
- Modify: `web/admin/src/features/quick-config/quick-config-page.tsx`
- Test: `web/admin/src/features/quick-config/lib/session.test.ts`

**Interfaces:**
- Consumes: 现有 `QuickConfigSession`。
- Produces: `QuickConfigSession.schemaVersion: 2`；`loadQuickConfigSession` 在 `schemaVersion` 缺失或 `!== 2` 时返回 null（旧版清空）。

- [x] **Step 1: 写失败测试**
`lib/session.test.ts` 增加：
```ts
it('旧版会话（无 schemaVersion）被清空', () => {
  const storage = fakeStorage()
  storage.setItem(SESSION_KEY, JSON.stringify({
    caseId: 7, mode: 'create', step: 1, updatedAt: '2026-08-21T09:00:00.000Z',
  }))
  expect(loadQuickConfigSession(storage)).toBeNull()
  expect(storage.getItem(SESSION_KEY)).toBeNull()
})
it('保存的会话带有 schemaVersion 2 并可读回', () => {
  const storage = fakeStorage()
  saveQuickConfigSession(storage, { ...session, schemaVersion: 2 })
  expect(loadQuickConfigSession(storage)).toEqual({ ...session, schemaVersion: 2 })
})
```
（同时把现有 `session` fixture 及「旧版会话缺少草稿字段时也能读回」用例改为附带 `schemaVersion: 2`。）

- [x] **Step 2: 运行验证失败**
`pnpm vitest run src/features/quick-config/lib/session.test.ts` → 新用例 FAIL（schemaVersion 未定义 / 旧版未清空）。

- [x] **Step 3: 最小实现**
`session.ts`：
```ts
export const SESSION_SCHEMA_VERSION = 2
export type QuickConfigSession = {
  caseId: number | null
  mode: 'create' | 'existing'
  step: number
  caseDraft: unknown
  routing: unknown
  pendingEntries: PendingMenuEntry[]
  schemaVersion: typeof SESSION_SCHEMA_VERSION
  updatedAt: string
}
```
`saveQuickConfigSession` 写入 `{ ...session, schemaVersion: SESSION_SCHEMA_VERSION }`。
`loadQuickConfigSession`：解析后先 `if ((candidate as Partial<QuickConfigSession>).schemaVersion !== SESSION_SCHEMA_VERSION) { storage.removeItem(SESSION_KEY); return null }`，再走原字段校验。

- [x] **Step 4: 运行验证通过**
`pnpm vitest run src/features/quick-config/lib/session.test.ts` → PASS。`quick-config-page.tsx` 的 `loadQuickConfigSession` 会自动随库清空（无需改页面逻辑，若页面读取 `schemaVersion` 有类型问题则补非空断言）。

---

### Task 2: 共享状态 + Flow 屏序重构

**Files:**
- Modify: `web/admin/src/features/quick-config/types.ts`
- Modify: `web/admin/src/features/quick-config/quick-config-flow.tsx`
- Modify: `web/admin/src/features/quick-config/wizard-chrome.tsx`
- Test: `web/admin/src/features/quick-config/quick-config.contract.test.ts`

**Interfaces:**
- Consumes: Task1 的 `QuickConfigSession`（含 schemaVersion）。
- Produces:
  - `WizardShared.selectedEdgeId: string | null`（+ `setSelectedEdge`）
  - `WizardShared.rulesMode: 'default' | 'editor'`（草稿态标记，规则分支用）
  - `WizardShared.ruleHandover: boolean`（规则分支已跳转标记）

- [x] **Step 1: 写失败测试**
`quick-config.contract.test.ts` 更新：
```ts
it('flow 由 Formity 定义四配置屏与 return', () => {
  expect(FLOW).toContain('Step1Workflow')
  expect(FLOW).toContain('Step2Node')
  expect(FLOW).toContain('Step3Rules')
  expect(FLOW).toContain('Step3Channels')
  expect(FLOW).toContain('DoneScreen')
})
it('不再在向导内嵌入处理流程画布', () => {
  expect(FLOW).not.toContain('Step2Processing')
  expect(FLOW).not.toContain('TaskFlowEditor')
  expect(FLOW).not.toContain('TaskFlowCanvas')
})
```
（移除原「第二步内嵌 TaskFlowCanvas」「第二步行内新建任务队列」两条用例的 STEP2 断言组。）保留「第一步复用 WorkflowEditor」「会话 i18n 成对」等。把原「flow 由 Formity…」用例断言 `Step2Processing`/`Step2Node` 替换。

- [x] **Step 2: 运行验证失败**
`pnpm vitest run src/features/quick-config/quick-config.contract.test.ts` → FAIL。

- [x] **Step 3: 最小实现**
`types.ts`：
```ts
export type RulesMode = 'default' | 'editor'
export type WizardShared = { /* 既有字段 */ 
  selectedEdgeId: string | null
  rulesMode: RulesMode
  ruleHandover: boolean
  updateSelectedEdge: (id: string | null) => void
  updateRulesMode: (m: RulesMode) => void
  updateRuleHandover: (v: boolean) => void
}
```
`quick-config-flow.tsx`：`struct` 保持 4 Form 屏 + Return；render 依次 `Step1Workflow`、`Step2Node`、`Step3Rules`、`Step3Channels`、`DoneScreen`；新增 state `selectedEdgeId`/`rulesMode`/`ruleHandover` 与对应回调并入 `shared`；移除 `import { Step2Processing }`。
`wizard-chrome.tsx`：`STEP_LABELS` 改为 `[t('quickConfig.workflowConfig'), t('quickConfig.nodeSelection'), t('quickConfig.rulesBranch'), t('quickConfig.channelPlacement')]`，进度文案 `Step {step} of 4`。

- [x] **Step 4: 运行验证通过**
contract test PASS。

---

### Task 3: 运行节点步骤（Step2）

**Files:**
- Create: `web/admin/src/features/quick-config/step2-node.tsx`
- Modify: `web/admin/src/features/quick-config/quick-config-flow.tsx`（已接线）
- Test: `web/admin/src/features/quick-config/quick-config.contract.test.ts`

**Interfaces:**
- Consumes: `WizardShared`（Task2）、`listEdges`/`listPresence`/`CreateEdgeWizard`。
- Produces: 运行节点步骤组件 `Step2Node({ shared, next, back })`；选择后置 `shared.selectedEdgeId`，无任何写请求。

- [x] **Step 1: 写失败测试**
contract 增加：
```ts
it('运行节点步骤不产生订阅/写请求', () => {
  const NODE = read('step2-node.tsx')
  expect(NODE).toContain('listEdges')
  expect(NODE).toContain('listPresence')
  expect(NODE).toContain('CreateEdgeWizard')
  expect(NODE).not.toContain('patchEdge')
  expect(NODE).not.toContain('subscribe_topics')
  expect(NODE).not.toContain('createEdge') // 新建走 CreateEdgeWizard，不直接 createEdge
})
```

- [x] **Step 2: 运行验证失败** → 新用例 FAIL（`step2-node.tsx` 不存在）。

- [x] **Step 3: 最小实现**
```tsx
// step2-node.tsx
import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Plus } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { listEdges, listPresence } from '@/lib/api/edges'
import { queryKeys } from '@/lib/api/query-keys'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { CreateEdgeWizard } from '@/features/edges/create-edge-wizard'
import { cn } from '@/lib/utils'
import type { StepActions, WizardShared } from './types'
import { WizardChrome } from './wizard-chrome'

export function Step2Node({ shared, next, back }: StepActions & { shared: WizardShared }) {
  const { t } = useTranslation()
  const [createOpen, setCreateOpen] = useState(false)
  const edgesQuery = useQuery({ queryKey: queryKeys.edges.all, queryFn: listEdges })
  const presenceQuery = useQuery({ queryKey: queryKeys.edges.presence, queryFn: listPresence })
  const edges = edgesQuery.data ?? []
  const presence = presenceQuery.data ?? []

  return (
    <WizardChrome step={2} onBack={() => back({})} onNext={() => next({})}
      nextLabel={t('quickConfig.next')} nextDisabled={!shared.selectedEdgeId}>
      <div className='space-y-3'>
        <div className='flex items-center justify-between'>
          <h3 className='text-sm font-semibold'>{t('quickConfig.nodeSelection')}</h3>
          <Button type='button' variant='outline' size='sm' onClick={() => setCreateOpen(true)}>
            <Plus className='size-4' /> {t('quickConfig.newNode')}
          </Button>
        </div>
        {edgesQuery.isLoading ? <LoadingSkeleton rows={2} />
          : edges.map((edge) => {
            const p = presence.find((r) => r.id === edge.id)
            const online = p?.edge_online === true && p?.comfy_running === true
            return (
              <button key={edge.id} type='button'
                onClick={() => shared.updateSelectedEdge(edge.id)}
                className={cn('flex w-full items-center justify-between rounded-md border border-border px-3 py-2.5 text-left',
                  shared.selectedEdgeId === edge.id && 'border-primary bg-muted/60')}>
                <span className='text-sm font-medium'>{edge.name}</span>
                <span className={cn('text-xs', online ? 'text-emerald-600' : 'text-muted-foreground')}>
                  {online ? t('quickConfig.nodeOnline') : t('quickConfig.nodeOffline')}
                </span>
              </button>
            )
          })}
      </div>
      <Dialog open={createOpen} onOpenChange={setCreateOpen}>
        <DialogContent>
          <DialogHeader><DialogTitle>{t('quickConfig.newNode')}</DialogTitle></DialogHeader>
          <CreateEdgeWizard onDone={(edge) => { shared.updateSelectedEdge(edge.id); setCreateOpen(false) }} />
        </DialogContent>
      </Dialog>
    </WizardChrome>
  )
}
```

- [x] **Step 4: 运行验证通过**（contract test PASS）

---

### Task 4: 特殊规则分支（Step3）

**Files:**
- Create: `web/admin/src/features/quick-config/step3-rules.tsx`
- Modify: `web/admin/src/features/quick-config/quick-config-flow.tsx`（已接线）
- Test: `web/admin/src/features/quick-config/quick-config.contract.test.ts`

**Interfaces:**
- Consumes: `WizardShared`、`caseId`、`createCase`/`patchCase`（规则分支落库取 caseId）。
- Produces: `Step3Rules({ shared, next, back })`；选「不需要」→ `updateRulesMode('default')` 并 `next`；选「需要」→ 先落库工作流取 `caseId`，`updateRulesMode('editor')`、`updateRuleHandover(true)`，跳转既有独立规则编辑页（本向导由 `onExit` 退出）。Default 分支的 default 订阅写入安排到完成页提交（Task5）。

- [x] **Step 1: 写失败测试**
```ts
const RULES = read('step3-rules.tsx')
it('特殊规则分支默认路由/跳独立页两路并存', () => {
  expect(RULES).toContain('updateRulesMode')
  expect(RULES).toContain("'default'")
  expect(RULES).toContain('ruleHandover')
  expect(RULES).toContain('patchCase') // 规则分支落库取 caseId
  // 不在向导内重建编辑器
  expect(RULES).not.toContain('TaskFlowEditor')
  expect(RULES).not.toContain('TaskFlowCanvas')
})
```

- [x] **Step 2: 运行验证失败** → FAIL（文件不存在）。

- [x] **Step 3: 最小实现**
```tsx
// step3-rules.tsx
import { useMutation } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { RouterLink } from '@/components/router-link' // 若存在；否则用 <a href>
import { createCase, patchCase } from '@/lib/api/cases'
import type { StepActions, WizardShared } from './types'
import { WizardChrome } from './wizard-chrome'

export function Step3Rules({ shared, next, back }: StepActions & { shared: WizardShared }) {
  const { t } = useTranslation()
  const handover = useMutation({
    mutationFn: async () => {
      const draft = shared.caseRecord
      if (!draft) throw new Error('quickConfig.notReady')
      const saved = shared.caseId != null
        ? await patchCase(shared.caseId, draft)
        : await createCase(draft)
      shared.updateCase(saved)
      return saved
    },
    onSuccess: () => {
      shared.updateRulesMode('editor')
      shared.updateRuleHandover(true)
      shared.onExit()
    },
  })
  return (
    <WizardChrome step={3} onBack={() => back({})} nextLabel={t('quickConfig.next')}
      onNext={() => { shared.updateRulesMode('default'); next({}) }}>
      <div className='space-y-3'>
        <h3 className='text-sm font-semibold'>{t('quickConfig.rulesQuestion')}</h3>
        <button type='button' onClick={() => { shared.updateRulesMode('default'); next({}) }}
          className='block w-full rounded-md border border-border px-3 py-2.5 text-left'>
          <span className='font-medium'>{t('quickConfig.noRules')}</span>
          <span className='mt-0.5 block text-xs text-muted-foreground'>{t('quickConfig.noRulesHint')}</span>
        </button>
        <button type='button' onClick={() => handover.mutate()} disabled={handover.isPending}
          className='block w-full rounded-md border border-border px-3 py-2.5 text-left'>
          <span className='font-medium'>{t('quickConfig.useRulesEditor')}</span>
          <span className='mt-0.5 block text-xs text-muted-foreground'>{t('quickConfig.useRulesEditorHint')}</span>
        </button>
      </div>
    </WizardChrome>
  )
}
```

- [x] **Step 4: 运行验证通过**（contract test PASS；注意 `RouterLink` import 是否真实存在，不存在则去掉并仅用按钮 + onExit）

---

### Task 5: 完成页就绪与提交

**Files:**
- Modify: `web/admin/src/features/quick-config/done-screen.tsx`
- Modify: `web/admin/src/features/quick-config/lib/readiness.ts`
- Test: `web/admin/src/features/quick-config/lib/readiness.test.ts`、`quick-config.contract.test.ts`

**Interfaces:**
- Consumes: `WizardShared.selectedEdgeId/rulesMode`、`patchEdge`、`listEdges`。
- Produces: `computeReadiness` 输入增加 `selectedNodeSelected: boolean`、`hasDefaultRoute: boolean`；就绪对象增加 `node: ReadinessLevel`；提交顺序 `Case → 节点 default 订阅(仅 default 分支) → 菜单`。

- [x] **Step 1: 写失败测试**
`readiness.test.ts` 增加：
```ts
it('节点未选则 node 就绪为 gap', () => {
  const base = { workflow: 'ready' as const, rules: [], enabledTopics: ['default'], boundTopics: ['default'], onlineTopics: ['default'], placements: [], selectedNodeSelected: false, hasDefaultRoute: true }
  expect(computeReadiness(base).node).toBe('gap'); expect(computeReadiness(base).processing).not.toBe('gap')
})
it('节点已选且 default 路由则全部就绪', () => {
  const r = computeReadiness({ workflow: 'ready', rules: [], enabledTopics: ['default'], boundTopics: ['default'], onlineTopics: [], placements: [{ channelId: 'c', label: 'x', mode: 'direct' }], selectedNodeSelected: true, hasDefaultRoute: true })
  expect(r).toEqual({ workflow: 'ready', processing: 'warn', placements: 'ready', node: 'ready' })
})
```
contract 更新：`DONE` 断言 `patchEdge`、`selectedEdgeId`、`node`。

- [x] **Step 2: 运行验证失败** → FAIL（readiness 无 node 字段）。

- [x] **Step 3: 最小实现**
`readiness.ts`：`ProcessingInput` 增 `selectedNodeSelected: boolean; hasDefaultRoute: boolean`；处理流程计算中，`rules.length===0 && !hasDefaultRoute` → `processing='gap'`（规则分支 handover 后由独立页负责，视为需回跳而非在向导内完成，仅 default 分支在向导内就绪）；`node = selectedNodeSelected ? (onlineNode ? 'ready' : 'warn') : 'gap'`。`Readiness` 增 `node`。
`done-screen.tsx`：
- 就绪计算：`node: shared.selectedEdgeId ? (选中节点在线 ? 'ready' : 'warn') : 'gap'`；`processing` 用现有逻辑但 default 分支保证就绪；`change rows` 增加 `{ key: 'node', title: 'quickConfig.nodeSelected' }`。
- 提交 `mutationFn`：default 分支（`shared.rulesMode === 'default' && shared.selectedEdgeId`）在保存 Case 后：
```ts
if (shared.rulesMode === 'default' && shared.selectedEdgeId) {
  const edge = edgesQuery.data?.find((e) => e.id === shared.selectedEdgeId)
  const topics = edge ? edge.subscribe_topics ?? [] : []
  if (!topics.includes(DEFAULT_TOPIC_KEY)) {
    await patchEdge(shared.selectedEdgeId, { subscribe_topics: [...topics, DEFAULT_TOPIC_KEY] })
  }
}
```
- `rows` 含 `node`；`computed.canPublish` 基于新 readiness。

- [x] **Step 4: 运行验证通过**（readiness 单测 + contract PASS）

- [x] **Step 5: 全量验证**
`cd web/admin && pnpm tsc -b && pnpm vitest run`

---

### Task 6: i18n 与文档

**Files:**
- Modify: `web/admin/src/lib/i18n/locales/zh.json`
- Modify: `web/admin/src/lib/i18n/locales/en.json`
- Modify: 管理配置指引（`docs/` 相关 md）

- [x] **Step 1: 补 i18n（先配对再引）**
`zh.json`/`en.json` `quickConfig.*` 补：`nodeSelection`、`rulesBranch`、`rulesQuestion`、`noRules`、`noRulesHint`、`useRulesEditor`、`useRulesEditorHint`、`nodeOnline`、`nodeOffline`、`nodeSelected`。contract test 的 i18n keys 数组同步增补。
- [x] **Step 2: 运行验证**
`pnpm vitest run src/features/quick-config/quick-config.contract.test.ts`（keys 成对断言通过）。
- [x] **Step 3: 更新指引**
`docs/` 管理配置指引补充四步向导说明与旧会话清空提示。
- [x] **Step 4: 全量验证**
`cd web/admin && pnpm tsc -b && pnpm vitest run`；`cd .. && go build ./... && go test ./...`
