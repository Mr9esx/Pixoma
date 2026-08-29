import type { ReactNode } from 'react'
import { cn } from '@/lib/utils'

export function MenuMapLayout({
  toolbar,
  keyboard,
  path,
  orphans,
  testId,
  className,
}: {
  toolbar?: ReactNode
  keyboard: ReactNode
  path: ReactNode
  orphans?: ReactNode
  testId?: string
  className?: string
}) {
  return (
    <div
      data-testid={testId}
      className={cn(
        'flex min-h-0 flex-1 flex-col overflow-hidden rounded-[8px] border border-border bg-card',
        className
      )}
    >
      {toolbar ? (
        <div className='flex shrink-0 items-center justify-between gap-3 border-b border-border px-4 py-3'>
          {toolbar}
        </div>
      ) : null}
      <div className='grid min-h-[420px] flex-1 grid-cols-1 min-[920px]:grid-cols-2'>
        <div className='min-h-0 min-w-0 overflow-y-auto border-border p-4 min-[920px]:border-r'>
          {keyboard}
        </div>
        <div
          className='min-h-0 min-w-0 overflow-y-auto p-4'
          data-testid='map-path'
        >
          {path}
        </div>
      </div>
      {orphans}
    </div>
  )
}
