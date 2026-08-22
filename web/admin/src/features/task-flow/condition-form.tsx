import { Plus, X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { cn } from '@/lib/utils'
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

type Leaf = Extract<Condition, { field: string }>

/**
 * 条件行（对齐蜂巢 .k-filter-item）：浅灰底、圆角 3px；
 * 固定 grid 列（字段 / 操作符 / 值 / 删除），保证多行严格对齐。
 */
function LeafRow({
  value,
  attributes,
  onChange,
  onRemove,
}: {
  value: Leaf
  attributes: AttributeDescriptor[]
  onChange: (next: Leaf) => void
  onRemove?: () => void
}) {
  const attr = attrFor(attributes, value.field)
  const type = attr?.schema.type ?? 'string'
  const ops = OPS_BY_TYPE[type] ?? OPS_BY_TYPE.string

  function patch(partial: Partial<Leaf>) {
    onChange({ ...value, ...partial })
  }

  return (
    <div
      className='grid grid-cols-[minmax(0,1fr)_5rem_minmax(0,1fr)_1.5rem] items-center gap-1.5 rounded-[3px] bg-[#f8f8f8] px-1.5 py-1.5'
      data-filter-item
    >
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
        <SelectTrigger size='sm' className='h-7 w-full border-0 bg-white text-xs shadow-none'>
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
        <SelectTrigger size='sm' className='h-7 w-full border-0 bg-white text-xs shadow-none'>
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
      <ValueControl attr={attr} op={value.op} value={value.value} onChange={(v) => patch({ value: v })} />
      {onRemove ? (
        <Button variant='ghost' size='icon' className='size-6 text-muted-foreground hover:text-destructive' onClick={onRemove} aria-label='删除条件'>
          <X className='size-3' />
        </Button>
      ) : (
        <span />
      )}
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
  if (op === 'exists') return <span className='self-center text-xs text-muted-foreground'>(仅判断是否有值)</span>
  const type = attr?.schema.type ?? 'string'
  if (type === 'boolean') {
    return (
      <div className='flex h-7 items-center'>
        <Switch checked={Boolean(value)} onCheckedChange={onChange} aria-label='布尔值' />
      </div>
    )
  }
  if (type === 'string' && attr?.schema.enum?.length) {
    return (
      <Select value={String(value ?? '')} onValueChange={onChange}>
        <SelectTrigger size='sm' className='h-7 w-full border-0 bg-white text-xs shadow-none'>
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
        className='h-7 w-full border-0 bg-white text-xs shadow-none'
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
        className='h-7 w-full border-0 bg-white text-xs shadow-none'
        type='number'
        value={String(value ?? '')}
        onChange={(e) => onChange(e.target.value === '' ? undefined : Number(e.target.value))}
      />
    )
  }
  return (
    <Input className='h-7 w-full border-0 bg-white text-xs shadow-none' value={String(value ?? '')} onChange={(e) => onChange(e.target.value)} />
  )
}

/** 连接柱上的「且/或」切换（对齐蜂巢 .k-filter-builder-condition-select：AND 淡紫、OR 淡黄胶囊）。 */
function GroupConditionSelect({ kind, onChange }: { kind: 'and' | 'or'; onChange: (next: 'and' | 'or') => void }) {
  return (
    <Select value={kind} onValueChange={(v) => onChange(v as 'and' | 'or')}>
      <SelectTrigger
        size='sm'
        className={cn(
          'h-5 w-12 shrink-0 justify-center rounded-full border px-0 text-[11px] shadow-none [&>svg]:hidden',
          kind === 'and'
            ? 'border-[#d8c9f0] bg-[#f9f6fe] text-violet-700'
            : 'border-[#fed58b] bg-[#feefd5] text-amber-800',
        )}
      >
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        <SelectItem value='and'>且</SelectItem>
        <SelectItem value='or'>或</SelectItem>
      </SelectContent>
    </Select>
  )
}

/**
 * 条件编辑（对齐蜂巢 k-filter-builder）：
 * - 单条件：组框内一行 + 「新增规则」；
 * - 多条件：左侧 40px 连接柱（上下 2px 彩线 + 中间「且/或」切换），右侧条件行列表；
 *   连接柱只包住条件行，「新增规则」按钮在柱线之外；
 * - 无顶部「条件组合」下拉：加第二条自动成组，删到一条自动回单条件。
 */
export function ConditionForm({ value, attributes, onChange }: Props) {
  const groupKind: 'and' | 'or' | null = 'and' in value ? 'and' : 'or' in value ? 'or' : null
  const leaves: Condition[] | null = groupKind
    ? groupKind === 'and'
      ? (value as { and: Condition[] }).and
      : (value as { or: Condition[] }).or
    : null

  const newLeaf = (): Leaf => ({
    field: attributes[0]?.key ?? '',
    op: 'eq',
    value: true,
  })

  function appendLeaf() {
    if (!leaves) {
      // 单条件 → 且组：保留当前条件，追加一条。
      onChange({ and: [value as Leaf, newLeaf()] })
      return
    }
    const items = [...leaves, newLeaf()]
    onChange(groupKind === 'or' ? { or: items } : { and: items })
  }

  function patchLeaf(index: number, next: Leaf) {
    if (!leaves) return
    const items: Condition[] = leaves.map((c, i) => (i === index ? next : c))
    onChange(groupKind === 'and' ? { and: items } : { or: items })
  }

  function removeLeaf(index: number) {
    if (!leaves) return
    const items: Condition[] = leaves.filter((_, i) => i !== index)
    if (items.length === 1) {
      // 删到只剩一条：回到单条件形态。
      onChange(items[0])
      return
    }
    onChange(items.length === 0 ? { and: [] } : groupKind === 'and' ? { and: items } : { or: items })
  }

  function setGroupKind(next: 'and' | 'or') {
    if (!leaves) return
    onChange(next === 'and' ? { and: leaves } : { or: leaves })
  }

  const lineColor = groupKind === 'or' ? '#fed58b' : '#d8c9f0'
  const showConnector = Boolean(leaves && leaves.length > 1)

  return (
    // .k-filter-group：细边框、圆角 4、padding 12
    <div className='rounded-[4px] border border-[#ecedef] p-3' data-filter-group>
      <div className='flex'>
        {showConnector && (
          // .k-filter-builder-condition-group-range：40px 连接柱，只包住条件行（不含底部按钮）。
          <div className='flex w-10 shrink-0 flex-col'>
            <div className='flex flex-1 items-stretch'>
              <div className='w-[calc(40%-2px)]' aria-hidden />
              <div
                className='w-[calc(60%+2px)] rounded-tl'
                style={{ borderTop: `2px solid ${lineColor}`, borderLeft: `2px solid ${lineColor}` }}
                aria-hidden
              />
            </div>
            <div className='flex items-center py-0.5'>
              <GroupConditionSelect kind={groupKind ?? 'and'} onChange={setGroupKind} />
            </div>
            <div className='flex flex-1 items-stretch'>
              <div className='w-[calc(40%-2px)]' aria-hidden />
              <div
                className='w-[calc(60%+2px)] rounded-bl'
                style={{ borderBottom: `2px solid ${lineColor}`, borderLeft: `2px solid ${lineColor}` }}
                aria-hidden
              />
            </div>
          </div>
        )}

        {/* 条件行列表 */}
        <div className='flex min-w-0 flex-1 flex-col gap-1.5'>
          {leaves ? (
            leaves.map((leaf, i) => (
              <LeafRow
                key={i}
                value={leaf as Leaf}
                attributes={attributes}
                onChange={(next) => patchLeaf(i, next)}
                onRemove={() => removeLeaf(i)}
              />
            ))
          ) : (
            <LeafRow value={value as Leaf} attributes={attributes} onChange={onChange} />
          )}
          {leaves && leaves.length === 0 && (
            <div className='rounded-[3px] bg-[#f8f8f8] px-3 py-2.5 text-xs text-muted-foreground'>
              暂无规则，请点击下方「新增规则」添加
            </div>
          )}
        </div>
      </div>

      {/* 新增规则：胶囊按钮，在连接柱线之外（对齐蜂巢底部按钮区）。 */}
      <Button
        variant='outline'
        size='sm'
        className={cn('mt-2 h-7 gap-1 rounded-full border-[#ecedef] text-xs', showConnector && 'ml-10')}
        onClick={appendLeaf}
      >
        <Plus className='size-3' />
        新增规则
      </Button>
    </div>
  )
}
