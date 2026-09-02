import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'
import { Skeleton } from '@/components/ui/skeleton'

type Props = {
  rows?: number
  className?: string
}

export function LoadingSkeleton({ rows = 4, className }: Props) {
  const { t } = useTranslation()
  return (
    <div
      className={cn('flex flex-col gap-3', className)}
      role='status'
      aria-label={t('common.loading')}
    >
      {Array.from({ length: rows }, (_, index) => (
        <Skeleton key={index} className='h-4 w-full' />
      ))}
    </div>
  )
}
