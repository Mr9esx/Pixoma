import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const here = dirname(fileURLToPath(import.meta.url))
const DETAIL = join(here, 'detail-panel.tsx')
const CASE_FORM = join(here, 'case-form.tsx')
const CONFIG_VIEW = join(here, 'sections/workflow-config-view.tsx')
const GRAPH = join(here, 'sections/workflow-graph-preview.tsx')
const PLACEMENTS = join(here, 'sections/menu-placements.tsx')

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
  })

  it('case form can hide basics while keeping config sections', () => {
    const source = readFileSync(CASE_FORM, 'utf8')
    expect(source).toContain('showBasics')
    expect(source).toContain('BasicsSection')
    expect(source).toContain('WorkflowImportSection')
    expect(source).toContain('PreviewSection')
  })

  it('config view shows filename and graph preview', () => {
    const source = readFileSync(CONFIG_VIEW, 'utf8')
    expect(source).toContain('cases.importFile')
    expect(source).toContain('WorkflowGraphPreview')
  })

  it('graph preview uses react flow with simple/full modes and highlights', () => {
    const source = readFileSync(GRAPH, 'utf8')
    expect(source).toContain("from '@xyflow/react'")
    expect(source).toContain('ReactFlow')
    expect(source).toContain('cases.graphSimple')
    expect(source).toContain('cases.graphFull')
    expect(source).toContain('inputBound')
    expect(source).toContain('outputBound')
  })

  it('placements section renders a table', () => {
    const source = readFileSync(PLACEMENTS, 'utf8')
    expect(source).toContain('<table')
    expect(source).toContain('cases.entriesColType')
    expect(source).toContain('cases.entriesColChannel')
  })
})
