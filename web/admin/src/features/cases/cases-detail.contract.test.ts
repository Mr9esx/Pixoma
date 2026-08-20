import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const here = dirname(fileURLToPath(import.meta.url))
const DETAIL = join(here, 'detail-panel.tsx')
const CASE_FORM = join(here, 'case-form.tsx')

describe('workflow detail panel', () => {
  it('is view-only with config and entries sections and edit modal', () => {
    const source = readFileSync(DETAIL, 'utf8')
    expect(source).toContain('SectionHead')
    expect(source).toContain('cases.sectionConfig')
    expect(source).toContain('cases.sectionEntries')
    expect(source).toContain('readOnly')
    expect(source).toContain('showBasics={false}')
    expect(source).toContain('<Dialog')
    expect(source).toContain('DialogContent')
    expect(source).toContain('<CaseForm')
    expect(source).toContain('setEditOpen(false)')
  })

  it('case form can hide basics while keeping config sections', () => {
    const source = readFileSync(CASE_FORM, 'utf8')
    expect(source).toContain('showBasics')
    expect(source).toContain('BasicsSection')
    expect(source).toContain('WorkflowImportSection')
    expect(source).toContain('PreviewSection')
  })
})
