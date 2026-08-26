import { useEffect, useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  createFileRoute,
  Link,
  useNavigate,
  useParams,
  useRouterState,
} from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { Plus, Radio } from 'lucide-react'
import { listChannels } from '@/lib/api/channels'
import { queryKeys } from '@/lib/api/query-keys'
import { Button } from '@/components/ui/button'
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
  const selectedId = id ?? (backToList ? undefined : items[0]?.id)
  const create = id === 'new'

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
        <Button asChild className={kit.btnPrimary}>
          <Link to='/channels/$id' params={{ id: 'new' }}>
            <Plus className='size-3.5' />
            {t('channels.new')}
          </Link>
        </Button>
      </div>
      <MasterDetailShell
        className='md:grid-cols-[280px_1fr]'
        hasSelection={Boolean(selectedId) || id === 'new'}
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
        detail={
          create ? (
            <div className={kit.createPage}>
              <h2 className={kit.title}>{t('channels.new')}</h2>
              <CreateChannelForm
                onDone={(ch) => {
                  void navigate({
                    to: '/channels/$id',
                    params: { id: ch.id },
                  })
                }}
                onCancel={() => void navigate({ to: '/channels' })}
              />
            </div>
          ) : selectedId ? (
            <ChannelDetailPanel id={selectedId} />
          ) : null
        }
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
                <EmptyDescription>
                  {t('channels.emptyDesc')}
                </EmptyDescription>
              </EmptyHeader>
              <EmptyContent className='flex-row justify-center gap-2'>
                <Button asChild className={kit.btnPrimary}>
                  <Link to='/channels/$id' params={{ id: 'new' }}>
                  {t('channels.new')}
                  </Link>
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
    </div>
  )
}
