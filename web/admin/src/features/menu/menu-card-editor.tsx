import { useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { listCases } from '@/lib/api/cases'
import {
  getMenu,
  listCards,
  type Card,
  type Menu,
} from '@/lib/api/channel-menu'
import { queryKeys } from '@/lib/api/query-keys'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { MenuCapabilityMap } from './menu-capability-map'
import { MenuEditorModal } from './menu-editor-modal'
import { toWorkflowRef, type WorkflowRef } from './node-view'

function emptyMenu(channelId: string): Menu {
  return { id: channelId, name: '', columns: 2, items: [] }
}

/** 主视图 = 只读能力地图。编辑只在「改菜单」弹层里。 */
export function MenuCardEditor({ channelId }: { channelId: string }) {
  const { t } = useTranslation()

  const [editorOpen, setEditorOpen] = useState(false)
  const [mapEpoch, setMapEpoch] = useState(0)

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

  const menu: Menu = menuQuery.data ?? emptyMenu(channelId)
  const cards: Card[] = useMemo(
    () =>
      (cardsQuery.data ?? []).map((c) => ({
        ...c,
        media: c.media ?? [],
        buttons: c.buttons ?? [],
      })),
    [cardsQuery.data]
  )
  const workflows: WorkflowRef[] = useMemo(
    () => (casesQuery.data ?? []).map((c) => toWorkflowRef(c)),
    [casesQuery.data]
  )

  if (menuQuery.isLoading || cardsQuery.isLoading) {
    return <LoadingSkeleton rows={8} />
  }
  if (menuQuery.isError || cardsQuery.isError) {
    return <ErrorBanner message={t('menu.mapLoadFailed')} />
  }

  return (
    <div data-testid='menu-card-editor'>
      <MenuCapabilityMap
        key={mapEpoch}
        menu={menu}
        cards={cards}
        workflows={workflows}
        onEdit={() => setEditorOpen(true)}
      />
      <MenuEditorModal
        open={editorOpen}
        onOpenChange={setEditorOpen}
        channelId={channelId}
        savedMenu={menu}
        savedCards={cards}
        onSaved={() => {
          setMapEpoch((n) => n + 1)
        }}
      />
    </div>
  )
}
