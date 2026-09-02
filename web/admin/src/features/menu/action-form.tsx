import { useTranslation } from 'react-i18next'
import type { Action, ActionType } from '@/lib/api/channel-menu'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
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
import { type WorkflowRef } from './node-view'
import { WorkflowInfoCard } from './workflow-info-card'
import { ListTasksPreview } from './list-tasks-preview'

type Props = {
  action: Action
  workflows: WorkflowRef[]
  onChange: (next: Action) => void
  disabled?: boolean
  compact?: boolean
  channelId?: string
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

// 平台能力（Pixoma）/ TG 能力（Telegram 消息层）
const WORKFLOW_CAPABILITIES: ActionType[] = ['open_workflow', 'list_tasks']
const TG_PLATFORM_CAPABILITIES: ActionType[] = [
  'open_card',
  'send_text',
  'send_media',
  'open_url',
  'copy_text',
]

export function ActionForm({
  action,
  workflows,
  onChange,
  disabled,
  compact,
  channelId,
}: Props) {
  const { t } = useTranslation()

  function switchType(type: ActionType) {
    switch (type) {
      case 'open_workflow':
        onChange({ type, workflow_id: '' })
        return
      case 'open_card':
        onChange({ type, card: { text: '' } })
        return
      case 'list_tasks':
        onChange({ type })
        return
      case 'send_text':
      case 'copy_text':
        onChange({ type, text: '' })
        return
      case 'send_media':
        onChange({ type, media: [] })
        return
      case 'open_url':
        onChange({ type, url: '' })
        return
    }
  }

  return (
    <div data-testid='action-form' className='flex flex-col gap-3'>
      <div className='flex flex-col gap-1.5'>
        {compact ? null : <Label>{t('menu.buttonAction')}</Label>}
        <Select
          value={action.type}
          onValueChange={(type) => switchType(type as ActionType)}
          disabled={disabled}
        >
          <SelectTrigger data-testid='action-type-select'>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectGroup>
              <SelectLabel>{t('menu.actionGroupWorkflow')}</SelectLabel>
              {ACTION_TYPES.filter((t) =>
                WORKFLOW_CAPABILITIES.includes(t)
              ).map((type) => (
                <SelectItem key={type} value={type} data-capability='workflow'>
                  {t(ACTION_KEYS[type])}
                </SelectItem>
              ))}
            </SelectGroup>
            <SelectGroup>
              <SelectLabel>{t('menu.actionGroupTg')}</SelectLabel>
              {ACTION_TYPES.filter((t) =>
                TG_PLATFORM_CAPABILITIES.includes(t)
              ).map((type) => (
                <SelectItem key={type} value={type} data-capability='tg'>
                  {t(ACTION_KEYS[type])}
                </SelectItem>
              ))}
            </SelectGroup>
          </SelectContent>
        </Select>
      </div>

      {action.type === 'open_card' ? (
        <div className='flex flex-col gap-1.5'>
          {compact ? null : <Label>{t('menu.cardText')}</Label>}
          <Textarea
            value={action.card?.text ?? ''}
            onChange={(e) =>
              onChange({
                ...action,
                card: { ...action.card, text: e.target.value },
              })
            }
            rows={3}
            disabled={disabled}
          />
        </div>
      ) : null}

      {action.type === 'open_workflow' ? (
        <div className='flex flex-col gap-1.5'>
          {compact ? null : <Label>{t('menu.workflowList')}</Label>}
          <Select
            value={action.workflow_id ?? ''}
            onValueChange={(v) => onChange({ ...action, workflow_id: v })}
            disabled={disabled}
          >
            <SelectTrigger data-testid='workflow-select'>
              <SelectValue placeholder={t('menu.workflowList')} />
            </SelectTrigger>
            <SelectContent>
              {workflows.map((w) => (
                <SelectItem key={w.id} value={String(w.id)}>
                  {w.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      ) : null}

      {action.type === 'open_workflow' && action.workflow_id ? (
        <WorkflowInfoCard
          workflow={
            workflows.find((w) => String(w.id) === action.workflow_id) ?? {
              id: Number(action.workflow_id),
              name: t('menu.mapWorkflowMissing'),
            }
          }
        />
      ) : null}

      {action.type === 'list_tasks' ? (
        <ListTasksPreview channelId={channelId} />
      ) : null}

      {action.type === 'send_text' || action.type === 'copy_text' ? (
        <div className='flex flex-col gap-1.5'>
          {compact ? null : <Label>{t('menu.cardText')}</Label>}
          <Textarea
            value={action.text ?? ''}
            onChange={(e) => onChange({ ...action, text: e.target.value })}
            rows={3}
            disabled={disabled}
          />
        </div>
      ) : null}

      {action.type === 'send_media' ? (
        <div className='flex flex-col gap-1.5'>
          {compact ? null : <Label>{t('menu.cardMedia')}</Label>}
          <Textarea
            value={(action.media ?? []).map((m) => m.url).join('\n')}
            onChange={(e) =>
              onChange({
                ...action,
                media: e.target.value
                  .split('\n')
                  .map((url) => ({ kind: 'image' as const, url: url.trim() }))
                  .filter((m) => m.url),
              })
            }
            rows={3}
            disabled={disabled}
            placeholder='https://…'
          />
        </div>
      ) : null}

      {action.type === 'open_url' ? (
        <div className='flex flex-col gap-1.5'>
          {compact ? null : <Label>{t('menu.actionOpenUrl')}</Label>}
          <Input
            value={action.url ?? ''}
            onChange={(e) => onChange({ ...action, url: e.target.value })}
            disabled={disabled}
            placeholder='https://…'
          />
        </div>
      ) : null}
    </div>
  )
}
