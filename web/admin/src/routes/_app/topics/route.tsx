import { useEffect, useMemo } from 'react'
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
import { listTopics } from '@/lib/api/topics'
import { queryKeys } from '@/lib/api/query-keys'
import { Button } from '@/components/ui/button'
import { MasterDetailShell } from '@/components/master-detail/master-detail-shell'
import { TopicDetailPanel } from '@/features/topics/topic-detail-panel'
import { TopicListPanel } from '@/features/topics/topic-list-panel'

export const Route = createFileRoute('/_app/topics')({
  component: TopicsLayout,
})

function errorMessage(err: unknown): string | undefined {
  return err instanceof Error ? err.message : undefined
}

function TopicsLayout() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { key } = useParams({ strict: false }) as { key?: string }
  const pathname = useRouterState({ select: (s) => s.location.pathname })
  const locationState = useRouterState({
    select: (s) => s.location.state,
  }) as { backToList?: boolean } | undefined

  const listQuery = useQuery({
    queryKey: queryKeys.topics.all,
    queryFn: () => listTopics(),
  })
  const items = useMemo(() => listQuery.data ?? [], [listQuery.data])
  const backToList = locationState?.backToList === true
  const selectedKey = key ?? (backToList ? undefined : items[0]?.key)

  useEffect(() => {
    if (pathname.endsWith('/new')) return
    if (key == null && !backToList && items.length > 0) {
      void navigate({
        to: '/topics/$key',
        params: { key: items[0].key },
        replace: true,
      })
    }
  }, [key, backToList, items, navigate, pathname])

  if (pathname.endsWith('/new')) return <Outlet />

  return (
    <div
      data-layout='fixed'
      className='flex min-h-0 flex-1 flex-col gap-3 overflow-hidden'
      data-testid='topics-page'
    >
      <div className='flex shrink-0 flex-col gap-4 lg:flex-row lg:items-center lg:justify-between'>
        <div className='flex min-w-0 flex-col gap-[6px]'>
          <h1 className='truncate text-2xl leading-tight font-semibold tracking-tight'>
            {t('topics.title')}
          </h1>
          <p className='text-sm text-muted-foreground'>
            {t('topics.description')}
          </p>
        </div>
        <Button asChild>
          <Link to='/topics/new'>{t('topics.new')}</Link>
        </Button>
      </div>
      <MasterDetailShell
        className='md:grid-cols-[280px_1fr]'
        hasSelection={Boolean(selectedKey)}
        onBackToList={() => {
          void navigate({
            to: '/topics',
            state: { backToList: true },
          } as never)
        }}
        detailClassName='flex min-h-0 flex-col overflow-auto p-0'
        list={
          <TopicListPanel
            items={items}
            selectedKey={selectedKey}
            isLoading={listQuery.isLoading}
            isError={listQuery.isError}
            errorMessage={errorMessage(listQuery.error)}
            onRetry={() => void listQuery.refetch()}
          />
        }
        detail={selectedKey ? <TopicDetailPanel topicKey={selectedKey} /> : null}
      />
    </div>
  )
}
