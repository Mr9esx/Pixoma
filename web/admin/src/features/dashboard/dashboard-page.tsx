import { useQuery } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { listCases } from '@/lib/api/cases'
import { listInstances } from '@/lib/api/instances'
import { queryKeys } from '@/lib/api/query-keys'
import { listTasks } from '@/lib/api/tasks'
import { aggregateDashboard } from '@/lib/dashboard/aggregate'
import { taskStatusLabelKey } from '@/features/tasks/list-panel'

const DASHBOARD_LIST_LIMIT = 200

function errorMessage(err: unknown): string | undefined {
  return err instanceof Error ? err.message : undefined
}

function RatioBar({
  segments,
}: {
  segments: { key: string; label: string; count: number; className: string }[]
}) {
  const total = segments.reduce((sum, s) => sum + s.count, 0)
  if (total === 0) {
    return (
      <div className='bg-muted h-2 w-full overflow-hidden rounded-full' />
    )
  }
  return (
    <div className='space-y-2'>
      <div className='bg-muted flex h-2 w-full overflow-hidden rounded-full'>
        {segments.map((s) => {
          const pct = (s.count / total) * 100
          if (pct <= 0) return null
          return (
            <div
              key={s.key}
              className={s.className}
              style={{ width: `${pct}%` }}
              title={`${s.label}: ${s.count}`}
            />
          )
        })}
      </div>
      <ul className='text-muted-foreground space-y-1 text-xs'>
        {segments.map((s) => (
          <li key={s.key} className='flex items-center justify-between gap-2'>
            <span className='flex items-center gap-2'>
              <span className={`inline-block size-2 rounded-sm ${s.className}`} />
              {s.label}
            </span>
            <span>{s.count}</span>
          </li>
        ))}
      </ul>
    </div>
  )
}

function InstancesCard() {
  const { t } = useTranslation()
  const q = useQuery({
    queryKey: queryKeys.instances.all,
    queryFn: listInstances,
  })

  if (q.isError) {
    return (
      <Card data-testid='dashboard-instances-card'>
        <CardHeader>
          <CardTitle className='text-sm font-medium'>
            {t('dashboard.instancesTitle')}
          </CardTitle>
        </CardHeader>
        <CardContent>
          <ErrorBanner
            message={errorMessage(q.error)}
            onRetry={() => void q.refetch()}
          />
        </CardContent>
      </Card>
    )
  }

  if (q.isLoading) {
    return (
      <Card data-testid='dashboard-instances-card'>
        <CardHeader>
          <CardTitle className='text-sm font-medium'>
            {t('dashboard.instancesTitle')}
          </CardTitle>
        </CardHeader>
        <CardContent>
          <LoadingSkeleton rows={3} />
        </CardContent>
      </Card>
    )
  }

  const stats = aggregateDashboard({
    instances: q.data ?? [],
    cases: [],
    tasks: [],
  })
  const disabled = stats.instanceTotal - stats.instanceEnabled

  return (
    <Card data-testid='dashboard-instances-card'>
      <CardHeader className='pb-2'>
        <CardTitle className='text-sm font-medium'>
          <Link to='/instances' className='hover:underline'>
            {t('dashboard.instancesTitle')}
          </Link>
        </CardTitle>
        <CardDescription>
          {t('dashboard.instancesSummary', {
            total: stats.instanceTotal,
            enabled: stats.instanceEnabled,
          })}
        </CardDescription>
      </CardHeader>
      <CardContent className='space-y-4'>
        <div className='text-2xl font-bold'>
          {stats.instanceEnabled}
          <span className='text-muted-foreground text-base font-normal'>
            {' '}
            / {stats.instanceTotal}
          </span>
        </div>
        <div>
          <p className='mb-2 text-xs font-medium'>
            {t('dashboard.instanceEnabledRatio')}
          </p>
          <RatioBar
            segments={[
              {
                key: 'enabled',
                label: t('dashboard.enabled'),
                count: stats.instanceEnabled,
                className: 'bg-emerald-500',
              },
              {
                key: 'disabled',
                label: t('dashboard.disabled'),
                count: disabled,
                className: 'bg-zinc-400',
              },
            ]}
          />
        </div>
      </CardContent>
    </Card>
  )
}

function CasesCard() {
  const { t } = useTranslation()
  const q = useQuery({
    queryKey: [...queryKeys.cases.all, { limit: DASHBOARD_LIST_LIMIT }] as const,
    queryFn: () => listCases({ limit: DASHBOARD_LIST_LIMIT }),
  })

  if (q.isError) {
    return (
      <Card data-testid='dashboard-cases-card'>
        <CardHeader>
          <CardTitle className='text-sm font-medium'>
            {t('dashboard.casesTitle')}
          </CardTitle>
        </CardHeader>
        <CardContent>
          <ErrorBanner
            message={errorMessage(q.error)}
            onRetry={() => void q.refetch()}
          />
        </CardContent>
      </Card>
    )
  }

  if (q.isLoading) {
    return (
      <Card data-testid='dashboard-cases-card'>
        <CardHeader>
          <CardTitle className='text-sm font-medium'>
            {t('dashboard.casesTitle')}
          </CardTitle>
        </CardHeader>
        <CardContent>
          <LoadingSkeleton rows={2} />
        </CardContent>
      </Card>
    )
  }

  const stats = aggregateDashboard({
    instances: [],
    cases: q.data ?? [],
    tasks: [],
  })

  return (
    <Card data-testid='dashboard-cases-card'>
      <CardHeader className='pb-2'>
        <CardTitle className='text-sm font-medium'>
          <Link to='/cases' className='hover:underline'>
            {t('dashboard.casesTitle')}
          </Link>
        </CardTitle>
        <CardDescription>{t('dashboard.casesEnabled')}</CardDescription>
      </CardHeader>
      <CardContent>
        <div className='text-2xl font-bold'>{stats.caseEnabled}</div>
      </CardContent>
    </Card>
  )
}

const TASK_STATUS_COLORS = [
  'bg-sky-500',
  'bg-amber-500',
  'bg-emerald-500',
  'bg-rose-500',
  'bg-violet-500',
  'bg-zinc-400',
]

function TasksCard() {
  const { t } = useTranslation()
  const q = useQuery({
    queryKey: [...queryKeys.tasks.all, { limit: DASHBOARD_LIST_LIMIT }] as const,
    queryFn: () => listTasks({ limit: DASHBOARD_LIST_LIMIT }),
  })

  if (q.isError) {
    return (
      <Card data-testid='dashboard-tasks-card'>
        <CardHeader>
          <CardTitle className='text-sm font-medium'>
            {t('dashboard.tasksTitle')}
          </CardTitle>
        </CardHeader>
        <CardContent>
          <ErrorBanner
            message={errorMessage(q.error)}
            onRetry={() => void q.refetch()}
          />
        </CardContent>
      </Card>
    )
  }

  if (q.isLoading) {
    return (
      <Card data-testid='dashboard-tasks-card'>
        <CardHeader>
          <CardTitle className='text-sm font-medium'>
            {t('dashboard.tasksTitle')}
          </CardTitle>
        </CardHeader>
        <CardContent>
          <LoadingSkeleton rows={4} />
        </CardContent>
      </Card>
    )
  }

  const stats = aggregateDashboard({
    instances: [],
    cases: [],
    tasks: q.data ?? [],
  })
  const entries = Object.entries(stats.taskByStatus).sort(([a], [b]) =>
    a.localeCompare(b),
  )

  return (
    <Card data-testid='dashboard-tasks-card' className='sm:col-span-2 lg:col-span-1'>
      <CardHeader className='pb-2'>
        <CardTitle className='text-sm font-medium'>
          <Link to='/tasks' className='hover:underline'>
            {t('dashboard.tasksTitle')}
          </Link>
        </CardTitle>
        <CardDescription>{t('dashboard.taskStatusDistribution')}</CardDescription>
      </CardHeader>
      <CardContent>
        {entries.length === 0 ? (
          <p className='text-muted-foreground text-sm'>{t('common.empty')}</p>
        ) : (
          <RatioBar
            segments={entries.map(([status, count], i) => {
              const statusKey = taskStatusLabelKey(status)
              return {
                key: status,
                label: statusKey ? t(statusKey) : status,
                count,
                className: TASK_STATUS_COLORS[i % TASK_STATUS_COLORS.length]!,
              }
            })}
          />
        )}
      </CardContent>
    </Card>
  )
}

export function DashboardPage() {
  const { t } = useTranslation()

  return (
    <div data-testid='dashboard-page' className='space-y-4'>
      <div>
        <h1 className='text-2xl font-bold tracking-tight'>
          {t('dashboard.title')}
        </h1>
        <p className='text-muted-foreground text-sm'>{t('common.sampleNote')}</p>
      </div>
      <div className='grid gap-4 sm:grid-cols-2 lg:grid-cols-3'>
        <InstancesCard />
        <CasesCard />
        <TasksCard />
      </div>
    </div>
  )
}
