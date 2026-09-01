import { Skeleton } from '@/components/ui/skeleton'

export function ChartSkeleton() {
  return (
    <div className='h-full min-h-[180px] w-full space-y-3' role='status'>
      <Skeleton className='h-3 w-24' />
      <Skeleton className='h-3 w-40' />
      <Skeleton className='h-[120px] w-full flex-1' />
    </div>
  )
}
