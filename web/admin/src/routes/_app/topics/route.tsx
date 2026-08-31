import { useEffect, useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  createFileRoute,
  useNavigate,
  useParams,
  useRouterState,
} from '@tanstack/react-router'
import { Plus, Tags } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { listCases } from '@/lib/api/cases'
import { listEdges, listPresence } from '@/lib/api/edges'
import { queryKeys } from '@/lib/api/query-keys'
import { listTopics } from '@/lib/api/topics'
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
import { kit } from '@/features/edges/kit-classes'
import { topicReferences } from '@/features/link-health/lib/references'
import { CreateTopicForm } from '@/features/topics/create-topic-form'
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
  const locationState = useRouterState({
    select: (s) => s.location.state,
  }) as { backToList?: boolean } | undefined

  const listQuery = useQuery({
    queryKey: queryKeys.topics.all,
    queryFn: () => listTopics(),
  })
  const casesQuery = useQuery({
    queryKey: queryKeys.cases.all,
    queryFn: () => listCases(),
  })
  const edgesQuery = useQuery({
    queryKey: queryKeys.edges.all,
    queryFn: listEdges,
  })
  const presenceQuery = useQuery({
    queryKey: queryKeys.edges.presence,
    queryFn: listPresence,
  })
  const [manualCreateOpen, setManualCreateOpen] = useState(false)
  const items = useMemo(() => listQuery.data ?? [], [listQuery.data])
  const listHealthReady =
    casesQuery.isSuccess && edgesQuery.isSuccess && presenceQuery.isSuccess
  const healthByTopic = useMemo(() => {
    if (!listHealthReady) return undefined
    const cases = casesQuery.data ?? []
    const edges = edgesQuery.data ?? []
    const presence = presenceQuery.data ?? []

    return Object.fromEntries(
      items.map((topic) => [
        topic.key,
        topicReferences(topic.key, { cases, edges, presence }).health,
      ])
    )
  }, [
    casesQuery.data,
    edgesQuery.data,
    items,
    listHealthReady,
    presenceQuery.data,
  ])
  const backToList = locationState?.backToList === true
  const selectedKey = key ?? (backToList ? undefined : items[0]?.key)
  const create = key === 'new'
  const createOpen = manualCreateOpen || create
  const closeCreateDialog = () => {
    setManualCreateOpen(false)
    if (create) {
      void navigate({
        to: '/topics',
        state: { backToList: true },
      } as never)
    }
  }
  const isEmpty =
    !listQuery.isLoading && !listQuery.isError && items.length === 0

  useEffect(() => {
    if (key == null && !backToList && items.length > 0) {
      void navigate({
        to: '/topics/$key',
        params: { key: items[0].key },
        replace: true,
      })
    }
  }, [key, backToList, items, navigate])
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
        {!isEmpty ? (
          <Button
            className={kit.btnPrimary}
            onClick={() => setManualCreateOpen(true)}
          >
            <Plus className='size-3.5' />
            {t('topics.new')}
          </Button>
        ) : null}
      </div>
      <MasterDetailShell
        className='md:grid-cols-[280px_1fr] @min-[1408px]/page:grid-cols-[300px_1fr]'
        hasSelection={Boolean(selectedKey) || key === 'new'}
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
            healthByTopic={healthByTopic}
            healthReady={listHealthReady}
            selectedKey={selectedKey}
            isLoading={listQuery.isLoading}
            isError={listQuery.isError}
            errorMessage={errorMessage(listQuery.error)}
            onRetry={() => void listQuery.refetch()}
          />
        }
        detail={
          selectedKey ? (
            <TopicDetailPanel key={selectedKey} topicKey={selectedKey} />
          ) : null
        }
        emptyDetail={
          !listQuery.isLoading && !listQuery.isError && items.length === 0 ? (
            <Empty>
              <EmptyHeader className='max-w-none'>
                <EmptyMedia variant='icon'>
                  <Tags />
                </EmptyMedia>
                <EmptyTitle className='text-sm font-medium'>
                  {t('topics.empty')}
                </EmptyTitle>
                <EmptyDescription>{t('topics.emptyDesc')}</EmptyDescription>
              </EmptyHeader>
              <EmptyContent className='flex-row justify-center gap-2'>
                <Button
                  className={kit.btnPrimary}
                  onClick={() => setManualCreateOpen(true)}
                >
                  {t('topics.new')}
                </Button>
              </EmptyContent>
            </Empty>
          ) : undefined
        }
      />
      <Dialog
        open={createOpen}
        onOpenChange={(open) => {
          if (!open) closeCreateDialog()
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t('topics.new')}</DialogTitle>
          </DialogHeader>
          <CreateTopicForm
            onDone={(topic) => {
              setManualCreateOpen(false)
              void navigate({
                to: '/topics/$key',
                params: { key: topic.key },
              })
            }}
            onCancel={closeCreateDialog}
          />
        </DialogContent>
      </Dialog>
    </div>
  )
}
