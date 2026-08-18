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
import { formatBytes, parseMetrics, type MetricsPoint } from './observation'

// 基准高度的一半：h-[200px] sm:h-[240px] lg:h-[280px] → 50%
const CHART_HALF_HEIGHT =
  'h-[100px] w-full min-w-0 sm:h-[120px] lg:h-[140px]'

function timeTick(v: number): string {
  return new Date(v).toLocaleTimeString()
}

function LineCardShell({
  title,
  value,
  config,
  children,
}: {
  title: string
  value: string
  config: ChartConfig
  children: ReactNode
}) {
  return (
    <div className='flex min-w-0 flex-1 flex-col gap-4 rounded-xl border bg-card p-4 sm:gap-6 sm:p-6'>
      <div className='flex flex-wrap items-center gap-2 sm:gap-4'>
        <div className='flex flex-1 flex-col gap-1'>
          <p className='text-xl leading-tight font-semibold tracking-tight sm:text-2xl'>
            {value}
          </p>
          <p className='text-xs text-muted-foreground'>{title}</p>
        </div>
        <div className='hidden items-center gap-3 sm:flex sm:gap-5'>
          {Object.entries(config).map(([key, entry]) => (
            <div
              key={key}
              className='flex items-center gap-1.5 transition-opacity duration-200 motion-reduce:transition-none'
            >
              <div
                className='size-2.5 rounded-full sm:size-3'
                style={{ backgroundColor: entry.color }}
              />
              <span className='text-[10px] text-muted-foreground sm:text-xs'>
                {entry.label}
              </span>
            </div>
          ))}
        </div>
      </div>
      <div className={CHART_HALF_HEIGHT}>
        <ChartContainer config={config} className='h-full w-full'>
          {children}
        </ChartContainer>
      </div>
    </div>
  )
}

export function CpuCard({ series }: { series: MetricsPoint[] }) {
  const { t } = useTranslation()
  const latest = series[series.length - 1]
  const data = series.map((p) => ({ time: p.time, cpu: p.cpu }))
  return (
    <LineCardShell
      title={t('edges.monitorCpu')}
      value={latest?.cpu != null ? `${latest.cpu.toFixed(1)}%` : '—'}
      config={{
        cpu: { label: t('edges.monitorCpu'), color: 'var(--primary)' },
      }}
    >
      <LineChart data={data} margin={{ left: 12, right: 12 }}>
        <CartesianGrid vertical={false} />
        <XAxis dataKey='time' tickFormatter={timeTick} tickLine={false} axisLine={false} />
        <YAxis
          domain={[0, 100]}
          tickFormatter={(v: number) => `${v}%`}
          tickLine={false}
          axisLine={false}
        />
        <ChartTooltip content={<ChartTooltipContent />} />
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
  const latest = series[series.length - 1]
  const data = series.map((p) => ({
    time: p.time,
    rate: p.memPct,
    used: p.memUsed,
  }))
  return (
    <LineCardShell
      title={t('edges.monitorMemRate')}
      value={latest?.memPct != null ? `${latest.memPct.toFixed(1)}%` : '—'}
      config={{
        rate: {
          label: t('edges.monitorMemRate'),
          color: 'var(--primary)',
        },
        used: {
          label: t('edges.monitorMemUsed'),
          color: 'color-mix(in oklch, var(--primary) 75%, var(--background))',
        },
      }}
    >
      <LineChart data={data} margin={{ left: 12, right: 12 }}>
        <CartesianGrid vertical={false} />
        <XAxis dataKey='time' tickFormatter={timeTick} tickLine={false} axisLine={false} />
        <YAxis
          yAxisId='rate'
          domain={[0, 100]}
          tickFormatter={(v: number) => `${v}%`}
          tickLine={false}
          axisLine={false}
        />
        <YAxis
          yAxisId='used'
          orientation='right'
          tickFormatter={(v: number) => formatBytes(v)}
          tickLine={false}
          axisLine={false}
          width={60}
        />
        <ChartTooltip content={<ChartTooltipContent />} />
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
        const latestGpu = last?.gpus[gi]
        const usage = latestGpu?.usage_percent
        const vramPct = latestGpu?.vram_usage_percent
        return (
          <div
            key={`${name}-${gi}`}
            className='flex flex-col gap-4 xl:flex-row'
          >
            <LineCardShell
              title={`${t('edges.monitorGpuUsage')} · ${name}`}
              value={usage != null ? `${usage.toFixed(1)}%` : '—'}
              config={{
                usage: {
                  label: t('edges.monitorGpuUsage'),
                  color: 'var(--primary)',
                },
              }}
            >
              <LineChart data={data} margin={{ left: 12, right: 12 }}>
                <CartesianGrid vertical={false} />
                <XAxis dataKey='time' tickFormatter={timeTick} tickLine={false} axisLine={false} />
                <YAxis
                  domain={[0, 100]}
                  tickFormatter={(v: number) => `${v}%`}
                  tickLine={false}
                  axisLine={false}
                />
                <ChartTooltip content={<ChartTooltipContent />} />
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
              value={vramPct != null ? `${vramPct.toFixed(1)}%` : '—'}
              config={{
                rate: {
                  label: t('edges.monitorVramRate'),
                  color: 'var(--primary)',
                },
                used: {
                  label: t('edges.monitorVram'),
                  color: 'color-mix(in oklch, var(--primary) 75%, var(--background))',
                },
              }}
            >
              <LineChart data={data} margin={{ left: 12, right: 12 }}>
                <CartesianGrid vertical={false} />
                <XAxis dataKey='time' tickFormatter={timeTick} tickLine={false} axisLine={false} />
                <YAxis
                  yAxisId='rate'
                  domain={[0, 100]}
                  tickFormatter={(v: number) => `${v}%`}
                  tickLine={false}
                  axisLine={false}
                />
                <YAxis
                  yAxisId='used'
                  orientation='right'
                  tickFormatter={(v: number) => formatBytes(v)}
                  tickLine={false}
                  axisLine={false}
                  width={60}
                />
                <ChartTooltip content={<ChartTooltipContent />} />
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
  return (
    <div className='flex min-w-0 flex-1 flex-col gap-4 rounded-xl border bg-card p-4 sm:gap-6 sm:p-6'>
      <div className='flex flex-wrap items-center gap-2 sm:gap-4'>
        <div className='flex flex-1 flex-col gap-1'>
          <p className='text-xs text-muted-foreground'>
            {t('edges.monitorIo')}
          </p>
        </div>
        <div className='hidden items-center gap-3 sm:flex sm:gap-5'>
          <div className='flex items-center gap-1.5 transition-opacity duration-200 motion-reduce:transition-none'>
            <div
              className='size-2.5 rounded-full sm:size-3'
              style={{ backgroundColor: 'var(--primary)' }}
            />
            <span className='text-[10px] text-muted-foreground sm:text-xs'>
              {t('edges.monitorIoRead')}
            </span>
          </div>
          <div className='flex items-center gap-1.5 transition-opacity duration-200 motion-reduce:transition-none'>
            <div
              className='size-2.5 rounded-full sm:size-3'
              style={{
                backgroundColor:
                  'color-mix(in oklch, var(--primary) 75%, var(--background))',
              }}
            />
            <span className='text-[10px] text-muted-foreground sm:text-xs'>
              {t('edges.monitorIoWrite')}
            </span>
          </div>
        </div>
      </div>
      <div className={CHART_HALF_HEIGHT}>
        <ChartContainer
          config={{
            read: {
              label: t('edges.monitorIoRead'),
              color: 'var(--primary)',
            },
            write: {
              label: t('edges.monitorIoWrite'),
              color:
                'color-mix(in oklch, var(--primary) 75%, var(--background))',
            },
          }}
          className='h-full w-full'
        >
          <AreaChart data={data} margin={{ left: 12, right: 12 }}>
            <defs>
              <linearGradient id='readGradient' x1='0' y1='0' x2='0' y2='1'>
                <stop
                  offset='0%'
                  stopColor='var(--color-read)'
                  stopOpacity={0.3}
                />
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
              tickFormatter={(v: number) => new Date(v).toLocaleTimeString()}
              tickLine={false}
              axisLine={false}
            />
            <YAxis
              tickFormatter={(v: number) => formatBytes(v)}
              tickLine={false}
              axisLine={false}
              width={60}
            />
            <ChartTooltip content={<ChartTooltipContent />} />
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
        </ChartContainer>
      </div>
    </div>
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
