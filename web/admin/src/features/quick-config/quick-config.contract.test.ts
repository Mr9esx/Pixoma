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
const STEP2Q = read('step2-queue.tsx')
const STEP2N = read('step2-node.tsx')
const STEP4 = read('step4-next.tsx')
const COMMIT = read('lib/commit.ts')
const MENU = read('../../config/menu.ts')
const WORKFLOW_EDITOR = read('../cases/workflow-editor.tsx')
const ZH = JSON.parse(read('../../lib/i18n/locales/zh.json')) as {
  menu: { quickConfig: string }
  quickConfig: Record<string, string>
}
const EN = JSON.parse(read('../../lib/i18n/locales/en.json')) as {
  menu: { quickConfig: string }
  quickConfig: Record<string, string>
}

describe('quick-create wizard contract', () => {
  it('flow 为工作流、队列、节点、还差一步', () => {
    expect(FLOW).toContain('Step1Workflow')
    expect(FLOW).toContain('Step2Queue')
    expect(FLOW).toContain('Step2Node')
    expect(FLOW).toContain('Step4Next')
    expect(FLOW).not.toContain('Step3Channels')
    expect(FLOW).not.toContain('DoneScreen')
  })

  it('无草稿时侧栏进入即向导，不经过创建按钮', () => {
    expect(PAGE).not.toContain('listCases')
    expect(PAGE).not.toContain('选择已有工作流')
    expect(PAGE).not.toContain('qc-create-btn')
    expect(PAGE).toContain('loadQuickConfigSession')
    expect(PAGE).toContain("to: '/'")
  })

  it('第一步隐藏处理流程', () => {
    expect(STEP1).toContain('hideProcessing')
    expect(STEP1).toContain('hideFooter')
    expect(STEP1).not.toContain('onNext')
    expect(WORKFLOW_EDITOR).toContain('hideProcessing')
  })

  it('节点步提交链写 always 规则并启用', () => {
    expect(STEP2N).toContain('commitQuickCreate')
    expect(COMMIT).toContain('always: true')
    expect(COMMIT).toContain('createCase')
    expect(COMMIT).toContain('enableCase')
    expect(COMMIT).toContain('patchEdge')
    expect(COMMIT).not.toContain('putMenu')
  })

  it('队列步可选已有或新建', () => {
    expect(STEP2Q).toContain('listTopics')
    expect(STEP2Q).toContain('CreateTopicForm')
    expect(STEP2Q).toContain('DEFAULT_TOPIC_KEY')
  })

  it('还差一步进消息平台且含未通提醒', () => {
    expect(STEP4).toContain('/channels')
    expect(STEP4).toContain('leftover-offline')
    expect(STEP4).toContain('goChannels')
  })

  it('chrome 四步标签', () => {
    expect(CHROME).toContain('queueStep')
    expect(CHROME).toContain('leftoverTitle')
  })

  it('侧栏入口仍为 /quick-config 且成对 i18n', () => {
    expect(MENU).toContain("'/quick-config'")
    expect(ZH.menu.quickConfig).toBeTruthy()
    expect(EN.menu.quickConfig).toBeTruthy()
    for (const k of [
      'queueStep',
      'leftoverTitle',
      'goChannels',
      'nodeNotReady',
    ] as const) {
      expect(ZH.quickConfig[k]).toBeTruthy()
      expect(EN.quickConfig[k]).toBeTruthy()
    }
  })
})
