import { useEffect, useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  createFileRoute,
  Link,
  useNavigate,
  useParams,
  useRouterState,
} from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { Boxes, Plus } from 'lucide-react'
import { listCases } from '@/lib/api/cases'
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
import { CaseForm } from '@/features/cases/case-form'
import { CaseDetailPanel } from '@/features/cases/detail-panel'
import {
  CaseListPanel,
  type CaseListFilters,
} from '@/features/cases/list-panel'
import { kit } from '@/features/edges/kit-classes'

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
  const listParams = {
    q: filters.q.trim() || undefined,
  }

  const listQuery = useQuery({
    queryKey: [...queryKeys.cases.all, listParams] as const,
    queryFn: () => listCases(listParams),
  })
  const items = useMemo(() => listQuery.data ?? [], [listQuery.data])
  const backToList = locationState?.backToList === true
  const selectedId = useMemo(() => {
    if (caseId == null || caseId === 'new') {
      return backToList ? undefined : items[0]?.id
    }
    const n = Number(caseId)
    return Number.isNaN(n) ? undefined : n
  }, [caseId, backToList, items])

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
          <h1 className='truncate text-2xl leading-tight font-semibold tracking-tight'>
            {t('cases.title')}
          </h1>
          <p className='text-sm text-muted-foreground'>
            {t('cases.description')}
          </p>
        </div>
        <Button asChild className={kit.btnPrimary}>
          <Link to='/cases/$caseId' params={{ caseId: 'new' }}>
            <Plus className='size-3.5' />
            {t('cases.createHeading')}
          </Link>
        </Button>
      </div>
      <MasterDetailShell
        className='md:grid-cols-[280px_minmax(0,1fr)]'
        hasSelection={Boolean(selectedId) || caseId === 'new'}
        onBackToList={() => {
          void navigate({ to: '/cases', state: { backToList: true } } as never)
        }}
        detailClassName='flex min-h-0 flex-col overflow-auto p-0'
        list={
          <CaseListPanel
            items={items}
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
          caseId === 'new' ? (
            <div className={kit.pageSection}>
              <h2 className={kit.title}>
                {t('cases.createHeading')}
              </h2>
              <CaseForm mode='create' />
            </div>
          ) : selectedId ? (
            <CaseDetailPanel id={selectedId} />
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
                <EmptyDescription>
                  {t('cases.emptyDesc')}
                </EmptyDescription>
              </EmptyHeader>
              <EmptyContent className='flex-row justify-center gap-2'>
                <Button asChild className={kit.btnPrimary}>
                  <Link to='/cases/$caseId' params={{ caseId: 'new' }}>
                    {t('common.create')}
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
