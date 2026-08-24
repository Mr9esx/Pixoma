import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const here = dirname(fileURLToPath(import.meta.url))
const ZH = join(here, '../../lib/i18n/locales/zh.json')
const EN = join(here, '../../lib/i18n/locales/en.json')
const IMPORT_SECTION = join(here, 'sections/workflow-import.tsx')
const CODE_EDITOR = join(here, '../../components/code-editor.tsx')
const FIELD_CARDS = join(here, 'sections/field-cards.tsx')
const CASE_FORM = join(here, 'case-form.tsx')
const PREVIEW = join(here, 'sections/preview.tsx')
const ADVANCED = join(here, 'sections/advanced.tsx')

const REQUIRED_KEYS = [
  'importHeading',
  'jsonTitle',
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
  'fieldEnumOptions',
  'typeAutoSourceOutput',
  'bindFlowHintOutput',
  'typeString',
  'typeImage',
  'typeAudio',
  'typeVideo',
  'typeNumber',
  'typeBoolean',
  'typeEnum',
  'typeText',
  'typeFile',
  'editInfo',
  'editWorkflow',
  'deleteWorkflow',
  'deleteWorkflowTitle',
  'deleteWorkflowBody',
  'deleteWorkflowNeedDisable',
  'deleteSuccess',
  'deleteFailed',
  'fieldBind',
  'bindPickPlaceholder',
  'bindDialogTitle',
  'bindFlowHint',
  'bindPickNodeFirst',
  'bindPickParam',
  'bindNoParams',
  'bindWillBind',
  'bindConfirm',
  'typeAuto',
  'typeCustom',
  'typeRestoreAuto',
  'typeAutoSource',
  'addInput',
  'addOutput',
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
    expect(source).toContain("'../lib/workflow-parse'")
    expect(source).toContain("<Upload className='size-3' />")
    expect(source).toContain('cases.importHeading')
    expect(source).toContain('cases.importNodesCount')
    expect(source).toContain('cases.importValid')
    expect(source).toContain('cases.importFailed')
    expect(source).toContain('fileRef.current?.click()')
    expect(source).toContain("role='button'")
    expect(source).toContain("accept='.json,application/json'")
    expect(source).toContain('CodeEditor')
    expect(source).toContain('cases.jsonTitle')
    expect(source).toContain('maxHeight={250}')
  })
})

describe('code editor component', () => {
  it('renders an IDE-style dark container with an in-border header', () => {
    const source = readFileSync(CODE_EDITOR, 'utf8')
    expect(source).toContain('export function CodeEditor')
    expect(source).toContain('title')
    expect(source).toContain('border-b border-[#30363d]')
    expect(source).toContain("backgroundColor: '#0d1117'")
    expect(source).toContain('maxHeight = 250')
    expect(source).toContain("color: '#7ee787'")
    expect(source).toContain("theme='none'")
    expect(source).toContain('lab(75.0771%')
    expect(source).not.toContain('focus-within')
  })
})

describe('field cards', () => {
  it('input card binds via flow dialog, filters refs, shows auto/custom type', () => {
    const source = readFileSync(FIELD_CARDS, 'utf8')
    expect(source).toContain("data-testid='input-field-card'")
    expect(source).toContain('BindNodeDialog')
    expect(source).toContain('cases.bindPickPlaceholder')
    expect(source).toContain('cases.bindConfirm')
    expect(source).toContain('!input.ref')
    expect(source).toContain('cases.typeAutoSource')
    expect(source).toContain('cases.typeRestoreAuto')
    expect(source).toContain('cases.typeCustom')
    expect(source).toContain('nodeVisualFor(')
    expect(source).toContain('inputKindFor(')
    expect(source).not.toContain('autoBoundHint')
  })

  it('output card binds via flow dialog slots with single-output auto note', () => {
    const source = readFileSync(FIELD_CARDS, 'utf8')
    expect(source).toContain("data-testid='output-field-card'")
    expect(source).toContain("mode='output'")
    expect(source).toContain('outputKindFor(')
    expect(source).toContain('selected.outputCount')
    expect(source).toContain('cases.singleOutputAuto')
    expect(source).toContain('cases.bindFlowHintOutput')
  })
})

describe('editor form assembly', () => {
  it('form renders step sections and no raw workflow textarea by default', () => {
    const form = readFileSync(CASE_FORM, 'utf8')
    expect(form).toContain('WorkflowImportSection')
    expect(form).toContain('InputFieldCard')
    expect(form).toContain('OutputFieldCard')
    expect(form).toContain('PreviewSection')
    expect(form).toContain('AdvancedSection')
    expect(form).toContain('deriveBindings(')
    expect(form).toContain('deriveInputSchema(')
    expect(form).not.toContain('WorkflowJsonSection')
    expect(form).not.toContain('InputSchemaSection')
    expect(form).not.toContain('BindingsSection')
    expect(form).not.toContain('IoFieldsSection')
  })

  it('preview is read-only and advanced is gated behind open + confirm', () => {
    const preview = readFileSync(PREVIEW, 'utf8')
    expect(preview).toContain("data-testid='case-section-preview'")
    expect(preview).toContain('aria-readonly')
    const advanced = readFileSync(ADVANCED, 'utf8')
    expect(advanced).toContain("data-testid='case-section-advanced'")
    expect(advanced).toContain('editMode')
    expect(advanced).toContain('cases.advancedEnterEdit')
    expect(advanced).toContain('readOnly')
  })
})
