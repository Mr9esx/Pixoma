import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import {
  Area,
  AreaChart,
  Bar,
  BarChart,
  CartesianGrid,
  Cell,
  Line,
  XAxis,
  YAxis,
} from 'recharts'
import { queryKeys } from '@/lib/api/query-keys'
import {
  listTaskCaseTopStats,
  listTaskDailyStats,
  listTaskErrorStats,
} from '@/lib/api/stats'
import {
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from '@/components/ui/chart'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { MonitorCard, type MonitorStat } from '@/components/monitor-card'
import type { StatsRange } from './date-range'
import { pickCaseItems, pickDays, pickErrorItems } from './task-stats-parse'

function errorMessage(err: unknown): string | undefined {
  return err instanceof Error ? err.message : undefined
}

const TICK = { tickLine: false, axisLine: false, tickMargin: 4 } as const
const MARGIN = { top: 10, right: 4, bottom: 0, left: 4 }

export function WorkbenchChartPairs({ range }: { range: StatsRange }) {
  const { t } = useTranslation()
  const cases = useQuery({
    queryKey: queryKeys.stats.casesTop(range.from, range.to),
    queryFn: () =>
      listTaskCaseTopStats({ from: range.from, to: range.to, limit: 10 }),
  })
  const daily = useQuery({
    queryKey: queryKeys.stats.tasksDaily(range.from, range.to),
    queryFn: () => listTaskDailyStats({ from: range.from, to: range.to }),
  })
  const errors = useQuery({
    queryKey: queryKeys.stats.tasksErrors(range.from, range.to),
    queryFn: () =>
      listTaskErrorStats({ from: range.from, to: range.to, limit: 5 }),
  })

  const items = pickCaseItems(cases.data)
  const days = pickDays(daily.data)
  const durationData = days.map((d) => ({
    date: d.date,
    queue: Math.round((d.avg_queue_ms ?? 0) / 1000),
    exec: Math.round((d.avg_exec_ms ?? 0) / 1000),
  }))
  const statusData = days.reduce(
    (acc, d) => {
      acc.succeeded += d.succeeded
      acc.failed += d.failed
      acc.cancelled += d.cancelled
      return acc
    },
    { succeeded: 0, failed: 0, cancelled: 0 }
  )
  const errorItems = pickErrorItems(errors.data)

  const durationConfig: ChartConfig = {
    queue: {
      label: t('dashboard.queueWait'),
      color: 'color-mix(in oklch, var(--primary) 55%, var(--background))',
    },
    exec: { label: t('dashboard.execTime'), color: 'var(--primary)' },
  }
  const statusConfig: ChartConfig = {
    succeeded: { label: t('dashboard.succeeded'), color: 'var(--primary)' },
    failed: { label: t('dashboard.failed'), color: 'var(--destructive)' },
    cancelled: {
      label: t('dashboard.cancelled'),
      color: 'var(--muted-foreground)',
    },
  }
  const errorConfig: ChartConfig = {
    errors: {
      label: t('dashboard.workbench.errorTopTitle'),
      color: 'var(--destructive)',
    },
  }
  const workflowConfig: ChartConfig = {
    cases: {
      label: t('dashboard.workbench.workflowTopTitle'),
      color: 'var(--primary)',
    },
  }

  const durationStats: MonitorStat[] = [
    {
      label: t('dashboard.queueWait'),
      value: `${Math.round(avgOf(durationData, 'queue'))}s`,
    },
    {
      label: t('dashboard.execTime'),
      value: `${Math.round(avgOf(durationData, 'exec'))}s`,
    },
    {
      label: t('dashboard.workbench.taskDurationTitle'),
      value: `${Math.round(avgOf(durationData, 'queue') + avgOf(durationData, 'exec'))}s`,
    },
  ]
  const workflowStats: MonitorStat[] = [
    {
      label: t('dashboard.workbench.taskCount'),
      value: `${items.reduce((s, c) => s + c.count, 0)}`,
    },
    {
      label: t('dashboard.workbench.workflowTopTitle'),
      value: `${items.length}`,
    },
  ]
  const statusStats: MonitorStat[] = [
    { label: t('dashboard.succeeded'), value: `${statusData.succeeded}` },
    { label: t('dashboard.failed'), value: `${statusData.failed}` },
    { label: t('dashboard.cancelled'), value: `${statusData.cancelled}` },
  ]
  const errorStats: MonitorStat[] = [
    {
      label: t('dashboard.workbench.errorTopTitle'),
      value: `${errorItems.reduce((s, e) => s + e.count, 0)}`,
    },
  ]

  const anyLoading = cases.isLoading || daily.isLoading || errors.isLoading
  const anyError = cases.isError || daily.isError || errors.isError

  if (anyError) {
    return (
      <ErrorBanner
        message={errorMessage(cases.error ?? daily.error ?? errors.error)}
        onRetry={() => {
          void cases.refetch()
          void daily.refetch()
          void errors.refetch()
        }}
      />
    )
  }

  if (anyLoading) {
    return (
      <div className='flex flex-col gap-4'>
        <LoadingSkeleton rows={4} />
        <LoadingSkeleton rows={4} />
      </div>
    )
  }

  return (
    <div className='flex flex-col gap-4'>
      <div className='flex flex-col gap-4 xl:flex-row'>
        <MonitorCard
          title={t('dashboard.workbench.taskDurationTitle')}
          config={durationConfig}
          stats={durationStats}
        >
          <AreaChart data={durationData} margin={MARGIN}>
            <defs>
              <linearGradient id='fillExec' x1='0' y1='0' x2='0' y2='1'>
                <stop
                  offset='0%'
                  stopColor='var(--color-exec)'
                  stopOpacity={0.3}
                />
                <stop
                  offset='100%'
                  stopColor='var(--color-exec)'
                  stopOpacity={0.02}
                />
              </linearGradient>
            </defs>
            <CartesianGrid vertical={false} />
            <XAxis dataKey='date' {...TICK} height={20} />
            <YAxis width={40} unit='s' {...TICK} />
            <ChartTooltip content={<ChartTooltipContent />} />
            <Area
              type='monotone'
              dataKey='exec'
              stroke='var(--color-exec)'
              fill='url(#fillExec)'
              strokeWidth={2}
            />
            <Line
              type='monotone'
              dataKey='queue'
              stroke='var(--color-queue)'
              strokeWidth={2}
              dot={false}
            />
          </AreaChart>
        </MonitorCard>

        <MonitorCard
          title={t('dashboard.workbench.workflowTopTitle')}
          config={workflowConfig}
          stats={workflowStats}
        >
          <BarChart
            data={items.map((c) => ({
              name: c.case_name?.trim() || `Case #${c.case_id}`,
              count: c.count,
            }))}
            margin={MARGIN}
          >
            <CartesianGrid vertical={false} />
            <XAxis dataKey='name' {...TICK} height={20} />
            <YAxis width={40} allowDecimals={false} {...TICK} />
            <ChartTooltip content={<ChartTooltipContent />} />
            <Bar
              dataKey='count'
              fill='var(--color-cases)'
              radius={[2, 2, 0, 0]}
            />
          </BarChart>
        </MonitorCard>
      </div>

      <div className='flex flex-col gap-4 xl:flex-row'>
        <MonitorCard
          title={t('dashboard.workbench.statusDistributionTitle')}
          config={statusConfig}
          stats={statusStats}
        >
          <BarChart
            data={[
              {
                name: t('dashboard.succeeded'),
                value: statusData.succeeded,
                fill: 'var(--color-succeeded)',
              },
              {
                name: t('dashboard.failed'),
                value: statusData.failed,
                fill: 'var(--color-failed)',
              },
              {
                name: t('dashboard.cancelled'),
                value: statusData.cancelled,
                fill: 'var(--color-cancelled)',
              },
            ]}
            margin={MARGIN}
          >
            <CartesianGrid vertical={false} />
            <XAxis dataKey='name' {...TICK} height={20} />
            <YAxis width={40} allowDecimals={false} {...TICK} />
            <ChartTooltip content={<ChartTooltipContent />} />
            <Bar dataKey='value' radius={[2, 2, 0, 0]}>
              {[
                {
                  name: t('dashboard.succeeded'),
                  fill: 'var(--color-succeeded)',
                },
                { name: t('dashboard.failed'), fill: 'var(--color-failed)' },
                {
                  name: t('dashboard.cancelled'),
                  fill: 'var(--color-cancelled)',
                },
              ].map((entry) => (
                <Cell key={entry.name} fill={entry.fill} />
              ))}
            </Bar>
          </BarChart>
        </MonitorCard>

        <MonitorCard
          title={t('dashboard.workbench.errorTopTitle')}
          config={errorConfig}
          stats={errorStats}
        >
          <BarChart
            data={errorItems.map((e) => ({
              name: e.error_code,
              count: e.count,
            }))}
            margin={MARGIN}
          >
            <CartesianGrid vertical={false} />
            <XAxis dataKey='name' {...TICK} height={20} />
            <YAxis width={40} allowDecimals={false} {...TICK} />
            <ChartTooltip content={<ChartTooltipContent />} />
            <Bar
              dataKey='count'
              fill='var(--color-errors)'
              radius={[2, 2, 0, 0]}
            />
          </BarChart>
        </MonitorCard>
      </div>
    </div>
  )
}

function avgOf(
  data: { queue: number; exec: number }[],
  key: 'queue' | 'exec'
): number {
  if (data.length === 0) return 0
  return data.reduce((s, d) => s + d[key], 0) / data.length
}
