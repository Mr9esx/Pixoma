import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const here = dirname(fileURLToPath(import.meta.url))
const read = (path: string) => readFileSync(join(here, path), 'utf8')

const TABLE = read('task-flow-table.tsx')
const CONDITION_FORM = read('condition-form.tsx')
const CASE_CONTEXT = read('../config-context/case-context-section.tsx')
const VISUAL_CONFIG = read('../visual-config/visual-config-page.tsx')

describe('task flow table', () => {
  it('uses an ordered rule table instead of the canvas editor', () => {
    expect(TABLE).toContain('<Table')
    expect(TABLE).toContain('TableBody')
    expect(TABLE).toContain(
      "className='sticky top-0 z-10 bg-background [&_th]:bg-background'"
    )
    expect(TABLE).toContain("className='w-full table-fixed'")
    expect(TABLE).toContain("wrapperClassName='max-h-[300px] overflow-y-auto'")
    expect(TABLE).toContain('data-rule-row')
    expect(TABLE).toContain('data-add-rule')
    expect(TABLE).toContain("variant='secondary'")
    expect(TABLE).not.toContain('justify-between gap-3 border-t')
    expect(TABLE).not.toContain(
      'flex h-12 shrink-0 items-center justify-end border-b border-border px-4'
    )
    expect(TABLE).toContain(
      "<div className='flex justify-end'>{headerActions}</div>"
    )
    expect(TABLE).toContain('validateRouting')
    expect(TABLE).toContain('ConditionForm')
    expect(CONDITION_FORM).toContain('data-condition-unconditional')
    expect(TABLE).not.toContain('@xyflow/react')
    expect(TABLE).not.toContain('TaskFlowCanvas')
    expect(TABLE).not.toContain('TaskFlowEditor')
  })

  it('real routing entry points use the table and expose save actions', () => {
    expect(CASE_CONTEXT).toContain('TaskFlowTable')
    expect(CASE_CONTEXT).toContain('hideActions')
    expect(CASE_CONTEXT).toContain("variant='outline'")
    expect(CASE_CONTEXT).toContain('data-case-routing-save')
    expect(CASE_CONTEXT).not.toContain('TaskFlowEditor')
    expect(VISUAL_CONFIG).toContain('TaskFlowTable')
    expect(VISUAL_CONFIG).toContain('data-visual-config-save')
    expect(VISUAL_CONFIG).not.toContain('TaskFlowEditor')
    expect(VISUAL_CONFIG).not.toContain('画布')
  })

  it('keeps first-match routing and explicit routing validation', () => {
    const OPERATIONS = read('lib/rule-operations.ts')
    const VALIDATE = read('lib/validate.ts')
    expect(OPERATIONS).toContain('moveRule')
    expect(OPERATIONS).toContain('removeRule')
    expect(VALIDATE).toContain('topic-missing')
    expect(VALIDATE).toContain('routing-empty')
  })

  it('lets unconditional rules switch to conditional editing', () => {
    expect(CONDITION_FORM).toContain('Switch')
    expect(CONDITION_FORM).toContain("always' in value")
    expect(CONDITION_FORM).toContain('无条件')
  })
})
