import { Skeleton } from '@/components/ui/skeleton'

export function ChartSkeleton() {
  return (
    <div
      className='flex h-full min-h-[180px] w-full flex-col gap-3'
      role='status'
    >
      <Skeleton className='h-3 w-24' />
      <Skeleton className='h-3 w-40' />
      <Skeleton className='h-[120px] w-full flex-1' />
    </div>
  )
}
