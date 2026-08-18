import type { ReactNode } from 'react'
import { Link } from '@tanstack/react-router'
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
import { useTranslation } from 'react-i18next'
import type { TaskRecord } from '@/lib/api/types'
import type { EdgeMetricsResponse } from '@/lib/api/types'
import { EmptyState } from '@/components/feedback/empty-state'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
} from '@/components/ui/chart'
import { taskStatusLabelKey } from '@/features/tasks/list-panel'
import { kit } from './kit-classes'
import { parseMetrics, type MetricsPoint } from './observation'
import { formatBytes } from './observation'

export function CpuCard({ series }: { series: MetricsPoint[] }) {
  const { t } = useTranslation()
  const latest = series[series.length - 1]
  const data = series.map((p) => ({ time: p.time, cpu: p.cpu }))
  return (
    <div className='bg-card flex min-w-0 flex-1 flex-col gap-4 rounded-xl border p-4 sm:gap-6 sm:p-6'>
      <div className='flex flex-wrap items-center gap-2 sm:gap-4'>
        <div className='flex flex-1 flex-col gap-1'>
          <p className='text-xl leading-tight font-semibold tracking-tight sm:text-2xl'>
            {latest?.cpu != null ? `${latest.cpu.toFixed(1)}%` : '—'}
          </p>
          <p className='text-muted-foreground text-xs'>{t('edges.monitorCpu')}</p>
        </div>
        <div className='hidden items-center gap-3 sm:flex sm:gap-5'>
          <div className='flex items-center gap-1.5 transition-opacity duration-200 motion-reduce:transition-none'>
            <div
              className='size-2.5 rounded-full sm:size-3'
              style={{ backgroundColor: 'var(--primary)' }}
            />
            <span className='text-muted-foreground text-[10px] sm:text-xs'>
              {t('edges.monitorCpu')}
            </span>
          </div>
        </div>
      </div>
      <div className='h-[200px] w-full min-w-0 sm:h-[240px] lg:h-[280px]'>
        <ChartContainer
          config={{
            cpu: { label: t('edges.monitorCpu'), color: 'var(--primary)' },
          }}
          className='h-full w-full'
        >
          <AreaChart data={data} margin={{ left: 12, right: 12 }}>
            <defs>
              <linearGradient id='cpuGradient' x1='0' y1='0' x2='0' y2='1'>
                <stop offset='0%' stopColor='var(--color-cpu)' stopOpacity={0.3} />
                <stop offset='100%' stopColor='var(--color-cpu)' stopOpacity={0.05} />
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
              domain={[0, 100]}
              tickFormatter={(v: number) => `${v}%`}
              tickLine={false}
              axisLine={false}
            />
            <ChartTooltip content={<ChartTooltipContent />} />
            <Area
              dataKey='cpu'
              type='natural'
              fill='url(#cpuGradient)'
              stroke='var(--color-cpu)'
              strokeWidth={2}
            />
          </AreaChart>
        </ChartContainer>
      </div>
    </div>
  )
}

export function MemCard({ point }: { point: MetricsPoint | null }) {
  const { t } = useTranslation()
  const pct = point?.memPct ?? 0
  const used = point?.memUsed ?? 0
  const total = point?.memTotal ?? 0
  const free = Math.max(total - used, 0)
  const data = [
    {
      name: t('edges.monitorMemUsed'),
      value: used,
      fill: 'var(--color-used)',
    },
    {
      name: t('edges.monitorMemFree'),
      value: free,
      fill: 'var(--color-free)',
    },
  ]
  return (
    <div className='bg-card flex flex-1 flex-col gap-4 rounded-xl border p-4 sm:p-5'>
      <div className='flex items-center justify-between'>
        <div className='flex items-center gap-2 sm:gap-2.5'>
          <div>
            <span className='text-sm font-medium sm:text-base'>
              {t('edges.monitorMem')}
            </span>
            <p className='text-muted-foreground text-[10px] sm:text-xs'>
              {total > 0 ? formatBytes(total) : '—'}
            </p>
          </div>
        </div>
      </div>
      <div className='flex flex-1 items-center gap-4 sm:gap-6'>
        <div className='relative size-[100px] shrink-0 sm:size-[120px]'>
          <ChartContainer
            config={{
              used: {
                label: t('edges.monitorMemUsed'),
                color: 'var(--primary)',
              },
              free: {
                label: t('edges.monitorMemFree'),
                color: 'color-mix(in oklch, var(--primary) 75%, var(--background))',
              },
            }}
            className='h-full w-full'
          >
            <PieChart>
              <ChartTooltip content={<ChartTooltipContent hideLabel />} />
              <Pie
                data={data}
                dataKey='value'
                nameKey='name'
                innerRadius={30}
                outerRadius={49.5}
                strokeWidth={0}
              >
                {data.map((entry) => (
                  <Cell key={entry.name} fill={entry.fill} />
                ))}
              </Pie>
            </PieChart>
          </ChartContainer>
          <div className='pointer-events-none absolute inset-0 flex flex-col items-center justify-center'>
            <span className='text-sm font-semibold sm:text-base'>
              {pct.toFixed(1)}%
            </span>
            <span className='text-muted-foreground text-[8px] sm:text-[10px]'>
              {t('edges.monitorMemUsed')}
            </span>
          </div>
        </div>
        <div className='flex flex-1 flex-col gap-2 sm:gap-3'>
          {data.map((entry) => (
            <div
              key={entry.name}
              className='flex items-center justify-between gap-2'
            >
              <div className='flex items-center gap-2'>
                <div
                  className='size-2 rounded-full sm:size-2.5'
                  style={{ backgroundColor: entry.fill }}
                />
                <span className='text-muted-foreground text-[10px] sm:text-xs'>
                  {entry.name}
                </span>
              </div>
              <div className='flex items-center gap-2 text-[10px] sm:text-xs'>
                <span className='font-medium tabular-nums'>
                  {formatBytes(entry.value)}
                </span>
                <span className='text-muted-foreground tabular-nums'>
                  {total > 0
                    ? `${((entry.value / total) * 100).toFixed(1)}%`
                    : '—'}
                </span>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}

function GpuDonut({
  center,
  label,
  series,
}: {
  center: string
  label: string
  series: { name: string; value: number; fill: string }[]
}) {
  return (
    <div className='relative size-[100px] shrink-0 sm:size-[120px]'>
      <ChartContainer
        config={{
          a: { label, color: 'var(--primary)' },
          b: {
            label,
            color: 'color-mix(in oklch, var(--primary) 75%, var(--background))',
          },
        }}
        className='h-full w-full'
      >
        <PieChart>
          <Pie
            data={series}
            dataKey='value'
            nameKey='name'
            innerRadius={30}
            outerRadius={49.5}
            strokeWidth={0}
          >
            {series.map((entry) => (
              <Cell key={entry.name} fill={entry.fill} />
            ))}
          </Pie>
        </PieChart>
      </ChartContainer>
      <div className='pointer-events-none absolute inset-0 flex flex-col items-center justify-center'>
        <span className='text-sm font-semibold sm:text-base'>{center}</span>
        <span className='text-muted-foreground text-[8px] sm:text-[10px]'>
          {label}
        </span>
      </div>
    </div>
  )
}

export function GpuCards({ point }: { point: MetricsPoint | null }) {
  const { t } = useTranslation()
  const gpus = point?.gpus ?? []
  if (gpus.length === 0) return null
  return (
    <>
      {gpus.map((gpu, index) => {
        const usage = gpu.usage_percent ?? 0
        const used = gpu.vram_used_bytes ?? 0
        const total = gpu.vram_total_bytes ?? 0
        const free = Math.max(total - used, 0)
        return (
          <div
            key={`${gpu.name}-${index}`}
            className='bg-card flex flex-1 flex-col gap-4 rounded-xl border p-4 sm:p-5'
          >
            <span className='text-sm font-medium sm:text-base'>
              {gpu.name || t('edges.fieldGpu')}
            </span>
            <div className='flex flex-1 flex-wrap items-center gap-4 sm:gap-6'>
              <GpuDonut
                center={`${usage.toFixed(1)}%`}
                label={t('edges.monitorGpuUsage')}
                series={[
                  {
                    name: t('edges.monitorGpuUsage'),
                    value: usage,
                    fill: 'var(--primary)',
                  },
                  {
                    name: t('edges.monitorGpuIdle'),
                    value: Math.max(100 - usage, 0),
                    fill: 'color-mix(in oklch, var(--primary) 75%, var(--background))',
                  },
                ]}
              />
              <GpuDonut
                center={total > 0 ? formatBytes(used) : '—'}
                label={t('edges.monitorVram')}
                series={[
                  {
                    name: t('edges.monitorMemUsed'),
                    value: used,
                    fill: 'var(--primary)',
                  },
                  {
                    name: t('edges.monitorMemFree'),
                    value: free,
                    fill: 'color-mix(in oklch, var(--primary) 75%, var(--background))',
                  },
                ]}
              />
              <div className='flex min-w-32 flex-col gap-2 sm:gap-3'>
                <div className='flex items-center justify-between gap-2'>
                  <span className='text-muted-foreground text-[10px] sm:text-xs'>
                    {t('edges.monitorGpuUsage')}
                  </span>
                  <span className='font-medium tabular-nums text-[10px] sm:text-xs'>
                    {usage.toFixed(1)}%
                  </span>
                </div>
                <div className='flex items-center justify-between gap-2'>
                  <span className='text-muted-foreground text-[10px] sm:text-xs'>
                    {t('edges.monitorVram')}
                  </span>
                  <span className='font-medium tabular-nums text-[10px] sm:text-xs'>
                    {total > 0
                      ? `${formatBytes(used)} / ${formatBytes(total)}`
                      : '—'}
                  </span>
                </div>
              </div>
            </div>
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
    <div className='bg-card flex min-w-0 flex-1 flex-col gap-4 rounded-xl border p-4 sm:gap-6 sm:p-6'>
      <div className='flex flex-wrap items-center gap-2 sm:gap-4'>
        <div className='flex flex-1 flex-col gap-1'>
          <p className='text-muted-foreground text-xs'>{t('edges.monitorIo')}</p>
        </div>
        <div className='hidden items-center gap-3 sm:flex sm:gap-5'>
          <div className='flex items-center gap-1.5 transition-opacity duration-200 motion-reduce:transition-none'>
            <div
              className='size-2.5 rounded-full sm:size-3'
              style={{ backgroundColor: 'var(--primary)' }}
            />
            <span className='text-muted-foreground text-[10px] sm:text-xs'>
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
            <span className='text-muted-foreground text-[10px] sm:text-xs'>
              {t('edges.monitorIoWrite')}
            </span>
          </div>
        </div>
      </div>
      <div className='h-[200px] w-full min-w-0 sm:h-[240px] lg:h-[280px]'>
        <ChartContainer
          config={{
            read: {
              label: t('edges.monitorIoRead'),
              color: 'var(--primary)',
            },
            write: {
              label: t('edges.monitorIoWrite'),
              color: 'color-mix(in oklch, var(--primary) 75%, var(--background))',
            },
          }}
          className='h-full w-full'
        >
          <AreaChart data={data} margin={{ left: 12, right: 12 }}>
            <defs>
              <linearGradient id='readGradient' x1='0' y1='0' x2='0' y2='1'>
                <stop offset='0%' stopColor='var(--color-read)' stopOpacity={0.3} />
                <stop offset='100%' stopColor='var(--color-read)' stopOpacity={0.05} />
              </linearGradient>
              <linearGradient id='writeGradient' x1='0' y1='0' x2='0' y2='1'>
                <stop offset='0%' stopColor='var(--color-write)' stopOpacity={0.2} />
                <stop offset='100%' stopColor='var(--color-write)' stopOpacity={0.02} />
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

function TasksSection({ tasks }: { tasks: TaskRecord[] }) {
  const { t } = useTranslation()
  return (
    <section className='flex flex-col gap-4'>
      <SectionHead
        title={t('edges.observationTasks')}
        hint={t('edges.observationTasksHint')}
      />
      {tasks.length === 0 ? (
        <EmptyState className='py-6' message={t('tasks.empty')} />
      ) : (
        <div className={kit.tableWrap}>
          <table className='w-full caption-bottom text-sm'>
            <thead>
              <tr className='border-b bg-muted/25'>
                <th className={`${kit.th} px-4`}>{t('tasks.fieldStatus')}</th>
                <th className={kit.th}>{t('tasks.fieldCaseId')}</th>
                <th className={kit.th}>{t('tasks.fieldId')}</th>
                <th className={kit.th}>{t('tasks.fieldUpdatedAt')}</th>
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
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      )}
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
        <EmptyState className='py-6' message={t('edges.monitorEmpty')} />
      ) : (
        <>
          <div className='flex flex-col gap-4 xl:flex-row'>
            <CpuCard series={data.series} />
            <div className='flex w-full flex-col gap-4 xl:w-[410px]'>
              <MemCard point={data.latest} />
            </div>
          </div>
          <GpuCards point={data.latest} />
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
}

export function ObservationPanel({ metricsQuery, tasksQuery }: Props) {
  return (
    <div className='flex flex-col gap-7' data-testid='edge-observation'>
      <BlockBody query={metricsQuery}>
        <MonitoringSection data={parseMetrics(metricsQuery.data)} />
      </BlockBody>
      <BlockBody query={tasksQuery}>
        <TasksSection tasks={tasksQuery.data ?? []} />
      </BlockBody>
    </div>
  )
}
