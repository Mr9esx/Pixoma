import type { ReactNode } from 'react'
import { cn } from '@/lib/utils'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { ChartContainer, type ChartConfig } from '@/components/ui/chart'
import { Skeleton } from '@/components/ui/skeleton'

export type MonitorStat = { label: string; value: string }

export function MonitorCard({
  title,
  config,
  stats,
  compact = false,
  className,
  'data-testid': dataTestId,
  children,
  plain = false,
}: {
  title: string
  config: ChartConfig
  stats: MonitorStat[]
  compact?: boolean
  className?: string
  'data-testid'?: string
  plain?: boolean
  children: ReactNode
}) {
  return (
    <Card
      className={cn('min-w-0 flex-1 gap-3 py-4', className)}
      data-testid={dataTestId}
    >
      <CardHeader className='gap-3 px-4'>
        <CardTitle className='text-base font-semibold'>{title}</CardTitle>
        {Object.keys(config).length > 0 ? (
          <CardDescription className='flex flex-wrap items-center gap-4 text-xs text-muted-foreground'>
            {Object.entries(config).map(([key, entry]) => (
              <span key={key} className='inline-flex items-center gap-1.5'>
                <span
                  className='size-2 rounded-full'
                  style={{ backgroundColor: entry.color }}
                />
                {entry.label}
              </span>
            ))}
          </CardDescription>
        ) : null}
      </CardHeader>
      <CardContent
        className={cn(
          'grid min-h-0 flex-1 gap-1.5 px-4',
          stats.length === 0
            ? 'grid-cols-1'
            : compact
              ? 'grid-cols-1 lg:grid-cols-[minmax(0,1fr)_122px]'
              : 'grid-cols-1 lg:grid-cols-[minmax(0,1fr)_72px]'
        )}
      >
        <div className='h-full min-h-[120px] w-full min-w-0'>
          {plain ? (
            <div className='h-full w-full'>{children}</div>
          ) : (
            <ChartContainer
              config={config}
              className='aspect-auto h-full w-full'
            >
              {children}
            </ChartContainer>
          )}
        </div>
        {stats.length > 0 ? (
          <div
            className={cn(
              'grid content-center gap-2 text-center',
              compact
                ? 'grid-cols-2 gap-1.5 lg:text-right'
                : 'grid-cols-3 lg:grid-cols-1 lg:text-right'
            )}
          >
            {stats.map((stat) => (
              <div key={stat.label}>
                <p
                  className={cn(
                    'font-semibold',
                    compact ? 'text-xs leading-5' : 'text-lg leading-6'
                  )}
                >
                  {stat.value}
                </p>
                <p className='text-xs text-muted-foreground'>{stat.label}</p>
              </div>
            ))}
          </div>
        ) : null}
      </CardContent>
    </Card>
  )
}

export function MonitorCardSkeleton({
  compact = false,
  statsCells = compact ? 4 : 3,
  className,
}: {
  compact?: boolean
  statsCells?: number
  className?: string
}) {
  return (
    <Card className={cn('min-w-0 flex-1 gap-3 py-4', className)}>
      <CardHeader className='gap-3 px-4'>
        <Skeleton className='h-4 w-28' />
        <Skeleton className='h-3 w-44' />
      </CardHeader>
      <CardContent
        className={cn(
          'grid min-h-0 flex-1 gap-1.5 px-4',
          compact
            ? 'lg:grid-cols-[minmax(0,1fr)_122px]'
            : 'lg:grid-cols-[minmax(0,1fr)_72px]'
        )}
      >
        <div className='h-full min-h-[150px] w-full min-w-0'>
          <Skeleton className='h-full w-full' />
        </div>
        <div
          className={cn(
            'grid content-center gap-2',
            compact
              ? 'grid-cols-2 gap-1.5'
              : 'grid-cols-3 lg:grid-cols-1 lg:text-right'
          )}
        >
          {Array.from({ length: statsCells }, (_, index) => (
            <div
              key={index}
              className={cn(
                'flex flex-col items-center gap-1',
                !compact && 'lg:items-end'
              )}
            >
              <Skeleton className='h-4 w-12' />
              <Skeleton className='h-2.5 w-10' />
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  )
}
