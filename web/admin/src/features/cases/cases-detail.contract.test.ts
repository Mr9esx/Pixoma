import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const here = dirname(fileURLToPath(import.meta.url))
const DETAIL = join(here, 'detail-panel.tsx')
const CASE_FORM = join(here, 'case-form.tsx')
const CONFIG_VIEW = join(here, 'sections/workflow-config-view.tsx')
const GRAPH = join(here, 'sections/workflow-graph-preview.tsx')

describe('workflow detail panel', () => {
  it('is view-only with config section and edit modal; entries moved into status & relations', () => {
    const source = readFileSync(DETAIL, 'utf8')
    expect(source).toContain('SectionHead')
    expect(source).toContain('cases.sectionConfig')
    expect(source).not.toContain('cases.sectionEntries')
    expect(source).not.toContain('MenuPlacementsSection')
    expect(source).toContain('WorkflowConfigView')
    expect(source).toContain('cases.editInfo')
    expect(source).toContain('cases.editWorkflow')
    expect(source).toContain("setEditDialog('info')")
    expect(source).toContain('showWorkflow={false}')
    expect(source).toContain('showBasics={false}')
    expect(source).toContain('<Dialog')
    expect(source).toContain('DialogContent')
    expect(source).toContain('<CaseForm')
    expect(source).toContain('setEditDialog(null)')
  })

  it('config view receives onSaved that syncs the detail query cache', () => {
    const source = readFileSync(DETAIL, 'utf8')
    expect(source).toMatch(
      /onSaved=\{\(next\) =>\s*queryClient\.setQueryData\(queryKeys\.cases\.detail\(id\), next\)\s*\}/
    )
    expect(source).toContain('<WorkflowConfigView')
  })

  it('case form can hide basics while keeping config sections', () => {
    const source = readFileSync(CASE_FORM, 'utf8')
    expect(source).toContain('showBasics')
    expect(source).toContain('BasicsSection')
    expect(source).toContain('WorkflowImportSection')
    expect(source).not.toContain('PreviewSection')
    expect(source).not.toContain('previewSection')
  })

  it('create form keeps actions in a sticky bottom footer', () => {
    const source = readFileSync(CASE_FORM, 'utf8')
    expect(source).toMatch(/sticky bottom-0/)
    expect(source).toMatch(/border-t bg-card/)
    expect(source).toMatch(/t\('common\.create'\)/)
    expect(source).toMatch(/t\('common\.cancel'\)/)
  })

  it('config view shows filename and graph preview', () => {
    const source = readFileSync(CONFIG_VIEW, 'utf8')
    expect(source).toContain('cases.importFile')
    expect(source).toContain('WorkflowGraphPreview')
  })

  it('graph preview: io cards on top, horizontal process flow below', () => {
    const source = readFileSync(GRAPH, 'utf8')
    expect(source).toContain('cases.sectionInputs')
    expect(source).toContain('cases.sectionOutputs')
    expect(source).toContain('cases.sectionProcess')
    expect(source).toContain('cases.editInputs')
    expect(source).toContain('cases.editOutputs')
    expect(source).toContain('bindings.inputs')
    expect(source).toContain('bindings.outputs')
    expect(source).toContain('overflow-x-auto')
    expect(source).toContain('bg-muted/30')
    expect(source).toContain('border-b border-border px-3 py-2')
    expect(source).toContain("className='flex flex-col gap-2 px-3 py-2.5'")
    expect(source).toContain('TooltipContent')
    expect(source).toContain('font-mono text-sm')
    expect(source).toContain('w-56')
    expect(source).not.toContain('expandedNodes')
    expect(source).not.toContain('cases.nodeExpand')
  })

  it('graph preview modals edit fields and save via patchCase + onSaved', () => {
    const source = readFileSync(GRAPH, 'utf8')
    expect(source).toContain('InputFieldCard')
    expect(source).toContain('OutputFieldCard')
    expect(source).toContain('deriveBindings(inputDrafts, outputDrafts)')
    expect(source).toContain('patchCase(record.id, body)')
    expect(source).toContain(
      'queryClient.setQueryData(queryKeys.cases.detail(record.id), next)'
    )
    expect(source).toContain('onSaved?.(next)')
    expect(source).toContain('cases.saveSuccess')
    expect(source).toContain('cases.saveFailed')
    expect(source).not.toContain('bg-slate-')
  })

  it('detail panel offers delete workflow with confirmation', () => {
    const source = readFileSync(DETAIL, 'utf8')
    expect(source).toContain('cases.deleteWorkflow')
    expect(source).toContain('Trash2')
    expect(source).toContain('AlertDialog')
    expect(source).toContain('deleteCase(record.id, ackRefs)')
    expect(source).toContain('cases.deleteWillRemoveRefs')
    expect(source).toContain('cases.deleteWillFailTasks')
    expect(source).toContain('cases.deleteWillEndSessions')
    expect(source).toContain('cases.deleteAckRefs')
    expect(source).toContain('useCaseReferences')
    expect(readFileSync(join(here, '../config-context/use-case-references.ts'), 'utf8')).toContain('getCaseMenuPlacements')
    expect(source).not.toContain('disableCase(record.id)')
    expect(source).toContain('cases.deleteSuccess')
    expect(source).toContain('cases.deleteFailed')
  })
})
