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
const WORKFLOW_EDITOR = join(here, 'workflow-editor.tsx')
const CASE_FORM = join(here, 'case-form.tsx')
const STEP1 = join(here, '../quick-config/step1-workflow.tsx')

const REQUIRED_KEYS = [
  'importHeading',
  'jsonTitle',
  'importHint',
  'importDropHint',
  'importValid',
  'importNodesCount',
  'importReimport',
  'viewDiagram',
  'viewSource',
  'importFailed',
  'importReasonNotComfy',
  'importReasonOldExport',
  'importReasonBroken',
  'inputsHeading',
  'inputsHint',
  'outputsHeading',
  'outputsHint',
  'processingPill',
  'processingHeading',
  'processingHint',
  'fieldFromNode',
  'fieldEnumOptions',
  'typeAutoSourceOutput',
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
  'deleteSuccess',
  'deleteFailed',
  'fieldBind',
  'bindPickPlaceholder',
  'bindNoParams',
  'bindSearchPlaceholder',
  'typeAuto',
  'typeCustom',
  'typeRestoreAuto',
  'restoreSuggestedType',
  'typeAutoSource',
  'addInput',
  'addOutput',
  'singleOutputAuto',
  'emptyWorkflowLock',
  'autoGenerateInputs',
  'autoGenerateOutputs',
] as const

function readJson(path: string) {
  return JSON.parse(readFileSync(path, 'utf8')) as {
    cases: Record<string, string>
  }
}

function read(path: string) {
  return readFileSync(path, 'utf8')
}

describe('workflow editor i18n', () => {
  it('zh and en define every editor key', () => {
    const zh = readJson(ZH)
    const en = readJson(EN)
    for (const key of REQUIRED_KEYS) {
      expect(zh.cases[key], `zh missing cases.${key}`).toBeTruthy()
      expect(en.cases[key], `en missing cases.${key}`).toBeTruthy()
    }
  })
})

describe('workflow import section', () => {
  it('renders import UI and uses parseWorkflow', () => {
    const source = read(IMPORT_SECTION)
    expect(source).toContain("data-testid='case-section-workflow-import'")
    expect(source).toContain("'../lib/workflow-parse'")
    expect(source).toContain(
      "rounded-full bg-primary text-xs font-semibold text-primary-foreground'"
    )
    expect(source).toMatch(/text-primary-foreground'[\s\S]*?\n\s*1\n/)
    expect(source).toContain('cases.importHeading')
    expect(source).toContain('cases.importNodesCount')
    expect(source).toContain('cases.importValid')
    expect(source).toContain('cases.importFailed')
    expect(source).toContain('fileRef.current?.click()')
    expect(source).toContain("from '@/components/ui/attachment'")
    expect(source).toContain('<Attachment')
    expect(source).toContain('AttachmentTrigger')
    expect(source).toContain("accept='.json,application/json'")
    expect(source).toContain('CodeEditor')
    expect(source).toContain('cases.jsonTitle')
    expect(source).toContain('maxHeight={180}')
  })

  it('defaults to diagram view and toggles source via a shadcn Tabs segmented control', () => {
    const source = read(IMPORT_SECTION)
    expect(source).toContain('cases.viewDiagram')
    expect(source).toContain('cases.viewSource')
    expect(source).toContain("data-testid='workflow-import-diagram'")
    expect(source).toContain("useState<ViewMode>('diagram')")
    expect(source).toContain('function NodeDiagram')
    expect(source).toContain('nodeLabel(')
    expect(source).toContain("from '@/components/ui/tabs'")
    expect(source).toContain('<Tabs')
    expect(source).toContain('<TabsList')
    expect(source).toContain('<TabsTrigger')
    expect(source).toContain(
      'onValueChange={(next) => onChange(next as ViewMode)}'
    )
    expect(source).not.toContain('aria-pressed')
  })
})

describe('code editor component', () => {
  it('renders a theme-consistent container with an in-border header', () => {
    const source = read(CODE_EDITOR)
    expect(source).toContain('export function CodeEditor')
    expect(source).toContain('title')
    expect(source).toContain('border-b border-border')
    expect(source).toContain('var(--card)')
    expect(source).toContain('var(--muted-foreground)')
    expect(source).toContain('maxHeight = 250')
    expect(source).toContain("theme='none'")
    expect(source).not.toContain('#0d1117')
    expect(source).not.toContain('#30363d')
    expect(source).not.toContain('focus-within')
  })
})

describe('field cards (table + anchored bind popover)', () => {
  it('input card binds via lightweight popover, filters refs, shows auto/custom type', () => {
    const source = read(FIELD_CARDS)
    expect(source).toContain("data-testid='input-field-card'")
    expect(source).toContain('BindNodePopover')
    expect(source).toContain('cases.bindPickPlaceholder')
    expect(source).toContain('cases.bindSearchPlaceholder')
    expect(source).toContain('<CommandInput')
    expect(source).toContain('<CommandSeparator')
    expect(source).toContain('<CommandGroup')
    expect(source).toContain('ChevronsUpDown')
    expect(source).toContain('!i.ref')
    expect(source).toContain('cases.typeAutoSource')
    expect(source).toContain('cases.typeAuto')
    expect(source).toContain('RestoreTypeButton')
    expect(source).toContain('cases.restoreSuggestedType')
    expect(source).toContain('<Tooltip')
    expect(source).toContain('RefreshCw')
    expect(source).toContain('nodeVisualFor(')
    expect(source).toContain('inputKindFor(')
    expect(source).not.toContain('BindNodeDialog')
    expect(source).not.toContain('autoBoundHint')
  })

  it('output card binds via popover slots with single-output auto note', () => {
    const source = read(FIELD_CARDS)
    expect(source).toContain("data-testid='output-field-card'")
    expect(source).toContain("mode='output'")
    expect(source).toContain('outputKindFor(')
    expect(source).toContain('cases.singleOutputAuto')
    expect(source).toContain("data-testid='output-fields-table'")
    expect(source).toContain("data-testid='input-fields-table'")
  })
})

describe('unified workflow editor assembly', () => {
  it('workflow-editor renders step sections and locks behind an info alert', () => {
    const source = read(WORKFLOW_EDITOR)
    expect(source).toContain('WorkflowImportSection')
    expect(source).toContain('EditableInputFields')
    expect(source).toContain('EditableOutputFields')
    expect(source).toContain('TaskFlowTable')
    expect(source).toContain('validateRouting(')
    expect(source).toContain('cases.processingHeading')
    expect(source).toContain('case-section-processing')
    expect(source).toContain('useIsWide')
    expect(source).toContain('cases.inputsHeading')
    expect(source).toContain('cases.outputsHeading')
    expect(source).toContain('deriveBindings(')
    expect(source).toContain('deriveInputSchema(')
    expect(source).toContain('autoGenerateInputs')
    expect(source).toContain('autoGenerateOutputs')
    expect(source).not.toContain('PreviewSection')
    expect(source).not.toContain('AdvancedSection')
  })

  it('keeps the processing table de-duplicated and gated until workflow import', () => {
    const source = read(WORKFLOW_EDITOR)
    expect(source).toContain('showHeader={false}')
    expect(source).not.toContain("title={t('cases.processingHeading')}")
    expect(source).not.toContain("className='rounded-lg border-border/70'")
    expect(
      source.match(
        /AlertTitle>\{t\('cases\.emptyWorkflowLock'\)\}<\/AlertTitle>/g
      )
    ).toHaveLength(3)
  })

  it('locks workflow sections behind an info alert', () => {
    const source = read(WORKFLOW_EDITOR)
    expect(source).toMatch(/Alert variant='info'/)
    expect(source).toMatch(
      /AlertTitle>\{t\('cases\.emptyWorkflowLock'\)\}<\/AlertTitle>/
    )
  })

  it('case-form reuses the unified editor (thin wrapper, no duplicated sections)', () => {
    const source = read(CASE_FORM)
    expect(source).toContain('WorkflowEditor')
    expect(source).not.toContain('WorkflowImportSection')
    expect(source).not.toContain('InputFieldCard')
    expect(source).not.toContain('WorkflowJsonSection')
    expect(source).not.toContain('InputSchemaSection')
  })

  it('quick-config step1 reuses the unified editor', () => {
    const source = read(STEP1)
    expect(source).toContain('WorkflowEditor')
    expect(source).toContain('hideProcessing')
    expect(source).not.toContain("from './case-form'")
  })

  it('hideProcessing skips the processing section marker when set', () => {
    const source = read(WORKFLOW_EDITOR)
    expect(source).toContain('showWorkflow && !hideProcessing')
  })
})
