import { useMemo } from 'react'
import { Plus, X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import type { AttributeDescriptor, Condition, ConditionOp } from './types'

type Props = {
  value: Condition
  attributes: AttributeDescriptor[]
  onChange: (next: Condition) => void
}

const OPS_BY_TYPE: Record<string, ConditionOp[]> = {
  boolean: ['eq', 'ne', 'exists'],
  string: ['eq', 'ne', 'in', 'exists'],
  number: ['eq', 'ne', 'gt', 'gte', 'lt', 'lte', 'exists'],
  array: ['in', 'exists'],
}

const OP_LABELS: Record<ConditionOp, string> = {
  eq: '等于',
  ne: '不等于',
  in: '属于',
  gt: '大于',
  gte: '大于等于',
  lt: '小于',
  lte: '小于等于',
  exists: '存在',
}

function attrFor(attributes: AttributeDescriptor[], field: string) {
  return attributes.find((a) => a.key === field)
}

function LeafRow({
  value,
  attributes,
  onChange,
  onRemove,
}: {
  value: Extract<Condition, { field: string }>
  attributes: AttributeDescriptor[]
  onChange: (next: Extract<Condition, { field: string }>) => void
  onRemove?: () => void
}) {
  const attr = attrFor(attributes, value.field)
  const type = attr?.schema.type ?? 'string'
  const ops = OPS_BY_TYPE[type] ?? OPS_BY_TYPE.string

  function patch(partial: Partial<Extract<Condition, { field: string }>>) {
    onChange({ ...value, ...partial })
  }

  return (
    <div className='grid grid-cols-[minmax(0,1fr)_7rem_minmax(0,1fr)_2rem] items-start gap-2 rounded-md border border-border bg-background p-2.5'>
      <Select
        value={value.field}
        onValueChange={(field) =>
          patch({
            field,
            op: OPS_BY_TYPE[attrFor(attributes, field)?.schema.type ?? 'string'][0],
            value: undefined,
          })
        }
      >
        <SelectTrigger size='sm' className='w-full'>
          <SelectValue placeholder='选择字段' />
        </SelectTrigger>
        <SelectContent>
          {attributes.map((a) => (
            <SelectItem key={a.key} value={a.key}>
              {a.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <Select value={value.op} onValueChange={(op) => patch({ op: op as ConditionOp })}>
        <SelectTrigger size='sm' className='w-full'>
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {ops.map((op) => (
            <SelectItem key={op} value={op}>
              {OP_LABELS[op]}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <div className='flex items-center gap-2'>
        <div className='min-w-0 flex-1'>
          <ValueControl attr={attr} op={value.op} value={value.value} onChange={(v) => patch({ value: v })} />
        </div>
        {onRemove && (
          <Button variant='ghost' size='icon' className='size-7 shrink-0 text-muted-foreground' onClick={onRemove} aria-label='删除条件'>
            <X className='size-3.5' />
          </Button>
        )}
      </div>
    </div>
  )
}

function ValueControl({
  attr,
  op,
  value,
  onChange,
}: {
  attr?: AttributeDescriptor
  op: ConditionOp
  value: unknown
  onChange: (v: unknown) => void
}) {
  if (op === 'exists') return <span className='self-center text-xs text-muted-foreground'>（仅判断是否有值）</span>
  const type = attr?.schema.type ?? 'string'
  if (type === 'boolean') {
    return (
      <Switch
        className='mt-1'
        checked={Boolean(value)}
        onCheckedChange={onChange}
        aria-label='布尔值'
      />
    )
  }
  if (type === 'string' && attr?.schema.enum?.length) {
    return (
      <Select value={String(value ?? '')} onValueChange={onChange}>
        <SelectTrigger size='sm' className='w-full'>
          <SelectValue placeholder='选择值' />
        </SelectTrigger>
        <SelectContent>
          {attr.schema.enum.map((opt) => (
            <SelectItem key={opt} value={opt}>
              {opt}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    )
  }
  if (type === 'array') {
    const text = Array.isArray(value) ? value.join(', ') : String(value ?? '')
    return (
      <Input
        className='w-full'
        placeholder='逗号分隔多个值'
        value={text}
        onChange={(e) =>
          onChange(
            e.target.value
              .split(',')
              .map((s) => s.trim())
              .filter(Boolean),
          )
        }
      />
    )
  }
  if (type === 'number') {
    return (
      <Input
        className='w-full'
        type='number'
        value={String(value ?? '')}
        onChange={(e) => onChange(e.target.value === '' ? undefined : Number(e.target.value))}
      />
    )
  }
  return (
    <Input className='w-full' value={String(value ?? '')} onChange={(e) => onChange(e.target.value)} />
  )
}

export function ConditionForm({ value, attributes, onChange }: Props) {
  const groupKind: 'and' | 'or' | null = 'and' in value ? 'and' : 'or' in value ? 'or' : null
  const leaves: Condition[] | null = groupKind
    ? groupKind === 'and'
      ? (value as { and: Condition[] }).and
      : (value as { or: Condition[] }).or
    : null

  const shape = useMemo(() => {
    if (leaves) return groupKind
    return 'leaf'
  }, [groupKind, leaves])

  function setShape(next: 'leaf' | 'and' | 'or') {
    if (next === 'leaf') {
      onChange({ field: attributes[0]?.key ?? '', op: 'eq', value: true })
      return
    }
    const items = leaves && leaves.length > 0 ? leaves : [{ field: attributes[0]?.key ?? '', op: 'eq' as ConditionOp, value: true }]
    onChange(next === 'and' ? { and: items } : { or: items })
  }

  function patchLeaf(index: number, next: Extract<Condition, { field: string }>) {
    if (!leaves) return
    const items: Condition[] = leaves.map((c, i) => (i === index ? next : c))
    onChange(groupKind === 'and' ? { and: items } : { or: items })
  }

  function removeLeaf(index: number) {
    if (!leaves) return
    const items: Condition[] = leaves.filter((_, i) => i !== index)
    onChange(items.length === 0 ? { and: [] } : groupKind === 'and' ? { and: items } : { or: items })
  }

  function appendLeaf() {
    const nextLeaf = {
      field: attributes[0]?.key ?? '',
      op: 'eq' as ConditionOp,
      value: true,
    }
    if (!leaves) {
      onChange({ and: [nextLeaf] })
      return
    }
    const items = [...leaves, nextLeaf]
    onChange(groupKind === 'or' ? { or: items } : { and: items })
  }

  return (
    <div className='space-y-3'>
      <div className='flex items-center gap-2'>
        <span className='text-xs text-muted-foreground'>条件组合</span>
        <Select value={shape ?? 'leaf'} onValueChange={(v) => setShape(v as 'leaf' | 'and' | 'or')}>
          <SelectTrigger size='sm' className='w-36'>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value='leaf'>单个条件</SelectItem>
            <SelectItem value='and'>同时满足（AND）</SelectItem>
            <SelectItem value='or'>满足任一（OR）</SelectItem>
          </SelectContent>
        </Select>
      </div>

      {shape === 'leaf' ? (
        <LeafRow
          value={value as Extract<Condition, { field: string }>}
          attributes={attributes}
          onChange={onChange}
        />
      ) : (
        <div className='space-y-2'>
          <div className='text-xs text-muted-foreground'>
            {shape === 'and' ? '以下条件全部满足时命中' : '以下条件任一满足时命中'}
          </div>
          {(leaves ?? []).map((leaf, i) => (
            <LeafRow
              key={i}
              value={leaf as Extract<Condition, { field: string }>}
              attributes={attributes}
              onChange={(next) => patchLeaf(i, next)}
              onRemove={() => removeLeaf(i)}
            />
          ))}
          <Button
            variant='outline'
            size='sm'
            className='gap-1.5'
            onClick={appendLeaf}
          >
            <Plus className='size-3.5' />
            添加子条件
          </Button>
        </div>
      )}
    </div>
  )
}
