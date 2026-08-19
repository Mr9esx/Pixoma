import { useQuery } from '@tanstack/react-query'
import {
  createFileRoute,
  Link,
  Outlet,
  useNavigate,
  useParams,
  useRouterState,
} from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { listChannels } from '@/lib/api/channels'
import { queryKeys } from '@/lib/api/query-keys'
import { Button } from '@/components/ui/button'
import { MasterDetailShell } from '@/components/master-detail/master-detail-shell'
import { ChannelDetailPanel } from '@/features/channels/channel-detail-panel'
import { ChannelListPanel } from '@/features/channels/channel-list-panel'

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
  const pathname = useRouterState({ select: (s) => s.location.pathname })

  const listQuery = useQuery({
    queryKey: queryKeys.channels.all,
    queryFn: listChannels,
  })

  if (pathname.endsWith('/new')) {
    return <Outlet />
  }

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
        <Button asChild>
          <Link to='/channels/new'>{t('channels.new')}</Link>
        </Button>
      </div>
      <MasterDetailShell
        className='md:grid-cols-[280px_1fr]'
        hasSelection={Boolean(id)}
        onBackToList={() => void navigate({ to: '/channels' })}
        detailClassName='flex min-h-0 flex-col overflow-auto p-0'
        list={
          <ChannelListPanel
            items={listQuery.data ?? []}
            selectedId={id}
            isLoading={listQuery.isLoading}
            isError={listQuery.isError}
            errorMessage={errorMessage(listQuery.error)}
            onRetry={() => void listQuery.refetch()}
          />
        }
        detail={id ? <ChannelDetailPanel id={id} /> : null}
      />
    </div>
  )
}
