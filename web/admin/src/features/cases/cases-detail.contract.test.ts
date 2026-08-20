import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const here = dirname(fileURLToPath(import.meta.url))
const DETAIL = join(here, 'detail-panel.tsx')

describe('workflow detail panel', () => {
  it('is view-only with config and entries sections and edit modal', () => {
    const source = readFileSync(DETAIL, 'utf8')
    expect(source).toContain('SectionHead')
    expect(source).toContain('cases.sectionConfig')
    expect(source).toContain('cases.sectionEntries')
    expect(source).toContain('readOnly')
    expect(source).toContain('<Dialog')
    expect(source).toContain('DialogContent')
    expect(source).toContain('<CaseForm')
    expect(source).toContain('setEditOpen(false)')
  })
})
