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
}

export function CardPicker({
  cards,
  value,
  onPick,
  onCreateNew,
  disabled,
}: Props) {
  const { t } = useTranslation()
  return (
    <div data-testid='card-picker' className='space-y-1.5'>
      <Label>{t('menu.targetCard')}</Label>
      <Select value={value ?? ''} onValueChange={onPick} disabled={disabled}>
        <SelectTrigger>
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
