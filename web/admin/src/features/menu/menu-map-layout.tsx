import type { ReactNode } from 'react'
import { cn } from '@/lib/utils'
import { Card, CardContent, CardHeader } from '@/components/ui/card'

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
    <Card
      data-testid={testId}
      className={cn(
        'min-h-0 flex-1 flex-col gap-0 overflow-hidden rounded-md py-0',
        className
      )}
    >
      {toolbar ? (
        <CardHeader className='flex shrink-0 flex-row items-center justify-between gap-3 border-b border-border px-4 py-3'>
          {toolbar}
        </CardHeader>
      ) : null}
      <CardContent className='grid min-h-[420px] flex-1 grid-cols-1 p-0 min-[920px]:grid-cols-2'>
        <div className='min-h-0 min-w-0 overflow-y-auto border-border p-4 min-[920px]:border-r'>
          {keyboard}
        </div>
        <div
          className='min-h-0 min-w-0 overflow-y-auto p-4'
          data-testid='map-path'
        >
          {path}
        </div>
      </CardContent>
      {orphans}
    </Card>
  )
}
