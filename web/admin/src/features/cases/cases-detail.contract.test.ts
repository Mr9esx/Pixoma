import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const here = dirname(fileURLToPath(import.meta.url))
const DETAIL = join(here, 'detail-panel.tsx')
const CONFIG_VIEW = join(here, 'sections/workflow-config-view.tsx')

describe('workflow detail panel', () => {
  it('is view-only with config and entries sections and edit modal', () => {
    const source = readFileSync(DETAIL, 'utf8')
    expect(source).toContain('SectionHead')
    expect(source).toContain('cases.sectionConfig')
    expect(source).toContain('cases.sectionEntries')
    expect(source).toContain('WorkflowConfigView')
    expect(source).toContain('<Dialog')
    expect(source).toContain('DialogContent')
    expect(source).toContain('<CaseForm')
    expect(source).toContain('setEditOpen(false)')
    expect(source).not.toContain('BasicsSection')
  })

  it('config view shows inputs, outputs and bindings only (no basics)', () => {
    const source = readFileSync(CONFIG_VIEW, 'utf8')
    expect(source).toContain('cases.sectionInputs')
    expect(source).toContain('cases.sectionOutputs')
    expect(source).toContain('cases.sectionBindingInputs')
    expect(source).toContain('cases.sectionBindingOutputs')
    expect(source).not.toContain('BasicsSection')
    expect(source).not.toContain('cases.fieldPrice')
  })
})
