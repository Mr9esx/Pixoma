import { createContext, useContext, useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { ChevronDown, Trash2 } from 'lucide-react'
import type { ActionType } from '@/lib/api/channel-menu'
import { Button } from '@/components/ui/button'
import {
  Collapsible,
  CollapsibleTrigger,
} from '@/components/ui/collapsible'
import { Field, FieldGroup, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { MarkdownTextField } from '@/features/cases/sections/markdown-text-field'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectLabel,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Textarea } from '@/components/ui/textarea'
import { ACTION_TYPES } from './action-templates'
import { ListTasksPreview } from './list-tasks-preview'
import { type WorkflowRef } from './node-view'
import { uid } from './puck-map'
import { WorkflowInfoCard } from './workflow-info-card'

export type MenuPuckMeta = {
  workflows: WorkflowRef[]
  channelId: string
}

const MenuPuckMetaContext = createContext<MenuPuckMeta>({
  workflows: [],
  channelId: '',
})

export function MenuPuckMetaProvider({
  value,
  children,
}: {
  value: MenuPuckMeta
  children: ReactNode
}) {
  return (
    <MenuPuckMetaContext.Provider value={value}>
      {children}
    </MenuPuckMetaContext.Provider>
  )
}

function meta(): MenuPuckMeta {
  return useContext(MenuPuckMetaContext)
}

const ACTION_KEYS: Record<ActionType, string> = {
  open_card: 'menu.actionOpenCard',
  open_workflow: 'menu.actionOpenWorkflow',
  list_tasks: 'menu.actionListTasks',
  send_text: 'menu.actionSendText',
  send_media: 'menu.actionSendMedia',
  open_url: 'menu.actionOpenUrl',
  copy_text: 'menu.actionCopyText',
}

const WORKFLOW_CAPABILITIES: ActionType[] = ['open_workflow', 'list_tasks']
const TG_PLATFORM_CAPABILITIES: ActionType[] = [
  'open_card',
  'send_text',
  'send_media',
  'open_url',
  'copy_text',
]

type FieldRender<T> = {
  value: T
  onChange: (value: T) => void
  readOnly?: boolean
}

type CardButtonDraft = {
  id: string
  label: string
  actionType: string
  workflow_id: string
  text: string
  url: string
  mediaUrl: string
  cardText: string
  nestedCardText: string
  cardButtons: CardButtonDraft[]
}

const cardButtonOpen = new Map<string, boolean>()

function emptyCardButton(): CardButtonDraft {
  const id = uid('btn')
  cardButtonOpen.set(id, true)
  return {
    id,
    label: '按钮',
    actionType: 'send_text',
    workflow_id: '',
    text: '',
    url: '',
    mediaUrl: '',
    cardText: '',
    nestedCardText: '',
    cardButtons: [],
  }
}

function itemKey(item: CardButtonDraft, index: number): string {
  return item.id || `card-btn-${index}`
}

export function ActionTypeField({ value, onChange, readOnly }: FieldRender<string>) {
  const { t } = useTranslation()
  const current = (value || 'send_text') as ActionType
  return (
    <Field>
      <FieldLabel>{t('menu.buttonAction')}</FieldLabel>
      <Select value={current} onValueChange={onChange} disabled={readOnly}>
        <SelectTrigger data-testid='action-type-select' className='w-full'>
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectGroup>
            <SelectLabel>{t('menu.actionGroupWorkflow')}</SelectLabel>
            {ACTION_TYPES.filter((type) =>
              WORKFLOW_CAPABILITIES.includes(type)
            ).map((type) => (
              <SelectItem key={type} value={type}>
                {t(ACTION_KEYS[type])}
              </SelectItem>
            ))}
          </SelectGroup>
          <SelectGroup>
            <SelectLabel>{t('menu.actionGroupTg')}</SelectLabel>
            {ACTION_TYPES.filter((type) =>
              TG_PLATFORM_CAPABILITIES.includes(type)
            ).map((type) => (
              <SelectItem key={type} value={type}>
                {t(ACTION_KEYS[type])}
              </SelectItem>
            ))}
          </SelectGroup>
        </SelectContent>
      </Select>
    </Field>
  )
}

export function WorkflowField({ value, onChange, readOnly }: FieldRender<string>) {
  const { t } = useTranslation()
  const { workflows } = meta()
  const selected = workflows.find((w) => String(w.id) === value)
  return (
    <Field>
      <FieldLabel>{t('menu.workflowList')}</FieldLabel>
      <Select
        value={value || undefined}
        onValueChange={onChange}
        disabled={readOnly}
      >
        <SelectTrigger data-testid='workflow-select' className='w-full'>
          <SelectValue placeholder={t('menu.workflowList')} />
        </SelectTrigger>
        <SelectContent>
          <SelectGroup>
            {workflows.map((w) => (
              <SelectItem key={w.id} value={String(w.id)}>
                {w.name}
              </SelectItem>
            ))}
          </SelectGroup>
        </SelectContent>
      </Select>
      {value ? (
        <WorkflowInfoCard
          workflow={
            selected ?? {
              id: Number(value),
              name: t('menu.mapWorkflowMissing'),
            }
          }
        />
      ) : null}
    </Field>
  )
}

export function ListTasksField() {
  const { t } = useTranslation()
  const { channelId } = meta()
  return (
    <Field>
      <FieldLabel>{t('menu.preview')}</FieldLabel>
      <ListTasksPreview channelId={channelId} />
    </Field>
  )
}

export function ButtonLabelField({
  value,
  onChange,
  readOnly,
}: FieldRender<string>) {
  const { t } = useTranslation()
  return (
    <Field>
      <FieldLabel>{t('menu.buttonLabel')}</FieldLabel>
      <Input
        value={value ?? ''}
        onChange={(e) => onChange(e.target.value)}
        disabled={readOnly}
      />
    </Field>
  )
}

export function TextPayloadField({
  value,
  onChange,
  readOnly,
}: FieldRender<string>) {
  const { t } = useTranslation()
  return (
    <Field>
      <FieldLabel>{t('menu.textPayload')}</FieldLabel>
      <MarkdownTextField
        value={value ?? ''}
        onChange={onChange}
        disabled={readOnly}
      />
    </Field>
  )
}

export function UrlField({ value, onChange, readOnly }: FieldRender<string>) {
  const { t } = useTranslation()
  return (
    <Field>
      <FieldLabel>{t('menu.actionOpenUrl')}</FieldLabel>
      <Input
        value={value ?? ''}
        onChange={(e) => onChange(e.target.value)}
        disabled={readOnly}
      />
    </Field>
  )
}

export function MediaField({ value, onChange, readOnly }: FieldRender<string>) {
  const { t } = useTranslation()
  return (
    <Field>
      <FieldLabel>{t('menu.cardMedia')}</FieldLabel>
      <Textarea
        value={value ?? ''}
        onChange={(e) => onChange(e.target.value)}
        rows={3}
        disabled={readOnly}
      />
    </Field>
  )
}

export function CardTextField({
  value,
  onChange,
  readOnly,
}: FieldRender<string>) {
  const { t } = useTranslation()
  return (
    <Field>
      <FieldLabel>{t('menu.cardText')}</FieldLabel>
      <MarkdownTextField
        value={value ?? ''}
        onChange={onChange}
        disabled={readOnly}
      />
    </Field>
  )
}

export function ColumnsField({
  value,
  onChange,
  readOnly,
}: FieldRender<number>) {
  const { t } = useTranslation()
  return (
    <Field>
      <FieldLabel>{t('menu.columnCount')}</FieldLabel>
      <Input
        type='number'
        min={1}
        max={6}
        value={Number(value) || 2}
        onChange={(e) => onChange(Number(e.target.value) || 2)}
        disabled={readOnly}
      />
    </Field>
  )
}

function NestedButtonFields({
  item,
  onChange,
  readOnly,
}: {
  item: CardButtonDraft
  onChange: (next: CardButtonDraft) => void
  readOnly?: boolean
}) {
  const type = item.actionType || 'send_text'
  return (
    <div className='flex flex-col gap-3 pt-2'>
      <ButtonLabelField
        value={item.label}
        onChange={(label) => onChange({ ...item, label })}
        readOnly={readOnly}
      />
      <ActionTypeField
        value={type}
        onChange={(actionType) => onChange({ ...item, actionType })}
        readOnly={readOnly}
      />
      {type === 'open_workflow' ? (
        <WorkflowField
          value={item.workflow_id}
          onChange={(workflow_id) => onChange({ ...item, workflow_id })}
          readOnly={readOnly}
        />
      ) : null}
      {type === 'list_tasks' ? <ListTasksField /> : null}
      {type === 'send_text' || type === 'copy_text' ? (
        <TextPayloadField
          value={item.text}
          onChange={(text) => onChange({ ...item, text })}
          readOnly={readOnly}
        />
      ) : null}
      {type === 'open_url' ? (
        <UrlField
          value={item.url}
          onChange={(url) => onChange({ ...item, url })}
          readOnly={readOnly}
        />
      ) : null}
      {type === 'send_media' ? (
        <MediaField
          value={item.mediaUrl}
          onChange={(mediaUrl) => onChange({ ...item, mediaUrl })}
          readOnly={readOnly}
        />
      ) : null}
      {type === 'open_card' ? (
        <>
          <CardTextField
            value={item.cardText || item.nestedCardText}
            onChange={(cardText) =>
              onChange({ ...item, cardText, nestedCardText: cardText })
            }
            readOnly={readOnly}
          />
          <CardButtonsField
            value={item.cardButtons ?? []}
            onChange={(cardButtons) => onChange({ ...item, cardButtons })}
            readOnly={readOnly}
          />
        </>
      ) : null}
    </div>
  )
}

function CardButtonRow({
  item,
  index,
  readOnly,
  onChange,
  onRemove,
}: {
  item: CardButtonDraft
  index: number
  readOnly?: boolean
  onChange: (next: CardButtonDraft) => void
  onRemove: () => void
}) {
  const { t } = useTranslation()
  const id = itemKey(item, index)
  const [open, setOpen] = useState(() => cardButtonOpen.get(id) ?? false)

  return (
    <Collapsible
      open={open}
      onOpenChange={(next) => {
        cardButtonOpen.set(id, next)
        setOpen(next)
      }}
      className='rounded-md border'
    >
      <div className='flex items-center gap-1 p-1'>
        <CollapsibleTrigger asChild>
          <Button
            type='button'
            variant='ghost'
            className='h-11 min-w-0 flex-1 justify-between'
          >
            <span className='truncate'>{item.label || t('menu.untitled')}</span>
            <ChevronDown data-icon='inline-end' />
          </Button>
        </CollapsibleTrigger>
        <Button
          type='button'
          variant='ghost'
          className='h-11'
          disabled={readOnly}
          aria-label={t('menu.deleteButton')}
          onClick={onRemove}
        >
          <Trash2 />
        </Button>
      </div>
      {open ? (
        <div className='border-t px-2 pb-2'>
          <NestedButtonFields
            item={item}
            readOnly={readOnly}
            onChange={onChange}
          />
        </div>
      ) : null}
    </Collapsible>
  )
}

export function CardButtonsField({
  value,
  onChange,
  readOnly,
}: FieldRender<CardButtonDraft[]>) {
  const { t } = useTranslation()
  const items = Array.isArray(value) ? value : []
  return (
    <Field data-testid='card-buttons-field'>
      <FieldLabel>{t('menu.cardButtons')}</FieldLabel>
      <FieldGroup className='gap-2'>
        {items.map((item, index) => (
          <CardButtonRow
            key={itemKey(item, index)}
            item={item}
            index={index}
            readOnly={readOnly}
            onChange={(next) =>
              onChange(items.map((row, i) => (i === index ? next : row)))
            }
            onRemove={() => onChange(items.filter((_, i) => i !== index))}
          />
        ))}
        <Button
          type='button'
          variant='outline'
          className='h-11 w-full'
          disabled={readOnly}
          onClick={() => onChange([...items, emptyCardButton()])}
        >
          {t('menu.addCardButton')}
        </Button>
      </FieldGroup>
    </Field>
  )
}
