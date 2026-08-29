import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const here = dirname(fileURLToPath(import.meta.url))

function read(rel: string) {
  return readFileSync(join(here, rel), 'utf8')
}

const FLOW = read('quick-config-flow.tsx')
const CHROME = read('wizard-chrome.tsx')
const PAGE = read('quick-config-page.tsx')
const STEP1 = read('step1-workflow.tsx')
const STEP3 = read('step3-channels.tsx')
const DONE = read('done-screen.tsx')
const READINESS = read('lib/readiness.ts')
const MENU = read('../../config/menu.ts')
const WORKFLOW_EDITOR = read('../../features/cases/workflow-editor.tsx')
const ZH = JSON.parse(read('../../lib/i18n/locales/zh.json')) as {
  menu: { quickConfig: string }
  quickConfig: Record<string, string>
}
const EN = JSON.parse(read('../../lib/i18n/locales/en.json')) as {
  menu: { quickConfig: string }
  quickConfig: Record<string, string>
}

describe('quick-config wizard contract', () => {
  it('flow 由 Formity 定义四个步骤屏幕与 return', () => {
    expect(FLOW).toContain('useFormity<WizardSchema>')
    expect(FLOW).toContain('Step1Workflow')
    expect(FLOW).toContain('Step2Node')
    expect(FLOW).toContain('Step3Rules')
    expect(FLOW).toContain('Step3Channels')
    expect(FLOW).toContain('DoneScreen')
    expect(FLOW).toContain('return:')
  })

  it('向导内不再内嵌处理流程画布（TaskFlowEditor 本期不接入）', () => {
    expect(FLOW).not.toContain('Step2Processing')
    expect(FLOW).not.toContain('TaskFlowEditor')
    expect(FLOW).not.toContain('TaskFlowCanvas')
  })

  it('第一步复用统一 WorkflowEditor 且新建不跳转、可强制导入', () => {
    expect(STEP1).toContain('WorkflowEditor')
    expect(STEP1).toContain('redirectAfterSave={false}')
    expect(STEP1).toContain('requestSubmit')
  })

  it('Step 1 左右分栏（左滚动配置/右固定基础信息），Header 仅保留步骤点', () => {
    expect(STEP1).toContain('splitPane')
    expect(STEP1).toContain('leftIntro')
    expect(WORKFLOW_EDITOR).toContain('lg:w-80')
    expect(WORKFLOW_EDITOR).toContain('bg-card')
    expect(CHROME).toContain('STEP_LABELS')
    expect(CHROME).not.toContain('text-lg font-semibold')
    expect(CHROME).not.toContain('ruleCount')
  })

  it('运行节点步骤选择/新建节点且不产生订阅或写请求', () => {
    const STEP2 = read('step2-node.tsx')
    expect(STEP2).toContain('listEdges')
    expect(STEP2).toContain('listPresence')
    expect(STEP2).toContain('CreateEdgeWizard')
    expect(STEP2).toContain('updateSelectedEdge')
    expect(STEP2).not.toContain('patchEdge')
    expect(STEP2).not.toContain('createEdge')
    expect(STEP2).not.toContain('subscribe_topics')
  })

  it('特殊规则分支默认路由与跳独立页两路并存，向导内不重建编辑器', () => {
    const RULES = read('step3-rules.tsx')
    expect(RULES).toContain('updateRulesMode')
    expect(RULES).toContain("'default'")
    expect(RULES).toContain('updateRuleHandover')
    expect(RULES).toContain('patchCase')
    expect(RULES).not.toContain('TaskFlowEditor')
    expect(RULES).not.toContain('TaskFlowCanvas')
  })

  it('会话保存当前步骤且向导文案 i18n 成对', () => {
    expect(FLOW).toContain('onStepChange')
    expect(FLOW).toMatch(/step,/)
    const keys = [
      'nextSave',
      'back',
      'workflowConfig',
      'nodeSelection',
      'rulesBranch',
      'channelPlacement',
      'done',
      'newNode',
      'newChannel',
      'channelToken',
      'noChannelsHint',
      'channelCreated',
      'saveOnlyGapHint',
      'finishChecklist',
      'publish',
      'rulesQuestion',
      'noRules',
      'noRulesHint',
      'useRulesEditor',
      'useRulesEditorHint',
      'nodeOnline',
      'nodeOffline',
      'nodeSelected',
    ]
    for (const k of keys) {
      expect(ZH.quickConfig[k]).toBeTruthy()
      expect(EN.quickConfig[k]).toBeTruthy()
    }
  })

  it('第三步入队待提交菜单，并支持空态就地新建消息平台引用', () => {
    expect(STEP3).toContain('updatePendingEntries')
    expect(STEP3).toContain('getCaseMenuPlacements')
    expect(STEP3).toContain('createChannel')
    expect(STEP3).toContain('noChannelsHint')
    expect(STEP3).not.toContain('putMenu')
  })

  it('完成页统一提交（Case → routing → 菜单）并发布', () => {
    expect(DONE).toContain('computeReadiness')
    expect(DONE).toContain('createCase')
    expect(DONE).toContain('patchCase')
    expect(DONE).toContain('addWorkflowMenuEntry')
    expect(DONE).toContain('getMenu')
    expect(DONE).toContain('putMenu')
    expect(DONE).toContain('patchEdge')
    expect(DONE).toContain('selectedEdgeId')
    expect(DONE).toContain('node')
    expect(DONE).toContain('enableCase')
    expect(DONE).toContain('canPublish')
  })

  it('就绪模型三态', () => {
    expect(READINESS).toContain("'ready' | 'warn' | 'gap'")
  })

  it('侧栏包含快速配置入口且 i18n 成对', () => {
    expect(PAGE).toContain('QuickConfigFlow')
    expect(MENU).toContain("'/quick-config'")
    expect(ZH.menu.quickConfig).toBeTruthy()
    expect(EN.menu.quickConfig).toBeTruthy()
  })
})
