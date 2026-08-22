import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { CartesianGrid, Scatter, ScatterChart, XAxis, YAxis, ZAxis } from 'recharts'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from '@/components/ui/chart'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { queryKeys } from '@/lib/api/query-keys'
import { listTaskCaseTopStats } from '@/lib/api/stats'
import { pickCaseItems } from './task-stats-parse'
import type { StatsRange } from './task-range-picker'

function errorMessage(err: unknown): string | undefined {
  return err instanceof Error ? err.message : undefined
}

export function CaseAnalysisSection({ range }: { range: StatsRange }) {
  const { t } = useTranslation()
  const cases = useQuery({
    queryKey: queryKeys.stats.casesTop(range.from, range.to),
    queryFn: () => listTaskCaseTopStats({ ...range, limit: 10 }),
  })
  const items = pickCaseItems(cases.data)
  const scatterData = items.map((c) => ({
    case_id: c.case_id,
    count: c.count,
    seconds: c.avg_duration_ms == null ? null : c.avg_duration_ms / 1000,
  }))
  const maxCount = Math.max(1, ...items.map((c) => c.count))
  const config: ChartConfig = {
    cases: { label: t('dashboard.caseScatterTitle'), color: 'var(--primary)' },
  }

  return (
    <div className='grid gap-4 lg:grid-cols-2'>
      <Card data-testid='case-scatter-card'>
        <CardHeader className='pb-2'>
          <CardTitle className='text-base font-semibold'>
            {t('dashboard.caseScatterTitle')}
          </CardTitle>
          <CardDescription>{t('dashboard.caseScatterHint')}</CardDescription>
        </CardHeader>
        <CardContent>
          {cases.isLoading ? (
            <LoadingSkeleton rows={4} />
          ) : cases.isError ? (
            <ErrorBanner message={errorMessage(cases.error)} onRetry={() => void cases.refetch()} />
          ) : (
            <ChartContainer config={config} className='h-[240px] w-full min-w-0'>
              <ScatterChart margin={{ top: 8, right: 12, bottom: 0, left: 4 }}>
                <CartesianGrid />
                <XAxis
                  type='number'
                  dataKey='count'
                  name={t('dashboard.caseCount')}
                  tickLine={false}
                  axisLine={false}
                  width={48}
                  allowDecimals={false}
                />
                <YAxis
                  type='number'
                  dataKey='seconds'
                  name={t('dashboard.caseAvgDuration')}
                  tickLine={false}
                  axisLine={false}
                  width={48}
                  unit='s'
                />
                <ZAxis type='number' dataKey='count' range={[60, 260]} />
                <ChartTooltip content={<ChartTooltipContent />} cursor={{ strokeDasharray: '3 3' }} />
                <Scatter data={scatterData} fill='var(--color-cases)' />
              </ScatterChart>
            </ChartContainer>
          )}
        </CardContent>
      </Card>

      <Card data-testid='case-top-card'>
        <CardHeader className='pb-2'>
          <CardTitle className='text-base font-semibold'>
            {t('dashboard.caseTopTitle')}
          </CardTitle>
          <CardDescription>{t('dashboard.caseTopHint')}</CardDescription>
        </CardHeader>
        <CardContent>
          {cases.isLoading ? (
            <LoadingSkeleton rows={5} />
          ) : cases.isError ? (
            <ErrorBanner message={errorMessage(cases.error)} onRetry={() => void cases.refetch()} />
          ) : items.length ? (
            <ul className='space-y-2 text-sm'>
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
  )
}
