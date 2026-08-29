import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import type { Card } from '@/lib/api/channel-menu'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

type Props = {
  cards: Card[]
  onSelect: (card: Card) => void
  onDelete: (card: Card) => void
}

export function CardListPanel({ cards, onSelect, onDelete }: Props) {
  const { t } = useTranslation()
  const [q, setQ] = useState('')
  const needle = q.trim().toLowerCase()
  const filtered = cards.filter(
    (c) =>
      !needle || c.name.toLowerCase().includes(needle) || c.id.includes(needle)
  )

  return (
    <div data-testid='card-list-panel' className='space-y-2'>
      <Input
        value={q}
        onChange={(e) => setQ(e.target.value)}
        placeholder={t('menu.cardList')}
        autoComplete='off'
      />
      <ul className='space-y-1'>
        {filtered.map((card) => (
          <li key={card.id} className='flex items-center gap-2 text-sm'>
            <button
              type='button'
              onClick={() => onSelect(card)}
              className='min-w-0 flex-1 truncate rounded-md px-2 py-1.5 text-left hover:bg-muted'
            >
              {card.name || card.id}
              <span className='ml-2 text-sm text-muted-foreground'>
                {card.id}
              </span>
            </button>
            <Button
              type='button'
              size='sm'
              variant='ghost'
              onClick={() => onDelete(card)}
            >
              {t('common.delete')}
            </Button>
          </li>
        ))}
      </ul>
    </div>
  )
}
