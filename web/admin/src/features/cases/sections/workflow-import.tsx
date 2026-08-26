import { useEffect, useRef, useState } from 'react'
import { ArrowRight } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'
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
      className='overflow-x-auto rounded-md border border-border bg-muted/10 p-3'
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
              <div className='mt-0.5 break-all font-mono text-[11px] text-muted-foreground'>
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
  // 默认展示图示而非 JSON；仅刚成功导入（无图 → 有图）时切回图示。
  const [view, setView] = useState<ViewMode>('diagram')
  const hadGraph = useRef(false)

  useEffect(() => {
    if (graph && !hadGraph.current) setView('diagram')
    hadGraph.current = Boolean(graph)
  }, [graph])

  async function onFile(file: File | undefined) {
    if (!file) return
    onChange(await file.text())
    onFileName?.(file.name)
  }

  const sourceEditor = (
    <CodeEditor
      title={t('cases.jsonTitle')}
      aria-label={t('cases.jsonTitle')}
      value={value}
      onChange={onChange}
      readOnly={disabled}
      placeholder={t('cases.importDropHint')}
      maxHeight={250}
    />
  )

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
        <div
          role='group'
          aria-label={t('cases.jsonTitle')}
          className='flex shrink-0 items-center gap-1 rounded-md border bg-muted/40 p-0.5'
        >
          <button
            type='button'
            aria-pressed={view === 'diagram'}
            onClick={() => setView('diagram')}
            className={cn(
              'rounded px-2 py-1 text-xs font-medium transition-colors',
              view === 'diagram'
                ? 'bg-background text-foreground shadow-sm'
                : 'text-muted-foreground hover:text-foreground'
            )}
          >
            {t('cases.viewDiagram')}
          </button>
          <button
            type='button'
            aria-pressed={view === 'source'}
            onClick={() => setView('source')}
            className={cn(
              'rounded px-2 py-1 text-xs font-medium transition-colors',
              view === 'source'
                ? 'bg-background text-foreground shadow-sm'
                : 'text-muted-foreground hover:text-foreground'
            )}
          >
            {t('cases.viewSource')}
          </button>
        </div>
      </div>

      <div
        role='button'
        tabIndex={0}
        className='cursor-pointer rounded-md border border-dashed p-4 text-center text-sm text-muted-foreground hover:bg-accent/50'
        onDragOver={(e) => e.preventDefault()}
        onDrop={(e) => {
          e.preventDefault()
          if (!disabled) void onFile(e.dataTransfer.files[0])
        }}
        onClick={() => {
          if (!disabled) fileRef.current?.click()
        }}
        onKeyDown={(e) => {
          if (disabled) return
          if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault()
            fileRef.current?.click()
          }
        }}
      >
        <p>{t('cases.importDropHint')}</p>
        {filename ? (
          <p className='mt-1 text-xs text-emerald-600 dark:text-emerald-400'>
            {t('cases.importFile')}: {filename}
          </p>
        ) : null}
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

      {graph ? (
        view === 'source' ? (
          sourceEditor
        ) : (
          <NodeDiagram nodes={graph.nodes} />
        )
      ) : null}

      {graph ? (
        <div
          data-testid='workflow-import-valid'
          className='rounded-md border border-emerald-600/30 bg-emerald-50 px-3 py-2 text-sm text-emerald-700 dark:bg-emerald-900/20 dark:text-emerald-400'
        >
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
        </div>
      ) : null}

      {error ? (
        <div
          data-testid='workflow-import-error'
          className='rounded-md border border-red-600/30 bg-red-50 px-3 py-2 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400'
        >
          <p className='font-medium'>{t('cases.importFailed')}</p>
          <p>{error}</p>
        </div>
      ) : null}
    </section>
  )
}
