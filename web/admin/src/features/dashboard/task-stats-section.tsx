import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { Bar, BarChart, CartesianGrid, XAxis, YAxis } from 'recharts'
import { Button } from '@/components/ui/button'
import { Calendar } from '@/components/ui/calendar'
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
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { queryKeys } from '@/lib/api/query-keys'
import {
  listTaskDailyStats,
  listTaskEdgeStats,
  listTaskErrorStats,
} from '@/lib/api/stats'
import { DAILY_PRESETS, daysAgo, formatDate } from './date-range'
import {
  pickDays,
  pickEdgeItems,
  pickErrorItems,
  pickSuccessRate,
} from './task-stats-parse'

function errorMessage(err: unknown): string | undefined {
  return err instanceof Error ? err.message : undefined
}

export function TaskStatsSection() {
  const { t } = useTranslation()
  const [range, setRange] = useState({ from: daysAgo(29), to: daysAgo(0) })

  const daily = useQuery({
    queryKey: queryKeys.stats.tasksDaily(range.from, range.to),
    queryFn: () => listTaskDailyStats(range),
  })
  const errors = useQuery({
    queryKey: queryKeys.stats.tasksErrors(range.from, range.to),
    queryFn: () => listTaskErrorStats({ ...range, limit: 5 }),
  })
  const edges = useQuery({
    queryKey: queryKeys.stats.tasksEdges(range.from, range.to),
    queryFn: () => listTaskEdgeStats(range),
  })

  const config: ChartConfig = {
    processed: {
      label: t('dashboard.processed'),
      color: 'var(--primary)',
    },
  }
  const days = pickDays(daily.data)
  const chartData = days.map((d) => ({
    date: d.date,
    processed: d.processed,
  }))
  const successRate = pickSuccessRate(daily.data)
  const errorItems = pickErrorItems(errors.data)
  const edgeItems = pickEdgeItems(edges.data)

  return (
    <div className='space-y-4'>
      <div className='flex flex-wrap items-center gap-2'>
        {DAILY_PRESETS.map((p) => (
          <Button
            key={p.labelKey}
            type='button'
            variant='outline'
            size='sm'
            onClick={() => setRange({ from: daysAgo(p.days - 1), to: daysAgo(0) })}
          >
            {t(p.labelKey)}
          </Button>
        ))}
        <Popover>
          <PopoverTrigger asChild>
            <Button type='button' variant='outline' size='sm'>
              {range.from} ~ {range.to}
            </Button>
          </PopoverTrigger>
          <PopoverContent align='end' className='w-auto p-0'>
            <Calendar
              mode='range'
              defaultMonth={new Date()}
              selected={{
                from: new Date(`${range.from}T00:00:00`),
                to: new Date(`${range.to}T00:00:00`),
              }}
              onSelect={(sel) => {
                if (sel?.from && sel?.to) {
                  setRange({ from: formatDate(sel.from), to: formatDate(sel.to) })
                }
              }}
            />
          </PopoverContent>
        </Popover>
      </div>

      <Card data-testid='task-daily-chart-card'>
        <CardHeader className='pb-2'>
          <CardTitle className='text-base font-semibold'>
            {t('dashboard.taskDailyTitle')}
          </CardTitle>
          <CardDescription>
            {t('dashboard.taskDailyDescription', {
              from: range.from,
              to: range.to,
            })}
          </CardDescription>
        </CardHeader>
        <CardContent>
          {daily.isLoading ? (
            <LoadingSkeleton rows={4} />
          ) : daily.isError ? (
            <ErrorBanner
              message={errorMessage(daily.error)}
              onRetry={() => void daily.refetch()}
            />
          ) : (
            <ChartContainer
              config={config}
              className='h-[200px] w-full min-w-0 sm:h-[240px] lg:h-[280px]'
            >
              <BarChart data={chartData} margin={{ top: 8, right: 4, bottom: 0, left: 4 }}>
                <CartesianGrid vertical={false} />
                <XAxis dataKey='date' tickLine={false} axisLine={false} tickMargin={4} />
                <YAxis tickLine={false} axisLine={false} allowDecimals={false} width={36} />
                <ChartTooltip content={<ChartTooltipContent />} />
                <Bar dataKey='processed' fill='var(--color-processed)' radius={4} />
              </BarChart>
            </ChartContainer>
          )}
        </CardContent>
      </Card>

      <div className='grid gap-4 sm:grid-cols-3'>
        <Card data-testid='task-stats-success-card'>
          <CardHeader className='pb-2'>
            <CardTitle className='text-sm font-medium'>
              {t('dashboard.statsSuccessRate')}
            </CardTitle>
          </CardHeader>
          <CardContent>
            {daily.isLoading ? (
              <LoadingSkeleton rows={1} />
            ) : daily.isError ? (
              <ErrorBanner
                message={errorMessage(daily.error)}
                onRetry={() => void daily.refetch()}
              />
            ) : (
              <div className='text-2xl font-bold tabular-nums'>
                {successRate == null
                  ? '—'
                  : `${(successRate * 100).toFixed(1)}%`}
              </div>
            )}
          </CardContent>
        </Card>

        <Card data-testid='task-stats-errors-card'>
          <CardHeader className='pb-2'>
            <CardTitle className='text-sm font-medium'>
              {t('dashboard.statsErrorTop')}
            </CardTitle>
          </CardHeader>
          <CardContent>
            {errors.isLoading ? (
              <LoadingSkeleton rows={3} />
            ) : errors.isError ? (
              <ErrorBanner
                message={errorMessage(errors.error)}
                onRetry={() => void errors.refetch()}
              />
            ) : errorItems.length ? (
              <ul className='space-y-1 text-sm'>
                {errorItems.map((e) => (
                  <li key={e.error_code} className='flex justify-between gap-2'>
                    <span className='truncate'>{e.error_code}</span>
                    <span className='tabular-nums'>{e.count}</span>
                  </li>
                ))}
              </ul>
            ) : (
              <p className='text-sm text-muted-foreground'>{t('common.empty')}</p>
            )}
          </CardContent>
        </Card>

        <Card data-testid='task-stats-edges-card'>
          <CardHeader className='pb-2'>
            <CardTitle className='text-sm font-medium'>
              {t('dashboard.statsEdgeLoad')}
            </CardTitle>
          </CardHeader>
          <CardContent>
            {edges.isLoading ? (
              <LoadingSkeleton rows={3} />
            ) : edges.isError ? (
              <ErrorBanner
                message={errorMessage(edges.error)}
                onRetry={() => void edges.refetch()}
              />
            ) : edgeItems.length ? (
              <ul className='space-y-1 text-sm'>
                {edgeItems.slice(0, 5).map((e) => (
                  <li key={e.edge_id} className='flex justify-between gap-2'>
                    <span className='truncate'>{e.edge_id}</span>
                    <span className='tabular-nums'>{e.count}</span>
                  </li>
                ))}
              </ul>
            ) : (
              <p className='text-sm text-muted-foreground'>{t('common.empty')}</p>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
