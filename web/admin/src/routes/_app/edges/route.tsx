import { useEffect, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  createFileRoute,
  Link,
  useNavigate,
  useParams,
} from '@tanstack/react-router'
import { Plus, Server } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { listEdges, listPresence } from '@/lib/api/edges'
import { queryKeys } from '@/lib/api/query-keys'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent } from '@/components/ui/dialog'
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty'
import { MasterDetailShell } from '@/components/master-detail/master-detail-shell'
import { CreateEdgeWizard } from '@/features/edges/create-edge-wizard'
import { EdgeDetailPanel } from '@/features/edges/detail-panel'
import { kit } from '@/features/edges/kit-classes'
import { EdgeListPanel } from '@/features/edges/list-panel'

export const Route = createFileRoute('/_app/edges')({
  component: EdgesLayout,
})

function errorMessage(err: unknown): string | undefined {
  return err instanceof Error ? err.message : undefined
}

function EdgesLayout() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { edgeId } = useParams({ strict: false }) as {
    edgeId?: string
  }
  const [createOpen, setCreateOpen] = useState(false)
  const [backToList, setBackToList] = useState(false)

  const listQuery = useQuery({
    queryKey: queryKeys.edges.all,
    queryFn: listEdges,
  })
  const presenceQuery = useQuery({
    queryKey: queryKeys.edges.presence,
    queryFn: listPresence,
    refetchInterval: 5000,
  })
  const presenceById = Object.fromEntries(
    (presenceQuery.data ?? []).map((row) => [row.id, row])
  )
  const items = listQuery.data ?? []
  const selectedId = edgeId ?? (backToList ? undefined : items[0]?.id)

  useEffect(() => {
    if (edgeId != null) setBackToList(false)
  }, [edgeId])

  useEffect(() => {
    if (edgeId == null && !backToList && items.length > 0) {
      void navigate({
        to: '/edges/$edgeId',
        params: { edgeId: items[0].id },
        replace: true,
      })
    }
  }, [edgeId, backToList, items, navigate])

  return (
    <div
      data-layout='fixed'
      className='flex min-h-0 flex-1 flex-col gap-3 overflow-hidden'
      data-testid='edges-page'
    >
      <div className='flex shrink-0 flex-col gap-4 lg:flex-row lg:items-center lg:justify-between'>
        <div className='flex min-w-0 flex-col gap-[6px]'>
          <h1 className='truncate text-2xl leading-tight font-semibold tracking-tight'>
            {t('edges.title')}
          </h1>
          <p className='text-sm text-muted-foreground'>
            {t('edges.description')}
          </p>
        </div>
        <Button
          type='button'
          className={kit.btnPrimary}
          onClick={() => setCreateOpen(true)}
        >
          <Plus className='size-3.5' />
          {t('edges.createNode')}
        </Button>
      </div>
      <MasterDetailShell
        className='md:grid-cols-[280px_1fr]'
        hasSelection={Boolean(selectedId)}
        onBackToList={() => {
          setBackToList(true)
          void navigate({ to: '/edges' })
        }}
        detailClassName='flex min-h-0 flex-col overflow-auto p-0'
        list={
          <EdgeListPanel
            items={items}
            presenceById={presenceById}
            selectedId={selectedId}
            isLoading={listQuery.isLoading}
            isError={listQuery.isError}
            errorMessage={errorMessage(listQuery.error)}
            onRetry={() => void listQuery.refetch()}
          />
        }
        detail={selectedId ? <EdgeDetailPanel id={selectedId} /> : null}
        emptyDetail={
          !listQuery.isLoading && !listQuery.isError && items.length === 0 ? (
            <Empty>
              <EmptyHeader className='max-w-none'>
                <EmptyMedia variant='icon'>
                  <Server />
                </EmptyMedia>
                <EmptyTitle className='text-sm font-medium'>
                  {t('edges.empty')}
                </EmptyTitle>
                <EmptyDescription>
                  {t('edges.emptyDesc')}
                </EmptyDescription>
              </EmptyHeader>
              <EmptyContent className='flex-row justify-center gap-2'>
                <Button
                  type='button'
                  className={kit.btnPrimary}
                  onClick={() => setCreateOpen(true)}
                >
                  {t('edges.createNode')}
                </Button>
                <Button
                  asChild
                  variant='outline'
                  className='h-8 gap-1.5 rounded-md px-3 text-xs'
                >
                  <Link to='/quick-config'>{t('menu.quickConfig')}</Link>
                </Button>
              </EmptyContent>
            </Empty>
          ) : undefined
        }
      />
      <Dialog open={createOpen} onOpenChange={setCreateOpen}>
        <DialogContent className='flex max-h-[85vh] flex-col sm:max-w-lg'>
          <CreateEdgeWizard
            onDone={(created, action) => {
              setCreateOpen(false)
              if (action === 'view') {
                void navigate({
                  to: '/edges/$edgeId',
                  params: { edgeId: created.id },
                })
              }
            }}
          />
        </DialogContent>
      </Dialog>
    </div>
  )
}
