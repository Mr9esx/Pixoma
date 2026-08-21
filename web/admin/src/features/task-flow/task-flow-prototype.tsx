import { useMemo, useState } from 'react'
import { Eye, EyeOff, PanelLeftClose, PanelLeftOpen } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import { DEFAULT_TOPIC_KEY, type AttributeDescriptor, type RoutingConfig, type TopicRecord } from './types'
import { mockAttributes, mockCases, mockEdges, mockPresence, mockTopics } from './mock-data'
import { TaskFlowCanvas } from './task-flow-canvas'

function JsonBlock({ title, value }: { title: string; value: unknown }) {
  return (
    <details className='group rounded-md border border-border bg-background'>
      <summary className='cursor-pointer px-3 py-2 text-xs font-medium text-muted-foreground'>
        {title}
      </summary>
      <pre className='overflow-auto border-t border-border p-3 text-[11px] leading-relaxed text-foreground'>
        {JSON.stringify(value, null, 2)}
      </pre>
    </details>
  )
}

/** 任务分流编辑器：Case 列表与画布融合为一个整体，点击左侧 Case 切换当前编辑的流程。 */
export function TaskFlowPrototype() {
  const [routing, setRouting] = useState<RoutingConfig | undefined>(mockCases[0].routing)
  const [readOnly, setReadOnly] = useState(false)
  const [caseId, setCaseId] = useState(mockCases[0].id)
  const [railOpen, setRailOpen] = useState(true)

  const topics: TopicRecord[] = mockTopics
  const attributes: AttributeDescriptor[] = mockAttributes.attributes
  const rules = useMemo(() => routing?.rules ?? [], [routing])
  const currentCase = mockCases.find((c) => c.id === caseId) ?? mockCases[0]

  function loadCase(id: number) {
    const next = mockCases.find((c) => c.id === id)
    if (!next || readOnly) return
    setCaseId(id)
    setRouting(next.routing)
  }

  return (
    <div className='mx-auto flex w-full max-w-[1440px] flex-col gap-4 px-6 py-6 md:px-8'>
      <div className='overflow-hidden rounded-xl border border-border bg-background shadow-sm'>
        {/* 顶栏：整体编辑器头部 */}
        <div className='flex items-center justify-between gap-3 border-b border-border px-4 py-3'>
          <div className='flex min-w-0 items-center gap-3'>
            <h1 className='truncate text-lg leading-tight font-semibold tracking-tight'>任务分流编辑器</h1>
            <span className='hidden shrink-0 text-xs text-muted-foreground sm:inline'>
              当前 Case：{currentCase.name}（id {currentCase.id}）
            </span>
          </div>
          <Button variant='outline' size='sm' className='gap-1.5' onClick={() => setReadOnly((v) => !v)}>
            {readOnly ? <Eye className='size-3.5' /> : <EyeOff className='size-3.5' />}
            {readOnly ? '查看模式' : '编辑模式'}
          </Button>
        </div>

        <div className='flex'>
          {/* 左侧 Case 列表：点击切换，与画布同属一个编辑器 */}
          <aside
            className={cn(
              'flex shrink-0 flex-col border-r border-border bg-muted/20 transition-[width] duration-150',
              railOpen ? 'w-56' : 'w-12',
            )}
          >
            <div className='flex h-10 items-center justify-between px-3'>
              {railOpen && <span className='truncate text-xs font-medium text-muted-foreground'>Case 列表</span>}
              <Button
                variant='ghost'
                size='icon'
                className='size-7 shrink-0 text-muted-foreground'
                onClick={() => setRailOpen((v) => !v)}
                aria-label={railOpen ? '收起 Case 列表' : '展开 Case 列表'}
              >
                {railOpen ? <PanelLeftClose className='size-3.5' /> : <PanelLeftOpen className='size-3.5' />}
              </Button>
            </div>
            {railOpen && (
              <>
                <div className='flex flex-1 flex-col gap-1.5 overflow-y-auto px-2 pb-3'>
                  {mockCases.map((c) => {
                    const active = c.id === caseId
                    return (
                      <button
                        key={c.id}
                        type='button'
                        data-case-item={c.id}
                        disabled={readOnly}
                        onClick={() => loadCase(c.id)}
                        className={cn(
                          'flex w-full flex-col gap-0.5 rounded-md border bg-background px-3 py-2.5 text-left transition-colors',
                          active
                            ? 'border-primary/60 ring-1 ring-primary/30'
                            : 'border-border hover:bg-muted/40',
                          readOnly && 'cursor-default opacity-60',
                        )}
                      >
                        <span className='truncate text-sm font-medium'>{c.name}</span>
                        <span className='text-[11px] text-muted-foreground'>
                          #{c.id} · {c.routing?.rules.length ?? 0} 条规则
                        </span>
                      </button>
                    )
                  })}
                </div>
                <div className='border-t border-border p-3 text-[11px] leading-relaxed text-muted-foreground'>
                  点击切换 Case 流程；在画布内编辑当前 Case 的路由与连线。
                </div>
              </>
            )}
          </aside>

          {/* 画布主体 */}
          <div className='h-[calc(100svh-220px)] min-h-[520px] min-w-0 flex-1 bg-muted/10'>
            <TaskFlowCanvas
              routing={routing}
              topics={topics}
              attributes={attributes}
              edges={mockEdges}
              presence={mockPresence}
              defaultTopicKey={DEFAULT_TOPIC_KEY}
              readOnly={readOnly}
              onChange={setRouting}
              caseName={currentCase.name}
            />
          </div>
        </div>

        {/* 底部数据预览：折叠在编辑器内 */}
        <details className='group border-t border-border'>
          <summary className='cursor-pointer select-none px-4 py-2 text-xs font-medium text-muted-foreground'>
            数据示例（与 API 逐字段一致）
          </summary>
          <div className='grid gap-3 border-t border-border p-4 md:grid-cols-2 xl:grid-cols-4'>
            <JsonBlock title='routing（保存载荷，实时生成）' value={{ rules }} />
            <JsonBlock title='GET /api/v1/topics' value={mockTopics} />
            <JsonBlock title='GET /api/v1/routing/attributes' value={mockAttributes} />
            <JsonBlock title='计算节点绑定 + 心跳' value={{ edges: mockEdges, presence: mockPresence }} />
          </div>
        </details>
      </div>
    </div>
  )
}
