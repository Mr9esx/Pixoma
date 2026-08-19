import { useTranslation } from 'react-i18next'
import type { Card, CardButton, Menu, MenuItem } from '@/lib/api/channel-menu'
import { cn } from '@/lib/utils'
import { buildTgFlow } from './lib/menu-flow'

type Props = {
  menu: Menu
  cards: Card[]
  path: string[]
  onSelectMenuItem: (item: MenuItem) => void
  onOpenCard: (card: Card, viaButton?: CardButton) => void
  onBack: () => void
}

export function PhoneSimulation({
  menu,
  cards,
  path,
  onSelectMenuItem,
  onOpenCard,
  onBack,
}: Props) {
  const { t } = useTranslation()
  const { keyboard, cardById } = buildTgFlow(menu, cards)
  const currentCardId = path[path.length - 1]
  const currentCard = currentCardId ? cardById.get(currentCardId) : undefined

  return (
    <div data-testid='phone-simulation' className='flex flex-col gap-3'>
      <div className='rounded-xl border bg-background p-3'>
        <div className='space-y-2 text-sm'>
          {currentCard ? (
            <>
              {currentCard.media.map((m, i) => (
                <div
                  key={i}
                  className='flex h-24 items-center justify-center rounded-md bg-muted text-xs text-muted-foreground'
                >
                  {m.kind}: {m.url}
                </div>
              ))}
              {currentCard.text ? (
                <p className='whitespace-pre-wrap'>{currentCard.text}</p>
              ) : null}
              {currentCard.buttons.length > 0 ? (
                <div className='flex flex-wrap gap-2 pt-1'>
                  {currentCard.buttons.map((b) => (
                    <button
                      key={b.id}
                      type='button'
                      onClick={() => onOpenCard(currentCard, b)}
                      className='rounded-md border border-cyan-500/40 bg-cyan-500/10 px-3 py-1.5 text-xs hover:bg-cyan-500/20'
                    >
                      {b.label}
                    </button>
                  ))}
                </div>
              ) : null}
              {path.length > 0 ? (
                <button
                  type='button'
                  onClick={onBack}
                  className='pt-1 text-xs text-muted-foreground hover:text-foreground'
                >
                  ‹ 返回
                </button>
              ) : null}
            </>
          ) : (
            <p className='text-muted-foreground'>
              欢迎使用，请选择功能 👇（{t('menu.cardList')}）
            </p>
          )}
        </div>
        <div className='mt-3 border-t pt-3'>
          {keyboard.map((row, i) => (
            <div key={i} className='mb-2 flex gap-2 last:mb-0'>
              {row.map((label) => {
                const item = menu.items.find((it) => it.label === label)
                return (
                  <button
                    key={label}
                    type='button'
                    onClick={() => item && onSelectMenuItem(item)}
                    className={cn(
                      'flex-1 rounded-md border px-2 py-2 text-sm',
                      'bg-muted/40 hover:bg-muted'
                    )}
                  >
                    {label}
                  </button>
                )
              })}
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
