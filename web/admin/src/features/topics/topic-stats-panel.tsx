import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { BadgeCheck, ListTodo, Timer } from 'lucide-react'
import {
  Area,
  AreaChart,
  CartesianGrid,
  XAxis,
  YAxis,
} from 'recharts'
import { getTopicStats } from '@/lib/api/topics'
import { queryKeys } from '@/lib/api/query-keys'
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from '@/components/ui/chart'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { kit } from '@/features/edges/kit-classes'
import { cn } from '@/lib/utils'

const STATUS_ORDER = ['pending', 'queued', 'running', 'succeeded', 'failed', 'cancelled'] as const

const STATUS_COLOR: Record<string, string> = {
  pending: 'bg-muted-foreground/40',
  queued: 'bg-amber-500',
  running: 'bg-blue-500',
  succeeded: 'bg-emerald-500',
  failed: 'bg-rose-500',
  cancelled: 'bg-muted-foreground/60',
}

const STATUS_LABEL_KEY: Record<string, string> = {
  pending: 'tasks.statusPending',
  queued: 'tasks.statusQueued',
  running: 'tasks.statusRunning',
  succeeded: 'tasks.statusSucceeded',
  failed: 'tasks.statusFailed',
  cancelled: 'tasks.statusCancelled',
}

const CHART_MARGIN = { top: 8, right: 4, bottom: 0, left: 4 } as const
const TICK_PROPS = { tickLine: false, axisLine: false, tickMargin: 6 } as const

type ThroughputChartPoint = { ts: number; count: number }

function errorMessage(err: unknown): string | undefined {
  return err instanceof Error ? err.message : undefined
}

function formatDuration(ms: number | null | undefined): string {
  if (ms == null || !Number.isFinite(ms)) return '—'
  if (ms < 1000) return `${Math.round(ms)}ms`
  if (ms < 60_000) return `${(ms / 1000).toFixed(1)}s`
  const m = Math.floor(ms / 60_000)
  const s = Math.round((ms % 60_000) / 1000)
  return `${m}m ${s}s`
}

function formatRate(rate: number | null | undefined): string {
  if (rate == null || !Number.isFinite(rate)) return '—'
  return `${(rate * 100).toFixed(1)}%`
}

function timeTick(v: number): string {
  return new Date(v).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

function formatThroughputLabel(payload: readonly unknown[]): string {
  const first = payload[0] as { payload?: ThroughputChartPoint } | undefined
  if (!first?.payload || !Number.isFinite(first.payload.ts)) return ''
  return new Date(first.payload.ts).toLocaleString()
}

export function TopicStatsPanel({ topicKey }: { topicKey: string }) {
  const { t } = useTranslation()
  const statsQuery = useQuery({
    queryKey: queryKeys.topics.stats(topicKey),
    queryFn: () => getTopicStats(topicKey),
  })

  if (statsQuery.isLoading) return <LoadingSkeleton rows={6} />
  if (statsQuery.isError || !statsQuery.data) {
    return (
      <ErrorBanner
        message={errorMessage(statsQuery.error) ?? t('common.errorGeneric')}
      />
    )
  }

  const stats = statsQuery.data
  const chartData = stats.throughput.map((p) => ({
    ts: Date.parse(p.ts),
    count: p.count,
  }))
  const chartConfig: ChartConfig = {
    count: { label: t('topics.statsThroughput'), color: 'var(--primary)' },
  }

  return (
    <div className='flex flex-col gap-4' data-testid='topic-stats-panel'>
      <div className={kit.statsWrap}>
        <div className={kit.statsGrid}>
          <div className={kit.statsCell[0]}>
            <p className={kit.statsLabel}>
              <ListTodo className='size-4' />
              {t('topics.statsTasks')}
            </p>
            <p className={kit.statsValue}>{stats.task_count}</p>
          </div>
          <div className={kit.statsCell[1]}>
            <p className={kit.statsLabel}>
              <Timer className='size-4' />
              {t('topics.statsSumRuntime')}
            </p>
            <p className={kit.statsValue}>
              {formatDuration(stats.runtime_ms.sum_ms)}
            </p>
          </div>
          <div className={kit.statsCell[2]}>
            <p className={kit.statsLabel}>
              <BadgeCheck className='size-4' />
              {t('topics.statsSuccessRate')}
            </p>
            <p className={kit.statsValue}>
              {formatRate(stats.success_rate)}
            </p>
          </div>
        </div>
      </div>

      <div className='flex min-w-0 flex-col rounded-[8px] border bg-card p-4 shadow-sm shadow-zinc-200/40 dark:border-white/10 dark:bg-[#161616] dark:shadow-none'>
        <div className='mb-3'>
          <h2 className='text-base font-semibold'>{t('topics.statsThroughput')}</h2>
          <div className='mt-3 flex flex-wrap items-center gap-4 text-[11px] text-muted-foreground'>
            <span className='inline-flex items-center gap-1.5'>
              <span
                className='size-2 rounded-full'
                style={{ backgroundColor: 'var(--primary)' }}
              />
              {t('topics.statsThroughput')}
            </span>
          </div>
        </div>
        <div className='h-[160px] w-full min-w-0'>
          <ChartContainer config={chartConfig} className='aspect-auto h-full w-full'>
            <AreaChart data={chartData} margin={CHART_MARGIN}>
              <CartesianGrid vertical={false} />
              <XAxis
                dataKey='ts'
                type='number'
                scale='time'
                domain={['dataMin', 'dataMax']}
                tickFormatter={timeTick}
                height={20}
                {...TICK_PROPS}
              />
              <YAxis width={32} allowDecimals={false} {...TICK_PROPS} />
              <ChartTooltip
                content={
                  <ChartTooltipContent
                    labelFormatter={(_, payload) =>
                      formatThroughputLabel(payload)
                    }
                  />
                }
              />
              <Area
                dataKey='count'
                type='monotone'
                stroke='var(--primary)'
                fill='var(--primary)'
                fillOpacity={0.15}
                strokeWidth={2}
                dot={false}
              />
            </AreaChart>
          </ChartContainer>
        </div>
      </div>

      <div className='grid gap-4 lg:grid-cols-2'>
        <div className='flex min-w-0 flex-col rounded-[8px] border bg-card p-4 shadow-sm shadow-zinc-200/40 dark:border-white/10 dark:bg-[#161616] dark:shadow-none'>
          <h2 className='mb-3 text-base font-semibold'>{t('topics.statsStatus')}</h2>
          <ul className='space-y-2'>
            {STATUS_ORDER.map((status) => {
              const count = stats.status[status] ?? 0
              const total = Math.max(stats.task_count, 1)
              const pct = Math.round((count / total) * 100)
              return (
                <li key={status} className='flex items-center gap-2 text-sm'>
                  <span className={cn('size-2 shrink-0 rounded-full', STATUS_COLOR[status])} />
                  <span className='w-24 shrink-0 truncate text-muted-foreground'>
                    {t(STATUS_LABEL_KEY[status])}
                  </span>
                  <div className='h-2 min-w-0 flex-1 overflow-hidden rounded-full bg-muted'>
                    <div
                      className={cn('h-full rounded-full', STATUS_COLOR[status])}
                      style={{ width: `${pct}%` }}
                    />
                  </div>
                  <span className='w-10 shrink-0 text-right tabular-nums'>{count}</span>
                </li>
              )
            })}
          </ul>
        </div>

        <div className='flex min-w-0 flex-col rounded-[8px] border bg-card p-4 shadow-sm shadow-zinc-200/40 dark:border-white/10 dark:bg-[#161616] dark:shadow-none'>
          <h2 className='mb-3 text-base font-semibold'>{t('topics.statsErrors')}</h2>
          {stats.error_codes.length === 0 ? (
            <p className='text-sm text-muted-foreground'>{t('topics.statsNoErrors')}</p>
          ) : (
            <ul className='space-y-2'>
              {stats.error_codes.map((e) => (
                <li key={e.code} className='flex items-center justify-between gap-2 text-sm'>
                  <span className='min-w-0 truncate font-mono text-xs'>{e.code}</span>
                  <span className='shrink-0 tabular-nums text-muted-foreground'>{e.count}</span>
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>
    </div>
  )
}
