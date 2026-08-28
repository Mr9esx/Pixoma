import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { Cpu, Gauge, Server } from 'lucide-react'
import { listEdges } from '@/lib/api/edges'
import { listFleetStats } from '@/lib/api/stats'
import { queryKeys } from '@/lib/api/query-keys'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { kit } from '@/features/edges/kit-classes'
import { pickFleetStats } from './task-stats-parse'

function errMessage(err: unknown): string | undefined {
  return err instanceof Error ? err.message : undefined
}

export function WorkbenchOverviewCards() {
  const { t } = useTranslation()
  const edges = useQuery({ queryKey: queryKeys.edges.all, queryFn: listEdges })
  const fleet = useQuery({
    queryKey: queryKeys.stats.fleet,
    queryFn: listFleetStats,
  })
  const edgeList = edges.data ?? []
  const enabled = edgeList.filter((e) => e.enabled).length
  const fleetData = pickFleetStats(fleet.data)
  const topByUsage = (fleetData?.nodes ?? [])
    .slice()
    .sort((a, b) => b.cpu_usage_percent - a.cpu_usage_percent)
    .slice(0, 5)
  const isLoading = edges.isLoading || fleet.isLoading
  const isError = edges.isError || fleet.isError
  const error = edges.error ?? fleet.error

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
          }} />
        </div>
      ) : (
        <div className={kit.statsGrid}>
          <div className={kit.statsCell[0]} data-testid='workbench-node-overview'>
            <p className={kit.statsLabel}>
              <Server className='size-4' />
              {t('dashboard.workbench.nodeOverviewTitle')}
            </p>
            <p className={kit.statsValue}>
              {enabled} / {edgeList.length}
            </p>
          </div>

          <div className={kit.statsCell[1]} data-testid='workbench-avg-load'>
            <p className={kit.statsLabel}>
              <Gauge className='size-4' />
              {t('dashboard.workbench.avgLoadTitle')}
            </p>
            <p className={kit.statsValue}>
              {fleetData?.avg_cpu_usage_percent.toFixed(0) ?? '—'}%
            </p>
          </div>

          <div className={kit.statsCell[2]} data-testid='workbench-top-load'>
            <p className={kit.statsLabel}>
              <Cpu className='size-4' />
              {t('dashboard.workbench.topLoadTitle')}
            </p>
            <div className='mt-1.5 space-y-1 text-xs'>
              {topByUsage.length ? (
                topByUsage.map((n) => (
                  <div key={n.edge_id} className='flex items-center justify-between gap-2'>
                    <span className='truncate'>{n.edge_id}</span>
                    <span className='tabular-nums'>
                      {n.cpu_usage_percent.toFixed(0)}%
                    </span>
                  </div>
                ))
              ) : (
                <span className='text-muted-foreground'>{t('common.empty')}</span>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
