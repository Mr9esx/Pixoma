import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { listEdges } from '@/lib/api/edges'
import { listFleetStats } from '@/lib/api/stats'
import { queryKeys } from '@/lib/api/query-keys'
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { pickFleetStats } from './task-stats-parse'

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

  return (
    <div className='grid gap-4 sm:grid-cols-3'>
      <Card data-testid='workbench-node-overview'>
        <CardHeader className='pb-2'>
          <CardTitle className='text-sm font-medium'>
            {t('dashboard.workbench.nodeOverviewTitle')}
          </CardTitle>
        </CardHeader>
        <CardContent>
          {edges.isLoading ? (
            <LoadingSkeleton rows={2} />
          ) : edges.isError ? (
            <ErrorBanner
              message={edges.error instanceof Error ? edges.error.message : undefined}
              onRetry={() => void edges.refetch()}
            />
          ) : (
            <>
              <div className='text-2xl font-bold'>
                {enabled}
                <span className='text-base font-normal text-muted-foreground'>
                  {' '}
                  / {edgeList.length}
                </span>
              </div>
              <div className='mt-2 h-2 w-full overflow-hidden rounded-full bg-muted'>
                <div
                  className='h-full rounded-full bg-emerald-500'
                  style={{
                    width: edgeList.length
                      ? `${(enabled / edgeList.length) * 100}%`
                      : '0%',
                  }}
                />
              </div>
            </>
          )}
        </CardContent>
      </Card>

      <Card data-testid='workbench-avg-load'>
        <CardHeader className='pb-2'>
          <CardTitle className='text-sm font-medium'>
            {t('dashboard.workbench.avgLoadTitle')}
          </CardTitle>
        </CardHeader>
        <CardContent>
          {fleet.isLoading ? (
            <LoadingSkeleton rows={3} />
          ) : fleet.isError ? (
            <ErrorBanner
              message={fleet.error instanceof Error ? fleet.error.message : undefined}
              onRetry={() => void fleet.refetch()}
            />
          ) : fleetData ? (
            <div className='space-y-1 text-sm'>
              <div className='flex justify-between'>
                <span>{t('dashboard.avgCpu')}</span>
                <b className='tabular-nums'>
                  {fleetData.avg_cpu_usage_percent.toFixed(0)}%
                </b>
              </div>
              <div className='flex justify-between'>
                <span>{t('dashboard.avgMem')}</span>
                <b className='tabular-nums'>
                  {fleetData.avg_mem_usage_percent.toFixed(0)}%
                </b>
              </div>
              <div className='flex justify-between'>
                <span>{t('dashboard.avgGpu')}</span>
                <b className='tabular-nums'>
                  {fleetData.avg_gpu_usage_percent == null
                    ? '—'
                    : `${fleetData.avg_gpu_usage_percent.toFixed(0)}%`}
                </b>
              </div>
            </div>
          ) : (
            <p className='text-sm text-muted-foreground'>{t('common.empty')}</p>
          )}
        </CardContent>
      </Card>

      <Card data-testid='workbench-top-load'>
        <CardHeader className='pb-2'>
          <CardTitle className='text-sm font-medium'>
            {t('dashboard.workbench.topLoadTitle')}
          </CardTitle>
        </CardHeader>
        <CardContent>
          {fleet.isLoading ? (
            <LoadingSkeleton rows={4} />
          ) : fleet.isError ? (
            <ErrorBanner
              message={fleet.error instanceof Error ? fleet.error.message : undefined}
              onRetry={() => void fleet.refetch()}
            />
          ) : topByUsage.length ? (
            <ul className='space-y-2 text-sm'>
              {topByUsage.map((n) => (
                <li key={n.edge_id} className='flex items-center justify-between gap-2'>
                  <span className='truncate'>{n.edge_id}</span>
                  <span className='tabular-nums'>
                    {n.cpu_usage_percent.toFixed(0)}%
                  </span>
                </li>
              ))}
            </ul>
          ) : (
            <p className='text-sm text-muted-foreground'>{t('common.empty')}</p>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
