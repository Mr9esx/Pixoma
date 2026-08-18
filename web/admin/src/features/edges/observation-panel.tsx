import { useState, type ReactNode } from 'react'
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
import type { EdgeMetricsResponse, TaskRecord } from '@/lib/api/types'
import { Button } from '@/components/ui/button'
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
import { EmptyState } from '@/components/feedback/empty-state'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { TaskDetailPanel } from '@/features/tasks/detail-panel'
import { taskStatusLabelKey } from '@/features/tasks/list-panel'
import { kit } from './kit-classes'
import {
  formatBytes,
  formatMetricValue,
  ioAxisTicks,
  parseMetrics,
  seriesStats,
  type MetricsPoint,
} from './observation'

// 基准高度的一半：h-[200px] sm:h-[240px] lg:h-[280px] → 50%
const CHART_HALF_HEIGHT = 'h-[100px] w-full min-w-0 sm:h-[120px] lg:h-[140px]'

// 统一图表内边距：顶部给 Y 轴最高刻度留白，底部最小化且各图一致
const CHART_MARGIN = { top: 12, right: 4, bottom: 0, left: 4 }
const TICK_PROPS = { tickLine: false, axisLine: false, tickMargin: 4 } as const

type StatItem = { label: string; value: string }

function timeTick(v: number): string {
  return new Date(v).toLocaleTimeString()
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
  t: (key: string) => string
): StatItem[] {
  const s = seriesStats(values, percentText)
  return [
    { label: t('edges.monitorCurrent'), value: s.current },
    { label: t('edges.monitorMax'), value: s.max },
    { label: t('edges.monitorAvg'), value: s.avg },
  ]
}

// I/O 系列右侧三值：当前 / 最高 / 平均（人类可读字节速率）。
function byteRateStats(
  values: Array<number | null>,
  series: string,
  t: (key: string) => string
): StatItem[] {
  const s = seriesStats(values, formatBytes)
  return [
    {
      label: `${series} · ${t('edges.monitorCurrent')}`,
      value: s.current,
    },
    { label: `${series} · ${t('edges.monitorMax')}`, value: s.max },
    { label: `${series} · ${t('edges.monitorAvg')}`, value: s.avg },
  ]
}

// Sprint health 卡片：标题在上，主体左图右值（桌面端右列 92px 统计）。
function LineCardShell({
  title,
  config,
  stats,
  children,
}: {
  title: string
  config: ChartConfig
  stats: StatItem[]
  children: ReactNode
}) {
  const legend = Object.entries(config)
  return (
    <div className='flex min-w-0 flex-1 flex-col rounded-[8px] border bg-card p-4 shadow-sm shadow-zinc-200/40 dark:shadow-none'>
      <div className='mb-3'>
        <h2 className='text-base font-semibold'>{title}</h2>
        {legend.length > 1 ? (
          <div className='mt-3 flex items-center gap-4 text-[11px] text-muted-foreground'>
            {legend.map(([key, entry]) => (
              <span key={key} className='inline-flex items-center gap-1.5'>
                <span
                  className='size-2 rounded-full'
                  style={{ backgroundColor: entry.color }}
                />
                {entry.label}
              </span>
            ))}
          </div>
        ) : null}
      </div>
      <div className='grid min-h-0 flex-1 gap-3 lg:grid-cols-[minmax(0,1fr)_92px]'>
        <div className={CHART_HALF_HEIGHT}>
          <ChartContainer config={config} className='h-full w-full'>
            {children}
          </ChartContainer>
        </div>
        <div className='grid grid-cols-3 content-center gap-2 text-center lg:grid-cols-1 lg:text-right'>
          {stats.map((stat, i) => (
            <div key={stat.label}>
              <p
                className={
                  i === 0
                    ? 'text-xl leading-6 font-semibold'
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
          width={48}
          {...TICK_PROPS}
        />
        <ChartTooltip
          content={
            <ChartTooltipContent
              formatter={(v, n) => chartTooltipFormatter(v, n, config)}
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
  const last = series[series.length - 1]
  const gpuNames = last?.gpus.map((gpu) => gpu.name) ?? []
  if (gpuNames.length === 0) return null
  return (
    <>
      {gpuNames.map((name, gi) => {
        const data = series.map((p) => ({
          time: p.time,
          usage: p.gpus[gi]?.usage_percent ?? null,
          vramPct: p.gpus[gi]?.vram_usage_percent ?? null,
          vramUsed: p.gpus[gi]?.vram_used_bytes ?? null,
        }))
        const usageConfig: ChartConfig = {
          usage: {
            label: t('edges.monitorGpuUsage'),
            color: 'var(--primary)',
          },
        }
        const vramConfig: ChartConfig = {
          rate: {
            label: t('edges.monitorVramRate'),
            color: 'var(--primary)',
          },
          used: {
            label: t('edges.monitorVram'),
            color: 'color-mix(in oklch, var(--primary) 75%, var(--background))',
          },
        }
        return (
          <div
            key={`${name}-${gi}`}
            className='flex flex-col gap-4 xl:flex-row'
          >
            <LineCardShell
              title={`${t('edges.monitorGpuUsage')} · ${name}`}
              config={usageConfig}
              stats={rateStats(
                series.map((p) => p.gpus[gi]?.usage_percent ?? null),
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
                      formatter={(v, n) =>
                        chartTooltipFormatter(v, n, usageConfig)
                      }
                    />
                  }
                />
                <Line
                  dataKey='usage'
                  type='natural'
                  stroke='var(--color-usage)'
                  strokeWidth={2}
                  dot={false}
                />
              </LineChart>
            </LineCardShell>
            <LineCardShell
              title={`${t('edges.monitorVramRate')} · ${name}`}
              config={vramConfig}
              stats={rateStats(
                series.map((p) => p.gpus[gi]?.vram_usage_percent ?? null),
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
                  width={48}
                  {...TICK_PROPS}
                />
                <ChartTooltip
                  content={
                    <ChartTooltipContent
                      formatter={(v, n) =>
                        chartTooltipFormatter(v, n, vramConfig)
                      }
                    />
                  }
                />
                <Line
                  yAxisId='rate'
                  dataKey='vramPct'
                  type='natural'
                  stroke='var(--color-rate)'
                  strokeWidth={2}
                  dot={false}
                />
                <Line
                  yAxisId='used'
                  dataKey='vramUsed'
                  type='natural'
                  stroke='var(--color-used)'
                  strokeWidth={2}
                  dot={false}
                />
              </LineChart>
            </LineCardShell>
          </div>
        )
      })}
    </>
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
      stats={[
        ...byteRateStats(
          series.map((p) => p.ioRead),
          t('edges.monitorIoRead'),
          t
        ),
        ...byteRateStats(
          series.map((p) => p.ioWrite),
          t('edges.monitorIoWrite'),
          t
        ),
      ]}
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

function SectionHead({ title, hint }: { title: string; hint: string }) {
  return (
    <div className='flex flex-col gap-1'>
      <div className='flex items-center gap-3'>
        <h2 className={kit.sectionTitle}>{title}</h2>
        <span className={kit.sectionDash} />
      </div>
      <p className='text-xs text-muted-foreground'>{hint}</p>
    </div>
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
      {tasks.length === 0 ? (
        <EmptyState className='py-6' message={t('tasks.empty')} />
      ) : (
        <>
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
                {tasks.map((task) => {
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
                })}
              </tbody>
            </table>
          </div>
        </>
      )}
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
}: {
  data: ReturnType<typeof parseMetrics>
}) {
  const { t } = useTranslation()
  const hasData = data.series.length > 0
  return (
    <section className='flex flex-col gap-4'>
      <SectionHead
        title={t('edges.observationSystem')}
        hint={t('edges.observationSystemHint')}
      />
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
  tasksQuery: QueryView<TaskRecord[] | undefined>
  tasksPagination: TasksPagination
}

export function ObservationPanel({
  metricsQuery,
  tasksQuery,
  tasksPagination,
}: Props) {
  return (
    <div className='flex flex-col gap-7' data-testid='edge-observation'>
      <BlockBody query={metricsQuery}>
        <MonitoringSection data={parseMetrics(metricsQuery.data)} />
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
