import { useRef, useState } from 'react'
import { ArrowRight, FileJson } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import {
  Attachment,
  AttachmentContent,
  AttachmentDescription,
  AttachmentMedia,
  AttachmentTitle,
  AttachmentTrigger,
} from '@/components/ui/attachment'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { CodeEditor } from '@/components/code-editor'
import { nodeLabel } from '../lib/node-catalog'
import type { WorkflowGraph, WorkflowNode } from '../lib/workflow-parse'

type Props = {
  value: string
  graph?: WorkflowGraph
  error?: string
  filename?: string
  onChange: (next: string) => void
  onFileName?: (name: string) => void
  disabled?: boolean
}

type ViewMode = 'diagram' | 'source'

/** 轻量节点流图示：横向节点卡 + 连线，超宽可横向滚动。 */
function NodeDiagram({ nodes }: { nodes: WorkflowNode[] }) {
  return (
    <div
      data-testid='workflow-import-diagram'
      className='overflow-auto max-h-[180px] bg-muted/20 p-3'
    >
      <ol className='flex min-w-max items-center gap-2'>
        {nodes.map((node, i) => (
          <li key={node.id} className='flex flex-none items-center gap-2'>
            <div className='w-40 rounded-md border border-border bg-card p-2.5'>
              <div className='truncate font-mono text-[11px] text-muted-foreground'>
                {node.id}
              </div>
              <div className='mt-0.5 truncate text-sm font-semibold'>
                {nodeLabel(node.class_type)}
              </div>
              <div className='mt-0.5 font-mono text-[11px] break-all text-muted-foreground'>
                {node.class_type}
              </div>
            </div>
            {i < nodes.length - 1 ? (
              <ArrowRight className='size-4 flex-none text-muted-foreground' />
            ) : null}
          </li>
        ))}
      </ol>
    </div>
  )
}

function ViewToggle({
  view,
  onChange,
}: {
  view: ViewMode
  onChange: (next: ViewMode) => void
}) {
  const { t } = useTranslation()
  return (
    <Tabs
      value={view}
      onValueChange={(next) => onChange(next as ViewMode)}
      aria-label={t('cases.jsonTitle')}
      className='shrink-0'
    >
      <TabsList className='h-7'>
        <TabsTrigger value='diagram' className='h-full px-2.5 text-xs'>
          {t('cases.viewDiagram')}
        </TabsTrigger>
        <TabsTrigger value='source' className='h-full px-2.5 text-xs'>
          {t('cases.viewSource')}
        </TabsTrigger>
      </TabsList>
    </Tabs>
  )
}

/** 图示 / 源码统一查看器：右上角切换。未导入时渲染为空。 */
export function WorkflowGraphViewer({
  value,
  graph,
  onChange,
  readOnly,
}: {
  value: string
  graph?: WorkflowGraph
  onChange: (next: string) => void
  readOnly?: boolean
}) {
  const { t } = useTranslation()
  const [view, setView] = useState<ViewMode>('diagram')
  if (!graph) return null
  return (
    <div
      data-testid='workflow-graph-viewer'
      className='overflow-hidden rounded-xl border border-border bg-card'
    >
      <header className='flex items-center justify-between gap-3 border-b border-border bg-muted/30 px-3 py-2'>
        <h4 className='min-w-0 truncate text-sm font-medium text-foreground'>
          {view === 'source' ? t('cases.jsonTitle') : t('cases.viewDiagram')}
        </h4>
        <ViewToggle view={view} onChange={setView} />
      </header>
      {view === 'source' ? (
        <CodeEditor
          title={undefined}
          className='rounded-none border-0'
          value={value}
          onChange={onChange}
          readOnly={readOnly}
          placeholder={t('cases.importDropHint')}
          maxHeight={180}
        />
      ) : (
        <NodeDiagram nodes={graph.nodes} />
      )}
    </div>
  )
}

export function WorkflowImportSection({
  value,
  graph,
  error,
  filename,
  onChange,
  onFileName,
  disabled,
}: Props) {
  const { t } = useTranslation()
  const fileRef = useRef<HTMLInputElement>(null)

  async function onFile(file: File | undefined) {
    if (!file) return
    onChange(await file.text())
    onFileName?.(file.name)
  }

  return (
    <section className='space-y-3' data-testid='case-section-workflow-import'>
      <div className='flex items-center gap-2'>
        <span className='flex size-5 shrink-0 items-center justify-center rounded-full bg-primary text-xs font-semibold text-primary-foreground'>
          1
        </span>
        <div className='min-w-0 flex-1'>
          <h3 className='text-sm font-semibold'>{t('cases.importHeading')}</h3>
          <p className='text-xs text-muted-foreground'>
            {t('cases.importHint')}
          </p>
        </div>
      </div>

      <div
        data-testid='workflow-import-dropzone'
        onDragOver={(e) => {
          if (!disabled) e.preventDefault()
        }}
        onDrop={(e) => {
          e.preventDefault()
          if (!disabled) void onFile(e.dataTransfer.files?.[0])
        }}
      >
        <Attachment
          state={error ? 'error' : value ? 'done' : 'idle'}
          className='w-full'
        >
          <AttachmentMedia variant='icon'>
            <FileJson />
          </AttachmentMedia>
          <AttachmentContent>
            <AttachmentTitle>
              {filename ||
                (value ? t('cases.importFile') : t('cases.importDropHint'))}
            </AttachmentTitle>
            {graph ? (
              <AttachmentDescription>
                {t('cases.importValid')} ·{' '}
                {t('cases.importNodesCount', {
                  count: graph.nodes.length,
                  inputs: graph.nodes.reduce(
                    (sum, node) => sum + node.inputs.length,
                    0
                  ),
                  outputs: graph.nodes.reduce(
                    (sum, node) => sum + node.outputCount,
                    0
                  ),
                })}
              </AttachmentDescription>
            ) : null}
          </AttachmentContent>
          {!disabled && (
            <AttachmentTrigger onClick={() => fileRef.current?.click()} />
          )}
        </Attachment>
        <input
          ref={fileRef}
          type='file'
          accept='.json,application/json'
          className='hidden'
          disabled={disabled}
          onChange={(e) => {
            void onFile(e.target.files?.[0])
            e.target.value = ''
          }}
        />
      </div>

      <WorkflowGraphViewer
        value={value}
        graph={graph}
        onChange={onChange}
        readOnly={disabled}
      />
      {error ? (
        <div
          data-testid='workflow-import-error'
          className='rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive'
        >
          <p className='font-medium'>{t('cases.importFailed')}</p>
          <p>{error}</p>
        </div>
      ) : null}
    </section>
  )
}
