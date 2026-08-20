import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { ChevronRight } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/input'
import {
  Dialog,
  DialogContent,
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

// ===== 绑定弹层：图上点选（复用详情页流程图视觉）=====

type BindNodeDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  nodes: WorkflowNode[]
  /** input = 绑字面量参数；output = 绑节点输出槽位。 */
  mode?: 'input' | 'output'
  onPick: (nodeId: string, pick: string) => void
}

export function BindNodeDialog({
  open,
  onOpenChange,
  nodes,
  mode = 'input',
  onPick,
}: BindNodeDialogProps) {
  const { t } = useTranslation()
  const [nodeId, setNodeId] = useState<string | null>(null)
  const [param, setParam] = useState<string | null>(null)

  // 拓扑排序（依赖 links），与详情页 process 分层一致。
  const ordered = useMemo(() => {
    const depth = new Map<string, number>(nodes.map((n) => [n.id, 0]))
    for (let pass = 0; pass < nodes.length; pass += 1) {
      let changed = false
      for (const n of nodes) {
        for (const link of n.links) {
          if (!depth.has(link.src)) continue
          const next = (depth.get(link.src) ?? 0) + 1
          if (next > (depth.get(n.id) ?? 0)) {
            depth.set(n.id, next)
            changed = true
          }
        }
      }
      if (!changed) break
    }
    return [...nodes].sort(
      (a, b) =>
        (depth.get(a.id) ?? 0) - (depth.get(b.id) ?? 0) ||
        a.id.localeCompare(b.id)
    )
  }, [nodes])

  const selected = nodes.find((n) => n.id === nodeId) ?? null
  // 只暴露字面量参数：引用（节点间连线）不是用户能提供的输入。
  const bindable = selected?.inputs.filter((input) => !input.ref) ?? []
  const literalMap = new Map(selected?.literals ?? [])

  function close() {
    setNodeId(null)
    setParam(null)
    onOpenChange(false)
  }

  function confirm() {
    if (!selected || !param) return
    onPick(selected.id, param)
    close()
  }

  return (
    <Dialog open={open} onOpenChange={(v) => (v ? undefined : close())}>
      <DialogContent className='flex max-h-[85vh] flex-col gap-0 p-0 sm:max-w-3xl'>
        <DialogHeader className='border-b px-5 py-4'>
          <DialogTitle>{t('cases.bindDialogTitle')}</DialogTitle>
        </DialogHeader>
        <p className='border-b px-5 py-2 text-xs text-muted-foreground'>
          {t(mode === 'output' ? 'cases.bindFlowHintOutput' : 'cases.bindFlowHint')}
        </p>
        <div className='overflow-x-auto border-b bg-muted/20 px-4 py-3'>
          <div className='flex min-w-max items-stretch gap-1.5'>
            {ordered.map((n, i) => {
              const { Icon, className } = nodeVisualFor(n.class_type)
              const active = n.id === nodeId
              return (
                <div key={n.id} className='flex items-center'>
                  <button
                    type='button'
                    onClick={() => {
                      setNodeId(n.id)
                      setParam(null)
                    }}
                    className={`flex w-40 shrink-0 flex-col rounded-md border bg-background p-2.5 text-left transition-colors ${
                      active
                        ? 'border-primary bg-primary/5'
                        : 'border-border hover:border-foreground/40'
                    }`}
                  >
                    <span className='flex items-center gap-2'>
                      <span
                        className={`grid size-6 shrink-0 place-items-center rounded-md ${className}`}
                      >
                        <Icon className='size-3.5' />
                      </span>
                      <span className='truncate font-mono text-[10px] text-muted-foreground'>
                        #{n.id}
                      </span>
                    </span>
                    <span className='mt-1 truncate text-xs font-semibold leading-tight'>
                      {nodeLabel(n.class_type)}
                    </span>
                    <span className='truncate font-mono text-[10px] text-muted-foreground'>
                      {n.class_type}
                    </span>
                  </button>
                  {i < ordered.length - 1 ? (
                    <ChevronRight className='mx-1 size-4 shrink-0 text-muted-foreground/40' />
                  ) : null}
                </div>
              )
            })}
          </div>
        </div>
        <div className='min-h-0 flex-1 overflow-auto px-5 py-4'>
          {!selected ? (
            <p className='py-8 text-center text-xs text-muted-foreground'>
              {t('cases.bindPickNodeFirst')}
            </p>
          ) : mode === 'output' ? (
            <>
              <h4 className='mb-2 text-sm font-medium text-foreground'>
                {nodeLabel(selected.class_type)} #{selected.id} —{' '}
                {t('cases.bindPickParam')}
              </h4>
              <div className='flex flex-col gap-1.5'>
                {Array.from({ length: selected.outputCount }, (_, i) => {
                  const active = param === String(i)
                  const kind = outputKindFor(selected.class_type)
                  return (
                    <button
                      key={i}
                      type='button'
                      onClick={() => setParam(String(i))}
                      className={`flex items-center gap-2 rounded-md border px-3 py-2 text-left text-xs transition-colors ${
                        active
                          ? 'border-emerald-500 bg-emerald-50 dark:bg-emerald-950/40'
                          : 'border-border hover:border-foreground/40'
                      }`}
                    >
                      <span className='font-mono font-semibold'>[{i}]</span>
                      <span className='rounded-sm border bg-muted px-1.5 py-px font-mono text-[10px] text-muted-foreground'>
                        {t(TYPE_LABEL_KEYS[kind] ?? 'cases.typeImage')}
                      </span>
                      {selected.outputCount === 1 ? (
                        <span className='text-[11px] text-emerald-600 dark:text-emerald-400'>
                          {t('cases.singleOutputAuto')}
                        </span>
                      ) : null}
                    </button>
                  )
                })}
              </div>
            </>
          ) : bindable.length === 0 ? (
            <p className='py-8 text-center text-xs text-muted-foreground'>
              {t('cases.bindNoParams')}
            </p>
          ) : (
            <>
              <h4 className='mb-2 text-sm font-medium text-foreground'>
                {nodeLabel(selected.class_type)} #{selected.id} —{' '}
                {t('cases.bindPickParam')}
              </h4>
              <div className='flex flex-col gap-1.5'>
                {bindable.map((input) => {
                  const active = param === input.name
                  const literal = literalMap.get(input.name)
                  return (
                    <button
                      key={input.name}
                      type='button'
                      onClick={() => setParam(input.name)}
                      className={`flex items-center gap-2 rounded-md border px-3 py-2 text-left text-xs transition-colors ${
                        active
                          ? 'border-emerald-500 bg-emerald-50 dark:bg-emerald-950/40'
                          : 'border-border hover:border-foreground/40'
                      }`}
                    >
                      <span className='font-mono font-semibold'>{input.name}</span>
                      <span className='rounded-sm border bg-muted px-1.5 py-px font-mono text-[10px] text-muted-foreground'>
                        {t(TYPE_LABEL_KEYS[input.kind] ?? 'cases.typeString')}
                      </span>
                      {literal !== undefined ? (
                        <span className='ml-auto max-w-[45%] truncate font-mono text-[10px] text-muted-foreground'>
                          {literal}
                        </span>
                      ) : null}
                    </button>
                  )
                })}
              </div>
            </>
          )}
        </div>
        <div className='flex items-center justify-between gap-3 border-t px-5 py-3'>
          <span className='min-w-0 text-xs text-muted-foreground'>
            {selected && param ? (
              <>
                {t('cases.bindWillBind')}：
                <b className='text-foreground'>
                  {nodeLabel(selected.class_type)} #{selected.id} ·{' '}
                  {mode === 'output' ? `[${param}]` : param}
                </b>
              </>
            ) : (
              t('cases.bindPickNodeFirst')
            )}
          </span>
          <div className='flex shrink-0 gap-2'>
            <Button type='button' variant='outline' onClick={close}>
              {t('common.cancel')}
            </Button>
            <Button type='button' disabled={!selected || !param} onClick={confirm}>
              {t('cases.bindConfirm')}
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}

// ===== 输入字段卡片 =====

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
  const [bindOpen, setBindOpen] = useState(false)
  const node = nodes.find((n) => n.id === value.node_id)
  const bound = !!(value.node_id && value.field_path)
  const autoType = bound && node ? safeAutoType(node.class_type, value.field_path) : ''
  const isCustom = bound && value.type !== autoType

  return (
    <li
      data-testid='input-field-card'
      className='space-y-2 rounded-md border p-3'
    >
      {/* 字段名 + 必填 */}
      <div className='flex flex-wrap items-center gap-2'>
        <span className='text-sm text-foreground'>
          {t('cases.fieldKey')}
        </span>
        <Input
          className='h-8 w-36'
          value={value.key}
          onChange={(e) => onChange({ ...value, key: e.target.value })}
          disabled={disabled}
          autoComplete='off'
        />
        <label className='ml-auto flex items-center gap-1.5 text-sm'>
          <Checkbox
            checked={value.required}
            onCheckedChange={(v) =>
              onChange({ ...value, required: v === true })
            }
            disabled={disabled}
          />
          {t('cases.fieldRequired')}
        </label>
      </div>

      {/* 绑定位置（图上点选） */}
      <div>
        <span className='text-sm text-foreground'>
          {t('cases.fieldBind')}
        </span>
        <Button
          type='button'
          variant='outline'
          disabled={disabled || nodes.length === 0}
          onClick={() => setBindOpen(true)}
          className='mt-1 flex h-auto w-full items-center justify-between gap-2 border-2 border-dashed border-foreground/30 px-3 py-2.5 text-left hover:border-foreground/50 hover:bg-muted/40'
        >
          {bound && node ? (
            <span className='flex min-w-0 items-center gap-2'>
              {(() => {
                const { Icon, className } = nodeVisualFor(node.class_type)
                return (
                  <span
                    className={`grid size-6 shrink-0 place-items-center rounded-md ${className}`}
                  >
                    <Icon className='size-3.5' />
                  </span>
                )
              })()}
              <span className='min-w-0 truncate'>
                <span className='text-sm font-medium'>
                  {nodeLabel(node.class_type)}
                </span>{' '}
                <span className='font-mono text-xs text-muted-foreground'>
                  #{node.id} · {value.field_path}
                </span>
              </span>
            </span>
          ) : (
            <span className='text-sm text-muted-foreground'>
              {nodes.length === 0
                ? t('cases.emptyWorkflowLock')
                : t('cases.bindPickPlaceholder')}
            </span>
          )}
          <ChevronRight className='size-4 shrink-0 text-muted-foreground' />
        </Button>
      </div>

      {/* 类型（自动为主，可覆盖） */}
      <div className='flex flex-wrap items-center gap-2'>
        <span className='text-sm text-foreground'>
          {t('cases.fieldType')}
        </span>
        <Select
          value={value.type}
          onValueChange={(type) =>
            onChange({ ...value, type: type as InputFieldDraft['type'] })
          }
          disabled={disabled}
        >
          <SelectTrigger className='h-8 w-28'>
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
              <Badge variant='secondary' className='border-amber-200 bg-amber-50 text-amber-600 dark:border-amber-900 dark:bg-amber-950/50 dark:text-amber-400'>
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
              <Badge variant='secondary' className='border-sky-200 bg-sky-50 text-sky-600 dark:border-sky-900 dark:bg-sky-950/50 dark:text-sky-400'>
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

      {/* 描述 + 删除 */}
      <div className='flex items-center gap-2'>
        <span className='text-sm text-foreground'>
          {t('cases.fieldDescription')}
        </span>
        <Input
          className='h-8 flex-1'
          value={value.description ?? ''}
          onChange={(e) => onChange({ ...value, description: e.target.value })}
          disabled={disabled}
          autoComplete='off'
        />
        <Button
          type='button'
          size='sm'
          variant='ghost'
          disabled={disabled}
          onClick={onRemove}
        >
          {t('cases.removeRow')}
        </Button>
      </div>

      <BindNodeDialog
        open={bindOpen}
        onOpenChange={setBindOpen}
        nodes={nodes}
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
    </li>
  )
}

// ===== 输出字段卡片 =====

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
  const [bindOpen, setBindOpen] = useState(false)
  const node = nodes.find((n) => n.id === value.node_id)
  const bound = !!value.node_id
  const autoType = bound && node ? outputKindFor(node.class_type) : ''
  const isCustom = bound && value.type !== autoType

  return (
    <li
      data-testid='output-field-card'
      className='space-y-2 rounded-md border p-3'
    >
      {/* 字段名 + 类型（自动为主，可覆盖） */}
      <div className='flex flex-wrap items-center gap-2'>
        <span className='text-sm text-foreground'>
          {t('cases.fieldKey')}
        </span>
        <Input
          className='h-8 w-36'
          value={value.key}
          onChange={(e) => onChange({ ...value, key: e.target.value })}
          disabled={disabled}
          autoComplete='off'
        />
        <span className='text-sm text-foreground'>
          {t('cases.fieldType')}
        </span>
        <Select
          value={value.type}
          onValueChange={(type) =>
            onChange({ ...value, type: type as OutputFieldDraft['type'] })
          }
          disabled={disabled}
        >
          <SelectTrigger className='h-8 w-28'>
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
              <Badge variant='secondary' className='border-amber-200 bg-amber-50 text-amber-600 dark:border-amber-900 dark:bg-amber-950/50 dark:text-amber-400'>
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
              <Badge variant='secondary' className='border-sky-200 bg-sky-50 text-sky-600 dark:border-sky-900 dark:bg-sky-950/50 dark:text-sky-400'>
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

      {/* 绑定位置（图上点选输出槽位） */}
      <div>
        <span className='text-sm text-foreground'>
          {t('cases.fieldBind')}
        </span>
        <Button
          type='button'
          variant='outline'
          disabled={disabled || nodes.length === 0}
          onClick={() => setBindOpen(true)}
          className='mt-1 flex h-auto w-full items-center justify-between gap-2 border-2 border-dashed border-foreground/30 px-3 py-2.5 text-left hover:border-foreground/50 hover:bg-muted/40'
        >
          {bound && node ? (
            <span className='flex min-w-0 items-center gap-2'>
              {(() => {
                const { Icon, className } = nodeVisualFor(node.class_type)
                return (
                  <span
                    className={`grid size-6 shrink-0 place-items-center rounded-md ${className}`}
                  >
                    <Icon className='size-3.5' />
                  </span>
                )
              })()}
              <span className='min-w-0 truncate'>
                <span className='text-sm font-medium'>
                  {nodeLabel(node.class_type)}
                </span>{' '}
                <span className='font-mono text-xs text-muted-foreground'>
                  #{node.id} · [{value.index ?? 0}]
                </span>
              </span>
            </span>
          ) : (
            <span className='text-sm text-muted-foreground'>
              {nodes.length === 0
                ? t('cases.emptyWorkflowLock')
                : t('cases.bindPickPlaceholder')}
            </span>
          )}
          <ChevronRight className='size-4 shrink-0 text-muted-foreground' />
        </Button>
      </div>

      {/* 描述 + 删除 */}
      <div className='flex items-center gap-2'>
        <span className='text-sm text-foreground'>
          {t('cases.fieldDescription')}
        </span>
        <Input
          className='h-8 flex-1'
          value={value.description ?? ''}
          onChange={(e) => onChange({ ...value, description: e.target.value })}
          disabled={disabled}
          autoComplete='off'
        />
        <Button
          type='button'
          size='sm'
          variant='ghost'
          disabled={disabled}
          onClick={onRemove}
        >
          {t('cases.removeRow')}
        </Button>
      </div>

      <BindNodeDialog
        mode='output'
        open={bindOpen}
        onOpenChange={setBindOpen}
        nodes={nodes}
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
    </li>
  )
}
