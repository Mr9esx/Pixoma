import { useTranslation } from 'react-i18next'
import { Skeleton } from '@/components/ui/skeleton'
import { cn } from '@/lib/utils'

type Props = {
  rows?: number
  className?: string
}

export function LoadingSkeleton({ rows = 4, className }: Props) {
  const { t } = useTranslation()
  return (
    <div
      className={cn('space-y-3', className)}
      role='status'
      aria-label={t('common.loading')}
    >
      {Array.from({ length: rows }, (_, i) => (
        <Skeleton key={i} className='h-4 w-full animate-pulse' />
      ))}
    </div>
  )
}
