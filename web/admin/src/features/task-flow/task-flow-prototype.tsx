import { useMemo, useState } from 'react'
import { CircleAlert, CircleCheck, Eye, EyeOff, GripVertical, Save } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { cn } from '@/lib/utils'
import { DEFAULT_TOPIC_KEY, type AttributeDescriptor, type RoutingConfig, type TopicRecord } from './types'
import { mockAttributes, mockCases, mockEdges, mockPresence, mockTopics } from './mock-data'
import { topicBindings } from './lib/topic-binding'
import { validateRouting } from './lib/validate'
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

/** 任务分流编辑器：顶栏 Case 下拉切换流程，右侧 Topic 池可拖入画布（孤立 Topic 待连线）。 */
export function TaskFlowPrototype() {
  const [routing, setRouting] = useState<RoutingConfig | undefined>(mockCases[0].routing)
  const [readOnly, setReadOnly] = useState(false)
  const [caseId, setCaseId] = useState(mockCases[0].id)
  const [savedAt, setSavedAt] = useState<string | null>(null)

  const topics: TopicRecord[] = mockTopics
  const attributes: AttributeDescriptor[] = mockAttributes.attributes
  const rules = useMemo(() => routing?.rules ?? [], [routing])
  const currentCase = mockCases.find((c) => c.id === caseId) ?? mockCases[0]
  const validation = useMemo(() => validateRouting(routing, topics, attributes), [routing, topics, attributes])
  const bindingByTopic = useMemo(() => {
    const map = new Map<string, (typeof bindings)[number]>()
    const bindings = topicBindings(mockEdges, mockPresence)
    for (const b of bindings) map.set(b.topic, b)
    return map
  }, [])

  function loadCase(id: number) {
    const next = mockCases.find((c) => c.id === id)
    if (!next || readOnly) return
    setCaseId(id)
    setRouting(next.routing)
    setSavedAt(null)
  }

  function handleSave() {
    if (!validation.valid) return
    setSavedAt(new Date().toLocaleTimeString('zh-CN', { hour12: false }))
  }

  return (
    <div className='mx-auto flex w-full max-w-[1600px] flex-col gap-4 px-6 py-6 md:px-8'>
      <div className='overflow-hidden rounded-xl border border-border bg-background shadow-sm'>
        <div className='flex'>
          {/* 左列：header（只覆盖画布）+ 画布主体 */}
          <div className='flex min-w-0 flex-1 flex-col'>
            {/* 顶栏：标题 + Case 下拉 + 校验/保存/模式切换；与右侧 Topic 池 head 同高水平对齐 */}
            <div className='flex h-14 items-center justify-between gap-3 border-b border-border px-4'>
          <div className='flex min-w-0 items-center gap-3'>
            <h1 className='truncate text-lg leading-tight font-semibold tracking-tight'>任务分流编辑器</h1>
            <Select
              value={String(caseId)}
              onValueChange={(v) => loadCase(Number(v))}
              disabled={readOnly}
            >
              <SelectTrigger size='sm' className='h-7 w-44 text-xs' aria-label='切换 Case'>
                <SelectValue placeholder='选择 Case' />
              </SelectTrigger>
              <SelectContent>
                {mockCases.map((c) => (
                  <SelectItem key={c.id} value={String(c.id)}>
                    {c.name}（#{c.id}）
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className='flex shrink-0 items-center gap-2'>
            <div
              data-save-status
              className={cn(
                'flex items-center gap-1.5 rounded-md border px-2.5 py-1.5 text-xs',
                validation.valid
                  ? 'border-emerald-500/40 bg-emerald-500/10 text-emerald-600'
                  : 'border-destructive/50 bg-destructive/10 text-destructive',
              )}
            >
              {validation.valid ? <CircleCheck className='size-3.5' /> : <CircleAlert className='size-3.5' />}
              <span>
                {validation.valid ? '校验通过，可保存' : `${validation.issues.length} 条规则未通过校验`}
              </span>
            </div>
            <Button
              variant='outline'
              size='sm'
              className='gap-1.5'
              disabled={!validation.valid || readOnly}
              onClick={handleSave}
              data-save-button
            >
              <Save className='size-3.5' />
              保存
              {savedAt && <span className='text-[10px] font-normal text-muted-foreground'>{savedAt}</span>}
            </Button>
            <Button variant='outline' size='sm' className='gap-1.5' onClick={() => setReadOnly((v) => !v)}>
              {readOnly ? <Eye className='size-3.5' /> : <EyeOff className='size-3.5' />}
              {readOnly ? '查看模式' : '编辑模式'}
            </Button>
          </div>
        </div>

          {/* 画布主体 */}
          <div className='h-[calc(100svh-220px)] min-h-[520px] bg-muted/10'>
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

          {/* 右侧 Topic 池：header 与左侧顶栏同高水平对齐；列出所有已启用 Topic，可拖入画布 */}
          <aside className='flex w-48 shrink-0 flex-col border-l border-border bg-muted/20'>
            <div className='flex h-14 items-center border-b border-border px-3 text-xs font-medium text-muted-foreground'>Topic 池</div>
            <div className='flex flex-1 flex-col gap-1.5 overflow-y-auto p-2'>
              {topics
                .filter((t) => t.enabled && t.key !== DEFAULT_TOPIC_KEY)
                .map((topic) => {
                  const binding = bindingByTopic.get(topic.key)
                  return (
                    <div
                      key={topic.key}
                      data-topic-pool-item={topic.key}
                      draggable={!readOnly}
                      onDragStart={(e) => {
                        e.dataTransfer.setData('application/pixoma-topic', topic.key)
                        e.dataTransfer.effectAllowed = 'move'
                      }}
                      className={cn(
                        'flex items-center gap-2 rounded-md border border-border bg-background px-2.5 py-2 text-xs transition-colors',
                        readOnly ? 'cursor-default opacity-60' : 'cursor-grab hover:border-primary/50 hover:bg-muted/40',
                      )}
                      title='拖入画布'
                    >
                      <GripVertical className='size-3.5 shrink-0 text-muted-foreground' />
                      <div className='min-w-0 flex-1'>
                        <div className='truncate font-medium'>{topic.name}</div>
                        <div className='truncate text-[10px] text-muted-foreground'>
                          {binding
                            ? binding.status === 'ready'
                              ? `${binding.onlineEdges.length} 台在线`
                              : binding.status === 'bound-offline'
                                ? '已绑定，无在线 agent'
                                : '未绑定'
                            : '未绑定'}
                        </div>
                      </div>
                      <span
                        className={cn(
                          'size-2 shrink-0 rounded-full',
                          binding?.status === 'ready'
                            ? 'bg-emerald-500'
                            : binding?.status === 'bound-offline'
                              ? 'bg-amber-500'
                              : 'bg-rose-400',
                        )}
                      />
                    </div>
                  )
                })}
            </div>
          </aside>
        </div>

        {/* 底部数据预览：折叠在编辑器内 */}
        <details className='group border-t border-border'>
          <summary className='cursor-pointer select-none px-4 py-2 text-xs font-medium text-muted-foreground'>
            数据示例（与 API 逐字段一致）
          </summary>
          <div className='grid gap-3 border-t border-border p-4 md:grid-cols-2 xl:grid-cols-4'>
            <JsonBlock title='routing（保存载荷，实时生成）' value={{ rules }} />
            <JsonBlock title='保存前校验' value={{ valid: validation.valid, issues: validation.issues }} />
            <JsonBlock title='GET /api/v1/topics' value={mockTopics} />
            <JsonBlock title='GET /api/v1/routing/attributes' value={mockAttributes} />
            <JsonBlock title='计算节点绑定 + 心跳' value={{ edges: mockEdges, presence: mockPresence }} />
          </div>
        </details>
      </div>
    </div>
  )
}
