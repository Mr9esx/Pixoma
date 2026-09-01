import { useEffect, useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  createFileRoute,
  Link,
  useNavigate,
  useParams,
  useRouterState,
} from '@tanstack/react-router'
import { ArrowLeft, Boxes, Plus } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { listCases } from '@/lib/api/cases'
import { getCaseMenuPlacements } from '@/lib/api/channel-menu'
import { listEdges, listPresence } from '@/lib/api/edges'
import { queryKeys } from '@/lib/api/query-keys'
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
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty'
import { PixomaLoading } from '@/components/feedback/pixoma-loading'
import { MasterDetailShell } from '@/components/master-detail/master-detail-shell'
import { CaseForm } from '@/features/cases/case-form'
import { CaseDetailPanel } from '@/features/cases/detail-panel'
import {
  CaseListPanel,
  type CaseListFilters,
} from '@/features/cases/list-panel'
import { WorkflowImportRequirement } from '@/features/cases/workflow-import-requirement'
import { kit } from '@/features/edges/kit-classes'
import { caseReferences } from '@/features/link-health/lib/references'

export const Route = createFileRoute('/_app/cases')({
  component: CasesLayout,
})

function errorMessage(err: unknown): string | undefined {
  return err instanceof Error ? err.message : undefined
}

function CasesLayout() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { caseId } = useParams({ strict: false }) as { caseId?: string }
  const locationState = useRouterState({
    select: (s) => s.location.state,
  }) as { backToList?: boolean } | undefined

  const [filters, setFilters] = useState<CaseListFilters>({
    q: '',
  })
  const [createPending, setCreatePending] = useState(false)
  const [dirty, setDirty] = useState(false)
  const [leaveTarget, setLeaveTarget] = useState<'back' | 'cancel' | null>(null)
  const listParams = {
    q: filters.q.trim() || undefined,
  }

  const listQuery = useQuery({
    queryKey: [...queryKeys.cases.all, listParams] as const,
    queryFn: () => listCases(listParams),
  })
  const items = useMemo(() => listQuery.data ?? [], [listQuery.data])
  const edgesQuery = useQuery({
    queryKey: queryKeys.edges.all,
    queryFn: listEdges,
  })
  const presenceQuery = useQuery({
    queryKey: queryKeys.edges.presence,
    queryFn: listPresence,
  })
  const placementsQuery = useQuery({
    queryKey: [
      ...queryKeys.cases.all,
      'menu-placements',
      items.map((item) => item.id),
    ] as const,
    queryFn: async () =>
      Promise.all(items.map((item) => getCaseMenuPlacements(item.id))),
    enabled: items.length > 0,
  })
  const listHealthReady =
    edgesQuery.isSuccess && presenceQuery.isSuccess && placementsQuery.isSuccess
  const placementsByCase = useMemo(() => {
    return Object.fromEntries(
      items.map((item, index) => [item.id, placementsQuery.data?.[index] ?? []])
    )
  }, [items, placementsQuery.data])
  const healthByCase = useMemo(() => {
    if (!listHealthReady) return undefined
    const edges = edgesQuery.data ?? []
    const presence = presenceQuery.data ?? []

    return Object.fromEntries(
      items.map((item) => [
        item.id,
        caseReferences(item.id, {
          cases: [item],
          edges,
          presence,
          placements: placementsByCase[item.id],
        }).health,
      ])
    )
  }, [
    edgesQuery.data,
    items,
    listHealthReady,
    placementsByCase,
    presenceQuery.data,
  ])
  const backToList = locationState?.backToList === true
  const selectedId = useMemo(() => {
    if (caseId == null || caseId === 'new') {
      return backToList ? undefined : items[0]?.id
    }
    const n = Number(caseId)
    return Number.isNaN(n) ? undefined : n
  }, [caseId, backToList, items])
  const isEmpty =
    !listQuery.isLoading && !listQuery.isError && items.length === 0

  function requestLeave(target: 'back' | 'cancel') {
    if (dirty) {
      setLeaveTarget(target)
      return
    }
    void navigate({ to: '/cases' })
  }

  useEffect(() => {
    if (caseId == null && !backToList && items.length > 0) {
      void navigate({
        to: '/cases/$caseId',
        params: { caseId: String(items[0].id) },
        replace: true,
      })
    }
  }, [caseId, backToList, items, navigate])

  return (
    <div
      data-layout='fixed'
      className='flex min-h-0 flex-1 flex-col gap-3 overflow-hidden'
      data-testid='cases-page'
    >
      <div className='flex shrink-0 flex-col gap-4 lg:flex-row lg:items-center lg:justify-between'>
        <div className='flex min-w-0 flex-col gap-[6px]'>
          <div className='flex min-w-0 items-center gap-2'>
            {caseId === 'new' ? (
              <button
                type='button'
                onClick={() => requestLeave('back')}
                title={t('common.backToList')}
                aria-label={t('common.backToList')}
                className='inline-flex size-10 shrink-0 items-center justify-center rounded-md text-muted-foreground hover:bg-accent hover:text-foreground'
              >
                <ArrowLeft className='size-4' />
              </button>
            ) : null}
            <h1 className='truncate text-2xl leading-tight font-semibold tracking-tight'>
              {caseId === 'new' ? t('cases.createHeading') : t('cases.title')}
            </h1>
          </div>
          {caseId === 'new' ? null : (
            <p className='text-sm text-muted-foreground'>
              {t('cases.description')}
            </p>
          )}
        </div>
        {caseId !== 'new' && !isEmpty ? (
          <Button asChild className={kit.btnPrimary}>
            <Link to='/cases/$caseId' params={{ caseId: 'new' }}>
              <Plus className='size-3.5' />
              {t('cases.createHeading')}
            </Link>
          </Button>
        ) : null}
      </div>
      {caseId === 'new' ? (
        <div
          data-layout='fixed'
          data-testid='cases-create-page'
          className='flex min-h-0 flex-1 flex-col overflow-hidden rounded-md border'
        >
          <div className='min-h-0 flex-1 overflow-auto px-5 py-4'>
            <CaseForm
              mode='create'
              splitPane
              hideActions
              stepRail
              leftIntro={<WorkflowImportRequirement />}
              onPendingChange={setCreatePending}
              onDirtyChange={setDirty}
              formId='create-case-form'
            />
          </div>
          <footer className='flex shrink-0 flex-wrap items-center gap-2 border-t bg-card px-5 py-3'>
            <Button
              type='button'
              variant='outline'
              disabled={createPending}
              onClick={() => requestLeave('cancel')}
            >
              {t('common.cancel')}
            </Button>
            <Button
              type='submit'
              form='create-case-form'
              disabled={createPending}
            >
              {createPending ? <PixomaLoading /> : null}
              {t('common.create')}
            </Button>
          </footer>
          <AlertDialog
            open={leaveTarget !== null}
            onOpenChange={(open) => {
              if (!open) setLeaveTarget(null)
            }}
          >
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>{t('cases.unsavedTitle')}</AlertDialogTitle>
                <AlertDialogDescription>
                  {t('cases.unsavedBody')}
                </AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel
                  type='button'
                  onClick={() => setLeaveTarget(null)}
                >
                  {t('cases.unsavedKeepEditing')}
                </AlertDialogCancel>
                <AlertDialogAction
                  type='button'
                  onClick={() => void navigate({ to: '/cases' })}
                >
                  {t('cases.unsavedDiscard')}
                </AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
        </div>
      ) : (
        <MasterDetailShell
          className='md:grid-cols-[280px_minmax(0,1fr)] @min-[1408px]/page:grid-cols-[300px_minmax(0,1fr)]'
          hasSelection={Boolean(selectedId)}
          onBackToList={() => {
            void navigate({
              to: '/cases',
              state: { backToList: true },
            } as never)
          }}
          detailClassName='flex min-h-0 flex-col overflow-auto p-0'
          list={
            <CaseListPanel
              items={items}
              healthByCase={healthByCase}
              healthReady={listHealthReady}
              selectedId={selectedId}
              filters={filters}
              onFiltersChange={setFilters}
              isLoading={listQuery.isLoading}
              isError={listQuery.isError}
              errorMessage={errorMessage(listQuery.error)}
              onRetry={() => void listQuery.refetch()}
            />
          }
          detail={
            selectedId ? (
              <CaseDetailPanel key={selectedId} id={selectedId} />
            ) : null
          }
          emptyDetail={
            !listQuery.isLoading && !listQuery.isError && items.length === 0 ? (
              <Empty>
                <EmptyHeader className='max-w-none'>
                  <EmptyMedia variant='icon'>
                    <Boxes />
                  </EmptyMedia>
                  <EmptyTitle className='text-sm font-medium'>
                    {t('cases.empty')}
                  </EmptyTitle>
                  <EmptyDescription>{t('cases.emptyDesc')}</EmptyDescription>
                </EmptyHeader>
                <EmptyContent className='flex-row justify-center gap-2'>
                  <Button asChild className={kit.btnPrimary}>
                    <Link to='/cases/$caseId' params={{ caseId: 'new' }}>
                      {t('cases.createHeading')}
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
      )}
    </div>
  )
}
