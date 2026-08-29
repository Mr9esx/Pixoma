import { useTranslation } from 'react-i18next'
import type { Card } from '@/lib/api/channel-menu'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

type Props = {
  cards: Card[]
  value?: string
  onPick: (cardId: string) => void
  onCreateNew?: () => void
  disabled?: boolean
  compact?: boolean
}

export function CardPicker({
  cards,
  value,
  onPick,
  onCreateNew,
  disabled,
  compact,
}: Props) {
  const { t } = useTranslation()
  return (
    <div
      data-testid='card-picker'
      className={compact ? 'flex items-center gap-2' : 'flex flex-col gap-1.5'}
    >
      {compact ? null : <Label>{t('menu.targetCard')}</Label>}
      <Select value={value ?? ''} onValueChange={onPick} disabled={disabled}>
        <SelectTrigger className={compact ? 'min-h-11 flex-1' : undefined}>
          <SelectValue placeholder={t('menu.pickExistingCard')} />
        </SelectTrigger>
        <SelectContent>
          {cards.map((c) => (
            <SelectItem key={c.id} value={c.id}>
              {c.name || c.id}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      {onCreateNew ? (
        <Button
          type='button'
          size='sm'
          variant='outline'
          onClick={onCreateNew}
          disabled={disabled}
        >
          {t('menu.newCard')}
        </Button>
      ) : null}
    </div>
  )
}
