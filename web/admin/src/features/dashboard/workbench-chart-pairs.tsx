import { useTranslation } from 'react-i18next'
import {
  Area,
  CartesianGrid,
  Cell,
  ComposedChart,
  Line,
  Pie,
  PieChart,
  XAxis,
  YAxis,
} from 'recharts'
import { useQuery } from '@tanstack/react-query'
import { MoreHorizontal } from 'lucide-react'
import {
  Card,
  CardAction,
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
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { queryKeys } from '@/lib/api/query-keys'
import {
  listTaskCaseTopStats,
  listTaskDailyStats,
  listTaskErrorStats,
} from '@/lib/api/stats'
import { pickCaseItems, pickDays, pickErrorItems } from './task-stats-parse'
import type { StatsRange } from './task-range-picker'

function errorMessage(err: unknown): string | undefined {
  return err instanceof Error ? err.message : undefined
}

function LegendDot({ color, label }: { color: string; label: string }) {
  return (
    <div className='flex items-center gap-1.5'>
      <div
        className='size-3 rounded-full border-2 bg-background'
        style={{ borderColor: color }}
      />
      <span className='text-xs text-muted-foreground'>{label}</span>
    </div>
  )
}

function CardMenu() {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant='ghost' size='icon' className='size-7'>
          <MoreHorizontal className='size-4' />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align='end'>
        <DropdownMenuItem>导出</DropdownMenuItem>
        <DropdownMenuItem>筛选</DropdownMenuItem>
        <DropdownMenuItem>分享</DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  )
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

  const durationConfig: ChartConfig = {
    queue: { label: t('dashboard.queueWait'), color: 'var(--chart-4)' },
    exec: { label: t('dashboard.execTime'), color: 'var(--chart-1)' },
  }
  const statusConfig: ChartConfig = {
    succeeded: { label: t('dashboard.succeeded'), color: 'var(--chart-1)' },
    failed: { label: t('dashboard.failed'), color: 'var(--chart-5)' },
    cancelled: { label: t('dashboard.cancelled'), color: 'var(--muted-foreground)' },
  }
  const statusColor = {
    succeeded: 'var(--chart-1)',
    failed: 'var(--chart-5)',
    cancelled: 'var(--muted-foreground)',
  }

  return (
    <div className='space-y-4'>
      <div className='grid gap-4 lg:grid-cols-2'>
        <Card data-testid='workbench-task-duration'>
          <CardHeader className='pb-3'>
            <CardTitle className='text-sm font-medium'>
              {t('dashboard.workbench.taskDurationTitle')}
            </CardTitle>
            <CardAction className='flex items-center gap-2'>
              <div className='flex items-center gap-4'>
                <LegendDot color='var(--chart-4)' label={t('dashboard.queueWait')} />
                <LegendDot color='var(--chart-1)' label={t('dashboard.execTime')} />
              </div>
              <CardMenu />
            </CardAction>
          </CardHeader>
          <CardContent className='px-2.5'>
            {daily.isLoading ? (
              <LoadingSkeleton rows={4} />
            ) : daily.isError ? (
              <ErrorBanner message={errorMessage(daily.error)} onRetry={() => void daily.refetch()} />
            ) : (
              <ChartContainer config={durationConfig} className='h-[260px] w-full min-w-0'>
                <ComposedChart data={durationData} margin={{ top: 8, right: 4, bottom: 0, left: 4 }}>
                  <defs>
                    <linearGradient id='fillExec' x1='0' y1='0' x2='0' y2='1'>
                      <stop offset='5%' stopColor='var(--color-exec)' stopOpacity={0.35} />
                      <stop offset='95%' stopColor='var(--color-exec)' stopOpacity={0.02} />
                    </linearGradient>
                    <linearGradient id='fillQueue' x1='0' y1='0' x2='0' y2='1'>
                      <stop offset='5%' stopColor='var(--color-queue)' stopOpacity={0.3} />
                      <stop offset='95%' stopColor='var(--color-queue)' stopOpacity={0.02} />
                    </linearGradient>
                  </defs>
                  <CartesianGrid vertical={false} />
                  <XAxis dataKey='date' tickLine={false} axisLine={false} tickMargin={4} />
                  <YAxis tickLine={false} axisLine={false} width={36} unit='s' />
                  <ChartTooltip content={<ChartTooltipContent />} />
                  <Area type='monotone' dataKey='exec' stroke='var(--color-exec)' fill='url(#fillExec)' strokeWidth={2} />
                  <Area type='monotone' dataKey='queue' stroke='var(--color-queue)' fill='url(#fillQueue)' strokeWidth={2} />
                  <Line type='monotone' dataKey='exec' stroke='var(--color-exec)' strokeWidth={2} dot={false} />
                </ComposedChart>
              </ChartContainer>
            )}
          </CardContent>
        </Card>

        <Card data-testid='workbench-workflow-top'>
          <CardHeader className='pb-3'>
            <CardTitle className='text-sm font-medium'>
              {t('dashboard.workbench.workflowTopTitle')}
            </CardTitle>
            <CardAction className='flex items-center gap-2'>
              <CardMenu />
            </CardAction>
          </CardHeader>
          <CardContent className='px-2.5'>
            {cases.isLoading ? (
              <LoadingSkeleton rows={5} />
            ) : cases.isError ? (
              <ErrorBanner message={errorMessage(cases.error)} onRetry={() => void cases.refetch()} />
            ) : items.length ? (
              <ul className='space-y-3 text-sm'>
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
      </div>

      <div className='grid gap-4 lg:grid-cols-2'>
        <Card data-testid='workbench-status-distribution'>
          <CardHeader className='pb-3'>
            <CardTitle className='text-sm font-medium'>
              {t('dashboard.workbench.statusDistributionTitle')}
            </CardTitle>
            <CardAction className='flex items-center gap-2'>
              <div className='flex items-center gap-4'>
                <LegendDot color={statusColor.succeeded} label={t('dashboard.succeeded')} />
                <LegendDot color={statusColor.failed} label={t('dashboard.failed')} />
                <LegendDot color={statusColor.cancelled} label={t('dashboard.cancelled')} />
              </div>
              <CardMenu />
            </CardAction>
          </CardHeader>
          <CardContent className='px-2.5'>
            {daily.isLoading ? (
              <LoadingSkeleton rows={4} />
            ) : daily.isError ? (
              <ErrorBanner message={errorMessage(daily.error)} onRetry={() => void daily.refetch()} />
            ) : (
              <ChartContainer config={statusConfig} className='h-[220px] w-full'>
                <PieChart>
                  <ChartTooltip content={<ChartTooltipContent />} />
                  <Pie data={statusData} dataKey='value' nameKey='name' innerRadius={48} outerRadius={72} paddingAngle={2}>
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
          <CardHeader className='pb-3'>
            <CardTitle className='text-sm font-medium'>
              {t('dashboard.workbench.errorTopTitle')}
            </CardTitle>
            <CardAction className='flex items-center gap-2'>
              <CardMenu />
            </CardAction>
          </CardHeader>
          <CardContent className='px-2.5'>
            {errors.isLoading ? (
              <LoadingSkeleton rows={4} />
            ) : errors.isError ? (
              <ErrorBanner message={errorMessage(errors.error)} onRetry={() => void errors.refetch()} />
            ) : errorData.length ? (
              <ul className='space-y-3 text-sm'>
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
