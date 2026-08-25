import { useMemo, useState } from 'react'
import { Eye, EyeOff, Save } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { type AttributeDescriptor, type RoutingConfig, type TopicRecord } from './types'
import { mockAttributes, mockCases, mockEdges, mockPresence, mockTopics } from './mock-data'
import { validateRouting } from './lib/validate'
import { topicBindings } from './lib/topic-binding'
import { TaskFlowEditor } from './task-flow-editor'

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

/** 任务分流编辑器原型页：复用 TaskFlowEditor（顶栏校验 + 画布 + Topic 池），Mock 数据演示。 */
export function TaskFlowPrototype() {
  const [routing, setRouting] = useState<RoutingConfig | undefined>(mockCases[0].routing)
  const [readOnly, setReadOnly] = useState(false)
  const [caseId, setCaseId] = useState(mockCases[0].id)
  const [savedAt, setSavedAt] = useState<string | null>(null)

  const topics: TopicRecord[] = mockTopics
  const attributes: AttributeDescriptor[] = mockAttributes.attributes
  const rules = useMemo(() => routing?.rules ?? [], [routing])
  const currentCase = mockCases.find((c) => c.id === caseId) ?? mockCases[0]
  const validation = useMemo(
    () => {
      const bindings = topicBindings(mockEdges, mockPresence)
      const boundTopicKeys = new Set(
        bindings
          .filter((binding) => binding.status !== 'unbound')
          .map((binding) => binding.topic),
      )
      return validateRouting(routing, topics, attributes, boundTopicKeys)
    },
    [routing, topics, attributes],
  )

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
      <TaskFlowEditor
        routing={routing}
        topics={topics}
        attributes={attributes}
        edges={mockEdges}
        presence={mockPresence}
        caseName={currentCase.name}
        onChange={setRouting}
        readOnly={readOnly}
        headerActions={
          <>
            <Select value={String(caseId)} onValueChange={(v) => loadCase(Number(v))} disabled={readOnly}>
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
          </>
        }
      />

      <details className='group rounded-md border border-border bg-background'>
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
  )
}
