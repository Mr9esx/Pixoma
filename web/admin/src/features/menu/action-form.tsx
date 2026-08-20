import { useTranslation } from 'react-i18next'
import type { Action, ActionType, Card } from '@/lib/api/channel-menu'
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
import { Textarea } from '@/components/ui/textarea'
import { CardPicker } from './card-picker'

const ACTION_KEYS: Record<ActionType, string> = {
  open_card: 'menu.actionOpenCard',
  open_workflow: 'menu.actionOpenWorkflow',
  send_text: 'menu.actionSendText',
  send_media: 'menu.actionSendMedia',
  open_url: 'menu.actionOpenUrl',
  copy_text: 'menu.actionCopyText',
  placeholder: 'menu.actionPlaceholder',
}

type Props = {
  action: Action
  cards: Card[]
  workflows: { id: number; name: string }[]
  onChange: (next: Action) => void
  onCreateNewCard?: () => void
  disabled?: boolean
}

export function ActionForm({
  action,
  cards,
  workflows,
  onChange,
  onCreateNewCard,
  disabled,
}: Props) {
  const { t } = useTranslation()
  const types = Object.keys(ACTION_KEYS) as ActionType[]

  return (
    <div data-testid='action-form' className='space-y-3'>
      <div className='space-y-1.5'>
        <Label>{t('menu.buttonAction')}</Label>
        <Select
          value={action.type}
          onValueChange={(type) => onChange({ type: type as ActionType })}
          disabled={disabled}
        >
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {types.map((type) => (
              <SelectItem key={type} value={type}>
                {t(ACTION_KEYS[type])}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {action.type === 'open_card' ? (
        <CardPicker
          cards={cards}
          value={action.card_id}
          onPick={(cardId) => onChange({ ...action, card_id: cardId })}
          onCreateNew={onCreateNewCard}
          disabled={disabled}
        />
      ) : null}

      {action.type === 'open_workflow' ? (
        <div className='space-y-1.5'>
          <Label>{t('menu.workflowList')}</Label>
          <div className='max-h-44 space-y-1.5 overflow-auto rounded-md border p-2'>
            {workflows.map((w) => (
              <label key={w.id} className='flex items-center gap-2 text-sm'>
                <Checkbox
                  checked={(action.workflow_ids ?? []).includes(w.id)}
                  onCheckedChange={(v) =>
                    onChange({
                      ...action,
                      workflow_ids: v
                        ? [...(action.workflow_ids ?? []), w.id]
                        : (action.workflow_ids ?? []).filter(
                            (id) => id !== w.id
                          ),
                    })
                  }
                  disabled={disabled}
                />
                <span>{w.name}</span>
              </label>
            ))}
          </div>
        </div>
      ) : null}

      {action.type === 'send_text' || action.type === 'copy_text' ? (
        <div className='space-y-1.5'>
          <Label>{t('menu.cardText')}</Label>
          <Textarea
            value={action.text ?? ''}
            onChange={(e) => onChange({ ...action, text: e.target.value })}
            rows={3}
            disabled={disabled}
          />
        </div>
      ) : null}

      {action.type === 'send_media' ? (
        <div className='space-y-1.5'>
          <Label>{t('menu.cardMedia')}</Label>
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
        <div className='space-y-1.5'>
          <Label>{t('menu.actionOpenUrl')}</Label>
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
