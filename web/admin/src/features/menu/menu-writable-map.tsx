import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import type { Action, Card, CardButton, Menu } from '@/lib/api/channel-menu'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
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
import { FilterSegment } from '@/components/filters/filter-segment'
import { ActionForm } from './action-form'
import { buildNewCardDraft, validateNewCardDraft } from './card-draft'
import {
  actionIsBroken,
  clampMenuColumns,
  createBlankCardButton,
  mediaKindLabelKey,
  orphanCards,
  pushTrailButton,
  sliceTrail,
  trailFromItem,
  trailFromOrphan,
  type MapTrailStep,
} from './lib/menu-flow'
import { MenuMapLayout } from './menu-map-layout'
import type { WorkflowRef } from './node-view'

const COLUMN_COUNTS = ['1', '2', '3', '4', '5', '6', '7', '8'] as const

function cardFromStep(step: MapTrailStep, cards: Card[]): Card | undefined {
  if (step.kind === 'orphan') return cards.find((c) => c.id === step.id)
  if (step.action.type === 'open_card' && step.action.card_id) {
    return cards.find((c) => c.id === step.action.card_id)
  }
  return undefined
}

function ownerCard(cards: Card[], buttonId: string): Card | undefined {
  return cards.find((c) => c.buttons.some((b) => b.id === buttonId))
}

function MediaThumbs({
  media,
  t,
}: {
  media: { kind: string; url: string; caption?: string }[]
  t: (key: string) => string
}) {
  if (media.length === 0) return null
  return (
    <div
      className={cn(
        'grid gap-px bg-border',
        media.length === 1 ? 'grid-cols-1' : 'grid-cols-2'
      )}
    >
      {media.map((m, i) => (
        <div
          key={`${m.url}-${i}`}
          className='grid aspect-video place-items-center bg-muted text-sm text-muted-foreground'
        >
          {m.url ? (
            <img src={m.url} alt='' className='size-full object-cover' />
          ) : (
            t(`menu.${mediaKindLabelKey(m.kind)}`)
          )}
        </div>
      ))}
    </div>
  )
}

function paneLabel(text: string) {
  return (
    <p className='mb-3 font-mono text-sm font-semibold tracking-wider text-muted-foreground uppercase'>
      {text}
    </p>
  )
}

export function MenuWritableMap({
  menu,
  cards,
  workflows,
  trail,
  onTrailChange,
  onMenuChange,
  onCardsChange,
  onColumnsChange,
  onAddKey,
}: {
  menu: Menu
  cards: Card[]
  workflows: WorkflowRef[]
  trail: MapTrailStep[]
  onTrailChange: (next: MapTrailStep[]) => void
  onMenuChange: (next: Menu) => void
  onCardsChange: (next: Card[]) => void
  onColumnsChange: (columns: number) => void
  onAddKey: () => void
}) {
  const { t } = useTranslation()
  const [creatingCard, setCreatingCard] = useState(false)
  const [newCardName, setNewCardName] = useState('')
  const cols = clampMenuColumns(menu.columns)
  const orphans = orphanCards(menu, cards)
  const last = trail[trail.length - 1]
  const selectedItemId = trail[0]?.kind === 'item' ? trail[0].id : null
  const selectedOrphanId = trail[0]?.kind === 'orphan' ? trail[0].id : null
  const selectedButtonId = last?.kind === 'btn' ? last.id : null
  const crumbRoot =
    trail[0]?.kind === 'orphan'
      ? t('menu.mapCrumbOrphan')
      : t('menu.mainKeyboard')
  const editingCard = last ? cardFromStep(last, cards) : undefined

  function patchLastAction(action: Action) {
    if (!last) return
    if (last.kind === 'item') {
      onMenuChange({
        ...menu,
        items: menu.items.map((it) =>
          it.id === last.id ? { ...it, action } : it
        ),
      })
    } else if (last.kind === 'btn') {
      const owner = ownerCard(cards, last.id)
      if (!owner) return
      onCardsChange(
        cards.map((c) =>
          c.id === owner.id
            ? {
                ...c,
                buttons: c.buttons.map((b) =>
                  b.id === last.id ? { ...b, action } : b
                ),
              }
            : c
        )
      )
    }
    onTrailChange(
      trail.map((s, i) => (i === trail.length - 1 ? { ...s, action } : s))
    )
  }

  function patchItemLabel(itemId: string, label: string) {
    onMenuChange({
      ...menu,
      items: menu.items.map((it) => (it.id === itemId ? { ...it, label } : it)),
    })
    onTrailChange(
      trail.map((s) =>
        s.kind === 'item' && s.id === itemId ? { ...s, label } : s
      )
    )
  }

  function patchButtonLabel(buttonId: string, label: string) {
    const owner = ownerCard(cards, buttonId)
    if (!owner) return
    onCardsChange(
      cards.map((c) =>
        c.id === owner.id
          ? {
              ...c,
              buttons: c.buttons.map((b) =>
                b.id === buttonId ? { ...b, label } : b
              ),
            }
          : c
      )
    )
    onTrailChange(
      trail.map((s) =>
        s.kind === 'btn' && s.id === buttonId ? { ...s, label } : s
      )
    )
  }

  function patchCard(next: Card) {
    onCardsChange(cards.map((c) => (c.id === next.id ? next : c)))
    onTrailChange(
      trail.map((s) =>
        s.kind === 'orphan' && s.id === next.id ? { ...s, label: next.name } : s
      )
    )
  }

  function commitNewCard() {
    if (!last || last.kind === 'orphan') return
    const draft = { name: newCardName, text: '' }
    if (!validateNewCardDraft(draft).ok) return
    const card = buildNewCardDraft(draft)
    const action: Action = { type: 'open_card', card_id: card.id }
    if (last.kind === 'item') {
      onCardsChange([...cards, card])
      onMenuChange({
        ...menu,
        items: menu.items.map((it) =>
          it.id === last.id ? { ...it, action } : it
        ),
      })
    } else {
      const owner = ownerCard(cards, last.id)
      if (!owner) return
      onCardsChange(
        [...cards, card].map((c) =>
          c.id === owner.id
            ? {
                ...c,
                buttons: c.buttons.map((b) =>
                  b.id === last.id ? { ...b, action } : b
                ),
              }
            : c
        )
      )
    }
    onTrailChange(
      trail.map((s, i) => (i === trail.length - 1 ? { ...s, action } : s))
    )
    setCreatingCard(false)
    setNewCardName('')
  }

  const selectedItem =
    last?.kind === 'item'
      ? menu.items.find((it) => it.id === last.id)
      : undefined
  const selectedButton =
    last?.kind === 'btn'
      ? ownerCard(cards, last.id)?.buttons.find((b) => b.id === last.id)
      : undefined

  return (
    <MenuMapLayout
      className='min-h-0 flex-1 rounded-none border-0'
      keyboard={
        <>
          {paneLabel(t('menu.mainKeyboard'))}
          <div className='mb-4 flex flex-col gap-1.5'>
            <Label className='text-sm text-muted-foreground'>
              {t('menu.columnCount')}
            </Label>
            <FilterSegment
              value={String(cols)}
              options={COLUMN_COUNTS.map((n) => ({ value: n, label: n }))}
              onValueChange={(n) => onColumnsChange(Number(n))}
              aria-label={t('menu.columnCount')}
              data-testid='menu-columns'
            />
          </div>
          <div
            data-testid='map-keyboard'
            className='grid gap-2'
            style={{ gridTemplateColumns: `repeat(${cols}, minmax(0, 1fr))` }}
          >
            {menu.items.length === 0 ? (
              <p className='m-0 text-sm text-muted-foreground'>
                {t('menu.mapEmpty')}
              </p>
            ) : (
              menu.items.map((it) => {
                const broken = actionIsBroken(it.action, cards, workflows)
                return (
                  <Button
                    key={it.id}
                    type='button'
                    data-testid='map-key'
                    aria-pressed={it.id === selectedItemId}
                    variant='outline'
                    className={cn(
                      'h-11 truncate rounded-md px-3 text-left text-sm font-semibold',
                      'hover:border-foreground/35',
                      it.id === selectedItemId && 'border-foreground bg-muted',
                      broken && 'border-destructive'
                    )}
                    onClick={() => onTrailChange(trailFromItem(it))}
                  >
                    {it.label || t('menu.untitled')}
                  </Button>
                )
              })
            )}
            <Button
              type='button'
              data-testid='add-key'
              variant='outline'
              onClick={onAddKey}
              className='flex h-11 flex-1 items-center justify-center rounded-md border-dashed text-sm text-muted-foreground'
            >
              {t('menu.addKey')}
            </Button>
          </div>
        </>
      }
      path={
        <>
          {paneLabel(t('menu.mapPath'))}
          {last ? (
            <div className='flex flex-col gap-4'>
              <div className='flex flex-wrap items-center gap-1 text-sm text-muted-foreground'>
                <span>{crumbRoot}</span>
                {trail.map((step, i) => (
                  <span
                    key={`${step.kind}-${step.id}-${i}`}
                    className='contents'
                  >
                    <span className='opacity-50'>/</span>
                    {i === trail.length - 1 ? (
                      <span className='font-medium text-foreground'>
                        {step.label || t('menu.untitled')}
                      </span>
                    ) : (
                      <Button
                        type='button'
                        variant='ghost'
                        size='sm'
                        className='h-auto rounded-none bg-transparent p-0 text-sm text-muted-foreground hover:bg-transparent hover:text-foreground'
                        onClick={() => onTrailChange(sliceTrail(trail, i))}
                      >
                        {step.label || t('menu.untitled')}
                      </Button>
                    )}
                  </span>
                ))}
              </div>

              {selectedItem ? (
                <div className='flex flex-col gap-1.5'>
                  <Label htmlFor='map-key-label'>{t('menu.keyName')}</Label>
                  <Input
                    id='map-key-label'
                    data-testid='map-key-label'
                    value={selectedItem.label}
                    onChange={(e) =>
                      patchItemLabel(selectedItem.id, e.target.value)
                    }
                    className='min-h-11'
                  />
                </div>
              ) : null}

              {selectedButton ? (
                <div className='flex flex-col gap-1.5'>
                  <Label htmlFor='map-button-label'>
                    {t('menu.buttonLabel')}
                  </Label>
                  <Input
                    id='map-button-label'
                    value={selectedButton.label}
                    onChange={(e) =>
                      patchButtonLabel(selectedButton.id, e.target.value)
                    }
                    className='min-h-11'
                  />
                </div>
              ) : null}

              {last.kind === 'orphan' ? null : (
                <div className='flex flex-col gap-2'>
                  <Label>{t('menu.actionRow')}</Label>
                  <ActionForm
                    action={last.action}
                    cards={cards}
                    workflows={workflows}
                    onChange={patchLastAction}
                    onCreateNewCard={() => setCreatingCard(true)}
                    compact
                  />
                  {creatingCard && last.action.type === 'open_card' ? (
                    <div className='flex flex-col gap-2'>
                      <Input
                        data-testid='new-card-name'
                        value={newCardName}
                        onChange={(e) => setNewCardName(e.target.value)}
                        placeholder={t('menu.newCardName')}
                        className='min-h-11'
                      />
                      <Button
                        type='button'
                        size='sm'
                        data-testid='new-card-confirm'
                        onClick={commitNewCard}
                      >
                        {t('menu.newCardConfirm')}
                      </Button>
                    </div>
                  ) : null}
                </div>
              )}

              {last.action.type === 'open_card' &&
              last.action.card_id &&
              !editingCard ? (
                <p className='m-0 text-sm text-destructive'>
                  {t('menu.mapCardMissing')}
                </p>
              ) : null}

              {editingCard ? (
                <EditCardMessage
                  card={editingCard}
                  selectedButtonId={selectedButtonId}
                  t={t}
                  onChange={patchCard}
                  onPushButton={(b) => onTrailChange(pushTrailButton(trail, b))}
                  onDeleteCard={() => {
                    onCardsChange(cards.filter((c) => c.id !== editingCard.id))
                    onTrailChange([])
                  }}
                />
              ) : null}

              {last.kind === 'item' ? (
                <div>
                  <Button
                    type='button'
                    size='sm'
                    variant='ghost'
                    data-testid='delete-menu-item'
                    onClick={() => {
                      onMenuChange({
                        ...menu,
                        items: menu.items.filter((it) => it.id !== last.id),
                      })
                      onTrailChange([])
                    }}
                  >
                    {t('menu.deleteKey')}
                  </Button>
                </div>
              ) : null}

              {last.kind === 'btn' ? (
                <div>
                  <Button
                    type='button'
                    size='sm'
                    variant='ghost'
                    data-testid='delete-button'
                    onClick={() => {
                      const owner = ownerCard(cards, last.id)
                      if (!owner) return
                      onCardsChange(
                        cards.map((c) =>
                          c.id === owner.id
                            ? {
                                ...c,
                                buttons: c.buttons.filter(
                                  (b) => b.id !== last.id
                                ),
                              }
                            : c
                        )
                      )
                      onTrailChange(sliceTrail(trail, trail.length - 2))
                    }}
                  >
                    {t('menu.deleteButton')}
                  </Button>
                </div>
              ) : null}
            </div>
          ) : null}
        </>
      }
      orphans={
        orphans.length > 0 ? (
          <div
            data-testid='map-orphans'
            className='shrink-0 border-t border-border px-4 py-3 pb-4'
          >
            {paneLabel(t('menu.mapOrphans', { n: orphans.length }))}
            <div className='flex flex-wrap gap-2'>
              {orphans.map((c) => (
                <Button
                  key={c.id}
                  type='button'
                  aria-pressed={c.id === selectedOrphanId}
                  variant='outline'
                  className={cn(
                    'h-11 rounded-md border-dashed px-3 text-sm',
                    'hover:border-solid hover:border-foreground',
                    c.id === selectedOrphanId &&
                      'border-solid border-foreground bg-muted'
                  )}
                  onClick={() => onTrailChange(trailFromOrphan(c))}
                >
                  {c.name}
                </Button>
              ))}
            </div>
          </div>
        ) : null
      }
    />
  )
}

function EditCardMessage({
  card,
  selectedButtonId,
  t,
  onChange,
  onPushButton,
  onDeleteCard,
}: {
  card: Card
  selectedButtonId: string | null
  t: (key: string, opts?: { name?: string }) => string
  onChange: (next: Card) => void
  onPushButton: (button: CardButton) => void
  onDeleteCard: () => void
}) {
  return (
    <div data-testid='edit-card' className='flex flex-col gap-3'>
      <div className='flex flex-col gap-1.5'>
        <Label htmlFor='edit-card-name'>{t('menu.cardName')}</Label>
        <Input
          id='edit-card-name'
          value={card.name}
          onChange={(e) => onChange({ ...card, name: e.target.value })}
          className='min-h-11'
        />
      </div>
      <div className='flex flex-col gap-1.5'>
        <Label htmlFor='edit-card-text'>{t('menu.cardText')}</Label>
        <Textarea
          id='edit-card-text'
          value={card.text}
          onChange={(e) => onChange({ ...card, text: e.target.value })}
          rows={3}
        />
      </div>
      <div className='flex flex-col gap-1.5'>
        <Label>{t('menu.cardMedia')}</Label>
        {(card.media ?? []).map((m, i) => (
          <div key={`${m.url}-${i}`} className='flex gap-2'>
            <Input
              value={m.url}
              onChange={(e) => {
                const media = (card.media ?? []).slice()
                media[i] = { ...media[i], url: e.target.value }
                onChange({ ...card, media })
              }}
              placeholder='https://…'
              className='min-h-11'
            />
            <Select
              value={m.kind || 'image'}
              onValueChange={(kind) => {
                const media = (card.media ?? []).slice()
                media[i] = { ...media[i], kind }
                onChange({ ...card, media })
              }}
            >
              <SelectTrigger className='min-h-11 w-28'>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value='image'>{t('menu.mapThumbImage')}</SelectItem>
                <SelectItem value='video'>{t('menu.mapThumbVideo')}</SelectItem>
                <SelectItem value='animation'>
                  {t('menu.mapThumbAnimation')}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
        ))}
        <Button
          type='button'
          size='sm'
          variant='outline'
          onClick={() =>
            onChange({
              ...card,
              media: [...(card.media ?? []), { kind: 'image', url: '' }],
            })
          }
        >
          {t('menu.addMedia')}
        </Button>
      </div>
      <div className='overflow-hidden rounded-lg border border-border bg-background'>
        <MediaThumbs media={card.media ?? []} t={t} />
        <div className='flex flex-col gap-2 px-3.5 py-3.5'>
          <Label>{t('menu.cardButtons')}</Label>
          <div className='flex flex-wrap gap-2'>
            {card.buttons.map((b) => (
              <Button
                key={b.id}
                type='button'
                data-testid='edit-card-button'
                variant='outline'
                className={cn(
                  'h-11 rounded-md px-3 font-medium',
                  'hover:border-foreground',
                  b.id === selectedButtonId && 'border-foreground bg-muted'
                )}
                onClick={() => onPushButton(b)}
              >
                {b.label || t('menu.untitled')}
              </Button>
            ))}
            <Button
              type='button'
              size='sm'
              variant='outline'
              data-testid='add-button'
              className='min-h-11'
              onClick={() =>
                onChange({
                  ...card,
                  buttons: [
                    ...card.buttons,
                    createBlankCardButton(
                      `b-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 6)}`
                    ),
                  ],
                })
              }
            >
              {t('menu.addCardButton')}
            </Button>
          </div>
        </div>
      </div>
      <div>
        <Button
          type='button'
          variant='ghost'
          data-testid='delete-card'
          onClick={onDeleteCard}
        >
          {t('menu.deleteCard')}
        </Button>
      </div>
    </div>
  )
}
