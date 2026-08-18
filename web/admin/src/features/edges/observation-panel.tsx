import type { ReactNode } from 'react'
import { Link } from '@tanstack/react-router'
import {
  Area,
  AreaChart,
  CartesianGrid,
  XAxis,
  YAxis,
} from 'recharts'
import { useTranslation } from 'react-i18next'
import type { TaskRecord } from '@/lib/api/types'
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
import type { MetricsPoint } from './observation'

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
  tasksQuery: QueryView<TaskRecord[] | undefined>
}

export function ObservationPanel({ tasksQuery }: Props) {
  return (
    <div className='flex flex-col gap-7' data-testid='edge-observation'>
      <BlockBody query={tasksQuery}>
        <TasksSection tasks={tasksQuery.data ?? []} />
      </BlockBody>
    </div>
  )
}
