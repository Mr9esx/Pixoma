import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { queryKeys } from '@/lib/api/query-keys'
import { listTaskDailyStats } from '@/lib/api/stats'
import { ChartSkeleton } from '@/components/feedback/chart-skeleton'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { MonitorCard } from '@/components/monitor-card'
import { dailyToActivity } from './daily-to-activity'
import { pickDays } from './task-stats-parse'
import { WorkbenchHeatmap } from './workbench-heatmap'

function errorMessage(err: unknown): string | undefined {
  return err instanceof Error ? err.message : undefined
}

export function WorkbenchContribution() {
  const { t } = useTranslation()
  const today = new Date()
  const from = `${today.getFullYear()}-01-01`
  const to = `${today.getFullYear()}-12-31`
  const daily = useQuery({
    queryKey: queryKeys.stats.tasksDaily(from, to),
    queryFn: () => listTaskDailyStats({ from, to }),
  })
  const acts = dailyToActivity(pickDays(daily.data))

  return (
    <MonitorCard
      title={t('dashboard.workbench.taskHeatTitle')}
      config={{}}
      stats={[]}
      plain
      data-testid='workbench-contribution'
    >
      {daily.isLoading ? (
        <ChartSkeleton />
      ) : daily.isError ? (
        <ErrorBanner
          message={errorMessage(daily.error)}
          onRetry={() => void daily.refetch()}
        />
      ) : acts.length ? (
        <div className='w-full'>
          <WorkbenchHeatmap data={acts} />
        </div>
      ) : (
        <p className='text-sm text-muted-foreground'>{t('common.empty')}</p>
      )}
    </MonitorCard>
  )
}
