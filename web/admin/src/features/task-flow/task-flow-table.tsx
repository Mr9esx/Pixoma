import { useMemo, useState, type ReactNode } from 'react'
import {
  ArrowDown,
  ArrowUp,
  CircleAlert,
  CircleCheck,
  Pencil,
  Plus,
  Trash2,
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { ConditionForm } from './condition-form'
import { describeCondition, moveRule, removeRule } from './lib/rule-operations'
import { topicBindings } from './lib/topic-binding'
import { validateRouting } from './lib/validate'
import type {
  AttributeDescriptor,
  Condition,
  EdgePresence,
  EdgeRecord,
  RoutingConfig,
  TopicRecord,
} from './types'

type TaskFlowTableProps = {
  routing: RoutingConfig | undefined
  topics: TopicRecord[]
  attributes: AttributeDescriptor[]
  edges: EdgeRecord[]
  presence: EdgePresence[]
  onChange: (next: RoutingConfig) => void
  title?: string
  headerActions?: ReactNode
  showHeader?: boolean
  hideActions?: boolean
  readOnly?: boolean
  preview?: boolean
  className?: string
}

export function TaskFlowTable({
  routing,
  topics,
  attributes,
  edges,
  presence,
  onChange,
  title = '任务分流规则',
  headerActions,
  showHeader = true,
  hideActions = false,
  readOnly = false,
  preview = false,
  className,
}: TaskFlowTableProps) {
  const [editingIndex, setEditingIndex] = useState<number | null>(null)
  const effectiveReadOnly = readOnly || preview
  const showActions = !effectiveReadOnly && !hideActions
  const rules = routing?.rules ?? []

  const bindings = useMemo(
    () => topicBindings(edges, presence),
    [edges, presence]
  )
  const bindingByTopic = useMemo(() => {
    const map = new Map<string, (typeof bindings)[number]>()
    for (const binding of bindings) map.set(binding.topic, binding)
    return map
  }, [bindings])
  const validation = useMemo(() => {
    const boundTopicKeys = new Set(
      bindings.filter((b) => b.status !== 'unbound').map((b) => b.topic)
    )
    return validateRouting(routing, topics, attributes, boundTopicKeys)
  }, [bindings, routing, topics, attributes])
  const issueByIndex = useMemo(
    () => new Map(validation.issues.map((issue) => [issue.index, issue])),
    [validation]
  )

  const topicName = (key: string) =>
    topics.find((topic) => topic.key === key)?.name ?? key
  const bindingText = (topic: string | undefined) => {
    if (!topic) return '未选择'
    const binding = bindingByTopic.get(topic)
    if (binding?.status === 'ready')
      return `${binding.onlineEdges.length} 台在线`
    if (binding?.status === 'bound-offline') return '已绑定，无在线 agent'
    return '未绑定'
  }

  const updateRule = (index: number, topic: string | undefined) => {
    onChange({
      rules: rules.map((rule, i) => (i === index ? { ...rule, topic } : rule)),
    })
  }

  const editingRule = editingIndex == null ? undefined : rules[editingIndex]
  const routingEmpty = validation.issues.some(
    (issue) => issue.kind === 'routing-empty'
  )

  return (
    <div data-routing-table className={cn('flex flex-col gap-3', className)}>
      {showHeader && !preview ? (
        <div className='flex h-14 shrink-0 items-center justify-between gap-3 border-b border-border px-4'>
          <div className='flex min-w-0 items-center gap-3'>
            <h2 className='truncate text-lg font-semibold tracking-tight'>
              {title}
            </h2>
            <Badge
              variant='outline'
              data-routing-status
              className={cn(
                'shrink-0 gap-1.5 px-2.5 py-1.5 text-xs',
                validation.valid
                  ? 'border-success/40 bg-success/10 text-success'
                  : 'border-destructive/50 bg-destructive/10 text-destructive'
              )}
            >
              {validation.valid ? (
                <CircleCheck className='size-3.5' />
              ) : (
                <CircleAlert className='size-3.5' />
              )}
              {validation.valid
                ? '校验通过，可保存'
                : routingEmpty
                  ? '路由规则不能为空'
                  : `${validation.issues.length} 条规则未通过校验`}
            </Badge>
          </div>
          <div className='flex shrink-0 items-center gap-2'>
            {headerActions}
          </div>
        </div>
      ) : null}

      <div className='overflow-hidden rounded-md border'>
        <Table
          className='w-full table-fixed'
          wrapperClassName='max-h-[300px] overflow-y-auto'
        >
          <TableHeader className='sticky top-0 z-10 bg-background [&_th]:bg-background'>
            <TableRow>
              <TableHead className='w-12'>顺序</TableHead>
              <TableHead className='min-w-0'>条件</TableHead>
              <TableHead className='w-56'>目标 Topic</TableHead>
              {showActions ? (
                <TableHead className='w-36 text-right'>操作</TableHead>
              ) : null}
            </TableRow>
          </TableHeader>
          <TableBody>
            {rules.map((rule, index) => {
              const issue = issueByIndex.get(index)
              return (
                <TableRow key={index} data-rule-row={index}>
                  <TableCell className='text-muted-foreground'>
                    {index + 1}
                  </TableCell>
                  <TableCell>
                    <div className='max-w-[420px]'>
                      <div
                        className={cn(
                          'truncate',
                          issue ? 'text-destructive' : 'text-foreground'
                        )}
                      >
                        {describeCondition(rule.when)}
                      </div>
                      {issue ? (
                        <div className='truncate text-xs text-destructive'>
                          {issue.message}
                        </div>
                      ) : null}
                    </div>
                  </TableCell>
                  <TableCell>
                    {effectiveReadOnly ? (
                      <div className='min-w-0'>
                        <div className='truncate text-sm font-medium'>
                          {rule.topic ? topicName(rule.topic) : '未选择'}
                        </div>
                        <div className='truncate text-xs text-muted-foreground'>
                          {bindingText(rule.topic)}
                        </div>
                      </div>
                    ) : (
                      <Select
                        value={rule.topic}
                        onValueChange={(topic) => updateRule(index, topic)}
                      >
                        <SelectTrigger
                          size='sm'
                          className='h-8 w-48'
                          aria-label={`规则 ${index + 1} 目标 Topic`}
                        >
                          <SelectValue placeholder='选择 Topic' />
                        </SelectTrigger>
                        <SelectContent>
                          {topics
                            .filter((topic) => topic.enabled)
                            .map((topic) => (
                              <SelectItem key={topic.key} value={topic.key}>
                                {topic.name}
                              </SelectItem>
                            ))}
                        </SelectContent>
                      </Select>
                    )}
                  </TableCell>
                  {showActions ? (
                    <TableCell className='text-end'>
                      <div className='flex items-center justify-end gap-0.5'>
                        <Button
                          type='button'
                          variant='ghost'
                          size='icon-xs'
                          aria-label='上移'
                          disabled={effectiveReadOnly || index === 0}
                          onClick={() =>
                            onChange(moveRule({ rules }, index, -1))
                          }
                        >
                          <ArrowUp className='size-3.5' />
                        </Button>
                        <Button
                          type='button'
                          variant='ghost'
                          size='icon-xs'
                          aria-label='下移'
                          disabled={
                            effectiveReadOnly || index === rules.length - 1
                          }
                          onClick={() =>
                            onChange(moveRule({ rules }, index, 1))
                          }
                        >
                          <ArrowDown className='size-3.5' />
                        </Button>
                        <Button
                          type='button'
                          variant='ghost'
                          size='icon-xs'
                          aria-label='编辑条件'
                          disabled={effectiveReadOnly}
                          onClick={() => setEditingIndex(index)}
                        >
                          <Pencil className='size-3.5' />
                        </Button>
                        <Button
                          type='button'
                          variant='ghost'
                          size='icon-xs'
                          className='text-destructive'
                          aria-label='删除规则'
                          disabled={effectiveReadOnly}
                          onClick={() => onChange(removeRule({ rules }, index))}
                        >
                          <Trash2 className='size-3.5' />
                        </Button>
                      </div>
                    </TableCell>
                  ) : null}
                </TableRow>
              )
            })}
          </TableBody>
        </Table>
      </div>

      {showHeader && preview && headerActions ? (
        <div className='flex justify-end'>{headerActions}</div>
      ) : null}

      {!effectiveReadOnly ? (
        <Button
          type='button'
          size='sm'
          variant='secondary'
          data-add-rule
          onClick={() =>
            onChange({
              rules: [
                ...rules,
                {
                  when: {
                    field: attributes[0]?.key ?? '',
                    op: 'exists',
                  },
                  topic: undefined,
                },
              ],
            })
          }
          className='gap-1.5'
        >
          <Plus className='size-3.5' />
          新增规则
        </Button>
      ) : null}

      <Dialog
        open={editingIndex != null}
        onOpenChange={(open) => {
          if (!open) setEditingIndex(null)
        }}
      >
        <DialogContent className='sm:max-w-2xl'>
          <DialogHeader>
            <DialogTitle>
              编辑条件 {editingIndex != null ? `#${editingIndex + 1}` : ''}
            </DialogTitle>
            <DialogDescription>
              按顺序执行；第一条命中的规则决定目标 Topic。
            </DialogDescription>
          </DialogHeader>
          {editingRule ? (
            <ConditionForm
              value={editingRule.when}
              attributes={attributes}
              onChange={(when: Condition) =>
                editingIndex != null &&
                onChange({
                  rules: rules.map((rule, i) =>
                    i === editingIndex ? { ...rule, when } : rule
                  ),
                })
              }
            />
          ) : null}
          <DialogFooter>
            <Button type='button' onClick={() => setEditingIndex(null)}>
              完成
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
