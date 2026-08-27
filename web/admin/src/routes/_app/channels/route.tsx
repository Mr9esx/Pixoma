import { useEffect, useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  createFileRoute,
  useNavigate,
  useParams,
  useRouterState,
} from '@tanstack/react-router'
import { Plus, Radio } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { listChannels } from '@/lib/api/channels'
import { queryKeys } from '@/lib/api/query-keys'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty'
import { MasterDetailShell } from '@/components/master-detail/master-detail-shell'
import { ChannelDetailPanel } from '@/features/channels/channel-detail-panel'
import { ChannelListPanel } from '@/features/channels/channel-list-panel'
import { CreateChannelForm } from '@/features/channels/create-channel-form'
import { kit } from '@/features/edges/kit-classes'

export const Route = createFileRoute('/_app/channels')({
  component: ChannelsLayout,
})

function errorMessage(err: unknown): string | undefined {
  return err instanceof Error ? err.message : undefined
}

function ChannelsLayout() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { id } = useParams({ strict: false }) as { id?: string }
  const locationState = useRouterState({
    select: (s) => s.location.state,
  }) as { backToList?: boolean } | undefined

  const listQuery = useQuery({
    queryKey: queryKeys.channels.all,
    queryFn: listChannels,
  })
  const items = useMemo(() => listQuery.data ?? [], [listQuery.data])
  const backToList = locationState?.backToList === true
  const selectedId =
    id && id !== 'new' ? id : backToList ? undefined : items[0]?.id
  const [createOpen, setCreateOpen] = useState(false)
  const isEmpty =
    !listQuery.isLoading && !listQuery.isError && items.length === 0

  useEffect(() => {
    if (id == null && !backToList && items.length > 0) {
      void navigate({
        to: '/channels/$id',
        params: { id: items[0].id },
        replace: true,
      })
    }
  }, [id, backToList, items, navigate])

  return (
    <div
      data-layout='fixed'
      className='flex min-h-0 flex-1 flex-col gap-3 overflow-hidden'
      data-testid='channels-page'
    >
      <div className='flex shrink-0 flex-col gap-4 lg:flex-row lg:items-center lg:justify-between'>
        <div className='flex min-w-0 flex-col gap-[6px]'>
          <h1 className='truncate text-2xl leading-tight font-semibold tracking-tight'>
            {t('channels.title')}
          </h1>
          <p className='text-sm text-muted-foreground'>
            {t('channels.description')}
          </p>
        </div>
        {!isEmpty ? (
          <Button
            className={kit.btnPrimary}
            onClick={() => setCreateOpen(true)}
          >
            <Plus className='size-3.5' />
            {t('channels.new')}
          </Button>
        ) : null}
      </div>
      <MasterDetailShell
        className='md:grid-cols-[280px_1fr]'
        hasSelection={Boolean(selectedId)}
        onBackToList={() => {
          void navigate({
            to: '/channels',
            state: { backToList: true },
          } as never)
        }}
        detailClassName='flex min-h-0 flex-col overflow-auto p-0'
        list={
          <ChannelListPanel
            items={items}
            selectedId={selectedId}
            isLoading={listQuery.isLoading}
            isError={listQuery.isError}
            errorMessage={errorMessage(listQuery.error)}
            onRetry={() => void listQuery.refetch()}
          />
        }
        detail={selectedId ? <ChannelDetailPanel id={selectedId} /> : null}
        emptyDetail={
          !listQuery.isLoading && !listQuery.isError && items.length === 0 ? (
            <Empty>
              <EmptyHeader className='max-w-none'>
                <EmptyMedia variant='icon'>
                  <Radio />
                </EmptyMedia>
                <EmptyTitle className='text-sm font-medium'>
                  {t('channels.empty')}
                </EmptyTitle>
                <EmptyDescription>{t('channels.emptyDesc')}</EmptyDescription>
              </EmptyHeader>
              <EmptyContent className='flex-row justify-center gap-2'>
                <Button
                  className={kit.btnPrimary}
                  onClick={() => setCreateOpen(true)}
                >
                  {t('channels.new')}
                </Button>
              </EmptyContent>
            </Empty>
          ) : undefined
        }
      />
      <Dialog open={createOpen} onOpenChange={setCreateOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t('channels.new')}</DialogTitle>
          </DialogHeader>
          <CreateChannelForm
            onDone={(ch) => {
              setCreateOpen(false)
              void navigate({
                to: '/channels/$id',
                params: { id: ch.id },
              })
            }}
            onCancel={() => setCreateOpen(false)}
          />
        </DialogContent>
      </Dialog>
    </div>
  )
}
