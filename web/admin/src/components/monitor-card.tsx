import type { ReactNode } from 'react'
import { ChartContainer, type ChartConfig } from '@/components/ui/chart'
import { cn } from '@/lib/utils'

export type MonitorStat = { label: string; value: string }

export function MonitorCard({
  title,
  config,
  stats,
  compact = false,
  className,
  "data-testid": dataTestId,
  children,
  plain = false,
}: {
  title: string
  config: ChartConfig
  stats: MonitorStat[]
  compact?: boolean
  className?: string
  "data-testid"?: string
  plain?: boolean
  children: ReactNode
}) {
  return (
    <div
      className={cn(
        'flex min-w-0 flex-1 flex-col rounded-xl border bg-card p-4 text-card-foreground shadow-none',
        className
      )}
      data-testid={dataTestId}
    >
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
        className={cn(
          'grid min-h-0 flex-1 gap-1.5',
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
            <ChartContainer config={config} className='aspect-auto h-full w-full'>
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
                <p className='text-[11px] text-muted-foreground'>{stat.label}</p>
              </div>
            ))}
          </div>
        ) : null}
      </div>
    </div>
  )
}
