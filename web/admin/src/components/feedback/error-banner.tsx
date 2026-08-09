import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

type Props = {
  message?: string
  onRetry?: () => void
  className?: string
}

export function ErrorBanner({ message, onRetry, className }: Props) {
  const { t } = useTranslation()
  return (
    <div
      role='alert'
      className={cn(
        'flex flex-wrap items-center justify-between gap-3 rounded-md border border-destructive/40 bg-destructive/10 px-4 py-3 text-sm',
        className
      )}
    >
      <p>{message ?? t('common.errorGeneric')}</p>
      {onRetry ? (
        <Button type='button' variant='outline' size='sm' onClick={onRetry}>
          {t('common.retry')}
        </Button>
      ) : null}
    </div>
  )
}
