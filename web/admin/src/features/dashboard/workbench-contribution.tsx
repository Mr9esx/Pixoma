import { useTranslation } from 'react-i18next'
import { useQuery } from '@tanstack/react-query'
import { listTaskDailyStats } from '@/lib/api/stats'
import { queryKeys } from '@/lib/api/query-keys'
import {
  ContributionGraph,
  ContributionGraphCalendar,
  ContributionGraphLegend,
  ContributionGraphBlock,
  ContributionGraphTotalCount,
  type Activity,
} from '@/components/kibo-ui/contribution-graph'
import { MonitorCard, type MonitorStat } from '@/components/monitor-card'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { dailyToActivity } from './daily-to-activity'
import { pickDays } from './task-stats-parse'
import { formatDate } from './date-range'

function errorMessage(err: unknown): string | undefined {
  return err instanceof Error ? err.message : undefined
}

export function WorkbenchContribution() {
  const { t } = useTranslation()
  const today = new Date()
  const yearAgo = new Date(today)
  yearAgo.setDate(yearAgo.getDate() - 364)
  const from = formatDate(yearAgo)
  const to = formatDate(today)
  const daily = useQuery({
    queryKey: queryKeys.stats.tasksDaily(from, to),
    queryFn: () => listTaskDailyStats({ from, to }),
  })
  const acts = dailyToActivity(pickDays(daily.data))
  const total = acts.reduce((s, a) => s + a.count, 0)
  const stats: MonitorStat[] = [
    { label: t('dashboard.workbench.taskCount'), value: String(total) },
  ]

  return (
    <MonitorCard
      title={t('dashboard.workbench.taskHeatTitle')}
      config={{
        tasks: { label: t('dashboard.workbench.taskHeatTitle'), color: 'var(--primary)' },
      }}
      stats={stats}
      plain
      data-testid='workbench-contribution'
    >
      {daily.isLoading ? (
        <LoadingSkeleton rows={4} />
      ) : daily.isError ? (
        <ErrorBanner message={errorMessage(daily.error)} onRetry={() => void daily.refetch()} />
      ) : acts.length ? (
        <div className='overflow-x-auto'>
          <ContributionGraph
            data={acts as Activity[]}
            labels={{
              months: t('dashboard.workbench.months', { returnObjects: true }) as string[],
              totalCount: t('dashboard.workbench.dailyTotal'),
              legend: {
                less: t('dashboard.workbench.legendLess'),
                more: t('dashboard.workbench.legendMore'),
              },
            }}
            blockSize={14}
            blockMargin={4}
          >
            <ContributionGraphCalendar>
              {(props) => <ContributionGraphBlock {...props} />}
            </ContributionGraphCalendar>
            <ContributionGraphLegend />
            <ContributionGraphTotalCount />
          </ContributionGraph>
        </div>
      ) : (
        <p className='text-sm text-muted-foreground'>{t('common.empty')}</p>
      )}
    </MonitorCard>
  )
}
