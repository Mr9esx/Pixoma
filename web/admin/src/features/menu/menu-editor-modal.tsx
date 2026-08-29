import { useMemo, useState } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { listCases } from '@/lib/api/cases'
import {
  createCard,
  deleteCard,
  getCardReferences,
  listCards,
  putMenu,
  updateCard,
  type Card,
  type Menu,
} from '@/lib/api/channel-menu'
import { queryKeys } from '@/lib/api/query-keys'
import { Alert, AlertDescription } from '@/components/ui/alert'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  clampMenuColumns,
  createBlankMenuItem,
  isMenuDraftDirty,
  trailFromItem,
  validateMenuConfig,
  type MapTrailStep,
} from './lib/menu-flow'
import { MenuWritableMap } from './menu-writable-map'
import { toWorkflowRef, type WorkflowRef } from './node-view'

const ERROR_KEYS: Record<string, string> = {
  label: 'menu.errLabel',
  card: 'menu.errCard',
  card_missing: 'menu.errCard',
  workflow: 'menu.errWorkflow',
  url: 'menu.errUrl',
  media: 'menu.errMedia',
}

function uid(prefix: string): string {
  return `${prefix}-${Date.now()}-${Math.floor(Math.random() * 1000)}`
}

type Props = {
  open: boolean
  onOpenChange: (open: boolean) => void
  channelId: string
  savedMenu: Menu
  savedCards: Card[]
  onSaved: () => void
}

/** 编辑菜单弹层：可写能力地图。 */
export function MenuEditorModal({
  open,
  onOpenChange,
  channelId,
  savedMenu,
  savedCards,
  onSaved,
}: Props) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()

  const [menuDraft, setMenuDraft] = useState<Menu | null>(null)
  const [cardsDraft, setCardsDraft] = useState<Card[] | null>(null)
  const [trail, setTrail] = useState<MapTrailStep[]>([])
  const [saveErrors, setSaveErrors] = useState<string[] | null>(null)
  const [discardOpen, setDiscardOpen] = useState(false)

  const casesQuery = useQuery({
    queryKey: [...queryKeys.cases.all, { limit: 200 }] as const,
    queryFn: () => listCases({ limit: 200 }),
    enabled: open,
  })

  const baseMenu: Menu = menuDraft ?? savedMenu
  const baseCards: Card[] = useMemo(
    () =>
      (cardsDraft ?? savedCards).map((c) => ({
        ...c,
        media: c.media ?? [],
        buttons: c.buttons ?? [],
      })),
    [cardsDraft, savedCards]
  )
  const workflows: WorkflowRef[] = (casesQuery.data ?? []).map((c) =>
    toWorkflowRef(c)
  )
  const dirty = isMenuDraftDirty(
    { menu: savedMenu, cards: savedCards },
    { menu: baseMenu, cards: baseCards }
  )

  function resetDrafts() {
    setMenuDraft(null)
    setCardsDraft(null)
    setTrail([])
    setSaveErrors(null)
    setDiscardOpen(false)
  }

  async function save() {
    const menu =
      baseMenu.id || channelId
        ? { ...baseMenu, id: baseMenu.id || channelId }
        : baseMenu
    const validation = validateMenuConfig(menu, baseCards)
    if (!validation.ok) {
      setSaveErrors([...new Set(validation.errors.map((e) => e.message))])
      return
    }
    setSaveErrors(null)
    try {
      await putMenu(channelId, menu)
      const existing = await listCards(channelId)
      const existingIds = new Set(existing.map((c) => c.id))
      for (const card of baseCards) {
        if (existingIds.has(card.id)) await updateCard(channelId, card)
        else await createCard(channelId, card)
      }
      for (const old of existing) {
        if (!baseCards.some((c) => c.id === old.id)) {
          const refs = await getCardReferences(channelId, old.id)
          if (refs.length === 0) await deleteCard(channelId, old.id)
        }
      }
      await queryClient.invalidateQueries({
        queryKey: queryKeys.channels.menu(channelId),
      })
      toast.success(t('common.successSaved'))
      onSaved()
      onOpenChange(false)
      resetDrafts()
    } catch {
      toast.error(t('common.errorGeneric'))
    }
  }

  function handleOpenChange(next: boolean) {
    if (!next && dirty) {
      setDiscardOpen(true)
      return
    }
    if (!next) resetDrafts()
    onOpenChange(next)
  }

  function confirmDiscard() {
    resetDrafts()
    onOpenChange(false)
  }

  return (
    <>
      <Dialog open={open} onOpenChange={handleOpenChange}>
        <DialogContent
          data-testid='menu-editor-modal'
          className='!inset-0 !top-0 !left-0 flex h-svh !max-h-svh w-svw !max-w-none !translate-x-0 !translate-y-0 flex-col gap-0 rounded-none p-0 sm:!max-w-none'
        >
          <DialogHeader className='shrink-0 border-b px-4 py-3'>
            <div className='flex items-center justify-between gap-3 pr-8'>
              <div className='flex min-w-0 items-center gap-3'>
                <DialogTitle>{t('menu.editMenu')}</DialogTitle>
                <DialogDescription
                  className={
                    dirty ? 'text-sm text-muted-foreground' : 'sr-only'
                  }
                >
                  {dirty ? t('menu.unsaved') : t('menu.editMenu')}
                </DialogDescription>
              </div>
              <Button
                type='button'
                size='sm'
                onClick={() => void save()}
                data-testid='save-menu'
              >
                {t('common.save')}
              </Button>
            </div>
          </DialogHeader>

          {saveErrors && saveErrors.length > 0 ? (
            <div className='p-3'>
              <Alert variant='destructive'>
                <AlertDescription>
                  {saveErrors
                    .map((e) => t(ERROR_KEYS[e] ?? 'menu.saveValidation'))
                    .join(' · ')}
                </AlertDescription>
              </Alert>
            </div>
          ) : null}

          <div className='flex min-h-0 flex-1 flex-col overflow-hidden'>
            <MenuWritableMap
              menu={baseMenu}
              cards={baseCards}
              workflows={workflows}
              trail={trail}
              onTrailChange={setTrail}
              onMenuChange={setMenuDraft}
              onCardsChange={setCardsDraft}
              onColumnsChange={(columns) =>
                setMenuDraft({
                  ...baseMenu,
                  columns: clampMenuColumns(columns),
                })
              }
              onAddKey={() => {
                const item = createBlankMenuItem(uid('mi'))
                setMenuDraft({ ...baseMenu, items: [...baseMenu.items, item] })
                setTrail(trailFromItem(item))
              }}
            />
          </div>
        </DialogContent>
      </Dialog>

      <AlertDialog open={discardOpen} onOpenChange={setDiscardOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('menu.discardEdits')}</AlertDialogTitle>
            <AlertDialogDescription className='sr-only'>
              {t('menu.discardEdits')}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t('common.cancel')}</AlertDialogCancel>
            <AlertDialogAction onClick={confirmDiscard}>
              {t('menu.discard')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
