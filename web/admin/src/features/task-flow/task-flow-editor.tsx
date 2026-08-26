import { useMemo, type ReactNode } from 'react'
import { CircleAlert, CircleCheck, GripVertical } from 'lucide-react'
import { cn } from '@/lib/utils'
import { topicBindings } from './lib/topic-binding'
import { validateRouting } from './lib/validate'
import { TaskFlowCanvas } from './task-flow-canvas'
import {
  DEFAULT_TOPIC_KEY,
  type AttributeDescriptor,
  type EdgePresence,
  type EdgeRecord,
  type RoutingConfig,
  type TopicRecord,
} from './types'

export type TaskFlowEditorProps = {
  routing: RoutingConfig | undefined
  topics: TopicRecord[]
  attributes: AttributeDescriptor[]
  edges: EdgeRecord[]
  presence: EdgePresence[]
  caseName: string
  onChange: (next: RoutingConfig) => void
  /** 新增/移除「Topic→计算节点」绑定（add=true 绑定，false 解绑），由调用方持久化节点订阅。 */
  onChangeEdgeSubscription?: (
    edgeId: string,
    topicKey: string,
    add: boolean
  ) => void
  readOnly?: boolean
  /** 预览模式：只读画布，隐藏顶栏与右侧 Topic 池。 */
  preview?: boolean
  title?: string
  headerActions?: ReactNode
  className?: string
}

/**
 * 任务分流编辑器（复用原型页的顶栏校验 + 画布 + 右侧 Topic 池）。
 * 快速配置 Step2 与 task-flow-prototype 共用同一编辑器外壳。
 */
export function TaskFlowEditor({
  routing,
  topics,
  attributes,
  edges,
  presence,
  caseName,
  onChange,
  onChangeEdgeSubscription,
  readOnly = false,
  preview = false,
  title = '任务分流编辑器',
  headerActions,
  className,
}: TaskFlowEditorProps) {
  const effectiveReadOnly = readOnly || preview
  const validation = useMemo(() => {
    const bindings = topicBindings(edges, presence)
    const boundTopicKeys = new Set(
      bindings
        .filter((binding) => binding.status !== 'unbound')
        .map((binding) => binding.topic)
    )
    return validateRouting(routing, topics, attributes, boundTopicKeys)
  }, [routing, topics, attributes, edges, presence])
  const bindingByTopic = useMemo(() => {
    const map = new Map<string, (typeof bindings)[number]>()
    const bindings = topicBindings(edges, presence)
    for (const b of bindings) map.set(b.topic, b)
    return map
  }, [edges, presence])

  return (
    <div
      className={cn(
        'overflow-hidden rounded-xl border border-border bg-background shadow-sm',
        className
      )}
    >
      <div className='flex'>
        <div className='flex min-w-0 flex-1 flex-col'>
          {!preview ? (
            <div className='flex h-14 items-center justify-between gap-3 border-b border-border px-4'>
              <div className='flex min-w-0 items-center gap-3'>
                <h1 className='truncate text-lg leading-tight font-semibold tracking-tight'>
                  {title}
                </h1>
              </div>
              <div className='flex shrink-0 items-center gap-2'>
                <div
                  data-save-status
                  className={cn(
                    'flex items-center gap-1.5 rounded-md border px-2.5 py-1.5 text-xs',
                    validation.valid
                      ? 'border-emerald-500/40 bg-emerald-500/10 text-emerald-600'
                      : 'border-destructive/50 bg-destructive/10 text-destructive'
                  )}
                >
                  {validation.valid ? (
                    <CircleCheck className='size-3.5' />
                  ) : (
                    <CircleAlert className='size-3.5' />
                  )}
                  <span>
                    {validation.valid
                      ? '校验通过，可保存'
                      : `${validation.issues.length} 条规则未通过校验`}
                  </span>
                </div>
                {headerActions}
              </div>
            </div>
          ) : null}
          <div
            className={cn(
              'bg-muted/10',
              preview ? 'min-h-[420px]' : 'min-h-[520px]'
            )}
          >
            <TaskFlowCanvas
              routing={routing}
              topics={topics}
              attributes={attributes}
              edges={edges}
              presence={presence}
              defaultTopicKey={DEFAULT_TOPIC_KEY}
              readOnly={effectiveReadOnly}
              onChange={onChange}
              onChangeEdgeSubscription={onChangeEdgeSubscription}
              caseName={caseName}
            />
          </div>
        </div>

        {!preview ? (
          <aside className='flex w-48 shrink-0 flex-col border-l border-border bg-muted/20'>
            <div className='flex h-14 items-center border-b border-border px-3 text-xs font-medium text-muted-foreground'>
              调度通道
            </div>
            <div className='flex flex-1 flex-col gap-1.5 overflow-y-auto p-2'>
              {topics
                .filter((t) => t.enabled && t.key !== DEFAULT_TOPIC_KEY)
                .map((topic) => {
                  const binding = bindingByTopic.get(topic.key)
                  return (
                    <div
                      key={topic.key}
                      data-topic-pool-item={topic.key}
                      draggable={!effectiveReadOnly}
                      onDragStart={(e) => {
                        e.dataTransfer.setData(
                          'application/pixoma-topic',
                          topic.key
                        )
                        e.dataTransfer.effectAllowed = 'move'
                      }}
                      className={cn(
                        'flex items-center gap-2 rounded-md border border-border bg-background px-2.5 py-2 text-xs transition-colors',
                        effectiveReadOnly
                          ? 'cursor-default opacity-60'
                          : 'cursor-grab hover:border-primary/50 hover:bg-muted/40'
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
                              : 'bg-rose-400'
                        )}
                      />
                    </div>
                  )
                })}
            </div>
          </aside>
        ) : null}
      </div>
    </div>
  )
}
