import { useRef, useState } from 'react'
import {
  ArrowRight,
  CheckCircle2,
  ChevronDownIcon,
  FileJson,
  TriangleAlert,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import {
  Attachment,
  AttachmentContent,
  AttachmentMedia,
  AttachmentTitle,
  AttachmentTrigger,
} from '@/components/ui/attachment'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { CodeEditor } from '@/components/code-editor'
import { nodeTitle, type InputKind } from '../lib/node-catalog'
import type { WorkflowGraph, WorkflowNode } from '../lib/workflow-parse'

type Props = {
  value: string
  graph?: WorkflowGraph
  error?: string
  filename?: string
  onChange: (next: string) => void
  onFileName?: (name: string) => void
  disabled?: boolean
  stepRail?: boolean
}

type ViewMode = 'diagram' | 'source'

/** 轻量节点流图示：横向节点卡 + 连线，超宽可横向滚动。 */
function NodeDiagram({ nodes }: { nodes: WorkflowNode[] }) {
  return (
    <div
      data-testid='workflow-import-diagram'
      className='max-h-[340px] overflow-auto bg-muted/20 px-3 pt-3 pb-7'
    >
      <ol className='flex min-w-max items-center gap-2'>
        {nodes.map((node, i) => (
          <li key={node.id} className='flex flex-none items-center gap-2'>
            <NodeCard node={node} />
            {i < nodes.length - 1 ? (
              <ArrowRight className='size-4 flex-none text-muted-foreground' />
            ) : null}
          </li>
        ))}
      </ol>
    </div>
  )
}

const INPUT_TYPE_LABELS: Record<InputKind, string> = {
  string: 'string',
  number: 'number',
  boolean: 'boolean',
  enum: 'enum',
  image: 'image',
  audio: 'audio',
  video: 'video',
  ref: 'ref',
  unknown: 'unknown',
}

/** 节点的可展示信息都放进 c-card-13 折叠卡：标题、ID、输出数、已连输入/总输入、类名、输入项。 */
function NodeCard({ node }: { node: WorkflowNode }) {
  const [isOpen, setIsOpen] = useState(false)
  const title = nodeTitle(node)

  return (
    <div className='relative'>
      <Card className='relative w-52 !gap-0 overflow-hidden !px-0 !py-0'>
        <CardHeader className='block border-b bg-muted/30 !px-3 !py-2'>
          <div className='flex items-start justify-between gap-2'>
            <div className='min-w-0'>
              <CardTitle className='truncate text-xs font-semibold text-foreground'>
                {title}
              </CardTitle>
              {node.class_type !== title ? (
                <p className='mt-1 truncate font-mono text-xs text-muted-foreground'>
                  {node.class_type}
                </p>
              ) : null}
            </div>
            <span className='font-mono text-xs text-muted-foreground'>
              {node.id}
            </span>
          </div>
        </CardHeader>
        <CardContent
          className={cn(
            'relative flex flex-col gap-2 overflow-hidden !px-3 !py-3 transition-all duration-500 ease-in-out',
            isOpen ? 'max-h-[300px]' : 'h-24'
          )}
        >
          <div className='flex justify-between rounded-lg bg-muted/60 px-2.5 py-1.5 text-xs text-muted-foreground'>
            <span>输出 · {node.outputCount}</span>
            <span>输入 · {node.inputs.length}</span>
          </div>
          {node.inputs.length ? (
            <ul className='flex flex-col gap-1.5'>
              {node.inputs.map((input) => (
                <li
                  key={input.name}
                  className='flex min-w-0 items-center justify-between gap-2 text-xs'
                >
                  <span className='min-w-0 truncate text-muted-foreground'>
                    {input.ref ? '↳ ' : ''}
                    {input.name}
                  </span>
                  <Badge
                    variant='outline'
                    className='shrink-0 rounded-full px-1.5 py-0 text-xs font-normal text-muted-foreground'
                  >
                    {INPUT_TYPE_LABELS[input.kind]}
                  </Badge>
                </li>
              ))}
            </ul>
          ) : null}
          <div
            className={cn(
              'pointer-events-none absolute inset-x-0 bottom-0 h-12 bg-linear-to-t from-card to-transparent transition-opacity duration-300',
              isOpen ? 'opacity-0' : 'opacity-100'
            )}
          />
        </CardContent>
      </Card>
      <div className='absolute -bottom-4 left-1/2 -translate-x-1/2'>
        <Button
          type='button'
          variant='outline'
          size='icon-sm'
          className='rounded-full bg-background shadow-sm hover:bg-background'
          onClick={() => setIsOpen(!isOpen)}
        >
          <ChevronDownIcon
            aria-hidden='true'
            className={cn(
              'transition-transform duration-300',
              isOpen && 'rotate-180'
            )}
          />
          <span className='sr-only'>展开/折叠</span>
        </Button>
      </div>
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

/** 节点图 / 源码统一查看器：右上角切换。未导入时渲染为空。 */
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
  stepRail,
}: Props) {
  const { t } = useTranslation()
  const fileRef = useRef<HTMLInputElement>(null)

  async function onFile(file: File | undefined) {
    if (!file) return
    onChange(await file.text())
    onFileName?.(file.name)
  }

  return (
    <section
      className={cn('relative flex flex-col gap-3', stepRail && 'pl-7')}
      data-testid='case-section-workflow-import'
    >
      <div className='flex items-center gap-2'>
        <span
          className={cn(
            'flex size-5 shrink-0 items-center justify-center rounded-full bg-primary text-xs font-semibold text-primary-foreground',
            stepRail && 'absolute top-0.5 left-0'
          )}
        >
          1
        </span>
        <div className='min-w-0 flex-1'>
          <h3 className='text-sm font-semibold'>{t('cases.importHeading')}</h3>
        </div>
      </div>

      {graph ? (
        <Alert
          variant='success'
          data-testid='workflow-import-status'
          className='px-3 py-2'
        >
          <CheckCircle2 aria-hidden='true' />
          <AlertTitle>{t('cases.importValid')}</AlertTitle>
          <AlertDescription>
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
          </AlertDescription>
        </Alert>
      ) : error ? (
        <Alert
          variant='destructive'
          data-testid='workflow-import-status'
          className='px-3 py-2'
        >
          <TriangleAlert aria-hidden='true' />
          <AlertTitle>{t('cases.importFailed')}</AlertTitle>
          <AlertDescription>
            {error.startsWith('cases.') ? t(error) : error}
          </AlertDescription>
        </Alert>
      ) : (
        <Alert
          variant='warn'
          data-testid='workflow-import-status'
          className='px-3 py-2'
        >
          <TriangleAlert aria-hidden='true' />
          <AlertTitle>{t('cases.workflowRequired')}</AlertTitle>
        </Alert>
      )}

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
    </section>
  )
}
