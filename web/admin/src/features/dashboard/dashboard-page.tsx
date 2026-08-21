import { useQuery } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { listCases } from '@/lib/api/cases'
import { listEdges } from '@/lib/api/edges'
import { queryKeys } from '@/lib/api/query-keys'
import { aggregateDashboard } from '@/lib/dashboard/aggregate'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { TaskStatsSection } from './task-stats-section'

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
    return <div className='h-2 w-full overflow-hidden rounded-full bg-muted' />
  }
  return (
    <div className='space-y-2'>
      <div className='flex h-2 w-full overflow-hidden rounded-full bg-muted'>
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
      <ul className='space-y-1 text-xs text-muted-foreground'>
        {segments.map((s) => (
          <li key={s.key} className='flex items-center justify-between gap-2'>
            <span className='flex items-center gap-2'>
              <span
                className={`inline-block size-2 rounded-sm ${s.className}`}
              />
              {s.label}
            </span>
            <span>{s.count}</span>
          </li>
        ))}
      </ul>
    </div>
  )
}

function EdgesCard() {
  const { t } = useTranslation()
  const q = useQuery({
    queryKey: queryKeys.edges.all,
    queryFn: listEdges,
  })

  if (q.isError) {
    return (
      <Card data-testid='dashboard-edges-card'>
        <CardHeader>
          <CardTitle className='text-sm font-medium'>
            {t('dashboard.edgesTitle')}
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
      <Card data-testid='dashboard-edges-card'>
        <CardHeader>
          <CardTitle className='text-sm font-medium'>
            {t('dashboard.edgesTitle')}
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
    <Card data-testid='dashboard-edges-card'>
      <CardHeader className='pb-2'>
        <CardTitle className='text-sm font-medium'>
          <Link to='/edges' className='hover:underline'>
            {t('dashboard.edgesTitle')}
          </Link>
        </CardTitle>
        <CardDescription>
          {t('dashboard.edgesSummary', {
            total: stats.instanceTotal,
            enabled: stats.instanceEnabled,
          })}
        </CardDescription>
      </CardHeader>
      <CardContent className='space-y-4'>
        <div className='text-2xl font-bold'>
          {stats.instanceEnabled}
          <span className='text-base font-normal text-muted-foreground'>
            {' '}
            / {stats.instanceTotal}
          </span>
        </div>
        <div>
          <p className='mb-2 text-xs font-medium'>
            {t('dashboard.edgeEnabledRatio')}
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
    queryKey: [
      ...queryKeys.cases.all,
      { limit: DASHBOARD_LIST_LIMIT },
    ] as const,
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

export function DashboardPage() {
  const { t } = useTranslation()

  return (
    <div
      data-testid='dashboard-page'
      className='min-h-0 flex-1 space-y-4 overflow-auto'
    >
      <div>
        <h1 className='text-2xl font-bold tracking-tight'>
          {t('dashboard.title')}
        </h1>
        <p className='text-sm text-muted-foreground'>
          {t('common.sampleNote')}
        </p>
      </div>
      <div className='grid gap-4 sm:grid-cols-2 lg:grid-cols-3'>
        <EdgesCard />
        <CasesCard />
        <div className='sm:col-span-2 lg:col-span-3'>
          <TaskStatsSection />
        </div>
      </div>
    </div>
  )
}
