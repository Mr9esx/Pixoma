import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import type { InputFieldDraft, OutputFieldDraft } from '../lib/derive'
import { inputKindFor, nodeLabel } from '../lib/node-catalog'
import type { WorkflowNode } from '../lib/workflow-parse'

const INPUT_TYPES = ['string', 'image', 'video', 'number', 'boolean', 'enum']
const OUTPUT_TYPES = ['image', 'text', 'file']

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

  return (
    <li
      data-testid='input-field-card'
      className='space-y-2 rounded-md border p-3'
    >
      <div className='flex flex-wrap items-center gap-2'>
        <Label className='text-xs text-muted-foreground'>
          {t('cases.fieldKey')}
        </Label>
        <Input
          className='h-8 w-36'
          value={value.key}
          onChange={(e) => onChange({ ...value, key: e.target.value })}
          disabled={disabled}
          autoComplete='off'
        />
        <span className='text-xs text-muted-foreground'>
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
              <SelectItem key={type} value={type}>
                {type}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <label className='ml-auto flex items-center gap-1.5 text-xs'>
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

      <div className='flex flex-wrap items-center gap-2'>
        <span className='text-xs text-muted-foreground'>
          {t('cases.fieldFromNode')}
        </span>
        <Select
          value={value.node_id || undefined}
          onValueChange={(nodeId) =>
            onChange({
              ...value,
              node_id: nodeId,
              field_path: '',
            })
          }
          disabled={disabled}
        >
          <SelectTrigger className='h-8 w-56'>
            <SelectValue placeholder={t('cases.nodeSearchPlaceholder')} />
          </SelectTrigger>
          <SelectContent>
            {nodes.map((n) => (
              <SelectItem key={n.id} value={n.id}>
                {nodeLabel(n.class_type)}（{n.id}）
              </SelectItem>
            ))}
          </SelectContent>
        </Select>

        <span className='text-xs text-muted-foreground'>
          {t('cases.fieldParam')}
        </span>
        <Select
          value={value.field_path || undefined}
          onValueChange={(fieldPath) =>
            onChange({
              ...value,
              field_path: fieldPath,
              type:
                value.type === 'string' || value.type === 'image'
                  ? value.type
                  : inputKindFor(node?.class_type ?? '', fieldPath),
            })
          }
          disabled={disabled || !node}
        >
          <SelectTrigger className='h-8 w-48'>
            <SelectValue
              placeholder={
                node ? t('cases.fieldParam') : t('cases.emptyWorkflowLock')
              }
            />
          </SelectTrigger>
          <SelectContent>
            {node?.inputs.map((input) => (
              <SelectItem key={input.name} value={input.name}>
                {input.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {value.type === 'enum' ? (
        <div className='flex items-center gap-2'>
          <span className='text-xs text-muted-foreground'>
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

      <div className='flex items-center gap-2'>
        <span className='text-xs text-muted-foreground'>
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
      <p className='text-xs text-emerald-600 dark:text-emerald-400'>
        {t('cases.autoBoundHint')}
      </p>
    </li>
  )
}

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

  return (
    <li
      data-testid='output-field-card'
      className='space-y-2 rounded-md border p-3'
    >
      <div className='flex flex-wrap items-center gap-2'>
        <span className='text-xs text-muted-foreground'>
          {t('cases.fieldKey')}
        </span>
        <Input
          className='h-8 w-36'
          value={value.key}
          onChange={(e) => onChange({ ...value, key: e.target.value })}
          disabled={disabled}
          autoComplete='off'
        />
        <span className='text-xs text-muted-foreground'>
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
              <SelectItem key={type} value={type}>
                {type}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <span className='text-xs text-muted-foreground'>
          {t('cases.fieldFromNode')}
        </span>
        <Select
          value={value.node_id || undefined}
          onValueChange={(nodeId) =>
            onChange({
              ...value,
              node_id: nodeId,
              index:
                nodes.find((n) => n.id === nodeId)?.outputCount === 1
                  ? 0
                  : value.index,
            })
          }
          disabled={disabled}
        >
          <SelectTrigger className='h-8 w-56'>
            <SelectValue placeholder={t('cases.nodeSearchPlaceholder')} />
          </SelectTrigger>
          <SelectContent>
            {nodes.map((n) => (
              <SelectItem key={n.id} value={n.id}>
                {nodeLabel(n.class_type)}（{n.id}）
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        {node && node.outputCount > 1 ? (
          <>
            <span className='text-xs text-muted-foreground'>
              {t('cases.fieldOutput')}
            </span>
            <Input
              className='h-8 w-20'
              type='number'
              min={0}
              value={value.index ?? 0}
              onChange={(e) =>
                onChange({
                  ...value,
                  index: Number.parseInt(e.target.value, 10) || 0,
                })
              }
              disabled={disabled}
            />
          </>
        ) : null}
        <Button
          type='button'
          size='sm'
          variant='ghost'
          className='ml-auto'
          disabled={disabled}
          onClick={onRemove}
        >
          {t('cases.removeRow')}
        </Button>
      </div>
      {node && node.outputCount === 1 ? (
        <p className='text-xs text-emerald-600 dark:text-emerald-400'>
          {t('cases.singleOutputAuto')}
        </p>
      ) : null}
    </li>
  )
}
