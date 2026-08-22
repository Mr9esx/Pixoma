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
import { listCases } from '@/lib/api/cases'
import { queryKeys } from '@/lib/api/query-keys'
import { Button } from '@/components/ui/button'
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
        <Button asChild>
          <Link to='/cases/$caseId' params={{ caseId: 'new' }}>
            {t('common.create')}
          </Link>
        </Button>
      </div>
      <MasterDetailShell
        className='md:grid-cols-[280px_minmax(0,1fr)]'
        hasSelection={Boolean(selectedId)}
        onBackToList={() => {
          void navigate({ to: '/cases', state: { backToList: true } } as never)
        }}
        detailClassName='flex min-h-0 flex-col p-0'
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
            <div className={`${kit.pageSection} min-h-0 flex-1 overflow-auto`}>
              <h2 className='text-lg font-semibold'>
                {t('cases.createHeading')}
              </h2>
              <CaseForm mode='create' />
            </div>
          ) : selectedId ? (
            <CaseDetailPanel id={selectedId} />
          ) : null
        }
      />
    </div>
  )
}
