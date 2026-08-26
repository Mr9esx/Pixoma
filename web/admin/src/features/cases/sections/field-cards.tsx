import { useEffect, useMemo, useRef, useState } from 'react'
import { ChevronRight, ChevronsUpDown, Search, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/input'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import type { InputFieldDraft, OutputFieldDraft } from '../lib/derive'
import {
  inputKindFor,
  outputKindFor,
  nodeLabel,
  nodeVisualFor,
} from '../lib/node-catalog'
import type { WorkflowNode } from '../lib/workflow-parse'

const INPUT_TYPES: Array<{ value: string; labelKey: string }> = [
  { value: 'string', labelKey: 'cases.typeString' },
  { value: 'image', labelKey: 'cases.typeImage' },
  { value: 'audio', labelKey: 'cases.typeAudio' },
  { value: 'video', labelKey: 'cases.typeVideo' },
  { value: 'number', labelKey: 'cases.typeNumber' },
  { value: 'boolean', labelKey: 'cases.typeBoolean' },
  { value: 'enum', labelKey: 'cases.typeEnum' },
]
const OUTPUT_TYPES: Array<{ value: string; labelKey: string }> = [
  { value: 'image', labelKey: 'cases.typeImage' },
  { value: 'text', labelKey: 'cases.typeText' },
  { value: 'file', labelKey: 'cases.typeFile' },
]

/** 类型 value → i18n key；用于列表徽标等处显示中文类型名。 */
export const TYPE_LABEL_KEYS: Record<string, string> = {
  string: 'cases.typeString',
  image: 'cases.typeImage',
  audio: 'cases.typeAudio',
  video: 'cases.typeVideo',
  number: 'cases.typeNumber',
  boolean: 'cases.typeBoolean',
  enum: 'cases.typeEnum',
  text: 'cases.typeText',
  file: 'cases.typeFile',
}

/** inputKindFor 的兜底：查表未命中（unknown）时按 string 处理。 */
function safeAutoType(classType: string, fieldPath: string): string {
  const kind = inputKindFor(classType, fieldPath)
  return kind === 'unknown' ? 'string' : kind
}

// ===== 轻量绑定浮层（锚定当前行，点选即绑定，不放大弹窗）=====

type BindNodePopoverProps = {
  nodes: WorkflowNode[]
  /** input = 绑字面量参数；output = 绑节点输出槽位。 */
  mode?: 'input' | 'output'
  /** 当前已绑定的节点（用于触发器展示）。 */
  node: WorkflowNode | undefined
  /** 已绑定的参数/索引描述。 */
  boundLabel?: string
  onPick: (nodeId: string, pick: string) => void
  disabled?: boolean
  compact?: boolean
}

export function BindNodePopover({
  nodes,
  mode = 'input',
  node,
  boundLabel,
  onPick,
  disabled,
  compact,
}: BindNodePopoverProps) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const [search, setSearch] = useState('')
  const searchRef = useRef<HTMLInputElement | null>(null)
  const bound = !!node

  // 打开时聚焦搜索框；关闭时清空 query
  useEffect(() => {
    if (open) {
      const id = window.requestAnimationFrame(() => searchRef.current?.focus())
      return () => window.cancelAnimationFrame(id)
    }
    setSearch('')
    return undefined
  }, [open])

  const query = search.trim().toLowerCase()
  const nodeMatches = (n: WorkflowNode) =>
    !query ||
    nodeLabel(n.class_type).toLowerCase().includes(query) ||
    String(n.id).toLowerCase().includes(query)

  // 输入：只暴露字面量参数（引用/节点间连线不是用户能提供的输入）。
  const inputGroups = useMemo(
    () =>
      nodes
        .map((n) => ({
          node: n,
          params: n.inputs
            .filter((i) => !i.ref)
            .filter(
              (i) => !query || i.name.toLowerCase().includes(query)
            ),
        }))
        .filter((g) => g.params.length > 0 && nodeMatches(g.node)),
    // nodeMatches 依赖 query（已在 deps）
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [nodes, query]
  )
  // 输出：暴露各节点输出槽位。
  const outputGroups = useMemo(
    () => nodes.filter((n) => n.outputCount > 0 && nodeMatches(n)),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [nodes, query]
  )

  function pick(nodeId: string, pick: string) {
    onPick(nodeId, pick)
    setOpen(false)
  }

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          type='button'
          variant='outline'
          disabled={disabled || nodes.length === 0}
          className={
            compact
              ? 'flex h-8 w-full items-center justify-between gap-1.5 border border-input bg-transparent px-2 text-left text-xs hover:bg-accent/50'
              : 'flex h-auto w-full items-center justify-between gap-2 border border-input bg-transparent px-3 py-2 text-left hover:bg-accent/50'
          }
        >
          {bound ? (
            <span className='flex min-w-0 items-center gap-1.5'>
              {(() => {
                const { Icon, className } = nodeVisualFor(node.class_type)
                return (
                  <span
                    className={`grid size-5 shrink-0 place-items-center rounded-md ${className}`}
                  >
                    <Icon className='size-3' />
                  </span>
                )
              })()}
              <span className='truncate text-xs'>
                <span className='font-medium'>
                  {nodeLabel(node.class_type)}
                </span>{' '}
                {boundLabel ? (
                  <span className='font-mono text-muted-foreground'>
                    · {boundLabel}
                  </span>
                ) : null}
              </span>
            </span>
          ) : (
            <span className='truncate text-muted-foreground'>
              {nodes.length === 0
                ? t('cases.emptyWorkflowLock')
                : t('cases.bindPickPlaceholder')}
            </span>
          )}
          <ChevronsUpDown className='size-3.5 shrink-0 text-muted-foreground' />
        </Button>
      </PopoverTrigger>
      <PopoverContent
        align='start'
        side='bottom'
        className='w-80 p-0'
      >
        <div className='border-b p-2'>
          <div className='relative'>
            <Search className='pointer-events-none absolute left-2 top-1/2 size-3.5 -translate-y-1/2 text-muted-foreground' />
            <Input
              ref={searchRef}
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder={t('cases.bindSearchPlaceholder')}
              className='h-8 pl-7 text-xs'
              autoComplete='off'
            />
          </div>
        </div>
        <div className='max-h-72 overflow-auto p-1'>
          {mode === 'output' ? (
            outputGroups.length === 0 ? (
              <p className='px-2 py-4 text-center text-xs text-muted-foreground'>
                {query ? t('cases.bindNoResults') : t('cases.bindNoParams')}
              </p>
            ) : (
              outputGroups.map((n) => {
                const { Icon, className } = nodeVisualFor(n.class_type)
                return (
                  <div key={n.id} className='mb-2'>
                    <div className='flex items-center gap-1.5 px-2 py-1.5'>
                      <span
                        className={`grid size-5 shrink-0 place-items-center rounded-md ${className}`}
                      >
                        <Icon className='size-3' />
                      </span>
                      <span className='truncate text-xs font-semibold'>
                        {nodeLabel(n.class_type)}
                      </span>
                      <span className='font-mono text-[10px] text-muted-foreground'>
                        #{n.id}
                      </span>
                      {n.outputCount === 1 ? (
                        <span className='ml-auto text-[10px] text-muted-foreground'>
                          {t('cases.singleOutputAuto')}
                        </span>
                      ) : null}
                    </div>
                    <div className='space-y-0.5'>
                      {Array.from({ length: n.outputCount }, (_, i) => (
                        <button
                          key={i}
                          type='button'
                          onClick={() => pick(n.id, String(i))}
                          className='flex w-full items-center gap-2 rounded-md py-1.5 pr-2 pl-6 text-left font-mono text-xs text-foreground/90 transition-colors hover:bg-accent'
                        >
                          <ChevronRight className='size-3 shrink-0 text-muted-foreground' />
                          <span>[{i}]</span>
                          <span className='ml-auto shrink-0 rounded bg-muted px-1.5 py-0.5 text-[10px] text-muted-foreground'>
                            {t(TYPE_LABEL_KEYS[outputKindFor(n.class_type)])}
                          </span>
                        </button>
                      ))}
                    </div>
                  </div>
                )
              })
            )
          ) : inputGroups.length === 0 ? (
            <p className='px-2 py-4 text-center text-xs text-muted-foreground'>
              {query ? t('cases.bindNoResults') : t('cases.bindNoParams')}
            </p>
          ) : (
            inputGroups.map(({ node: n, params }) => {
              const { Icon, className } = nodeVisualFor(n.class_type)
              return (
                <div key={n.id} className='mb-2'>
                  <div className='flex items-center gap-1.5 px-2 py-1.5'>
                    <span
                      className={`grid size-5 shrink-0 place-items-center rounded-md ${className}`}
                    >
                      <Icon className='size-3' />
                    </span>
                    <span className='truncate text-xs font-semibold'>
                      {nodeLabel(n.class_type)}
                    </span>
                    <span className='font-mono text-[10px] text-muted-foreground'>
                      #{n.id}
                    </span>
                  </div>
                  <div className='space-y-0.5'>
                    {params.map((p) => (
                      <button
                        key={p.name}
                        type='button'
                        onClick={() => pick(n.id, p.name)}
                        className='flex w-full items-center gap-2 rounded-md py-1.5 pr-2 pl-6 text-left text-xs text-foreground/90 transition-colors hover:bg-accent'
                      >
                        <ChevronRight className='size-3 shrink-0 text-muted-foreground' />
                        <span className='truncate font-mono'>{p.name}</span>
                        <span className='ml-auto shrink-0 rounded bg-muted px-1.5 py-0.5 text-[10px] text-muted-foreground'>
                          {t(TYPE_LABEL_KEYS[p.kind] ?? 'cases.typeString')}
                        </span>
                      </button>
                    ))}
                  </div>
                </div>
              )
            })
          )}
        </div>
      </PopoverContent>
    </Popover>
  )
}

// ===== 删除按钮（带确认弹层）=====

type RemoveFieldButtonProps = {
  fieldKey: string
  disabled?: boolean
  onRemove: () => void
  compact?: boolean
}

export function RemoveFieldButton({
  fieldKey,
  disabled,
  onRemove,
  compact,
}: RemoveFieldButtonProps) {
  const { t } = useTranslation()
  return (
    <AlertDialog>
      <AlertDialogTrigger asChild>
        <Button
          type='button'
          variant='ghost'
          size={compact ? 'sm' : 'icon'}
          disabled={disabled}
          aria-label={t('cases.removeRow')}
          className={
            compact
              ? 'h-7 w-7 text-muted-foreground hover:text-destructive'
              : 'text-muted-foreground hover:text-destructive'
          }
        >
          <Trash2 className={compact ? 'h-3.5 w-3.5' : 'h-4 w-4'} />
          <span className='sr-only'>{t('cases.removeRow')}</span>
        </Button>
      </AlertDialogTrigger>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{t('cases.deleteFieldTitle')}</AlertDialogTitle>
          <AlertDialogDescription>
            {t('cases.deleteFieldBody', { key: fieldKey })}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel type='button'>
            {t('common.cancel')}
          </AlertDialogCancel>
          <AlertDialogAction
            type='button'
            onClick={onRemove}
            className='bg-destructive text-white hover:bg-destructive/90'
          >
            {t('common.delete')}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}

// ===== 输入字段卡片（窄屏降级）=====

type InputCardProps = {
  nodes: WorkflowNode[]
  value: InputFieldDraft
  onChange: (next: InputFieldDraft) => void
  onRemove: () => void
  disabled?: boolean
}

export function InputFieldCard({
  nodes,
  value,
  onChange,
  onRemove,
  disabled,
}: InputCardProps) {
  const { t } = useTranslation()
  const node = nodes.find((n) => n.id === value.node_id)
  const bound = !!(value.node_id && value.field_path)
  const autoType =
    bound && node ? safeAutoType(node.class_type, value.field_path) : ''
  const isCustom = bound && value.type !== autoType

  return (
    <li
      data-testid='input-field-card'
      className='space-y-2 rounded-md border p-3'
    >
      <div className='flex flex-wrap items-center gap-2'>
        <span className='text-sm text-foreground'>{t('cases.fieldKey')}</span>
        <Input
          className='h-8 w-36'
          value={value.key}
          onChange={(e) => onChange({ ...value, key: e.target.value })}
          disabled={disabled}
          autoComplete='off'
        />
      </div>

      <div className='flex items-center gap-2'>
        <span className='shrink-0 text-sm text-foreground'>
          {t('cases.fieldDescription')}
        </span>
        <Input
          className='h-8 flex-1'
          value={value.description ?? ''}
          onChange={(e) => onChange({ ...value, description: e.target.value })}
          disabled={disabled}
          autoComplete='off'
        />
      </div>

      <div>
        <span className='text-sm text-foreground'>{t('cases.fieldBind')}</span>
        <BindNodePopover
          mode='input'
          nodes={nodes}
          node={bound ? node : undefined}
          boundLabel={value.field_path}
          disabled={disabled}
          onPick={(nodeId, fieldPath) => {
            const picked = nodes.find((n) => n.id === nodeId)
            onChange({
              ...value,
              node_id: nodeId,
              field_path: fieldPath,
              type: picked
                ? safeAutoType(picked.class_type, fieldPath)
                : value.type,
            })
          }}
        />
      </div>

      <div className='flex flex-wrap items-center gap-2'>
        <span className='text-sm text-foreground'>{t('cases.fieldType')}</span>
        <Select
          value={value.type}
          onValueChange={(type) =>
            onChange({ ...value, type: type as InputFieldDraft['type'] })
          }
          disabled={disabled}
        >
          <SelectTrigger size='sm' className='w-28'>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {INPUT_TYPES.map((type) => (
              <SelectItem key={type.value} value={type.value}>
                {t(type.labelKey)}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        {bound ? (
          isCustom ? (
            <>
              <Badge
                variant='secondary'
                className='border-amber-200 bg-amber-50 text-amber-600 dark:border-amber-900 dark:bg-amber-950/50 dark:text-amber-400'
              >
                {t('cases.typeCustom')}
              </Badge>
              <button
                type='button'
                disabled={disabled}
                onClick={() => onChange({ ...value, type: autoType })}
                className='text-xs text-sky-600 underline disabled:opacity-50 dark:text-sky-400'
              >
                {t('cases.typeRestoreAuto')}
              </button>
            </>
          ) : (
            <>
              <Badge
                variant='secondary'
                className='border-sky-200 bg-sky-50 text-sky-600 dark:border-sky-900 dark:bg-sky-950/50 dark:text-sky-400'
              >
                {t('cases.typeAuto')}
              </Badge>
              <span className='text-xs text-muted-foreground'>
                {t('cases.typeAutoSource', {
                  node: nodeLabel(node?.class_type ?? ''),
                  param: value.field_path,
                })}
              </span>
            </>
          )
        ) : null}
      </div>

      {value.type === 'enum' ? (
        <div className='flex items-center gap-2'>
          <span className='text-sm text-foreground'>
            {t('cases.fieldEnumOptions')}
          </span>
          <Input
            className='h-8 flex-1'
            value={(value.enum_values ?? []).join(', ')}
            onChange={(e) =>
              onChange({
                ...value,
                enum_values: e.target.value
                  .split(',')
                  .map((s) => s.trim())
                  .filter(Boolean),
              })
            }
            disabled={disabled}
            autoComplete='off'
          />
        </div>
      ) : null}

      <div className='flex items-center justify-between border-t pt-2'>
        <label className='flex items-center gap-1.5 text-sm'>
          <Checkbox
            checked={value.required}
            onCheckedChange={(v) =>
              onChange({ ...value, required: v === true })
            }
            disabled={disabled}
          />
          {t('cases.fieldRequired')}
        </label>
        <RemoveFieldButton
          fieldKey={value.key}
          disabled={disabled}
          onRemove={onRemove}
        />
      </div>
    </li>
  )
}

// ===== 输出字段卡片（窄屏降级）=====

type OutputCardProps = {
  nodes: WorkflowNode[]
  value: OutputFieldDraft
  onChange: (next: OutputFieldDraft) => void
  onRemove: () => void
  disabled?: boolean
}

export function OutputFieldCard({
  nodes,
  value,
  onChange,
  onRemove,
  disabled,
}: OutputCardProps) {
  const { t } = useTranslation()
  const node = nodes.find((n) => n.id === value.node_id)
  const bound = !!value.node_id
  const autoType = bound && node ? outputKindFor(node.class_type) : ''
  const isCustom = bound && value.type !== autoType

  return (
    <li
      data-testid='output-field-card'
      className='space-y-2 rounded-md border p-3'
    >
      <div className='flex flex-wrap items-center gap-2'>
        <span className='text-sm text-foreground'>{t('cases.fieldKey')}</span>
        <Input
          className='h-8 w-36'
          value={value.key}
          onChange={(e) => onChange({ ...value, key: e.target.value })}
          disabled={disabled}
          autoComplete='off'
        />
        <span className='text-sm text-foreground'>{t('cases.fieldType')}</span>
        <Select
          value={value.type}
          onValueChange={(type) =>
            onChange({ ...value, type: type as OutputFieldDraft['type'] })
          }
          disabled={disabled}
        >
          <SelectTrigger size='sm' className='w-28'>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {OUTPUT_TYPES.map((type) => (
              <SelectItem key={type.value} value={type.value}>
                {t(type.labelKey)}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        {bound ? (
          isCustom ? (
            <>
              <Badge
                variant='secondary'
                className='border-amber-200 bg-amber-50 text-amber-600 dark:border-amber-900 dark:bg-amber-950/50 dark:text-amber-400'
              >
                {t('cases.typeCustom')}
              </Badge>
              <button
                type='button'
                disabled={disabled}
                onClick={() => onChange({ ...value, type: autoType })}
                className='text-xs text-sky-600 underline disabled:opacity-50 dark:text-sky-400'
              >
                {t('cases.typeRestoreAuto')}
              </button>
            </>
          ) : (
            <>
              <Badge
                variant='secondary'
                className='border-sky-200 bg-sky-50 text-sky-600 dark:border-sky-900 dark:bg-sky-950/50 dark:text-sky-400'
              >
                {t('cases.typeAuto')}
              </Badge>
              <span className='text-xs text-muted-foreground'>
                {t('cases.typeAutoSourceOutput', {
                  node: nodeLabel(node?.class_type ?? ''),
                })}
              </span>
            </>
          )
        ) : null}
      </div>

      <div className='flex items-center gap-2'>
        <span className='shrink-0 text-sm text-foreground'>
          {t('cases.fieldDescription')}
        </span>
        <Input
          className='h-8 flex-1'
          value={value.description ?? ''}
          onChange={(e) => onChange({ ...value, description: e.target.value })}
          disabled={disabled}
          autoComplete='off'
        />
      </div>

      <div>
        <span className='text-sm text-foreground'>{t('cases.fieldBind')}</span>
        <BindNodePopover
          mode='output'
          nodes={nodes}
          node={bound ? node : undefined}
          boundLabel={`[${value.index ?? 0}]`}
          disabled={disabled}
          onPick={(nodeId, pick) => {
            const picked = nodes.find((n) => n.id === nodeId)
            onChange({
              ...value,
              node_id: nodeId,
              index: Number.parseInt(pick, 10) || 0,
              type: picked ? outputKindFor(picked.class_type) : value.type,
            })
          }}
        />
      </div>

      <div className='flex justify-end border-t pt-2'>
        <RemoveFieldButton
          fieldKey={value.key}
          disabled={disabled}
          onRemove={onRemove}
        />
      </div>
    </li>
  )
}

// ===== 宽屏表格化批量编辑 =====

type InputTableProps = {
  nodes: WorkflowNode[]
  fields: InputFieldDraft[]
  onChange: (index: number, next: InputFieldDraft) => void
  onRemove: (index: number) => void
  disabled?: boolean
}

export function InputFieldsTable({
  nodes,
  fields,
  onChange,
  onRemove,
  disabled,
}: InputTableProps) {
  const { t } = useTranslation()
  return (
    <div className='overflow-hidden rounded-md border'>
      <div className='max-h-[200px] overflow-auto'>
        <Table data-testid='input-fields-table' className='table-fixed'>
        <TableHeader className='sticky top-0 z-10 bg-background [&_th]:bg-background'>
          <TableRow>
            <TableHead className='w-40'>
              {t('cases.fieldKey')}
            </TableHead>
            <TableHead className='w-32'>
              {t('cases.fieldType')}
            </TableHead>
            <TableHead className='w-48'>
              {t('cases.fieldBind')}
            </TableHead>
            <TableHead className='w-14 text-center'>
              {t('cases.fieldRequired')}
            </TableHead>
            <TableHead className='min-w-0'>
              {t('cases.fieldDescription')}
            </TableHead>
            <TableHead className='w-14' />
          </TableRow>
        </TableHeader>
        <TableBody>
          {fields.map((value, index) => {
            const node = nodes.find((n) => n.id === value.node_id)
            const bound = !!(value.node_id && value.field_path)
            return (
              <TableRow key={`input-${index}`} data-testid='input-table-row'>
                <TableCell>
                  <div className='flex items-center gap-1'>
                    <Input
                      className='h-8 min-w-0 flex-1'
                      value={value.key}
                      onChange={(e) =>
                        onChange(index, { ...value, key: e.target.value })
                      }
                      disabled={disabled}
                      autoComplete='off'
                    />
                    {value.required ? (
                      <span
                        className='shrink-0 text-destructive'
                        aria-label={t('cases.fieldRequired')}
                        title={t('cases.fieldRequired')}
                      >
                        *
                      </span>
                    ) : null}
                  </div>
                </TableCell>
                <TableCell>
                  <Select
                    value={value.type}
                    onValueChange={(type) =>
                      onChange(index, {
                        ...value,
                        type: type as InputFieldDraft['type'],
                      })
                    }
                    disabled={disabled}
                  >
                    <SelectTrigger size='sm' className='w-28'>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {INPUT_TYPES.map((type) => (
                        <SelectItem key={type.value} value={type.value}>
                          {t(type.labelKey)}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  {bound &&
                  value.type !==
                    (node
                      ? safeAutoType(node.class_type, value.field_path)
                      : '') ? (
                    <div className='mt-1 flex items-center gap-1.5'>
                      <Badge
                        variant='secondary'
                        className='border-amber-200 bg-amber-50 text-amber-600 dark:border-amber-900 dark:bg-amber-950/50 dark:text-amber-400'
                      >
                        {t('cases.typeCustom')}
                      </Badge>
                      <button
                        type='button'
                        disabled={disabled}
                        onClick={() =>
                          onChange(index, {
                            ...value,
                            type: node
                              ? safeAutoType(node.class_type, value.field_path)
                              : value.type,
                          })
                        }
                        className='text-xs text-sky-600 underline disabled:opacity-50 dark:text-sky-400'
                      >
                        {t('cases.typeRestoreAuto')}
                      </button>
                    </div>
                  ) : null}
                </TableCell>
                <TableCell>
                  <div className='flex items-center gap-2'>
                    <BindNodePopover
                      compact
                      mode='input'
                      nodes={nodes}
                      node={bound ? node : undefined}
                      boundLabel={value.field_path}
                      disabled={disabled}
                      onPick={(nodeId, fieldPath) => {
                        const picked = nodes.find((n) => n.id === nodeId)
                        onChange(index, {
                          ...value,
                          node_id: nodeId,
                          field_path: fieldPath,
                          type: picked
                            ? safeAutoType(picked.class_type, fieldPath)
                            : value.type,
                        })
                      }}
                    />
                  </div>
                </TableCell>
                <TableCell className='text-center'>
                  <Checkbox
                    checked={value.required}
                    onCheckedChange={(v) =>
                      onChange(index, { ...value, required: v === true })
                    }
                    disabled={disabled}
                  />
                </TableCell>
                <TableCell>
                  <Input
                    className='h-8'
                    value={value.description ?? ''}
                    onChange={(e) =>
                      onChange(index, {
                        ...value,
                        description: e.target.value,
                      })
                    }
                    disabled={disabled}
                    autoComplete='off'
                  />
                </TableCell>
                <TableCell>
                  <RemoveFieldButton
                    compact
                    fieldKey={value.key}
                    disabled={disabled}
                    onRemove={() => onRemove(index)}
                  />
                </TableCell>
              </TableRow>
            )
          })}
          {fields.length === 0 ? (
            <TableRow data-testid='input-table-empty'>
              <TableCell colSpan={6} className='py-6 text-center text-sm text-muted-foreground'>
                {t('cases.noRows')}
              </TableCell>
            </TableRow>
          ) : null}
        </TableBody>
      </Table>
      </div>
    </div>
  )
}

type OutputTableProps = {
  nodes: WorkflowNode[]
  fields: OutputFieldDraft[]
  onChange: (index: number, next: OutputFieldDraft) => void
  onRemove: (index: number) => void
  disabled?: boolean
}

export function OutputFieldsTable({
  nodes,
  fields,
  onChange,
  onRemove,
  disabled,
}: OutputTableProps) {
  const { t } = useTranslation()
  return (
    <div className='overflow-hidden rounded-md border'>
      <div className='max-h-[200px] overflow-auto'>
        <Table data-testid='output-fields-table' className='table-fixed'>
        <TableHeader className='sticky top-0 z-10 bg-background [&_th]:bg-background'>
          <TableRow>
            <TableHead className='w-40'>
              {t('cases.fieldKey')}
            </TableHead>
            <TableHead className='w-32'>
              {t('cases.fieldType')}
            </TableHead>
            <TableHead className='w-48'>
              {t('cases.fieldBind')}
            </TableHead>
            <TableHead className='min-w-0'>
              {t('cases.fieldDescription')}
            </TableHead>
            <TableHead className='w-14' />
          </TableRow>
        </TableHeader>
        <TableBody>
          {fields.map((value, index) => {
            const node = nodes.find((n) => n.id === value.node_id)
            const bound = !!value.node_id
            return (
              <TableRow key={`output-${index}`} data-testid='output-table-row'>
                <TableCell>
                  <Input
                    className='h-8'
                    value={value.key}
                    onChange={(e) =>
                      onChange(index, { ...value, key: e.target.value })
                    }
                    disabled={disabled}
                    autoComplete='off'
                  />
                </TableCell>
                <TableCell>
                  <Select
                    value={value.type}
                    onValueChange={(type) =>
                      onChange(index, {
                        ...value,
                        type: type as OutputFieldDraft['type'],
                      })
                    }
                    disabled={disabled}
                  >
                    <SelectTrigger size='sm' className='w-28'>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {OUTPUT_TYPES.map((type) => (
                        <SelectItem key={type.value} value={type.value}>
                          {t(type.labelKey)}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  {bound &&
                  node &&
                  value.type !== outputKindFor(node.class_type) ? (
                    <div className='mt-1 flex items-center gap-1.5'>
                      <Badge
                        variant='secondary'
                        className='border-amber-200 bg-amber-50 text-amber-600 dark:border-amber-900 dark:bg-amber-950/50 dark:text-amber-400'
                      >
                        {t('cases.typeCustom')}
                      </Badge>
                      <button
                        type='button'
                        disabled={disabled}
                        onClick={() =>
                          onChange(index, {
                            ...value,
                            type: outputKindFor(node.class_type),
                          })
                        }
                        className='text-xs text-sky-600 underline disabled:opacity-50 dark:text-sky-400'
                      >
                        {t('cases.typeRestoreAuto')}
                      </button>
                    </div>
                  ) : null}
                </TableCell>
                <TableCell>
                  <div className='flex items-center gap-2'>
                    <BindNodePopover
                      compact
                      mode='output'
                      nodes={nodes}
                      node={bound ? node : undefined}
                      boundLabel={`[${value.index ?? 0}]`}
                      disabled={disabled}
                      onPick={(nodeId, pick) => {
                        const picked = nodes.find((n) => n.id === nodeId)
                        onChange(index, {
                          ...value,
                          node_id: nodeId,
                          index: Number.parseInt(pick, 10) || 0,
                          type: picked
                            ? outputKindFor(picked.class_type)
                            : value.type,
                        })
                      }}
                    />
                  </div>
                </TableCell>
                <TableCell>
                  <Input
                    className='h-8'
                    value={value.description ?? ''}
                    onChange={(e) =>
                      onChange(index, {
                        ...value,
                        description: e.target.value,
                      })
                    }
                    disabled={disabled}
                    autoComplete='off'
                  />
                </TableCell>
                <TableCell>
                  <RemoveFieldButton
                    compact
                    fieldKey={value.key}
                    disabled={disabled}
                    onRemove={() => onRemove(index)}
                  />
                </TableCell>
              </TableRow>
            )
          })}
          {fields.length === 0 ? (
            <TableRow data-testid='output-table-empty'>
              <TableCell colSpan={5} className='py-6 text-center text-sm text-muted-foreground'>
                {t('cases.noRows')}
              </TableCell>
            </TableRow>
          ) : null}
        </TableBody>
      </Table>
      </div>
    </div>
  )
}



