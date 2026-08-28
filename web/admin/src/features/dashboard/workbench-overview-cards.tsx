import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { listCases } from '@/lib/api/cases'
import { listEdges } from '@/lib/api/edges'
import { listFleetStats, listTaskDailyStats } from '@/lib/api/stats'
import { queryKeys } from '@/lib/api/query-keys'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { kit } from '@/features/edges/kit-classes'
import { pickFleetStats } from './task-stats-parse'

function errMessage(err: unknown): string | undefined {
  return err instanceof Error ? err.message : undefined
}

const CELL =
  'min-w-0 border-b border-border/70 p-3.5 sm:p-4 md:border-b-0 md:border-r last:border-r-0'

export function WorkbenchOverviewCards() {
  const { t } = useTranslation()
  const edges = useQuery({ queryKey: queryKeys.edges.all, queryFn: listEdges })
  const fleet = useQuery({
    queryKey: queryKeys.stats.fleet,
    queryFn: listFleetStats,
  })
  const cases = useQuery({
    queryKey: queryKeys.cases.all,
    queryFn: () => listCases(),
  })
  const today = new Date()
  const yearFrom = `${today.getFullYear()}-01-01`
  const yearTo = `${today.getFullYear()}-12-31`
  const daily = useQuery({
    queryKey: queryKeys.stats.tasksDaily(yearFrom, yearTo),
    queryFn: () => listTaskDailyStats({ from: yearFrom, to: yearTo }),
  })

  const edgeList = edges.data ?? []
  const enabled = edgeList.filter((e) => e.enabled).length
  const fleetData = pickFleetStats(fleet.data)
  const workflowCount = cases.data?.length ?? 0
  const processed = daily.data?.summary.processed ?? 0
  const successRate = daily.data?.summary.success_rate

  const isLoading =
    edges.isLoading || fleet.isLoading || cases.isLoading || daily.isLoading
  const isError =
    edges.isError || fleet.isError || cases.isError || daily.isError
  const error = edges.error ?? fleet.error ?? cases.error ?? daily.error

  return (
    <div data-testid='workbench-overview-cards' className={kit.statsWrap}>
      {isLoading ? (
        <div className='p-4'>
          <LoadingSkeleton rows={3} />
        </div>
      ) : isError ? (
        <div className='p-4'>
          <ErrorBanner message={errMessage(error)} onRetry={() => {
            void edges.refetch()
            void fleet.refetch()
            void cases.refetch()
            void daily.refetch()
          }} />
        </div>
      ) : (
        <div className='grid grid-cols-2 md:grid-cols-5'>
          <div className={CELL} data-testid='workbench-node-overview'>
            <p className={kit.statsLabel}>
              {t('dashboard.workbench.nodeOverviewTitle')}
            </p>
            <p className={kit.statsValue}>{enabled} / {edgeList.length}</p>
          </div>

          <div className={CELL} data-testid='workbench-avg-load'>
            <p className={kit.statsLabel}>
              {t('dashboard.workbench.avgLoadTitle')}
            </p>
            <p className={kit.statsValue}>
              {fleetData?.avg_cpu_usage_percent.toFixed(0) ?? '—'}%
            </p>
          </div>

          <div className={CELL} data-testid='workbench-workflow-count'>
            <p className={kit.statsLabel}>
              {t('dashboard.workbench.workflowCountTitle')}
            </p>
            <p className={kit.statsValue}>{workflowCount}</p>
          </div>

          <div className={CELL} data-testid='workbench-task-total'>
            <p className={kit.statsLabel}>
              {t('dashboard.workbench.taskTotalTitle')}
            </p>
            <p className={kit.statsValue}>{processed}</p>
          </div>

          <div className={CELL} data-testid='workbench-success-rate'>
            <p className={kit.statsLabel}>
              {t('dashboard.workbench.successRateTitle')}
            </p>
            <p className={kit.statsValue}>
              {successRate == null ? '—' : `${(successRate * 100).toFixed(0)}%`}
            </p>
          </div>
        </div>
      )}
    </div>
  )
}
