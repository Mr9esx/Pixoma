import { Fragment, useState, type ReactNode } from 'react'
import { Link } from '@tanstack/react-router'
import { Activity } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import {
  Area,
  AreaChart,
  CartesianGrid,
  Line,
  LineChart,
  XAxis,
  YAxis,
} from 'recharts'
import type { MetricsRange, MetricsWindow } from '@/lib/api/edges'
import type { EdgeMetricsResponse, TaskRecord } from '@/lib/api/types'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Calendar } from '@/components/ui/calendar'
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from '@/components/ui/chart'
import { Dialog, DialogContent } from '@/components/ui/dialog'
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import { EmptyState } from '@/components/feedback/empty-state'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { SectionHead } from '@/components/section-head'
import { TaskDetailPanel } from '@/features/tasks/detail-panel'
import { taskStatusLabelKey } from '@/features/tasks/list-panel'
import { kit } from './kit-classes'
import {
  applyTimeToDate,
  formatFullTime,
  formatBytes,
  formatMetricValue,
  formatTimeInput,
  ioAxisTicks,
  parseMetrics,
  seriesStats,
  type MetricsPoint,
} from './observation'

// 统一图表内边距：顶部给 Y 轴最高刻度留白，底部最小化且各图一致
const CHART_MARGIN = { top: 12, right: 4, bottom: 0, left: 4 }
const TICK_PROPS = { tickLine: false, axisLine: false, tickMargin: 4 } as const

// 多 GPU 折线配色：第一块沿用主题主色，后续走 chart 调色板。
const GPU_SERIES_COLORS = [
  'var(--primary)',
  'var(--chart-2)',
  'var(--chart-3)',
  'var(--chart-4)',
  'var(--chart-5)',
]

type StatItem = { label: string; value: string }

function timeTick(v: number): string {
  return formatFullTime(v)
}

function tooltipLabelFormatter(
  _value: unknown,
  payload: readonly unknown[]
): ReactNode {
  const first = payload[0] as { payload?: { time?: number } } | undefined
  if (!first?.payload?.time) return null
  return formatFullTime(first.payload.time)
}

function chartTooltipFormatter(
  value: unknown,
  name: unknown,
  config: ChartConfig
): ReactNode {
  const key = String(name)
  const label = config[key]?.label ?? key
  return `${label}: ${formatMetricValue(value, key)}`
}

function percentText(v: number): string {
  return `${v.toFixed(1)}%`
}

// 占用率系列右侧三值：当前 / 最高 / 平均（百分比）。
function rateStats(
  values: Array<number | null>,
  t: (key: string) => string,
  prefix?: string
): StatItem[] {
  const s = seriesStats(values, percentText)
  const label = (key: string) => (prefix ? `${prefix} · ${t(key)}` : t(key))
  return [
    { label: label('edges.monitorCurrent'), value: s.current },
    { label: label('edges.monitorMax'), value: s.max },
    { label: label('edges.monitorAvg'), value: s.avg },
  ]
}

// I/O 卡右侧：当前读/写、最高读/写、平均读/写，每个值明确归属。
function ioStats(
  read: Array<number | null>,
  write: Array<number | null>,
  t: (key: string) => string
): StatItem[] {
  const r = seriesStats(read, formatBytes)
  const w = seriesStats(write, formatBytes)
  return [
    { label: t('edges.monitorIoReadCurrent'), value: r.current },
    { label: t('edges.monitorIoWriteCurrent'), value: w.current },
    { label: t('edges.monitorIoReadMax'), value: r.max },
    { label: t('edges.monitorIoWriteMax'), value: w.max },
    { label: t('edges.monitorIoReadAvg'), value: r.avg },
    { label: t('edges.monitorIoWriteAvg'), value: w.avg },
  ]
}

// Sprint health 卡片：标题在上，主体左图右值（桌面端右列 92px 统计）。
function LineCardShell({
  title,
  config,
  stats,
  compact = false,
  children,
}: {
  title: string
  config: ChartConfig
  stats: StatItem[]
  compact?: boolean
  children: ReactNode
}) {
  return (
    <div className='flex min-w-0 flex-1 flex-col rounded-[8px] border bg-card p-4 shadow-sm shadow-zinc-200/40 dark:border-white/10 dark:bg-[#161616] dark:shadow-none'>
      <div className='mb-3 flex flex-wrap items-start justify-between gap-3'>
        <div>
          <h2 className='text-base font-semibold'>{title}</h2>
          <div className='mt-3 flex flex-wrap items-center gap-4 text-[11px] text-muted-foreground'>
            {Object.entries(config).map(([key, entry]) => (
              <span key={key} className='inline-flex items-center gap-1.5'>
                <span
                  className='size-2 rounded-full'
                  style={{ backgroundColor: entry.color }}
                />
                {entry.label}
              </span>
            ))}
          </div>
        </div>
      </div>
      <div
        className={
          compact
            ? 'grid min-h-0 flex-1 gap-1.5 lg:grid-cols-[minmax(0,1fr)_122px]'
            : 'grid min-h-0 flex-1 gap-1.5 lg:grid-cols-[minmax(0,1fr)_72px]'
        }
      >
        <div className='h-full min-h-[120px] w-full min-w-0'>
          <ChartContainer config={config} className='aspect-auto h-full w-full'>
            {children}
          </ChartContainer>
        </div>
        <div
          className={
            compact
              ? 'grid grid-cols-2 content-center gap-1.5 text-center lg:text-right'
              : 'grid grid-cols-3 content-center gap-2 text-center lg:grid-cols-1 lg:text-right'
          }
        >
          {stats.map((stat) => (
            <div key={stat.label}>
              <p
                className={
                  compact
                    ? 'text-xs leading-5 font-semibold'
                    : 'text-lg leading-6 font-semibold'
                }
              >
                {stat.value}
              </p>
              <p className='text-[11px] text-muted-foreground'>{stat.label}</p>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}

export function CpuCard({ series }: { series: MetricsPoint[] }) {
  const { t } = useTranslation()
  const data = series.map((p) => ({ time: p.time, cpu: p.cpu }))
  const config: ChartConfig = {
    cpu: { label: t('edges.monitorCpu'), color: 'var(--primary)' },
  }
  return (
    <LineCardShell
      title={t('edges.monitorCpu')}
      config={config}
      stats={rateStats(
        series.map((p) => p.cpu),
        t
      )}
    >
      <LineChart data={data} margin={CHART_MARGIN}>
        <CartesianGrid vertical={false} />
        <XAxis
          dataKey='time'
          tickFormatter={timeTick}
          {...TICK_PROPS}
          height={20}
        />
        <YAxis
          width={36}
          domain={[0, 100]}
          tickFormatter={(v: number) => `${v}%`}
          {...TICK_PROPS}
        />
        <ChartTooltip
          content={
            <ChartTooltipContent
              formatter={(v, n) => chartTooltipFormatter(v, n, config)}
              labelFormatter={tooltipLabelFormatter}
            />
          }
        />
        <Line
          dataKey='cpu'
          type='natural'
          stroke='var(--color-cpu)'
          strokeWidth={2}
          dot={false}
        />
      </LineChart>
    </LineCardShell>
  )
}

export function MemRateCard({ series }: { series: MetricsPoint[] }) {
  const { t } = useTranslation()
  const data = series.map((p) => ({
    time: p.time,
    rate: p.memPct,
    used: p.memUsed,
  }))
  const config: ChartConfig = {
    rate: {
      label: t('edges.monitorMemRate'),
      color: 'var(--primary)',
    },
    used: {
      label: t('edges.monitorMemUsed'),
      color: 'color-mix(in oklch, var(--primary) 75%, var(--background))',
    },
  }
  return (
    <LineCardShell
      title={t('edges.monitorMemRate')}
      config={config}
      stats={rateStats(
        series.map((p) => p.memPct),
        t
      )}
    >
      <LineChart data={data} margin={CHART_MARGIN}>
        <CartesianGrid vertical={false} />
        <XAxis
          dataKey='time'
          tickFormatter={timeTick}
          {...TICK_PROPS}
          height={20}
        />
        <YAxis
          yAxisId='rate'
          width={36}
          domain={[0, 100]}
          tickFormatter={(v: number) => `${v}%`}
          {...TICK_PROPS}
        />
        <YAxis
          yAxisId='used'
          orientation='right'
          tickFormatter={(v: number) => formatBytes(v)}
          width={70}
          {...TICK_PROPS}
        />
        <ChartTooltip
          content={
            <ChartTooltipContent
              formatter={(v, n) => chartTooltipFormatter(v, n, config)}
              labelFormatter={tooltipLabelFormatter}
            />
          }
        />
        <Line
          yAxisId='rate'
          dataKey='rate'
          type='natural'
          stroke='var(--color-rate)'
          strokeWidth={2}
          dot={false}
        />
        <Line
          yAxisId='used'
          dataKey='used'
          type='natural'
          stroke='var(--color-used)'
          strokeWidth={2}
          dot={false}
        />
      </LineChart>
    </LineCardShell>
  )
}

export function GpuLineCards({ series }: { series: MetricsPoint[] }) {
  const { t } = useTranslation()
  const last =
    [...series].reverse().find((p) => p.gpus.length > 0) ??
    series[series.length - 1]
  const gpuNames = last?.gpus.map((gpu) => gpu.name) ?? []
  if (gpuNames.length === 0) return null
  const multi = gpuNames.length > 1
  const gpuPrefix = (gi: number) => (multi ? `GPU${gi}` : undefined)

  const usageData = series.map((p) => {
    const row: Record<string, number | null> = { time: p.time }
    gpuNames.forEach((_, gi) => {
      row[`gpu${gi}`] = p.gpus[gi]?.usage_percent ?? null
    })
    return row
  })
  const usageConfig: ChartConfig = {}
  gpuNames.forEach((name, gi) => {
    usageConfig[`gpu${gi}`] = {
      label: name,
      color: GPU_SERIES_COLORS[gi % GPU_SERIES_COLORS.length],
    }
  })

  const vramData = series.map((p) => {
    const row: Record<string, number | null> = { time: p.time }
    gpuNames.forEach((_, gi) => {
      const gpu = p.gpus[gi]
      row[`gpu${gi}Rate`] = gpu?.vram_usage_percent ?? null
      row[`gpu${gi}Used`] = gpu?.vram_used_bytes ?? null
    })
    return row
  })
  const vramConfig: ChartConfig = {}
  gpuNames.forEach((name, gi) => {
    const color = GPU_SERIES_COLORS[gi % GPU_SERIES_COLORS.length]
    const label = (key: string) => (multi ? `${name} · ${t(key)}` : t(key))
    vramConfig[`gpu${gi}Rate`] = {
      label: label('edges.monitorVramRate'),
      color,
    }
    vramConfig[`gpu${gi}Used`] = {
      label: label('edges.monitorVram'),
      color: `color-mix(in oklch, ${color} 75%, var(--background))`,
    }
  })

  return (
    <div className='flex flex-col gap-4 xl:flex-row'>
      <LineCardShell
        title={t('edges.monitorGpuUsage')}
        config={usageConfig}
        stats={gpuNames.flatMap((_, gi) =>
          rateStats(
            series.map((p) => p.gpus[gi]?.usage_percent ?? null),
            t,
            gpuPrefix(gi)
          )
        )}
      >
        <LineChart data={usageData} margin={CHART_MARGIN}>
          <CartesianGrid vertical={false} />
          <XAxis
            dataKey='time'
            tickFormatter={timeTick}
            {...TICK_PROPS}
            height={20}
          />
          <YAxis
            width={36}
            domain={[0, 100]}
            tickFormatter={(v: number) => `${v}%`}
            {...TICK_PROPS}
          />
          <ChartTooltip
            content={
              <ChartTooltipContent
                formatter={(v, n) => chartTooltipFormatter(v, n, usageConfig)}
                labelFormatter={tooltipLabelFormatter}
              />
            }
          />
          {gpuNames.map((_, gi) => (
            <Line
              key={gi}
              dataKey={`gpu${gi}`}
              type='natural'
              stroke={`var(--color-gpu${gi})`}
              strokeWidth={2}
              dot={false}
            />
          ))}
        </LineChart>
      </LineCardShell>
      <LineCardShell
        title={t('edges.monitorVramRate')}
        config={vramConfig}
        stats={gpuNames.flatMap((_, gi) =>
          rateStats(
            series.map((p) => p.gpus[gi]?.vram_usage_percent ?? null),
            t,
            gpuPrefix(gi)
          )
        )}
      >
        <LineChart data={vramData} margin={CHART_MARGIN}>
          <CartesianGrid vertical={false} />
          <XAxis
            dataKey='time'
            tickFormatter={timeTick}
            {...TICK_PROPS}
            height={20}
          />
          <YAxis
            yAxisId='rate'
            width={36}
            domain={[0, 100]}
            tickFormatter={(v: number) => `${v}%`}
            {...TICK_PROPS}
          />
          <YAxis
            yAxisId='used'
            orientation='right'
            tickFormatter={(v: number) => formatBytes(v)}
            width={70}
            {...TICK_PROPS}
          />
          <ChartTooltip
            content={
              <ChartTooltipContent
                formatter={(v, n) => chartTooltipFormatter(v, n, vramConfig)}
                labelFormatter={tooltipLabelFormatter}
              />
            }
          />
          {gpuNames.map((_, gi) => (
            <Fragment key={gi}>
              <Line
                yAxisId='rate'
                dataKey={`gpu${gi}Rate`}
                type='natural'
                stroke={`var(--color-gpu${gi}Rate)`}
                strokeWidth={2}
                dot={false}
              />
              <Line
                yAxisId='used'
                dataKey={`gpu${gi}Used`}
                type='natural'
                stroke={`var(--color-gpu${gi}Used)`}
                strokeWidth={2}
                dot={false}
              />
            </Fragment>
          ))}
        </LineChart>
      </LineCardShell>
    </div>
  )
}

export function IOCard({ series }: { series: MetricsPoint[] }) {
  const { t } = useTranslation()
  const data = series.map((p) => ({
    time: p.time,
    read: p.ioRead,
    write: p.ioWrite,
  }))
  const ioAxis = ioAxisTicks(
    data
      .flatMap((d) => [d.read, d.write])
      .filter((v): v is number => typeof v === 'number' && Number.isFinite(v))
  )
  const config: ChartConfig = {
    read: {
      label: t('edges.monitorIoRead'),
      color: 'var(--primary)',
    },
    write: {
      label: t('edges.monitorIoWrite'),
      color: 'color-mix(in oklch, var(--primary) 75%, var(--background))',
    },
  }
  return (
    <LineCardShell
      title={t('edges.monitorIo')}
      config={config}
      stats={ioStats(
        series.map((p) => p.ioRead),
        series.map((p) => p.ioWrite),
        t
      )}
      compact
    >
      <AreaChart data={data} margin={CHART_MARGIN}>
        <defs>
          <linearGradient id='readGradient' x1='0' y1='0' x2='0' y2='1'>
            <stop offset='0%' stopColor='var(--color-read)' stopOpacity={0.3} />
            <stop
              offset='100%'
              stopColor='var(--color-read)'
              stopOpacity={0.05}
            />
          </linearGradient>
          <linearGradient id='writeGradient' x1='0' y1='0' x2='0' y2='1'>
            <stop
              offset='0%'
              stopColor='var(--color-write)'
              stopOpacity={0.2}
            />
            <stop
              offset='100%'
              stopColor='var(--color-write)'
              stopOpacity={0.02}
            />
          </linearGradient>
        </defs>
        <CartesianGrid vertical={false} />
        <XAxis
          dataKey='time'
          tickFormatter={timeTick}
          {...TICK_PROPS}
          height={20}
        />
        <YAxis
          ticks={ioAxis.ticks}
          tickFormatter={ioAxis.format}
          width={48}
          {...TICK_PROPS}
        />
        <ChartTooltip
          content={
            <ChartTooltipContent
              formatter={(v, n) => chartTooltipFormatter(v, n, config)}
              labelFormatter={tooltipLabelFormatter}
            />
          }
        />
        <Area
          dataKey='read'
          type='natural'
          fill='url(#readGradient)'
          stroke='var(--color-read)'
          strokeWidth={2}
        />
        <Area
          dataKey='write'
          type='natural'
          fill='url(#writeGradient)'
          stroke='var(--color-write)'
          strokeWidth={2}
        />
      </AreaChart>
    </LineCardShell>
  )
}

function errorMessage(err: unknown): string | undefined {
  return err instanceof Error ? err.message : undefined
}

type QueryView<T> = {
  isLoading: boolean
  isError: boolean
  error: unknown
  data: T
  refetch: () => void
}

// 单个 time input（HH:MM），图标由全局 CSS 隐藏，体积比两个下拉小。
function TimeInput({
  value,
  onChange,
  className,
}: {
  value: string
  onChange: (time: string) => void
  className?: string
}) {
  return (
    <Input
      type='time'
      step={60}
      className={cn('h-7 w-full px-2 text-xs', className)}
      value={value}
      onChange={(e) => onChange(e.target.value)}
    />
  )
}

function formatTime(iso: string): string {
  const ms = Date.parse(iso)
  if (!Number.isFinite(ms)) return iso
  return new Date(ms).toLocaleString()
}

function statusTagClass(status: string): string {
  if (status === 'succeeded') return kit.tagSmOn
  if (status === 'failed') return kit.tagSmFail
  return kit.tagSmOff
}

export type TasksPagination = {
  offset: number
  pageSize: number
  onOffsetChange: (offset: number) => void
}

function TasksSection({
  tasks,
  pagination,
}: {
  tasks: TaskRecord[]
  pagination: TasksPagination
}) {
  const { t } = useTranslation()
  const [detailId, setDetailId] = useState<string | null>(null)
  const page = Math.floor(pagination.offset / pagination.pageSize) + 1
  const hasPrev = pagination.offset > 0
  const hasNext = tasks.length >= pagination.pageSize
  return (
    <section className='flex flex-col gap-4'>
      <SectionHead
        title={t('edges.observationTasks')}
        hint={t('edges.observationTasksHint')}
      />
      <div className={kit.tableWrap}>
        <table className='w-full caption-bottom text-sm'>
          <thead>
            <tr className='border-b bg-muted/25'>
              <th className={`${kit.th} px-4`}>{t('tasks.fieldStatus')}</th>
              <th className={kit.th}>{t('tasks.fieldCaseId')}</th>
              <th className={kit.th}>{t('tasks.fieldId')}</th>
              <th className={kit.th}>{t('tasks.fieldUpdatedAt')}</th>
              <th className={kit.th}>{t('edges.taskDetail')}</th>
            </tr>
          </thead>
          <tbody>
            {tasks.length === 0 ? (
              <tr>
                <td colSpan={5} className='p-0'>
                  <EmptyState
                    className='py-6'
                    message={t('tasks.empty')}
                  />
                </td>
              </tr>
            ) : (
              tasks.map((task) => {
                const statusKey = taskStatusLabelKey(task.status)
                return (
                  <tr key={task.id} className='border-b last:border-0'>
                    <td className='px-4 py-3'>
                      <span className={statusTagClass(task.status)}>
                        {statusKey ? t(statusKey) : task.status}
                      </span>
                    </td>
                    <td className='max-w-32 truncate py-3'>{task.case_id}</td>
                    <td className='max-w-40 truncate py-3 font-mono text-xs'>
                      <Link
                        to='/tasks/$taskId'
                        params={{ taskId: task.id }}
                        className='underline-offset-4 hover:underline'
                      >
                        {task.id}
                      </Link>
                    </td>
                    <td className='py-3 whitespace-nowrap text-muted-foreground'>
                      {formatTime(task.updated_at)}
                    </td>
                    <td className='py-3'>
                      <Button
                        type='button'
                        variant='outline'
                        size='sm'
                        onClick={() => setDetailId(task.id)}
                      >
                        {t('edges.taskDetail')}
                      </Button>
                    </td>
                  </tr>
                )
              })
            )}
          </tbody>
        </table>
      </div>
      {hasPrev || hasNext ? (
        <div className='flex items-center justify-between gap-2'>
          <span className='text-xs text-muted-foreground'>
            {t('edges.tasksPage', { page })}
          </span>
          <div className='flex gap-2'>
            <Button
              type='button'
              variant='outline'
              size='sm'
              disabled={!hasPrev}
              onClick={() =>
                pagination.onOffsetChange(
                  pagination.offset - pagination.pageSize
                )
              }
            >
              {t('edges.prevPage')}
            </Button>
            <Button
              type='button'
              variant='outline'
              size='sm'
              disabled={!hasNext}
              onClick={() =>
                pagination.onOffsetChange(
                  pagination.offset + pagination.pageSize
                )
              }
            >
              {t('edges.nextPage')}
            </Button>
          </div>
        </div>
      ) : null}
      <Dialog
        open={detailId != null}
        onOpenChange={(open) => {
          if (!open) setDetailId(null)
        }}
      >
        <DialogContent className='max-h-[90vh] overflow-y-auto sm:max-w-lg'>
          {detailId ? <TaskDetailPanel id={detailId} /> : null}
        </DialogContent>
      </Dialog>
    </section>
  )
}

function MonitoringSection({
  data,
  metricsRange,
  onMetricsRangeChange,
}: {
  data: ReturnType<typeof parseMetrics>
  metricsRange: MetricsRange
  onMetricsRangeChange: (range: MetricsRange) => void
}) {
  const { t } = useTranslation()
  const hasData = data.series.length > 0
  const [customOpen, setCustomOpen] = useState(false)
  const [draftRange, setDraftRange] = useState<{
    from: Date
    to: Date
  } | null>(null)
  const [draftFromTime, setDraftFromTime] = useState('00:00')
  const [draftToTime, setDraftToTime] = useState('23:59')
  const [customError, setCustomError] = useState<string | null>(null)

  const openCustom = () => {
    const active =
      metricsRange.kind === 'custom'
        ? {
            from: new Date(Date.parse(metricsRange.from)),
            to: new Date(Date.parse(metricsRange.to)),
          }
        : {
            from: new Date(Date.now() - 3600_000),
            to: new Date(),
          }
    setDraftRange({ from: active.from, to: active.to })
    setDraftFromTime(formatTimeInput(active.from.getTime()))
    setDraftToTime(formatTimeInput(active.to.getTime()))
    setCustomError(null)
  }

  const applyCustom = () => {
    if (!draftRange) {
      setCustomError(t('edges.monitorCustomInvalid'))
      return
    }
    const from = applyTimeToDate(draftRange.from, draftFromTime)
    const to = applyTimeToDate(draftRange.to, draftToTime)
    if (!from || !to || from.getTime() >= to.getTime()) {
      setCustomError(t('edges.monitorCustomInvalid'))
      return
    }
    onMetricsRangeChange({
      kind: 'custom',
      from: from.toISOString(),
      to: to.toISOString(),
    })
    setCustomOpen(false)
  }

  const activePreset =
    metricsRange.kind === 'preset' ? metricsRange.window : null
  const activeCustom =
    metricsRange.kind === 'custom'
      ? `${formatFullTime(Date.parse(metricsRange.from))} ~ ${formatFullTime(Date.parse(metricsRange.to))}`
      : null
  return (
    <section className='flex flex-col gap-4'>
      <SectionHead
        title={t('edges.observationSystem')}
        hint={t('edges.observationSystemHint')}
      />
      <Popover
        open={customOpen}
        onOpenChange={(open) => {
          if (open) openCustom()
          setCustomOpen(open)
        }}
      >
        <div className='flex flex-col items-end gap-1'>
          <div
            className='flex items-center gap-1 rounded-md border bg-muted/40 p-0.5'
            role='group'
            aria-label={t('edges.monitorRange')}
          >
            {(['1h', '6h', '24h'] as MetricsWindow[]).map((w) => (
              <button
                key={w}
                type='button'
                onClick={() =>
                  onMetricsRangeChange({ kind: 'preset', window: w })
                }
                className={cn(
                  'rounded px-2 py-1 text-xs font-medium transition-colors',
                  activePreset === w
                    ? 'bg-background text-foreground shadow-sm'
                    : 'text-muted-foreground hover:text-foreground'
                )}
              >
                {w}
              </button>
            ))}
            <PopoverTrigger asChild>
              <button
                type='button'
                className={cn(
                  'rounded px-2 py-1 text-xs font-medium transition-colors',
                  metricsRange.kind === 'custom'
                    ? 'bg-background text-foreground shadow-sm'
                    : 'text-muted-foreground hover:text-foreground'
                )}
              >
                {t('edges.monitorCustom')}
              </button>
            </PopoverTrigger>
          </div>
          {activeCustom ? (
            <span className='max-w-56 truncate text-[11px] text-muted-foreground'>
              {activeCustom}
            </span>
          ) : null}
        </div>
        <PopoverContent align='end' className='w-auto p-0'>
          <Calendar
            mode='range'
            defaultMonth={draftRange?.from}
            selected={
              draftRange
                ? { from: draftRange.from, to: draftRange.to }
                : undefined
            }
            onSelect={(sel) => {
              if (sel?.from) {
                setDraftRange({ from: sel.from, to: sel.to ?? sel.from })
              }
            }}
            disabled={(date: Date) => date > new Date()}
          />
          <div className='px-3 pb-3'>
            <div className='mt-2 flex flex-col gap-1'>
              <div className='flex items-center gap-2'>
                <Label className='min-w-0 flex-1 text-center text-xs'>
                  {t('edges.monitorCustomFrom')}
                </Label>
                <span className='w-4 shrink-0' />
                <Label className='min-w-0 flex-1 text-center text-xs'>
                  {t('edges.monitorCustomTo')}
                </Label>
              </div>
              <div className='flex items-center gap-2'>
                <TimeInput
                  className='min-w-0 flex-1'
                  value={draftFromTime}
                  onChange={setDraftFromTime}
                />
                <span className='w-4 shrink-0 text-center text-muted-foreground'>
                  ~
                </span>
                <TimeInput
                  className='min-w-0 flex-1'
                  value={draftToTime}
                  onChange={setDraftToTime}
                />
              </div>
            </div>
            {customError ? (
              <p className='mt-1 text-xs text-destructive'>{customError}</p>
            ) : null}
            <div className='mt-5 flex justify-end gap-2'>
              <Button
                type='button'
                variant='outline'
                size='sm'
                onClick={() => setCustomOpen(false)}
              >
                {t('common.cancel')}
              </Button>
              <Button type='button' size='sm' onClick={applyCustom}>
                {t('edges.monitorCustomApply')}
              </Button>
            </div>
          </div>
        </PopoverContent>
      </Popover>
      {!hasData ? (
        <Empty className='border p-3 md:p-6'>
          <EmptyHeader className='max-w-none'>
            <EmptyMedia variant='icon'>
              <Activity />
            </EmptyMedia>
            <EmptyTitle className='text-sm font-medium'>
              {t('edges.monitorEmpty')}
            </EmptyTitle>
            <EmptyDescription className='whitespace-nowrap'>
              {t('edges.monitorEmptyHint')}
            </EmptyDescription>
          </EmptyHeader>
        </Empty>
      ) : (
        <>
          <div className='flex flex-col gap-4 xl:flex-row'>
            <CpuCard series={data.series} />
            <MemRateCard series={data.series} />
          </div>
          <GpuLineCards series={data.series} />
          <IOCard series={data.series} />
        </>
      )}
    </section>
  )
}

function BlockBody({
  query,
  children,
}: {
  query: QueryView<unknown>
  children: ReactNode
}) {
  if (query.isError) {
    return (
      <ErrorBanner
        message={errorMessage(query.error)}
        onRetry={() => void query.refetch()}
      />
    )
  }
  if (query.isLoading) return <LoadingSkeleton rows={3} />
  return children
}

type Props = {
  metricsQuery: QueryView<EdgeMetricsResponse | undefined>
  metricsRange: MetricsRange
  onMetricsRangeChange: (range: MetricsRange) => void
  tasksQuery: QueryView<TaskRecord[] | undefined>
  tasksPagination: TasksPagination
}

export function ObservationPanel({
  metricsQuery,
  metricsRange,
  onMetricsRangeChange,
  tasksQuery,
  tasksPagination,
}: Props) {
  return (
    <div className='flex flex-col gap-7' data-testid='edge-observation'>
      <BlockBody query={metricsQuery}>
        <MonitoringSection
          data={parseMetrics(metricsQuery.data)}
          metricsRange={metricsRange}
          onMetricsRangeChange={onMetricsRangeChange}
        />
      </BlockBody>
      <BlockBody query={tasksQuery}>
        <TasksSection
          tasks={tasksQuery.data ?? []}
          pagination={tasksPagination}
        />
      </BlockBody>
    </div>
  )
}
