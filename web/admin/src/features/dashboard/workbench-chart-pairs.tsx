import { useTranslation } from 'react-i18next'
import {
  Bar,
  CartesianGrid,
  Cell,
  ComposedChart,
  Pie,
  PieChart,
  XAxis,
  YAxis,
} from 'recharts'
import { useQuery } from '@tanstack/react-query'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from '@/components/ui/chart'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { queryKeys } from '@/lib/api/query-keys'
import {
  listTaskCaseTopStats,
  listTaskDailyStats,
  listTaskErrorStats,
} from '@/lib/api/stats'
import {
  pickCaseItems,
  pickDays,
  pickErrorItems,
} from './task-stats-parse'
import type { StatsRange } from './task-range-picker'

function errorMessage(err: unknown): string | undefined {
  return err instanceof Error ? err.message : undefined
}

export function WorkbenchChartPairs({ range }: { range: StatsRange }) {
  const { t } = useTranslation()
  const cases = useQuery({
    queryKey: queryKeys.stats.casesTop(range.from, range.to),
    queryFn: () => listTaskCaseTopStats({ ...range, limit: 10 }),
  })
  const daily = useQuery({
    queryKey: queryKeys.stats.tasksDaily(range.from, range.to),
    queryFn: () => listTaskDailyStats(range),
  })
  const errors = useQuery({
    queryKey: queryKeys.stats.tasksErrors(range.from, range.to),
    queryFn: () => listTaskErrorStats({ ...range, limit: 5 }),
  })

  const items = pickCaseItems(cases.data)
  const maxCount = Math.max(1, ...items.map((c) => c.count))
  const days = pickDays(daily.data)
  const durationData = days.map((d) => ({
    date: d.date,
    queue: Math.round((d.avg_queue_ms ?? 0) / 1000),
    exec: Math.round((d.avg_exec_ms ?? 0) / 1000),
  }))
  const statusData = [
    { key: 'succeeded', name: t('dashboard.succeeded'), value: daily.data?.summary.succeeded ?? 0 },
    { key: 'failed', name: t('dashboard.failed'), value: daily.data?.summary.failed ?? 0 },
    { key: 'cancelled', name: t('dashboard.cancelled'), value: daily.data?.summary.cancelled ?? 0 },
  ].filter((s) => s.value > 0)
  const errorItems = pickErrorItems(errors.data)
  const errorData = errorItems.map((e, i) => ({ name: e.error_code, value: e.count, index: i }))

  const durationConfig: ChartConfig = {
    queue: { label: t('dashboard.queueWait'), color: 'var(--chart-4)' },
    exec: { label: t('dashboard.execTime'), color: 'var(--chart-1)' },
  }
  const statusConfig: ChartConfig = {
    succeeded: { label: t('dashboard.succeeded'), color: 'var(--chart-1)' },
    failed: { label: t('dashboard.failed'), color: 'var(--destructive)' },
    cancelled: { label: t('dashboard.cancelled'), color: 'var(--muted-foreground)' },
  }
  return (
    <div className='space-y-4'>
      <div className='grid gap-4 lg:grid-cols-2'>
        <Card data-testid='workbench-workflow-top'>
          <CardHeader className='pb-2'>
            <CardTitle className='text-base font-semibold'>
              {t('dashboard.workbench.workflowTopTitle')}
            </CardTitle>
          </CardHeader>
          <CardContent>
            {cases.isLoading ? (
              <LoadingSkeleton rows={5} />
            ) : cases.isError ? (
              <ErrorBanner message={errorMessage(cases.error)} onRetry={() => void cases.refetch()} />
            ) : items.length ? (
              <ul className='space-y-2 text-sm'>
                {items.map((c) => (
                  <li key={c.case_id}>
                    <div className='flex justify-between gap-2'>
                      <span className='truncate'>Case #{c.case_id}</span>
                      <span className='tabular-nums'>
                        {c.count} ·{' '}
                        {c.avg_duration_ms == null
                          ? '—'
                          : `${(c.avg_duration_ms / 1000).toFixed(0)}s`}
                      </span>
                    </div>
                    <div className='mt-1 h-2 w-full overflow-hidden rounded-full bg-muted'>
                      <div
                        className='h-full rounded-full bg-foreground/80'
                        style={{ width: `${(c.count / maxCount) * 100}%` }}
                      />
                    </div>
                  </li>
                ))}
              </ul>
            ) : (
              <p className='text-sm text-muted-foreground'>{t('common.empty')}</p>
            )}
          </CardContent>
        </Card>

        <Card data-testid='workbench-task-duration'>
          <CardHeader className='pb-2'>
            <CardTitle className='text-base font-semibold'>
              {t('dashboard.workbench.taskDurationTitle')}
            </CardTitle>
            <CardDescription>{t('dashboard.queueExecHint')}</CardDescription>
          </CardHeader>
          <CardContent>
            {daily.isLoading ? (
              <LoadingSkeleton rows={4} />
            ) : daily.isError ? (
              <ErrorBanner message={errorMessage(daily.error)} onRetry={() => void daily.refetch()} />
            ) : (
              <ChartContainer config={durationConfig} className='h-[200px] w-full min-w-0 sm:h-[240px]'>
                <ComposedChart data={durationData} margin={{ top: 8, right: 4, bottom: 0, left: 4 }}>
                  <CartesianGrid vertical={false} />
                  <XAxis dataKey='date' tickLine={false} axisLine={false} tickMargin={4} />
                  <YAxis tickLine={false} axisLine={false} width={36} unit='s' />
                  <ChartTooltip content={<ChartTooltipContent />} />
                  <Bar dataKey='queue' stackId='a' fill='var(--color-queue)' radius={[0, 0, 2, 2]} />
                  <Bar dataKey='exec' stackId='a' fill='var(--color-exec)' radius={[2, 2, 0, 0]} />
                </ComposedChart>
              </ChartContainer>
            )}
          </CardContent>
        </Card>
      </div>

      <div className='grid gap-4 lg:grid-cols-2'>
        <Card data-testid='workbench-status-distribution'>
          <CardHeader className='pb-2'>
            <CardTitle className='text-base font-semibold'>
              {t('dashboard.workbench.statusDistributionTitle')}
            </CardTitle>
          </CardHeader>
          <CardContent>
            {daily.isLoading ? (
              <LoadingSkeleton rows={3} />
            ) : daily.isError ? (
              <ErrorBanner message={errorMessage(daily.error)} onRetry={() => void daily.refetch()} />
            ) : (
              <ChartContainer
                config={statusConfig}
                className='mx-auto h-[180px] w-full'
              >
                <PieChart>
                  <ChartTooltip content={<ChartTooltipContent />} />
                  <Pie data={statusData} dataKey='value' nameKey='name' innerRadius={52} outerRadius={72} paddingAngle={2}>
                    {statusData.map((s) => (
                      <Cell key={s.key} fill={`var(--color-${s.key})`} />
                    ))}
                  </Pie>
                </PieChart>
              </ChartContainer>
            )}
          </CardContent>
        </Card>

        <Card data-testid='workbench-error-top'>
          <CardHeader className='pb-2'>
            <CardTitle className='text-base font-semibold'>
              {t('dashboard.workbench.errorTopTitle')}
            </CardTitle>
          </CardHeader>
          <CardContent>
            {errors.isLoading ? (
              <LoadingSkeleton rows={3} />
            ) : errors.isError ? (
              <ErrorBanner message={errorMessage(errors.error)} onRetry={() => void errors.refetch()} />
            ) : errorData.length ? (
              <ul className='space-y-2 text-sm'>
                {errorData.map((e) => (
                  <li key={e.name} className='flex items-center justify-between gap-2'>
                    <span className='truncate'>{e.name}</span>
                    <span className='tabular-nums'>{e.value}</span>
                  </li>
                ))}
              </ul>
            ) : (
              <p className='text-sm text-muted-foreground'>{t('common.empty')}</p>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
