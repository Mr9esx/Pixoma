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
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { dailyToActivity } from './daily-to-activity'
import { pickDays } from './task-stats-parse'
import type { StatsRange } from './task-range-picker'

function errorMessage(err: unknown): string | undefined {
  return err instanceof Error ? err.message : undefined
}

export function WorkbenchContribution({ range }: { range: StatsRange }) {
  const { t } = useTranslation()
  const daily = useQuery({
    queryKey: queryKeys.stats.tasksDaily(range.from, range.to),
    queryFn: () => listTaskDailyStats({ from: range.from, to: range.to }),
  })
  const acts = dailyToActivity(pickDays(daily.data))

  return (
    <Card data-testid='workbench-contribution'>
      <CardHeader className='pb-2'>
        <CardTitle className='text-base font-semibold'>
          {t('dashboard.workbench.taskHeatTitle')}
        </CardTitle>
        <CardDescription>
          {t('dashboard.taskDailyDescription', { from: range.from, to: range.to })}
        </CardDescription>
      </CardHeader>
      <CardContent>
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
      </CardContent>
    </Card>
  )
}
