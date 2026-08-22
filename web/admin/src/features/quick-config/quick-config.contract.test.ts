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
const STEP2 = read('step2-processing.tsx')
const STEP3 = read('step3-channels.tsx')
const DONE = read('done-screen.tsx')
const READINESS = read('lib/readiness.ts')
const MENU = read('../../config/menu.ts')
const CASE_FORM = read('../../features/cases/case-form.tsx')
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
    expect(FLOW).toContain('Step2Processing')
    expect(FLOW).toContain('Step3Channels')
    expect(FLOW).toContain('DoneScreen')
    expect(FLOW).toContain('return:')
  })

  it('第一步复用 CaseForm 且新建不跳转、可强制导入', () => {
    expect(STEP1).toContain('CaseForm')
    expect(STEP1).toContain('redirectAfterSave={false}')
    expect(STEP1).toContain('requestSubmit')
  })

  it('Step 1 左右分栏（左滚动配置/右固定基础信息），Header 仅保留步骤点', () => {
    expect(STEP1).toContain('splitPane')
    expect(STEP1).toContain('leftIntro')
    expect(CASE_FORM).toContain('sticky top-0 w-80')
    expect(CASE_FORM).toContain('bg-card')
    expect(CHROME).toContain('STEP_LABELS')
    expect(CHROME).not.toContain('text-lg font-semibold')
    expect(CHROME).not.toContain('ruleCount')
  })

  it('第二步内嵌 TaskFlowCanvas 且只收集 routing 不落库', () => {
    expect(STEP2).toContain('TaskFlowCanvas')
    expect(STEP2).toContain('updateRouting')
    expect(STEP2).not.toContain('patchCase')
  })

  it('第二步支持行内新建 Topic 与计算节点并即时落库刷新', () => {
    expect(STEP2).toContain('createTopic')
    expect(STEP2).toContain('createEdge')
    expect(STEP2).toContain('isValidTopicKey')
    expect(STEP2).toContain('invalidateQueries')
  })

  it('会话保存当前步骤且向导文案 i18n 成对', () => {
    expect(FLOW).toContain('onStepChange')
    expect(FLOW).toMatch(/step,/)
    const keys = [
      'nextSave',
      'back',
      'workflowConfig',
      'processing',
      'channelPlacement',
      'done',
      'requireRule',
      'unboundRule',
      'newTopic',
      'newNode',
      'finishChecklist',
      'publish',
    ]
    for (const k of keys) {
      expect(ZH.quickConfig[k]).toBeTruthy()
      expect(EN.quickConfig[k]).toBeTruthy()
    }
  })

  it('第三步入队待提交菜单，不直接写渠道', () => {
    expect(STEP3).toContain('updatePendingEntries')
    expect(STEP3).toContain('getCaseMenuPlacements')
    expect(STEP3).not.toContain('putMenu')
  })

  it('完成页统一提交（Case → routing → 菜单）并发布', () => {
    expect(DONE).toContain('computeReadiness')
    expect(DONE).toContain('createCase')
    expect(DONE).toContain('patchCase')
    expect(DONE).toContain('addWorkflowMenuEntry')
    expect(DONE).toContain('getMenu')
    expect(DONE).toContain('putMenu')
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
