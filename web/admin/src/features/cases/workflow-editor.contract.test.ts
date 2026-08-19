import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const here = dirname(fileURLToPath(import.meta.url))
const ZH = join(here, '../../lib/i18n/locales/zh.json')
const EN = join(here, '../../lib/i18n/locales/en.json')
const IMPORT_SECTION = join(here, 'sections/workflow-import.tsx')
const FIELD_CARDS = join(here, 'sections/field-cards.tsx')

const REQUIRED_KEYS = [
  'importHeading',
  'importHint',
  'importDropHint',
  'importValid',
  'importNodesCount',
  'importReimport',
  'importFailed',
  'importReasonNotComfy',
  'importReasonOldExport',
  'importReasonBroken',
  'inputsHeading',
  'inputsHint',
  'outputsHeading',
  'outputsHint',
  'fieldFromNode',
  'fieldParam',
  'fieldOutput',
  'fieldEnumOptions',
  'addInput',
  'addOutput',
  'autoBoundHint',
  'singleOutputAuto',
  'nodeSearchPlaceholder',
  'advancedLabel',
  'advancedReadonlyHint',
  'advancedEnterEdit',
  'advancedOverwriteWarn',
  'previewHeading',
  'previewHint',
  'emptyWorkflowLock',
  'saveWorkflow',
  'errDuplicateKey',
  'errInputNotBound',
  'errNoOutput',
] as const

function read(path: string) {
  return JSON.parse(readFileSync(path, 'utf8')) as {
    cases: Record<string, string>
  }
}

describe('workflow editor i18n', () => {
  it('zh and en define every editor key', () => {
    const zh = read(ZH)
    const en = read(EN)
    for (const key of REQUIRED_KEYS) {
      expect(zh.cases[key], `zh missing cases.${key}`).toBeTruthy()
      expect(en.cases[key], `en missing cases.${key}`).toBeTruthy()
    }
  })
})

describe('workflow import section', () => {
  it('renders import UI and uses parseWorkflow', () => {
    const source = readFileSync(IMPORT_SECTION, 'utf8')
    expect(source).toContain("data-testid='case-section-workflow-import'")
    expect(source).toContain('parseWorkflow(')
    expect(source).toContain('cases.importNodesCount')
    expect(source).toContain('cases.importValid')
    expect(source).toContain('cases.importFailed')
  })
})

describe('field cards', () => {
  it('input card picks node then parameter from workflow nodes', () => {
    const source = readFileSync(FIELD_CARDS, 'utf8')
    expect(source).toContain("data-testid='input-field-card'")
    expect(source).toContain('cases.fieldFromNode')
    expect(source).toContain('cases.fieldParam')
    expect(source).toContain('nodeLabel(')
    expect(source).toContain('nodes.find(')
  })

  it('output card hides index when node has a single output', () => {
    const source = readFileSync(FIELD_CARDS, 'utf8')
    expect(source).toContain("data-testid='output-field-card'")
    expect(source).toContain('node.outputCount')
    expect(source).toContain('cases.fieldOutput')
  })
})
