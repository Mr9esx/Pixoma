import { ArrowDown, ArrowUp, Cable, GripVertical, Trash2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import type { AttributeDescriptor, Condition, RoutingRule, TopicRecord } from './types'
import { ConditionForm } from './condition-form'

type Props = {
  rule: RoutingRule
  index: number
  total: number
  topics: TopicRecord[]
  attributes: AttributeDescriptor[]
  onConditionChange: (when: Condition) => void
  onMove: (dir: -1 | 1) => void
  onRemove: () => void
  /** 画布内编辑时显示拖拽手柄（整卡可拖动排序）。 */
  dragHandle?: boolean
}

export function RuleEditor({
  rule,
  index,
  total,
  topics,
  attributes,
  onConditionChange,
  onMove,
  onRemove,
  dragHandle = false,
}: Props) {
  const target = rule.topic ? topics.find((t) => t.key === rule.topic) : undefined
  return (
    <div className='flex flex-col'>
      <div className='flex items-center justify-between gap-2 px-4 py-2.5'>
        <div className='flex min-w-0 items-center gap-2'>
          {dragHandle && <GripVertical className='size-4 shrink-0 cursor-grab text-muted-foreground' />}
          <span className='truncate text-sm font-medium'>条件 #{index + 1}</span>
        </div>
        <div className='nodrag flex shrink-0 items-center gap-0.5'>
          <Button variant='ghost' size='icon' className='size-7 text-muted-foreground' disabled={index === 0} onClick={() => onMove(-1)} aria-label='上移'>
            <ArrowUp className='size-3.5' />
          </Button>
          <Button
            variant='ghost'
            size='icon'
            className='size-7 text-muted-foreground'
            disabled={index >= total - 1}
            onClick={() => onMove(1)}
            aria-label='下移'
          >
            <ArrowDown className='size-3.5' />
          </Button>
          <Button variant='ghost' size='icon' className='size-7 text-destructive' onClick={onRemove} aria-label='删除规则'>
            <Trash2 className='size-3.5' />
          </Button>
        </div>
      </div>

      <div className='border-t border-border px-4 py-3'>
        <div className='nodrag'>
          <ConditionForm value={rule.when} attributes={attributes} onChange={onConditionChange} />
        </div>
      </div>

      <div className='flex items-center gap-1.5 border-t border-border px-4 py-2.5'>
        <Cable className='size-3.5 shrink-0 text-muted-foreground' />
        {target ? (
          <span className='truncate text-xs text-muted-foreground'>
            已连接 → <span className='font-medium text-foreground'>{target.name}（{rule.topic}）</span>
          </span>
        ) : (
          <span className='text-xs text-muted-foreground'>未连接 · 从右侧圆点拖出连线到调度通道</span>
        )}
      </div>
    </div>
  )
}
