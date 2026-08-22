import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import {
  Bar,
  CartesianGrid,
  Cell,
  ComposedChart,
  Line,
  Pie,
  PieChart,
  XAxis,
  YAxis,
} from 'recharts'
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
  listTaskDailyStats,
  listTaskEdgeStats,
  listTaskErrorStats,
} from '@/lib/api/stats'
import {
  pickDays,
  pickEdgeItems,
  pickErrorItems,
  pickSuccessRate,
} from './task-stats-parse'
import type { StatsRange } from './task-range-picker'

function errorMessage(err: unknown): string | undefined {
  return err instanceof Error ? err.message : undefined
}

export function TaskStatsSection({ range }: { range: StatsRange }) {
  const { t } = useTranslation()
  const daily = useQuery({
    queryKey: queryKeys.stats.tasksDaily(range.from, range.to),
    queryFn: () => listTaskDailyStats(range),
  })
  const errors = useQuery({
    queryKey: queryKeys.stats.tasksErrors(range.from, range.to),
    queryFn: () => listTaskErrorStats({ ...range, limit: 5 }),
  })
  const edges = useQuery({
    queryKey: queryKeys.stats.tasksEdges(range.from, range.to),
    queryFn: () => listTaskEdgeStats(range),
  })

  const days = pickDays(daily.data)
  const successRate = pickSuccessRate(daily.data)
  const errorItems = pickErrorItems(errors.data)
  const edgeItems = pickEdgeItems(edges.data)

  const dailyData = days.map((d) => ({
    date: d.date,
    processed: d.processed,
    rate: successRate == null ? null : successRate * 100,
  }))
  const queueExecData = days.map((d) => ({
    date: d.date,
    queue: Math.round((d.avg_queue_ms ?? 0) / 1000),
    exec: Math.round((d.avg_exec_ms ?? 0) / 1000),
  }))
  const statusData = [
    { key: 'succeeded', name: t('dashboard.succeeded'), value: daily.data?.summary.succeeded ?? 0 },
    { key: 'failed', name: t('dashboard.failed'), value: daily.data?.summary.failed ?? 0 },
    { key: 'cancelled', name: t('dashboard.cancelled'), value: daily.data?.summary.cancelled ?? 0 },
  ].filter((s) => s.value > 0)
  const errorData = errorItems.map((e, i) => ({ name: e.error_code, value: e.count, index: i }))

  const dailyConfig: ChartConfig = {
    processed: { label: t('dashboard.processed'), color: 'var(--primary)' },
    rate: {
      label: t('dashboard.statsSuccessRate'),
      color: 'color-mix(in oklch, var(--primary) 55%, var(--background))',
    },
  }
  const queueConfig: ChartConfig = {
    queue: { label: t('dashboard.queueWait'), color: 'color-mix(in oklch, var(--primary) 55%, var(--background))' },
    exec: { label: t('dashboard.execTime'), color: 'var(--primary)' },
  }
  const donutColors = {
    succeeded: 'var(--primary)',
    failed: 'var(--destructive)',
    cancelled: 'var(--muted-foreground)',
  }
  const errorColors = [
    'var(--primary)',
    'var(--destructive)',
    'color-mix(in oklch, var(--primary) 60%, var(--background))',
    'color-mix(in oklch, var(--destructive) 60%, var(--background))',
    'var(--muted-foreground)',
  ]
  const maxEdgeCount = Math.max(1, ...edgeItems.map((e) => e.count))

  return (
    <div className='space-y-4'>
      <div className='grid gap-4 lg:grid-cols-2'>
        <Card data-testid='task-daily-chart-card'>
          <CardHeader className='pb-2'>
            <CardTitle className='text-base font-semibold'>
              {t('dashboard.taskDailyTitle')}
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
            ) : (
              <ChartContainer config={dailyConfig} className='h-[200px] w-full min-w-0 sm:h-[240px]'>
                <ComposedChart data={dailyData} margin={{ top: 8, right: 4, bottom: 0, left: 4 }}>
                  <CartesianGrid vertical={false} />
                  <XAxis dataKey='date' tickLine={false} axisLine={false} tickMargin={4} />
                  <YAxis yAxisId='left' tickLine={false} axisLine={false} allowDecimals={false} width={36} />
                  <YAxis yAxisId='right' orientation='right' domain={[0, 100]} tickLine={false} axisLine={false} width={36} unit='%' />
                  <ChartTooltip content={<ChartTooltipContent />} />
                  <Bar yAxisId='left' dataKey='processed' fill='var(--color-processed)' radius={4} />
                  <Line yAxisId='right' type='monotone' dataKey='rate' stroke='var(--color-rate)' strokeWidth={2} dot={false} />
                </ComposedChart>
              </ChartContainer>
            )}
          </CardContent>
        </Card>

        <Card data-testid='task-queue-exec-card'>
          <CardHeader className='pb-2'>
            <CardTitle className='text-base font-semibold'>
              {t('dashboard.queueExecTitle')}
            </CardTitle>
            <CardDescription>{t('dashboard.queueExecHint')}</CardDescription>
          </CardHeader>
          <CardContent>
            {daily.isLoading ? (
              <LoadingSkeleton rows={4} />
            ) : daily.isError ? (
              <ErrorBanner message={errorMessage(daily.error)} onRetry={() => void daily.refetch()} />
            ) : (
              <ChartContainer config={queueConfig} className='h-[200px] w-full min-w-0 sm:h-[240px]'>
                <ComposedChart data={queueExecData} margin={{ top: 8, right: 4, bottom: 0, left: 4 }}>
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

      <div className='grid gap-4 sm:grid-cols-3'>
        <Card data-testid='task-status-donut-card'>
          <CardHeader className='pb-2'>
            <CardTitle className='text-sm font-medium'>
              {t('dashboard.statusDonutTitle')}
            </CardTitle>
          </CardHeader>
          <CardContent>
            {daily.isLoading ? (
              <LoadingSkeleton rows={3} />
            ) : daily.isError ? (
              <ErrorBanner message={errorMessage(daily.error)} onRetry={() => void daily.refetch()} />
            ) : (
              <ChartContainer
                config={{
                  succeeded: { label: t('dashboard.succeeded'), color: donutColors.succeeded },
                  failed: { label: t('dashboard.failed'), color: donutColors.failed },
                  cancelled: { label: t('dashboard.cancelled'), color: donutColors.cancelled },
                }}
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

        <Card data-testid='task-error-donut-card'>
          <CardHeader className='pb-2'>
            <CardTitle className='text-sm font-medium'>
              {t('dashboard.errorDonutTitle')}
            </CardTitle>
          </CardHeader>
          <CardContent>
            {errors.isLoading ? (
              <LoadingSkeleton rows={3} />
            ) : errors.isError ? (
              <ErrorBanner message={errorMessage(errors.error)} onRetry={() => void errors.refetch()} />
            ) : errorData.length ? (
              <ChartContainer
                config={Object.fromEntries(
                  errorItems.map((e, i) => [e.error_code, { label: e.error_code, color: errorColors[i % errorColors.length] }])
                )}
                className='mx-auto h-[180px] w-full'
              >
                <PieChart>
                  <ChartTooltip content={<ChartTooltipContent />} />
                  <Pie data={errorData} dataKey='value' nameKey='name' innerRadius={52} outerRadius={72} paddingAngle={2}>
                    {errorData.map((e) => (
                      <Cell key={e.name} fill={errorColors[e.index % errorColors.length]} />
                    ))}
                  </Pie>
                </PieChart>
              </ChartContainer>
            ) : (
              <p className='text-sm text-muted-foreground'>{t('common.empty')}</p>
            )}
          </CardContent>
        </Card>

        <Card data-testid='task-node-load-card'>
          <CardHeader className='pb-2'>
            <CardTitle className='text-sm font-medium'>
              {t('dashboard.perNodeLoadTitle')}
            </CardTitle>
          </CardHeader>
          <CardContent>
            {edges.isLoading ? (
              <LoadingSkeleton rows={4} />
            ) : edges.isError ? (
              <ErrorBanner message={errorMessage(edges.error)} onRetry={() => void edges.refetch()} />
            ) : edgeItems.length ? (
              <ul className='space-y-2 text-sm'>
                {edgeItems.map((e) => (
                  <li key={e.edge_id}>
                    <div className='flex justify-between gap-2'>
                      <span className='truncate'>{e.edge_id}</span>
                      <span className='tabular-nums'>
                        {e.count} ·{' '}
                        {e.success_rate == null ? '—' : `${(e.success_rate * 100).toFixed(0)}%`}
                      </span>
                    </div>
                    <div className='mt-1 h-2 w-full overflow-hidden rounded-full bg-muted'>
                      <div
                        className='h-full rounded-full bg-foreground/80'
                        style={{ width: `${(e.count / maxEdgeCount) * 100}%` }}
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
      </div>
    </div>
  )
}
