import { useTranslation } from 'react-i18next'
import {
  Area,
  AreaChart,
  CartesianGrid,
  Cell,
  Pie,
  PieChart,
  XAxis,
  YAxis,
} from 'recharts'
import { useQuery } from '@tanstack/react-query'
import {
  Card,
  CardContent,
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
    queryFn: () => listTaskCaseTopStats({ from: range.from, to: range.to, limit: 10 }),
  })
  const daily = useQuery({
    queryKey: queryKeys.stats.tasksDaily(range.from, range.to),
    queryFn: () => listTaskDailyStats({ from: range.from, to: range.to }),
  })
  const errors = useQuery({
    queryKey: queryKeys.stats.tasksErrors(range.from, range.to),
    queryFn: () => listTaskErrorStats({ from: range.from, to: range.to, limit: 5 }),
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
  const totalCases = items.reduce((s, c) => s + c.count, 0)
  const totalErrors = errorItems.reduce((s, e) => s + e.count, 0)
  const totalProcessed = daily.data?.summary.processed ?? 0
  const successRate = daily.data?.summary.success_rate

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
            <CardTitle className='text-sm font-medium'>
              {t('dashboard.workbench.workflowTopTitle')}
            </CardTitle>
            <div className='flex items-baseline gap-2'>
              <span className='text-2xl font-bold tabular-nums'>{totalCases}</span>
              <span className='text-xs text-muted-foreground'>
                {t('dashboard.workbench.taskCount')}
              </span>
            </div>
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
            <CardTitle className='text-sm font-medium'>
              {t('dashboard.workbench.taskDurationTitle')}
            </CardTitle>
            <div className='flex items-baseline gap-2'>
              <span className='text-2xl font-bold tabular-nums'>
                {averageDuration(durationData) == null ? '—' : `${Math.round(averageDuration(durationData)!)}s`}
              </span>
              <span className='text-xs text-muted-foreground'>
                {t('dashboard.queueExecHint')}
              </span>
            </div>
          </CardHeader>
          <CardContent>
            {daily.isLoading ? (
              <LoadingSkeleton rows={4} />
            ) : daily.isError ? (
              <ErrorBanner message={errorMessage(daily.error)} onRetry={() => void daily.refetch()} />
            ) : (
              <ChartContainer config={durationConfig} className='h-[200px] w-full min-w-0 sm:h-[240px]'>
                <AreaChart data={durationData} margin={{ top: 8, right: 4, bottom: 0, left: 4 }}>
                  <defs>
                    <linearGradient id='fillQueue' x1='0' y1='0' x2='0' y2='1'>
                      <stop offset='5%' stopColor='var(--color-queue)' stopOpacity={0.8} />
                      <stop offset='95%' stopColor='var(--color-queue)' stopOpacity={0.1} />
                    </linearGradient>
                    <linearGradient id='fillExec' x1='0' y1='0' x2='0' y2='1'>
                      <stop offset='5%' stopColor='var(--color-exec)' stopOpacity={0.7} />
                      <stop offset='95%' stopColor='var(--color-exec)' stopOpacity={0.05} />
                    </linearGradient>
                  </defs>
                  <CartesianGrid vertical={false} />
                  <XAxis dataKey='date' tickLine={false} axisLine={false} tickMargin={4} />
                  <YAxis tickLine={false} axisLine={false} width={36} unit='s' />
                  <ChartTooltip content={<ChartTooltipContent />} />
                  <Area type='monotone' dataKey='exec' stroke='var(--color-exec)' fill='url(#fillExec)' strokeWidth={2} />
                  <Area type='monotone' dataKey='queue' stroke='var(--color-queue)' fill='url(#fillQueue)' strokeWidth={2} />
                </AreaChart>
              </ChartContainer>
            )}
          </CardContent>
        </Card>
      </div>

      <div className='grid gap-4 lg:grid-cols-2'>
        <Card data-testid='workbench-status-distribution'>
          <CardHeader className='pb-2'>
            <CardTitle className='text-sm font-medium'>
              {t('dashboard.workbench.statusDistributionTitle')}
            </CardTitle>
            <div className='flex items-baseline gap-2'>
              <span className='text-2xl font-bold tabular-nums'>{totalProcessed}</span>
              <span className='text-xs text-muted-foreground'>
                {successRate == null ? t('dashboard.workbench.taskCount') : `${(successRate * 100).toFixed(0)}% ${t('dashboard.statsSuccessRate')}`}
              </span>
            </div>
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
            <CardTitle className='text-sm font-medium'>
              {t('dashboard.workbench.errorTopTitle')}
            </CardTitle>
            <div className='flex items-baseline gap-2'>
              <span className='text-2xl font-bold tabular-nums'>{totalErrors}</span>
              <span className='text-xs text-muted-foreground'>
                {t('dashboard.workbench.taskCount')}
              </span>
            </div>
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

function averageDuration(data: { queue: number; exec: number }[]): number | null {
  if (data.length === 0) return null
  const total = data.reduce((s, d) => s + d.exec + d.queue, 0)
  return total / data.length
}
