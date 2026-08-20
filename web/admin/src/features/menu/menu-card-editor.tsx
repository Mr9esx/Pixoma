import { useMemo, useState } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { listCases } from '@/lib/api/cases'
import {
  createCard,
  deleteCard,
  getCardReferences,
  getMenu,
  listCards,
  putMenu,
  updateCard,
  type Action,
  type Card,
  type CardButton,
  type Menu,
  type MenuItem,
} from '@/lib/api/channel-menu'
import { queryKeys } from '@/lib/api/query-keys'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { ActionForm } from './action-form'
import { CardListPanel } from './card-list-panel'
import { validateMenuConfig } from './lib/menu-flow'
import { PhoneSimulation } from './phone-simulation'

const ERROR_KEYS: Record<string, string> = {
  label: 'menu.errLabel',
  card: 'menu.errCard',
  card_missing: 'menu.errCard',
  workflow: 'menu.errWorkflow',
  url: 'menu.errUrl',
  media: 'menu.errMedia',
}

const ACTION_KEY: Record<string, string> = {
  open_card: 'menu.actionOpenCard',
  open_workflow: 'menu.actionOpenWorkflow',
  send_text: 'menu.actionSendText',
  send_media: 'menu.actionSendMedia',
  open_url: 'menu.actionOpenUrl',
  copy_text: 'menu.actionCopyText',
  placeholder: 'menu.actionPlaceholder',
}

function emptyMenu(channelId: string): Menu {
  return { id: channelId, name: '', columns: 2, items: [] }
}

function uid(prefix: string): string {
  return `${prefix}-${Date.now()}-${Math.floor(Math.random() * 1000)}`
}

export function MenuCardEditor({ channelId }: { channelId: string }) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [menuDraft, setMenuDraft] = useState<Menu | null>(null)
  const [cardsDraft, setCardsDraft] = useState<Card[] | null>(null)
  const [selectedItemId, setSelectedItemId] = useState<string | null>(null)
  const [path, setPath] = useState<string[]>([])
  const [editingButton, setEditingButton] = useState<{
    cardId: string
    buttonId: string
  } | null>(null)
  const [saveErrors, setSaveErrors] = useState<string[] | null>(null)

  const menuQuery = useQuery({
    queryKey: queryKeys.channels.menu(channelId),
    queryFn: () => getMenu(channelId),
  })
  const cardsQuery = useQuery({
    queryKey: [...queryKeys.channels.menu(channelId), 'cards'] as const,
    queryFn: () => listCards(channelId),
  })
  const casesQuery = useQuery({
    queryKey: [...queryKeys.cases.all, { limit: 200 }] as const,
    queryFn: () => listCases({ limit: 200 }),
  })

  const menu = menuDraft ?? menuQuery.data ?? emptyMenu(channelId)
  const cards = cardsDraft ?? cardsQuery.data ?? []
  const selectedItem = menu.items.find((it) => it.id === selectedItemId)
  const currentCardId = path[path.length - 1]
  const currentCard = currentCardId
    ? cards.find((c) => c.id === currentCardId)
    : undefined
  const editingButtonObj =
    editingButton && currentCard
      ? currentCard.buttons.find((b) => b.id === editingButton.buttonId)
      : undefined

  const dirtyCount = useMemo(() => {
    const baseMenu = menuQuery.data ?? emptyMenu(channelId)
    const baseCards = cardsQuery.data ?? []
    let n = 0
    if (JSON.stringify(menu) !== JSON.stringify(baseMenu)) n += 1
    if (JSON.stringify(cards) !== JSON.stringify(baseCards)) n += 1
    return n
  }, [menu, cards, menuQuery.data, cardsQuery.data, channelId])

  function updateMenu(patch: Partial<Menu>) {
    setMenuDraft({ ...menu, ...patch })
  }

  function updateItem(itemId: string, patch: Partial<MenuItem>) {
    setMenuDraft({
      ...menu,
      items: menu.items.map((it) =>
        it.id === itemId ? { ...it, ...patch } : it
      ),
    })
  }

  function updateCardById(cardId: string, patch: Partial<Card>) {
    setCardsDraft(cards.map((c) => (c.id === cardId ? { ...c, ...patch } : c)))
  }

  function updateButton(
    cardId: string,
    buttonId: string,
    patch: Partial<CardButton>
  ) {
    setCardsDraft(
      cards.map((c) =>
        c.id === cardId
          ? {
              ...c,
              buttons: c.buttons.map((b) =>
                b.id === buttonId ? { ...b, ...patch } : b
              ),
            }
          : c
      )
    )
  }

  function addMenuItem() {
    const item: MenuItem = {
      id: uid('mi'),
      label: '',
      action: { type: 'placeholder' },
    }
    updateMenu({ items: [...menu.items, item] })
    setSelectedItemId(item.id)
  }

  function addCardButton(cardId: string) {
    const button: CardButton = {
      id: uid('cb'),
      label: '',
      action: { type: 'placeholder' },
    }
    const card = cards.find((c) => c.id === cardId)
    if (!card) return
    updateCardById(cardId, { buttons: [...card.buttons, button] })
    setEditingButton({ cardId, buttonId: button.id })
  }

  function openMenuItem(item: MenuItem) {
    setSelectedItemId(item.id)
    setEditingButton(null)
    if (item.action.type === 'open_card' && item.action.card_id) {
      setPath([item.action.card_id])
    } else {
      setPath([])
    }
  }

  function openCard(card: Card, viaButton?: CardButton) {
    if (
      viaButton &&
      viaButton.action.type === 'open_card' &&
      viaButton.action.card_id
    ) {
      setPath([...path, viaButton.action.card_id])
    } else if (!viaButton) {
      setPath([...path, card.id])
    }
    setEditingButton(null)
  }

  function createCardInPlace() {
    const id = uid('card')
    const card: Card = { id, name: '', media: [], text: '', buttons: [] }
    setCardsDraft([...cards, card])
    const action: Action = { type: 'open_card', card_id: id }
    if (editingButton) {
      updateButton(editingButton.cardId, editingButton.buttonId, {
        action: {
          ...(editingButtonObj?.action ?? { type: 'placeholder' }),
          ...action,
        },
      })
      setEditingButton({ cardId: id, buttonId: '' })
    } else if (selectedItemId) {
      const item = menu.items.find((it) => it.id === selectedItemId)
      if (item)
        updateItem(selectedItemId, { action: { ...item.action, ...action } })
    }
    setPath([...path, id])
  }

  function back() {
    setPath((prev) => prev.slice(0, -1))
    setEditingButton(null)
  }

  async function save() {
    const validation = validateMenuConfig(menu, cards)
    if (!validation.ok) {
      setSaveErrors([...new Set(validation.errors.map((e) => e.message))])
      return
    }
    setSaveErrors(null)
    await putMenu(channelId, { ...menu, id: menu.id || channelId })
    const existing = await listCards(channelId)
    const existingIds = new Set(existing.map((c) => c.id))
    for (const card of cards) {
      if (existingIds.has(card.id)) await updateCard(channelId, card)
      else await createCard(channelId, card)
    }
    for (const old of existing) {
      if (!cards.some((c) => c.id === old.id)) {
        const refs = await getCardReferences(channelId, old.id)
        if (refs.length === 0) await deleteCard(channelId, old.id)
      }
    }
    setMenuDraft(null)
    setCardsDraft(null)
    await queryClient.invalidateQueries({
      queryKey: queryKeys.channels.menu(channelId),
    })
    toast.success(t('common.successSaved'))
  }

  if (menuQuery.isLoading || cardsQuery.isLoading) {
    return <LoadingSkeleton rows={8} />
  }
  if (menuQuery.isError || cardsQuery.isError) {
    return <ErrorBanner message={t('common.errorGeneric')} />
  }

  const workflows = (casesQuery.data ?? []).map((c) => ({
    id: c.id,
    name: c.name,
  }))

  return (
    <div
      data-testid='menu-card-editor'
      className='flex min-h-0 flex-1 flex-col gap-3'
    >
      <div className='flex shrink-0 items-center justify-between gap-3'>
        <div className='min-w-0'>
          <h2 className='text-sm font-semibold'>{t('menu.mainKeyboard')}</h2>
          <p className='truncate text-xs text-muted-foreground'>
            {dirtyCount > 0
              ? t('menu.unsavedCount', { count: dirtyCount })
              : ''}
          </p>
        </div>
        <Button type='button' size='sm' onClick={() => void save()}>
          {t('common.save')}
        </Button>
      </div>

      {saveErrors && saveErrors.length > 0 ? (
        <div
          role='alert'
          className='rounded-md border border-red-600/30 bg-red-50 px-3 py-2 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400'
        >
          {saveErrors
            .map((e) => t(ERROR_KEYS[e] ?? 'menu.saveValidation'))
            .join(' · ')}
        </div>
      ) : null}

      <div className='grid min-h-0 flex-1 gap-3 md:grid-cols-[minmax(0,1fr)_440px]'>
        <div className='min-h-0 overflow-auto rounded-md border p-4'>
          <PhoneSimulation
            menu={menu}
            cards={cards}
            path={path}
            onSelectMenuItem={openMenuItem}
            onOpenCard={openCard}
            onBack={back}
          />
        </div>

        <div className='min-h-0 space-y-3 overflow-auto rounded-md border p-4'>
          {/* ⓪ 菜单 */}
          <section className='space-y-2 rounded-md border p-3'>
            <div className='flex items-center justify-between'>
              <h3 className='text-sm font-semibold'>
                {t('menu.mainKeyboard')}
              </h3>
              <Button
                type='button'
                size='sm'
                variant='outline'
                onClick={addMenuItem}
              >
                {t('menu.addMenuItem')}
              </Button>
            </div>
            <div className='flex items-center gap-2'>
              <Label>{t('menu.columnsPerRow')}</Label>
              <Input
                className='h-8 w-20'
                type='number'
                min={1}
                max={8}
                value={menu.columns}
                onChange={(e) =>
                  updateMenu({ columns: Number(e.target.value) || 2 })
                }
              />
            </div>
            <ul className='space-y-1'>
              {menu.items.map((it) => (
                <li key={it.id}>
                  <button
                    type='button'
                    onClick={() => openMenuItem(it)}
                    className='w-full rounded-md px-2 py-1.5 text-left text-sm hover:bg-muted'
                  >
                    {it.label.trim() || t('menu.untitled')}
                    <span className='ml-2 text-xs text-muted-foreground'>
                      {t(
                        ACTION_KEY[it.action.type] ?? 'menu.actionPlaceholder'
                      )}
                    </span>
                  </button>
                </li>
              ))}
            </ul>
          </section>

          {/* ① 菜单项 */}
          {selectedItem ? (
            <section className='space-y-2 rounded-md border p-3'>
              <h3 className='text-sm font-semibold'>
                {t('menu.menuItemLabel')}
              </h3>
              <Input
                value={selectedItem.label}
                onChange={(e) =>
                  updateItem(selectedItem.id, { label: e.target.value })
                }
              />
              <ActionForm
                action={selectedItem.action}
                cards={cards}
                workflows={workflows}
                onChange={(action) => updateItem(selectedItem.id, { action })}
                onCreateNewCard={createCardInPlace}
              />
            </section>
          ) : null}

          {/* ② 卡片 */}
          {currentCard ? (
            <section className='space-y-2 rounded-md border border-primary/40 p-3'>
              <div className='flex items-center justify-between'>
                <h3 className='text-sm font-semibold'>{t('menu.cardName')}</h3>
                <Button
                  type='button'
                  size='sm'
                  variant='outline'
                  onClick={() => addCardButton(currentCard.id)}
                >
                  {t('menu.addCardButton')}
                </Button>
              </div>
              <Input
                value={currentCard.name}
                onChange={(e) =>
                  updateCardById(currentCard.id, { name: e.target.value })
                }
                placeholder={t('menu.cardName')}
              />
              <Textarea
                value={currentCard.text}
                onChange={(e) =>
                  updateCardById(currentCard.id, { text: e.target.value })
                }
                placeholder={t('menu.cardText')}
                rows={3}
              />
              <ul className='space-y-1'>
                {currentCard.buttons.map((b) => (
                  <li key={b.id}>
                    <button
                      type='button'
                      onClick={() =>
                        setEditingButton({
                          cardId: currentCard.id,
                          buttonId: b.id,
                        })
                      }
                      className='w-full rounded-md px-2 py-1.5 text-left text-sm hover:bg-muted'
                    >
                      {b.label.trim() || t('menu.untitled')}
                      <span className='ml-2 text-xs text-muted-foreground'>
                        {t(
                          ACTION_KEY[b.action.type] ?? 'menu.actionPlaceholder'
                        )}
                      </span>
                    </button>
                  </li>
                ))}
              </ul>
            </section>
          ) : null}

          {/* ③ 卡片按钮 */}
          {editingButtonObj && editingButton ? (
            <section className='space-y-2 rounded-md border p-3'>
              <h3 className='text-sm font-semibold'>{t('menu.buttonLabel')}</h3>
              <Input
                value={editingButtonObj.label}
                onChange={(e) =>
                  updateButton(editingButton.cardId, editingButton.buttonId, {
                    label: e.target.value,
                  })
                }
              />
              <ActionForm
                action={editingButtonObj.action}
                cards={cards}
                workflows={workflows}
                onChange={(action) =>
                  updateButton(editingButton.cardId, editingButton.buttonId, {
                    action,
                  })
                }
                onCreateNewCard={createCardInPlace}
              />
            </section>
          ) : null}

          {/* 卡片清单（复用/查找） */}
          <section className='rounded-md border p-3'>
            <CardListPanel
              cards={cards}
              onSelect={(card) => {
                setPath([card.id])
                setEditingButton(null)
              }}
              onDelete={(card) =>
                setCardsDraft(cards.filter((c) => c.id !== card.id))
              }
            />
          </section>
        </div>
      </div>
    </div>
  )
}
