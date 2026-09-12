import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const here = dirname(fileURLToPath(import.meta.url))
const DETAIL = join(here, 'detail-panel.tsx')
const CASE_FORM = join(here, 'case-form.tsx')
const WORKFLOW_EDITOR = join(here, 'workflow-editor.tsx')
const CONFIG_VIEW = join(here, 'sections/workflow-config-view.tsx')
const GRAPH = join(here, 'sections/workflow-graph-preview.tsx')
const LIGHTBOX = join(here, 'components/media-lightbox.tsx')

describe('workflow detail panel', () => {
  it('media lightbox has an opaque themed surface and visible close icon', () => {
    const source = readFileSync(LIGHTBOX, 'utf8')
    expect(source).toContain('bg-foreground')
    expect(source).not.toContain('bg-foreground/95')
    expect(source).toContain("[&_[data-slot='dialog-close']]:text-background")
    expect(source).toContain(
      "[&_[data-slot='dialog-close']]:hover:bg-background/60"
    )
    expect(source).not.toContain("[&_[data-slot='dialog-close']]:bg-background")
    expect(source).not.toContain(
      "[&_[data-slot='dialog-close']]:text-foreground"
    )
    expect(source).not.toContain('&apos;')
  })

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

  it('workflow editor can hide basics while keeping config sections', () => {
    const source = readFileSync(WORKFLOW_EDITOR, 'utf8')
    expect(source).toContain('showBasics')
    expect(source).toContain('BasicsSection')
    expect(source).toContain('nameError={nameError}')
    expect(source).toContain('noValidate')
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

  it('config view shows the graph preview without a filename line', () => {
    const source = readFileSync(CONFIG_VIEW, 'utf8')
    expect(source).toContain('WorkflowGraphPreview')
    expect(source).not.toContain('cases.importFile')
  })

  it('graph preview: io aligned with editor, no process overview or vertical scroll trap', () => {
    const source = readFileSync(GRAPH, 'utf8')
    // 标题/文案对齐编辑工作流。
    expect(source).toContain('cases.inputsHeading')
    expect(source).toContain('cases.outputsHeading')
    expect(source).toContain('cases.editInputs')
    expect(source).toContain('cases.editOutputs')
    // 展示层复用编辑工作流的表格（只读）。
    expect(source).toContain('InputFieldsTable')
    expect(source).toContain('OutputFieldsTable')
    expect(source).toContain('onChange={noop}')
    expect(source).toContain('disabled')
    expect(source.match(/hideActions/g)).toHaveLength(2)
    // 工作流配置复用编辑工作流的节点图 / JSON 查看器。
    expect(source).toContain('WorkflowGraphViewer')
    expect(source).toContain('readOnly')
    expect(source).toContain('bindings.inputs')
    expect(source).toContain('bindings.outputs')
    // 不再包一层带标题/边框的容器（避免 border 套 border）。
    expect(source).not.toContain('cases.jsonTitle')
    expect(source).not.toContain('overflow-hidden rounded-lg border')
    // 已移除处理概览与其横向滚动容器（修复滚轮陷阱）。
    expect(source).not.toContain('cases.sectionProcess')
    expect(source).not.toContain('overflow-x-auto')
    expect(source).not.toContain('w-56')
    expect(source).not.toContain('TooltipContent')
    expect(source).not.toContain('expandedNodes')
    expect(source).not.toContain('cases.nodeExpand')
  })

  it('graph preview modals reuse editor field lists and save via patchCase + onSaved', () => {
    const source = readFileSync(GRAPH, 'utf8')
    expect(source).toContain('EditableInputFields')
    expect(source).toContain('EditableOutputFields')
    expect(source).toContain('useIsWide')
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
    expect(source).toContain('ConfirmDialog')
    expect(source).toContain('deleteCase(record.id, ackRefs)')
    expect(source).toContain('cases.deleteWillRemoveRefs')
    expect(source).toContain('cases.deleteWillFailTasks')
    expect(source).toContain('cases.deleteWillEndSessions')
    expect(source).toContain('cases.deleteAckRefs')
    expect(source).toContain('useCaseReferences')
    expect(
      readFileSync(
        join(here, '../config-context/use-case-references.ts'),
        'utf8'
      )
    ).toContain('getCaseMenuPlacements')
    expect(source).not.toContain('disableCase(record.id)')
    expect(source).toContain('cases.deleteSuccess')
    expect(source).toContain('cases.deleteFailed')
  })
})
